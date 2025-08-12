package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type RedisManager struct {
	client   *redis.Client
	enabled  bool
	prefix   string
	ctx      context.Context
}

type CacheConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Prefix   string `json:"prefix"`
	URL      string `json:"url"` // For Upstash Redis URL
}

type CacheStats struct {
	Connected       bool          `json:"connected"`
	TotalKeys       int64         `json:"total_keys"`
	UsedMemory      string        `json:"used_memory"`
	HitRate         float64       `json:"hit_rate"`
	MissRate        float64       `json:"miss_rate"`
	TotalHits       int64         `json:"total_hits"`
	TotalMisses     int64         `json:"total_misses"`
	TotalOperations int64         `json:"total_operations"`
	LastUpdated     time.Time     `json:"last_updated"`
}

var (
	globalRedisManager *RedisManager
	cacheStats         = &CacheStats{
		LastUpdated: time.Now(),
	}
)

func NewRedisManager(config CacheConfig) *RedisManager {
	ctx := context.Background()
	
	rm := &RedisManager{
		enabled: config.Enabled,
		prefix:  config.Prefix,
		ctx:     ctx,
	}

	if !config.Enabled {
		logrus.Info("[CACHE] Redis cache is disabled")
		return rm
	}

	// Try to connect to Redis with retries
	var rdb *redis.Client
	var err error
	
	for attempt := 1; attempt <= 3; attempt++ {
		logrus.Infof("[CACHE] Attempting to connect to Redis (attempt %d/3)...", attempt)
		
		// Initialize Redis client - prioritize URL for cloud Redis like Upstash
		if config.URL != "" {
			// Use Redis URL (for Upstash or other cloud Redis)
			logrus.Infof("[CACHE] Using Redis URL: %s", config.URL)
			opt, parseErr := redis.ParseURL(config.URL)
			if parseErr != nil {
				logrus.Errorf("[CACHE] Failed to parse Redis URL: %v", parseErr)
				rm.enabled = false
				return rm
			}
			rdb = redis.NewClient(opt)
		} else {
			// Use individual connection parameters (fallback)
			logrus.Infof("[CACHE] Using connection parameters: %s:%d", config.Host, config.Port)
			rdb = redis.NewClient(&redis.Options{
				Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
				Password: config.Password,
				DB:       config.DB,
				TLSConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			})
		}

		// Test connection with timeout
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Increased timeout for cloud Redis
		
		logrus.Infof("[CACHE] Testing Redis connection (attempt %d)...", attempt)
		_, err = rdb.Ping(pingCtx).Result()
		cancel()
		
		if err == nil {
			logrus.Infof("[CACHE] Successfully connected to Redis on attempt %d", attempt)
			break
		}
		
		logrus.Warnf("[CACHE] Redis connection attempt %d failed: %v", attempt, err)
		
		// Close the failed connection
		rdb.Close()
		
		if attempt < 3 {
			// Wait before retrying with exponential backoff
			waitTime := time.Duration(attempt) * 3 * time.Second
			logrus.Infof("[CACHE] Waiting %v before retry...", waitTime)
			time.Sleep(waitTime)
		}
	}

	// Check if we successfully connected after retries
	if err != nil {
		logrus.Warnf("[CACHE] Failed to connect to Redis after 3 attempts: %v", err)
		logrus.Warnf("[CACHE] Redis cache will be disabled. Application will continue without caching.")
		rm.enabled = false
		return rm
	}

	rm.client = rdb
	globalRedisManager = rm

	// Start stats collection
	go rm.startStatsCollection()

	logrus.Info("[CACHE] Redis cache initialized successfully")
	return rm
}

// Set stores a value in Redis with optional expiration
func (rm *RedisManager) Set(key string, value interface{}, expiration time.Duration) error {
	if !rm.enabled {
		// Silently fail when Redis is disabled instead of returning error
		return nil
	}

	fullKey := rm.getFullKey(key)
	
	// Serialize value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		cacheStats.TotalMisses++
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = rm.client.Set(rm.ctx, fullKey, data, expiration).Err()
	if err != nil {
		cacheStats.TotalMisses++
		return fmt.Errorf("failed to set cache: %w", err)
	}

	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Set key: %s (expires in %v)", key, expiration)
	return nil
}

// Get retrieves a value from Redis
func (rm *RedisManager) Get(key string, dest interface{}) error {
	if !rm.enabled {
		// Return cache miss when Redis is disabled
		return fmt.Errorf("cache miss")
	}

	fullKey := rm.getFullKey(key)
	
	data, err := rm.client.Get(rm.ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			cacheStats.TotalMisses++
			return fmt.Errorf("cache miss")
		}
		cacheStats.TotalMisses++
		return fmt.Errorf("failed to get cache: %w", err)
	}

	// Deserialize JSON to destination
	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		cacheStats.TotalMisses++
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	cacheStats.TotalHits++
	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Hit key: %s", key)
	return nil
}

// Delete removes a key from Redis
func (rm *RedisManager) Delete(key string) error {
	if !rm.enabled {
		return fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	err := rm.client.Del(rm.ctx, fullKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Deleted key: %s", key)
	return nil
}

// Exists checks if a key exists in Redis
func (rm *RedisManager) Exists(key string) (bool, error) {
	if !rm.enabled {
		return false, fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	count, err := rm.client.Exists(rm.ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	cacheStats.TotalOperations++
	return count > 0, nil
}

// SetHash stores a hash in Redis
func (rm *RedisManager) SetHash(key string, fields map[string]interface{}, expiration time.Duration) error {
	if !rm.enabled {
		return fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	// Convert values to strings
	stringFields := make(map[string]interface{})
	for k, v := range fields {
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal field %s: %w", k, err)
		}
		stringFields[k] = string(data)
	}

	err := rm.client.HMSet(rm.ctx, fullKey, stringFields).Err()
	if err != nil {
		cacheStats.TotalMisses++
		return fmt.Errorf("failed to set hash: %w", err)
	}

	if expiration > 0 {
		rm.client.Expire(rm.ctx, fullKey, expiration)
	}

	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Set hash: %s with %d fields", key, len(fields))
	return nil
}

// GetHash retrieves a hash from Redis
func (rm *RedisManager) GetHash(key string) (map[string]string, error) {
	if !rm.enabled {
		return nil, fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	result, err := rm.client.HGetAll(rm.ctx, fullKey).Result()
	if err != nil {
		cacheStats.TotalMisses++
		return nil, fmt.Errorf("failed to get hash: %w", err)
	}

	if len(result) == 0 {
		cacheStats.TotalMisses++
		return nil, fmt.Errorf("cache miss")
	}

	cacheStats.TotalHits++
	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Hit hash: %s with %d fields", key, len(result))
	return result, nil
}

// SetList stores a list in Redis
func (rm *RedisManager) SetList(key string, values []interface{}, expiration time.Duration) error {
	if !rm.enabled {
		return fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	// Clear existing list
	rm.client.Del(rm.ctx, fullKey)

	// Add values to list
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal list value: %w", err)
		}
		rm.client.RPush(rm.ctx, fullKey, string(data))
	}

	if expiration > 0 {
		rm.client.Expire(rm.ctx, fullKey, expiration)
	}

	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Set list: %s with %d items", key, len(values))
	return nil
}

// GetList retrieves a list from Redis
func (rm *RedisManager) GetList(key string) ([]string, error) {
	if !rm.enabled {
		return nil, fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	result, err := rm.client.LRange(rm.ctx, fullKey, 0, -1).Result()
	if err != nil {
		cacheStats.TotalMisses++
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	if len(result) == 0 {
		cacheStats.TotalMisses++
		return nil, fmt.Errorf("cache miss")
	}

	cacheStats.TotalHits++
	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Hit list: %s with %d items", key, len(result))
	return result, nil
}

// Increment increments a numeric value in Redis
func (rm *RedisManager) Increment(key string, delta int64) (int64, error) {
	if !rm.enabled {
		return 0, fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	result, err := rm.client.IncrBy(rm.ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Incremented key: %s by %d, new value: %d", key, delta, result)
	return result, nil
}

// SetExpiration sets expiration for an existing key
func (rm *RedisManager) SetExpiration(key string, expiration time.Duration) error {
	if !rm.enabled {
		return fmt.Errorf("redis cache is disabled")
	}

	fullKey := rm.getFullKey(key)
	
	err := rm.client.Expire(rm.ctx, fullKey, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	cacheStats.TotalOperations++
	logrus.Debugf("[CACHE] Set expiration for key: %s to %v", key, expiration)
	return nil
}

// FlushAll clears all keys with the current prefix
func (rm *RedisManager) FlushAll() error {
	if !rm.enabled {
		return fmt.Errorf("redis cache is disabled")
	}

	pattern := rm.getFullKey("*")
	keys, err := rm.client.Keys(rm.ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) > 0 {
		err = rm.client.Del(rm.ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	cacheStats.TotalOperations++
	logrus.Infof("[CACHE] Flushed %d keys", len(keys))
	return nil
}

// GetStats returns cache statistics
func (rm *RedisManager) GetStats() *CacheStats {
	if !rm.enabled {
		return &CacheStats{
			Connected:   false,
			LastUpdated: time.Now(),
		}
	}

	// Update connection status
	_, err := rm.client.Ping(rm.ctx).Result()
	cacheStats.Connected = err == nil

	// Calculate hit rate
	if cacheStats.TotalOperations > 0 {
		cacheStats.HitRate = float64(cacheStats.TotalHits) / float64(cacheStats.TotalOperations) * 100
		cacheStats.MissRate = float64(cacheStats.TotalMisses) / float64(cacheStats.TotalOperations) * 100
	}

	cacheStats.LastUpdated = time.Now()
	return cacheStats
}

// IsEnabled returns whether Redis cache is enabled
func (rm *RedisManager) IsEnabled() bool {
	return rm.enabled
}

// ListKeys lists all keys matching a pattern
func (rm *RedisManager) ListKeys(pattern string) ([]string, error) {
	if !rm.enabled {
		return nil, fmt.Errorf("redis cache is disabled")
	}

	fullPattern := rm.getFullKey(pattern)
	
	keys, err := rm.client.Keys(rm.ctx, fullPattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	// Remove prefix from keys for cleaner output
	cleanKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		if rm.prefix != "" && strings.HasPrefix(key, rm.prefix+":") {
			cleanKeys = append(cleanKeys, strings.TrimPrefix(key, rm.prefix+":"))
		} else {
			cleanKeys = append(cleanKeys, key)
		}
	}

	return cleanKeys, nil
}

// TestConnection tests the Redis connection and returns detailed status
func (rm *RedisManager) TestConnection() map[string]interface{} {
	if !rm.enabled {
		return map[string]interface{}{
			"enabled":   false,
			"connected": false,
			"message":   "Redis cache is disabled",
		}
	}

	if rm.client == nil {
		return map[string]interface{}{
			"enabled":   true,
			"connected": false,
			"message":   "Redis client not initialized",
		}
	}

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rm.client.Ping(ctx).Result()
	
	if err == nil {
		return map[string]interface{}{
			"enabled":   true,
			"connected": true,
			"message":   "Redis connection successful",
		}
	}

	return map[string]interface{}{
		"enabled":   true,
		"connected": false,
		"message":   fmt.Sprintf("Redis connection failed: %v", err),
		"error":     err.Error(),
	}
}

// Private methods

func (rm *RedisManager) getFullKey(key string) string {
	if rm.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", rm.prefix, key)
}

func (rm *RedisManager) startStatsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !rm.enabled {
			continue
		}

		// Get Redis info
		info, err := rm.client.Info(rm.ctx, "memory", "keyspace").Result()
		if err != nil {
			continue
		}

		// Parse memory usage (simplified)
		if usedMemoryIdx := findInfoValue(info, "used_memory_human"); usedMemoryIdx != "" {
			cacheStats.UsedMemory = usedMemoryIdx
		}

		// Count keys with our prefix
		pattern := rm.getFullKey("*")
		keys, err := rm.client.Keys(rm.ctx, pattern).Result()
		if err == nil {
			cacheStats.TotalKeys = int64(len(keys))
		}
	}
}

func findInfoValue(info, key string) string {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			return strings.TrimPrefix(line, key+":")
		}
	}
	return ""
}

// Global cache functions for easy access

func InitializeCache(config CacheConfig) {
	NewRedisManager(config)
}

func Set(key string, value interface{}, expiration time.Duration) error {
	if globalRedisManager == nil {
		// Silently fail when cache is not initialized
		return nil
	}
	return globalRedisManager.Set(key, value, expiration)
}

func Get(key string, dest interface{}) error {
	if globalRedisManager == nil {
		// Return cache miss when cache is not initialized
		return fmt.Errorf("cache miss")
	}
	return globalRedisManager.Get(key, dest)
}

func Delete(key string) error {
	if globalRedisManager == nil {
		return fmt.Errorf("cache not initialized")
	}
	return globalRedisManager.Delete(key)
}

func Exists(key string) (bool, error) {
	if globalRedisManager == nil {
		return false, fmt.Errorf("cache not initialized")
	}
	return globalRedisManager.Exists(key)
}

func GetStats() *CacheStats {
	if globalRedisManager == nil {
		return &CacheStats{Connected: false, LastUpdated: time.Now()}
	}
	return globalRedisManager.GetStats()
}

func IsEnabled() bool {
	if globalRedisManager == nil {
		return false
	}
	return globalRedisManager.IsEnabled()
}

// TestConnection tests the global Redis connection
func TestConnection() map[string]interface{} {
	if globalRedisManager == nil {
		return map[string]interface{}{
			"enabled":   false,
			"connected": false,
			"message":   "Redis cache not initialized",
		}
	}
	return globalRedisManager.TestConnection()
}

// ListKeys lists all keys matching a pattern
func ListKeys(pattern string) ([]string, error) {
	if globalRedisManager == nil {
		return nil, fmt.Errorf("cache not initialized")
	}
	return globalRedisManager.ListKeys(pattern)
}
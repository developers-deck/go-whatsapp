package rest

import (
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/cache"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Cache struct {
	manager *cache.RedisManager
}

func InitRestCache(app fiber.Router) Cache {
	// Initialize cache from config
	cacheConfig := cache.CacheConfig{
		Enabled:  config.RedisEnabled,
		Host:     config.RedisHost,
		Port:     config.RedisPort,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
		Prefix:   config.RedisPrefix,
		URL:      config.RedisURL, // For Upstash Redis
	}

	rm := cache.NewRedisManager(cacheConfig)
	rest := Cache{manager: rm}

	// Cache management routes
	app.Post("/cache/set", rest.SetCache)
	app.Get("/cache/get/:key", rest.GetCache)
	app.Delete("/cache/:key", rest.DeleteCache)
	app.Get("/cache/exists/:key", rest.ExistsCache)
	app.Post("/cache/hash", rest.SetHash)
	app.Get("/cache/hash/:key", rest.GetHash)
	app.Post("/cache/list", rest.SetList)
	app.Get("/cache/list/:key", rest.GetList)
	app.Post("/cache/increment", rest.Increment)
	app.Post("/cache/expire", rest.SetExpiration)
	app.Delete("/cache/flush", rest.FlushAll)
	app.Get("/cache/stats", rest.GetStats)
	app.Get("/cache/health", rest.GetHealth)
	app.Get("/cache/test", rest.TestConnection)
	app.Get("/cache/keys", rest.ListKeys)

	return rest
}

func (handler *Cache) SetCache(c *fiber.Ctx) error {
	var request struct {
		Key        string      `json:"key"`
		Value      interface{} `json:"value"`
		Expiration int         `json:"expiration"` // seconds
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	expiration := time.Duration(request.Expiration) * time.Second
	if request.Expiration == 0 {
		expiration = 0 // No expiration
	}

	err := handler.manager.Set(request.Key, request.Value, expiration)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache set successfully",
		Results: map[string]interface{}{
			"key":        request.Key,
			"expiration": request.Expiration,
		},
	})
}

func (handler *Cache) GetCache(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	var value interface{}
	err := handler.manager.Get(key, &value)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "CACHE_MISS",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache retrieved successfully",
		Results: map[string]interface{}{
			"key":   key,
			"value": value,
		},
	})
}

func (handler *Cache) DeleteCache(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	err := handler.manager.Delete(key)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache deleted successfully",
		Results: map[string]interface{}{
			"key": key,
		},
	})
}

func (handler *Cache) ExistsCache(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	exists, err := handler.manager.Exists(key)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache existence checked",
		Results: map[string]interface{}{
			"key":    key,
			"exists": exists,
		},
	})
}

func (handler *Cache) SetHash(c *fiber.Ctx) error {
	var request struct {
		Key        string                 `json:"key"`
		Fields     map[string]interface{} `json:"fields"`
		Expiration int                    `json:"expiration"` // seconds
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Key == "" || len(request.Fields) == 0 {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key and fields are required",
			Results: nil,
		})
	}

	expiration := time.Duration(request.Expiration) * time.Second
	if request.Expiration == 0 {
		expiration = 0 // No expiration
	}

	err := handler.manager.SetHash(request.Key, request.Fields, expiration)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Hash set successfully",
		Results: map[string]interface{}{
			"key":         request.Key,
			"field_count": len(request.Fields),
			"expiration":  request.Expiration,
		},
	})
}

func (handler *Cache) GetHash(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	fields, err := handler.manager.GetHash(key)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "CACHE_MISS",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Hash retrieved successfully",
		Results: map[string]interface{}{
			"key":    key,
			"fields": fields,
		},
	})
}

func (handler *Cache) SetList(c *fiber.Ctx) error {
	var request struct {
		Key        string        `json:"key"`
		Values     []interface{} `json:"values"`
		Expiration int           `json:"expiration"` // seconds
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Key == "" || len(request.Values) == 0 {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key and values are required",
			Results: nil,
		})
	}

	expiration := time.Duration(request.Expiration) * time.Second
	if request.Expiration == 0 {
		expiration = 0 // No expiration
	}

	err := handler.manager.SetList(request.Key, request.Values, expiration)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "List set successfully",
		Results: map[string]interface{}{
			"key":        request.Key,
			"item_count": len(request.Values),
			"expiration": request.Expiration,
		},
	})
}

func (handler *Cache) GetList(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	values, err := handler.manager.GetList(key)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "CACHE_MISS",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "List retrieved successfully",
		Results: map[string]interface{}{
			"key":    key,
			"values": values,
		},
	})
}

func (handler *Cache) Increment(c *fiber.Ctx) error {
	var request struct {
		Key   string `json:"key"`
		Delta int64  `json:"delta"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Key == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key is required",
			Results: nil,
		})
	}

	if request.Delta == 0 {
		request.Delta = 1 // Default increment
	}

	newValue, err := handler.manager.Increment(request.Key, request.Delta)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Value incremented successfully",
		Results: map[string]interface{}{
			"key":       request.Key,
			"delta":     request.Delta,
			"new_value": newValue,
		},
	})
}

func (handler *Cache) SetExpiration(c *fiber.Ctx) error {
	var request struct {
		Key        string `json:"key"`
		Expiration int    `json:"expiration"` // seconds
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Key == "" || request.Expiration <= 0 {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Key and positive expiration are required",
			Results: nil,
		})
	}

	expiration := time.Duration(request.Expiration) * time.Second
	err := handler.manager.SetExpiration(request.Key, expiration)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Expiration set successfully",
		Results: map[string]interface{}{
			"key":        request.Key,
			"expiration": request.Expiration,
		},
	})
}

func (handler *Cache) FlushAll(c *fiber.Ctx) error {
	err := handler.manager.FlushAll()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache flushed successfully",
		Results: nil,
	})
}

func (handler *Cache) GetStats(c *fiber.Ctx) error {
	stats := handler.manager.GetStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache statistics retrieved",
		Results: stats,
	})
}

func (handler *Cache) GetHealth(c *fiber.Ctx) error {
	stats := handler.manager.GetStats()
	health := map[string]interface{}{
		"status":    "healthy",
		"connected": stats.Connected,
		"redis": map[string]interface{}{
			"enabled":  handler.manager.IsEnabled(),
			"host":     config.RedisHost,
			"port":     config.RedisPort,
			"database": config.RedisDB,
		},
		"timestamp": time.Now(),
	}
	
	if !stats.Connected {
		health["status"] = "unhealthy"
		health["message"] = "Redis connection failed"
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache health check completed",
		Results: health,
	})
}

func (handler *Cache) TestConnection(c *fiber.Ctx) error {
	connectionStatus := handler.manager.TestConnection()
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Redis connection test completed",
		Results: connectionStatus,
	})
}

func (handler *Cache) ListKeys(c *fiber.Ctx) error {
	// Get pattern from query parameter, default to all keys
	pattern := c.Query("pattern", "*")
	
	keys, err := handler.manager.ListKeys(pattern)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CACHE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Cache keys retrieved successfully",
		Results: map[string]interface{}{
			"keys":  keys,
			"count": len(keys),
		},
	})
}
package multiinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/isolation"
	"github.com/sirupsen/logrus"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type InstanceManager struct {
	instances         map[string]*WhatsAppInstance
	isolationMgr      *isolation.ProcessIsolationManager
	dbIsolationMgr    *isolation.DatabaseIsolationManager
	sessionIsolationMgr *isolation.SessionIsolationManager
	mutex             sync.RWMutex
	basePath          string
	ctx               context.Context
	cancel            context.CancelFunc
	isolationConfig   isolation.IsolationConfig
}

type WhatsAppInstance struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Phone       string            `json:"phone"`
	Status      InstanceStatus    `json:"status"`
	Port        int               `json:"port"`
	PID         int               `json:"pid"`
	WorkingDir  string            `json:"working_dir"`
	ConfigPath  string            `json:"config_path"`
	LogPath     string            `json:"log_path"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	LastSeen    *time.Time        `json:"last_seen,omitempty"`
	Config      InstanceConfig    `json:"config"`
	Process     *os.Process       `json:"-"`
	Metadata    map[string]string `json:"metadata"`
	mutex       sync.RWMutex      `json:"-"`
}

type InstanceStatus string

const (
	StatusStopped    InstanceStatus = "stopped"
	StatusStarting   InstanceStatus = "starting"
	StatusRunning    InstanceStatus = "running"
	StatusStopping   InstanceStatus = "stopping"
	StatusError      InstanceStatus = "error"
	StatusRestarting InstanceStatus = "restarting"
)

type InstanceConfig struct {
	Port                int               `json:"port"`
	Debug               bool              `json:"debug"`
	OS                  string            `json:"os"`
	BasicAuth           []string          `json:"basic_auth"`
	BasePath            string            `json:"base_path"`
	DBURI               string            `json:"db_uri"`
	DBKeysURI           string            `json:"db_keys_uri"`
	AutoReply           string            `json:"auto_reply"`
	AutoMarkRead        bool              `json:"auto_mark_read"`
	Webhooks            []string          `json:"webhooks"`
	WebhookSecret       string            `json:"webhook_secret"`
	AccountValidation   bool              `json:"account_validation"`
	Environment         map[string]string `json:"environment"`
}

type InstanceStats struct {
	TotalInstances   int                        `json:"total_instances"`
	RunningInstances int                        `json:"running_instances"`
	StoppedInstances int                        `json:"stopped_instances"`
	ErrorInstances   int                        `json:"error_instances"`
	InstancesByStatus map[InstanceStatus]int    `json:"instances_by_status"`
	ResourceUsage    map[string]ResourceUsage   `json:"resource_usage"`
	LastUpdated      time.Time                  `json:"last_updated"`
}

type ResourceUsage struct {
	CPU    float64 `json:"cpu_percent"`
	Memory int64   `json:"memory_mb"`
	PID    int     `json:"pid"`
}

func NewInstanceManager() *InstanceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	basePath := filepath.Join(config.PathStorages, "instances")
	os.MkdirAll(basePath, 0755)

	// Initialize process isolation configuration
	isolationConfig := isolation.IsolationConfig{
		EnableResourceLimits: true,
		DefaultMemoryLimit:   512, // 512MB per instance
		DefaultCPULimit:      50.0, // 50% CPU limit
		DefaultTimeout:       30 * time.Minute,
		MonitoringInterval:   10 * time.Second,
		AutoRestart:          true,
		MaxRestarts:          3,
		PathStorages:         config.PathStorages,
	}

	// Initialize database isolation manager based on configuration
	var dbIsolationMgr *isolation.DatabaseIsolationManager
	if strings.HasPrefix(config.DBURI, "postgres:") {
		// Use PostgreSQL for multi-instance isolation
		dbIsolationMgr = isolation.NewPostgresDatabaseIsolationManager(config.PathStorages, config.DBURI)
		logrus.Info("[MULTIINSTANCE] Using PostgreSQL for database isolation")
	} else {
		// Use SQLite for multi-instance isolation
		dbIsolationMgr = isolation.NewDatabaseIsolationManager(config.PathStorages)
		logrus.Info("[MULTIINSTANCE] Using SQLite for database isolation")
	}

	im := &InstanceManager{
		instances:       make(map[string]*WhatsAppInstance),
		isolationMgr:    isolation.NewProcessIsolationManager(isolationConfig),
		dbIsolationMgr:  dbIsolationMgr,
		basePath:        basePath,
		ctx:             ctx,
		cancel:          cancel,
		isolationConfig: isolationConfig,
	}

	// Load existing instances
	im.loadInstances()

	// Start monitoring
	go im.startMonitoring()

	logrus.Info("[MULTIINSTANCE] Instance manager initialized with process isolation and database support")
	return im
}

// CreateInstance creates a new WhatsApp instance
func (im *InstanceManager) CreateInstance(name, phone string, config InstanceConfig) (*WhatsAppInstance, error) {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Generate unique ID
	instanceID := im.generateInstanceID(name)

	// Check if port is available
	if config.Port == 0 {
		config.Port = im.findAvailablePort()
	}

	// Create instance directory
	instanceDir := filepath.Join(im.basePath, instanceID)
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create instance directory: %w", err)
	}

	// Create subdirectories
	dirs := []string{"storages", "statics/qrcode", "statics/senditems", "statics/media", "logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(instanceDir, dir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create isolated database for this instance
	isolatedDB, err := im.dbIsolationMgr.CreateIsolatedDatabase(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create isolated database: %w", err)
	}

	// Set database paths - each instance gets its own isolated database
	if config.DBURI == "" {
		config.DBURI = fmt.Sprintf("file:%s?_foreign_keys=on", isolatedDB.DatabasePath)
	}
	
	// Set isolated session storage
	if config.DBKeysURI == "" {
		config.DBKeysURI = fmt.Sprintf("file:%s?_foreign_keys=on", isolatedDB.KeysPath)
	}

	instance := &WhatsAppInstance{
		ID:         instanceID,
		Name:       name,
		Phone:      phone,
		Status:     StatusStopped,
		Port:       config.Port,
		WorkingDir: instanceDir,
		ConfigPath: filepath.Join(instanceDir, "config.json"),
		LogPath:    filepath.Join(instanceDir, "logs", "app.log"),
		CreatedAt:  time.Now(),
		Config:     config,
		Metadata:   make(map[string]string),
	}

	// Save instance configuration
	if err := im.saveInstanceConfig(instance); err != nil {
		return nil, fmt.Errorf("failed to save instance config: %w", err)
	}

	im.instances[instanceID] = instance
	
	// Save instances list
	im.saveInstances()

	logrus.Infof("[MULTIINSTANCE] Created instance: %s (%s) on port %d", name, instanceID, config.Port)
	return instance, nil
}

// StartInstance starts a WhatsApp instance using process isolation
func (im *InstanceManager) StartInstance(instanceID string) error {
	im.mutex.RLock()
	instance, exists := im.instances[instanceID]
	im.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status == StatusRunning {
		return fmt.Errorf("instance is already running")
	}

	instance.Status = StatusStarting
	logrus.Infof("[MULTIINSTANCE] Starting isolated instance: %s", instanceID)

	// Get executable path
	executable, err := os.Executable()
	if err != nil {
		instance.Status = StatusError
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare command arguments for isolated process
	args := []string{"rest"}
	args = append(args, "--port", fmt.Sprintf("%d", instance.Config.Port))
	
	if instance.Config.Debug {
		args = append(args, "--debug", "true")
	}
	
	if instance.Config.OS != "" {
		args = append(args, "--os", instance.Config.OS)
	}
	
	if len(instance.Config.BasicAuth) > 0 {
		for _, auth := range instance.Config.BasicAuth {
			args = append(args, "--basic-auth", auth)
		}
	}
	
	if instance.Config.BasePath != "" {
		args = append(args, "--base-path", instance.Config.BasePath)
	}
	
	args = append(args, "--db-uri", instance.Config.DBURI)
	
	if instance.Config.DBKeysURI != "" {
		args = append(args, "--db-keys-uri", instance.Config.DBKeysURI)
	}
	
	if instance.Config.AutoReply != "" {
		args = append(args, "--autoreply", instance.Config.AutoReply)
	}
	
	if instance.Config.AutoMarkRead {
		args = append(args, "--auto-mark-read", "true")
	}
	
	if len(instance.Config.Webhooks) > 0 {
		for _, webhook := range instance.Config.Webhooks {
			args = append(args, "--webhook", webhook)
		}
	}
	
	if instance.Config.WebhookSecret != "" {
		args = append(args, "--webhook-secret", instance.Config.WebhookSecret)
	}
	
	if !instance.Config.AccountValidation {
		args = append(args, "--account-validation", "false")
	}

	// Add instance-specific environment variables for complete isolation
	instanceEnv := make(map[string]string)
	for key, value := range instance.Config.Environment {
		instanceEnv[key] = value
	}
	
	// Set instance-specific paths for complete isolation
	instanceEnv["WHATSAPP_INSTANCE_ID"] = instanceID
	instanceEnv["WHATSAPP_INSTANCE_NAME"] = instance.Name
	instanceEnv["WHATSAPP_INSTANCE_PHONE"] = instance.Phone
	instanceEnv["WHATSAPP_STORAGE_PATH"] = filepath.Join(instance.WorkingDir, "storages")
	instanceEnv["WHATSAPP_STATIC_PATH"] = filepath.Join(instance.WorkingDir, "statics")
	instanceEnv["WHATSAPP_LOG_PATH"] = filepath.Join(instance.WorkingDir, "logs")

	// Create isolated process using the isolation manager
	isolatedProcess, err := im.isolationMgr.CreateIsolatedProcess(
		instanceID,
		fmt.Sprintf("whatsapp-%s", instance.Name),
		executable,
		args,
		im.isolationConfig,
	)
	if err != nil {
		instance.Status = StatusError
		return fmt.Errorf("failed to create isolated process: %w", err)
	}

	// Set environment variables for the isolated process
	for key, value := range instanceEnv {
		isolatedProcess.Environment[key] = value
	}

	// Start the isolated process
	if err := im.isolationMgr.StartProcess(instanceID); err != nil {
		instance.Status = StatusError
		return fmt.Errorf("failed to start isolated process: %w", err)
	}

	// Get the process details from isolation manager
	isolatedProcess, err = im.isolationMgr.GetProcess(instanceID)
	if err != nil {
		instance.Status = StatusError
		return fmt.Errorf("failed to get isolated process: %w", err)
	}

	instance.PID = isolatedProcess.PID
	instance.Status = StatusRunning
	now := time.Now()
	instance.StartedAt = &now
	instance.LastSeen = &now

	logrus.Infof("[MULTIINSTANCE] Started isolated instance: %s (PID: %d)", instanceID, instance.PID)
	return nil
}

// StopInstance stops a WhatsApp instance using process isolation
func (im *InstanceManager) StopInstance(instanceID string) error {
	im.mutex.RLock()
	instance, exists := im.instances[instanceID]
	im.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	instance.mutex.Lock()
	defer instance.mutex.Unlock()

	if instance.Status != StatusRunning {
		return fmt.Errorf("instance is not running")
	}

	instance.Status = StatusStopping
	logrus.Infof("[MULTIINSTANCE] Stopping isolated instance: %s", instanceID)

	// Stop the isolated process
	if err := im.isolationMgr.StopProcess(instanceID); err != nil {
		logrus.Errorf("[MULTIINSTANCE] Failed to stop isolated process: %v", err)
		// Continue with cleanup even if stop failed
	}

	instance.Status = StatusStopped
	instance.Process = nil
	instance.PID = 0

	logrus.Infof("[MULTIINSTANCE] Stopped isolated instance: %s", instanceID)
	return nil
}

// RestartInstance restarts a WhatsApp instance using process isolation
func (im *InstanceManager) RestartInstance(instanceID string) error {
	instance, exists := im.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	instance.Status = StatusRestarting
	logrus.Infof("[MULTIINSTANCE] Restarting isolated instance: %s", instanceID)

	// Use the isolation manager's restart functionality
	if err := im.isolationMgr.RestartProcess(instanceID); err != nil {
		return fmt.Errorf("failed to restart isolated process: %w", err)
	}

	// Update instance status
	isolatedProcess, err := im.isolationMgr.GetProcess(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get restarted process: %w", err)
	}

	instance.mutex.Lock()
	instance.PID = isolatedProcess.PID
	instance.Status = StatusRunning
	now := time.Now()
	instance.StartedAt = &now
	instance.LastSeen = &now
	instance.mutex.Unlock()

	logrus.Infof("[MULTIINSTANCE] Restarted isolated instance: %s (PID: %d)", instanceID, instance.PID)
	return nil
}

// DeleteInstance removes a WhatsApp instance and its isolated process
func (im *InstanceManager) DeleteInstance(instanceID string) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	instance, exists := im.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	// Stop instance if running
	if instance.Status == StatusRunning {
		im.StopInstance(instanceID)
		time.Sleep(2 * time.Second)
	}

	// Delete the isolated process
	if err := im.isolationMgr.DeleteProcess(instanceID); err != nil {
		logrus.Warnf("[MULTIINSTANCE] Failed to delete isolated process: %v", err)
	}

	// Delete the isolated database
	if err := im.dbIsolationMgr.DeleteIsolatedDatabase(instanceID); err != nil {
		logrus.Warnf("[MULTIINSTANCE] Failed to delete isolated database: %v", err)
	}

	// Remove instance directory
	if err := os.RemoveAll(instance.WorkingDir); err != nil {
		logrus.Warnf("[MULTIINSTANCE] Failed to remove instance directory: %v", err)
	}

	// Remove from instances map
	delete(im.instances, instanceID)
	
	// Save instances list
	im.saveInstances()

	logrus.Infof("[MULTIINSTANCE] Deleted isolated instance: %s", instanceID)
	return nil
}

// GetInstance retrieves an instance by ID
func (im *InstanceManager) GetInstance(instanceID string) (*WhatsAppInstance, error) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	instance, exists := im.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	return instance, nil
}

// ListInstances returns all instances
func (im *InstanceManager) ListInstances() []*WhatsAppInstance {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	instances := make([]*WhatsAppInstance, 0, len(im.instances))
	for _, instance := range im.instances {
		instances = append(instances, instance)
	}

	return instances
}

// GetStats returns instance statistics
func (im *InstanceManager) GetStats() *InstanceStats {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	stats := &InstanceStats{
		TotalInstances:    len(im.instances),
		InstancesByStatus: make(map[InstanceStatus]int),
		ResourceUsage:     make(map[string]ResourceUsage),
		LastUpdated:       time.Now(),
	}

	for _, instance := range im.instances {
		instance.mutex.RLock()
		stats.InstancesByStatus[instance.Status]++
		
		switch instance.Status {
		case StatusRunning:
			stats.RunningInstances++
		case StatusStopped:
			stats.StoppedInstances++
		case StatusError:
			stats.ErrorInstances++
		}

		// Get resource usage if running
		if instance.Status == StatusRunning && instance.PID > 0 {
			stats.ResourceUsage[instance.ID] = ResourceUsage{
				PID:    instance.PID,
				CPU:    0.0, // Would be calculated from system metrics
				Memory: 0,   // Would be calculated from system metrics
			}
		}
		instance.mutex.RUnlock()
	}

	return stats
}

// Private methods

func (im *InstanceManager) generateInstanceID(name string) string {
	// Create a safe ID from name + timestamp
	safeName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	safeName = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(safeName, "")
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s_%s", safeName, timestamp)
}

func (im *InstanceManager) findAvailablePort() int {
	// Start from port 3001 and find the first available port
	for port := 3001; port < 4000; port++ {
		if im.isPortAvailable(port) {
			return port
		}
	}
	return 3001 // Fallback
}

func (im *InstanceManager) isPortAvailable(port int) bool {
	// Check if port is already used by existing instances
	for _, instance := range im.instances {
		if instance.Config.Port == port {
			return false
		}
	}
	return true
}

func (im *InstanceManager) saveInstanceConfig(instance *WhatsAppInstance) error {
	data, err := json.MarshalIndent(instance.Config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(instance.ConfigPath, data, 0644)
}

func (im *InstanceManager) saveInstances() error {
	instancesFile := filepath.Join(im.basePath, "instances.json")
	
	// Create a simplified version for saving
	saveData := make(map[string]interface{})
	for id, instance := range im.instances {
		saveData[id] = map[string]interface{}{
			"id":          instance.ID,
			"name":        instance.Name,
			"phone":       instance.Phone,
			"port":        instance.Port,
			"working_dir": instance.WorkingDir,
			"config_path": instance.ConfigPath,
			"log_path":    instance.LogPath,
			"created_at":  instance.CreatedAt,
			"metadata":    instance.Metadata,
		}
	}
	
	data, err := json.MarshalIndent(saveData, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(instancesFile, data, 0644)
}

func (im *InstanceManager) loadInstances() {
	instancesFile := filepath.Join(im.basePath, "instances.json")
	
	data, err := os.ReadFile(instancesFile)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Errorf("[MULTIINSTANCE] Failed to read instances file: %v", err)
		}
		return
	}

	var saveData map[string]interface{}
	if err := json.Unmarshal(data, &saveData); err != nil {
		logrus.Errorf("[MULTIINSTANCE] Failed to unmarshal instances: %v", err)
		return
	}

	for id, data := range saveData {
		instanceData := data.(map[string]interface{})
		
		// Load instance config
		configPath := instanceData["config_path"].(string)
		configData, err := os.ReadFile(configPath)
		if err != nil {
			logrus.Warnf("[MULTIINSTANCE] Failed to load config for instance %s: %v", id, err)
			continue
		}

		var config InstanceConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			logrus.Warnf("[MULTIINSTANCE] Failed to unmarshal config for instance %s: %v", id, err)
			continue
		}

		instance := &WhatsAppInstance{
			ID:         instanceData["id"].(string),
			Name:       instanceData["name"].(string),
			Phone:      instanceData["phone"].(string),
			Status:     StatusStopped,
			Port:       int(instanceData["port"].(float64)),
			WorkingDir: instanceData["working_dir"].(string),
			ConfigPath: instanceData["config_path"].(string),
			LogPath:    instanceData["log_path"].(string),
			Config:     config,
			Metadata:   make(map[string]string),
		}

		// Parse created_at
		if createdAtStr, ok := instanceData["created_at"].(string); ok {
			if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
				instance.CreatedAt = createdAt
			}
		}

		im.instances[id] = instance
	}

	logrus.Infof("[MULTIINSTANCE] Loaded %d instances", len(im.instances))
}

func (im *InstanceManager) monitorInstance(instance *WhatsAppInstance, cmd *exec.Cmd, logFile *os.File) {
	defer logFile.Close()

	// Wait for process to exit
	err := cmd.Wait()
	
	instance.mutex.Lock()
	if instance.Status == StatusStopping {
		instance.Status = StatusStopped
	} else {
		instance.Status = StatusError
	}
	instance.Process = nil
	instance.PID = 0
	instance.mutex.Unlock()

	if err != nil {
		logrus.Errorf("[MULTIINSTANCE] Instance %s exited with error: %v", instance.ID, err)
	} else {
		logrus.Infof("[MULTIINSTANCE] Instance %s exited normally", instance.ID)
	}
}

func (im *InstanceManager) startMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-im.ctx.Done():
			return
		case <-ticker.C:
			im.updateInstanceStatus()
		}
	}
}

func (im *InstanceManager) updateInstanceStatus() {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	for _, instance := range im.instances {
		instance.mutex.Lock()
		if instance.Status == StatusRunning {
			// Check isolated process status
			isolatedProcess, err := im.isolationMgr.GetProcess(instance.ID)
			if err != nil {
				// Process not found in isolation manager
				instance.Status = StatusError
				instance.Process = nil
				instance.PID = 0
			} else {
				// Update status from isolated process
				switch isolatedProcess.Status {
				case isolation.ProcessStatusRunning:
					instance.Status = StatusRunning
					instance.PID = isolatedProcess.PID
					now := time.Now()
					instance.LastSeen = &now
				case isolation.ProcessStatusStopped:
					instance.Status = StatusStopped
					instance.PID = 0
				case isolation.ProcessStatusCrashed, isolation.ProcessStatusError:
					instance.Status = StatusError
					instance.PID = 0
				}
			}
		}
		instance.mutex.Unlock()
	}
}

// Stop gracefully stops the instance manager and all isolated processes
func (im *InstanceManager) Stop() {
	logrus.Info("[MULTIINSTANCE] Stopping instance manager...")
	
	// Stop all running instances
	for _, instance := range im.instances {
		if instance.Status == StatusRunning {
			im.StopInstance(instance.ID)
		}
	}
	
	// Stop the isolation manager
	im.isolationMgr.Stop()
	
	// Stop the database isolation manager
	im.dbIsolationMgr.Stop()
	
	im.cancel()
	logrus.Info("[MULTIINSTANCE] Instance manager stopped")
}
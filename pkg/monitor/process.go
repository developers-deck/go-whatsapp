package monitor

import (
	"context"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
)

type ProcessStats struct {
	PID              int                    `json:"pid"`
	StartTime        time.Time              `json:"start_time"`
	Uptime           time.Duration          `json:"uptime"`
	MemoryUsage      runtime.MemStats       `json:"memory_usage"`
	GoRoutines       int                    `json:"goroutines"`
	WhatsAppStatus   map[string]interface{} `json:"whatsapp_status"`
	HealthChecks     map[string]bool        `json:"health_checks"`
	LastHealthCheck  time.Time              `json:"last_health_check"`
	RestartCount     int                    `json:"restart_count"`
	ConfiguredLimits map[string]interface{} `json:"configured_limits"`
}

type ProcessMonitor struct {
	startTime    time.Time
	restartCount int
	mutex        sync.RWMutex
	healthTicker *time.Ticker
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewProcessMonitor() *ProcessMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	pm := &ProcessMonitor{
		startTime:    time.Now(),
		restartCount: 0,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start health check monitoring
	pm.startHealthMonitoring()
	
	return pm
}

// GetProcessStats returns current process statistics
func (pm *ProcessMonitor) GetProcessStats() ProcessStats {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get WhatsApp client status
	isConnected, isLoggedIn, deviceID := whatsapp.GetConnectionStatus()
	whatsappStatus := map[string]interface{}{
		"is_connected": isConnected,
		"is_logged_in": isLoggedIn,
		"device_id":    deviceID,
	}

	// Perform health checks
	healthChecks := pm.performHealthChecks()

	// Get configured limits
	configuredLimits := map[string]interface{}{
		"max_image_size":    config.WhatsappSettingMaxImageSize,
		"max_file_size":     config.WhatsappSettingMaxFileSize,
		"max_video_size":    config.WhatsappSettingMaxVideoSize,
		"max_download_size": config.WhatsappSettingMaxDownloadSize,
		"session_backup":    config.SessionBackupEnabled,
		"auto_restore":      config.SessionAutoRestore,
	}

	return ProcessStats{
		PID:              os.Getpid(),
		StartTime:        pm.startTime,
		Uptime:           time.Since(pm.startTime),
		MemoryUsage:      memStats,
		GoRoutines:       runtime.NumGoroutine(),
		WhatsAppStatus:   whatsappStatus,
		HealthChecks:     healthChecks,
		LastHealthCheck:  time.Now(),
		RestartCount:     pm.restartCount,
		ConfiguredLimits: configuredLimits,
	}
}

// performHealthChecks runs various health checks
func (pm *ProcessMonitor) performHealthChecks() map[string]bool {
	checks := make(map[string]bool)

	// Check if WhatsApp client is initialized
	client := whatsapp.GetClient()
	checks["whatsapp_client_initialized"] = client != nil

	// Check if client is connected
	if client != nil {
		checks["whatsapp_connected"] = client.IsConnected()
		checks["whatsapp_logged_in"] = client.IsLoggedIn()
	} else {
		checks["whatsapp_connected"] = false
		checks["whatsapp_logged_in"] = false
	}

	// Check database connectivity
	db := whatsapp.GetDB()
	if db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		_, err := db.GetAllDevices(ctx)
		checks["database_accessible"] = err == nil
	} else {
		checks["database_accessible"] = false
	}

	// Check storage directories
	checks["qr_directory_writable"] = pm.checkDirectoryWritable(config.PathQrCode)
	checks["send_items_directory_writable"] = pm.checkDirectoryWritable(config.PathSendItems)
	checks["media_directory_writable"] = pm.checkDirectoryWritable(config.PathMedia)
	checks["storage_directory_writable"] = pm.checkDirectoryWritable(config.PathStorages)

	// Check memory usage (warn if over 80% of available)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryUsageMB := float64(memStats.Alloc) / 1024 / 1024
	checks["memory_usage_healthy"] = memoryUsageMB < 500 // Less than 500MB

	// Check goroutine count (warn if over 1000)
	checks["goroutine_count_healthy"] = runtime.NumGoroutine() < 1000

	return checks
}

// checkDirectoryWritable checks if a directory is writable
func (pm *ProcessMonitor) checkDirectoryWritable(dirPath string) bool {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Try to create it
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return false
		}
	}

	// Try to create a test file
	testFile := dirPath + "/.health_check"
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	
	// Clean up test file
	os.Remove(testFile)
	return true
}

// startHealthMonitoring starts periodic health monitoring
func (pm *ProcessMonitor) startHealthMonitoring() {
	if config.SessionHealthCheckInterval <= 0 {
		return
	}

	pm.healthTicker = time.NewTicker(time.Duration(config.SessionHealthCheckInterval) * time.Second)
	
	go func() {
		defer pm.healthTicker.Stop()
		
		for {
			select {
			case <-pm.ctx.Done():
				logrus.Info("[MONITOR] Stopping health monitoring")
				return
			case <-pm.healthTicker.C:
				pm.runHealthCheck()
			}
		}
	}()

	logrus.Infof("[MONITOR] Started health monitoring (every %d seconds)", config.SessionHealthCheckInterval)
}

// runHealthCheck performs a health check and logs issues
func (pm *ProcessMonitor) runHealthCheck() {
	checks := pm.performHealthChecks()
	issues := []string{}

	for check, passed := range checks {
		if !passed {
			issues = append(issues, check)
		}
	}

	if len(issues) > 0 {
		logrus.Warnf("[MONITOR] Health check failed for: %v", issues)
		
		// Auto-recovery attempts
		pm.attemptAutoRecovery(issues)
	} else {
		logrus.Debug("[MONITOR] All health checks passed")
	}
}

// attemptAutoRecovery tries to recover from common issues
func (pm *ProcessMonitor) attemptAutoRecovery(issues []string) {
	for _, issue := range issues {
		switch issue {
		case "whatsapp_connected":
			logrus.Info("[MONITOR] Attempting to reconnect WhatsApp client")
			client := whatsapp.GetClient()
			if client != nil {
				go func() {
					if err := client.Connect(); err != nil {
						logrus.Errorf("[MONITOR] Auto-reconnect failed: %v", err)
					} else {
						logrus.Info("[MONITOR] Auto-reconnect successful")
					}
				}()
			}
			
		case "qr_directory_writable", "send_items_directory_writable", 
			 "media_directory_writable", "storage_directory_writable":
			logrus.Info("[MONITOR] Attempting to recreate directories")
			os.MkdirAll(config.PathQrCode, 0755)
			os.MkdirAll(config.PathSendItems, 0755)
			os.MkdirAll(config.PathMedia, 0755)
			os.MkdirAll(config.PathStorages, 0755)
		}
	}
}

// GetMemoryStats returns detailed memory statistics
func (pm *ProcessMonitor) GetMemoryStats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"alloc_mb":        float64(memStats.Alloc) / 1024 / 1024,
		"total_alloc_mb":  float64(memStats.TotalAlloc) / 1024 / 1024,
		"sys_mb":          float64(memStats.Sys) / 1024 / 1024,
		"heap_alloc_mb":   float64(memStats.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":     float64(memStats.HeapSys) / 1024 / 1024,
		"heap_idle_mb":    float64(memStats.HeapIdle) / 1024 / 1024,
		"heap_inuse_mb":   float64(memStats.HeapInuse) / 1024 / 1024,
		"stack_inuse_mb":  float64(memStats.StackInuse) / 1024 / 1024,
		"stack_sys_mb":    float64(memStats.StackSys) / 1024 / 1024,
		"num_gc":          memStats.NumGC,
		"gc_cpu_fraction": memStats.GCCPUFraction,
		"goroutines":      runtime.NumGoroutine(),
	}
}

// IncrementRestartCount increments the restart counter
func (pm *ProcessMonitor) IncrementRestartCount() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.restartCount++
}

// Stop stops the process monitor
func (pm *ProcessMonitor) Stop() {
	if pm.cancel != nil {
		pm.cancel()
	}
	if pm.healthTicker != nil {
		pm.healthTicker.Stop()
	}
	logrus.Info("[MONITOR] Process monitor stopped")
}
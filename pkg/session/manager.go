package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type SessionInfo struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	PushName    string    `json:"push_name"`
	IsConnected bool      `json:"is_connected"`
	IsLoggedIn  bool      `json:"is_logged_in"`
	LastSeen    time.Time `json:"last_seen"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SessionManager struct {
	backupPath string
}

func NewSessionManager() *SessionManager {
	backupPath := filepath.Join(config.PathStorages, "session_backups")
	os.MkdirAll(backupPath, 0755)
	
	return &SessionManager{
		backupPath: backupPath,
	}
}

// BackupSession creates a backup of the current session
func (sm *SessionManager) BackupSession(ctx context.Context, sessionInfo SessionInfo) error {
	if !config.SessionBackupEnabled {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(sm.backupPath, fmt.Sprintf("session_%s_%s.json", sessionInfo.ID, timestamp))

	sessionInfo.UpdatedAt = time.Now()
	
	data, err := json.MarshalIndent(sessionInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session info: %w", err)
	}

	err = os.WriteFile(backupFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	logrus.Infof("[SESSION] Session backup created: %s", backupFile)
	
	// Cleanup old backups
	go sm.cleanupOldBackups()
	
	return nil
}

// RestoreSession attempts to restore the latest session backup
func (sm *SessionManager) RestoreSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	if !config.SessionAutoRestore {
		return nil, fmt.Errorf("session auto-restore is disabled")
	}

	pattern := filepath.Join(sm.backupPath, fmt.Sprintf("session_%s_*.json", sessionID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find backup files: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no backup files found for session %s", sessionID)
	}

	// Get the latest backup file
	latestFile := matches[len(matches)-1]
	
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	var sessionInfo SessionInfo
	err = json.Unmarshal(data, &sessionInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session info: %w", err)
	}

	logrus.Infof("[SESSION] Session restored from backup: %s", latestFile)
	return &sessionInfo, nil
}

// GetSessionHealth returns the current session health status
func (sm *SessionManager) GetSessionHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"timestamp":        time.Now(),
		"backup_enabled":   config.SessionBackupEnabled,
		"auto_restore":     config.SessionAutoRestore,
		"backup_interval":  config.SessionBackupInterval,
		"backup_retention": config.SessionBackupRetention,
	}

	// Check if backup directory exists and is writable
	if _, err := os.Stat(sm.backupPath); err != nil {
		health["backup_directory_status"] = "error: " + err.Error()
	} else {
		health["backup_directory_status"] = "ok"
	}

	// Count backup files
	pattern := filepath.Join(sm.backupPath, "session_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		health["backup_count"] = "error: " + err.Error()
	} else {
		health["backup_count"] = len(matches)
	}

	return health
}

// cleanupOldBackups removes backup files older than retention period
func (sm *SessionManager) cleanupOldBackups() {
	if config.SessionBackupRetention <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -config.SessionBackupRetention)
	pattern := filepath.Join(sm.backupPath, "session_*.json")
	
	matches, err := filepath.Glob(pattern)
	if err != nil {
		logrus.Errorf("[SESSION] Failed to find backup files for cleanup: %v", err)
		return
	}

	cleaned := 0
	for _, file := range matches {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				logrus.Errorf("[SESSION] Failed to remove old backup %s: %v", file, err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		logrus.Infof("[SESSION] Cleaned up %d old backup files", cleaned)
	}
}

// StartPeriodicBackup starts a goroutine that performs periodic session backups
func (sm *SessionManager) StartPeriodicBackup(ctx context.Context, getSessionInfo func() SessionInfo) {
	if !config.SessionBackupEnabled || config.SessionBackupInterval <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(config.SessionBackupInterval) * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logrus.Info("[SESSION] Stopping periodic backup due to context cancellation")
				return
			case <-ticker.C:
				sessionInfo := getSessionInfo()
				if sessionInfo.IsLoggedIn {
					if err := sm.BackupSession(ctx, sessionInfo); err != nil {
						logrus.Errorf("[SESSION] Periodic backup failed: %v", err)
					}
				}
			}
		}
	}()

	logrus.Infof("[SESSION] Started periodic backup every %d seconds", config.SessionBackupInterval)
}
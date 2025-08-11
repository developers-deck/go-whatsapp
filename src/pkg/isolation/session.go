package isolation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type SessionIsolationManager struct {
	sessions map[string]*IsolatedSession
	mutex    sync.RWMutex
	basePath string
}

type IsolatedSession struct {
	InstanceID    string                 `json:"instance_id"`
	SessionPath   string                 `json:"session_path"`
	QRCodePath    string                 `json:"qr_code_path"`
	MediaPath     string                 `json:"media_path"`
	StaticPath    string                 `json:"static_path"`
	SessionData   map[string]interface{} `json:"session_data"`
	LastUpdated   time.Time              `json:"last_updated"`
	IsActive      bool                   `json:"is_active"`
	ConnectionID  string                 `json:"connection_id"`
	DeviceInfo    DeviceInfo             `json:"device_info"`
	mutex         sync.RWMutex           `json:"-"`
}

type DeviceInfo struct {
	DeviceID    string    `json:"device_id"`
	Platform    string    `json:"platform"`
	AppVersion  string    `json:"app_version"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`
}

func NewSessionIsolationManager(basePath string) *SessionIsolationManager {
	return &SessionIsolationManager{
		sessions: make(map[string]*IsolatedSession),
		basePath: basePath,
	}
}

// CreateIsolatedSession creates an isolated session for an instance
func (sim *SessionIsolationManager) CreateIsolatedSession(instanceID string) (*IsolatedSession, error) {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	if _, exists := sim.sessions[instanceID]; exists {
		return nil, fmt.Errorf("session for instance %s already exists", instanceID)
	}

	// Create session directories
	sessionDir := filepath.Join(sim.basePath, "instances", instanceID)
	qrCodeDir := filepath.Join(sessionDir, "statics", "qrcode")
	mediaDir := filepath.Join(sessionDir, "statics", "media")
	staticDir := filepath.Join(sessionDir, "statics")

	dirs := []string{sessionDir, qrCodeDir, mediaDir, staticDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	session := &IsolatedSession{
		InstanceID:   instanceID,
		SessionPath:  filepath.Join(sessionDir, "session.json"),
		QRCodePath:   qrCodeDir,
		MediaPath:    mediaDir,
		StaticPath:   staticDir,
		SessionData:  make(map[string]interface{}),
		LastUpdated:  time.Now(),
		IsActive:     false,
		ConnectionID: "",
		DeviceInfo:   DeviceInfo{},
	}

	// Save initial session data
	if err := sim.saveSessionData(session); err != nil {
		return nil, fmt.Errorf("failed to save initial session data: %w", err)
	}

	sim.sessions[instanceID] = session
	logrus.Infof("[SESSION_ISOLATION] Created isolated session for instance: %s", instanceID)
	return session, nil
}

// GetIsolatedSession retrieves the isolated session for an instance
func (sim *SessionIsolationManager) GetIsolatedSession(instanceID string) (*IsolatedSession, error) {
	sim.mutex.RLock()
	defer sim.mutex.RUnlock()

	session, exists := sim.sessions[instanceID]
	if !exists {
		return nil, fmt.Errorf("session for instance %s not found", instanceID)
	}

	return session, nil
}

// UpdateSessionData updates session data for an instance
func (sim *SessionIsolationManager) UpdateSessionData(instanceID string, data map[string]interface{}) error {
	session, err := sim.GetIsolatedSession(instanceID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// Update session data
	for key, value := range data {
		session.SessionData[key] = value
	}
	session.LastUpdated = time.Now()

	// Save updated session data
	return sim.saveSessionData(session)
}

// SetSessionActive marks a session as active/inactive
func (sim *SessionIsolationManager) SetSessionActive(instanceID string, active bool, connectionID string) error {
	session, err := sim.GetIsolatedSession(instanceID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.IsActive = active
	session.ConnectionID = connectionID
	session.LastUpdated = time.Now()

	if active {
		session.DeviceInfo.ConnectedAt = time.Now()
	}
	session.DeviceInfo.LastSeen = time.Now()

	return sim.saveSessionData(session)
}

// UpdateDeviceInfo updates device information for a session
func (sim *SessionIsolationManager) UpdateDeviceInfo(instanceID string, deviceInfo DeviceInfo) error {
	session, err := sim.GetIsolatedSession(instanceID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.DeviceInfo = deviceInfo
	session.LastUpdated = time.Now()

	return sim.saveSessionData(session)
}

// BackupSession creates a backup of the session data
func (sim *SessionIsolationManager) BackupSession(instanceID, backupPath string) error {
	session, err := sim.GetIsolatedSession(instanceID)
	if err != nil {
		return err
	}

	session.mutex.RLock()
	defer session.mutex.RUnlock()

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup session data
	sessionBackupPath := filepath.Join(backupPath, "session.json")
	if err := copyFile(session.SessionPath, sessionBackupPath); err != nil {
		return fmt.Errorf("failed to backup session data: %w", err)
	}

	// Backup static files
	staticBackupPath := filepath.Join(backupPath, "statics")
	if err := copyDir(session.StaticPath, staticBackupPath); err != nil {
		return fmt.Errorf("failed to backup static files: %w", err)
	}

	logrus.Infof("[SESSION_ISOLATION] Backed up session for instance: %s", instanceID)
	return nil
}

// RestoreSession restores session data from backup
func (sim *SessionIsolationManager) RestoreSession(instanceID, backupPath string) error {
	session, err := sim.GetIsolatedSession(instanceID)
	if err != nil {
		return err
	}

	session.mutex.Lock()
	defer session.mutex.Unlock()

	// Restore session data
	sessionBackupPath := filepath.Join(backupPath, "session.json")
	if err := copyFile(sessionBackupPath, session.SessionPath); err != nil {
		return fmt.Errorf("failed to restore session data: %w", err)
	}

	// Restore static files
	staticBackupPath := filepath.Join(backupPath, "statics")
	if err := copyDir(staticBackupPath, session.StaticPath); err != nil {
		return fmt.Errorf("failed to restore static files: %w", err)
	}

	// Reload session data
	if err := sim.loadSessionData(session); err != nil {
		return fmt.Errorf("failed to reload session data: %w", err)
	}

	logrus.Infof("[SESSION_ISOLATION] Restored session for instance: %s", instanceID)
	return nil
}

// DeleteIsolatedSession removes the isolated session for an instance
func (sim *SessionIsolationManager) DeleteIsolatedSession(instanceID string) error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	session, exists := sim.sessions[instanceID]
	if !exists {
		return fmt.Errorf("session for instance %s not found", instanceID)
	}

	// Remove session files
	sessionDir := filepath.Dir(session.SessionPath)
	if err := os.RemoveAll(sessionDir); err != nil {
		logrus.Warnf("[SESSION_ISOLATION] Failed to remove session directory: %v", err)
	}

	delete(sim.sessions, instanceID)
	logrus.Infof("[SESSION_ISOLATION] Deleted isolated session for instance: %s", instanceID)
	return nil
}

// ListSessions returns all isolated sessions
func (sim *SessionIsolationManager) ListSessions() []*IsolatedSession {
	sim.mutex.RLock()
	defer sim.mutex.RUnlock()

	sessions := make([]*IsolatedSession, 0, len(sim.sessions))
	for _, session := range sim.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetActiveSessionsCount returns the number of active sessions
func (sim *SessionIsolationManager) GetActiveSessionsCount() int {
	sim.mutex.RLock()
	defer sim.mutex.RUnlock()

	count := 0
	for _, session := range sim.sessions {
		if session.IsActive {
			count++
		}
	}

	return count
}

// CleanupInactiveSessions removes sessions that have been inactive for too long
func (sim *SessionIsolationManager) CleanupInactiveSessions(maxInactiveTime time.Duration) error {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	for instanceID, session := range sim.sessions {
		if !session.IsActive && now.Sub(session.LastUpdated) > maxInactiveTime {
			toDelete = append(toDelete, instanceID)
		}
	}

	for _, instanceID := range toDelete {
		if err := sim.DeleteIsolatedSession(instanceID); err != nil {
			logrus.Errorf("[SESSION_ISOLATION] Failed to cleanup session %s: %v", instanceID, err)
		}
	}

	if len(toDelete) > 0 {
		logrus.Infof("[SESSION_ISOLATION] Cleaned up %d inactive sessions", len(toDelete))
	}

	return nil
}

// Private methods

func (sim *SessionIsolationManager) saveSessionData(session *IsolatedSession) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	return os.WriteFile(session.SessionPath, data, 0644)
}

func (sim *SessionIsolationManager) loadSessionData(session *IsolatedSession) error {
	data, err := os.ReadFile(session.SessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create empty session
			return sim.saveSessionData(session)
		}
		return fmt.Errorf("failed to read session data: %w", err)
	}

	return json.Unmarshal(data, session)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// Stop gracefully stops the session isolation manager
func (sim *SessionIsolationManager) Stop() {
	sim.mutex.Lock()
	defer sim.mutex.Unlock()

	// Save all session data before stopping
	for instanceID, session := range sim.sessions {
		if err := sim.saveSessionData(session); err != nil {
			logrus.Errorf("[SESSION_ISOLATION] Failed to save session data for %s: %v", instanceID, err)
		}
	}

	logrus.Info("[SESSION_ISOLATION] Session isolation manager stopped")
}
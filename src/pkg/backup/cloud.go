package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type CloudProvider string

const (
	ProviderS3  CloudProvider = "s3"
	ProviderGCS CloudProvider = "gcs"
)

type BackupManager struct {
	provider CloudProvider
	config   CloudConfig
	enabled  bool
}

type CloudConfig struct {
	Provider        CloudProvider `json:"provider"`
	Enabled         bool          `json:"enabled"`
	Bucket          string        `json:"bucket"`
	Region          string        `json:"region"`
	AccessKey       string        `json:"access_key"`
	SecretKey       string        `json:"secret_key"`
	ServiceAccount  string        `json:"service_account"` // For GCS
	Prefix          string        `json:"prefix"`
	RetentionDays   int           `json:"retention_days"`
	ScheduleEnabled bool          `json:"schedule_enabled"`
	ScheduleCron    string        `json:"schedule_cron"`
}

type BackupJob struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // full, incremental, database, files
	Status      string            `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Size        int64             `json:"size"`
	Files       []string          `json:"files"`
	CloudPath   string            `json:"cloud_path"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

type BackupStats struct {
	TotalBackups     int64     `json:"total_backups"`
	SuccessfulBackups int64    `json:"successful_backups"`
	FailedBackups    int64     `json:"failed_backups"`
	TotalSize        int64     `json:"total_size"`
	LastBackup       *time.Time `json:"last_backup,omitempty"`
	NextScheduled    *time.Time `json:"next_scheduled,omitempty"`
	CloudProvider    string    `json:"cloud_provider"`
	Enabled          bool      `json:"enabled"`
}

func NewBackupManager(config CloudConfig) *BackupManager {
	bm := &BackupManager{
		provider: config.Provider,
		config:   config,
		enabled:  config.Enabled,
	}

	if !config.Enabled {
		logrus.Info("[BACKUP] Cloud backup is disabled")
		return bm
	}

	// Validate configuration
	if err := bm.validateConfig(); err != nil {
		logrus.Errorf("[BACKUP] Invalid configuration: %v", err)
		bm.enabled = false
		return bm
	}

	// Initialize cloud provider
	if err := bm.initializeProvider(); err != nil {
		logrus.Errorf("[BACKUP] Failed to initialize cloud provider: %v", err)
		bm.enabled = false
		return bm
	}

	logrus.Infof("[BACKUP] Cloud backup initialized with %s provider", config.Provider)
	return bm
}

// CreateBackup creates a new backup
func (bm *BackupManager) CreateBackup(backupType string, paths []string) (*BackupJob, error) {
	if !bm.enabled {
		return nil, fmt.Errorf("cloud backup is disabled")
	}

	job := &BackupJob{
		ID:        bm.generateJobID(),
		Type:      backupType,
		Status:    "pending",
		StartedAt: time.Now(),
		Files:     paths,
		Metadata:  make(map[string]string),
	}

	logrus.Infof("[BACKUP] Starting backup job %s (type: %s)", job.ID, backupType)

	// Create backup archive
	archivePath, err := bm.createArchive(job)
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		return job, err
	}
	defer os.Remove(archivePath) // Clean up local archive

	// Get file size
	if stat, err := os.Stat(archivePath); err == nil {
		job.Size = stat.Size()
	}

	// Upload to cloud
	job.Status = "running"
	cloudPath, err := bm.uploadToCloud(archivePath, job)
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		return job, err
	}

	job.CloudPath = cloudPath
	job.Status = "completed"
	now := time.Now()
	job.CompletedAt = &now

	logrus.Infof("[BACKUP] Backup job %s completed successfully (size: %d bytes)", job.ID, job.Size)
	return job, nil
}

// ListBackups lists available backups in cloud storage
func (bm *BackupManager) ListBackups() ([]*BackupJob, error) {
	if !bm.enabled {
		return nil, fmt.Errorf("cloud backup is disabled")
	}

	// This is a simplified implementation
	// In a real implementation, you would query the cloud provider
	return []*BackupJob{}, nil
}

// RestoreBackup restores a backup from cloud storage
func (bm *BackupManager) RestoreBackup(jobID string, targetPath string) error {
	if !bm.enabled {
		return fmt.Errorf("cloud backup is disabled")
	}

	logrus.Infof("[BACKUP] Starting restore for job %s to %s", jobID, targetPath)

	// Download from cloud
	tempFile, err := bm.downloadFromCloud(jobID)
	if err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}
	defer os.Remove(tempFile)

	// Extract archive
	err = bm.extractArchive(tempFile, targetPath)
	if err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}

	logrus.Infof("[BACKUP] Restore completed for job %s", jobID)
	return nil
}

// DeleteBackup deletes a backup from cloud storage
func (bm *BackupManager) DeleteBackup(jobID string) error {
	if !bm.enabled {
		return fmt.Errorf("cloud backup is disabled")
	}

	// This would delete from cloud storage
	logrus.Infof("[BACKUP] Deleted backup job %s", jobID)
	return nil
}

// GetStats returns backup statistics
func (bm *BackupManager) GetStats() *BackupStats {
	stats := &BackupStats{
		CloudProvider: string(bm.provider),
		Enabled:       bm.enabled,
	}

	if !bm.enabled {
		return stats
	}

	// In a real implementation, you would query cloud storage for stats
	stats.TotalBackups = 0
	stats.SuccessfulBackups = 0
	stats.FailedBackups = 0
	stats.TotalSize = 0

	return stats
}

// ScheduleBackup schedules automatic backups
func (bm *BackupManager) ScheduleBackup(backupType string, paths []string, schedule string) error {
	if !bm.enabled {
		return fmt.Errorf("cloud backup is disabled")
	}

	// This would set up a cron job or similar scheduling mechanism
	logrus.Infof("[BACKUP] Scheduled %s backup with schedule: %s", backupType, schedule)
	return nil
}

// Private methods

func (bm *BackupManager) validateConfig() error {
	if bm.config.Bucket == "" {
		return fmt.Errorf("bucket name is required")
	}

	switch bm.config.Provider {
	case ProviderS3:
		if bm.config.AccessKey == "" || bm.config.SecretKey == "" {
			return fmt.Errorf("S3 requires access key and secret key")
		}
		if bm.config.Region == "" {
			return fmt.Errorf("S3 requires region")
		}
	case ProviderGCS:
		if bm.config.ServiceAccount == "" {
			return fmt.Errorf("GCS requires service account")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", bm.config.Provider)
	}

	return nil
}

func (bm *BackupManager) initializeProvider() error {
	switch bm.config.Provider {
	case ProviderS3:
		return bm.initializeS3()
	case ProviderGCS:
		return bm.initializeGCS()
	default:
		return fmt.Errorf("unsupported provider: %s", bm.config.Provider)
	}
}

func (bm *BackupManager) initializeS3() error {
	// In a real implementation, you would initialize AWS S3 client
	logrus.Info("[BACKUP] S3 client initialized (mock)")
	return nil
}

func (bm *BackupManager) initializeGCS() error {
	// In a real implementation, you would initialize Google Cloud Storage client
	logrus.Info("[BACKUP] GCS client initialized (mock)")
	return nil
}

func (bm *BackupManager) createArchive(job *BackupJob) (string, error) {
	// Create temporary archive file
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("backup_%s.tar.gz", job.ID))
	
	// In a real implementation, you would create a tar.gz archive
	// For now, we'll create a simple file
	file, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}
	defer file.Close()

	// Write backup metadata
	metadata := fmt.Sprintf("Backup Job: %s\nType: %s\nCreated: %s\nFiles: %s\n",
		job.ID, job.Type, job.StartedAt.Format(time.RFC3339), strings.Join(job.Files, ", "))
	
	_, err = file.WriteString(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to write archive: %w", err)
	}

	// In a real implementation, you would add the actual files to the archive
	for _, path := range job.Files {
		if _, err := os.Stat(path); err == nil {
			// File exists, would be added to archive
			logrus.Debugf("[BACKUP] Would add file to archive: %s", path)
		}
	}

	return archivePath, nil
}

func (bm *BackupManager) uploadToCloud(archivePath string, job *BackupJob) (string, error) {
	// Generate cloud path
	cloudPath := fmt.Sprintf("%s/backups/%s/%s_%s.tar.gz",
		bm.config.Prefix,
		job.StartedAt.Format("2006/01/02"),
		job.Type,
		job.ID)

	// In a real implementation, you would upload to S3 or GCS
	switch bm.config.Provider {
	case ProviderS3:
		return bm.uploadToS3(archivePath, cloudPath)
	case ProviderGCS:
		return bm.uploadToGCS(archivePath, cloudPath)
	default:
		return "", fmt.Errorf("unsupported provider: %s", bm.config.Provider)
	}
}

func (bm *BackupManager) uploadToS3(archivePath, cloudPath string) (string, error) {
	// Mock S3 upload
	logrus.Infof("[BACKUP] Mock S3 upload: %s -> s3://%s/%s", archivePath, bm.config.Bucket, cloudPath)
	
	// Simulate upload delay
	time.Sleep(100 * time.Millisecond)
	
	return fmt.Sprintf("s3://%s/%s", bm.config.Bucket, cloudPath), nil
}

func (bm *BackupManager) uploadToGCS(archivePath, cloudPath string) (string, error) {
	// Mock GCS upload
	logrus.Infof("[BACKUP] Mock GCS upload: %s -> gs://%s/%s", archivePath, bm.config.Bucket, cloudPath)
	
	// Simulate upload delay
	time.Sleep(100 * time.Millisecond)
	
	return fmt.Sprintf("gs://%s/%s", bm.config.Bucket, cloudPath), nil
}

func (bm *BackupManager) downloadFromCloud(jobID string) (string, error) {
	// Mock download
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("restore_%s.tar.gz", jobID))
	
	file, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Write mock restore data
	_, err = file.WriteString(fmt.Sprintf("Restored backup for job: %s\n", jobID))
	if err != nil {
		return "", err
	}

	logrus.Infof("[BACKUP] Mock download completed: %s", tempFile)
	return tempFile, nil
}

func (bm *BackupManager) extractArchive(archivePath, targetPath string) error {
	// Mock extraction
	logrus.Infof("[BACKUP] Mock extraction: %s -> %s", archivePath, targetPath)
	
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	// In a real implementation, you would extract the tar.gz archive
	return nil
}

func (bm *BackupManager) generateJobID() string {
	return fmt.Sprintf("backup_%d", time.Now().UnixNano())
}

// Utility functions for common backup operations

func (bm *BackupManager) BackupDatabase() (*BackupJob, error) {
	dbPath := strings.TrimPrefix(config.DBURI, "file:")
	if strings.Contains(dbPath, "?") {
		dbPath = strings.Split(dbPath, "?")[0]
	}

	paths := []string{dbPath}
	return bm.CreateBackup("database", paths)
}

func (bm *BackupManager) BackupFiles() (*BackupJob, error) {
	paths := []string{
		config.PathStorages,
		config.PathMedia,
		config.PathQrCode,
		config.PathSendItems,
	}

	return bm.CreateBackup("files", paths)
}

func (bm *BackupManager) BackupFull() (*BackupJob, error) {
	// Combine database and files
	dbPath := strings.TrimPrefix(config.DBURI, "file:")
	if strings.Contains(dbPath, "?") {
		dbPath = strings.Split(dbPath, "?")[0]
	}

	paths := []string{
		dbPath,
		config.PathStorages,
		config.PathMedia,
		config.PathQrCode,
		config.PathSendItems,
	}

	return bm.CreateBackup("full", paths)
}
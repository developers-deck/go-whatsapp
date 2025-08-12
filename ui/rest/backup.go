package rest

import (
	"strconv"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/backup"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

type Backup struct {
	manager *backup.BackupManager
}

func InitRestBackup(app fiber.Router) Backup {
	// Load backup configuration from environment variables
	backupConfig := loadBackupConfig()

	bm := backup.NewBackupManager(backupConfig)
	rest := Backup{manager: bm}

	// Backup management routes
	app.Post("/backup/create", rest.CreateBackup)
	app.Post("/backup/database", rest.BackupDatabase)
	app.Post("/backup/files", rest.BackupFiles)
	app.Post("/backup/full", rest.BackupFull)
	app.Get("/backup/list", rest.ListBackups)
	app.Post("/backup/restore/:jobId", rest.RestoreBackup)
	app.Delete("/backup/:jobId", rest.DeleteBackup)
	app.Post("/backup/schedule", rest.ScheduleBackup)
	app.Get("/backup/stats", rest.GetStats)
	app.Get("/backup/config", rest.GetConfig)

	return rest
}

func loadBackupConfig() backup.CloudConfig {
	// Default configuration
	config := backup.CloudConfig{
		Provider:        backup.ProviderB2,
		Enabled:         true, // Enable by default
		Bucket:          "whatsapp-backups",
		Region:          "us-east-1",
		AccessKey:       "",
		SecretKey:       "",
		ApplicationKey:  "",
		KeyID:           "",
		ServiceAccount:  "",
		Prefix:          "whatsapp-backups",
		RetentionDays:   30,
		ScheduleEnabled: true,
		ScheduleCron:    "0 2 * * *", // Daily at 2 AM
	}

	// Load from environment variables
	if enabled := viper.GetString("BACKUP_ENABLED"); enabled != "" {
		config.Enabled = strings.ToLower(enabled) == "true"
	}

	if provider := viper.GetString("BACKUP_PROVIDER"); provider != "" {
		switch strings.ToLower(provider) {
		case "b2":
			config.Provider = backup.ProviderB2
		case "gcs":
			config.Provider = backup.ProviderGCS
		}
	}

	// B2 specific configuration
	if keyID := viper.GetString("B2_KEY_ID"); keyID != "" {
		config.KeyID = keyID
	}
	if appKey := viper.GetString("B2_APPLICATION_KEY"); appKey != "" {
		config.ApplicationKey = appKey
	}
	if bucket := viper.GetString("BACKUP_BUCKET"); bucket != "" {
		config.Bucket = bucket
	}
	if region := viper.GetString("BACKUP_REGION"); region != "" {
		config.Region = region
	}
	if prefix := viper.GetString("BACKUP_PREFIX"); prefix != "" {
		config.Prefix = prefix
	}

	// Schedule configuration
	if schedule := viper.GetString("BACKUP_SCHEDULE_CRON"); schedule != "" {
		config.ScheduleEnabled = true
		config.ScheduleCron = schedule
	} else if schedule := viper.GetString("BACKUP_SCHEDULE"); schedule != "" {
		config.ScheduleEnabled = true
		switch strings.ToLower(schedule) {
		case "daily":
			config.ScheduleCron = "0 2 * * *" // Daily at 2 AM
		case "weekly":
			config.ScheduleCron = "0 2 * * 0" // Weekly on Sunday at 2 AM
		case "monthly":
			config.ScheduleCron = "0 2 1 * *" // Monthly on 1st at 2 AM
		default:
			// Assume it's a custom cron expression
			config.ScheduleCron = schedule
		}
	}

	// Retention days
	if retention := viper.GetString("BACKUP_RETENTION_DAYS"); retention != "" {
		if days, err := strconv.Atoi(retention); err == nil {
			config.RetentionDays = days
		}
	}

	return config
}

func (handler *Backup) CreateBackup(c *fiber.Ctx) error {
	var request struct {
		Type  string   `json:"type"`
		Paths []string `json:"paths"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Type == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Backup type is required",
			Results: nil,
		})
	}

	if len(request.Paths) == 0 {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "At least one path is required",
			Results: nil,
		})
	}

	job, err := handler.manager.CreateBackup(request.Type, request.Paths)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "BACKUP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Backup created successfully",
		Results: job,
	})
}

func (handler *Backup) BackupDatabase(c *fiber.Ctx) error {
	job, err := handler.manager.BackupDatabase()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "BACKUP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Database backup created successfully",
		Results: job,
	})
}

func (handler *Backup) BackupFiles(c *fiber.Ctx) error {
	job, err := handler.manager.BackupFiles()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "BACKUP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Files backup created successfully",
		Results: job,
	})
}

func (handler *Backup) BackupFull(c *fiber.Ctx) error {
	job, err := handler.manager.BackupFull()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "BACKUP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Full backup created successfully",
		Results: job,
	})
}

func (handler *Backup) ListBackups(c *fiber.Ctx) error {
	backups, err := handler.manager.ListBackups()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "BACKUP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Backups listed successfully",
		Results: map[string]interface{}{
			"backups": backups,
			"count":   len(backups),
		},
	})
}

func (handler *Backup) RestoreBackup(c *fiber.Ctx) error {
	jobID := c.Params("jobId")
	if jobID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Job ID is required",
			Results: nil,
		})
	}

	var request struct {
		TargetPath string `json:"target_path"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.TargetPath == "" {
		request.TargetPath = "./restored" // Default restore path
	}

	err := handler.manager.RestoreBackup(jobID, request.TargetPath)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "RESTORE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Backup restored successfully",
		Results: map[string]interface{}{
			"job_id":      jobID,
			"target_path": request.TargetPath,
		},
	})
}

func (handler *Backup) DeleteBackup(c *fiber.Ctx) error {
	jobID := c.Params("jobId")
	if jobID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Job ID is required",
			Results: nil,
		})
	}

	err := handler.manager.DeleteBackup(jobID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "DELETE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Backup deleted successfully",
		Results: map[string]interface{}{
			"job_id": jobID,
		},
	})
}

func (handler *Backup) ScheduleBackup(c *fiber.Ctx) error {
	var request struct {
		Type     string   `json:"type"`
		Paths    []string `json:"paths"`
		Schedule string   `json:"schedule"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Type == "" || request.Schedule == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Type and schedule are required",
			Results: nil,
		})
	}

	err := handler.manager.ScheduleBackup(request.Type, request.Paths, request.Schedule)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "SCHEDULE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Backup scheduled successfully",
		Results: map[string]interface{}{
			"type":     request.Type,
			"schedule": request.Schedule,
		},
	})
}

func (handler *Backup) GetStats(c *fiber.Ctx) error {
	stats := handler.manager.GetStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Backup statistics retrieved",
		Results: stats,
	})
}

func (handler *Backup) GetConfig(c *fiber.Ctx) error {
	config := handler.manager.GetConfig()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Backup configuration retrieved",
		Results: config,
	})
}
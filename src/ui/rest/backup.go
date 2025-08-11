package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/backup"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Backup struct {
	manager *backup.BackupManager
}

func InitRestBackup(app fiber.Router) Backup {
	// Initialize backup manager with default config
	backupConfig := backup.CloudConfig{
		Provider:        backup.ProviderS3, // Default to S3
		Enabled:         false,             // Disabled by default
		Bucket:          "",
		Region:          "us-east-1",
		AccessKey:       "",
		SecretKey:       "",
		ServiceAccount:  "",
		Prefix:          "whatsapp-backups",
		RetentionDays:   30,
		ScheduleEnabled: false,
		ScheduleCron:    "0 2 * * *", // Daily at 2 AM
	}

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

	return rest
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
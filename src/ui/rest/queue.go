package rest

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/queue"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Queue struct {
	manager *queue.QueueManager
}

func InitRestQueue(app fiber.Router) Queue {
	qm := queue.NewQueueManager()
	rest := Queue{manager: qm}

	// Register default handlers
	rest.registerDefaultHandlers()

	// Queue management routes
	app.Post("/queue/jobs", rest.AddJob)
	app.Post("/queue/jobs/schedule", rest.ScheduleJob)
	app.Get("/queue/jobs", rest.ListJobs)
	app.Get("/queue/jobs/:id", rest.GetJob)
	app.Delete("/queue/jobs/:id", rest.CancelJob)
	app.Get("/queue/stats", rest.GetStats)
	app.Post("/queue/handlers/:type", rest.RegisterHandler)

	return rest
}

func (handler *Queue) registerDefaultHandlers() {
	// Register message sending handler
	handler.manager.RegisterHandler("send_message", handler.handleSendMessage)
	handler.manager.RegisterHandler("send_media", handler.handleSendMedia)
	handler.manager.RegisterHandler("send_bulk", handler.handleSendBulk)
	handler.manager.RegisterHandler("cleanup", handler.handleCleanup)
	handler.manager.RegisterHandler("backup", handler.handleBackup)
}

func (handler *Queue) AddJob(c *fiber.Ctx) error {
	var request struct {
		Type     string                 `json:"type"`
		Data     map[string]interface{} `json:"data"`
		Priority int                    `json:"priority"`
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
			Message: "Job type is required",
			Results: nil,
		})
	}

	// Validate priority
	priority := queue.Priority(request.Priority)
	if priority < queue.PriorityLow || priority > queue.PriorityUrgent {
		priority = queue.PriorityNormal
	}

	job, err := handler.manager.AddJob(request.Type, request.Data, priority)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "QUEUE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Job added to queue successfully",
		Results: job,
	})
}

func (handler *Queue) ScheduleJob(c *fiber.Ctx) error {
	var request struct {
		Type        string                 `json:"type"`
		Data        map[string]interface{} `json:"data"`
		Priority    int                    `json:"priority"`
		ScheduledAt string                 `json:"scheduled_at"` // RFC3339 format
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Type == "" || request.ScheduledAt == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Job type and scheduled_at are required",
			Results: nil,
		})
	}

	// Parse scheduled time
	scheduledAt, err := time.Parse(time.RFC3339, request.ScheduledAt)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid scheduled_at format. Use RFC3339 format",
			Results: nil,
		})
	}

	// Validate priority
	priority := queue.Priority(request.Priority)
	if priority < queue.PriorityLow || priority > queue.PriorityUrgent {
		priority = queue.PriorityNormal
	}

	job, err := handler.manager.ScheduleJob(request.Type, request.Data, priority, scheduledAt)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "QUEUE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Job scheduled successfully",
		Results: job,
	})
}

func (handler *Queue) ListJobs(c *fiber.Ctx) error {
	status := queue.JobStatus(c.Query("status", ""))
	jobType := c.Query("type", "")
	limitStr := c.Query("limit", "50")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	jobs := handler.manager.ListJobs(status, jobType, limit)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Jobs retrieved successfully",
		Results: map[string]interface{}{
			"jobs":   jobs,
			"count":  len(jobs),
			"status": status,
			"type":   jobType,
		},
	})
}

func (handler *Queue) GetJob(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Job ID is required",
			Results: nil,
		})
	}

	job, err := handler.manager.GetJob(jobID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Job retrieved successfully",
		Results: job,
	})
}

func (handler *Queue) CancelJob(c *fiber.Ctx) error {
	jobID := c.Params("id")
	if jobID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Job ID is required",
			Results: nil,
		})
	}

	err := handler.manager.CancelJob(jobID)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "CANCEL_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Job cancelled successfully",
		Results: nil,
	})
}

func (handler *Queue) GetStats(c *fiber.Ctx) error {
	stats := handler.manager.GetQueueStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Queue statistics retrieved",
		Results: stats,
	})
}

func (handler *Queue) RegisterHandler(c *fiber.Ctx) error {
	jobType := c.Params("type")
	if jobType == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Job type is required",
			Results: nil,
		})
	}

	// This is a placeholder - in a real implementation, you'd register actual handlers
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Handler registration endpoint (placeholder)",
		Results: map[string]interface{}{
			"job_type": jobType,
			"message":  "Use the Go API to register actual handlers",
		},
	})
}

// Default job handlers

func (handler *Queue) handleSendMessage(ctx context.Context, job *queue.Job) error {
	// Extract message data
	phone, ok := job.Data["phone"].(string)
	if !ok {
		return fmt.Errorf("phone number is required")
	}

	message, ok := job.Data["message"].(string)
	if !ok {
		return fmt.Errorf("message content is required")
	}

	// Simulate message sending (replace with actual WhatsApp sending logic)
	time.Sleep(100 * time.Millisecond) // Simulate API call delay
	
	job.Result = map[string]interface{}{
		"message_id": fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		"phone":      phone,
		"message":    message,
		"sent_at":    time.Now(),
	}

	return nil
}

func (handler *Queue) handleSendMedia(ctx context.Context, job *queue.Job) error {
	// Extract media data
	phone, ok := job.Data["phone"].(string)
	if !ok {
		return fmt.Errorf("phone number is required")
	}

	mediaType, ok := job.Data["media_type"].(string)
	if !ok {
		return fmt.Errorf("media type is required")
	}

	mediaPath, ok := job.Data["media_path"].(string)
	if !ok {
		return fmt.Errorf("media path is required")
	}

	// Simulate media sending (replace with actual WhatsApp sending logic)
	time.Sleep(500 * time.Millisecond) // Simulate longer delay for media
	
	job.Result = map[string]interface{}{
		"message_id": fmt.Sprintf("media_%d", time.Now().UnixNano()),
		"phone":      phone,
		"media_type": mediaType,
		"media_path": mediaPath,
		"sent_at":    time.Now(),
	}

	return nil
}

func (handler *Queue) handleSendBulk(ctx context.Context, job *queue.Job) error {
	// Extract bulk data
	recipients, ok := job.Data["recipients"].([]interface{})
	if !ok {
		return fmt.Errorf("recipients list is required")
	}

	message, ok := job.Data["message"].(string)
	if !ok {
		return fmt.Errorf("message content is required")
	}

	// Simulate bulk sending
	results := make([]map[string]interface{}, 0)
	
	for _, recipient := range recipients {
		phone, ok := recipient.(string)
		if !ok {
			continue
		}

		// Simulate individual message sending
		time.Sleep(50 * time.Millisecond)
		
		results = append(results, map[string]interface{}{
			"message_id": fmt.Sprintf("bulk_%d", time.Now().UnixNano()),
			"phone":      phone,
			"status":     "sent",
			"sent_at":    time.Now(),
		})
	}

	job.Result = map[string]interface{}{
		"total_recipients": len(recipients),
		"successful_sends": len(results),
		"results":          results,
		"completed_at":     time.Now(),
	}

	return nil
}

func (handler *Queue) handleCleanup(ctx context.Context, job *queue.Job) error {
	// Simulate cleanup operations
	cleanupType, ok := job.Data["type"].(string)
	if !ok {
		cleanupType = "general"
	}

	time.Sleep(2 * time.Second) // Simulate cleanup time
	
	job.Result = map[string]interface{}{
		"cleanup_type":   cleanupType,
		"files_cleaned":  42,
		"space_freed":    "150MB",
		"completed_at":   time.Now(),
	}

	return nil
}

func (handler *Queue) handleBackup(ctx context.Context, job *queue.Job) error {
	// Simulate backup operations
	backupType, ok := job.Data["type"].(string)
	if !ok {
		backupType = "full"
	}

	time.Sleep(5 * time.Second) // Simulate backup time
	
	job.Result = map[string]interface{}{
		"backup_type":    backupType,
		"backup_size":    "2.5GB",
		"backup_path":    "/backups/backup_" + time.Now().Format("20060102_150405") + ".tar.gz",
		"completed_at":   time.Now(),
	}

	return nil
}
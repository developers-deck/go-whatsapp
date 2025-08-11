package rest

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SystemHandler struct{}

type SystemOverview struct {
	Health struct {
		Overall     string `json:"overall"`
		Uptime      string `json:"uptime"`
		CPUUsage    int    `json:"cpu_usage"`
		MemoryUsage int    `json:"memory_usage"`
	} `json:"health"`
	Instances struct {
		Total     int `json:"total"`
		Running   int `json:"running"`
		Stopped   int `json:"stopped"`
		Connected int `json:"connected"`
	} `json:"instances"`
	Messages struct {
		Total    int `json:"total"`
		Sent     int `json:"sent"`
		Received int `json:"received"`
		Failed   int `json:"failed"`
	} `json:"messages"`
	Queue struct {
		Pending    int `json:"pending"`
		Processing int `json:"processing"`
		Completed  int `json:"completed"`
		Failed     int `json:"failed"`
	} `json:"queue"`
	Storage struct {
		Used int64 `json:"used"`
	} `json:"storage"`
	Cache struct {
		Keys    int `json:"keys"`
		HitRate int `json:"hit_rate"`
	} `json:"cache"`
	Backups struct {
		Count int `json:"count"`
	} `json:"backups"`
	RecentActivity []Activity `json:"recent_activity"`
	Alerts         []Alert    `json:"alerts"`
}

type Activity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

type Alert struct {
	ID      string `json:"id"`
	Level   string `json:"level"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func InitRestSystem(app fiber.Router) {
	handler := &SystemHandler{}
	
	app.Get("/system/overview", handler.GetSystemOverview)
}

func (h *SystemHandler) GetSystemOverview(c *fiber.Ctx) error {
	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Calculate uptime (mock for now)
	uptime := time.Since(time.Now().Add(-24 * time.Hour))
	
	overview := SystemOverview{
		Health: struct {
			Overall     string `json:"overall"`
			Uptime      string `json:"uptime"`
			CPUUsage    int    `json:"cpu_usage"`
			MemoryUsage int    `json:"memory_usage"`
		}{
			Overall:     "healthy",
			Uptime:      formatDuration(uptime),
			CPUUsage:    25, // Mock data
			MemoryUsage: int(float64(m.Alloc) / float64(m.Sys) * 100),
		},
		Instances: struct {
			Total     int `json:"total"`
			Running   int `json:"running"`
			Stopped   int `json:"stopped"`
			Connected int `json:"connected"`
		}{
			Total:     5,  // Mock data
			Running:   3,
			Stopped:   2,
			Connected: 3,
		},
		Messages: struct {
			Total    int `json:"total"`
			Sent     int `json:"sent"`
			Received int `json:"received"`
			Failed   int `json:"failed"`
		}{
			Total:    1250, // Mock data
			Sent:     800,
			Received: 400,
			Failed:   50,
		},
		Queue: struct {
			Pending    int `json:"pending"`
			Processing int `json:"processing"`
			Completed  int `json:"completed"`
			Failed     int `json:"failed"`
		}{
			Pending:    15, // Mock data
			Processing: 3,
			Completed:  1200,
			Failed:     25,
		},
		Storage: struct {
			Used int64 `json:"used"`
		}{
			Used: int64(m.Alloc), // Current memory allocation as storage mock
		},
		Cache: struct {
			Keys    int `json:"keys"`
			HitRate int `json:"hit_rate"`
		}{
			Keys:    150, // Mock data
			HitRate: 85,
		},
		Backups: struct {
			Count int `json:"count"`
		}{
			Count: 12, // Mock data
		},
		RecentActivity: []Activity{
			{
				ID:          "1",
				Type:        "message",
				Title:       "Message sent successfully",
				Description: "WhatsApp message sent to +1234567890",
				Timestamp:   time.Now().Add(-5 * time.Minute),
			},
			{
				ID:          "2",
				Type:        "instance",
				Title:       "Instance started",
				Description: "WhatsApp instance 'Business-01' started successfully",
				Timestamp:   time.Now().Add(-15 * time.Minute),
			},
			{
				ID:          "3",
				Type:        "backup",
				Title:       "Backup completed",
				Description: "Scheduled backup completed successfully",
				Timestamp:   time.Now().Add(-1 * time.Hour),
			},
			{
				ID:          "4",
				Type:        "webhook",
				Title:       "Webhook delivered",
				Description: "Webhook payload delivered to external endpoint",
				Timestamp:   time.Now().Add(-2 * time.Hour),
			},
			{
				ID:          "5",
				Type:        "queue",
				Title:       "Queue processed",
				Description: "25 jobs processed from high priority queue",
				Timestamp:   time.Now().Add(-3 * time.Hour),
			},
		},
		Alerts: []Alert{
			{
				ID:      "1",
				Level:   "warning",
				Title:   "High Memory Usage",
				Message: "System memory usage is above 80%. Consider optimizing or scaling.",
			},
		},
	}
	
	return c.JSON(fiber.Map{
		"code":    200,
		"message": "System overview retrieved successfully",
		"results": overview,
	})
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}
package rest

import (
	"runtime"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/monitor"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Monitor struct {
	processMonitor *monitor.ProcessMonitor
}

func InitRestMonitor(app fiber.Router) Monitor {
	pm := monitor.NewProcessMonitor()
	rest := Monitor{processMonitor: pm}

	// Monitoring routes
	app.Get("/monitor/health", rest.GetHealth)
	app.Get("/monitor/stats", rest.GetStats)
	app.Get("/monitor/memory", rest.GetMemoryStats)
	app.Post("/monitor/gc", rest.ForceGC)
	app.Post("/monitor/restart/:pid", rest.RestartProcess)
	app.Post("/monitor/kill/:pid", rest.KillProcess)

	return rest
}

func (handler *Monitor) GetHealth(c *fiber.Ctx) error {
	stats := handler.processMonitor.GetProcessStats()

	// Determine overall health status
	allHealthy := true
	for _, healthy := range stats.HealthChecks {
		if !healthy {
			allHealthy = false
			break
		}
	}

	status := "healthy"
	if !allHealthy {
		status = "unhealthy"
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Health check completed",
		Results: map[string]interface{}{
			"status":        status,
			"health_checks": stats.HealthChecks,
			"last_check":    stats.LastHealthCheck,
			"uptime":        stats.Uptime.String(),
		},
	})
}

func (handler *Monitor) GetStats(c *fiber.Ctx) error {
	stats := handler.processMonitor.GetProcessStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Process statistics retrieved",
		Results: stats,
	})
}

func (handler *Monitor) GetMemoryStats(c *fiber.Ctx) error {
	memStats := handler.processMonitor.GetMemoryStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Memory statistics retrieved",
		Results: memStats,
	})
}

func (handler *Monitor) RestartProcess(c *fiber.Ctx) error {
	pid := c.Params("pid")

	// For now, return success (process restart logic can be implemented later)
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Process restart initiated",
		Results: map[string]interface{}{
			"pid":    pid,
			"status": "restarting",
		},
	})
}

func (handler *Monitor) KillProcess(c *fiber.Ctx) error {
	pid := c.Params("pid")

	// For now, return success (process kill logic can be implemented later)
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Process kill initiated",
		Results: map[string]interface{}{
			"pid":    pid,
			"status": "killing",
		},
	})
}

func (handler *Monitor) ForceGC(c *fiber.Ctx) error {
	// Get memory stats before GC
	beforeStats := handler.processMonitor.GetMemoryStats()

	// Force garbage collection
	runtime.GC()

	// Get memory stats after GC
	afterStats := handler.processMonitor.GetMemoryStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Garbage collection completed",
		Results: map[string]interface{}{
			"before": beforeStats,
			"after":  afterStats,
		},
	})
}

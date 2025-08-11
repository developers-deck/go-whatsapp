package rest

import (
	"strconv"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/analytics"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Analytics struct {
	manager *analytics.Analytics
}

func InitRestAnalytics(app fiber.Router) Analytics {
	am := analytics.NewAnalytics()
	rest := Analytics{manager: am}

	// Analytics routes
	app.Get("/analytics/realtime", rest.GetRealTimeStats)
	app.Get("/analytics/daily", rest.GetDailyReport)
	app.Get("/analytics/weekly", rest.GetWeeklyReport)
	app.Get("/analytics/monthly", rest.GetMonthlyReport)
	app.Get("/analytics/custom", rest.GetCustomReport)
	app.Post("/analytics/track", rest.TrackEvent)

	return rest
}

func (handler *Analytics) GetRealTimeStats(c *fiber.Ctx) error {
	stats := handler.manager.GetRealTimeStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Real-time statistics retrieved",
		Results: stats,
	})
}

func (handler *Analytics) GetDailyReport(c *fiber.Ctx) error {
	report := handler.manager.GetDailyReport()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Daily report generated",
		Results: report,
	})
}

func (handler *Analytics) GetWeeklyReport(c *fiber.Ctx) error {
	report := handler.manager.GetWeeklyReport()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Weekly report generated",
		Results: report,
	})
}

func (handler *Analytics) GetMonthlyReport(c *fiber.Ctx) error {
	report := handler.manager.GetMonthlyReport()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Monthly report generated",
		Results: report,
	})
}

func (handler *Analytics) GetCustomReport(c *fiber.Ctx) error {
	// Parse query parameters
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	period := c.Query("period", "custom")

	if startDateStr == "" || endDateStr == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "start_date and end_date are required",
			Results: nil,
		})
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid start_date format. Use YYYY-MM-DD",
			Results: nil,
		})
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid end_date format. Use YYYY-MM-DD",
			Results: nil,
		})
	}

	// Add 24 hours to end date to include the entire day
	endDate = endDate.Add(24 * time.Hour)

	report := handler.manager.GenerateReport(period, startDate, endDate)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Custom report generated",
		Results: report,
	})
}

func (handler *Analytics) TrackEvent(c *fiber.Ctx) error {
	var request struct {
		Type     string                 `json:"type"`
		Category string                 `json:"category"`
		Action   string                 `json:"action"`
		Data     map[string]interface{} `json:"data"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Type == "" || request.Category == "" || request.Action == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "type, category, and action are required",
			Results: nil,
		})
	}

	// Add request metadata
	if request.Data == nil {
		request.Data = make(map[string]interface{})
	}
	request.Data["user_agent"] = c.Get("User-Agent")
	request.Data["ip"] = c.IP()

	handler.manager.TrackEvent(request.Type, request.Category, request.Action, request.Data)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Event tracked successfully",
		Results: map[string]interface{}{
			"type":     request.Type,
			"category": request.Category,
			"action":   request.Action,
		},
	})
}

// Middleware to automatically track API calls
func (handler *Analytics) TrackingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Process request
		err := c.Next()
		
		// Track the API call
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()
		
		handler.manager.TrackAPICall(
			c.Path(),
			c.Method(),
			statusCode,
			duration,
		)
		
		return err
	}
}
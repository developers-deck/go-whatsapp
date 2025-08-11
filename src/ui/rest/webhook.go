package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/webhook"
	"github.com/gofiber/fiber/v2"
)

type Webhook struct {
	manager *webhook.WebhookManager
}

func InitRestWebhook(app fiber.Router) Webhook {
	wm := webhook.NewWebhookManager()
	rest := Webhook{manager: wm}

	// Webhook management routes
	app.Post("/webhooks", rest.AddEndpoint)
	app.Get("/webhooks", rest.ListEndpoints)
	app.Get("/webhooks/:id", rest.GetEndpoint)
	app.Put("/webhooks/:id", rest.UpdateEndpoint)
	app.Delete("/webhooks/:id", rest.RemoveEndpoint)
	app.Post("/webhooks/:id/test", rest.TestEndpoint)
	app.Post("/webhooks/send", rest.SendEvent)
	app.Get("/webhooks/stats", rest.GetStats)

	return rest
}

func (handler *Webhook) AddEndpoint(c *fiber.Ctx) error {
	var endpoint webhook.WebhookEndpoint

	if err := c.BodyParser(&endpoint); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if endpoint.URL == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Webhook URL is required",
			Results: nil,
		})
	}

	err := handler.manager.AddEndpoint(&endpoint)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "WEBHOOK_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Webhook endpoint added successfully",
		Results: endpoint,
	})
}

func (handler *Webhook) ListEndpoints(c *fiber.Ctx) error {
	endpoints := handler.manager.ListEndpoints()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Webhook endpoints retrieved successfully",
		Results: map[string]interface{}{
			"endpoints": endpoints,
			"count":     len(endpoints),
		},
	})
}

func (handler *Webhook) GetEndpoint(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Endpoint ID is required",
			Results: nil,
		})
	}

	endpoint, err := handler.manager.GetEndpoint(id)
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
		Message: "Webhook endpoint retrieved successfully",
		Results: endpoint,
	})
}

func (handler *Webhook) UpdateEndpoint(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Endpoint ID is required",
			Results: nil,
		})
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	err := handler.manager.UpdateEndpoint(id, updates)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "UPDATE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	// Get updated endpoint
	endpoint, _ := handler.manager.GetEndpoint(id)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Webhook endpoint updated successfully",
		Results: endpoint,
	})
}

func (handler *Webhook) RemoveEndpoint(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Endpoint ID is required",
			Results: nil,
		})
	}

	err := handler.manager.RemoveEndpoint(id)
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
		Message: "Webhook endpoint removed successfully",
		Results: nil,
	})
}

func (handler *Webhook) TestEndpoint(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Endpoint ID is required",
			Results: nil,
		})
	}

	delivery, err := handler.manager.TestEndpoint(id)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "TEST_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Webhook test completed",
		Results: delivery,
	})
}

func (handler *Webhook) SendEvent(c *fiber.Ctx) error {
	var event webhook.WebhookEvent

	if err := c.BodyParser(&event); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if event.Type == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Event type is required",
			Results: nil,
		})
	}

	err := handler.manager.SendEvent(&event)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "SEND_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Event sent to webhooks successfully",
		Results: map[string]interface{}{
			"event_id": event.ID,
			"type":     event.Type,
		},
	})
}

func (handler *Webhook) GetStats(c *fiber.Ctx) error {
	stats := handler.manager.GetStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Webhook statistics retrieved",
		Results: stats,
	})
}
package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/multiinstance"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type MultiInstance struct {
	manager *multiinstance.InstanceManager
}

func InitRestMultiInstance(app fiber.Router) MultiInstance {
	im := multiinstance.NewInstanceManager()
	rest := MultiInstance{manager: im}

	// Multi-instance management routes
	app.Post("/instances", rest.CreateInstance)
	app.Get("/instances", rest.ListInstances)
	app.Get("/instances/:id", rest.GetInstance)
	app.Post("/instances/:id/start", rest.StartInstance)
	app.Post("/instances/:id/stop", rest.StopInstance)
	app.Post("/instances/:id/restart", rest.RestartInstance)
	app.Delete("/instances/:id", rest.DeleteInstance)
	app.Get("/instances/stats", rest.GetStats)
	
	// Alias routes for compatibility with frontend
	app.Get("/multiinstance/list", rest.ListInstances)

	return rest
}

func (handler *MultiInstance) CreateInstance(c *fiber.Ctx) error {
	var request struct {
		Name   string                        `json:"name"`
		Phone  string                        `json:"phone"`
		Config multiinstance.InstanceConfig `json:"config"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Name == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Instance name is required",
			Results: nil,
		})
	}

	// Set default values if not provided
	if request.Config.Port == 0 {
		request.Config.Port = 0 // Will be auto-assigned
	}
	if request.Config.OS == "" {
		request.Config.OS = "Chrome"
	}
	if request.Config.Environment == nil {
		request.Config.Environment = make(map[string]string)
	}

	instance, err := handler.manager.CreateInstance(request.Name, request.Phone, request.Config)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CREATE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Instance created successfully",
		Results: instance,
	})
}

func (handler *MultiInstance) ListInstances(c *fiber.Ctx) error {
	instances := handler.manager.ListInstances()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Instances retrieved successfully",
		Results: map[string]interface{}{
			"instances": instances,
			"count":     len(instances),
		},
	})
}

func (handler *MultiInstance) GetInstance(c *fiber.Ctx) error {
	instanceID := c.Params("id")
	if instanceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Instance ID is required",
			Results: nil,
		})
	}

	instance, err := handler.manager.GetInstance(instanceID)
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
		Message: "Instance retrieved successfully",
		Results: instance,
	})
}

func (handler *MultiInstance) StartInstance(c *fiber.Ctx) error {
	instanceID := c.Params("id")
	if instanceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Instance ID is required",
			Results: nil,
		})
	}

	err := handler.manager.StartInstance(instanceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "START_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	// Get updated instance info
	instance, _ := handler.manager.GetInstance(instanceID)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Instance started successfully",
		Results: instance,
	})
}

func (handler *MultiInstance) StopInstance(c *fiber.Ctx) error {
	instanceID := c.Params("id")
	if instanceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Instance ID is required",
			Results: nil,
		})
	}

	err := handler.manager.StopInstance(instanceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "STOP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	// Get updated instance info
	instance, _ := handler.manager.GetInstance(instanceID)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Instance stopped successfully",
		Results: instance,
	})
}

func (handler *MultiInstance) RestartInstance(c *fiber.Ctx) error {
	instanceID := c.Params("id")
	if instanceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Instance ID is required",
			Results: nil,
		})
	}

	err := handler.manager.RestartInstance(instanceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "RESTART_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	// Get updated instance info
	instance, _ := handler.manager.GetInstance(instanceID)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Instance restarted successfully",
		Results: instance,
	})
}

func (handler *MultiInstance) DeleteInstance(c *fiber.Ctx) error {
	instanceID := c.Params("id")
	if instanceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Instance ID is required",
			Results: nil,
		})
	}

	err := handler.manager.DeleteInstance(instanceID)
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
		Message: "Instance deleted successfully",
		Results: map[string]interface{}{
			"instance_id": instanceID,
		},
	})
}

func (handler *MultiInstance) GetStats(c *fiber.Ctx) error {
	stats := handler.manager.GetStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Instance statistics retrieved",
		Results: stats,
	})
}
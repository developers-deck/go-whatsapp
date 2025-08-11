package rest

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/updater"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Updater struct {
	manager *updater.UpdateManager
}

func InitRestUpdater(app fiber.Router) Updater {
	um := updater.NewUpdateManager()
	rest := Updater{manager: um}

	// Update management routes
	app.Get("/updater/check", rest.CheckForUpdates)
	app.Post("/updater/update", rest.PerformUpdate)
	app.Get("/updater/history", rest.GetUpdateHistory)
	app.Get("/updater/version", rest.GetCurrentVersion)
	app.Post("/updater/channel", rest.SetUpdateChannel)
	app.Post("/updater/auto-update", rest.SetAutoUpdate)

	return rest
}

func (handler *Updater) CheckForUpdates(c *fiber.Ctx) error {
	updateInfo, err := handler.manager.CheckForUpdates()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "UPDATE_CHECK_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Update check completed",
		Results: updateInfo,
	})
}

func (handler *Updater) PerformUpdate(c *fiber.Ctx) error {
	// First check for updates
	updateInfo, err := handler.manager.CheckForUpdates()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "UPDATE_CHECK_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	if !updateInfo.Available {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "NO_UPDATE_AVAILABLE",
			Message: "No update available",
			Results: updateInfo,
		})
	}

	// Perform the update
	status, err := handler.manager.PerformUpdate(updateInfo)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "UPDATE_ERROR",
			Message: err.Error(),
			Results: status,
		})
	}

	message := "Update completed successfully"
	if status.Status == "completed" {
		message += ". Please restart the application to use the new version."
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: message,
		Results: map[string]interface{}{
			"update_info": updateInfo,
			"status":      status,
		},
	})
}

func (handler *Updater) GetUpdateHistory(c *fiber.Ctx) error {
	history, err := handler.manager.GetUpdateHistory()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "HISTORY_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Update history retrieved",
		Results: map[string]interface{}{
			"releases": history,
			"count":    len(history),
		},
	})
}

func (handler *Updater) GetCurrentVersion(c *fiber.Ctx) error {
	version := handler.manager.GetCurrentVersion()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Current version retrieved",
		Results: map[string]interface{}{
			"version": version,
		},
	})
}

func (handler *Updater) SetUpdateChannel(c *fiber.Ctx) error {
	var request struct {
		Channel string `json:"channel"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Channel == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Channel is required",
			Results: nil,
		})
	}

	err := handler.manager.SetUpdateChannel(request.Channel)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_CHANNEL",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Update channel set successfully",
		Results: map[string]interface{}{
			"channel": request.Channel,
		},
	})
}

func (handler *Updater) SetAutoUpdate(c *fiber.Ctx) error {
	var request struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	handler.manager.SetAutoUpdate(request.Enabled)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Auto-update setting updated",
		Results: map[string]interface{}{
			"auto_update_enabled": request.Enabled,
		},
	})
}
package rest

import (
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/templates"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Templates struct {
	manager *templates.TemplateManager
}

func InitRestTemplates(app fiber.Router) Templates {
	tm := templates.NewTemplateManager()
	rest := Templates{manager: tm}

	// Template management routes
	app.Post("/templates", rest.CreateTemplate)
	app.Post("/templates/advanced", rest.CreateAdvancedTemplate)
	app.Get("/templates", rest.ListTemplates)
	app.Get("/templates/search", rest.SearchTemplates)
	app.Get("/templates/:id", rest.GetTemplate)
	app.Put("/templates/:id", rest.UpdateTemplate)
	app.Delete("/templates/:id", rest.DeleteTemplate)
	app.Post("/templates/:id/render", rest.RenderTemplate)
	app.Post("/templates/:id/render-advanced", rest.RenderAdvancedTemplate)
	app.Post("/templates/:id/clone", rest.CloneTemplate)
	app.Get("/templates/:id/versions", rest.GetTemplateVersions)
	app.Post("/templates/:id/restore/:version", rest.RestoreTemplateVersion)
	app.Put("/templates/bulk", rest.BulkUpdateTemplates)
	app.Get("/templates/stats", rest.GetStats)

	return rest
}

func (handler *Templates) CreateTemplate(c *fiber.Ctx) error {
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
		Category    string `json:"category"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	template, err := handler.manager.CreateTemplate(
		request.Name,
		request.Description,
		request.Content,
		request.Category,
	)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "CREATE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Template created successfully",
		Results: template,
	})
}

func (handler *Templates) ListTemplates(c *fiber.Ctx) error {
	category := c.Query("category", "")
	templates := handler.manager.ListTemplates(category)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Templates retrieved successfully",
		Results: map[string]interface{}{
			"templates": templates,
			"count":     len(templates),
			"category":  category,
		},
	})
}

func (handler *Templates) GetTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	template, err := handler.manager.GetTemplate(id)
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
		Message: "Template retrieved successfully",
		Results: template,
	})
}

func (handler *Templates) UpdateTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
		Category    string `json:"category"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	template, err := handler.manager.UpdateTemplate(
		id,
		request.Name,
		request.Description,
		request.Content,
		request.Category,
	)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "UPDATE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Template updated successfully",
		Results: template,
	})
}

func (handler *Templates) DeleteTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	err := handler.manager.DeleteTemplate(id)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "DELETE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Template deleted successfully",
		Results: nil,
	})
}

func (handler *Templates) RenderTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	var request struct {
		Variables map[string]string `json:"variables"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.Variables == nil {
		request.Variables = make(map[string]string)
	}

	renderedContent, err := handler.manager.RenderTemplate(id, request.Variables)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "RENDER_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Template rendered successfully",
		Results: map[string]interface{}{
			"template_id":      id,
			"rendered_content": renderedContent,
			"variables_used":   request.Variables,
		},
	})
}

func (handler *Templates) CreateAdvancedTemplate(c *fiber.Ctx) error {
	var template templates.Template

	if err := c.BodyParser(&template); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	createdTemplate, err := handler.manager.CreateAdvancedTemplate(&template)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "CREATE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Advanced template created successfully",
		Results: createdTemplate,
	})
}

func (handler *Templates) SearchTemplates(c *fiber.Ctx) error {
	query := c.Query("q", "")
	
	filters := make(map[string]interface{})
	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}
	if isActive := c.Query("is_active"); isActive != "" {
		filters["is_active"] = isActive == "true"
	}
	if tags := c.Query("tags"); tags != "" {
		filters["tags"] = strings.Split(tags, ",")
	}

	templates := handler.manager.SearchTemplates(query, filters)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Templates search completed",
		Results: map[string]interface{}{
			"templates": templates,
			"count":     len(templates),
			"query":     query,
			"filters":   filters,
		},
	})
}

func (handler *Templates) RenderAdvancedTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	var context templates.RenderContext
	if err := c.BodyParser(&context); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	// Set context metadata
	context.UserAgent = c.Get("User-Agent")
	context.IP = c.IP()
	context.Timestamp = time.Now()

	if context.Variables == nil {
		context.Variables = make(map[string]interface{})
	}

	renderedContent, err := handler.manager.RenderAdvancedTemplate(id, &context)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "RENDER_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Advanced template rendered successfully",
		Results: map[string]interface{}{
			"template_id":      id,
			"rendered_content": renderedContent,
			"context":          context,
		},
	})
}

func (handler *Templates) CloneTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	var request struct {
		NewName string `json:"new_name"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	if request.NewName == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "New name is required",
			Results: nil,
		})
	}

	clonedTemplate, err := handler.manager.CloneTemplate(id, request.NewName)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "CLONE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "SUCCESS",
		Message: "Template cloned successfully",
		Results: clonedTemplate,
	})
}

func (handler *Templates) GetTemplateVersions(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID is required",
			Results: nil,
		})
	}

	versions, err := handler.manager.GetTemplateVersions(id)
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
		Message: "Template versions retrieved successfully",
		Results: map[string]interface{}{
			"template_id": id,
			"versions":    versions,
			"count":       len(versions),
		},
	})
}

func (handler *Templates) RestoreTemplateVersion(c *fiber.Ctx) error {
	id := c.Params("id")
	version := c.Params("version")
	
	if id == "" || version == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Template ID and version are required",
			Results: nil,
		})
	}

	err := handler.manager.RestoreTemplateVersion(id, version)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "RESTORE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Template version restored successfully",
		Results: map[string]interface{}{
			"template_id": id,
			"version":     version,
		},
	})
}

func (handler *Templates) BulkUpdateTemplates(c *fiber.Ctx) error {
	var updates map[string]map[string]interface{}

	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Results: nil,
		})
	}

	err := handler.manager.BulkUpdateTemplates(updates)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BULK_UPDATE_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Templates updated successfully",
		Results: map[string]interface{}{
			"updated_count": len(updates),
		},
	})
}

func (handler *Templates) GetStats(c *fiber.Ctx) error {
	stats := handler.manager.GetTemplateStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Template statistics retrieved",
		Results: stats,
	})
}
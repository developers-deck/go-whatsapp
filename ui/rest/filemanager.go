package rest

import (
	"strconv"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/filemanager"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type FileManager struct {
	manager *filemanager.FileManager
}

func InitRestFileManager(app fiber.Router) FileManager {
	fm := filemanager.NewFileManager()
	rest := FileManager{manager: fm}

	// Start periodic cleanup
	fm.StartPeriodicCleanup()

	// File management routes
	app.Post("/files/upload", rest.UploadFile)
	app.Get("/files/download/:fileId", rest.DownloadFile)
	app.Delete("/files/:fileId", rest.DeleteFile)
	app.Get("/files/list", rest.ListFiles)
	app.Get("/files/stats", rest.GetStorageStats)
	app.Post("/files/cleanup", rest.CleanupFiles)

	return rest
}

func (handler *FileManager) UploadFile(c *fiber.Ctx) error {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "No file uploaded",
			Results: nil,
		})
	}

	// Get category (default: upload)
	category := c.FormValue("category", "upload")

	// Upload file
	fileInfo, err := handler.manager.UploadFile(file, category)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "UPLOAD_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "File uploaded successfully",
		Results: fileInfo,
	})
}

func (handler *FileManager) DownloadFile(c *fiber.Ctx) error {
	fileID := c.Params("fileId")
	if fileID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "File ID is required",
			Results: nil,
		})
	}

	fileInfo, file, err := handler.manager.DownloadFile(fileID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "FILE_NOT_FOUND",
			Message: err.Error(),
			Results: nil,
		})
	}
	defer file.Close()

	// Set appropriate headers
	c.Set("Content-Disposition", "attachment; filename=\""+fileInfo.OriginalName+"\"")
	c.Set("Content-Type", fileInfo.MimeType)
	c.Set("Content-Length", strconv.FormatInt(fileInfo.Size, 10))

	// Send file
	return c.SendStream(file)
}

func (handler *FileManager) DeleteFile(c *fiber.Ctx) error {
	fileID := c.Params("fileId")
	if fileID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "File ID is required",
			Results: nil,
		})
	}

	err := handler.manager.DeleteFile(fileID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "FILE_NOT_FOUND",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "File deleted successfully",
		Results: nil,
	})
}

func (handler *FileManager) ListFiles(c *fiber.Ctx) error {
	category := c.Query("category", "upload")
	limitStr := c.Query("limit", "50")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	files, err := handler.manager.ListFiles(category, limit)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "LIST_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Files listed successfully",
		Results: map[string]interface{}{
			"files":    files,
			"category": category,
			"count":    len(files),
		},
	})
}

func (handler *FileManager) GetStorageStats(c *fiber.Ctx) error {
	stats := handler.manager.GetStorageStats()

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Storage statistics retrieved",
		Results: stats,
	})
}

func (handler *FileManager) CleanupFiles(c *fiber.Ctx) error {
	err := handler.manager.CleanupExpiredFiles()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CLEANUP_ERROR",
			Message: err.Error(),
			Results: nil,
		})
	}

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "File cleanup completed",
		Results: nil,
	})
}
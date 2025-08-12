package filemanager

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type FileInfo struct {
	ID          string    `json:"id"`
	OriginalName string   `json:"original_name"`
	StoredName   string   `json:"stored_name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	MimeType    string    `json:"mime_type"`
	MD5Hash     string    `json:"md5_hash"`
	UploadedAt  time.Time `json:"uploaded_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type FileManager struct {
	uploadPath   string
	downloadPath string
	tempPath     string
}

func NewFileManager() *FileManager {
	uploadPath := filepath.Join(config.PathSendItems, "uploads")
	downloadPath := filepath.Join(config.PathMedia, "downloads")
	tempPath := filepath.Join(config.PathSendItems, "temp")

	// Create directories if they don't exist
	os.MkdirAll(uploadPath, 0755)
	os.MkdirAll(downloadPath, 0755)
	os.MkdirAll(tempPath, 0755)

	return &FileManager{
		uploadPath:   uploadPath,
		downloadPath: downloadPath,
		tempPath:     tempPath,
	}
}

// UploadFile handles file upload with advanced features
func (fm *FileManager) UploadFile(file *multipart.FileHeader, category string) (*FileInfo, error) {
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	fileID := fm.generateFileID()
	ext := filepath.Ext(file.Filename)
	storedName := fmt.Sprintf("%s_%s%s", category, fileID, ext)
	
	// Determine storage path based on category
	var storagePath string
	switch category {
	case "upload", "send":
		storagePath = fm.uploadPath
	case "download", "received":
		storagePath = fm.downloadPath
	case "temp", "temporary":
		storagePath = fm.tempPath
	default:
		storagePath = fm.uploadPath
	}

	fullPath := filepath.Join(storagePath, storedName)

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file content and calculate MD5 hash
	hash := md5.New()
	multiWriter := io.MultiWriter(dst, hash)
	
	size, err := io.Copy(multiWriter, src)
	if err != nil {
		os.Remove(fullPath) // Cleanup on error
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Create file info
	fileInfo := &FileInfo{
		ID:           fileID,
		OriginalName: file.Filename,
		StoredName:   storedName,
		Path:         fullPath,
		Size:         size,
		MimeType:     file.Header.Get("Content-Type"),
		MD5Hash:      hex.EncodeToString(hash.Sum(nil)),
		UploadedAt:   time.Now(),
	}

	// Set expiration for temp files (24 hours)
	if category == "temp" || category == "temporary" {
		expiresAt := time.Now().Add(24 * time.Hour)
		fileInfo.ExpiresAt = &expiresAt
	}

	logrus.Infof("[FILE] Uploaded file: %s -> %s (size: %d bytes, hash: %s)", 
		file.Filename, storedName, size, fileInfo.MD5Hash)

	return fileInfo, nil
}

// DownloadFile retrieves file information and content
func (fm *FileManager) DownloadFile(fileID string) (*FileInfo, *os.File, error) {
	// Search for file in all directories
	searchPaths := []string{fm.uploadPath, fm.downloadPath, fm.tempPath}
	
	for _, searchPath := range searchPaths {
		pattern := filepath.Join(searchPath, fmt.Sprintf("*_%s.*", fileID))
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		if len(matches) > 0 {
			filePath := matches[0]
			
			// Get file info
			stat, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			// Open file
			file, err := os.Open(filePath)
			if err != nil {
				continue
			}

			// Extract original info from filename
			fileName := filepath.Base(filePath)
			parts := strings.Split(fileName, "_")
			if len(parts) < 2 {
				file.Close()
				continue
			}

			fileInfo := &FileInfo{
				ID:         fileID,
				StoredName: fileName,
				Path:       filePath,
				Size:       stat.Size(),
				UploadedAt: stat.ModTime(),
			}

			return fileInfo, file, nil
		}
	}

	return nil, nil, fmt.Errorf("file not found: %s", fileID)
}

// DeleteFile removes a file from storage
func (fm *FileManager) DeleteFile(fileID string) error {
	fileInfo, file, err := fm.DownloadFile(fileID)
	if err != nil {
		return err
	}
	file.Close()

	err = os.Remove(fileInfo.Path)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logrus.Infof("[FILE] Deleted file: %s", fileInfo.StoredName)
	return nil
}

// ListFiles returns a list of files in a category
func (fm *FileManager) ListFiles(category string, limit int) ([]*FileInfo, error) {
	var searchPath string
	switch category {
	case "upload", "send":
		searchPath = fm.uploadPath
	case "download", "received":
		searchPath = fm.downloadPath
	case "temp", "temporary":
		searchPath = fm.tempPath
	default:
		searchPath = fm.uploadPath
	}

	pattern := filepath.Join(searchPath, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var files []*FileInfo
	count := 0
	
	for _, filePath := range matches {
		if limit > 0 && count >= limit {
			break
		}

		stat, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if stat.IsDir() {
			continue
		}

		fileName := filepath.Base(filePath)
		parts := strings.Split(fileName, "_")
		if len(parts) < 2 {
			continue
		}

		// Extract file ID from filename
		fileID := strings.TrimSuffix(parts[1], filepath.Ext(fileName))

		fileInfo := &FileInfo{
			ID:         fileID,
			StoredName: fileName,
			Path:       filePath,
			Size:       stat.Size(),
			UploadedAt: stat.ModTime(),
		}

		files = append(files, fileInfo)
		count++
	}

	return files, nil
}

// CleanupExpiredFiles removes expired temporary files
func (fm *FileManager) CleanupExpiredFiles() error {
	pattern := filepath.Join(fm.tempPath, "*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find temp files: %w", err)
	}

	cleaned := 0
	cutoff := time.Now().Add(-24 * time.Hour) // Files older than 24 hours

	for _, filePath := range matches {
		stat, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if stat.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				logrus.Errorf("[FILE] Failed to remove expired file %s: %v", filePath, err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		logrus.Infof("[FILE] Cleaned up %d expired files", cleaned)
	}

	return nil
}

// GetStorageStats returns storage statistics
func (fm *FileManager) GetStorageStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Calculate stats for each directory
	dirs := map[string]string{
		"uploads":   fm.uploadPath,
		"downloads": fm.downloadPath,
		"temp":      fm.tempPath,
	}

	totalSize := int64(0)
	totalFiles := 0

	for category, path := range dirs {
		size, count := fm.calculateDirStats(path)
		stats[category] = map[string]interface{}{
			"size":  size,
			"count": count,
		}
		totalSize += size
		totalFiles += count
	}

	stats["total"] = map[string]interface{}{
		"size":  totalSize,
		"count": totalFiles,
	}

	return stats
}

// generateFileID creates a unique file identifier
func (fm *FileManager) generateFileID() string {
	return fmt.Sprintf("%d_%s", time.Now().UnixNano(), 
		hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))[:8])
}

// calculateDirStats calculates size and file count for a directory
func (fm *FileManager) calculateDirStats(dirPath string) (int64, int) {
	var totalSize int64
	var fileCount int

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return totalSize, fileCount
}

// StartPeriodicCleanup starts a goroutine that performs periodic cleanup
func (fm *FileManager) StartPeriodicCleanup() {
	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := fm.CleanupExpiredFiles(); err != nil {
				logrus.Errorf("[FILE] Periodic cleanup failed: %v", err)
			}
		}
	}()

	logrus.Info("[FILE] Started periodic cleanup (every 1 hour)")
}
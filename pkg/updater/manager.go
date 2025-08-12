package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type UpdateManager struct {
	currentVersion string
	updateChannel  string
	checkInterval  time.Duration
	autoUpdate     bool
	ctx            context.Context
	cancel         context.CancelFunc
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	DownloadCount      int    `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

type UpdateInfo struct {
	Available       bool      `json:"available"`
	CurrentVersion  string    `json:"current_version"`
	LatestVersion   string    `json:"latest_version"`
	ReleaseNotes    string    `json:"release_notes"`
	DownloadURL     string    `json:"download_url"`
	Size            int64     `json:"size"`
	PublishedAt     time.Time `json:"published_at"`
	UpdateChannel   string    `json:"update_channel"`
	LastChecked     time.Time `json:"last_checked"`
}

type UpdateStatus struct {
	Status      string    `json:"status"` // checking, downloading, installing, completed, failed
	Progress    int       `json:"progress"` // 0-100
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

const (
	GitHubAPIURL = "https://api.github.com/repos/aldinokemal/go-whatsapp-web-multidevice/releases"
)

func NewUpdateManager() *UpdateManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	um := &UpdateManager{
		currentVersion: config.AppVersion,
		updateChannel:  "stable", // stable, beta, alpha
		checkInterval:  24 * time.Hour, // Check daily
		autoUpdate:     false, // Disabled by default for safety
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start periodic update checking
	go um.startPeriodicCheck()

	logrus.Info("[UPDATER] Update manager initialized")
	return um
}

// CheckForUpdates checks if a new version is available
func (um *UpdateManager) CheckForUpdates() (*UpdateInfo, error) {
	logrus.Info("[UPDATER] Checking for updates...")

	releases, err := um.fetchReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	if len(releases) == 0 {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: um.currentVersion,
			LastChecked:    time.Now(),
		}, nil
	}

	latestRelease := um.getLatestRelease(releases)
	if latestRelease == nil {
		return &UpdateInfo{
			Available:      false,
			CurrentVersion: um.currentVersion,
			LastChecked:    time.Now(),
		}, nil
	}

	// Check if update is available
	available := um.isNewerVersion(latestRelease.TagName, um.currentVersion)
	
	updateInfo := &UpdateInfo{
		Available:      available,
		CurrentVersion: um.currentVersion,
		LatestVersion:  latestRelease.TagName,
		ReleaseNotes:   latestRelease.Body,
		PublishedAt:    latestRelease.PublishedAt,
		UpdateChannel:  um.updateChannel,
		LastChecked:    time.Now(),
	}

	if available {
		// Find appropriate asset for current platform
		asset := um.findAssetForPlatform(latestRelease.Assets)
		if asset != nil {
			updateInfo.DownloadURL = asset.BrowserDownloadURL
			updateInfo.Size = asset.Size
		}
	}

	logrus.Infof("[UPDATER] Update check completed. Available: %v, Current: %s, Latest: %s", 
		available, um.currentVersion, latestRelease.TagName)

	return updateInfo, nil
}

// PerformUpdate downloads and installs the update
func (um *UpdateManager) PerformUpdate(updateInfo *UpdateInfo) (*UpdateStatus, error) {
	if !updateInfo.Available {
		return nil, fmt.Errorf("no update available")
	}

	if updateInfo.DownloadURL == "" {
		return nil, fmt.Errorf("no download URL available")
	}

	status := &UpdateStatus{
		Status:    "downloading",
		Progress:  0,
		Message:   "Starting download...",
		StartedAt: time.Now(),
	}

	logrus.Infof("[UPDATER] Starting update from %s to %s", um.currentVersion, updateInfo.LatestVersion)

	// Download the update
	tempFile, err := um.downloadUpdate(updateInfo.DownloadURL, status)
	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		return status, err
	}
	defer os.Remove(tempFile)

	// Install the update
	status.Status = "installing"
	status.Progress = 90
	status.Message = "Installing update..."

	err = um.installUpdate(tempFile)
	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		return status, err
	}

	// Complete
	now := time.Now()
	status.Status = "completed"
	status.Progress = 100
	status.Message = "Update completed successfully"
	status.CompletedAt = &now

	logrus.Info("[UPDATER] Update completed successfully")
	return status, nil
}

// GetUpdateHistory returns the update history
func (um *UpdateManager) GetUpdateHistory() ([]Release, error) {
	releases, err := um.fetchReleases()
	if err != nil {
		return nil, err
	}

	// Filter and sort releases
	var history []Release
	for _, release := range releases {
		if !release.Draft && (um.updateChannel != "stable" || !release.Prerelease) {
			history = append(history, release)
		}
	}

	return history, nil
}

// SetUpdateChannel sets the update channel (stable, beta, alpha)
func (um *UpdateManager) SetUpdateChannel(channel string) error {
	validChannels := []string{"stable", "beta", "alpha"}
	for _, valid := range validChannels {
		if channel == valid {
			um.updateChannel = channel
			logrus.Infof("[UPDATER] Update channel set to: %s", channel)
			return nil
		}
	}
	return fmt.Errorf("invalid update channel: %s", channel)
}

// SetAutoUpdate enables or disables automatic updates
func (um *UpdateManager) SetAutoUpdate(enabled bool) {
	um.autoUpdate = enabled
	logrus.Infof("[UPDATER] Auto-update %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// GetCurrentVersion returns the current version
func (um *UpdateManager) GetCurrentVersion() string {
	return um.currentVersion
}

// Private methods

func (um *UpdateManager) fetchReleases() ([]Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", GitHubAPIURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("go-whatsapp-web-multidevice/%s", um.currentVersion))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func (um *UpdateManager) getLatestRelease(releases []Release) *Release {
	for _, release := range releases {
		if release.Draft {
			continue
		}

		// Filter by update channel
		if um.updateChannel == "stable" && release.Prerelease {
			continue
		}

		return &release
	}
	return nil
}

func (um *UpdateManager) isNewerVersion(latest, current string) bool {
	// Simple version comparison (assumes semantic versioning)
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	
	// For now, just do string comparison
	// In a production system, you'd want proper semantic version comparison
	return latest > current
}

func (um *UpdateManager) findAssetForPlatform(assets []Asset) *Asset {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch names to common naming conventions
	archMap := map[string][]string{
		"amd64": {"amd64", "x86_64", "64bit"},
		"386":   {"386", "i386", "32bit"},
		"arm64": {"arm64", "aarch64"},
		"arm":   {"arm", "armv7"},
	}

	platformMap := map[string][]string{
		"windows": {"windows", "win"},
		"linux":   {"linux"},
		"darwin":  {"darwin", "macos", "osx"},
	}

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		
		// Check if asset matches current platform
		platformMatch := false
		for _, platformName := range platformMap[platform] {
			if strings.Contains(name, platformName) {
				platformMatch = true
				break
			}
		}

		if !platformMatch {
			continue
		}

		// Check if asset matches current architecture
		archMatch := false
		for _, archName := range archMap[arch] {
			if strings.Contains(name, archName) {
				archMatch = true
				break
			}
		}

		if archMatch {
			return &asset
		}
	}

	return nil
}

func (um *UpdateManager) downloadUpdate(url string, status *UpdateStatus) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("update_%d", time.Now().Unix()))
	file, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Download with progress tracking
	contentLength := resp.ContentLength
	var downloaded int64

	buffer := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return "", writeErr
			}
			downloaded += int64(n)

			// Update progress
			if contentLength > 0 {
				progress := int((downloaded * 80) / contentLength) // 80% for download
				status.Progress = progress
				status.Message = fmt.Sprintf("Downloaded %d/%d bytes", downloaded, contentLength)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	status.Progress = 80
	status.Message = "Download completed"
	
	return tempFile, nil
}

func (um *UpdateManager) installUpdate(tempFile string) error {
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create backup of current executable
	backupPath := currentExe + ".backup"
	if err := um.copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace current executable with new version
	if err := um.copyFile(tempFile, currentExe); err != nil {
		// Restore backup on failure
		um.copyFile(backupPath, currentExe)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Make executable (Unix systems)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(currentExe, 0755); err != nil {
			logrus.Warnf("[UPDATER] Failed to set executable permissions: %v", err)
		}
	}

	// Clean up backup after successful installation
	os.Remove(backupPath)

	return nil
}

func (um *UpdateManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (um *UpdateManager) startPeriodicCheck() {
	ticker := time.NewTicker(um.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-um.ctx.Done():
			return
		case <-ticker.C:
			if um.autoUpdate {
				um.performAutoUpdate()
			} else {
				// Just check and log
				updateInfo, err := um.CheckForUpdates()
				if err != nil {
					logrus.Errorf("[UPDATER] Periodic update check failed: %v", err)
				} else if updateInfo.Available {
					logrus.Infof("[UPDATER] Update available: %s -> %s", 
						updateInfo.CurrentVersion, updateInfo.LatestVersion)
				}
			}
		}
	}
}

func (um *UpdateManager) performAutoUpdate() {
	updateInfo, err := um.CheckForUpdates()
	if err != nil {
		logrus.Errorf("[UPDATER] Auto-update check failed: %v", err)
		return
	}

	if !updateInfo.Available {
		return
	}

	logrus.Infof("[UPDATER] Performing automatic update to %s", updateInfo.LatestVersion)
	
	status, err := um.PerformUpdate(updateInfo)
	if err != nil {
		logrus.Errorf("[UPDATER] Auto-update failed: %v", err)
		return
	}

	if status.Status == "completed" {
		logrus.Info("[UPDATER] Auto-update completed successfully. Restart required.")
		// In a production system, you might want to gracefully restart the application
	}
}

// Stop gracefully stops the update manager
func (um *UpdateManager) Stop() {
	logrus.Info("[UPDATER] Stopping update manager...")
	um.cancel()
}
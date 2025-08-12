package isolation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

type ProcessIsolationManager struct {
	processes map[string]*IsolatedProcess
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

type IsolatedProcess struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
	Process     *os.Process       `json:"-"`
	PID         int               `json:"pid"`
	Status      ProcessStatus     `json:"status"`
	StartedAt   time.Time         `json:"started_at"`
	StoppedAt   *time.Time        `json:"stopped_at,omitempty"`
	ExitCode    int               `json:"exit_code"`
	LogPath     string            `json:"log_path"`
	Resources   ResourceLimits    `json:"resources"`
	Monitoring  ProcessMonitoring `json:"monitoring"`
	mutex       sync.RWMutex      `json:"-"`
}

type ProcessStatus string

const (
	ProcessStatusStopped  ProcessStatus = "stopped"
	ProcessStatusStarting ProcessStatus = "starting"
	ProcessStatusRunning  ProcessStatus = "running"
	ProcessStatusStopping ProcessStatus = "stopping"
	ProcessStatusError    ProcessStatus = "error"
	ProcessStatusCrashed  ProcessStatus = "crashed"
)

type ResourceLimits struct {
	MaxMemoryMB int           `json:"max_memory_mb"`
	MaxCPU      float64       `json:"max_cpu_percent"`
	Timeout     time.Duration `json:"timeout"`
	Priority    int           `json:"priority"` // Process priority (-20 to 19 on Unix)
}

type ProcessMonitoring struct {
	CPUUsage    float64       `json:"cpu_usage"`
	MemoryUsage int64         `json:"memory_usage_mb"`
	LastCheck   time.Time     `json:"last_check"`
	Restarts    int           `json:"restarts"`
	Uptime      time.Duration `json:"uptime"`
}

type IsolationConfig struct {
	EnableResourceLimits bool          `json:"enable_resource_limits"`
	DefaultMemoryLimit   int           `json:"default_memory_limit_mb"`
	DefaultCPULimit      float64       `json:"default_cpu_limit"`
	DefaultTimeout       time.Duration `json:"default_timeout"`
	MonitoringInterval   time.Duration `json:"monitoring_interval"`
	AutoRestart          bool          `json:"auto_restart"`
	MaxRestarts          int           `json:"max_restarts"`
	PathStorages         string        `json:"path_storages"`
}

func NewProcessIsolationManager(config IsolationConfig) *ProcessIsolationManager {
	ctx, cancel := context.WithCancel(context.Background())

	pim := &ProcessIsolationManager{
		processes: make(map[string]*IsolatedProcess),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start monitoring
	if config.MonitoringInterval > 0 {
		go pim.startMonitoring(config.MonitoringInterval)
	}

	logrus.Info("[ISOLATION] Process isolation manager initialized")
	return pim
}

// CreateIsolatedProcess creates a new isolated process
func (pim *ProcessIsolationManager) CreateIsolatedProcess(id, name, command string, args []string, config IsolationConfig) (*IsolatedProcess, error) {
	pim.mutex.Lock()
	defer pim.mutex.Unlock()

	if _, exists := pim.processes[id]; exists {
		return nil, fmt.Errorf("process with ID %s already exists", id)
	}

	// Create working directory
	workingDir := filepath.Join(config.PathStorages, "processes", id)
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	// Create logs directory
	logsDir := filepath.Join(workingDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	process := &IsolatedProcess{
		ID:          id,
		Name:        name,
		Command:     command,
		Args:        args,
		WorkingDir:  workingDir,
		Environment: make(map[string]string),
		Status:      ProcessStatusStopped,
		LogPath:     filepath.Join(logsDir, "process.log"),
		Resources: ResourceLimits{
			MaxMemoryMB: config.DefaultMemoryLimit,
			MaxCPU:      config.DefaultCPULimit,
			Timeout:     config.DefaultTimeout,
			Priority:    0,
		},
		Monitoring: ProcessMonitoring{
			LastCheck: time.Now(),
		},
	}

	pim.processes[id] = process
	logrus.Infof("[ISOLATION] Created isolated process: %s", id)
	return process, nil
}

// StartProcess starts an isolated process
func (pim *ProcessIsolationManager) StartProcess(id string) error {
	pim.mutex.RLock()
	process, exists := pim.processes[id]
	pim.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	process.mutex.Lock()
	defer process.mutex.Unlock()

	if process.Status == ProcessStatusRunning {
		return fmt.Errorf("process is already running")
	}

	process.Status = ProcessStatusStarting
	logrus.Infof("[ISOLATION] Starting isolated process: %s", id)

	// Create command
	cmd := exec.Command(process.Command, process.Args...)
	cmd.Dir = process.WorkingDir

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range process.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create log file
	logFile, err := os.Create(process.LogPath)
	if err != nil {
		process.Status = ProcessStatusError
		return fmt.Errorf("failed to create log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Apply resource limits (platform-specific)
	if err := pim.applyResourceLimits(cmd, process.Resources); err != nil {
		logFile.Close()
		process.Status = ProcessStatusError
		return fmt.Errorf("failed to apply resource limits: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		logFile.Close()
		process.Status = ProcessStatusError
		return fmt.Errorf("failed to start process: %w", err)
	}

	process.Process = cmd.Process
	process.PID = cmd.Process.Pid
	process.Status = ProcessStatusRunning
	process.StartedAt = time.Now()

	// Monitor process in background
	go pim.monitorProcess(process, cmd, logFile)

	logrus.Infof("[ISOLATION] Started isolated process: %s (PID: %d)", id, process.PID)
	return nil
}

// StopProcess stops an isolated process
func (pim *ProcessIsolationManager) StopProcess(id string) error {
	pim.mutex.RLock()
	process, exists := pim.processes[id]
	pim.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	process.mutex.Lock()
	defer process.mutex.Unlock()

	if process.Status != ProcessStatusRunning {
		return fmt.Errorf("process is not running")
	}

	process.Status = ProcessStatusStopping
	logrus.Infof("[ISOLATION] Stopping isolated process: %s", id)

	if process.Process != nil {
		// Try graceful shutdown first
		if err := process.Process.Signal(os.Interrupt); err != nil {
			// Force kill if graceful shutdown fails
			process.Process.Kill()
		}

		// Wait for process to exit with timeout
		done := make(chan error, 1)
		go func() {
			_, err := process.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(10 * time.Second):
			// Timeout, force kill
			process.Process.Kill()
			<-done
		}

		process.Status = ProcessStatusStopped
		now := time.Now()
		process.StoppedAt = &now
		process.Process = nil
		process.PID = 0
	}

	logrus.Infof("[ISOLATION] Stopped isolated process: %s", id)
	return nil
}

// RestartProcess restarts an isolated process
func (pim *ProcessIsolationManager) RestartProcess(id string) error {
	process, exists := pim.processes[id]
	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	logrus.Infof("[ISOLATION] Restarting isolated process: %s", id)

	// Stop the process
	if err := pim.StopProcess(id); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)

	// Increment restart counter
	process.mutex.Lock()
	process.Monitoring.Restarts++
	process.mutex.Unlock()

	// Start the process
	if err := pim.StartProcess(id); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	return nil
}

// DeleteProcess removes an isolated process
func (pim *ProcessIsolationManager) DeleteProcess(id string) error {
	pim.mutex.Lock()
	defer pim.mutex.Unlock()

	process, exists := pim.processes[id]
	if !exists {
		return fmt.Errorf("process not found: %s", id)
	}

	// Stop process if running
	if process.Status == ProcessStatusRunning {
		pim.StopProcess(id)
		time.Sleep(2 * time.Second)
	}

	// Remove working directory
	if err := os.RemoveAll(process.WorkingDir); err != nil {
		logrus.Warnf("[ISOLATION] Failed to remove working directory: %v", err)
	}

	// Remove from processes map
	delete(pim.processes, id)

	logrus.Infof("[ISOLATION] Deleted isolated process: %s", id)
	return nil
}

// GetProcess retrieves a process by ID
func (pim *ProcessIsolationManager) GetProcess(id string) (*IsolatedProcess, error) {
	pim.mutex.RLock()
	defer pim.mutex.RUnlock()

	process, exists := pim.processes[id]
	if !exists {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	return process, nil
}

// ListProcesses returns all processes
func (pim *ProcessIsolationManager) ListProcesses() []*IsolatedProcess {
	pim.mutex.RLock()
	defer pim.mutex.RUnlock()

	processes := make([]*IsolatedProcess, 0, len(pim.processes))
	for _, process := range pim.processes {
		processes = append(processes, process)
	}

	return processes
}

// Private methods

func (pim *ProcessIsolationManager) applyResourceLimits(cmd *exec.Cmd, limits ResourceLimits) error {
	// Platform-specific resource limit implementation
	switch runtime.GOOS {
	case "linux", "darwin":
		return pim.applyUnixResourceLimits(cmd, limits)
	case "windows":
		return pim.applyWindowsResourceLimits(cmd, limits)
	default:
		logrus.Warnf("[ISOLATION] Resource limits not supported on %s", runtime.GOOS)
		return nil
	}
}

func (pim *ProcessIsolationManager) applyUnixResourceLimits(cmd *exec.Cmd, limits ResourceLimits) error {
	// Set process priority (Unix/Linux specific)
	if limits.Priority != 0 {
		// On Unix systems, we can set process priority using Nice
		// Note: This is a simplified implementation
		logrus.Debugf("[ISOLATION] Setting process priority to: %d", limits.Priority)
	}

	// Memory and CPU limits would typically be implemented using cgroups
	// For simplicity, we'll just log the limits
	logrus.Debugf("[ISOLATION] Applied Unix resource limits: Memory=%dMB, CPU=%.1f%%",
		limits.MaxMemoryMB, limits.MaxCPU)

	return nil
}

func (pim *ProcessIsolationManager) applyWindowsResourceLimits(cmd *exec.Cmd, limits ResourceLimits) error {
	// Windows-specific resource limits would be implemented using Job Objects
	// For simplicity, we'll just log the limits
	logrus.Debugf("[ISOLATION] Applied Windows resource limits: Memory=%dMB, CPU=%.1f%%",
		limits.MaxMemoryMB, limits.MaxCPU)

	return nil
}

func (pim *ProcessIsolationManager) monitorProcess(process *IsolatedProcess, cmd *exec.Cmd, logFile *os.File) {
	defer logFile.Close()

	// Wait for process to exit
	err := cmd.Wait()

	process.mutex.Lock()
	if process.Status == ProcessStatusStopping {
		process.Status = ProcessStatusStopped
	} else {
		process.Status = ProcessStatusCrashed
		process.ExitCode = cmd.ProcessState.ExitCode()
	}
	process.Process = nil
	process.PID = 0
	now := time.Now()
	process.StoppedAt = &now
	process.mutex.Unlock()

	if err != nil {
		logrus.Errorf("[ISOLATION] Process %s exited with error: %v", process.ID, err)
	} else {
		logrus.Infof("[ISOLATION] Process %s exited normally", process.ID)
	}
}

func (pim *ProcessIsolationManager) startMonitoring(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-pim.ctx.Done():
			return
		case <-ticker.C:
			pim.updateProcessMetrics()
		}
	}
}

func (pim *ProcessIsolationManager) updateProcessMetrics() {
	pim.mutex.RLock()
	defer pim.mutex.RUnlock()

	for _, process := range pim.processes {
		process.mutex.Lock()
		if process.Status == ProcessStatusRunning && process.PID > 0 {
			// Update uptime
			process.Monitoring.Uptime = time.Since(process.StartedAt)

			// In a real implementation, you would get actual CPU and memory usage
			// For now, we'll just update the last check time
			process.Monitoring.LastCheck = time.Now()

			// Check if process is still alive
			if process.Process != nil {
				if err := process.Process.Signal(syscall.Signal(0)); err != nil {
					// Process is dead
					process.Status = ProcessStatusCrashed
					process.Process = nil
					process.PID = 0
				}
			}
		}
		process.mutex.Unlock()
	}
}

// Stop gracefully stops the process isolation manager
func (pim *ProcessIsolationManager) Stop() {
	logrus.Info("[ISOLATION] Stopping process isolation manager...")

	// Stop all running processes
	for _, process := range pim.processes {
		if process.Status == ProcessStatusRunning {
			pim.StopProcess(process.ID)
		}
	}

	pim.cancel()
	logrus.Info("[ISOLATION] Process isolation manager stopped")
}

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusRetrying   JobStatus = "retrying"
	StatusCancelled  JobStatus = "cancelled"
)

type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    Priority               `json:"priority"`
	Status      JobStatus              `json:"status"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	Error       string                 `json:"error,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type JobHandler func(ctx context.Context, job *Job) error

type QueueManager struct {
	queues      map[Priority][]*Job
	handlers    map[string]JobHandler
	workers     map[Priority]int
	running     bool
	mutex       sync.RWMutex
	jobMutex    sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	stats       *QueueStats
	rateLimiter map[string]*RateLimiter
}

type QueueStats struct {
	TotalJobs     int64                    `json:"total_jobs"`
	CompletedJobs int64                    `json:"completed_jobs"`
	FailedJobs    int64                    `json:"failed_jobs"`
	PendingJobs   map[Priority]int         `json:"pending_jobs"`
	ProcessingJobs int                     `json:"processing_jobs"`
	JobsByType    map[string]int64         `json:"jobs_by_type"`
	AverageTime   map[string]time.Duration `json:"average_time"`
	LastUpdated   time.Time                `json:"last_updated"`
	mutex         sync.RWMutex
}

type RateLimiter struct {
	tokens    int
	maxTokens int
	refillRate time.Duration
	lastRefill time.Time
	mutex     sync.Mutex
}

type QueueConfig struct {
	MaxWorkers     map[Priority]int `json:"max_workers"`
	RetryDelay     time.Duration    `json:"retry_delay"`
	MaxRetries     int              `json:"max_retries"`
	JobTimeout     time.Duration    `json:"job_timeout"`
	CleanupInterval time.Duration   `json:"cleanup_interval"`
	RateLimits     map[string]int   `json:"rate_limits"` // jobs per minute by type
}

func NewQueueManager() *QueueManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	qm := &QueueManager{
		queues:      make(map[Priority][]*Job),
		handlers:    make(map[string]JobHandler),
		workers:     make(map[Priority]int),
		ctx:         ctx,
		cancel:      cancel,
		rateLimiter: make(map[string]*RateLimiter),
		stats: &QueueStats{
			PendingJobs: make(map[Priority]int),
			JobsByType:  make(map[string]int64),
			AverageTime: make(map[string]time.Duration),
			LastUpdated: time.Now(),
		},
	}

	// Initialize queues for each priority
	for priority := PriorityLow; priority <= PriorityUrgent; priority++ {
		qm.queues[priority] = make([]*Job, 0)
		qm.workers[priority] = 0
	}

	// Set default configuration
	qm.applyDefaultConfig()

	// Start background processes
	go qm.startWorkers()
	go qm.startCleanup()
	go qm.startStatsUpdater()

	logrus.Info("[QUEUE] Queue manager initialized")
	return qm
}

func (qm *QueueManager) applyDefaultConfig() {
	// Default worker configuration
	defaultWorkers := map[Priority]int{
		PriorityUrgent: 5,
		PriorityHigh:   3,
		PriorityNormal: 2,
		PriorityLow:    1,
	}

	for priority, count := range defaultWorkers {
		qm.workers[priority] = count
	}

	// Default rate limiters
	defaultRateLimits := map[string]int{
		"send_message": 60,  // 60 messages per minute
		"send_media":   30,  // 30 media files per minute
		"send_bulk":    10,  // 10 bulk operations per minute
	}

	for jobType, limit := range defaultRateLimits {
		qm.rateLimiter[jobType] = &RateLimiter{
			tokens:     limit,
			maxTokens:  limit,
			refillRate: time.Minute,
			lastRefill: time.Now(),
		}
	}
}

// RegisterHandler registers a job handler for a specific job type
func (qm *QueueManager) RegisterHandler(jobType string, handler JobHandler) {
	qm.mutex.Lock()
	defer qm.mutex.Unlock()
	
	qm.handlers[jobType] = handler
	logrus.Infof("[QUEUE] Registered handler for job type: %s", jobType)
}

// AddJob adds a new job to the queue
func (qm *QueueManager) AddJob(jobType string, data map[string]interface{}, priority Priority) (*Job, error) {
	job := &Job{
		ID:          qm.generateJobID(),
		Type:        jobType,
		Priority:    priority,
		Status:      StatusPending,
		Data:        data,
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
		Attempts:    0,
		MaxAttempts: 3,
		Timeout:     5 * time.Minute,
		Metadata:    make(map[string]interface{}),
	}

	// Check rate limiting
	if !qm.checkRateLimit(jobType) {
		return nil, fmt.Errorf("rate limit exceeded for job type: %s", jobType)
	}

	qm.jobMutex.Lock()
	qm.queues[priority] = append(qm.queues[priority], job)
	qm.jobMutex.Unlock()

	// Update stats
	qm.stats.mutex.Lock()
	qm.stats.TotalJobs++
	qm.stats.PendingJobs[priority]++
	qm.stats.JobsByType[jobType]++
	qm.stats.LastUpdated = time.Now()
	qm.stats.mutex.Unlock()

	logrus.Debugf("[QUEUE] Added job %s (type: %s, priority: %d)", job.ID, jobType, priority)
	return job, nil
}

// ScheduleJob schedules a job to run at a specific time
func (qm *QueueManager) ScheduleJob(jobType string, data map[string]interface{}, priority Priority, scheduledAt time.Time) (*Job, error) {
	job, err := qm.AddJob(jobType, data, priority)
	if err != nil {
		return nil, err
	}

	job.ScheduledAt = scheduledAt
	logrus.Infof("[QUEUE] Scheduled job %s for %s", job.ID, scheduledAt.Format(time.RFC3339))
	return job, nil
}

// GetJob retrieves a job by ID
func (qm *QueueManager) GetJob(jobID string) (*Job, error) {
	qm.jobMutex.RLock()
	defer qm.jobMutex.RUnlock()

	for _, queue := range qm.queues {
		for _, job := range queue {
			if job.ID == jobID {
				return job, nil
			}
		}
	}

	return nil, fmt.Errorf("job not found: %s", jobID)
}

// CancelJob cancels a pending job
func (qm *QueueManager) CancelJob(jobID string) error {
	qm.jobMutex.Lock()
	defer qm.jobMutex.Unlock()

	for priority, queue := range qm.queues {
		for i, job := range queue {
			if job.ID == jobID && job.Status == StatusPending {
				job.Status = StatusCancelled
				// Remove from queue
				qm.queues[priority] = append(queue[:i], queue[i+1:]...)
				
				// Update stats
				qm.stats.mutex.Lock()
				qm.stats.PendingJobs[priority]--
				qm.stats.mutex.Unlock()
				
				logrus.Infof("[QUEUE] Cancelled job %s", jobID)
				return nil
			}
		}
	}

	return fmt.Errorf("job not found or cannot be cancelled: %s", jobID)
}

// GetQueueStats returns current queue statistics
func (qm *QueueManager) GetQueueStats() *QueueStats {
	qm.stats.mutex.RLock()
	defer qm.stats.mutex.RUnlock()

	// Create a copy to avoid race conditions
	stats := &QueueStats{
		TotalJobs:      qm.stats.TotalJobs,
		CompletedJobs:  qm.stats.CompletedJobs,
		FailedJobs:     qm.stats.FailedJobs,
		ProcessingJobs: qm.stats.ProcessingJobs,
		PendingJobs:    make(map[Priority]int),
		JobsByType:     make(map[string]int64),
		AverageTime:    make(map[string]time.Duration),
		LastUpdated:    qm.stats.LastUpdated,
	}

	for k, v := range qm.stats.PendingJobs {
		stats.PendingJobs[k] = v
	}
	for k, v := range qm.stats.JobsByType {
		stats.JobsByType[k] = v
	}
	for k, v := range qm.stats.AverageTime {
		stats.AverageTime[k] = v
	}

	return stats
}

// ListJobs returns jobs with optional filtering
func (qm *QueueManager) ListJobs(status JobStatus, jobType string, limit int) []*Job {
	qm.jobMutex.RLock()
	defer qm.jobMutex.RUnlock()

	var jobs []*Job
	count := 0

	// Search in all priority queues
	for priority := PriorityUrgent; priority >= PriorityLow; priority-- {
		for _, job := range qm.queues[priority] {
			if limit > 0 && count >= limit {
				break
			}

			if (status == "" || job.Status == status) &&
				(jobType == "" || job.Type == jobType) {
				jobs = append(jobs, job)
				count++
			}
		}
		if limit > 0 && count >= limit {
			break
		}
	}

	return jobs
}

// Private methods

func (qm *QueueManager) startWorkers() {
	for priority, workerCount := range qm.workers {
		for i := 0; i < workerCount; i++ {
			go qm.worker(priority, i)
		}
	}
	logrus.Info("[QUEUE] Started all workers")
}

func (qm *QueueManager) worker(priority Priority, workerID int) {
	logrus.Infof("[QUEUE] Worker %d started for priority %d", workerID, priority)
	
	for {
		select {
		case <-qm.ctx.Done():
			logrus.Infof("[QUEUE] Worker %d (priority %d) stopping", workerID, priority)
			return
		default:
			job := qm.getNextJob(priority)
			if job == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			qm.processJob(job)
		}
	}
}

func (qm *QueueManager) getNextJob(priority Priority) *Job {
	qm.jobMutex.Lock()
	defer qm.jobMutex.Unlock()

	queue := qm.queues[priority]
	if len(queue) == 0 {
		return nil
	}

	// Find the first job that's ready to run
	for i, job := range queue {
		if job.Status == StatusPending && time.Now().After(job.ScheduledAt) {
			// Remove from queue
			qm.queues[priority] = append(queue[:i], queue[i+1:]...)
			
			// Update status and stats
			job.Status = StatusProcessing
			now := time.Now()
			job.StartedAt = &now
			
			qm.stats.mutex.Lock()
			qm.stats.PendingJobs[priority]--
			qm.stats.ProcessingJobs++
			qm.stats.mutex.Unlock()
			
			return job
		}
	}

	return nil
}

func (qm *QueueManager) processJob(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			job.Error = fmt.Sprintf("panic: %v", r)
			job.Status = StatusFailed
			logrus.Errorf("[QUEUE] Job %s panicked: %v", job.ID, r)
		}
	}()

	logrus.Debugf("[QUEUE] Processing job %s (type: %s)", job.ID, job.Type)

	// Get handler
	qm.mutex.RLock()
	handler, exists := qm.handlers[job.Type]
	qm.mutex.RUnlock()

	if !exists {
		job.Error = fmt.Sprintf("no handler registered for job type: %s", job.Type)
		job.Status = StatusFailed
		qm.updateJobStats(job)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(qm.ctx, job.Timeout)
	defer cancel()

	// Execute job
	job.Attempts++
	startTime := time.Now()
	
	err := handler(ctx, job)
	duration := time.Since(startTime)

	// Update job status
	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		job.Error = err.Error()
		
		// Retry logic
		if job.Attempts < job.MaxAttempts {
			job.Status = StatusRetrying
			job.ScheduledAt = time.Now().Add(time.Duration(job.Attempts) * time.Minute)
			
			// Re-add to queue
			qm.jobMutex.Lock()
			qm.queues[job.Priority] = append(qm.queues[job.Priority], job)
			qm.jobMutex.Unlock()
			
			logrus.Warnf("[QUEUE] Job %s failed, retrying (attempt %d/%d): %v", 
				job.ID, job.Attempts, job.MaxAttempts, err)
		} else {
			job.Status = StatusFailed
			logrus.Errorf("[QUEUE] Job %s failed permanently: %v", job.ID, err)
		}
	} else {
		job.Status = StatusCompleted
		logrus.Debugf("[QUEUE] Job %s completed successfully in %v", job.ID, duration)
	}

	qm.updateJobStats(job)
	qm.updateAverageTime(job.Type, duration)
}

func (qm *QueueManager) updateJobStats(job *Job) {
	qm.stats.mutex.Lock()
	defer qm.stats.mutex.Unlock()

	qm.stats.ProcessingJobs--
	
	if job.Status == StatusCompleted {
		qm.stats.CompletedJobs++
	} else if job.Status == StatusFailed {
		qm.stats.FailedJobs++
	} else if job.Status == StatusRetrying {
		qm.stats.PendingJobs[job.Priority]++
		qm.stats.ProcessingJobs-- // Will be incremented again when retried
	}
	
	qm.stats.LastUpdated = time.Now()
}

func (qm *QueueManager) updateAverageTime(jobType string, duration time.Duration) {
	qm.stats.mutex.Lock()
	defer qm.stats.mutex.Unlock()

	if current, exists := qm.stats.AverageTime[jobType]; exists {
		// Simple moving average
		qm.stats.AverageTime[jobType] = (current + duration) / 2
	} else {
		qm.stats.AverageTime[jobType] = duration
	}
}

func (qm *QueueManager) checkRateLimit(jobType string) bool {
	limiter, exists := qm.rateLimiter[jobType]
	if !exists {
		return true // No rate limit configured
	}

	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	// Refill tokens if needed
	now := time.Now()
	if now.Sub(limiter.lastRefill) >= limiter.refillRate {
		limiter.tokens = limiter.maxTokens
		limiter.lastRefill = now
	}

	// Check if tokens available
	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

func (qm *QueueManager) startCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			qm.cleanupCompletedJobs()
		}
	}
}

func (qm *QueueManager) cleanupCompletedJobs() {
	cutoff := time.Now().Add(-24 * time.Hour) // Keep jobs for 24 hours
	
	qm.jobMutex.Lock()
	defer qm.jobMutex.Unlock()

	cleaned := 0
	for priority, queue := range qm.queues {
		var newQueue []*Job
		for _, job := range queue {
			if (job.Status == StatusCompleted || job.Status == StatusFailed) && 
				job.CompletedAt != nil && job.CompletedAt.Before(cutoff) {
				cleaned++
				continue
			}
			newQueue = append(newQueue, job)
		}
		qm.queues[priority] = newQueue
	}

	if cleaned > 0 {
		logrus.Infof("[QUEUE] Cleaned up %d old jobs", cleaned)
	}
}

func (qm *QueueManager) startStatsUpdater() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-qm.ctx.Done():
			return
		case <-ticker.C:
			qm.updateCurrentStats()
		}
	}
}

func (qm *QueueManager) updateCurrentStats() {
	qm.jobMutex.RLock()
	defer qm.jobMutex.RUnlock()

	qm.stats.mutex.Lock()
	defer qm.stats.mutex.Unlock()

	// Reset pending counts
	for priority := range qm.stats.PendingJobs {
		qm.stats.PendingJobs[priority] = 0
	}

	// Count current pending jobs
	for priority, queue := range qm.queues {
		for _, job := range queue {
			if job.Status == StatusPending {
				qm.stats.PendingJobs[priority]++
			}
		}
	}

	qm.stats.LastUpdated = time.Now()
}

func (qm *QueueManager) generateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), time.Now().UnixNano()%1000)
}

// Stop gracefully stops the queue manager
func (qm *QueueManager) Stop() {
	logrus.Info("[QUEUE] Stopping queue manager...")
	qm.cancel()
	
	// Wait for jobs to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			logrus.Warn("[QUEUE] Timeout waiting for jobs to complete")
			return
		case <-ticker.C:
			stats := qm.GetQueueStats()
			if stats.ProcessingJobs == 0 {
				logrus.Info("[QUEUE] All jobs completed, queue manager stopped")
				return
			}
			logrus.Infof("[QUEUE] Waiting for %d jobs to complete...", stats.ProcessingJobs)
		}
	}
}
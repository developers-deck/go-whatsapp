package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Category  string                 `json:"category"`
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	UserAgent string                 `json:"user_agent,omitempty"`
	IP        string                 `json:"ip,omitempty"`
}

type Analytics struct {
	dataPath string
	events   []Event
	mutex    sync.RWMutex
}

type Report struct {
	Period    string                 `json:"period"`
	StartDate time.Time              `json:"start_date"`
	EndDate   time.Time              `json:"end_date"`
	Summary   map[string]interface{} `json:"summary"`
	Details   map[string]interface{} `json:"details"`
}

func NewAnalytics() *Analytics {
	dataPath := filepath.Join(config.PathStorages, "analytics")
	os.MkdirAll(dataPath, 0755)

	analytics := &Analytics{
		dataPath: dataPath,
		events:   make([]Event, 0),
	}

	// Load existing events
	analytics.loadEvents()

	// Start periodic save
	go analytics.startPeriodicSave()

	return analytics
}

// TrackEvent records a new analytics event
func (a *Analytics) TrackEvent(eventType, category, action string, data map[string]interface{}) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	event := Event{
		ID:        a.generateEventID(),
		Type:      eventType,
		Category:  category,
		Action:    action,
		Data:      data,
		Timestamp: time.Now(),
	}

	a.events = append(a.events, event)
	
	logrus.Debugf("[ANALYTICS] Tracked event: %s/%s/%s", eventType, category, action)
}

// TrackMessageSent tracks when a message is sent
func (a *Analytics) TrackMessageSent(messageType, recipient string, size int64) {
	a.TrackEvent("message", "outbound", "sent", map[string]interface{}{
		"message_type": messageType,
		"recipient":    recipient,
		"size":         size,
	})
}

// TrackMessageReceived tracks when a message is received
func (a *Analytics) TrackMessageReceived(messageType, sender string, size int64) {
	a.TrackEvent("message", "inbound", "received", map[string]interface{}{
		"message_type": messageType,
		"sender":       sender,
		"size":         size,
	})
}

// TrackAPICall tracks API endpoint usage
func (a *Analytics) TrackAPICall(endpoint, method string, statusCode int, duration time.Duration) {
	a.TrackEvent("api", "request", method, map[string]interface{}{
		"endpoint":    endpoint,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	})
}

// TrackLogin tracks login events
func (a *Analytics) TrackLogin(method string, success bool) {
	action := "success"
	if !success {
		action = "failure"
	}
	
	a.TrackEvent("auth", "login", action, map[string]interface{}{
		"method": method,
	})
}

// TrackError tracks error events
func (a *Analytics) TrackError(errorType, message string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["error_message"] = message
	
	a.TrackEvent("error", errorType, "occurred", data)
}

// GenerateReport creates an analytics report for a given period
func (a *Analytics) GenerateReport(period string, startDate, endDate time.Time) *Report {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// Filter events by date range
	var filteredEvents []Event
	for _, event := range a.events {
		if event.Timestamp.After(startDate) && event.Timestamp.Before(endDate) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	report := &Report{
		Period:    period,
		StartDate: startDate,
		EndDate:   endDate,
		Summary:   make(map[string]interface{}),
		Details:   make(map[string]interface{}),
	}

	// Generate summary statistics
	report.Summary["total_events"] = len(filteredEvents)
	report.Summary["period_days"] = int(endDate.Sub(startDate).Hours() / 24)

	// Count events by type
	eventTypes := make(map[string]int)
	categories := make(map[string]int)
	actions := make(map[string]int)
	hourlyDistribution := make(map[int]int)

	for _, event := range filteredEvents {
		eventTypes[event.Type]++
		categories[event.Category]++
		actions[event.Action]++
		hourlyDistribution[event.Timestamp.Hour()]++
	}

	report.Details["event_types"] = eventTypes
	report.Details["categories"] = categories
	report.Details["actions"] = actions
	report.Details["hourly_distribution"] = hourlyDistribution

	// Message-specific analytics
	messageStats := a.generateMessageStats(filteredEvents)
	report.Details["messages"] = messageStats

	// API-specific analytics
	apiStats := a.generateAPIStats(filteredEvents)
	report.Details["api"] = apiStats

	// Error analytics
	errorStats := a.generateErrorStats(filteredEvents)
	report.Details["errors"] = errorStats

	return report
}

// GetDailyReport generates a report for today
func (a *Analytics) GetDailyReport() *Report {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	return a.GenerateReport("daily", startOfDay, endOfDay)
}

// GetWeeklyReport generates a report for the past 7 days
func (a *Analytics) GetWeeklyReport() *Report {
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	
	return a.GenerateReport("weekly", weekAgo, now)
}

// GetMonthlyReport generates a report for the past 30 days
func (a *Analytics) GetMonthlyReport() *Report {
	now := time.Now()
	monthAgo := now.AddDate(0, 0, -30)
	
	return a.GenerateReport("monthly", monthAgo, now)
}

// GetRealTimeStats returns current real-time statistics
func (a *Analytics) GetRealTimeStats() map[string]interface{} {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	now := time.Now()
	lastHour := now.Add(-1 * time.Hour)
	lastDay := now.AddDate(0, 0, -1)

	stats := map[string]interface{}{
		"total_events": len(a.events),
		"last_hour":    0,
		"last_day":     0,
	}

	for _, event := range a.events {
		if event.Timestamp.After(lastHour) {
			stats["last_hour"] = stats["last_hour"].(int) + 1
		}
		if event.Timestamp.After(lastDay) {
			stats["last_day"] = stats["last_day"].(int) + 1
		}
	}

	return stats
}

// Private methods

func (a *Analytics) generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), len(a.events))
}

func (a *Analytics) generateMessageStats(events []Event) map[string]interface{} {
	stats := map[string]interface{}{
		"total_sent":     0,
		"total_received": 0,
		"by_type":        make(map[string]int),
		"total_size":     int64(0),
	}

	for _, event := range events {
		if event.Type == "message" {
			if event.Category == "outbound" {
				stats["total_sent"] = stats["total_sent"].(int) + 1
			} else if event.Category == "inbound" {
				stats["total_received"] = stats["total_received"].(int) + 1
			}

			if msgType, ok := event.Data["message_type"].(string); ok {
				byType := stats["by_type"].(map[string]int)
				byType[msgType]++
			}

			if size, ok := event.Data["size"].(int64); ok {
				stats["total_size"] = stats["total_size"].(int64) + size
			}
		}
	}

	return stats
}

func (a *Analytics) generateAPIStats(events []Event) map[string]interface{} {
	stats := map[string]interface{}{
		"total_requests":   0,
		"by_endpoint":      make(map[string]int),
		"by_method":        make(map[string]int),
		"by_status":        make(map[string]int),
		"avg_duration_ms":  0.0,
	}

	totalDuration := int64(0)
	requestCount := 0

	for _, event := range events {
		if event.Type == "api" {
			stats["total_requests"] = stats["total_requests"].(int) + 1
			requestCount++

			if endpoint, ok := event.Data["endpoint"].(string); ok {
				byEndpoint := stats["by_endpoint"].(map[string]int)
				byEndpoint[endpoint]++
			}

			if method, ok := event.Data["method"].(string); ok {
				byMethod := stats["by_method"].(map[string]int)
				byMethod[method]++
			}

			if statusCode, ok := event.Data["status_code"].(int); ok {
				byStatus := stats["by_status"].(map[string]int)
				statusRange := fmt.Sprintf("%dxx", statusCode/100)
				byStatus[statusRange]++
			}

			if duration, ok := event.Data["duration_ms"].(int64); ok {
				totalDuration += duration
			}
		}
	}

	if requestCount > 0 {
		stats["avg_duration_ms"] = float64(totalDuration) / float64(requestCount)
	}

	return stats
}

func (a *Analytics) generateErrorStats(events []Event) map[string]interface{} {
	stats := map[string]interface{}{
		"total_errors": 0,
		"by_type":      make(map[string]int),
	}

	for _, event := range events {
		if event.Type == "error" {
			stats["total_errors"] = stats["total_errors"].(int) + 1

			byType := stats["by_type"].(map[string]int)
			byType[event.Category]++
		}
	}

	return stats
}

func (a *Analytics) loadEvents() {
	filePath := filepath.Join(a.dataPath, "events.json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Errorf("[ANALYTICS] Failed to read events file: %v", err)
		}
		return
	}

	if err := json.Unmarshal(data, &a.events); err != nil {
		logrus.Errorf("[ANALYTICS] Failed to unmarshal events: %v", err)
		return
	}

	logrus.Infof("[ANALYTICS] Loaded %d events", len(a.events))
}

func (a *Analytics) saveEvents() error {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	filePath := filepath.Join(a.dataPath, "events.json")
	
	data, err := json.MarshalIndent(a.events, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (a *Analytics) startPeriodicSave() {
	ticker := time.NewTicker(5 * time.Minute) // Save every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		if err := a.saveEvents(); err != nil {
			logrus.Errorf("[ANALYTICS] Failed to save events: %v", err)
		} else {
			logrus.Debug("[ANALYTICS] Events saved successfully")
		}
	}
}
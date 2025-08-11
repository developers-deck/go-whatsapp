package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

type WebhookManager struct {
	endpoints   map[string]*WebhookEndpoint
	client      *http.Client
	retryPolicy *RetryPolicy
	mutex       sync.RWMutex
	stats       *WebhookStats
}

type WebhookEndpoint struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Secret      string            `json:"secret"`
	Events      []string          `json:"events"`
	Headers     map[string]string `json:"headers"`
	Timeout     time.Duration     `json:"timeout"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	LastUsed    *time.Time        `json:"last_used,omitempty"`
	SuccessRate float64           `json:"success_rate"`
	TotalCalls  int64             `json:"total_calls"`
	FailedCalls int64             `json:"failed_calls"`
}

type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type WebhookDelivery struct {
	ID           string            `json:"id"`
	EndpointID   string            `json:"endpoint_id"`
	EventID      string            `json:"event_id"`
	URL          string            `json:"url"`
	Status       string            `json:"status"` // pending, success, failed, retrying
	StatusCode   int               `json:"status_code"`
	Response     string            `json:"response"`
	Error        string            `json:"error,omitempty"`
	Attempts     int               `json:"attempts"`
	MaxAttempts  int               `json:"max_attempts"`
	CreatedAt    time.Time         `json:"created_at"`
	DeliveredAt  *time.Time        `json:"delivered_at,omitempty"`
	NextRetryAt  *time.Time        `json:"next_retry_at,omitempty"`
	Headers      map[string]string `json:"headers"`
	Duration     time.Duration     `json:"duration"`
}

type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	BaseDelay   time.Duration `json:"base_delay"`
	MaxDelay    time.Duration `json:"max_delay"`
	Multiplier  float64       `json:"multiplier"`
}

type WebhookStats struct {
	TotalEndpoints   int                        `json:"total_endpoints"`
	ActiveEndpoints  int                        `json:"active_endpoints"`
	TotalDeliveries  int64                      `json:"total_deliveries"`
	SuccessfulDeliveries int64                  `json:"successful_deliveries"`
	FailedDeliveries int64                      `json:"failed_deliveries"`
	AverageResponseTime time.Duration           `json:"average_response_time"`
	DeliveriesByStatus map[string]int64         `json:"deliveries_by_status"`
	DeliveriesByEvent  map[string]int64         `json:"deliveries_by_event"`
	LastUpdated      time.Time                  `json:"last_updated"`
	mutex            sync.RWMutex
}

func NewWebhookManager() *WebhookManager {
	wm := &WebhookManager{
		endpoints: make(map[string]*WebhookEndpoint),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryPolicy: &RetryPolicy{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Second,
			MaxDelay:    60 * time.Second,
			Multiplier:  2.0,
		},
		stats: &WebhookStats{
			DeliveriesByStatus: make(map[string]int64),
			DeliveriesByEvent:  make(map[string]int64),
			LastUpdated:        time.Now(),
		},
	}

	// Load existing webhooks from config
	wm.loadConfiguredWebhooks()

	logrus.Info("[WEBHOOK] Webhook manager initialized")
	return wm
}

// AddEndpoint adds a new webhook endpoint
func (wm *WebhookManager) AddEndpoint(endpoint *WebhookEndpoint) error {
	if endpoint.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if endpoint.ID == "" {
		endpoint.ID = wm.generateEndpointID()
	}

	if endpoint.Timeout == 0 {
		endpoint.Timeout = 30 * time.Second
	}

	if endpoint.Events == nil {
		endpoint.Events = []string{"*"} // All events by default
	}

	if endpoint.Headers == nil {
		endpoint.Headers = make(map[string]string)
	}

	endpoint.CreatedAt = time.Now()
	endpoint.UpdatedAt = time.Now()
	endpoint.Enabled = true

	wm.mutex.Lock()
	wm.endpoints[endpoint.ID] = endpoint
	wm.mutex.Unlock()

	wm.updateStats()
	logrus.Infof("[WEBHOOK] Added endpoint: %s (%s)", endpoint.Name, endpoint.URL)
	return nil
}

// UpdateEndpoint updates an existing webhook endpoint
func (wm *WebhookManager) UpdateEndpoint(id string, updates map[string]interface{}) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	endpoint, exists := wm.endpoints[id]
	if !exists {
		return fmt.Errorf("webhook endpoint not found: %s", id)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		endpoint.Name = name
	}
	if url, ok := updates["url"].(string); ok {
		endpoint.URL = url
	}
	if secret, ok := updates["secret"].(string); ok {
		endpoint.Secret = secret
	}
	if events, ok := updates["events"].([]string); ok {
		endpoint.Events = events
	}
	if headers, ok := updates["headers"].(map[string]string); ok {
		endpoint.Headers = headers
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		endpoint.Enabled = enabled
	}

	endpoint.UpdatedAt = time.Now()
	wm.updateStats()

	logrus.Infof("[WEBHOOK] Updated endpoint: %s", id)
	return nil
}

// RemoveEndpoint removes a webhook endpoint
func (wm *WebhookManager) RemoveEndpoint(id string) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if _, exists := wm.endpoints[id]; !exists {
		return fmt.Errorf("webhook endpoint not found: %s", id)
	}

	delete(wm.endpoints, id)
	wm.updateStats()

	logrus.Infof("[WEBHOOK] Removed endpoint: %s", id)
	return nil
}

// GetEndpoint retrieves a webhook endpoint
func (wm *WebhookManager) GetEndpoint(id string) (*WebhookEndpoint, error) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	endpoint, exists := wm.endpoints[id]
	if !exists {
		return nil, fmt.Errorf("webhook endpoint not found: %s", id)
	}

	return endpoint, nil
}

// ListEndpoints returns all webhook endpoints
func (wm *WebhookManager) ListEndpoints() []*WebhookEndpoint {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	endpoints := make([]*WebhookEndpoint, 0, len(wm.endpoints))
	for _, endpoint := range wm.endpoints {
		endpoints = append(endpoints, endpoint)
	}

	return endpoints
}

// SendEvent sends an event to all matching webhook endpoints
func (wm *WebhookManager) SendEvent(event *WebhookEvent) error {
	if event.ID == "" {
		event.ID = wm.generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	wm.mutex.RLock()
	matchingEndpoints := wm.getMatchingEndpoints(event.Type)
	wm.mutex.RUnlock()

	if len(matchingEndpoints) == 0 {
		logrus.Debugf("[WEBHOOK] No matching endpoints for event type: %s", event.Type)
		return nil
	}

	// Send to all matching endpoints concurrently
	var wg sync.WaitGroup
	for _, endpoint := range matchingEndpoints {
		wg.Add(1)
		go func(ep *WebhookEndpoint) {
			defer wg.Done()
			wm.deliverEvent(ep, event)
		}(endpoint)
	}

	wg.Wait()
	logrus.Debugf("[WEBHOOK] Event %s sent to %d endpoints", event.ID, len(matchingEndpoints))
	return nil
}

// TestEndpoint tests a webhook endpoint with a sample event
func (wm *WebhookManager) TestEndpoint(id string) (*WebhookDelivery, error) {
	endpoint, err := wm.GetEndpoint(id)
	if err != nil {
		return nil, err
	}

	// Create test event
	testEvent := &WebhookEvent{
		ID:   "test_" + wm.generateEventID(),
		Type: "webhook.test",
		Data: map[string]interface{}{
			"message": "This is a test webhook event",
			"test":    true,
		},
		Timestamp: time.Now(),
		Source:    "webhook_manager",
		Metadata: map[string]interface{}{
			"endpoint_id": id,
		},
	}

	return wm.deliverEvent(endpoint, testEvent), nil
}

// GetStats returns webhook statistics
func (wm *WebhookManager) GetStats() *WebhookStats {
	wm.stats.mutex.RLock()
	defer wm.stats.mutex.RUnlock()

	// Create a copy to avoid race conditions
	stats := &WebhookStats{
		TotalEndpoints:       wm.stats.TotalEndpoints,
		ActiveEndpoints:      wm.stats.ActiveEndpoints,
		TotalDeliveries:      wm.stats.TotalDeliveries,
		SuccessfulDeliveries: wm.stats.SuccessfulDeliveries,
		FailedDeliveries:     wm.stats.FailedDeliveries,
		AverageResponseTime:  wm.stats.AverageResponseTime,
		DeliveriesByStatus:   make(map[string]int64),
		DeliveriesByEvent:    make(map[string]int64),
		LastUpdated:          wm.stats.LastUpdated,
	}

	for k, v := range wm.stats.DeliveriesByStatus {
		stats.DeliveriesByStatus[k] = v
	}
	for k, v := range wm.stats.DeliveriesByEvent {
		stats.DeliveriesByEvent[k] = v
	}

	return stats
}

// Private methods

func (wm *WebhookManager) loadConfiguredWebhooks() {
	// Load webhooks from existing config
	for i, url := range config.WhatsappWebhook {
		endpoint := &WebhookEndpoint{
			ID:      fmt.Sprintf("config_%d", i),
			Name:    fmt.Sprintf("Configured Webhook %d", i+1),
			URL:     url,
			Secret:  config.WhatsappWebhookSecret,
			Events:  []string{"*"},
			Headers: make(map[string]string),
			Timeout: 30 * time.Second,
			Enabled: true,
		}
		wm.AddEndpoint(endpoint)
	}
}

func (wm *WebhookManager) getMatchingEndpoints(eventType string) []*WebhookEndpoint {
	var matching []*WebhookEndpoint

	for _, endpoint := range wm.endpoints {
		if !endpoint.Enabled {
			continue
		}

		// Check if endpoint accepts this event type
		for _, acceptedEvent := range endpoint.Events {
			if acceptedEvent == "*" || acceptedEvent == eventType {
				matching = append(matching, endpoint)
				break
			}
		}
	}

	return matching
}

func (wm *WebhookManager) deliverEvent(endpoint *WebhookEndpoint, event *WebhookEvent) *WebhookDelivery {
	delivery := &WebhookDelivery{
		ID:          wm.generateDeliveryID(),
		EndpointID:  endpoint.ID,
		EventID:     event.ID,
		URL:         endpoint.URL,
		Status:      "pending",
		Attempts:    0,
		MaxAttempts: wm.retryPolicy.MaxAttempts,
		CreatedAt:   time.Now(),
		Headers:     make(map[string]string),
	}

	// Attempt delivery with retries
	for delivery.Attempts < delivery.MaxAttempts {
		delivery.Attempts++
		startTime := time.Now()

		success := wm.attemptDelivery(endpoint, event, delivery)
		delivery.Duration = time.Since(startTime)

		if success {
			delivery.Status = "success"
			now := time.Now()
			delivery.DeliveredAt = &now
			endpoint.LastUsed = &now
			break
		}

		// Calculate next retry time
		if delivery.Attempts < delivery.MaxAttempts {
			delay := wm.calculateRetryDelay(delivery.Attempts)
			nextRetry := time.Now().Add(delay)
			delivery.NextRetryAt = &nextRetry
			delivery.Status = "retrying"
			
			logrus.Warnf("[WEBHOOK] Delivery failed, retrying in %v (attempt %d/%d): %s", 
				delay, delivery.Attempts, delivery.MaxAttempts, delivery.Error)
			
			time.Sleep(delay)
		} else {
			delivery.Status = "failed"
			logrus.Errorf("[WEBHOOK] Delivery failed permanently after %d attempts: %s", 
				delivery.Attempts, delivery.Error)
		}
	}

	// Update endpoint stats
	wm.updateEndpointStats(endpoint, delivery)
	wm.updateGlobalStats(delivery)

	return delivery
}

func (wm *WebhookManager) attemptDelivery(endpoint *WebhookEndpoint, event *WebhookEvent, delivery *WebhookDelivery) bool {
	// Prepare payload
	payload, err := json.Marshal(event)
	if err != nil {
		delivery.Error = fmt.Sprintf("failed to marshal event: %v", err)
		return false
	}

	// Create request
	ctx, cancel := context.WithTimeout(context.Background(), endpoint.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewBuffer(payload))
	if err != nil {
		delivery.Error = fmt.Sprintf("failed to create request: %v", err)
		return false
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("WhatsApp-Webhook/%s", config.AppVersion))
	
	for key, value := range endpoint.Headers {
		req.Header.Set(key, value)
		delivery.Headers[key] = value
	}

	// Add signature if secret is configured
	if endpoint.Secret != "" {
		signature := wm.generateSignature(payload, endpoint.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
		delivery.Headers["X-Webhook-Signature"] = signature
	}

	// Send request
	resp, err := wm.client.Do(req)
	if err != nil {
		delivery.Error = fmt.Sprintf("request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	delivery.StatusCode = resp.StatusCode

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		delivery.Response = "failed to read response"
	} else {
		delivery.Response = string(responseBody)
	}

	// Check if delivery was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}

	delivery.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, delivery.Response)
	return false
}

func (wm *WebhookManager) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func (wm *WebhookManager) calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(float64(wm.retryPolicy.BaseDelay) * 
		(wm.retryPolicy.Multiplier * float64(attempt-1)))
	
	if delay > wm.retryPolicy.MaxDelay {
		delay = wm.retryPolicy.MaxDelay
	}
	
	return delay
}

func (wm *WebhookManager) updateEndpointStats(endpoint *WebhookEndpoint, delivery *WebhookDelivery) {
	endpoint.TotalCalls++
	
	if delivery.Status == "success" {
		endpoint.SuccessRate = float64(endpoint.TotalCalls-endpoint.FailedCalls) / float64(endpoint.TotalCalls)
	} else if delivery.Status == "failed" {
		endpoint.FailedCalls++
		endpoint.SuccessRate = float64(endpoint.TotalCalls-endpoint.FailedCalls) / float64(endpoint.TotalCalls)
	}
}

func (wm *WebhookManager) updateGlobalStats(delivery *WebhookDelivery) {
	wm.stats.mutex.Lock()
	defer wm.stats.mutex.Unlock()

	wm.stats.TotalDeliveries++
	wm.stats.DeliveriesByStatus[delivery.Status]++
	
	if delivery.Status == "success" {
		wm.stats.SuccessfulDeliveries++
	} else if delivery.Status == "failed" {
		wm.stats.FailedDeliveries++
	}

	// Update average response time
	if wm.stats.AverageResponseTime == 0 {
		wm.stats.AverageResponseTime = delivery.Duration
	} else {
		wm.stats.AverageResponseTime = (wm.stats.AverageResponseTime + delivery.Duration) / 2
	}

	wm.stats.LastUpdated = time.Now()
}

func (wm *WebhookManager) updateStats() {
	wm.stats.mutex.Lock()
	defer wm.stats.mutex.Unlock()

	wm.stats.TotalEndpoints = len(wm.endpoints)
	wm.stats.ActiveEndpoints = 0

	for _, endpoint := range wm.endpoints {
		if endpoint.Enabled {
			wm.stats.ActiveEndpoints++
		}
	}

	wm.stats.LastUpdated = time.Now()
}

func (wm *WebhookManager) generateEndpointID() string {
	return fmt.Sprintf("endpoint_%d", time.Now().UnixNano())
}

func (wm *WebhookManager) generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

func (wm *WebhookManager) generateDeliveryID() string {
	return fmt.Sprintf("delivery_%d", time.Now().UnixNano())
}
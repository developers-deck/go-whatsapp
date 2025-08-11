package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type Client struct {
	ID           string                 `json:"id"`
	Connection   *websocket.Conn        `json:"-"`
	ConnectedAt  time.Time              `json:"connected_at"`
	LastPing     time.Time              `json:"last_ping"`
	Subscriptions map[string]bool       `json:"subscriptions"`
	Metadata     map[string]interface{} `json:"metadata"`
	mutex        sync.RWMutex           `json:"-"`
}

type BroadcastMessage struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Result    any                    `json:"result"`
	Timestamp time.Time              `json:"timestamp"`
	Channel   string                 `json:"channel,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type WebSocketMessage struct {
	Type    string                 `json:"type"`
	Channel string                 `json:"channel,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

var (
	Clients    = make(map[string]*Client)
	clientsMux = sync.RWMutex{}
	Register   = make(chan *Client)
	Broadcast  = make(chan BroadcastMessage)
	Unregister = make(chan *Client)
	
	// Channels for different message types
	channels = map[string]bool{
		"whatsapp":   true,
		"system":     true,
		"health":     true,
		"files":      true,
		"monitoring": true,
	}
)

func handleRegister(client *Client) {
	clientsMux.Lock()
	defer clientsMux.Unlock()
	
	Clients[client.ID] = client
	logrus.Infof("[WS] Client registered: %s", client.ID)
	
	// Send welcome message
	welcomeMsg := BroadcastMessage{
		Code:      "CONNECTED",
		Message:   "WebSocket connection established",
		Timestamp: time.Now(),
		Channel:   "system",
		Result: map[string]interface{}{
			"client_id":    client.ID,
			"connected_at": client.ConnectedAt,
			"channels":     getAvailableChannels(),
		},
	}
	
	sendToClient(client, welcomeMsg)
}

func handleUnregister(client *Client) {
	clientsMux.Lock()
	defer clientsMux.Unlock()
	
	delete(Clients, client.ID)
	logrus.Infof("[WS] Client unregistered: %s", client.ID)
}

func broadcastMessage(message BroadcastMessage) {
	message.Timestamp = time.Now()
	
	clientsMux.RLock()
	defer clientsMux.RUnlock()
	
	for _, client := range Clients {
		// Check if client is subscribed to this channel
		if message.Channel != "" && !client.IsSubscribed(message.Channel) {
			continue
		}
		
		sendToClient(client, message)
	}
}

func sendToClient(client *Client, message BroadcastMessage) {
	marshalMessage, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("[WS] Marshal error for client %s: %v", client.ID, err)
		return
	}

	if err := client.Connection.WriteMessage(websocket.TextMessage, marshalMessage); err != nil {
		logrus.Errorf("[WS] Write error for client %s: %v", client.ID, err)
		closeConnection(client)
	}
}

func closeConnection(client *Client) {
	if err := client.Connection.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
		logrus.Errorf("[WS] Write close message error for client %s: %v", client.ID, err)
	}
	if err := client.Connection.Close(); err != nil {
		logrus.Errorf("[WS] Close connection error for client %s: %v", client.ID, err)
	}
	
	clientsMux.Lock()
	delete(Clients, client.ID)
	clientsMux.Unlock()
}

func RunHub() {
	// Start periodic ping to keep connections alive
	go startPingTicker()
	
	for {
		select {
		case client := <-Register:
			handleRegister(client)

		case client := <-Unregister:
			handleUnregister(client)

		case message := <-Broadcast:
			logrus.Debugf("[WS] Broadcasting message: %s to channel: %s", message.Code, message.Channel)
			broadcastMessage(message)
		}
	}
}

func RegisterRoutes(app fiber.Router, service domainApp.IAppUsecase) {
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		// Create new client
		client := &Client{
			ID:            generateClientID(),
			Connection:    conn,
			ConnectedAt:   time.Now(),
			LastPing:      time.Now(),
			Subscriptions: make(map[string]bool),
			Metadata:      make(map[string]interface{}),
		}
		
		// Subscribe to default channels
		client.Subscribe("system")
		client.Subscribe("whatsapp")

		defer func() {
			Unregister <- client
			_ = conn.Close()
		}()

		Register <- client

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logrus.Errorf("[WS] Read error for client %s: %v", client.ID, err)
				}
				return
			}

			if messageType == websocket.TextMessage {
				handleClientMessage(client, message, service)
			} else if messageType == websocket.PongMessage {
				client.UpdateLastPing()
			} else {
				logrus.Warnf("[WS] Unsupported message type from client %s: %d", client.ID, messageType)
			}
		}
	}))
}

// Client methods
func (c *Client) Subscribe(channel string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Subscriptions[channel] = true
}

func (c *Client) Unsubscribe(channel string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.Subscriptions, channel)
}

func (c *Client) IsSubscribed(channel string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Subscriptions[channel]
}

func (c *Client) UpdateLastPing() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.LastPing = time.Now()
}

// Helper functions
func generateClientID() string {
	return time.Now().Format("20060102150405") + "_" + 
		   string(rune(65 + time.Now().UnixNano()%26)) + 
		   string(rune(65 + (time.Now().UnixNano()/1000)%26))
}

func getAvailableChannels() []string {
	var channelList []string
	for channel := range channels {
		channelList = append(channelList, channel)
	}
	return channelList
}

func handleClientMessage(client *Client, message []byte, service domainApp.IAppUsecase) {
	var wsMessage WebSocketMessage
	if err := json.Unmarshal(message, &wsMessage); err != nil {
		logrus.Errorf("[WS] Unmarshal error for client %s: %v", client.ID, err)
		return
	}

	switch wsMessage.Type {
	case "FETCH_DEVICES":
		devices, _ := service.FetchDevices(context.Background())
		response := BroadcastMessage{
			Code:    "LIST_DEVICES",
			Message: "Devices retrieved",
			Result:  devices,
			Channel: "whatsapp",
		}
		sendToClient(client, response)

	case "SUBSCRIBE":
		if channel, ok := wsMessage.Data["channel"].(string); ok {
			if channels[channel] {
				client.Subscribe(channel)
				response := BroadcastMessage{
					Code:    "SUBSCRIBED",
					Message: "Subscribed to channel: " + channel,
					Channel: "system",
					Result:  map[string]interface{}{"channel": channel},
				}
				sendToClient(client, response)
			}
		}

	case "UNSUBSCRIBE":
		if channel, ok := wsMessage.Data["channel"].(string); ok {
			client.Unsubscribe(channel)
			response := BroadcastMessage{
				Code:    "UNSUBSCRIBED",
				Message: "Unsubscribed from channel: " + channel,
				Channel: "system",
				Result:  map[string]interface{}{"channel": channel},
			}
			sendToClient(client, response)
		}

	case "GET_HEALTH":
		health, _ := service.GetSessionHealth(context.Background())
		response := BroadcastMessage{
			Code:    "HEALTH_STATUS",
			Message: "Health status retrieved",
			Result:  health,
			Channel: "health",
		}
		sendToClient(client, response)

	case "PING":
		client.UpdateLastPing()
		response := BroadcastMessage{
			Code:    "PONG",
			Message: "Pong",
			Channel: "system",
			Result:  map[string]interface{}{"timestamp": time.Now()},
		}
		sendToClient(client, response)

	default:
		logrus.Warnf("[WS] Unknown message type from client %s: %s", client.ID, wsMessage.Type)
	}
}

func startPingTicker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		clientsMux.RLock()
		for _, client := range Clients {
			// Check if client is still alive (last ping within 60 seconds)
			if time.Since(client.LastPing) > 60*time.Second {
				logrus.Warnf("[WS] Client %s appears to be dead, closing connection", client.ID)
				closeConnection(client)
				continue
			}

			// Send ping
			if err := client.Connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logrus.Errorf("[WS] Failed to send ping to client %s: %v", client.ID, err)
				closeConnection(client)
			}
		}
		clientsMux.RUnlock()
	}
}

// Utility functions for broadcasting to specific channels
func BroadcastToChannel(channel string, message BroadcastMessage) {
	message.Channel = channel
	Broadcast <- message
}

func GetConnectedClients() map[string]*Client {
	clientsMux.RLock()
	defer clientsMux.RUnlock()
	
	result := make(map[string]*Client)
	for id, client := range Clients {
		result[id] = client
	}
	return result
}

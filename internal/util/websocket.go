package util

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"portolio-backend/internal/model/dto"
)

type Client struct {
	UserID uint
	Conn   *websocket.Conn
	Send   chan []byte 
	Stop chan struct{}
}

type Hub struct {
	clients    map[uint][]*Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan []byte
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint][]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan []byte),
	}
}

func (h *Hub) RegisterClient(client *Client) {
	h.Register <- client
}

func (h *Hub) UnregisterClient(client *Client) {
	h.Unregister <- client
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; !ok {
				h.clients[client.UserID] = []*Client{}
			}
			h.clients[client.UserID] = append(h.clients[client.UserID], client)
			h.mu.Unlock()
			log.Printf("Client registered: UserID %d, Total connections for user: %d. Send buffer size: %d", client.UserID, len(h.clients[client.UserID]), cap(client.Send))

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				found := false
				for i, c := range clients {
					if c == client {
						close(c.Stop) 
						h.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
						found = true
						break
					}
				}
				if found {
					if len(h.clients[client.UserID]) == 0 {
						delete(h.clients, client.UserID)
					}
					select {
					case <-client.Send: 
					default:
						close(client.Send)
					}
					log.Printf("Client unregistered: UserID %d. Remaining connections for user: %d", client.UserID, len(h.clients[client.UserID]))
				} else {
					log.Printf("Attempted to unregister a client for UserID %d that was not found.", client.UserID)
				}
			} else {
				log.Printf("Attempted to unregister a client for UserID %d but no clients were found for that ID.", client.UserID)
			}
			h.mu.Unlock()

		case message := <-h.Broadcast:
			h.mu.RLock()
			for userID, clients := range h.clients {
				for _, client := range clients {
					select {
					case client.Send <- message:
					default:
						log.Printf("Failed to send broadcast message to UserID %d (channel full or blocked). Consider client health.", userID)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func SendNotificationToUser(userID uint, notification dto.NotificationResponse) {
	jsonMsg, err := json.Marshal(map[string]interface{}{
		"type":    "notification",
		"message": notification,
	})
	if err != nil {
		log.Printf("Error marshalling notification: %v", err)
		return
	}

	SendToUser(userID, jsonMsg)
}

func SendToUser(userID uint, message []byte) {
	if WebsocketHub == nil {
		log.Println("WebsocketHub is not initialized.")
		return
	}

	WebsocketHub.mu.RLock()
	defer WebsocketHub.mu.RUnlock()

	if clients, ok := WebsocketHub.clients[userID]; ok && len(clients) > 0 {
		for _, client := range clients {
			select {
			case client.Send <- message:
				log.Printf("Message sent to UserID %d via one of their connections.", userID)
			default:
				log.Printf("Failed to send message to UserID %d (channel full) for connection %p. Consider its health.", userID, client.Conn)
			}
		}
	} else {
		log.Printf("No active WebSocket connection for UserID %d.", userID)
	}
}

// Global Hub instance (initialized in main.go)
var WebsocketHub *Hub
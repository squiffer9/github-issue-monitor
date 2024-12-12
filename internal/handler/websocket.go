package handler

import (
	"github-issue-monitor/internal/models"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type WebSocketHandler struct {
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	upgrader     websocket.Upgrader
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Development only
			},
			HandshakeTimeout: 10 * time.Second,
		},
	}
}

// HandleConnection is an HTTP handler that upgrades the connection to a WebSocket
func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	h.clientsMutex.Lock()
	h.clients[conn] = true
	h.clientsMutex.Unlock()

	// Send a connection confirmation message
	conn.WriteMessage(websocket.TextMessage,
		[]byte("Connected to GitHub Issue Monitor. Waiting for events..."))

	// Set up Keep-Alive
	conn.SetPingHandler(func(string) error {
		return conn.WriteControl(websocket.PongMessage, []byte{},
			time.Now().Add(10*time.Second))
	})

	// Wait until the connection is closed
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			h.clientsMutex.Lock()
			delete(h.clients, conn)
			h.clientsMutex.Unlock()
			return
		}
	}
}

// BroadcastEvent sends the event to all connected clients
func (h *WebSocketHandler) BroadcastEvent(event *models.IssueEvent) {
	message := event.FormatMessage()

	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()

	for client := range h.clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			client.Close()
			h.removeClient(client)
		}
	}
}

// removeClient removes a client from the list of connected clients
func (h *WebSocketHandler) removeClient(conn *websocket.Conn) {
	h.clientsMutex.Lock()
	delete(h.clients, conn)
	h.clientsMutex.Unlock()
}

// ClientCount returns the number of connected clients
func (h *WebSocketHandler) ClientCount() int {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	return len(h.clients)
}

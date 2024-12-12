package handler

import (
	"encoding/json"
	"github-issue-monitor/internal/models"
	"io"
	"log"
	"net/http"
)

type WebhookHandler struct {
	wsHandler *WebSocketHandler
}

func NewWebhookHandler(wsHandler *WebSocketHandler) *WebhookHandler {
	return &WebhookHandler{
		wsHandler: wsHandler,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check the event type from GitHub
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "issues" {
		log.Printf("Received non-issue event: %s", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse the event
	var event models.IssueEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Failed to parse webhook payload", http.StatusBadRequest)
		return
	}

	// Broadcast to WebSocket clients
	go h.wsHandler.BroadcastEvent(&event)

	w.WriteHeader(http.StatusOK)
}

package main

import (
	"github-issue-monitor/internal/handler"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Initialize handlers
	wsHandler := handler.NewWebSocketHandler()
	webhookHandler := handler.NewWebhookHandler(wsHandler)

	// Set up routes
	http.HandleFunc("/ws", wsHandler.HandleConnection)
	http.HandleFunc("/webhook", webhookHandler.HandleWebhook)

	// Announce server startup
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	//log.Printf("Starting server on port %s...", port)
	//log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
	//log.Printf("Webhook endpoint: http://localhost:%s/webhook", port)

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the server
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("Server error:", err)
		}
	}()

	// Waith for the signal
	<-signalChan
	log.Println("Shutting down server...")
}

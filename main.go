package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/bluele/gcache"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nithish-95/chat-webapp/cleaner"
	"github.com/nithish-95/chat-webapp/database"
	"github.com/nithish-95/chat-webapp/handlers"
	"github.com/nithish-95/chat-webapp/ui"
)

func main() {
	// Create a new router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	ctx := context.Background()

	// Initialize the SQLite database
	dao, err := database.NewSQLite("./chat_messages.db")
	if err != nil {
		log.Fatalf("Cannot open connection to database: %v", err)
	}
	// Initialize the database schema
	err = dao.InitDatabase(ctx)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer dao.Close(ctx)

	// Create an active channels cache using gcache
	activeChannels := gcache.New(20).Simple().Expiration(time.Minute).Build()

	// Create a handler with injected dependencies (activeChannels and dao)
	h := handlers.NewHandler(activeChannels, dao)

	// Create a cleaner to periodically delete old messages
	c := cleaner.NewTTL(dao, 5*time.Minute)
	go c.StartCleaning()

	// Start handling WebSocket connections and broadcasting messages
	go handleConnections(h)

	// Set up the UI and routes using the UI controller
	ui, err := ui.NewUIEndpoint(ui.Controller{})
	if err != nil {
		log.Fatal("Error creating UI endpoint:", err)
	}
	r.Mount(ui.Path(), ui.Routes())

	// Define the HTTP routes
	r.Get("/Active/channels", h.GetActiveChannels)
	r.Get("/ws/{channel}", h.WebSocketHandler(ctx))

	// Start the server on port 8080
	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", r)
}

// handleConnections manages WebSocket client connections and message broadcasts
func handleConnections(h handlers.Handler) {
	for {
		select {
		case client := <-h.Register():
			// Handle client registration (add client to the channel)
			log.Printf("Client registered: %v", client)

			// You can add additional logic here to manage activeChannels if necessary

		case client := <-h.Unregister():
			// Handle client unregistration (remove client from the channel)
			log.Printf("Client unregistered: %v", client)

			// You can add logic here to handle the removal of users from activeChannels if needed

		case message := <-h.Broadcast():
			// Handle broadcasting messages to clients in the same channel
			log.Printf("Broadcasting message: %v", message)

			// Implement the logic to send the message to all clients in the relevant channel
		}
	}
}

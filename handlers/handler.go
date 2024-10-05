package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/nithish-95/chat-webapp/database"
	"github.com/nithish-95/chat-webapp/models"
)

type Channel struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
	Timer *time.Timer
}

type Client struct {
	Conn     *websocket.Conn
	Username string
	Channel  string
}

type handler struct {
	activeChannels gcache.Cache
	register       chan Client
	unregister     chan Client
	broadcast      chan models.Message
	mutex          sync.Mutex
	upgrader       websocket.Upgrader
	dao            database.DAO // Injected dependency
}

// NewHandler creates a new handler instance with injected dependencies
func NewHandler(activeChannels gcache.Cache, dao database.DAO) Handler {
	return &handler{
		activeChannels: activeChannels,
		register:       make(chan Client),
		unregister:     make(chan Client),
		broadcast:      make(chan models.Message),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		dao: dao, // Dependency injection
	}
}

// GetActiveChannels returns the list of active channels
func (h *handler) GetActiveChannels(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var channels []string
	keys := h.activeChannels.Keys(false)
	for _, key := range keys {
		channels = append(channels, key.(string))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// WebSocketHandler handles the WebSocket connection for a channel
func (h *handler) WebSocketHandler(ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelName := chi.URLParam(r, "channel")
		userName := r.URL.Query().Get("UserName")

		conn, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}

		client := Client{Conn: conn, Username: userName, Channel: channelName}
		h.register <- client

		// Fetch and send old messages to the new client
		messages, err := h.dao.GetMessages(ctx, channelName) // Added context
		if err != nil {
			log.Printf("error retrieving messages: %v", err)
			return
		}

		for _, msg := range messages {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("error sending message: %v", err)
				break
			}
		}

		go h.handleClientConnection(ctx, client, channelName, userName)
	}
}

// handleClientConnection handles incoming messages from the WebSocket client
func (h *handler) handleClientConnection(ctx context.Context, client Client, channelName, userName string) {
	defer func() {
		h.unregister <- client
	}()

	for {
		var msg models.Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			break
		}
		msg.Channel = channelName
		msg.Username = userName
		h.broadcast <- msg
	}
}

// Register provides access to the register channel
func (h *handler) Register() chan Client {
	return h.register
}

// Unregister provides access to the unregister channel
func (h *handler) Unregister() chan Client {
	return h.unregister
}

// Broadcast provides access to the broadcast channel
func (h *handler) Broadcast() chan models.Message {
	return h.broadcast
}

package main

import (
	"chat-webapp/channels"
	"chat-webapp/messages"
	"chat-webapp/storage"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

type Server struct {
	Router         *chi.Mux
	DB             storage.DB
	ActiveChannels gcache.Cache
	Mutex          sync.Mutex
	Upgrader       websocket.Upgrader
}

// GetDB returns the database instance.
func (s *Server) GetDB() storage.DB {
	return s.DB
}

// GetActiveChannels returns the active channels cache.
func (s *Server) GetActiveChannels() gcache.Cache {
	return s.ActiveChannels
}

// GetMutex returns the server's mutex.
func (s *Server) GetMutex() *sync.Mutex {
	return &s.Mutex
}

// NewServer initializes a new Server instance.
func NewServer() *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	db := storage.NewSQLiteDB("./chat_messages.db")
	if db == nil {
		log.Fatal("Failed to initialize database")
	}

	activeChannels := gcache.New(20).Simple().Expiration(5 * time.Minute).Build()

	return &Server{
		Router:         r,
		DB:             db,
		ActiveChannels: activeChannels,
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) GetActiveChannelsMap() map[string]bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	activeChannelsMap := make(map[string]bool)
	keys := s.ActiveChannels.Keys(false)
	for _, key := range keys {
		channelName, ok := key.(string)
		if ok {
			activeChannelsMap[channelName] = true
		}
	}
	return activeChannelsMap
}

func GetIndexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "src/index.html")
}

func GetChannelsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "src/channels.html")
}

func GetChatPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "src/chat.html")
}

func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	channelName := chi.URLParam(r, "channel")
	userName := r.URL.Query().Get("UserName")

	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}

	client := channels.Client{Conn: conn, Username: userName, Channel: channelName}
	channels.Register <- client

	// Fetch and send old messages
	messages, err := s.DB.GetMessages(channelName, 5*time.Minute)
	if err != nil {
		log.Printf("Error retrieving messages: %v", err)
		return
	}

	for _, msg := range messages {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}
	}

	go func() {
		defer func() {
			channels.Unregister <- client
		}()

		for {
			var msg storage.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("Error: %v", err)
				break
			}
			msg.Channel = channelName
			msg.Username = userName
			// Convert storage.Message to messages.Message
			convertedMsg := convertStorageMessage(msg)
			channels.Broadcast <- convertedMsg
		}
	}()
}
func convertStorageMessage(msg storage.Message) messages.Message {
	return messages.Message{
		Channel:  msg.Channel,
		Username: msg.Username,
		Content:  msg.Content,
		Time:     msg.Time,
	}
}

func (s *Server) Run() {
	go channels.HandleConnections(s)

	s.Router.Get("/", GetIndexPage)
	s.Router.Get("/channels", GetChannelsPage)
	s.Router.Get("/chat", GetChatPage)
	s.Router.Get("/Active/channels", func(w http.ResponseWriter, r *http.Request) {
		activeChannelsMap := s.GetActiveChannelsMap()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(activeChannelsMap); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			log.Printf("Error encoding active channels JSON: %v", err)
		}
	})

	s.Router.Get("/ws/{channel}", s.websocketHandler)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", s.Router))
}

func main() {
	server := NewServer()
	server.Run()
}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nithish-95/chat-webapp/cleaner"
	"github.com/nithish-95/chat-webapp/database"
	"github.com/nithish-95/chat-webapp/models"
)

var (
	activeChannels = gcache.New(20).Simple().Expiration(time.Minute).Build()
	mutex          sync.Mutex
	upgrader       = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	register   = make(chan Client)
	unregister = make(chan Client)
	broadcast  = make(chan models.Message)
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

func GetIndexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func GetChannelsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "channels.html")
}

func GetChatPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "chat.html")
}

func GetActiveChannels(w http.ResponseWriter, r *http.Request) {
	var channels []string
	keys := activeChannels.Keys(false)
	for _, key := range keys {
		channels = append(channels, key.(string))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// Handles the channel connections
// also updates the database before broadcasting the message
func handleConnections(db *sql.DB) {
	clients := make(map[*websocket.Conn]Client)

	for {
		select {
		case client := <-register:
			clients[client.Conn] = client
			updateChannel(client, true)

		case client := <-unregister:
			if _, ok := clients[client.Conn]; ok {
				delete(clients, client.Conn)
				client.Conn.Close()
				updateChannel(client, false)
			}

		case message := <-broadcast:
			log.Printf("Broadcasting message: %v", message)
			insertMessage(db, message.Channel, message.Username, message.Content)

			for conn, client := range clients {
				if client.Channel == message.Channel {
					if err := conn.WriteJSON(message); err != nil {
						conn.Close()
						delete(clients, conn)
					}
				}
			}
		}
	}
}

// Updates the channel by Adding or Removing a User
func updateChannel(client Client, add bool) {
	mutex.Lock()
	defer mutex.Unlock()

	channel, err := activeChannels.Get(client.Channel)
	if err != nil {
		if add {
			// Create a new channel with the user
			newChannel := Channel{Name: client.Channel, Users: []string{client.Username}}
			activeChannels.Set(client.Channel, newChannel)
		}
		return
	}

	ch := channel.(Channel)
	if add {
		// Stop any existing timer since a user has joined
		if ch.Timer != nil {
			ch.Timer.Stop()
			ch.Timer = nil
		}
		ch.Users = append(ch.Users, client.Username)
	} else {
		// Remove the user from the channel
		for i, user := range ch.Users {
			if user == client.Username {
				ch.Users = append(ch.Users[:i], ch.Users[i+1:]...)
				break
			}
		}

		if len(ch.Users) == 0 {
			log.Printf("No users left in channel %s", client.Channel)
			// Start a 5-minute timer before removing the channel
			ch.Timer = time.AfterFunc(1*time.Minute, func() {
				mutex.Lock()
				defer mutex.Unlock()
				activeChannels.Remove(client.Channel)
			})
		}
	}
	activeChannels.Set(client.Channel, ch)
}

// Upgrades the http connection to a websocket connection
func websocketHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelName := chi.URLParam(r, "channel")
		userName := r.URL.Query().Get("UserName")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
			return
		}

		client := Client{Conn: conn, Username: userName, Channel: channelName}
		register <- client

		// Fetch and send old messages to the new client
		messages, err := getMessages(db, channelName)
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

		go func() {
			defer func() {
				unregister <- client
			}()

			for {
				var msg Message
				err := conn.ReadJSON(&msg)
				if err != nil {
					log.Printf("error: %v", err)
					break
				}
				msg.Channel = channelName
				msg.Username = userName
				broadcast <- msg
			}
		}()
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	ctx := context.Background()

	// Initialize the database
	dao, err := database.NewSQLlite("./chat_messages.db")
	if err != nil {
		log.Fatalf("Cannot open conn to db +v", err)
	}
	err = dao.InitDatabase(ctx)
	if err != nil {
		log.Fatalf("error calling initDatabase +v", err)
	}
	defer dao.Close(ctx)

	c := cleaner.NewTTL(dao, 5*time.Minute)
	go c.StartCleaning()

	go handleConnections(db)

	r.Get("/", GetIndexPage)
	r.Get("/channels", GetChannelsPage)
	r.Get("/chat", GetChatPage)
	r.Get("/Active/channels", GetActiveChannels)

	// Use the combined handler
	r.Get("/ws/{channel}", websocketHandler(db))

	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", r)
}

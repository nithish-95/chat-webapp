Certainly! Let's break down the code step by step to help you understand how it works.

### 1. Importing Packages
```go
import (
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
)
```
- **`database/sql`**: Used for interacting with SQL databases.
- **`encoding/json`**: Allows encoding and decoding of JSON data.
- **`log`**: Provides logging capabilities.
- **`net/http`**: Handles HTTP requests and responses.
- **`sync`**: Provides synchronization primitives like `Mutex`.
- **`time`**: Allows working with date and time.
- **`gcache`**: A caching library used to store active channels.
- **`chi`**: A lightweight HTTP router for building Go applications.
- **`websocket`**: A package for WebSocket communication.
- **`sqlite3`**: A driver for SQLite databases.

### 2. Declaring Global Variables
```go
var (
	activeChannels = gcache.New(20).Simple().Build()
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
	broadcast  = make(chan Message)
)
```
- **`activeChannels`**: A cache to store active channels and their users.
- **`mutex`**: Ensures that only one goroutine accesses shared data at a time.
- **`upgrader`**: Configures WebSocket settings, such as buffer size and origin checking.
- **`register`**: A channel to register new clients (users).
- **`unregister`**: A channel to unregister clients who disconnect.
- **`broadcast`**: A channel to send messages to all clients in a specific channel.

### 3. Defining Structs
```go
type Channel struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
}

type Client struct {
	Conn     *websocket.Conn
	Username string
	Channel  string
}

type Message struct {
	Channel  string    `json:"channel"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	Time     time.Time `json:"time"`
}
```
- **`Channel`**: Represents a chat channel with a name and a list of users.
- **`Client`**: Represents a user connected to a WebSocket with a username and channel.
- **`Message`**: Represents a message sent in a channel, including the sender's name, the content, and the time it was sent.

### 4. Initializing the Database
```go
func initDatabase() *sql.DB {
	db, err := sql.Open("sqlite3", "./chat_messages.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		username TEXT NOT NULL,
		message TEXT NOT NULL,
		time DATETIME DEFAULT CURRENT_TIME
	);`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()

	return db
}
```
- **`initDatabase()`**: Opens an SQLite database and creates a table for storing messages if it doesn't already exist. The table has fields for `channel`, `username`, `message`, and `time`.

### 5. Inserting a Message into the Database
```go
func insertMessage(db *sql.DB, channel, username, message string) {
	insertSQL := `INSERT INTO messages(channel, username, message, time) VALUES (?, ?, ?, ?)`
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		log.Fatal(err)
	}

	_, err = statement.Exec(channel, username, message, time.Now())
	if err != nil {
		log.Fatal(err)
	}
}
```
- **`insertMessage()`**: Adds a new message to the `messages` table with the current time.

### 6. Retrieving Old Messages
```go
func getMessages(db *sql.DB, channel string) ([]Message, error) {
	var messages []Message

	query := `SELECT username, message, time FROM messages WHERE channel = ? AND datetime(time) >= datetime('now', '-5 minutes', 'localtime') ORDER BY time ASC`

	rows, err := db.Query(query, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var username, message string
		var timestamp time.Time
		if err := rows.Scan(&username, &message, &timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			Channel:  channel,
			Username: username,
			Content:  message,
			Time:     timestamp,
		})
	}

	return messages, nil
}
```
- **`getMessages()`**: Retrieves messages from the `messages` table for a specific channel within the last 5 minutes.

### 7. HTTP Handlers for Pages
```go
func GetIndexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func GetChannelsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "channels.html")
}

func GetChatPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "chat.html")
}
```
- These functions serve HTML pages to the client when they visit specific routes (e.g., `/` serves `index.html`).

### 8. Fetching Active Channels
```go
func GetActiveChannels(w http.ResponseWriter, r *http.Request) {
	var channels []string
	keys := activeChannels.Keys(false)
	for _, key := range keys {
		channels = append(channels, key.(string))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}
```
- **`GetActiveChannels()`**: Retrieves the names of active channels from the cache and sends them as a JSON response.

### 9. Handling WebSocket Connections
```go
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
```
- **`handleConnections()`**: Manages client connections, disconnections, and message broadcasting:
  - **`register`**: Adds a new client to the `clients` map and updates the channel's user list.
  - **`unregister`**: Removes a client from the `clients` map and updates the channel's user list.
  - **`broadcast`**: Saves the message to the database and sends it to all clients in the same channel.

### 10. Updating Channel User Lists
```go
func updateChannel(client Client, add bool) {
	mutex.Lock()
	defer mutex.Unlock()

	channel, err := activeChannels.Get(client.Channel)
	if err != nil {
		if add {
			newChannel := Channel{Name: client.Channel, Users: []string{client.Username}}
			activeChannels.Set(client.Channel, newChannel)
		}
		return
	}

	ch := channel.(Channel)
	if add {
		ch.Users = append(ch.Users, client.Username)
	} else {
		for i, user := range ch.Users {
			if user == client.Username {
				ch.Users = append(ch.Users[:i], ch.Users[i+1:]...)
				break
			}
		}
		if len(ch.Users) == 0 {
			activeChannels.Remove(client.Channel)
			return
		}
	}
	activeChannels.Set(client.Channel, ch)
}
```
- **`updateChannel()`**: Adds or removes users from a channel's user list in the cache.

### 11. WebSocket Handler
```go
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

		messages, err := getMessages(db, channelName)
		if err != nil {
			log.Printf("error retrieving messages

: %v", err)
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
```
- **`websocketHandler()`**: Handles WebSocket connections, allowing clients to join channels, receive old messages, and broadcast new ones. It registers the client and starts a goroutine to handle incoming messages.

### 12. Starting the Server
```go
func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	db := initDatabase()
	defer db.Close()

	go handleConnections(db)

	r.Get("/", GetIndexPage)
	r.Get("/channels", GetChannelsPage)
	r.Get("/chat", GetChatPage)
	r.Get("/Active/channels", GetActiveChannels)

	r.Get("/ws/{channel}", websocketHandler(db))

	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", r)
}
```
- **`main()`**: The entry point of the program:
  - Initializes the router and database.
  - Starts the `handleConnections()` goroutine to manage WebSocket connections.
  - Defines HTTP routes to serve pages and handle WebSocket connections.
  - Starts the HTTP server on port 8080.

---

This program creates a simple chat application where users can join channels, send and receive messages in real-time, and view old messages sent within the last 5 minutes. The messages are stored in an SQLite database, and the application uses WebSockets for real-time communication.
package channels

import (
	"chat-webapp/messages"
	"chat-webapp/storage"
	"log"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/gorilla/websocket"
)

// ServerInterface abstracts the Server dependency.
type ServerInterface interface {
	GetDB() storage.DB
	GetActiveChannels() gcache.Cache
	GetMutex() *sync.Mutex
}

// Client represents a connected WebSocket client.
type Client struct {
	Conn     *websocket.Conn
	Username string
	Channel  string
}

var (
	Register   = make(chan Client)
	Unregister = make(chan Client)
	Broadcast  = make(chan messages.Message)
)

type Channel struct {
	Name  string
	Users []string
	Timer *time.Timer
}

// HandleConnections handles WebSocket connections and channel management.
func HandleConnections(s ServerInterface) {
	clients := make(map[*websocket.Conn]Client)

	for {
		select {
		case client := <-Register:
			clients[client.Conn] = client
			updateChannel(s, client, true)
			log.Printf("User %s joined channel %s", client.Username, client.Channel)

		case client := <-Unregister:
			if _, ok := clients[client.Conn]; ok {
				client.Conn.Close()
				delete(clients, client.Conn)
				updateChannel(s, client, false)
				log.Printf("User %s left channel %s", client.Username, client.Channel)
			}

		case message := <-Broadcast:
			if err := messages.InsertMessage(s.GetDB(), message); err != nil {
				log.Printf("Error inserting message: %v", err)
			}

			for conn, client := range clients {
				if client.Channel == message.Channel {
					err := conn.WriteJSON(message)
					if err != nil {
						log.Printf("Write error: %v", err)
						conn.Close()
						delete(clients, conn)
					}
				}
			}
		}
	}
}

func updateChannel(s ServerInterface, client Client, add bool) {
	s.GetMutex().Lock()
	defer s.GetMutex().Unlock()

	channel, err := s.GetActiveChannels().Get(client.Channel)
	if err != nil && add {
		// Channel does not exist; add it
		newChannel := Channel{Name: client.Channel, Users: []string{client.Username}}
		s.GetActiveChannels().Set(client.Channel, newChannel)
		return
	}

	// Update existing channel
	if ch, ok := channel.(Channel); ok {
		if add {
			ch.Users = append(ch.Users, client.Username)
		} else {
			// Remove user and handle empty channel
			for i, user := range ch.Users {
				if user == client.Username {
					ch.Users = append(ch.Users[:i], ch.Users[i+1:]...)
					break
				}
			}
			if len(ch.Users) == 0 {
				s.GetActiveChannels().Remove(client.Channel)
			}
		}
		s.GetActiveChannels().Set(client.Channel, ch)
	}
}

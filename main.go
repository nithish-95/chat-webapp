package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/bluele/gcache"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/olahol/melody.v1"
)

type Channel struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
}

var (
	activeChannels = gcache.New(20).Simple().Build()
	mutex          sync.Mutex
)

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
	mutex.Lock()
	defer mutex.Unlock()

	var channels []string
	keys := activeChannels.Keys(false)
	for _, key := range keys {
		channels = append(channels, key.(string))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

func main() {
	r := chi.NewRouter()
	m := melody.New()

	r.Use(middleware.Logger)

	// Routes
	r.Get("/", GetIndexPage)
	r.Get("/channels", GetChannelsPage)
	r.Get("/chat", GetChatPage)
	r.Get("/Active/channels", GetActiveChannels)

	// Handle WebSocket connections
	r.Get("/ws/{channel}", func(w http.ResponseWriter, r *http.Request) {
		channelName := chi.URLParam(r, "channel")
		m.HandleRequestWithKeys(w, r, map[string]interface{}{"channel": channelName})
	})

	// Handle messages
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		channelName := s.Keys["channel"].(string)
		m.BroadcastFilter(msg, func(q *melody.Session) bool {
			return q.Keys["channel"] == channelName
		})
	})

	m.HandleConnect(func(s *melody.Session) {
		channelName := s.Keys["channel"].(string)
		userName := s.Request.URL.Query().Get("UserName")

		mutex.Lock()
		defer mutex.Unlock()

		channel, err := activeChannels.Get(channelName)
		if err != nil {
			newChannel := Channel{Name: channelName, Users: []string{userName}}
			activeChannels.Set(channelName, newChannel)
		} else {
			ch := channel.(Channel)
			ch.Users = append(ch.Users, userName)
			activeChannels.Set(channelName, ch)
		}
	})

	m.HandleDisconnect(func(s *melody.Session) {
		channelName := s.Keys["channel"].(string)
		userName := s.Request.URL.Query().Get("UserName")

		mutex.Lock()
		defer mutex.Unlock()

		channel, err := activeChannels.Get(channelName)
		if err == nil {
			ch := channel.(Channel)
			for i, user := range ch.Users {
				if user == userName {
					ch.Users = append(ch.Users[:i], ch.Users[i+1:]...)
					break
				}
			}
			if len(ch.Users) == 0 {
				activeChannels.Remove(channelName)
			} else {
				activeChannels.Set(channelName, ch)
			}
		}
	})

	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", r)
}

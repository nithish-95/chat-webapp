package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olahol/melody"
)

var (
	channels = make(map[string]*melody.Melody)
	mu       sync.Mutex
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

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Routes
	r.Get("/", GetIndexPage)
	r.Get("/channels", GetChannelsPage)
	r.Get("/chat", GetChatPage)

	// WebSocket route
	r.Get("/ws/{channel}", func(w http.ResponseWriter, r *http.Request) {
		channel := chi.URLParam(r, "channel")

		mu.Lock()
		m, exists := channels[channel]
		if !exists {
			m = melody.New()
			channels[channel] = m

			// Handle incoming WebSocket messages for this channel
			m.HandleMessage(func(s *melody.Session, msg []byte) {
				m.Broadcast(msg)
			})
		}
		mu.Unlock()

		m.HandleRequest(w, r)
	})

	fmt.Println("Starting server on :3000")
	http.ListenAndServe(":3000", r)
}

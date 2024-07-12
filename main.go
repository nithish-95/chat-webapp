package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/olahol/melody"
)

func parseFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func parseChatFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "chat.html")
}

func main() {
	r := chi.NewRouter()
	m := melody.New()

	r.Use(middleware.Logger)

	r.Get("/", parseFile)
	r.Get("/chat", parseChatFile)

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})

	http.ListenAndServe(":3000", r)
}

package ui

import (
	"github.com/go-chi/chi/v5"
	"github.com/nithish-95/chat-webapp/service"
)

type UIEndpoint struct {
	c *Controller
}

// Path implements service.Endpoint.
func (u *UIEndpoint) Path() string {
	return "ui"
}

// Routes implements service.Endpoint.
func (u *UIEndpoint) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", u.c.GetIndexPage)
	r.Get("/channels", u.c.GetChannelsPage)
	r.Get("/chat", u.c.GetChatPage)
	return r
}

func NewUIEndpoint(c Controller) (service.Endpoint, error) {
	return &UIEndpoint{}, nil
}

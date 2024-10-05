package ui

import "net/http"

type Controller struct {
}

func (c *Controller) GetIndexPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (c *Controller) GetChannelsPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "channels.html")
}

func (c *Controller) GetChatPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "chat.html")
}

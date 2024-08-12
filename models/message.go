package models

import "time"

type Message struct {
	Channel  string    `json:"channel"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	Time     time.Time `json:"time"`
}

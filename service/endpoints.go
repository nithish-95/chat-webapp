package service

import "github.com/go-chi/chi/v5"

type Endpoint interface {
	Routes() chi.Router
	Path() string
}

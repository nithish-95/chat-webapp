package handlers

import (
	"context"
	"net/http"
)

type Handler interface {
	GetActiveChannels(ctx context.Context, w http.ResponseWriter, r *http.Request)
	WebSocketHandler(ctx context.Context) http.HandlerFunc
}

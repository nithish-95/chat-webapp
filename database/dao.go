package database

import (
	"context"
	"time"

	"github.com/nithish-95/chat-webapp/models"
)

type Dao interface {
	InitDatabase(ctx context.Context) error
	InsertMessage(ctx context.Context, channel, username, message string) error
	GetMessages(ctx context.Context, channel string) ([]models.Message, error)
	Close(ctx context.Context) error
	DeleteMessages(ctx context.Context, cutoff time.Time) error
}

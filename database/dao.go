package database

import (
	"context"
)

type Dao interface {
	InitDatabase()
	InsertMessage(ctx context.Context, channel, username, message string)
	DeleteChat(ctx context.Context, channel string)
}

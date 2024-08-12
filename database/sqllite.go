package database

import (
	"context"
	"database/sql"
)

type sqliteDao struct {
	db *sql.DB
}

// DeleteChat implements Dao.
func (s *sqliteDao) DeleteChat(ctx context.Context, channel string) {
	panic("unimplemented")
}

// InitDatabase implements Dao.
func (s *sqliteDao) InitDatabase() {
	panic("unimplemented")
}

// InsertMessage implements Dao.
func (s *sqliteDao) InsertMessage(ctx context.Context, channel string, username string, message string) {
	panic("unimplemented")
}

func NewSQLlite(filename string) (Dao, error) {
	db, err := sql.Open("sqlite3", "./chat_messages.db")
	if err != nil {
		return nil, err
	}
	return &sqliteDao{
		db: db,
	}, nil

}

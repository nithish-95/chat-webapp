package database

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/nithish-95/chat-webapp/models"
)

type sqliteDao struct {
	db *sql.DB
}

// DeleteChat implements Dao.
func (s *sqliteDao) DeleteChat(ctx context.Context, channel string) {
	panic("unimplemented")
}

// InitDatabase implements Dao.
func (s *sqliteDao) InitDatabase(ctx context.Context) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		username TEXT NOT NULL,
		message TEXT NOT NULL,
		time DATETIME DEFAULT CURRENT_TIME
	);`

	statement, err := s.db.Prepare(createTableSQL)
	if err != nil {
		return err
	}
	statement.Exec()
	return nil
}

// InsertMessage implements Dao.
func (s *sqliteDao) InsertMessage(ctx context.Context, channel string, username string, message string) error {
	log.Printf("Inserting message: channel=%s, username=%s, message=%s", channel, username, message)
	insertSQL := `INSERT INTO messages(channel, username, message, time) VALUES (?, ?, ?, ?)`
	statement, err := s.db.Prepare(insertSQL)
	if err != nil {
		return err
	}

	_, err = statement.Exec(channel, username, message, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func (s *sqliteDao) GetMessages(ctx context.Context, channel string) ([]models.Message, error) {
	messages := make([]models.Message, 0)

	query := `SELECT username, message, time FROM messages WHERE channel = ? AND datetime(time) >= datetime('now', '-5 minutes', 'localtime') ORDER BY time ASC`

	rows, err := s.db.Query(query, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var username, message string
		var timestamp time.Time
		if err := rows.Scan(&username, &message, &timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, models.Message{
			Channel:  channel,
			Username: username,
			Content:  message,
			Time:     timestamp,
		})
	}

	return messages, nil
}

func (s *sqliteDao) DeleteMessages(ctx context.Context, cutoff time.Time) error {
	log.Printf("deleting messages older than %v", cutoff)

	transaction, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	// Delete messages for channels that are not in the activeChannelsSet
	deleteSQL := `DELETE FROM messages WHERE time < ?`
	_, err = transaction.Exec(deleteSQL, cutoff)
	if err != nil {
		log.Fatal(err)
	}

	if err = transaction.Commit(); err != nil {
		return nil
	}
	return nil
}

func (s *sqliteDao) Close(ctx context.Context) error {
	return s.db.Close()
}

func NewSQLlite(filename string) (Dao, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	return &sqliteDao{
		db: db,
	}, nil
}

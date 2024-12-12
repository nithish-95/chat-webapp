package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB interface defines methods for interacting with the database.
type DB interface {
	Init() error
	InsertMessage(channel, username, message string, msgTime time.Time) error
	GetMessages(channel string, timeLimit time.Duration) ([]Message, error)
	ClearClosedChannels(activeChannels map[string]bool) error
	Close() error
}

// Message struct represents a chat message.
type Message struct {
	Channel  string    `json:"channel"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	Time     time.Time `json:"time"`
}

// SQLiteDB is a wrapper around an SQLite database connection.
type SQLiteDB struct {
	*sql.DB
}

// NewSQLiteDB initializes a new SQLiteDB instance.
func NewSQLiteDB(filepath string) DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Printf("Error opening SQLite database: %v", err)
		return nil
	}

	sqlitedb := &SQLiteDB{db}
	if err := sqlitedb.Init(); err != nil {
		log.Printf("Error initializing SQLite database: %v", err)
		return nil
	}

	return sqlitedb
}

// Init sets up the required tables in the SQLite database.
func (db *SQLiteDB) Init() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel TEXT NOT NULL,
		username TEXT NOT NULL,
		message TEXT NOT NULL,
		time DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(createTableSQL)
	return err
}

// InsertMessage adds a new message to the database.
func (db *SQLiteDB) InsertMessage(channel, username, message string, msgTime time.Time) error {
	insertSQL := `INSERT INTO messages (channel, username, message, time) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(insertSQL, channel, username, message, msgTime)
	if err != nil {
		return fmt.Errorf("failed to insert message: %v", err)
	}
	return nil
}

// GetMessages retrieves messages from a specific channel within a time limit.
func (db *SQLiteDB) GetMessages(channel string, timeLimit time.Duration) ([]Message, error) {
	query := `SELECT username, message, time FROM messages WHERE channel = ? AND time >= ? ORDER BY time ASC`
	rows, err := db.Query(query, channel, time.Now().Add(-timeLimit))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %v", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Username, &msg.Content, &msg.Time); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %v", err)
		}
		msg.Channel = channel
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	return messages, nil
}

// ClearClosedChannels removes messages for channels that are no longer active.
func (db *SQLiteDB) ClearClosedChannels(activeChannels map[string]bool) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // Ensure rollback on error

	query := `SELECT DISTINCT channel FROM messages`
	rows, err := tx.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query channels: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var channel string
		if err := rows.Scan(&channel); err != nil {
			return fmt.Errorf("failed to scan channel: %v", err)
		}

		if _, active := activeChannels[channel]; !active {
			_, err := tx.Exec("DELETE FROM messages WHERE channel = ?", channel)
			if err != nil {
				return fmt.Errorf("failed to delete messages for channel %s: %v", channel, err)
			}
			log.Printf("Cleared messages for inactive channel: %s", channel)
		}
	}

	return tx.Commit()
}

// Close closes the SQLite database connection.
func (db *SQLiteDB) Close() error {
	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("failed to close database: %v", err)
	}
	return nil
}

// StartDatabaseCleanup periodically clears messages for inactive channels.
func StartDatabaseCleanup(db DB, interval time.Duration, getActiveChannels func() map[string]bool) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			activeChannels := getActiveChannels()
			if err := db.ClearClosedChannels(activeChannels); err != nil {
				log.Printf("Error clearing closed channels: %v", err)
			}
		}
	}()
}

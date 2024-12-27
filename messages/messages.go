package messages

import (
	"log"
	"time"

	"github.com/nithish-95/chat-webapp/storage"
)

// Message struct represents a chat message.
type Message struct {
	Channel  string    `json:"channel"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	Time     time.Time `json:"time"`
}

const messageRetrievalDuration = 5 * time.Minute // Duration for message retrieval

// Helper function to convert storage messages to package-specific messages.
func convertStorageMessages(storageMessages []storage.Message) []Message {
	var messages []Message
	for _, sm := range storageMessages {
		messages = append(messages, Message{
			Channel:  sm.Channel,
			Username: sm.Username,
			Content:  sm.Content,
			Time:     sm.Time,
		})
	}
	return messages
}

// InsertMessage inserts a message into the database.
func InsertMessage(db storage.DB, msg Message) error {
	err := db.InsertMessage(msg.Channel, msg.Username, msg.Content, time.Now())
	if err != nil {
		log.Printf("Error inserting message: %v", err)
		return err
	}
	return nil
}

// GetMessages retrieves messages for a channel within a time limit.
func GetMessages(db storage.DB, channel string) ([]Message, error) {
	// Retrieve messages from the database
	storageMessages, err := db.GetMessages(channel, messageRetrievalDuration)
	if err != nil {
		return nil, err
	}

	// Convert storage.Message to messages.Message
	return convertStorageMessages(storageMessages), nil
}

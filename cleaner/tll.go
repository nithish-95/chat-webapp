package cleaner

import (
	"context"
	"time"

	"github.com/nithish-95/chat-webapp/database"
)

type ttl struct {
	dao    database.DAO
	ticker *time.Ticker
	d      time.Duration
}

// StartCleaning implements Cleaner.
func (t *ttl) StartCleaning() error {
	for range t.ticker.C {
		t.dao.DeleteMessages(context.Background(), time.Now().Add(-t.d))
	}
	return nil
}

// NewTTL creates a new instance of ttl with a given DAO and duration.
func NewTTL(dao database.DAO, d time.Duration) Cleaner {
	t := time.NewTicker(d)
	return &ttl{dao: dao, ticker: t, d: d}
}

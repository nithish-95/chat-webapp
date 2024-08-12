package cleaner

import (
	"context"
	"time"

	"github.com/nithish-95/chat-webapp/database"
)

type ttl struct {
	dao    database.Dao
	ticker *time.Ticker
	d      time.Duration
}

// Clear the Database for closed channels
// StartCleaning implements Cleaner.
func (t *ttl) StartCleaning() error {
	for range t.ticker.C {
		t.dao.DeleteMessages(context.Background(), time.Now().Add(-t.d))
	}
	return nil
}

func NewTTL(dao database.Dao, d time.Duration) Cleaner {
	t := time.NewTicker(d)
	return &ttl{dao: dao, ticker: t}
}

package postgres

import (
	"sync"
	"time"

	"github.com/jackc/pgx"
	"gopkg.in/inconshreveable/log15.v2"
)

// Listen creates a listener for the given channel, returning the listener
// and the first connection error (nil on successful connection).
func (db *DB) Listen(channel string, log log15.Logger) (*Listener, error) {
	conn, err := db.Acquire()
	if err != nil {
		return nil, err
	}
	l := &Listener{
		Notify:  make(chan *pgx.Notification),
		channel: channel,
		log:     log,
		db:      db,
		conn:    conn,
	}
	if err := l.conn.Listen(channel); err != nil {
		return nil, err
	}
	go l.listen()
	return l, nil
}

type Listener struct {
	Notify chan *pgx.Notification
	Err    error

	channel   string
	closeOnce sync.Once
	log       log15.Logger
	db        *DB
	conn      *pgx.Conn
}

func (l *Listener) Close() (err error) {
	l.closeOnce.Do(func() {
		l.conn.Exec("UNLISTEN " + l.channel)
		l.db.Release(l.conn)
	})
	return
}

func (l *Listener) listen() {
	for {
		n, err := l.conn.WaitForNotification(10 * time.Second)
		if err == pgx.ErrNotificationTimeout {
			continue
		}
		if err != nil {
			l.Err = err
			close(l.Notify)
			return
		}
		l.Notify <- n
	}
}

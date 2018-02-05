package nats

import (
	"io"

	"github.com/nats-io/go-nats-streaming"
)

// TODO -
func New(conn stan.Conn) *NATS {
	return &NATS{
		conn: conn,
	}
}

type NATS struct {
	conn stan.Conn
}

func (n *NATS) SubscribeQueue(subj string, f func(uint64, []byte)) (io.Closer, error) {
	return n.conn.QueueSubscribe(
		subj,
		"ingest",
		func(m *stan.Msg) {
			f(m.Sequence, m.Data)
		},
	)
}

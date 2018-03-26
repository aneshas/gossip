package nats

import (
	"io"
	"time"

	"github.com/nats-io/go-nats-streaming"
)

// TODO - don't pass in conn
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

func (n *NATS) SubscribeSeq(id string, nick string, start uint64, f func(uint64, []byte)) (io.Closer, error) {
	return n.conn.Subscribe(
		id,
		func(m *stan.Msg) {
			f(m.Sequence, m.Data)
		},
		stan.StartAtSequence(start),
	)
}

func (n *NATS) SubscribeTimestamp(id string, nick string, t time.Time, f func(uint64, []byte)) (io.Closer, error) {
	return n.conn.Subscribe(
		id,
		func(m *stan.Msg) {
			f(m.Sequence, m.Data)
		},
		stan.StartAtTime(t),
	)
}

func (n *NATS) Send(id string, msg []byte) error {
	return n.conn.Publish(id, msg)
}

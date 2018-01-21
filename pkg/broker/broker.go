// Package broker provides chat broker functionality
package broker

import (
	"log"
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/tonto/gossip/pkg/chat"
)

// New creates new chat broker instance
func New(conn stan.Conn) *Broker {
	return &Broker{
		nats: conn,
	}
}

// Broker represents chat broker
type Broker struct {
	nats stan.Conn
}

// Subscribe subscribes to provided chat id at start sequence
// Returns close subscription func, or an error.
func (b *Broker) Subscribe(id chat.ID, s chat.UserID, start uint64, c chan *Msg) (func(), error) {
	sub, err := b.nats.Subscribe(
		string(id),
		func(m *stan.Msg) {
			log.Println("nats - got message from: ", id)
			msg, err := DecodeMsg(m.Data)
			if err != nil {
				log.Fatalf("broker: error decoding message: %v", err)
				return
			}

			msg.Seq = m.Sequence

			if msg.From != s {
				c <- msg
			}
		},
		stan.StartAtSequence(start),
	)

	if err != nil {
		return nil, err
	}

	return func() { sub.Close() }, nil
}

// SubscribeNew subscribes to provided chat id subject starting from time.Now()
// Returns close subscription func, or an error.
func (b *Broker) SubscribeNew(id chat.ID, s chat.UserID, c chan *Msg) (func(), error) {
	sub, err := b.nats.Subscribe(
		string(id),
		func(m *stan.Msg) {
			msg, err := DecodeMsg(m.Data)
			if err != nil {
				log.Fatalf("broker: error decoding message: %v", err)
				return
			}

			msg.Seq = m.Sequence

			if msg.From != s {
				c <- msg
			}
		},
		stan.StartAtTime(time.Now()),
	)

	if err != nil {
		return nil, err
	}

	return func() { sub.Close() }, nil
}

// Send sends new message to a given chat
func (b *Broker) Send(id chat.ID, msg *Msg) error {
	data, err := EncodeMsg(msg)
	if err != nil {
		return err
	}

	return b.nats.Publish(string(id), data)
}

// Package broker provides chat broker functionality
package broker

import (
	"log"
	"time"

	"github.com/nats-io/go-nats-streaming"
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
func (b *Broker) Subscribe(id string, nick string, start uint64, c chan *Msg) (func(), error) {
	sub, err := b.nats.Subscribe(
		"chat."+id,
		func(m *stan.Msg) {
			msg := decodeMsg(m.Data)
			msg.Seq = m.Sequence

			if msg.From != nick {
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
func (b *Broker) SubscribeNew(id string, nick string, c chan *Msg) (func(), error) {
	sub, err := b.nats.Subscribe(
		"chat."+id,
		func(m *stan.Msg) {
			msg := decodeMsg(m.Data)
			msg.Seq = m.Sequence

			if msg.From != nick {
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
func (b *Broker) Send(id string, msg *Msg) error {
	data, err := EncodeMsg(msg)
	if err != nil {
		return err
	}

	return b.nats.Publish("chat."+id, data)
}

func decodeMsg(b []byte) *Msg {
	msg, err := DecodeMsg(b)
	if err != nil {
		log.Printf("broker: error decoding message: %v", err)
		msg.Text = "message unavailable."
	}
	return msg
}

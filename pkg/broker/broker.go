// Package broker provides chat broker functionality
package broker

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/nats-io/go-nats-streaming"
)

// New creates new chat broker instance
func New(conn stan.Conn, ig Ingester) *Broker {
	return &Broker{
		nats: conn,
		ig:   ig,
	}
}

// Broker represents chat broker
type Broker struct {
	nats stan.Conn
	ig   Ingester
}

// Ingester represents chat history read model ingester
type Ingester interface {
	Run(string) (func(), error)
}

// MQ represents message broker interface
type MQ interface {
	SubscribeSeq(string, string, uint64, func(uint64, []byte)) (io.Closer, error)
	SubscribeTimestamp(string, string, time.Time, func(uint64, []byte)) (io.Closer, error)
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

	close, err := b.ig.Run(id)
	if err != nil {
		sub.Close()
		return nil, fmt.Errorf("broker: unable to run ingest for chat. try again")
	}

	return func() { sub.Close(); close() }, nil
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

	close, err := b.ig.Run(id)
	if err != nil {
		sub.Close()
		return nil, fmt.Errorf("broker: unable to run ingest for chat. try again")
	}

	return func() { sub.Close(); close() }, nil
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

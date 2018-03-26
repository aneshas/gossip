// Package broker provides chat broker functionality
package broker

import (
	"fmt"
	"io"
	"time"
)

// New creates new chat broker instance
func New(mq MQ, store ChatStore, ig Ingester) *Broker {
	return &Broker{
		mq:    mq,
		ig:    ig,
		store: store,
	}
}

// Broker represents chat broker
type Broker struct {
	mq    MQ
	ig    Ingester
	store ChatStore
}

// MQ represents message broker interface
type MQ interface {
	Send(string, []byte) error
	SubscribeSeq(string, string, uint64, func(uint64, []byte)) (io.Closer, error)
	SubscribeTimestamp(string, string, time.Time, func(uint64, []byte)) (io.Closer, error)
}

// Ingester represents chat history read model ingester
type Ingester interface {
	Run(string) (func(), error)
}

// ChatStore represents chat store interface
type ChatStore interface {
	UpdateLastClientSeq(string, string, uint64)
}

// Subscribe subscribes to provided chat id at start sequence
// Returns close subscription func, or an error.
func (b *Broker) Subscribe(id string, nick string, start uint64, c chan *Msg) (func(), error) {
	closer, err := b.mq.SubscribeSeq("chat."+id, nick, start, func(seq uint64, data []byte) {
		msg, err := DecodeMsg(data)
		if err != nil {
			msg = &Msg{
				From: "broker",
				Text: "broker: message unavailable: decoding error",
				Time: time.Now(),
			}
		}

		msg.Seq = seq

		if msg.From != nick {
			c <- msg
		} else {
			b.store.UpdateLastClientSeq(msg.From, id, seq)
		}
	})

	if err != nil {
		return nil, err
	}

	cleanup, err := b.ig.Run(id)
	if err != nil {
		closer.Close()
		return nil, fmt.Errorf("broker: unable to run ingest for chat. try again")
	}

	return func() { closer.Close(); cleanup() }, nil
}

// SubscribeNew subscribes to provided chat id subject starting from time.Now()
// Returns close subscription func, or an error.
func (b *Broker) SubscribeNew(id string, nick string, c chan *Msg) (func(), error) {
	closer, err := b.mq.SubscribeTimestamp("chat."+id, nick, time.Now(), func(seq uint64, data []byte) {
		msg, err := DecodeMsg(data)
		if err != nil {
			msg = &Msg{
				From: "broker",
				Text: "broker: message unavailable: decoding error",
				Time: time.Now(),
			}
		}

		msg.Seq = seq

		if msg.From != nick {
			c <- msg
		}
	})

	if err != nil {
		return nil, err
	}

	cleanup, err := b.ig.Run(id)
	if err != nil {
		closer.Close()
		return nil, fmt.Errorf("broker: unable to run ingest for chat. try again")
	}

	return func() { closer.Close(); cleanup() }, nil
}

// Send sends new message to a given chat
func (b *Broker) Send(id string, msg *Msg) error {
	data, err := EncodeMsg(msg)
	if err != nil {
		return err
	}

	return b.mq.Send("chat."+id, data)
}

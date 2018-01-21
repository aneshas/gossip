// Package ingest provides functionality for
// updating per chat read models (recent history)
package ingest

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/go-nats-streaming"
	"github.com/tonto/gossip/pkg/broker"
	"github.com/tonto/gossip/pkg/chat"
)

func New(nats stan.Conn, store ChatStore) *Ingest {
	return &Ingest{
		nats:  nats,
		store: store,
	}
}

// Ingest represents chat ingester
type Ingest struct {
	nats  stan.Conn
	store ChatStore
}

// ChatStore represents chat store interface
type ChatStore interface {
	AppendMessage(chat.ID, *broker.Msg) error
}

// Run runs the ingestor
// Make sure to cancel the context when done
func (i *Ingest) Run(c context.Context) error {
	_, err := i.nats.QueueSubscribe(
		"general",
		"ingest",
		func(m *stan.Msg) {
			msg, err := broker.DecodeMsg(m.Data)
			if err != nil {
				log.Printf("ingest: error decoding message: %v", err)
				return
			}

			msg.Seq = m.Sequence

			i.store.AppendMessage(chat.ID(m.Subject), msg)
		},
	)

	if err != nil {
		return fmt.Errorf("ingest: could not subscribe: %v", err)
	}

	return nil
}

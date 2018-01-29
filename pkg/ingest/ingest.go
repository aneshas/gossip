// Package ingest provides functionality for
// updating per chat read models (recent history)
package ingest

import (
	"fmt"
	"log"

	"github.com/nats-io/go-nats-streaming"
	"github.com/tonto/gossip/pkg/broker"
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
	AppendMessage(string, *broker.Msg) error
}

// RunIngest subscribes to ingest queue group and updates chat read model
func (i *Ingest) RunIngest(id string) (func(), error) {
	sub, err := i.nats.QueueSubscribe(
		"chat."+id,
		"ingest",
		func(m *stan.Msg) {
			msg, err := broker.DecodeMsg(m.Data)
			if err != nil {
				log.Printf("ingest: error decoding message: %v", err)
				return
			}

			msg.Seq = m.Sequence

			i.store.AppendMessage(id, msg)
		},
	)

	if err != nil {
		return nil, fmt.Errorf("ingest: could not subscribe: %v", err)
	}

	return func() { sub.Close() }, nil
}

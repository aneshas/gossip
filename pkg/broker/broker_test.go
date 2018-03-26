package broker_test

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/tonto/gossip/pkg/broker"
)

func TestSubscribe(t *testing.T) {
	cases := []struct {
		name    string
		chat    string
		nick    string
		start   uint64
		n       int
		queue   queue
		ingest  ingest
		want    []broker.Msg
		wantErr bool
	}{
		{
			name: "subscribe seq error",
			chat: "general",
			nick: "me",
			queue: queue{
				SubscribeSeqFunc: func(c string, n string, s uint64, f func(uint64, []byte)) (io.Closer, error) {
					return nil, fmt.Errorf("error")
				},
			},
			start:   0,
			wantErr: true,
		},
		{
			name: "ingest run error",
			chat: "general",
			nick: "me",
			queue: queue{
				SubscribeSeqFunc: func(c string, n string, s uint64, f func(uint64, []byte)) (io.Closer, error) {
					return &cl{}, nil
				},
			},
			ingest: ingest{
				RunFunc: func(string) (func(), error) {
					return nil, fmt.Errorf("error")
				},
			},
			start:   0,
			wantErr: true,
		},
		{
			name:  "dont send own messages",
			chat:  "general",
			nick:  "me",
			start: 0,
			n:     3,
			queue: queue{
				SubscribeSeqFunc: func(c string, n string, s uint64, f func(uint64, []byte)) (io.Closer, error) {
					msgs := []broker.Msg{
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
					}

					go func() {
						for i, m := range msgs {
							var buff bytes.Buffer
							gob.NewEncoder(&buff).Encode(m)
							f(uint64(i), buff.Bytes())
						}
					}()

					return &cl{}, nil
				},
			},
			ingest: ingest{
				RunFunc: func(string) (func(), error) { return func() {}, nil },
			},
			want: []broker.Msg{
				{From: "john", Text: "foo msg", Seq: 0},
				{From: "john", Text: "foo msg", Seq: 2},
				{From: "john", Text: "foo msg", Seq: 3},
			},
			wantErr: false,
		},
		{
			name: "decoding error",
			chat: "general",
			nick: "me",
			n:    3,
			queue: queue{
				SubscribeSeqFunc: func(c string, n string, s uint64, f func(uint64, []byte)) (io.Closer, error) {
					msgs := []broker.Msg{
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
					}

					go func() {
						for i, m := range msgs {
							var buff bytes.Buffer
							gob.NewEncoder(&buff).Encode(m)
							if i == 2 {
								f(uint64(i), []byte("xxxx"))
								continue
							}
							f(uint64(i), buff.Bytes())
						}
					}()

					return &cl{}, nil
				},
			},
			ingest: ingest{
				RunFunc: func(string) (func(), error) { return func() {}, nil },
			},
			want: []broker.Msg{
				{From: "john", Text: "foo msg", Seq: 0},
				{From: "broker", Text: "broker: message unavailable: decoding error", Seq: 2},
				{From: "john", Text: "foo msg", Seq: 3},
			},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := broker.New(&tc.queue, store{}, &tc.ingest)

			c := make(chan *broker.Msg)

			close, err := b.Subscribe(tc.chat, tc.nick, tc.start, c)

			if (err != nil) != tc.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if err != nil {
				return
			}

			defer close()

			var msgs []broker.Msg

			for i := 0; i < tc.n; i++ {
				msg := <-c
				msgs = append(msgs, *msg)
			}

			if len(tc.want) != len(msgs) {
				t.Errorf("invalid number of messages. want: %d, got: %d", len(tc.want), len(msgs))
			}

			for _, w := range tc.want {
				found := false
				for _, g := range msgs {
					if w.Text == g.Text && w.Seq == g.Seq {
						found = true
					}
				}
				if !found {
					t.Errorf("unexpected response. want: %v, got: %v", tc.want, msgs)
				}
			}

			if !tc.ingest.RunCalled {
				t.Errorf("ingest run should have been called but was not")
			}
		})
	}
}

func TestSubscribeNew(t *testing.T) {
	cases := []struct {
		name    string
		chat    string
		nick    string
		n       int
		queue   queue
		ingest  ingest
		want    []broker.Msg
		wantErr bool
	}{
		{
			name: "subscribe seq error",
			chat: "general",
			nick: "me",
			queue: queue{
				SubscribeTimestampFunc: func(c string, n string, t time.Time, f func(uint64, []byte)) (io.Closer, error) {
					return nil, fmt.Errorf("error")
				},
			},
			wantErr: true,
		},
		{
			name: "ingest run error",
			chat: "general",
			nick: "me",
			queue: queue{
				SubscribeTimestampFunc: func(c string, n string, t time.Time, f func(uint64, []byte)) (io.Closer, error) {
					return &cl{}, nil
				},
			},
			ingest: ingest{
				RunFunc: func(string) (func(), error) {
					return nil, fmt.Errorf("error")
				},
			},
			wantErr: true,
		},
		{
			name: "dont send own messages",
			chat: "general",
			nick: "me",
			n:    3,
			queue: queue{
				SubscribeTimestampFunc: func(c string, n string, t time.Time, f func(uint64, []byte)) (io.Closer, error) {
					msgs := []broker.Msg{
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
					}

					go func() {
						for i, m := range msgs {
							var buff bytes.Buffer
							gob.NewEncoder(&buff).Encode(m)
							f(uint64(i), buff.Bytes())
						}
					}()

					return &cl{}, nil
				},
			},
			ingest: ingest{
				RunFunc: func(string) (func(), error) { return func() {}, nil },
			},
			want: []broker.Msg{
				{From: "john", Text: "foo msg", Seq: 0},
				{From: "john", Text: "foo msg", Seq: 2},
				{From: "john", Text: "foo msg", Seq: 3},
			},
			wantErr: false,
		},
		{
			name: "decoding error",
			chat: "general",
			nick: "me",
			n:    3,
			queue: queue{
				SubscribeTimestampFunc: func(c string, n string, t time.Time, f func(uint64, []byte)) (io.Closer, error) {
					msgs := []broker.Msg{
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: "john", Text: "foo msg"},
						{From: n, Text: "foo msg"},
					}

					go func() {
						for i, m := range msgs {
							var buff bytes.Buffer
							gob.NewEncoder(&buff).Encode(m)
							if i == 2 {
								f(uint64(i), []byte("xxxx"))
								continue
							}
							f(uint64(i), buff.Bytes())
						}
					}()

					return &cl{}, nil
				},
			},
			ingest: ingest{
				RunFunc: func(string) (func(), error) { return func() {}, nil },
			},
			want: []broker.Msg{
				{From: "john", Text: "foo msg", Seq: 0},
				{From: "broker", Text: "broker: message unavailable: decoding error", Seq: 2},
				{From: "john", Text: "foo msg", Seq: 3},
			},
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := broker.New(&tc.queue, store{}, &tc.ingest)

			c := make(chan *broker.Msg)

			close, err := b.SubscribeNew(tc.chat, tc.nick, c)

			if (err != nil) != tc.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if err != nil {
				return
			}

			defer close()

			var msgs []broker.Msg

			for i := 0; i < tc.n; i++ {
				msg := <-c
				msgs = append(msgs, *msg)
			}

			if len(tc.want) != len(msgs) {
				t.Errorf("invalid number of messages. want: %d got: %d", len(tc.want), len(msgs))
			}

			for _, w := range tc.want {
				found := false
				for _, g := range msgs {
					if w.Text == g.Text && w.Seq == g.Seq {
						found = true
					}
				}
				if !found {
					t.Errorf("unexpected response. want: %v, got: %v", tc.want, msgs)
				}
			}

			if !tc.ingest.RunCalled {
				t.Errorf("ingest run should have been called but was not")
			}
		})
	}
}

func TestSend(t *testing.T) {

}

type queue struct {
	SubscribeSeqFunc       func(string, string, uint64, func(uint64, []byte)) (io.Closer, error)
	SubscribeTimestampFunc func(string, string, time.Time, func(uint64, []byte)) (io.Closer, error)
}

func (q *queue) SubscribeSeq(id string, nick string, start uint64, f func(uint64, []byte)) (io.Closer, error) {
	return q.SubscribeSeqFunc(id, nick, start, f)
}

func (q *queue) SubscribeTimestamp(id string, nick string, t time.Time, f func(uint64, []byte)) (io.Closer, error) {
	return q.SubscribeTimestampFunc(id, nick, t, f)
}

func (q *queue) Send(string, []byte) error {
	panic("not implemented")
}

type cl struct{}

func (c *cl) Close() error { return nil }

type ingest struct {
	RunCalled bool
	RunFunc   func(string) (func(), error)
}

func (i *ingest) Run(s string) (func(), error) {
	i.RunCalled = true
	return i.RunFunc(s)
}

type store struct{}

func (s store) UpdateLastClientSeq(string, string, uint64) {}

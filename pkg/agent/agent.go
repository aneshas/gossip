package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tonto/gossip/pkg/broker"
	"github.com/tonto/gossip/pkg/chat"
)

// New creates new connection agent instance
func New(broker *broker.Broker, store ChatStore) *Agent {
	return &Agent{
		broker: broker,
		store:  store,
		done:   make(chan struct{}, 1),
	}
}

// Agent represents chat connection agent which handles
// end to end comm client - broker
type Agent struct {
	chat          *chat.Chat
	connectedUser string
	conn          *websocket.Conn
	broker        *broker.Broker
	store         ChatStore
	done          chan struct{}
	closeSub      func()
	closed        bool
}

// ChatStore represents chat store interface
type ChatStore interface {
	Get(string) (*chat.Chat, error)
	GetRecent(string, int64) ([]broker.Msg, uint64, error)
}

type msgT int

const (
	chatMsg msgT = iota
	historyMsg
	errorMsg
	infoMsg
	historyReqMsg
)

const (
	maxHistoryCount uint64 = 3 // 150
)

type msg struct {
	Type  msgT        `json:"type"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// HandleConn handles websocket communication for requested chat/client
// TODO - Better goroutine lifecycle management in general
func (a *Agent) HandleConn(conn *websocket.Conn, req *initConReq) {
	a.conn = conn

	a.conn.SetCloseHandler(func(code int, text string) error {
		a.closed = true
		a.done <- struct{}{}
		return nil
	})

	ct, err := a.store.Get(req.Channel)
	if err != nil {
		writeFatal(a.conn, "agent: unable to find chat")
		return
	}

	if ct == nil {
		writeFatal(a.conn, "agent: this chat does not exist")
		return
	}

	user, err := ct.Join(req.Nick, req.Secret)
	if err != nil {
		writeFatal(a.conn, err.Error())
		return
	}

	a.chat = ct
	a.connectedUser = user.Nick

	mc := make(chan *broker.Msg)
	{
		var close func()

		if req.LastSeq != nil {
			close, err = a.broker.Subscribe(req.Channel, user.Nick, *req.LastSeq, mc)
		} else {
			if seq, err := a.pushRecent(); err != nil {
				writeErr(a.conn, "agent: unable to fetch chat history. try reconnecting")
				close, err = a.broker.SubscribeNew(req.Channel, user.Nick, mc)
			} else {
				close, err = a.broker.Subscribe(req.Channel, user.Nick, seq, mc)
			}
		}

		if err != nil {
			writeFatal(a.conn, "agent: unable to subscribe to chat updates. closing connection")
			return
		}

		a.closeSub = close
	}

	a.loop(mc)
}

func (a *Agent) pushRecent() (uint64, error) {
	msgs, seq, err := a.store.GetRecent(a.chat.Name, 100)
	if err != nil {
		return 0, err
	}

	if msgs == nil {
		return 0, nil
	}

	return seq, a.conn.WriteJSON(msg{
		Type: historyMsg,
		Data: msgs,
	})
}

func (a *Agent) loop(mc chan *broker.Msg) {
	go func() {
		for {
			if a.closed {
				return
			}

			_, r, err := a.conn.NextReader()
			if err != nil {
				writeErr(a.conn, err.Error())
				continue
			}

			a.handleClientMsg(r)
		}
	}()

	go func() {
		defer a.closeSub()
		defer a.conn.Close()
		for {
			select {
			case m := <-mc:
				a.conn.WriteJSON(msg{
					Type: chatMsg,
					Data: m,
				})

			case <-a.done:
				return
			}
		}
	}()
}

func (a *Agent) handleClientMsg(r io.Reader) {
	var message struct {
		Type msgT            `json:"type"`
		Data json.RawMessage `json:"data,omitempty"`
	}

	err := json.NewDecoder(r).Decode(&message)
	if err != nil {
		writeErr(a.conn, fmt.Sprintf("invalid message format: %v", err))
		return
	}

	switch message.Type {
	case chatMsg:
		a.handleChatMsg(message.Data)
	case historyReqMsg:
		a.handleHistoryReqMsg(message.Data)
	}
}

func (a *Agent) handleChatMsg(raw json.RawMessage) {
	var msg broker.Msg

	err := json.Unmarshal(raw, &msg)
	if err != nil {
		writeErr(a.conn, fmt.Sprintf("invalid text message format: %v", err))
		return
	}

	if msg.Text == "" {
		writeErr(a.conn, "sent empty message")
		return
	}

	if len(msg.Text) > 1024 {
		writeErr(a.conn, "exceeded max message length of 1024 characters")
		return
	}

	msg.From = a.connectedUser
	msg.Time = time.Now()

	err = a.broker.Send(a.chat.Name, &msg)
	if err != nil {
		writeErr(a.conn, fmt.Sprintf("could not forward your message. try again: %v", err))
	}
}

func (a *Agent) handleHistoryReqMsg(raw json.RawMessage) {
	var req struct {
		To uint64 `json:"to"`
	}

	err := json.Unmarshal(raw, &req)
	if err != nil {
		writeErr(a.conn, fmt.Sprintf("invalid history request message format: %v", err))
		return
	}

	if req.To <= 0 {
		return
	}

	msgs, err := a.buildHistoryBatch(req.To)
	if err != nil {
		writeErr(a.conn, "could not fetch chat history")
		return
	}

	log.Println("go history msgs: ", msgs)

	a.conn.WriteJSON(msg{
		Type: historyMsg,
		Data: msgs,
	})
}

func (a *Agent) buildHistoryBatch(to uint64) ([]*broker.Msg, error) {
	var offset uint64

	// TODO - Are Seqs sequential per subject???
	if to >= maxHistoryCount {
		offset = to - maxHistoryCount
	}

	mc := make(chan *broker.Msg)

	close, err := a.broker.Subscribe(a.chat.Name, "", offset, mc)
	if err != nil {
		return nil, err
	}

	defer close()

	var msgs []*broker.Msg

	for {
		msg := <-mc
		if msg.Seq >= to {
			break
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

func writeErr(conn *websocket.Conn, err string) {
	conn.WriteJSON(msg{Error: err, Type: errorMsg})
}

func writeFatal(conn *websocket.Conn, err string) {
	conn.WriteJSON(msg{Error: err, Type: errorMsg})
	conn.Close()
}

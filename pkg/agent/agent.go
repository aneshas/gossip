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

// TODO - Agent handles creation of private chats - adds members (leave privates last)

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

	ct, err := a.store.Get(req.Channel)
	if err != nil {
		a.writeFatal("agent: unable to find chat")
		return
	}

	if ct == nil {
		a.writeFatal("agent: this chat does not exist")
		return
	}

	user, err := ct.Join(req.Nick, req.Secret)
	if err != nil {
		a.writeFatal(err.Error())
		return
	}

	a.chat = ct
	a.connectedUser = user.Nick

	mc := make(chan *broker.Msg)
	{
		var close func()

		if seq, err := a.pushRecent(); err != nil {
			a.writeErr("agent: unable to fetch chat history. try reconnecting")
			close, err = a.broker.SubscribeNew(req.Channel, user.Nick, mc)
		} else {
			close, err = a.broker.Subscribe(req.Channel, user.Nick, seq, mc)
		}

		if err != nil {
			a.writeFatal("agent: unable to subscribe to chat updates. closing connection")
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
			t, r, err := a.conn.NextReader()
			if err != nil {
				a.writeErr(err.Error())
				continue
			}

			if t == websocket.CloseMessage {
				a.done <- struct{}{}
				return
			}

			a.handleClientMsg(r)
		}
	}()

	go func() {
		defer a.conn.Close()
		defer a.closeSub()
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
		a.writeErr(fmt.Sprintf("invalid message format: %v", err))
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
		a.writeErr(fmt.Sprintf("invalid text message format: %v", err))
		return
	}

	msg.From = a.connectedUser
	msg.Time = time.Now()

	err = a.broker.Send(a.chat.Name, &msg)
	if err != nil {
		a.writeErr(fmt.Sprintf("could not forward your message. try again: %v", err))
	}
}

func (a *Agent) handleHistoryReqMsg(raw json.RawMessage) {
	var req struct {
		To uint64 `json:"to"`
	}

	err := json.Unmarshal(raw, &req)
	if err != nil {
		a.writeErr(fmt.Sprintf("invalid history request message format: %v", err))
		return
	}

	if req.To <= 0 {
		return
	}

	msgs, err := a.buildHistoryBatch(req.To)
	if err != nil {
		a.writeErr("could not fetch chat history")
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

func (a *Agent) writeErr(err string) {
	a.conn.WriteJSON(msg{Error: err, Type: errorMsg})
}

func (a *Agent) writeFatal(err string) {
	a.conn.WriteJSON(msg{Error: err, Type: errorMsg})
	a.conn.Close()
}

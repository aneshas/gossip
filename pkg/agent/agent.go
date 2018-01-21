package agent

import (
	"encoding/json"
	"fmt"
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
	connectedUser chat.UserID
	conn          *websocket.Conn
	broker        *broker.Broker
	store         ChatStore
	done          chan struct{}
	closeSub      func()
}

// ChatStore represents chat store interface
type ChatStore interface {
	Get(chat.ID) (*chat.Chat, error)
	GetUser(string) (*chat.User, error)
	GetRecent(chat.ID) ([]broker.Msg, uint64, error)
}

// HandleConn handles websocket communication for requested chat/client
// TODO - Better lifecycle management in general
func (a *Agent) HandleConn(conn *websocket.Conn, req *initConReq) {
	a.conn = conn

	// TODO - Fetch chat from redis and join chat
	ct, err := a.store.Get(req.ChatID)
	if err != nil {
		a.writeFatal("agent: unable to find chat")
		return
	}

	if ct == nil {
		a.writeFatal("agent: this chat does not exist")
		return
	}

	_, err = a.store.GetUser(string(req.ClientID))
	if err != nil {
		a.writeFatal(fmt.Sprintf("agent: unable to join chat. nick not registered"))
		return
	}

	a.chat = ct

	err = a.chat.Join(req.ClientID)
	if err != nil {
		a.writeFatal(fmt.Sprintf("agent: unable to join chat: %v", err))
		return
	}

	a.connectedUser = req.ClientID

	mc := make(chan *broker.Msg)
	{
		var close func()

		if seq, err := a.pushRecent(); err != nil {
			a.writeErr("agent: unable to fetch chat history. try reconnecting")
			close, err = a.broker.SubscribeNew(req.ChatID, req.ClientID, mc)
		} else {
			close, err = a.broker.Subscribe(req.ChatID, req.ClientID, seq, mc)
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
	msgs, seq, err := a.store.GetRecent(a.chat.ID)
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
			m, err := a.fetchNext()
			if err != nil {
				if err == errConnClosed {
					return
				}
				a.writeErr(err.Error())
				continue
			}

			err = a.broker.Send(a.chat.ID, m)
			if err != nil {
				a.writeErr(fmt.Sprintf("could not forward your message. try again: %v", err))
			}
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

func (a *Agent) fetchNext() (*broker.Msg, error) {
	t, r, err := a.conn.NextReader()
	if err != nil || t == websocket.CloseMessage {
		a.done <- struct{}{}
		return nil, errConnClosed
	}

	var message struct {
		Type msgT       `json:"type"`
		Data broker.Msg `json:"data,omitempty"`
	}

	err = json.NewDecoder(r).Decode(&message)
	if err != nil {
		return nil, fmt.Errorf("could not decode message: %v", err)
	}

	message.Data.From = a.connectedUser
	message.Data.Time = time.Now()

	return &message.Data, nil
}

func (a *Agent) writeErr(err string) {
	a.conn.WriteJSON(msg{Error: err, Type: errorMsg})
}

func (a *Agent) writeFatal(err string) {
	a.conn.WriteJSON(msg{Error: err, Type: errorMsg})
	a.conn.Close()
}

type msgT int

const (
	chatMsg msgT = iota
	historyMsg
	errorMsg
)

type msg struct {
	Type  msgT        `json:"type"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

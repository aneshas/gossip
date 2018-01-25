package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/tonto/gossip/pkg/broker"

	"github.com/gorilla/websocket"
	h "github.com/tonto/kit/http"
	"github.com/tonto/kit/http/respond"
)

// NewAPI creates new websocket api
func NewAPI(broker *broker.Broker, store ChatStore) *API {
	api := API{
		broker: broker,
		store:  store,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}

	api.RegisterHandler("GET", "/connect", api.connect)

	return &api
}

// API represents websocket api service
type API struct {
	h.BaseService
	broker   *broker.Broker
	store    ChatStore
	upgrader websocket.Upgrader
}

// Prefix returns api prefix for this service
func (api *API) Prefix() string { return "agent" }

func (api *API) connect(c context.Context, w http.ResponseWriter, r *http.Request) {
	conn, err := api.upgrader.Upgrade(w, r, nil)
	if err != nil {
		respond.WithJSON(w, r, fmt.Errorf("ws api: unable to upgrade to ws connection: %v", err))
		return
	}

	req, err := api.waitConnInit(conn)
	if err != nil {
		if err == errConnClosed {
			return
		}
		conn.WriteJSON(msg{Error: err.Error()})
		return
	}

	agent := New(api.broker, api.store)
	agent.HandleConn(conn, req)
}

type initConReq struct {
	Channel string `json:"channel"`
	Nick    string `json:"nick"`
	Secret  string `json:"secret"` // User secret
}

func (ir *initConReq) Validate() error {
	// TODO - Validate lengthe alphanumeric etc...
	if ir.Channel == "" || ir.Secret == "" || ir.Nick == "" {
		return fmt.Errorf("join fail: channel_id, nick and secret are required")
	}
	return nil
}

var errConnClosed = errors.New("connection closed")

func (api *API) waitConnInit(conn *websocket.Conn) (*initConReq, error) {
	t, wsr, err := conn.NextReader()
	if err != nil || t == websocket.CloseMessage {
		return nil, errConnClosed
	}

	var req initConReq

	err = json.NewDecoder(wsr).Decode(&req)
	if err != nil {
		return nil, err
	}

	err = req.Validate()
	if err != nil {
		return nil, err
	}

	return &req, nil
}

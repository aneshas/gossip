package chat

import (
	"context"
	"fmt"
	"net/http"

	h "github.com/tonto/kit/http"
)

// NewAPI creates new websocket api
func NewAPI(store Store, admin, password string) *API {
	api := API{
		store: store,
	}

	api.RegisterEndpoint(
		"POST",
		"/create_channel",
		api.createChannel,
		WithHTTPBasicAuth(admin, password),
	)

	api.RegisterEndpoint("POST", "/register_nick", api.registerNick)

	return &api
}

// API represents websocket api service
type API struct {
	h.BaseService
	store Store
}

// Store represents chat store interface
type Store interface {
	Save(*Chat) error
	Get(string) (*Chat, error)
}

// Prefix returns api prefix for this service
func (api *API) Prefix() string { return "chat" }

type createChanReq struct {
	Name string `json:"name"`
}

type createChanResp struct {
	Secret string `json:"secret"`
}

func (cr *createChanReq) Validate() error {
	if cr.Name == "" {
		// TODO - validate: no spaces, alphanumeric, length etc... etc...
		return fmt.Errorf("name must not be empty")
	}
	return nil
}

// TODO - Should be accessible only by admin
func (api *API) createChannel(c context.Context, w http.ResponseWriter, req *createChanReq) (*h.Response, error) {
	ch := NewChannel(req.Name)
	if err := api.store.Save(ch); err != nil {
		return nil, fmt.Errorf("could not create channel at this moment")
	}
	return h.NewResponse(createChanResp{Secret: ch.Secret}, http.StatusOK), nil
}

type registerNickReq struct {
	Nick          string `json:"nick"`
	FullName      string `json:"name"`
	Email         string `json:"email"`
	Channel       string `json:"channel"`
	ChannelSecret string `json:"channel_secret"`
}

type registerNickResp struct {
	Secret string `json:"secret"`
}

func (r *registerNickReq) Validate() error {
	// TODO - validate: no spaces, alphanumeric, length etc... etc...
	if r.Nick == "" {
		return fmt.Errorf("nick is required")
	}
	if r.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	if r.ChannelSecret == "" {
		return fmt.Errorf("channel_secret is required")
	}
	return nil
}

func (api *API) registerNick(c context.Context, w http.ResponseWriter, req *registerNickReq) (*h.Response, error) {
	ch, err := api.store.Get(req.Channel)
	if err != nil {
		return nil, fmt.Errorf("could not fetch channel")
	}

	if ch.Secret != req.ChannelSecret {
		return nil, fmt.Errorf("invalid secret")
	}

	secret, err := ch.Register(&User{
		Nick:     req.Nick,
		FullName: req.FullName,
		Email:    req.Email,
	})

	if err != nil {
		return nil, err
	}

	// TODO - Need transaction
	err = api.store.Save(ch)
	if err != nil {
		return nil, fmt.Errorf("could not update channel membership")
	}

	return h.NewResponse(registerNickResp{Secret: secret}, http.StatusOK), nil
}

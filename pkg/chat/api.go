package chat

import (
	"context"
	"fmt"
	"net/http"

	h "github.com/tonto/kit/http"
)

// NewAPI creates new websocket api
func NewAPI(store Store) *API {
	api := API{
		store: store,
	}

	api.RegisterEndpoint("POST", "/create_channel", api.createChannel)
	api.RegisterEndpoint("POST", "/register_user", api.registerUser)

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
	SaveUser(*User) error
}

// Prefix returns api prefix for this service
func (api *API) Prefix() string { return "chat" }

// apis:
// - create_private

type createChanReq struct {
	Name string `json:"name"`
}

func (cr *createChanReq) Validate() error {
	if cr.Name == "" {
		// TODO - validate: no spaces etc...
		return fmt.Errorf("name must not be empty")
	}
	return nil
}

func (api *API) createChannel(c context.Context, w http.ResponseWriter, req *createChanReq) error {
	if err := api.store.Save(NewChan(req.Name)); err != nil {
		return fmt.Errorf("could not create channel at this moment")
	}
	return nil
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

func (r *registerUserReq) Validate() error {
	if r.Nick == "" {
		return fmt.Errorf("nick is required")
	}
	return nil
}

func (api *API) registerUser(c context.Context, w http.ResponseWriter, req *registerUserReq) error {
	if err := api.store.SaveUser(
		&User{
			Nick: req.Nick,
			Name: req.Name,
		},
	); err != nil {
		return fmt.Errorf("could not create nick at this moment")
	}
	return nil
}

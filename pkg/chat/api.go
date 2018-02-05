package chat

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	h "github.com/tonto/kit/http"
	"github.com/tonto/kit/http/respond"
)

// NewAPI creates new websocket api
func NewAPI(store Store, admin, password string) *API {
	api := API{
		store: store,
	}

	api.RegisterEndpoint(
		"POST",
		"/admin/create_channel",
		api.createChannel,
		WithHTTPBasicAuth(admin, password),
	)

	api.RegisterHandler("GET", "/list_channels", api.listChannels)
	api.RegisterEndpoint("POST", "/register_nick", api.registerNick)
	api.RegisterEndpoint("POST", "/channel_members", api.channelMembers)

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
	ListChannels() ([]string, error)
}

// Prefix returns api prefix for this service
func (api *API) Prefix() string { return "chat" }

type createChanReq struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
}

type createChanResp struct {
	Secret string `json:"secret"`
}

func (cr *createChanReq) Validate() error {
	if cr.Name == "" {
		return fmt.Errorf("name must not be empty")
	}
	if len(cr.Name) < 3 || len(cr.Name) > 10 {
		return fmt.Errorf("name must be between 3 and 10 characters long")
	}
	if match, err := regexp.Match("^[a-zA-Z0-9_]*$", []byte(cr.Name)); !match || err != nil {
		return fmt.Errorf("name must only contain alphanumeric and underscores")
	}
	return nil
}

func (api *API) createChannel(c context.Context, w http.ResponseWriter, req *createChanReq) (*h.Response, error) {
	ch := NewChannel(req.Name, req.Private)
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
	ChannelSecret string `json:"channel_secret"` // Tennant
}

type registerNickResp struct {
	Secret string `json:"secret"`
}

func (r *registerNickReq) Validate() error {
	if r.Nick == "" {
		return fmt.Errorf("nick is required")
	}
	if r.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	if len(r.Nick) < 3 || len(r.Nick) > 10 {
		return fmt.Errorf("nick must be between 3 and 10 characters long")
	}
	if match, err := regexp.Match("^[a-zA-Z0-9_]*$", []byte(r.Nick)); !match || err != nil {
		return fmt.Errorf("nick must only contain alphanumeric and underscores")
	}
	if len(r.FullName) > 20 ||
		len(r.Email) > 20 ||
		len(r.ChannelSecret) > 20 {
		return fmt.Errorf("exceeded max field length of 20")
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

type channelMembersReq struct {
	Channel       string `json:"channel"`
	ChannelSecret string `json:"channel_secret"`
}

func (r *channelMembersReq) Validate() error {
	if r.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	if len(r.Channel) > 10 {
		return fmt.Errorf("channel name must not exceed 10 characters")
	}
	if len(r.ChannelSecret) > 20 {
		return fmt.Errorf("channel_secret must not exceed 20 characters")
	}
	return nil
}

func (api *API) channelMembers(c context.Context, w http.ResponseWriter, req *channelMembersReq) (*h.Response, error) {
	ch, err := api.store.Get(req.Channel)
	if err != nil {
		return nil, fmt.Errorf("could not fetch channel")
	}

	members := []User{}

	if ch.Members != nil && len(ch.Members) > 0 {
		for _, u := range ch.Members {
			members = append(members, u)
		}
	}

	return h.NewResponse(members, http.StatusOK), nil
}

func (api *API) listChannels(c context.Context, w http.ResponseWriter, r *http.Request) {
	chans, err := api.store.ListChannels()
	if err != nil {
		respond.WithJSON(
			w, r,
			h.NewError(http.StatusInternalServerError, err),
		)
		return
	}
	respond.WithJSON(
		w, r,
		chans,
	)
}

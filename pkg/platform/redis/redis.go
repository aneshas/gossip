package redis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis"
	"github.com/tonto/gossip/pkg/broker"
	"github.com/tonto/gossip/pkg/chat"
)

func NewStore(host string) (*Store, error) {
	opts := redis.Options{
		Addr: host + ":6379",
	}

	client := redis.NewClient(&opts)

	_, err := client.Ping().Result()
	if err != nil {
		return nil, err
	}

	return &Store{
		client: client,
	}, nil
}

type Store struct {
	client *redis.Client
}

func (s *Store) Get(id chat.ID) (*chat.Chat, error) {
	val, err := s.client.Get(string(id)).Result()
	if err != nil {
		return nil, err
	}

	var ct chat.Chat

	err = json.Unmarshal([]byte(val), &ct)
	if err != nil {
		return nil, fmt.Errorf("store: unable to unmarshal chat. invalid format: %v", err)
	}

	return &ct, nil
}

func (s *Store) GetRecent(id chat.ID) ([]broker.Msg, uint64, error) {
	cmd := s.client.LRange(fmt.Sprintf("%s_history", id), 0, 100)
	if cmd.Err() != nil {
		return nil, 0, cmd.Err()
	}

	data, err := cmd.Result()
	if err != nil {
		return nil, 0, err
	}

	if data == nil || len(data) == 0 {
		return nil, 0, nil
	}

	var seq uint64
	msgs := make([]broker.Msg, len(data))

	for i, m := range data {
		var msg broker.Msg
		err = json.NewDecoder(strings.NewReader(m)).Decode(&msgs[i])
		if err != nil {
			msg.Text = "message unavailable!"
		} else {
			seq = msgs[i].Seq
		}
	}

	return msgs, seq, nil
}

func (s *Store) AppendMessage(id chat.ID, m *broker.Msg) error {
	data, err := json.Marshal(m)
	if err != nil {
		data = []byte(`{"text":"message unavailable, unable to encode","from":"gossip/store"}`)
	}

	cmd := s.client.RPush(fmt.Sprintf("%s_history", id), data)

	return cmd.Err()
}

func (s *Store) Save(ct *chat.Chat) error {
	data, err := json.Marshal(ct)
	if err != nil {
		return err
	}

	cmd := s.client.Set(string(ct.ID), data, 0)

	return cmd.Err()
}

func (s *Store) SaveUser(u *chat.User) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}

	cmd := s.client.Set(string(u.Nick), data, 0)

	return cmd.Err()
}

func (s *Store) GetUser(nick string) (*chat.User, error) {
	val, err := s.client.Get(nick).Result()
	if err != nil {
		return nil, err
	}

	var u chat.User

	err = json.Unmarshal([]byte(val), &u)
	if err != nil {
		return nil, fmt.Errorf("store: unable to unmarshal user. invalid format: %v", err)
	}

	return &u, nil
}

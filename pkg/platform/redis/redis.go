package redis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-redis/redis"
	"github.com/tonto/gossip/pkg/broker"
	"github.com/tonto/gossip/pkg/chat"
)

const (
	maxHistorySize int64 = 10
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

func (s *Store) Get(id string) (*chat.Chat, error) {
	val, err := s.client.Get(id).Result()
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

func (s *Store) GetRecent(id string, n int64) ([]broker.Msg, uint64, error) {
	cmd := s.client.LRange(fmt.Sprintf("history.chat.%s", id), -n, -1)
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

	return msgs, (seq + 1), nil
}

func (s *Store) AppendMessage(id string, m *broker.Msg) error {
	data, err := json.Marshal(m)
	if err != nil {
		data = []byte(`{"text":"message unavailable, unable to encode","from":"gossip/store"}`)
	}

	key := fmt.Sprintf("history.%s", id)

	if err := s.client.RPush(key, data).Err(); err != nil {
		return err
	}

	return s.client.LTrim(key, -maxHistorySize, -1).Err()
}

func (s *Store) Save(ct *chat.Chat) error {
	data, err := json.Marshal(ct)
	if err != nil {
		return err
	}

	cmd := s.client.Set(ct.Name, data, 0)

	return cmd.Err()
}

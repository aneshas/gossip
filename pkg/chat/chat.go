package chat

import (
	"fmt"

	"github.com/segmentio/ksuid"
)

// NewChannel creates new channel chat
func NewChannel(name string, private bool) *Chat {
	ch := Chat{
		Name:    name,
		Members: make(map[string]User),
	}

	if private {
		ch.Secret = newSecret()
	}

	return &ch
}

// Chat represents private or channel chat
type Chat struct {
	Name    string          `json:"name"`
	Secret  string          `json:"secret"`
	Members map[string]User `json:"members"`
}

// TODO
// Private chats work the same - can only init private chat with people in the same channel

// Register registers user with a chat and returns secret which should
// be stored on the client side, and used for subsequent join requests
func (c *Chat) Register(u *User, secret string) (string, error) {
	if _, ok := c.Members[u.Nick]; ok {
		return "", fmt.Errorf("chat: this nick is already taken")
	}
	u.Secret = secret
	if secret == "" {
		u.Secret = newSecret()
	}
	c.Members[u.Nick] = *u
	return u.Secret, nil
}

// Join attempts to join user to chat
func (c *Chat) Join(nick, secret string) (*User, error) {
	u, ok := c.Members[nick]
	if !ok {
		return nil, fmt.Errorf("chat: nick not registered")
	}
	if u.Secret != secret {
		return nil, fmt.Errorf("chat: invalid secret")
	}
	u.Secret = ""
	return &u, nil
}

func newSecret() string {
	return ksuid.New().String()
}

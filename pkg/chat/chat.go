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

// Join attempts to join user to chat
func (c *Chat) Join(nick, secret string) (*User, error) {
	user, ok := c.Members[secret]
	if !ok {
		return nil, fmt.Errorf("chat: invalid secret")
	}

	if user.Nick != nick {
		return nil, fmt.Errorf("chat: secret and nick do not match")
	}

	return &user, nil
}

// Register registeres user to a chat and returns secret
// to be used for subsequent join request for channel
func (c *Chat) Register(u *User) (string, error) {
	for i := range c.Members {
		if c.Members[i].Nick == u.Nick {
			return "", fmt.Errorf("chat: this nick is already taken")
		}
	}
	secret := newSecret()
	c.Members[secret] = *u
	return secret, nil
}

func newSecret() string {
	return ksuid.New().String()
}

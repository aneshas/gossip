package chat

import (
	"fmt"
)

// ID represents chat identity
type ID string

// Type represents chat type
type Type int

const (
	// ChanChat marks chat as channel type (eg. room)
	ChanChat Type = iota

	// PvtChat marks chat as private chat
	PvtChat
)

// UserID represents chat user id
type UserID string

// NewPvt creates new private chat
func NewPvt(from, to UserID) *Chat {
	return &Chat{
		ID:      ID(fmt.Sprintf("%s-%s", from, to)),
		Type:    PvtChat,
		Members: []UserID{from, to},
	}
}

// NewChan creates new channel
func NewChan(name string) *Chat {
	return &Chat{
		ID:   ID(name),
		Type: ChanChat,
	}
}

// Chat represents private or channel chat
type Chat struct {
	ID      ID              `json:"id"`
	Type    Type            `json:"type"`
	Secret  string          `json:"secret"`
	Members map[string]User `json:"members"`
}

// TODO
// channels have members (secret:Member)
// Join checks if secret is correct and joins ...
// Private chats work the same - can only init prvate chat with people in the same channel

// Join attempts to join user to chat
func (c *Chat) Join(id UserID) error {
	switch c.Type {
	case ChanChat:
		for i := range c.BanList {
			if id == c.BanList[i] {
				return fmt.Errorf("chat: unable to join, you have been banned from this channel")
			}
		}
		return nil
	case PvtChat:
		for i := range c.Members {
			if c.Members[i] == id {
				return nil
			}
		}
		return fmt.Errorf("chat: you are not a member of this chat")
	}
	return nil
}

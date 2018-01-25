package broker

import (
	"bytes"
	"encoding/gob"
	"time"
)

// Msg represents chat message
type Msg struct {
	Meta map[string]string `json:"meta"`
	Time time.Time         `json:"time"`
	Seq  uint64            `json:"seq"`
	Text string            `json:"text"`
	From string            `json:"from"`
}

// DecodeMsg tries to decode gob in b to Msg
func DecodeMsg(b []byte) (*Msg, error) {
	var msg Msg
	r := bytes.NewReader(b)
	err := gob.NewDecoder(r).Decode(&msg)
	return &msg, err
}

// EncodeMsg gob encodes provided chat Msg
func EncodeMsg(msg *Msg) ([]byte, error) {
	var buff bytes.Buffer
	err := gob.NewEncoder(&buff).Encode(msg)
	return buff.Bytes(), err
}

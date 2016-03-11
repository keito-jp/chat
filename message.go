package main

import (
	"encoding/json"
	"log"
	"time"
)

// messageは1つのメッセージを表します。
type message struct {
	Name      string
	Message   string
	When      time.Time
	AvatarURL string
}

func (m *message) encodeJson() []byte {
	b, err := json.Marshal(m)
	if err != nil {
		log.Fatal("JSON encodeに失敗", err)
		return nil
	}
	return b
}

func decodeJson(b []byte) *message {
	var m message
	err := json.Unmarshal(b, &m)
	if err != nil {
		log.Fatal("JSON decodeに失敗", err)
		return nil
	}
	return &m
}

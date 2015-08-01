package utils

import "encoding/json"

type Message struct {
	From    string `json:"from,omitempty"`
	Room    uint   `json:"to,omitempty"`
	Content string `json:"content,omitempty"`
}

type Command struct {
	Name    string `json:"name,omitempty"`
	Room    uint   `json:"room,omitempty"`
	Content string `json:"content,omitempty"`
}

type ManyCommand struct {
	Name    string          `json:"name,omitempty"`
	Room    uint            `json:"room,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
}

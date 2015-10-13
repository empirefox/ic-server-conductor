package conn

import (
	"bytes"
	"encoding/json"
)

// copy from client one
type Ipcam struct {
	Id     string `json:",omitempty"`
	Off    bool   `json:",omitempty"`
	Online bool   `json:",omitempty"`
}

type Ipcams map[string]Ipcam

type ServerCommand struct {
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
}

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

func (c *ManyCommand) Value() []byte {
	return bytes.Trim(c.Content, `"`)
}

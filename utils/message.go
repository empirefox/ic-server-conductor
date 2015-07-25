package utils

import (
	"encoding/json"
)

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

// Message Pool
var messagePool = make(chan *Message, 128)

func EmptyMessage() (msg *Message) {
	select {
	case msg = <-messagePool:
		// Got one from the pool.
		msg.From = ""
		msg.Room = 0
		msg.Content = ""
	default:
		msg = new(Message)
	}
	return msg
}

func (m *Message) Free() {
	select {
	case messagePool <- m:
		// Stored message in the pool
	default:
		// Pool is full. It's a job for the GC.
	}
}

// Command Pool
var commandPool = make(chan *Command, 128)

func EmptyCommand() (c *Command) {
	select {
	case c = <-commandPool:
		c.Name = ""
		c.Room = 0
		c.Content = ""
	default:
		c = new(Command)
	}
	return c
}

func (c *Command) Free() {
	select {
	case commandPool <- c:
		// Stored message in the pool
	default:
		// Pool is full. It's a job for the GC.
	}
}

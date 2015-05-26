package utils

type Message struct {
	From    string `json:"from,omitempty"`
	Room    uint   `json:"to,omitempty"`
	Content string `json:"content,omitempty"`
}

type Command struct {
	Name       string `json:"name,omitempty"`
	Room       uint   `json:"room,omitempty"`
	RoomName   string `json:"room_name,omitempty"`
	Camera     string `json:"camera,omitempty"`
	CameraName string `json:"camera_name,omitempty"`
	CameraOff  bool   `json:"camera_off,omitempty"`
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
		c.RoomName = ""
		c.Camera = ""
		c.CameraName = ""
		c.CameraOff = false
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

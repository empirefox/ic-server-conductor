package connections

import (
	"io"

	"github.com/gorilla/websocket"
)

// copy from gorilla/websocket
type fakeWsConn struct {
	MsgType    int
	RecieveStr []byte
	OnWrite    func(int, []byte) error
	used       bool
}

func (c *fakeWsConn) ReadMessage() (messageType int, p []byte, err error) {
	if c.used {
		return websocket.TextMessage, nil, io.EOF
	}
	c.used = true
	return websocket.TextMessage, []byte(c.RecieveStr), nil
}

func (c *fakeWsConn) WriteMessage(messageType int, data []byte) error {
	return c.OnWrite(messageType, data)
}

func (c *fakeWsConn) Close() error {
	return nil
}

func newFakeWsConn(msgType int, msg string) Connection {
	return &fakeWsConn{MsgType: msgType, RecieveStr: []byte(msg)}
}

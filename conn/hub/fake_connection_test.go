package hub

import "github.com/gorilla/websocket"

type fakeConn struct {
	id               uint
	dataSent         []byte
	messageType      int
	dataForRead      []byte
	dataWrote        []byte
	closeCalledTimes int
}

func (c *fakeConn) Id() uint        { return c.id }
func (c *fakeConn) Send(msg []byte) { c.dataSent = msg }
func (c *fakeConn) Close() error    { c.closeCalledTimes++; return nil }

func (c *fakeConn) ReadMessage() (messageType int, p []byte, err error) {
	if c.messageType == 0 {
		c.messageType = websocket.TextMessage
	}
	return c.messageType, c.dataForRead, nil
}

func (c *fakeConn) WriteMessage(messageType int, data []byte) error {
	c.dataWrote = data
	return nil
}

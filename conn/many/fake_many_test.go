package many

import (
	"github.com/empirefox/ic-server-ws-signal/account"
	"github.com/gorilla/websocket"
)

func newFakeDbOauth() *account.Oauth {
	o := &account.Oauth{}
	return o
}

type fakeMany struct {
	*controlUser
	dataSent         []byte
	messageType      int
	dataForRead      []byte
	dataWrote        []byte
	closeCalledTimes int
}

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

package many

import (
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/gorilla/websocket"
)

func newFakeDbOauth(id uint) *account.Oauth {
	o := &account.Oauth{}
	o.Account.ID = id
	return o
}

func newFakeIdMany(id uint) *fakeMany {
	many := &fakeMany{}
	many.Oauth = newFakeDbOauth(id)
	return many
}

type fakeMany struct {
	*controlUser
	dataSent         [][]byte
	messageType      int
	dataForRead      []byte
	dataWrote        []byte
	closeCalledTimes int
}

func (c *fakeMany) Send(msg []byte) { c.dataSent = append(c.dataSent, msg) }
func (c *fakeMany) Close() error    { c.closeCalledTimes++; return nil }

func (c *fakeMany) ReadMessage() (messageType int, p []byte, err error) {
	if c.messageType == 0 {
		c.messageType = websocket.TextMessage
	}
	return c.messageType, c.dataForRead, nil
}

func (c *fakeMany) WriteMessage(messageType int, data []byte) error {
	c.dataWrote = data
	return nil
}

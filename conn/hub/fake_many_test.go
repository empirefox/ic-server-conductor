package hub

import "github.com/empirefox/ic-server-conductor/account"

func newFakeDbOauth() *account.Oauth {
	o := &account.Oauth{}
	return o
}

type fakeMany struct {
	fakeConn
	ipcamsSentTimes int
	ones            []account.One
	oauth           *account.Oauth
}

func (many *fakeMany) SendIpcams()                      { many.ipcamsSentTimes++ }
func (many *fakeMany) RoomOnes() ([]account.One, error) { return many.ones, nil }
func (many *fakeMany) GetOauth() *account.Oauth         { return many.oauth }

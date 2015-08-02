// from hub
package many

import (
	"github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/conn"
)

func newFakeFriend(id uint) account.Account {
	return account.Account{
		BaseModel: account.BaseModel{
			ID: id,
		},
	}
}

func newFakeDbOne(id uint) *account.One {
	one := &account.One{}
	one.ID = id
	return one
}

type fakeRoom struct {
	fakeConn
	dataBroadcasted []byte
	ipcams          Ipcams
	friends         []account.Account
	onlines         map[uint]ControlUser
	one             *account.One
}

func (room *fakeRoom) Broadcast(msg []byte)                { room.dataBroadcasted = msg }
func (room *fakeRoom) Ipcams() Ipcams                      { return room.ipcams }
func (room *fakeRoom) Friends() ([]account.Account, error) { return room.friends, nil }
func (room *fakeRoom) GetOne() *account.One                { return room.one }

func (room *fakeRoom) AddOnline(id uint, cu ControlUser) {
	room.onlines[id] = cu
}

func (room *fakeRoom) GetOnline(id uint) (cu ControlUser, ok bool) {
	cu, ok = room.onlines[id]
	return
}

func (room *fakeRoom) RemoveOnline(id uint) {
	if room.onlines != nil {
		delete(room.onlines, id)
	}
}

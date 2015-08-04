package hub

import (
	"testing"

	"github.com/empirefox/ic-server-conductor/account"
	. "github.com/empirefox/ic-server-conductor/conn"
	. "github.com/smartystreets/goconvey/convey"
)

func newFakeClient(id uint) ControlUser {
	return &fakeMany{fakeConn: fakeConn{id: id}}
}

func Test__reg(t *testing.T) {
	Convey("onReg/onUnreg should work", t, func() {
		h := NewHub().(*hub)

		// user 601/602 already online
		h.clients = map[uint]ControlUser{
			601: newFakeClient(601),
			602: newFakeClient(602),
		}

		// this room only know 601/603
		room := &fakeRoom{
			fakeConn: fakeConn{id: 101},
			friends: []account.Account{
				newFakeFriend(601),
				newFakeFriend(603),
			},
			onlines: make(map[uint]ControlUser),
			one:     newFakeDbOne(101),
		}

		// reg
		h.onReg(room)
		// room reg ok
		So(h.rooms[101], ShouldNotBeNil)
		// online friends added ok
		So(len(room.onlines), ShouldEqual, 1)
		So(room.onlines[601], ShouldNotBeNil)

		// unreg
		h.onUnreg(room)
		// room unreg ok
		So(len(h.rooms), ShouldEqual, 0)
	})
}

func Test__join_leave(t *testing.T) {
	Convey("onJoin/onLeave should work", t, func() {
		h := NewHub().(*hub)

		// this room only know 601/603
		room := &fakeRoom{
			fakeConn: fakeConn{id: 101},
			friends: []account.Account{
				newFakeFriend(601),
				newFakeFriend(603),
			},
			onlines: make(map[uint]ControlUser),
			one:     newFakeDbOne(101),
		}
		h.rooms[101] = room

		many := &fakeMany{
			fakeConn: fakeConn{id: 601},
			ones:     []account.One{*newFakeDbOne(101), *newFakeDbOne(102)},
			oauth:    newFakeDbOauth(),
		}

		// 601 come online
		h.onJoin(many)

		// 101 should online
		So(h.clients[601], ShouldNotBeNil)
		// 601 should know
		So(len(room.onlines), ShouldEqual, 1)
		So(room.onlines[601], ShouldNotBeNil)

		// 601 gone offline
		h.onLeave(many)

		// 101 should offline
		So(len(h.clients), ShouldEqual, 0)
		// 601 should know
		So(len(room.onlines), ShouldEqual, 0)
	})
}

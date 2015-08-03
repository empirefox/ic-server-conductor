package many

import (
	"encoding/json"
	"testing"

	. "github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/conn"
	. "github.com/smartystreets/goconvey/convey"
)

func findRoomById(rs []CameraRoom, id uint) *CameraRoom {
	for _, room := range rs {
		if room.Id == id {
			return &room
		}
	}
	return nil
}

func Test_genCameraList(t *testing.T) {
	Convey("genCameraList", t, func() {
		// user 601 has room 101/102/103
		as := &fakeService{
			dataGetOnes: []One{*newFakeDbOne(101), *newFakeDbOne(102), *newFakeDbOne(103)},
		}
		SetService(as)
		defer SetService(nil)

		// user 601
		many := &fakeMany{
			controlUser: newControlUser(nil, nil),
		}
		many.Oauth = newFakeDbOauth(601)

		// room 101 has ipcams
		room101 := newFakeIdRoom(101)
		room101.ipcams = Ipcams{
			"ic1": Ipcam{Id: "ic1"},
		}

		h := &fakeHub{
			// room 101/102 online
			rooms: map[uint]ControlRoom{
				101: room101,
				102: newFakeIdRoom(102),
			},
			// user 601 online
			clients: map[uint]ControlUser{
				601: many,
			},
		}
		many.hub = h

		r, err := many.genCameraList()
		// should ok
		So(err, ShouldBeNil)

		var ics CameraList
		err = json.Unmarshal(r, &ics)
		So(err, ShouldBeNil)
		So(len(ics.Rooms), ShouldEqual, 3)

		r10x := findRoomById(ics.Rooms, 101)
		So(r10x, ShouldNotBeNil)
		So(len(r10x.Cameras), ShouldEqual, 1)
		So(r10x.Cameras[0].Id, ShouldEqual, "ic1")

		r10x = findRoomById(ics.Rooms, 102)
		So(r10x, ShouldNotBeNil)
		So(len(r10x.Cameras), ShouldEqual, 0)

		r10x = findRoomById(ics.Rooms, 103)
		So(r10x, ShouldNotBeNil)
		So(len(r10x.Cameras), ShouldEqual, 0)
	})
}

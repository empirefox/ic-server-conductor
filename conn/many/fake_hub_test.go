package many

import (
	"encoding/json"

	. "github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/utils"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type fakeHub struct {
	rooms   map[uint]ControlRoom
	clients map[uint]ControlUser
}

func (h *fakeHub) Run() {}
func (h *fakeHub) GetRoom(id uint) (room ControlRoom, ok bool) {
	room, ok = h.rooms[id]
	return
}

/////////////////////////////////////////
// Copy from private method
/////////////////////////////////////////
func (h *fakeHub) OnReg(room ControlRoom) {
	h.rooms[room.Id()] = room
	friends, err := room.Friends()
	if err != nil {
		glog.Infoln(err)
		return
	}
	for _, friend := range friends {
		if many, ok := h.clients[friend.ID]; ok {
			room.AddOnline(friend.ID, many)
		}
	}
}
func (h *fakeHub) OnUnreg(room ControlRoom) {
	if room.GetOne() == nil {
		return
	}
	delete(h.rooms, room.Id())
}
func (h *fakeHub) OnCmd(cmd *Command) {
	room, ok := h.rooms[cmd.Room]
	if !ok {
		glog.Errorln("Room not found in command")
		return
	}
	cmdStr, err := json.Marshal(cmd)
	if err != nil {
		glog.Errorln(err)
		return
	}
	room.Send(cmdStr)
}
func (h *fakeHub) OnMsg(msg *Message) {
	msgStr, err := utils.GetTypedMsg("Chat", msg)
	if err != nil {
		glog.Errorln(err)
		return
	}
	if room, ok := h.rooms[msg.Room]; ok {
		room.Broadcast(msgStr)
	}
}
func (h *fakeHub) OnJoin(many ControlUser) {
	h.clients[many.Id()] = many
	ones, err := many.RoomOnes()
	if err != nil {
		return
	}
	for _, one := range ones {
		if room, ok := h.rooms[one.ID]; ok {
			room.AddOnline(many.Id(), many)
		}
	}
}
func (h *fakeHub) OnLeave(many ControlUser) {
	if many.GetOauth() == nil {
		return
	}
	delete(h.clients, many.Id())
	ones, err := many.RoomOnes()
	if err != nil {
		return
	}
	for _, one := range ones {
		if room, ok := h.rooms[one.ID]; ok {
			room.RemoveOnline(many.Id())
		}
	}
}

func (h *fakeHub) WaitForProcess(reciever string) (chan *websocket.Conn, error)  { return nil, nil }
func (h *fakeHub) ProcessFromWait(reciever string) (chan *websocket.Conn, error) { return nil, nil }
func (h *fakeHub) NewInviteCode(room uint) string                                { return "" }
func (h *fakeHub) ValidateInviteCode(room uint, code string) bool                { return false }

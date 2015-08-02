package hub

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	. "github.com/empirefox/ic-server-ws-signal/conn"
	"github.com/empirefox/ic-server-ws-signal/utils"
)

var (
	RecieverNotFound   = errors.New("Reciever not found")
	RecieverDuplicated = errors.New("Reciever duplicated")
)

type hub struct {
	rooms         map[uint]ControlRoom
	clients       map[uint]ControlUser
	msg           chan *Message
	cmd           chan *Command
	reg           chan ControlRoom
	unreg         chan ControlRoom
	join          chan ControlUser
	leave         chan ControlUser
	sigResWaitMap map[string]chan *websocket.Conn
	sigResMutex   sync.Mutex
	inviteCodes   map[uint]codes
	inviteMutex   sync.Mutex
	tokenSecret   []byte
}

func NewHub() Hub {
	return &hub{
		rooms:         make(map[uint]ControlRoom),
		clients:       make(map[uint]ControlUser),
		msg:           make(chan *Message, 64),
		cmd:           make(chan *Command, 64),
		reg:           make(chan ControlRoom, 64),
		unreg:         make(chan ControlRoom, 64),
		join:          make(chan ControlUser, 64),
		leave:         make(chan ControlUser, 64),
		sigResWaitMap: make(map[string]chan *websocket.Conn),
		sigResMutex:   sync.Mutex{},
		inviteCodes:   make(map[uint]codes),
		inviteMutex:   sync.Mutex{},
		tokenSecret:   []byte(uniuri.New()),
	}
}

func (h *hub) Run() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()
	for {
		select {
		case msg := <-h.msg:
			h.onMsg(msg)
		case cmd := <-h.cmd:
			h.onCmd(cmd)
		case room := <-h.reg:
			h.onReg(room)
		case room := <-h.unreg:
			h.onUnreg(room)
		case many := <-h.join:
			h.onJoin(many)
		case many := <-h.leave:
			h.onLeave(many)
		}
	}
}

func (h *hub) OnReg(room ControlRoom) { h.reg <- room }
func (h *hub) onReg(room ControlRoom) {
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

func (h *hub) OnUnreg(room ControlRoom) { h.unreg <- room }
func (h *hub) onUnreg(room ControlRoom) {
	if room.GetOne() == nil {
		return
	}
	delete(h.rooms, room.Id())
}

func (h *hub) OnMsg(msg *Message) { h.msg <- msg }
func (h *hub) onMsg(msg *Message) {
	msgStr, err := utils.GetTypedMsg("Chat", msg)
	if err != nil {
		glog.Errorln(err)
		return
	}
	if room, ok := h.rooms[msg.Room]; ok {
		room.Broadcast(msgStr)
	}
}

func (h *hub) OnCmd(cmd *Command) { h.cmd <- cmd }
func (h *hub) onCmd(cmd *Command) {
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

func (h *hub) OnJoin(many ControlUser) { h.join <- many }
func (h *hub) onJoin(many ControlUser) {
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

func (h *hub) OnLeave(many ControlUser) { h.leave <- many }
func (h *hub) onLeave(many ControlUser) {
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

func (h *hub) GetRoom(id uint) (room ControlRoom, ok bool) {
	room, ok = h.rooms[id]
	return
}

func (h *hub) WaitForProcess(reciever string) (chan *websocket.Conn, error) {
	h.sigResMutex.Lock()
	defer h.sigResMutex.Unlock()
	if _, ok := h.sigResWaitMap[reciever]; ok {
		return nil, RecieverDuplicated
	}
	resWait := make(chan *websocket.Conn)
	h.sigResWaitMap[reciever] = resWait
	return resWait, nil
}

func (h *hub) ProcessFromWait(reciever string) (chan *websocket.Conn, error) {
	h.sigResMutex.Lock()
	defer h.sigResMutex.Unlock()
	if resWait, ok := h.sigResWaitMap[reciever]; ok {
		delete(h.sigResWaitMap, reciever)
		return resWait, nil
	}
	return nil, RecieverNotFound
}

type codes map[string]chan bool

func (cs codes) genCode() (string, chan bool) {
	code := uniuri.NewLen(8)
	if _, ok := cs[code]; ok {
		return cs.genCode()
	}
	stop := make(chan bool)
	cs[code] = stop
	return code, stop
}

func (h *hub) waitForStop(cs codes, code string, stop chan bool) {
	if err := recover(); err != nil {
		glog.Errorln(err)
	}
	select {
	case <-stop:
	case <-time.After(time.Minute * 10):
	}
	h.inviteMutex.Lock()
	defer h.inviteMutex.Unlock()
	delete(cs, code)
}

func (h *hub) ValidateInviteCode(room uint, code string) bool {
	h.inviteMutex.Lock()
	defer h.inviteMutex.Unlock()
	if cs, ok := h.inviteCodes[room]; ok {
		if stop, has := cs[code]; has {
			stop <- true
			return true
		}
	}
	return false
}

func (h *hub) NewInviteCode(room uint) string {
	h.inviteMutex.Lock()
	defer h.inviteMutex.Unlock()
	cs, ok := h.inviteCodes[room]
	if !ok {
		cs = make(codes)
		h.inviteCodes[room] = cs
	}
	code, stop := cs.genCode()
	go h.waitForStop(cs, code, stop)
	return code
}

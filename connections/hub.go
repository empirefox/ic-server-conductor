package connections

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	. "github.com/empirefox/ic-server-ws-signal/utils"
)

var (
	RecieverNotFound   = errors.New("Reciever not found")
	RecieverDuplicated = errors.New("Reciever duplicated")
)

type ResponseToMany struct {
	Room    *ControlRoom
	To      uint
	Content []byte
}

type Hub struct {
	rooms         map[uint]*ControlRoom
	clients       map[uint]*ManyControlConn
	msg           chan *Message
	cmd           chan *Command
	reg           chan *ControlRoom
	unreg         chan *ControlRoom
	join          chan *ManyControlConn
	leave         chan *ManyControlConn
	sigResWaitMap map[string]chan *websocket.Conn
	sigResMutex   sync.Mutex
	inviteCodes   map[uint]codes
	inviteMutex   sync.Mutex
	tokenSecret   []byte
}

func NewHub() *Hub {
	return &Hub{
		rooms:         make(map[uint]*ControlRoom),
		clients:       make(map[uint]*ManyControlConn),
		msg:           make(chan *Message, 64),
		cmd:           make(chan *Command, 64),
		reg:           make(chan *ControlRoom, 64),
		unreg:         make(chan *ControlRoom, 64),
		join:          make(chan *ManyControlConn, 64),
		leave:         make(chan *ManyControlConn, 64),
		sigResWaitMap: make(map[string]chan *websocket.Conn),
		sigResMutex:   sync.Mutex{},
		inviteCodes:   make(map[uint]codes),
		inviteMutex:   sync.Mutex{},
		tokenSecret:   []byte(uniuri.New()),
	}
}

func (h *Hub) Run() {
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

func (h *Hub) onReg(room *ControlRoom) {
	h.rooms[room.ID] = room
	if err := room.Viewers(); err != nil {
		glog.Infoln("Viewers err:", err)
		return
	}
	for _, user := range room.Accounts {
		if many, ok := h.clients[user.ID]; ok {
			room.Participants[user.ID] = many
		}
	}
}

func (h *Hub) onUnreg(room *ControlRoom) {
	if room.One == nil {
		return
	}
	if room, ok := h.rooms[room.ID]; ok {
		delete(h.rooms, room.ID)
	}
}

func (h *Hub) onMsg(msg *Message) {
	defer msg.Free()
	msgStr, err := GetTypedMsg("Chat", msg)
	if err != nil {
		glog.Errorln(err)
		return
	}
	if room, ok := h.rooms[msg.Room]; ok {
		room.broadcast(msgStr)
	}
}

func (h *Hub) onCmd(cmd *Command) {
	defer cmd.Free()
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
	room.Send <- cmdStr
}

func (h *Hub) onJoin(many *ManyControlConn) {
	h.clients[many.Account.ID] = many
	if err := many.GetOnes(); err != nil {
		return
	}
	for _, one := range many.Account.Ones {
		room, ok := h.rooms[one.ID]
		if !ok {
			glog.Errorln("Room not found in command")
			continue
		}
		room.Participants[many.Account.ID] = many
	}
}

func (h *Hub) onLeave(many *ManyControlConn) {
	if many.Oauth == nil {
		return
	}
	if _, ok := h.clients[many.Account.ID]; ok {
		delete(h.clients, many.Account.ID)
	}
	if err := many.GetOnes(); err != nil {
		return
	}
	for _, one := range many.Account.Ones {
		room, ok := h.rooms[one.ID]
		if !ok {
			glog.Errorln("Room not found in command")
			return
		}
		if _, ok := room.Participants[many.Account.ID]; ok {
			delete(room.Participants, many.Account.ID)
		}
	}
}

func (h *Hub) waitForProcess(reciever string) (chan *websocket.Conn, error) {
	h.sigResMutex.Lock()
	defer h.sigResMutex.Unlock()
	if _, ok := h.sigResWaitMap[reciever]; ok {
		return nil, RecieverDuplicated
	}
	resWait := make(chan *websocket.Conn)
	h.sigResWaitMap[reciever] = resWait
	return resWait, nil
}

func (h *Hub) processFromWait(reciever string) (chan *websocket.Conn, error) {
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

func (h *Hub) waitForStop(cs codes, code string, stop chan bool) {
	if err := recover(); err != nil {
		glog.Errorln(err)
	}
	select {
	case <-stop:
	case <-time.After(time.Minute * 10):
	}
	h.inviteMutex.Lock()
	defer h.inviteMutex.Unlock()
	if _, ok := cs[code]; ok {
		delete(cs, code)
	}
}

func (h *Hub) ValidateInviteCode(room uint, code string) bool {
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

func (h *Hub) NewInviteCode(room uint) string {
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

package main

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type Hub struct {
	rooms         map[int64]*ControlRoom
	msg           chan *Message
	cmd           chan *Command
	reg           chan *ControlRoom
	unreg         chan *ControlRoom
	join          chan *ManyControlConn
	leave         chan *ManyControlConn
	sigResWaitMap map[string]chan *websocket.Conn
}

func New() *Hub {
	return &Hub{
		make(map[int64]*ControlRoom),
		make(chan *Message),
		make(chan *Command),
		make(chan *ControlRoom),
		make(chan *ControlRoom),
		make(chan *ManyControlConn),
		make(chan *ManyControlConn),
		make(map[string]chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
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
	h.rooms[room.Id] = room
}

func (h *Hub) onUnreg(room *ControlRoom) {
	if room, ok := h.rooms[room.Id]; ok {
		delete(h.rooms, room.Id)
	}
}

func (h *Hub) onMsg(msg *Message) {
	if room, ok := h.rooms[msg.Room]; ok {
		room.broadcast(h, msg)
	}
	msg.Free()
}

func (h *Hub) onCmd(cmd *Command) {
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
	room.SendCtrlToOne <- cmdStr
	cmd.Free()
}

func (h *Hub) onJoin(many *ManyControlConn) {
	for _, roomId := range many.rooms {
		room, ok := h.rooms[roomId]
		if !ok {
			glog.Errorln("Room not found in command")
			continue
		}
		room.Participants[many.Id] = many
	}
}

func (h *Hub) onLeave(many *ManyControlConn) {
	for _, roomId := range many.rooms {
		room, ok := h.rooms[roomId]
		if !ok {
			glog.Errorln("Room not found in command")
			return
		}
		if _, ok := room.Participants[many.Id]; ok {
			delete(room.Participants, many.Id)
		}
	}
}

type CreateSignalingConnectionCommand struct {
	Name     string `json:"name"`
	Reciever string `json:"reciever"`
	Camera   string `json:"camera"`
}

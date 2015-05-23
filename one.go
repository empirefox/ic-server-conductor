package main

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	GinKeyOne = "one"
)

// copy from client one
type Ipcam struct {
	Id     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Off    bool   `json:"off,omitempty"`
	Online bool   `json:"online,omitempty"`
}

type Ipcams map[string]Ipcam

type ControlRoom struct {
	*websocket.Conn
	*One
	Cameras      Ipcams
	Participants map[uint]*ManyControlConn
	Send         chan []byte
	Hub          *Hub
}

func NewControlRoom(h *Hub, ws *websocket.Conn) *ControlRoom {
	return &ControlRoom{
		Conn:         ws,
		Hub:          h,
		Cameras:      make(Ipcams),
		Send:         make(chan []byte, 64),
		Participants: make(map[uint]*ManyControlConn),
	}
}

func (room *ControlRoom) Broadcast(msg *Message) {
	msgStr, err := GetTypedMsg("ChatMsg", msg)
	if err != nil {
		glog.Errorln(err)
		return
	}
	for _, ctrl := range room.Participants {
		select {
		case ctrl.Send <- msgStr:
		default:
			room.Close()
			room.Hub.onLeave(ctrl)
		}
	}
}

// no ping
func (room *ControlRoom) WritePump() {
	defer func() {
		room.Close()
	}()
	for {
		select {
		case msg, ok := <-room.Send:
			if !ok {
				room.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := room.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
			glog.Infoln("ws send ", string(msg))
		}
	}
}

func (room *ControlRoom) ReadPump() {
	for {
		_, b, err := room.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		glog.Infoln("From one client:", string(b))
		if !bytes.HasPrefix(b, []byte("one:")) {
			glog.Errorln("Wrong message from one")
			continue
		}
		raws := bytes.SplitN(b, []byte{':'}, 3)

		switch string(raws[1]) {
		case "IpcamsInfo":
			OnIpcamsInfo(room, raws[2])
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}

func (room *ControlRoom) WaitLogin() (ok bool) {
	_, addrb, err := room.ReadMessage()
	if err != nil {
		glog.Errorln(err)
		return
	}
	if !bytes.HasPrefix(addrb, []byte("addr:")) {
		glog.Errorln("Wrong addr from one")
		return
	}
	one, err := FindOne(addrb[5:])
	if err != nil {
		glog.Errorln(err)
		return
	}
	room.One = one
	ok = true
	return
}

func HandleOneCtrl(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		onOneCtrl(NewControlRoom(h, ws))
	}
}

func onOneCtrl(room *ControlRoom) {
	glog.Infoln("oneControlling start")
	if !room.WaitLogin() {
		return
	}

	room.Hub.reg <- room
	defer func() { room.Hub.unreg <- room }()

	go room.WritePump()
	room.ReadPump()
}

func OnIpcamsInfo(room *ControlRoom, info []byte) {
	var ipcams Ipcams
	if err := json.Unmarshal(info, &ipcams); err != nil {
		glog.Errorln(err)
		return
	}
	room.Cameras = ipcams
}

func HandleOneSignaling(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		glog.Infoln("one response signaling coming")
		res, err := h.RemoveReciever(c.Params.ByName("reciever"))
		if err != nil {
			panic(err)
		}
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			panic(err)
		}
		defer ws.Close()
		res <- ws
		<-res
	}
}

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
	*One
	Cameras       Ipcams
	Participants  map[int64]*ManyControlConn
	SendCtrlToOne chan []byte
}

func NewControlRoom(one *One) *ControlRoom {
	return &ControlRoom{
		One:           one,
		SendCtrlToOne: make(chan []byte, 64),
		Participants:  make(map[int64]*ManyControlConn),
	}
}

func (room *ControlRoom) broadcast(h *Hub, msg *Message) {
	msgStr, err := json.Marshal(msg)
	if err != nil {
		glog.Errorln(err)
		return
	}
	for _, ctrl := range room.Participants {
		select {
		case ctrl.send <- msgStr:
		default:
			close(ctrl.send)
			h.onLeave(ctrl)
		}
	}
}

func oneControlling(ws *websocket.Conn, c *gin.Context, h *Hub) {
	glog.Infoln("oneControlling start")
	// one is set in prev handler
	ione, err := c.Get(GinKeyOne)
	if err != nil {
		glog.Errorln(err)
		return
	}
	one, ok := ione.(*One)
	if !ok {
		glog.Errorln("One struct error")
		return
	}
	room := NewControlRoom(one)
	h.reg <- room
	defer func() { h.unreg <- room }()

	go writing(ws, room.SendCtrlToOne)

	glog.Infoln("starting read from one ctrl")

	for {
		_, b, err := ws.ReadMessage()
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

func OnIpcamsInfo(room *ControlRoom, info []byte) {
	var ipcams Ipcams
	if err := json.Unmarshal(info, &ipcams); err != nil {
		glog.Errorln(err)
		return
	}
	room.Cameras = ipcams
}

func oneSignaling(h *Hub) gin.HandlerFunc {
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

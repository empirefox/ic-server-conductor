package connections

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	. "github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/utils"
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
	Connection
	*One
	Cameras      Ipcams
	Participants map[uint]*ManyControlConn
	Send         chan []byte
	Hub          *Hub
}

func newControlRoom(h *Hub, ws *websocket.Conn) *ControlRoom {
	return &ControlRoom{
		Connection:   ws,
		Hub:          h,
		Cameras:      make(Ipcams),
		Send:         make(chan []byte, 64),
		Participants: make(map[uint]*ManyControlConn),
	}
}

func (room *ControlRoom) broadcast(msg []byte) {
	for _, ctrl := range room.Participants {
		select {
		case ctrl.Send <- msg:
		default:
			room.Close()
			room.Hub.onLeave(ctrl)
		}
	}
}

// no ping
func (room *ControlRoom) writePump() {
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

func (room *ControlRoom) readPump() {
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
		case "Ipcams":
			onOneIpcams(room, raws[2])
		case "ResponseToMany":
			onOneResponseToMany(room, raws[2])
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}

func (room *ControlRoom) waitLogin() (ok bool) {
	_, addrb, err := room.ReadMessage()
	if err != nil {
		glog.Errorln(err)
		return
	}
	if !bytes.HasPrefix(addrb, []byte("addr:")) {
		glog.Errorln("Wrong addr from one")
		return
	}
	one := &One{}
	if err = one.Find(addrb[5:]); err != nil {
		glog.Errorln(err)
		room.Send <- []byte(`{"name":"LoginAddrError"}`)
		return
	}
	room.One = one
	ok = true
	return
}

func HandleOneCtrl(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		handleOneCtrl(newControlRoom(h, ws))
	}
}

func handleOneCtrl(room *ControlRoom) {
	glog.Infoln("oneControlling start")
	if !room.waitLogin() {
		return
	}

	room.Hub.reg <- room
	defer func() { room.Hub.unreg <- room }()

	go room.writePump()
	room.readPump()
}

func onOneResponseToMany(room *ControlRoom, infoWithTo []byte) {
	// Ignore error if room/many closed
	defer recover()
	raws := bytes.SplitN(infoWithTo, []byte{':'}, 2)
	to, err := strconv.Atoi(string(raws[0]))
	if err != nil {
		glog.Errorln(err)
		return
	}
	room.Participants[uint(to)].Send <- raws[1]
}

func onOneIpcams(room *ControlRoom, info []byte) {
	var ipcams Ipcams
	if err := json.Unmarshal(info, &ipcams); err != nil {
		glog.Errorln(err)
		return
	}
	room.Cameras = ipcams
	for _, ctrl := range room.Participants {
		ctrl.sendCameraList()
	}
}

func HandleOneSignaling(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		glog.Infoln("one response signaling coming")
		res, err := h.processFromWait(c.Params.ByName("reciever"))
		if err != nil {
			panic(err)
		}
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			panic(err)
		}
		defer ws.Close()
		res <- ws
		<-res
	}
}

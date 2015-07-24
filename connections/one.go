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
func (room *ControlRoom) writePump(wait chan bool) {
	defer func() {
		room.Close()
		wait <- true
	}()
	for {
		select {
		case msg, ok := <-room.Send:
			if !ok {
				room.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := room.WriteMessage(websocket.TextMessage, msg); err != nil {
				glog.Infoln("ws send err:", err, string(msg))
				return
			}
		}
	}
}

func (room *ControlRoom) readPump() {
	defer room.Close()
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
		room.onRead(raws[1], raws[2])
	}
}

func (room *ControlRoom) onRead(typ, content []byte) {
	defer func() {
		if err := recover(); err != nil {
			glog.Infoln("onRead", string(typ), string(content), err)
		}
	}()
	switch string(typ) {
	case "Ipcams":
		onOneIpcams(room, content)
	case "ResponseToMany":
		onOneResponseToMany(room, content)
	case "ServerCommand":
		onServerCommand(room, content)
	default:
		glog.Errorln("Unknow command json:", string(typ), string(content))
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
		glog.Errorln(err, string(addrb))
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
		defer func() { ws.WriteMessage(websocket.CloseMessage, []byte{}) }()
		handleOneCtrl(newControlRoom(h, ws))
	}
}

func handleOneCtrl(room *ControlRoom) {
	glog.Infoln("oneControlling start")
	wait := make(chan bool)
	go room.writePump(wait)
	defer func() { <-wait }()

	if !room.waitLogin() {
		room.Send <- []byte(`{"name":"LoginAddrError"}`)
		return
	}
	room.Send <- []byte(`{"name":"LoginAddrOk"}`)

	room.Hub.reg <- room
	defer func() { room.Hub.unreg <- room }()

	room.readPump()
}

func onOneResponseToMany(room *ControlRoom, infoWithTo []byte) {
	raws := bytes.SplitN(infoWithTo, []byte{':'}, 2)
	to, err := strconv.Atoi(string(raws[0]))
	if err != nil {
		glog.Errorln(err)
		return
	}
	room.Participants[uint(to)].Send <- raws[1]
}

func onServerCommand(room *ControlRoom, command []byte) {
	var cmd ServerCommand
	if err := json.Unmarshal(command, &cmd); err != nil {
		glog.Errorln(err)
		return
	}
	glog.Infoln("cmd", cmd)
	switch cmd.Name {
	case "RemoveRoom":
		if err := room.One.Owner.RemoveOne(room.One); err != nil {
			glog.Errorln(err)
		}
		room.Send <- []byte(`{"name":"LoginAddrError"}`)
	}
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

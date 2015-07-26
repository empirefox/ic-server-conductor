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
func (room *ControlRoom) writePump() {
	defer func() { glog.Infoln("writePump close") }()
	defer room.Close()
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
			glog.Infoln("ws send to one:", string(msg))
		}
	}
}

func (room *ControlRoom) readPump() {
	defer func() { glog.Infoln("readPump close") }()
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
		if len(raws) < 3 {
			glog.Errorln("Not enough info in one message")
			continue
		}
		room.onRead(raws[1], raws[2])
	}
}

func (room *ControlRoom) onRead(typ, content []byte) {
	defer func() {
		if err := recover(); err != nil {
			glog.Infof("read from one, authed:%t, type:%s, content:%s, err:%v\n", typ, content, err)
		}
	}()
	if room.One != nil {
		room.onReadAuthed(typ, content)
	} else {
		room.onReadNotAuthed(typ, content)
	}
}

func (room *ControlRoom) onReadAuthed(typ, content []byte) {
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

func (room *ControlRoom) onReadNotAuthed(typ, content []byte) {
	switch string(typ) {
	case "Login":
		room.onLogin(content)
	default:
		glog.Errorln("Unknow command json:", string(typ), string(content))
	}
}

func (room *ControlRoom) onLogin(addr []byte) {
	one := &One{}
	if err := one.Find(addr); err != nil {
		glog.Errorln(err, string(addr))
		room.Send <- []byte(`{"name":"LoginAddrError"}`)
		return
	}
	room.One = one
	room.Hub.reg <- room
	room.Send <- []byte(`{"name":"LoginAddrOk"}`)
	glog.Infoln("one log ok")
}

func HandleOneCtrl(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		glog.Infoln("before ctrl upgrade")
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		glog.Infoln("oneControlling start")

		room := newControlRoom(h, ws)
		defer func() {
			glog.Infoln("unreg room")
			h.unreg <- room
		}()
		go room.writePump()
		room.readPump()
	}
}

func onOneResponseToMany(room *ControlRoom, infoWithTo []byte) {
	// [manyId]:[transfer]
	raws := bytes.SplitN(infoWithTo, []byte{':'}, 2)
	if len(raws) < 2 {
		glog.Errorln("No transfer data from one")
		return
	}
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
	glog.Infoln(ipcams)
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

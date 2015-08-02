package one

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	. "github.com/empirefox/ic-server-ws-signal/account"
	"github.com/empirefox/ic-server-ws-signal/conn"
	. "github.com/empirefox/ic-server-ws-signal/utils"
)

const (
	GinKeyOne = "one"
)

var (
	ErrRoomNotAuthed = errors.New("Room not authed")
)

type controlRoom struct {
	*websocket.Conn
	*One
	ipcams  conn.Ipcams
	onlines map[uint]conn.ControlUser
	send    chan []byte
	hub     conn.Hub
}

func newControlRoom(h conn.Hub, ws *websocket.Conn) *controlRoom {
	return &controlRoom{
		Conn:    ws,
		hub:     h,
		ipcams:  make(conn.Ipcams),
		send:    make(chan []byte, 64),
		onlines: make(map[uint]conn.ControlUser),
	}
}

func (room *controlRoom) GetOne() *One {
	return room.One
}

func (room *controlRoom) Id() uint {
	if room.One == nil {
		return 0
	}
	return room.ID
}

func (room *controlRoom) Send(msg []byte) {
	room.send <- msg
}

func (room *controlRoom) Broadcast(msg []byte) {
	for _, ctrl := range room.onlines {
		ctrl.Send(msg)
	}
}

func (room *controlRoom) Ipcams() conn.Ipcams {
	return room.ipcams
}

func (room *controlRoom) Friends() ([]Account, error) {
	if room.One == nil {
		return nil, ErrRoomNotAuthed
	}
	if err := room.Viewers(); err != nil {
		return nil, err
	}
	return room.Accounts, nil
}

func (room *controlRoom) AddOnline(id uint, cu conn.ControlUser) {
	room.onlines[id] = cu
}

func (room *controlRoom) GetOnline(id uint) (cu conn.ControlUser, ok bool) {
	cu, ok = room.onlines[id]
	return
}

func (room *controlRoom) RemoveOnline(id uint) {
	if room.onlines != nil {
		delete(room.onlines, id)
	}
}

// no ping
func (room *controlRoom) writePump() {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
		room.Close()
	}()
	for {
		select {
		case msg, ok := <-room.send:
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

func (room *controlRoom) readPump() {
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

func (room *controlRoom) onRead(typ, content []byte) {
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

func (room *controlRoom) onReadAuthed(typ, content []byte) {
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

func (room *controlRoom) onReadNotAuthed(typ, content []byte) {
	switch string(typ) {
	case "Login":
		room.onLogin(content)
	default:
		glog.Errorln("Unknow command json:", string(typ), string(content))
	}
}

func (room *controlRoom) onLogin(addr []byte) {
	one := &One{}
	if err := one.Find(addr); err != nil {
		glog.Errorln(err, string(addr))
		room.send <- []byte(`{"name":"LoginAddrError"}`)
		return
	}
	room.One = one
	room.hub.OnReg(room)
	room.send <- []byte(`{"name":"LoginAddrOk"}`)
}

func HandleOneCtrl(h conn.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()

		room := newControlRoom(h, ws)
		defer func() {
			h.OnUnreg(room)
			room.Broadcast([]byte(fmt.Sprintf(`{"type":"RoomOffline","content":%d}`, room.Id())))
		}()
		go room.writePump()
		room.readPump()
	}
}

func onOneResponseToMany(room *controlRoom, infoWithTo []byte) {
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
	room.onlines[uint(to)].Send(raws[1])
}

func onServerCommand(room *controlRoom, command []byte) {
	var cmd conn.ServerCommand
	if err := json.Unmarshal(command, &cmd); err != nil {
		glog.Errorln(err)
		return
	}
	glog.Infoln("cmd", cmd)
	switch cmd.Name {
	case "RemoveRoom":
		if err := room.Owner.RemoveOne(room.One); err != nil {
			glog.Errorln(err)
		}
		room.send <- []byte(`{"name":"LoginAddrError"}`)
	}
}

func onOneIpcams(room *controlRoom, info []byte) {
	var ipcams conn.Ipcams
	if err := json.Unmarshal(info, &ipcams); err != nil {
		glog.Errorln(err)
		return
	}
	room.ipcams = ipcams
	for _, ctrl := range room.onlines {
		ctrl.SendIpcams()
	}
}

func HandleOneSignaling(h conn.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := h.ProcessFromWait(c.Params.ByName("reciever"))
		if err != nil {
			glog.Errorln(err)
			return
		}
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		res <- ws
		<-res
	}
}

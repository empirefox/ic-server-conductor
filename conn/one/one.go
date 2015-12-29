package one

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dchest/uniuri"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	. "github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/utils"
)

var (
	ErrRoomNotAuthed = errors.New("Room not authed")
)

type controlRoom struct {
	*websocket.Conn
	*One
	ipcams     json.RawMessage
	onlines    map[uint]conn.ControlUser
	send       chan []byte
	hub        conn.Hub
	alg        string
	manyVerify conn.VerifyFunc
}

func newControlRoom(h conn.Hub, ws *websocket.Conn, alg string, manyVerify conn.VerifyFunc) *controlRoom {
	return &controlRoom{
		Conn:       ws,
		hub:        h,
		send:       make(chan []byte, 64),
		onlines:    make(map[uint]conn.ControlUser),
		alg:        alg,
		manyVerify: manyVerify,
	}
}

func (room *controlRoom) Tag() string { return "room" }

func (room *controlRoom) GetOne() *One { return room.One }

func (room *controlRoom) Id() uint {
	if room.One == nil {
		return 0
	}
	return room.ID
}

func (room *controlRoom) Send(msg []byte) { room.send <- msg }

func (room *controlRoom) Broadcast(msg []byte) {
	for _, ctrl := range room.onlines {
		ctrl.Send(msg)
	}
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

func (room *controlRoom) AddOnline(id uint, cu conn.ControlUser, tag string) {
	room.onlines[id] = cu
	switch tag {
	case "room":
		cu.Send([]byte(fmt.Sprintf(`{"type":"RoomOnline","ID":%d}`, room.Id())))
	case "user":
		room.send <- []byte(fmt.Sprintf(`{"from":%d,"name":"UserOnline"}`, id))
	default:
		glog.Errorln("Unknown tag:", tag)
	}
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
			glog.Infof("read from one, authed:%t, type:%s, content:%s, err:%v\n", room.One != nil, typ, content, err)
		}
	}()
	if room.One != nil {
		room.onReadAuthed(typ, content)
	} else {
		room.onReadNotAuthed(typ, content)
	}
}

func (room *controlRoom) doTargetT2M(to uint, k []byte, part json.RawMessage) {
	room.onlines[to].T2M(room.Id(), k, &part)
}

func (room *controlRoom) BroadcastT2M(k []byte, part json.RawMessage) {
	for _, ctrl := range room.onlines {
		ctrl.T2M(room.Id(), k, &part)
	}
}

func (room *controlRoom) onReadAuthed(typ, content []byte) {
	switch string(typ) {
	case "T2M":
		// "IcIds", "Ic", "IcIdCh", "XIc"
		room.onT2M(content)
	case "ServerCommand":
		onServerCommand(room, content)
	default:
		glog.Errorln("Unknow command json:", string(typ), string(content))
	}
}

func (room *controlRoom) onReadNotAuthed(typ, content []byte) {
	switch string(typ) {
	case "Login":
		room.send <- []byte(fmt.Sprintf(`{"name":"%s"}`, room.onLogin(content)))
	case "RegRoom":
		n, c := room.onRegRoom(content)
		room.send <- []byte(fmt.Sprintf(`{"name":"%s","content":"%s"}`, n, c))
	default:
		glog.Errorln("Unknow command json:", string(typ), string(content))
	}
}

func (room *controlRoom) onT2M(withTo []byte) {
	// [to(From)]:[raw json part]
	to, k, part, err := utils.ReadO2MSeg(withTo)
	if err != nil {
		glog.Errorln(err)
		return
	}
	if to == 0 {
		room.BroadcastT2M(k, json.RawMessage(part))
	} else {
		room.doTargetT2M(to, k, json.RawMessage(part))
	}
}

type regRoomData struct {
	Name string `json:"name"`
}

func (room *controlRoom) onRegRoom(regInfo []byte) (res, roomToken string) {
	res = "BadRegToken"
	// 1. Check msg format
	// [regToken]:[json]
	raws := bytes.SplitN(regInfo, []byte{':'}, 2)
	if len(raws) < 2 {
		glog.Errorln("No transfer data from one")
		return
	}
	// 2. Validate RegToken with many secret
	o := &Oauth{}
	err := room.manyVerify(o, raws[0])
	if err != nil {
		glog.Infoln("manyVerify err:", err)
		return
	}
	// 3. Parse RoomData
	var data regRoomData
	if err = json.Unmarshal(raws[1], &data); err != nil {
		glog.Infoln("Unmarshal err", err)
		return
	}
	// 4. Reg Room
	res = "RegError"
	one := &One{Addr: utils.NewRandom()}
	one.Name = data.Name
	if err = o.Account.RegOne(one); err != nil {
		glog.Infoln("RegOne err:", err)
		return
	}
	// 5. Generate RoomToken
	token := jwt.New(jwt.GetSigningMethod(room.alg))
	token.Claims["rid"] = one.ID
	token.Claims["aid"] = o.Account.ID
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["rnd"] = uniuri.New()
	roomToken, err = token.SignedString([]byte(one.Addr))
	glog.Infoln(one.Addr)
	if err != nil {
		glog.Infoln("SignedString err:", err)
		return
	}

	res = "SetRoomToken"
	return
}

func (room *controlRoom) onLogin(tokenBytes []byte) (res string) {
	res = "BadRoomToken"
	one := &One{}
	token, err := jwt.Parse(string(tokenBytes), func(token *jwt.Token) (interface{}, error) {
		rid, _ := token.Claims["rid"].(float64)
		aid, _ := token.Claims["aid"].(float64)
		glog.Infoln(token.Claims)
		if err := one.FindIfOwner(uint(rid), uint(aid)); err != nil {
			return nil, err
		}
		glog.Infoln(one.Addr)
		return []byte(one.Addr), nil
	})
	if err != nil || !token.Valid {
		glog.Infoln("Token is not valid:", err)
		return
	}
	room.One = one
	room.hub.OnReg(room)
	return "Broadcast"
}

func (room *controlRoom) offline() {
	if room.One == nil {
		return
	}
	room.Broadcast([]byte(fmt.Sprintf(`{"type":"RoomOffline","ID":%d}`, room.Id())))
	room.hub.OnUnreg(room)
	room.One = nil
}

func (room *controlRoom) Remove() {
	if room.One == nil {
		return
	}
	if err := room.Owner.RemoveOne(room.One); err != nil {
		glog.Errorln(err)
	}
	room.Broadcast([]byte(fmt.Sprintf(`{"type":"XRoom","ID":%d}`, room.Id())))
	room.hub.OnUnreg(room)
	room.send <- []byte(`{"name":"BadRoomToken"}`)
	room.One = nil
}

func HandleOneCtrl(h conn.Hub, alg string, manyVerify conn.VerifyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := utils.Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()

		room := newControlRoom(h, ws, alg, manyVerify)
		defer room.offline()
		go room.writePump()
		room.readPump()
	}
}

func onServerCommand(room *controlRoom, command []byte) {
	var cmd conn.ServerCommand
	if err := json.Unmarshal(command, &cmd); err != nil {
		glog.Errorln(err)
		return
	}
	switch cmd.Name {
	case "RemoveRoom":
		room.Remove()
	}
}

package many

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	. "github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/conn"
	. "github.com/empirefox/ic-server-conductor/utils"
)

var (
	ErrUserNotAuthed = errors.New("User not authed")
)

type controlUser struct {
	*websocket.Conn
	*Oauth
	send chan []byte
	hub  conn.Hub
	Exp  time.Time
}

func newControlUser(h conn.Hub, ws *websocket.Conn) *controlUser {
	return &controlUser{
		Conn: ws,
		hub:  h,
		send: make(chan []byte, 64),
	}
}

func (many *controlUser) Id() uint {
	if many.Oauth == nil {
		return 0
	}
	return many.Account.ID
}

func (many *controlUser) GetOauth() *Oauth {
	return many.Oauth
}

func (many *controlUser) Send(msg []byte) {
	many.send <- msg
}

func (many *controlUser) SendObj(obj interface{}) {
	msg, err := json.Marshal(obj)
	if err != nil {
		many.Send(GetTypedInfo(err.Error()))
		return
	}
	many.Send(msg)
}

func (many *controlUser) RoomOnes() ([]One, error) {
	if many.Oauth == nil {
		return nil, ErrUserNotAuthed
	}
	if err := many.Account.GetOnes(); err != nil {
		return nil, err
	}
	return many.Account.Ones, nil
}

func (many *controlUser) SendUserRoomList() {
	ones, err := many.RawRooms()
	if err != nil {
		many.Send(GetTypedInfo("Cannot get rooms"))
		return
	}
	many.SendObj(gin.H{"type": "Rooms", "content": ones})
}

func (many *controlUser) SendChangeRoomContent(oneId uint, ipcams *json.RawMessage) {
	many.SendObj(gin.H{"type": "Room", "id": oneId, "content": ipcams})
}

func (many *controlUser) SendUserRoomsContent() {
	ones := many.Account.Ones
	if ones == nil {
		return
	}
	for i := range ones {
		if room, ok := many.hub.GetRoom(ones[i].ID); ok {
			many.SendChangeRoomContent(ones[i].ID, room.Ipcams())
		}
	}
}

func (many *controlUser) SendUserRoomsView() {
	if views, err := many.Oauth.RawViewsByViewer(); err != nil {
		many.Send(GetTypedInfo("Cannot get views"))
	} else {
		many.SendObj(gin.H{"type": "RoomViews", "content": views})
	}
}

func (many *controlUser) SendUserIpcams() {
	many.SendUserRoomList()
	many.SendUserRoomsContent()
	many.SendUserRoomsView()
}

// with ping
func (many *controlUser) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
		ticker.Stop()
		many.Close()
	}()
	for {
		select {
		case msg, ok := <-many.send:
			if !ok {
				many.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := many.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
			glog.Infoln("ws send to many:", string(msg))
		case <-ticker.C:
			if err := many.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (many *controlUser) readPump() {
	for {
		_, b, err := many.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		glog.Infoln("From many client:", string(b))
		if !bytes.HasPrefix(b, []byte("many:")) {
			glog.Errorln("Wrong message from many")
			continue
		}
		// many:Chat:{"":""}
		raws := bytes.SplitN(b, []byte{':'}, 3)
		many.onRead(raws[1], raws[2])
	}
}

func (many *controlUser) onRead(typ, content []byte) {
	defer func() {
		if err := recover(); err != nil {
			glog.Infof("read from many, authed:%t, type:%s, content:%s, err:%v\n", typ, content, err)
		}
	}()
	if many.Oauth != nil {
		many.onReadAuthed(typ, content)
	} else {
		many.onReadNotAuthed(typ, content)
	}
}

func (many *controlUser) onReadAuthed(typ, content []byte) {
	switch string(typ) {
	case "Chat":
		many.onManyChat(content)
	case "Command":
		many.onManyCommand(content)
	case "GetManyData":
		many.onManyGetData(content)
	default:
		glog.Errorln("Unknow authed:", string(typ), string(content))
	}
}

func (many *controlUser) onReadNotAuthed(typ, content []byte) {
	glog.Errorln("Unknow unauthed:", string(typ), string(content))
}

func AuthMws(ws conn.Ws, vf conn.VerifyFunc) (*Oauth, error) {
	_, token, err := ws.ReadMessage()
	if err != nil {
		glog.Infoln("Read message err:", err)
		return nil, err
	}
	o := &Oauth{}
	if err = vf(o, token); err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"LoginFailed"}`))
		return nil, err
	}
	if err = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"LoginOk"}`)); err != nil {
		return nil, err
	}
	return o, nil
}

func HandleManyCtrl(h conn.Hub, vf conn.VerifyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		o, err := AuthMws(ws, vf)
		if err != nil {
			glog.Infoln("Auth failed:", err)
			return
		}
		many := newControlUser(h, ws)
		many.Oauth = o

		many.hub.OnJoin(many)
		defer func() { many.hub.OnLeave(many) }()

		go many.writePump()
		many.readPump()
	}
}

// on many control message

func (many *controlUser) onManyChat(bmsg []byte) {
	msg := &conn.Message{}
	if err := json.Unmarshal(bmsg, msg); err != nil {
		glog.Errorln(err)
		return
	}
	msg.From = many.Account.Name
	many.hub.OnMsg(msg)
}

func (many *controlUser) onManyCommand(bcmd []byte) {
	cmd := conn.ManyCommand{}
	if err := json.Unmarshal(bcmd, &cmd); err != nil {
		glog.Errorln(err)
		return
	}

	one := &One{}
	if err := one.FindIfOwner(cmd.Room, many.Account.ID); err != nil {
		glog.Errorln(err)
		return
	}

	switch cmd.Name {
	case "ManageSetRoomName":
		// Content: new_name
		// Proccess in server
		one.Name = string(cmd.Value())
		if err := one.Save(); err != nil {
			glog.Errorln(err)
			many.Send(GetTypedInfo("SetRoomName Error"))
			return
		}
		msg := []byte(fmt.Sprintf(`{
			"type":"Response","to":"ManageSetRoomName",
			"content":{"ID":%d,"Name":"%s"}
		}`, one.ID, one.Name))
		many.Send(msg)
	case "ManageDelRoom":
		if err := one.Delete(); err != nil {
			glog.Errorln(err)
			many.Send(GetTypedInfo("DelRoom Error"))
			return
		}
		room, ok := many.hub.GetRoom(cmd.Room)
		if ok {
			room.Close()
		}
		msg := []byte(fmt.Sprintf(`{
			"type":"Response","to":"ManageDelRoom",
			"content":%d
		}`, one.ID))
		many.Send(msg)
	case "ManageGetIpcam", "ManageSetIpcam", "ManageDelIpcam", "ManageReconnectIpcam":
		// Content(string): ipcam_id/ipcam/ipcam_id
		// Pass to One
		room, ok := many.hub.GetRoom(cmd.Room)
		if !ok {
			many.Send(GetTypedInfo("Room not online"))
			return
		}
		room.Send(GetNamedCmd(many.Account.ID, []byte(cmd.Name), cmd.Content))
	default:
		glog.Errorln("Unknow Command name:", cmd.Name)
		many.Send(GetTypedInfo("Unknow Command name:" + cmd.Name))
	}
}

func (many *controlUser) onManyGetData(name []byte) {
	switch string(name) {
	case "UserCameras":
		many.SendUserIpcams()
	default:
		glog.Errorln("Unknow GetManyData name:", string(name))
		many.Send(GetTypedInfo("Unknow GetManyData name:" + string(name)))
	}
}

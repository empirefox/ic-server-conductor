package connections

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/empirefox/gin-oauth2"
	. "github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/utils"
)

const (
	GinKeyUser = "user"
)

type ManyControlConn struct {
	Key string
	Connection
	*Oauth
	Send chan []byte
	Hub  *Hub
}

func newManyControlConn(h *Hub, ws *websocket.Conn) *ManyControlConn {
	return &ManyControlConn{
		Connection: ws,
		Hub:        h,
		Send:       make(chan []byte, 64),
	}
}

func (conn *ManyControlConn) getOauth(c *gin.Context) bool {
	iuser, err := c.Get(GinKeyUser)
	if err != nil {
		glog.Infoln(err)
		return false
	}
	user, ok := iuser.(*Oauth)
	if !ok {
		glog.Errorln("Account error")
		return false
	}
	conn.Oauth = user
	return true
}

type CameraRoom struct {
	Id      uint    `json:"id,omitempty"`
	Name    string  `json:"name,omitempty"`
	IsOwner bool    `json:"isOwner,omitempty"`
	Cameras []Ipcam `json:"cameras,omitempty"`
}

type CameraList struct {
	Type  string       `json:"type,omitempty"`
	Rooms []CameraRoom `json:"content,omitempty"`
}

func (conn *ManyControlConn) genCameraList() ([]byte, error) {
	list := CameraList{
		Type:  "CameraList",
		Rooms: make([]CameraRoom, 0, len(conn.Account.Ones)),
	}
	for _, one := range conn.Account.Ones {
		if room, ok := conn.Hub.rooms[one.ID]; ok {
			r := CameraRoom{
				Id:      one.ID,
				Name:    one.Name,
				IsOwner: one.OwnerId == conn.Account.ID,
				Cameras: make([]Ipcam, 0, len(room.Cameras)),
			}
			j := 0
			for _, ipcam := range room.Cameras {
				r.Cameras[j] = ipcam
				j++
			}
			list.Rooms = append(list.Rooms, r)
		}
	}
	cameraList, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	return cameraList, nil
}

func (conn *ManyControlConn) sendCameraList() {
	cameras, err := conn.genCameraList()
	if err != nil {
		conn.Send <- GetTypedInfo("Cannot get cameras")
		return
	}
	conn.Send <- cameras
}

// with ping
func (conn *ManyControlConn) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case msg, ok := <-conn.Send:
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
			glog.Infoln("ws send ", string(msg))
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (conn *ManyControlConn) readPump() {
	for {
		_, b, err := conn.ReadMessage()
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

		switch string(raws[1]) {
		case "Chat":
			onManyChat(conn, raws[2])
		case "Command":
			onManyCommand(conn, raws[2])
		case "GetManyData":
			onManyGetData(conn, raws[2])
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}

func HandleManyCtrl(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		conn := newManyControlConn(h, ws)
		if !conn.getOauth(c) {
			return
		}
		handleManyCtrl(conn)
	}
}

// TODO next add manage api
func handleManyCtrl(conn *ManyControlConn) {
	glog.Infoln("oneControlling start")

	conn.Hub.join <- conn
	defer func() { conn.Hub.leave <- conn }()

	go conn.writePump()
	conn.readPump()
}

// on many control message

func onManyChat(many *ManyControlConn, bmsg []byte) {
	msg := EmptyMessage()
	if err := json.Unmarshal(bmsg, msg); err != nil {
		glog.Errorln(err)
		msg.Free()
		return
	}
	msg.From = many.Account.Name
	many.Hub.msg <- msg
}

func onManyCommand(many *ManyControlConn, bcmd []byte) {
	cmd := EmptyCommand()
	defer cmd.Free()
	if err := json.Unmarshal(bcmd, cmd); err != nil {
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
		one.Name = cmd.Content
		if err := one.Save(); err != nil {
			glog.Errorln(err)
			many.Send <- GetTypedInfo("SetRoomName Error")
			return
		}
		many.sendCameraList()
	case "ManageGetIpcam", "ManageSetIpcam", "ManageReconnectIpcam":
		// Content(string): ipcam_id/ipcam/ipcam_id
		// Pass to One
		room, ok := many.Hub.rooms[cmd.Room]
		if !ok {
			many.Send <- GetTypedInfo("Room not online")
			return
		}
		room.Send <- GetNamedCmd(many.ID, cmd.Name, cmd.Content)
	default:
		glog.Errorln("Unknow Command name:", cmd.Name)
		many.Send <- GetTypedInfo("Unknow Command name:" + cmd.Name)
	}
}

func onManyGetData(many *ManyControlConn, name []byte) {
	switch string(name) {
	case "Userinfo":
		many.Send <- GetTypedMsgStr(string(name), many.Account.Name)
	case "CameraList":
		many.sendCameraList()
	default:
		glog.Errorln("Unknow GetManyData name:", string(name))
		many.Send <- GetTypedInfo("Unknow GetManyData name:" + string(name))
	}
}

// many signaling

func HandleManySignaling(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		glog.Infoln("many signaling coming")
		res, reciever := preProccessSignaling(h, c)
		var resWs *websocket.Conn
		select {
		case resWs = <-res:
		case <-time.After(time.Second * 15):
			h.processFromWait(reciever)
			panic("Wait for one signaling timeout")
		}
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			panic(err)
		}
		defer ws.Close()
		Pipe(ws, resWs)
		res <- nil
	}
}

type CreateSignalingSubCommand struct {
	Camera   string `json:"camera,omitempty"`
	Reciever string `json:"reciever,omitempty"`
}

type CreateSignalingConnectionCommand struct {
	Name    string                    `json:"name"`
	Content CreateSignalingSubCommand `json:"content"`
}

func preProccessSignaling(h *Hub, c *gin.Context) (res chan *websocket.Conn, reciever string) {
	roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
	if err != nil {
		panic("No room set in context")
	}
	room, ok := h.rooms[uint(roomId)]
	if !ok {
		panic("Room not found in request")
	}
	cameras := room.Cameras
	if cameras == nil {
		panic("Cameras not found in room")
	}
	cmd := CreateSignalingConnectionCommand{
		Name: "CreateSignalingConnection",
		Content: CreateSignalingSubCommand{
			Camera:   c.Params.ByName("camera"),
			Reciever: c.Params.ByName("reciever"),
		},
	}
	_, ok = cameras[cmd.Content.Camera]
	if !ok {
		panic("Camera not found in room")
	}
	cmdStr, err := json.Marshal(cmd)
	if err != nil {
		panic(err)
	}
	res, err = h.waitForProcess(cmd.Content.Reciever)
	if err != nil {
		panic(err)
	}
	room.Send <- cmdStr
	return res, cmd.Content.Reciever
}

func HandleManyCheckLogin(config *goauth.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, _ := config.CheckStatus(c, goauth.Permitted); ok {
			c.JSON(http.StatusOK, "")
		} else {
			c.JSON(http.StatusUnauthorized, "")
		}
	}
}

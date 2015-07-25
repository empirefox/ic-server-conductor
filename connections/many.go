package connections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	"github.com/empirefox/gin-oauth2"
	gws "github.com/empirefox/gotool/ws"
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
	iuser, ok := c.Get(GinKeyUser)
	if !ok {
		glog.Infoln("user not found")
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
		Rooms: make([]CameraRoom, 0),
	}
	for _, one := range conn.Account.Ones {
		if room, ok := conn.Hub.rooms[one.ID]; ok {
			r := CameraRoom{
				Id:      one.ID,
				Name:    one.Name,
				IsOwner: one.OwnerId == conn.Account.ID,
				Cameras: make([]Ipcam, 0),
			}
			for _, ipcam := range room.Cameras {
				r.Cameras = append(r.Cameras, ipcam)
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
			glog.Infoln("ws send to many:", string(msg))
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
	cmd := ManyCommand{}
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
		one.Name = string(cmd.Content)
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
		room.Send <- GetNamedCmd(many.ID, []byte(cmd.Name), cmd.Content)
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
		if res == nil {
			return
		}
		var resWs *websocket.Conn
		select {
		case resWs = <-res:
		case <-time.After(time.Second * 15):
			h.processFromWait(reciever)
			glog.Infoln("Wait for one signaling timeout")
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Infoln("Upgrade failed:", err)
			c.AbortWithStatus(http.StatusBadGateway)
			return
		}
		defer ws.Close()
		gws.Pipe(ws, resWs)
		res <- nil
	}
}

// to ic-one-client Center.Command
type CreateSignalingConnectionCommand struct {
	From    uint   `json:"from"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func preProccessSignaling(h *Hub, c *gin.Context) (res chan *websocket.Conn, reciever string) {
	roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
	if err != nil {
		glog.Infoln("No room set in context:", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	room, ok := h.rooms[uint(roomId)]
	if !ok {
		glog.Infoln("Room not found in request")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	cameras := room.Cameras
	if cameras == nil {
		glog.Infoln("Cameras not found in room")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	camera := c.Params.ByName("camera")
	_, ok = cameras[camera]
	if !ok {
		glog.Infoln("Camera not found in room")
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	reciever = c.Params.ByName("reciever")
	res, err = h.waitForProcess(reciever)
	if err != nil {
		glog.Infoln("Wait for process:", err)
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	cmd := fmt.Sprintf(`{
		"name":"CreateSignalingConnection",
		"from":%d,
		"content":{
			"camera":"%s", "reciever":"%s"
		}
	}`, room.ID, camera, reciever)
	room.Send <- []byte(cmd)
	return res, reciever
}

func HandleManyCheckLogin(conf *goauth.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ok, _ := conf.CheckStatus(c, goauth.Permitted); ok {
			c.JSON(http.StatusOK, "")
		} else {
			c.JSON(http.StatusUnauthorized, "")
		}
	}
}

type regRoomData struct {
	Name string `json:"name"`
}

func HandleManyRegRoom(h *Hub, conf *goauth.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data regRoomData
		if err := c.Bind(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err, "content": data})
			return
		}

		one := One{SecretAddress: NewUUID()}
		one.Name = data.Name
		if err := c.Keys[conf.UserGinKey].(*Oauth).Account.RegOne(&one); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err, "content": one})
			return
		}
		c.JSON(http.StatusOK, gin.H{"addr": one.SecretAddress})
	}
}

func HandleManyLogoff(h *Hub, conf *goauth.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		iuser, ok := c.Get(conf.UserGinKey)
		if !ok {
			c.JSON(http.StatusForbidden, `{"error":1,"content":"user not authed"}`)
			return
		}
		if err := iuser.(*Oauth).Account.Logoff(); err != nil {
			c.JSON(http.StatusInternalServerError, `{"error":1,"content":"cannot del user"}`)
			return
		}
		c.Redirect(http.StatusSeeOther, conf.PathLogout)
	}
}

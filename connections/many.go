package connections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"

	gws "github.com/empirefox/gotool/ws"
	. "github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/utils"
)

const (
	UserKey = "user"
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
	iuser, ok := c.Get(UserKey)
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
	if err := conn.GetOnes(); err != nil {
		return nil, err
	}
	list := CameraList{
		Type:  "CameraList",
		Rooms: make([]CameraRoom, 0),
	}
	for _, one := range conn.Account.Ones {
		r := CameraRoom{
			Id:      one.ID,
			Name:    one.Name,
			IsOwner: one.OwnerId == conn.Account.ID,
			Cameras: make([]Ipcam, 0),
		}
		if room, ok := conn.Hub.rooms[one.ID]; ok {
			for _, ipcam := range room.Cameras {
				r.Cameras = append(r.Cameras, ipcam)
			}
		}
		list.Rooms = append(list.Rooms, r)
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
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
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
		conn.onRead(raws[1], raws[2])
	}
}

func (conn *ManyControlConn) onRead(typ, content []byte) {
	defer func() {
		if err := recover(); err != nil {
			glog.Infof("read from many, authed:%t, type:%s, content:%s, err:%v\n", typ, content, err)
		}
	}()
	if conn.Oauth != nil {
		conn.onReadAuthed(typ, content)
	} else {
		conn.onReadNotAuthed(typ, content)
	}
}

func (conn *ManyControlConn) onReadAuthed(typ, content []byte) {
	switch string(typ) {
	case "Chat":
		onManyChat(conn, content)
	case "Command":
		onManyCommand(conn, content)
	case "GetManyData":
		onManyGetData(conn, content)
	default:
		glog.Errorln("Unknow authed:", string(typ), string(content))
	}
}

func (conn *ManyControlConn) onReadNotAuthed(typ, content []byte) {
	switch string(typ) {
	case "Login":
		conn.onLogin(content)
	default:
		glog.Errorln("Unknow unauthed:", string(typ), string(content))
	}
}

func (conn *ManyControlConn) onLogin(tokenBytes []byte) {
	token, err := jwt.Parse(string(tokenBytes), func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return conn.Hub.tokenSecret, nil
	})

	if err != nil {
		glog.Infoln("Parse token:", err)
		conn.Close()
		return
	}
	if !token.Valid {
		glog.Infoln("Token is not valid")
		conn.Close()
		return
	}
	o := &Oauth{}
	oa := []byte(token.Claims["oauth"].(string))
	err = json.Unmarshal(oa, o)
	if err != nil {
		glog.Infoln("Unmarshal err:", err)
		conn.Close()
		return
	}
	conn.Oauth = o
	conn.Hub.join <- conn
	conn.Send <- []byte(`{"type":"Login","content":1}`)
}

func AuthMws(conn Connection, secret interface{}) (*Oauth, error) {
	token, err := AuthWs(conn, secret)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, `{"type":"Info","content":"Auth token failed"}`)
		return nil, err
	}
	o := &Oauth{}
	if err = o.FromToken(token); err != nil {
		conn.WriteMessage(websocket.TextMessage, `{"type":"Info","content":"Auth failed"}`)
		return nil, err
	}
	return o, nil
}

func HandleManyCtrl(h *Hub, secret interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		o, err := AuthMws(ws, secret)
		if err != nil {
			return
		}
		conn := newManyControlConn(h, ws)
		conn.Oauth = o
		handleManyCtrl(conn)
	}
}

// TODO next add manage api
func handleManyCtrl(conn *ManyControlConn) {
	conn.Hub.join <- conn
	defer func() { conn.Hub.leave <- conn }()

	go conn.writePump()
	conn.readPump()
}

// on many control message

func onManyChat(many *ManyControlConn, bmsg []byte) {
	msg := &Message{}
	if err := json.Unmarshal(bmsg, msg); err != nil {
		glog.Errorln(err)
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
		many.Send <- []byte(fmt.Sprintf(`{
			"type":"ManageSetRoomName",
			"content":{"name":"%s"}
		}`, one.Name))
	case "ManageGetIpcam", "ManageSetIpcam", "ManageDelIpcam", "ManageReconnectIpcam":
		// Content(string): ipcam_id/ipcam/ipcam_id
		// Pass to One
		room, ok := many.Hub.rooms[cmd.Room]
		if !ok {
			many.Send <- GetTypedInfo("Room not online")
			return
		}
		room.Send <- GetNamedCmd(many.Account.ID, []byte(cmd.Name), cmd.Content)
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

type StartSignalingInfo struct {
	Room     uint   `json:"room"`
	Camera   string `json:"camera"`
	Reciever string `json:"reciever"`
}

// many signaling
func HandleManySignaling(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Infoln("Upgrade failed:", err)
			c.AbortWithStatus(http.StatusBadGateway)
			return
		}
		defer ws.Close()
		_, err := AuthMws(ws, secret)
		if err != nil {
			return
		}

		ws.WriteMessage(websocket.TextMessage, `{"type":"accepted"}`)
		_, startInfo, err := ws.ReadMessage()
		if err != nil {
			glog.Infoln("Read start info err:", err)
			return
		}

		var info StartSignalingInfo
		if err := json.Unmarshal(startInfo, &info); err != nil {
			glog.Infoln("Unmarshal info err:", err)
			return
		}

		res := preProccessSignaling(h, c, &info)
		if res == nil {
			return
		}
		var resWs *websocket.Conn
		select {
		case resWs = <-res:
		case <-time.After(time.Second * 15):
			h.processFromWait(info.Reciever)
			glog.Infoln("Wait for one signaling timeout")
			return
		}
		gws.Pipe(ws, resWs)
		res <- nil
	}
}

func preProccessSignaling(h *Hub, c *gin.Context, info *StartSignalingInfo) chan *websocket.Conn {
	room, ok := h.rooms[info.Room]
	if !ok {
		glog.Infoln("Room not found in request")
		return nil
	}
	cameras := room.Cameras
	if cameras == nil {
		glog.Infoln("Cameras not found in room")
		return nil
	}
	_, ok = cameras[info.Camera]
	if !ok {
		glog.Infoln("Camera not found in room")
		return nil
	}
	res, err := h.waitForProcess(info.Reciever)
	if err != nil {
		glog.Infoln("Wait for process:", err)
		return nil
	}
	cmd := fmt.Sprintf(`{
		"name":"CreateSignalingConnection",
		"from":%d,
		"content":{
			"camera":"%s", "reciever":"%s"
		}
	}`, info.Room, info.Camera, info.Reciever)
	room.Send <- []byte(cmd)
	return res
}

type regRoomData struct {
	Name string `json:"name"`
}

func HandleManyRegRoom(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data regRoomData
		if err := c.Bind(&data); err != nil {
			glog.Infoln(err)
			c.JSON(http.StatusBadRequest, Err("User bind err"))
			return
		}

		one := One{SecretAddress: NewUUID()}
		one.Name = data.Name
		if err := c.Keys[UserKey].(*Oauth).Account.RegOne(&one); err != nil {
			c.JSON(http.StatusInternalServerError, Err("Reg err"))
			return
		}
		c.JSON(http.StatusOK, OK(one.SecretAddress))
	}
}

func HandleManyLogoff(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		iuser, ok := c.Get(UserKey)
		if !ok {
			c.JSON(http.StatusForbidden, `{"error":1,"content":"user not authed"}`)
			return
		}
		if err := iuser.(*Oauth).Account.Logoff(); err != nil {
			c.JSON(http.StatusInternalServerError, `{"error":1,"content":"cannot del user"}`)
			return
		}
		c.JSON(http.StatusOK, `{"error":0,"content":"log off ok"}`)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	GinKeyUser = "user"
)

type ManyControlConn struct {
	*websocket.Conn
	*Oauth
	Send chan []byte
	Hub  *Hub
}

func NewManyControlConn(h *Hub, ws *websocket.Conn) *ManyControlConn {
	return &ManyControlConn{
		Conn: ws,
		Hub:  h,
		Send: make(chan []byte, 64),
	}
}

func (conn *ManyControlConn) GetOauth(c *gin.Context) bool {
	iuser, err := c.Get(GinKeyUser)
	if err != nil {
		glog.Errorln(err)
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
	Cameras []Ipcam `json:"cameras,omitempty"`
}

type CameraList struct {
	Type  string       `json:"type,omitempty"`
	Rooms []CameraRoom `json:"rooms,omitempty"`
}

func (conn *ManyControlConn) SendCameraList() error {
	list := CameraList{
		Type:  "CameraList",
		Rooms: make([]CameraRoom, len(conn.Oauth.Account.Ones)),
	}
	for i, one := range conn.Oauth.Account.Ones {
		if room, ok := conn.Hub.rooms[one.ID]; ok {
			list.Rooms[i] = CameraRoom{
				Id:      one.ID,
				Name:    one.Name,
				Cameras: make([]Ipcam, len(room.Cameras)),
			}
			j := 0
			for _, ipcam := range room.Cameras {
				list.Rooms[i].Cameras[j] = ipcam
				j++
			}
		}
	}
	cameraList, err := json.Marshal(list)
	if err != nil {
		return err
	}
	conn.Send <- cameraList
	return nil
}

// with ping
func (conn *ManyControlConn) WritePump() {
	ticker := time.NewTicker(pingPeriod)
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

func (conn *ManyControlConn) ReadPump() {
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
			OnManyChat(conn, raws[2])
		case "Command":
			OnManyCommand(conn, raws[2])
		case "GetManyData":
			OnManyGetData(conn, raws[2])
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}

func HandleManyCtrl(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		conn := NewManyControlConn(h, ws)
		if !conn.GetOauth(c) {
			return
		}
		onManyCtrl(conn)
	}
}

// TODO next add manage api
func onManyCtrl(conn *ManyControlConn) {
	glog.Infoln("oneControlling start")

	conn.Hub.join <- conn
	defer func() { conn.Hub.leave <- conn }()

	go conn.WritePump()
	conn.ReadPump()
}

// on many control message

func OnManyChat(many *ManyControlConn, bmsg []byte) {
	msg := EmptyMessage()
	if err := json.Unmarshal(bmsg, msg); err != nil {
		glog.Errorln(err)
		msg.Free()
		return
	}
	msg.From = many.Account.Name
	many.Hub.msg <- msg
}

func OnManyCommand(many *ManyControlConn, bcmd []byte) {
	cmd := EmptyCommand()
	if err := json.Unmarshal(bcmd, cmd); err != nil {
		glog.Errorln(err)
		cmd.Free()
		return
	}
	many.Hub.cmd <- cmd
}

func OnManyGetData(many *ManyControlConn, name []byte) {
	switch string(name) {
	case "Username":
		msg, err := GetTypedMsg(string(name), many.Account.Name)
		if err != nil {
			glog.Errorln(err)
			return
		}
		many.Send <- msg
	case "CameraList":
		many.SendCameraList()
	default:
		glog.Errorln("Unknow GetManyData name:", string(name))
	}
}

// many signaling

type CreateSignalingConnectionCommand struct {
	Name     string `json:"name"`
	Reciever string `json:"reciever"`
	Camera   string `json:"camera"`
}

func HandleManySignaling(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		glog.Infoln("many signaling coming")
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
			Name:     "CreateSignalingConnection",
			Camera:   c.Params.ByName("camera"),
			Reciever: c.Params.ByName("reciever"),
		}
		_, ok = cameras[cmd.Camera]
		if !ok {
			panic("Camera not found in room")
		}
		cmdStr, err := json.Marshal(cmd)
		if err != nil {
			panic(err)
		}
		res, err := h.AddReciever(cmd.Reciever)
		if err != nil {
			panic(err)
		}
		room.Send <- cmdStr
		var resWs *websocket.Conn
		select {
		case resWs = <-res:
		case <-time.After(time.Second * 15):
			h.RemoveReciever(cmd.Reciever)
			panic("Wait for one signaling timeout")
		}
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			panic(err)
		}
		defer ws.Close()
		Pipe(ws, resWs)
		res <- nil
	}
}

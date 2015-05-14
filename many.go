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
	*Account
	send chan []byte
}

type CameraRoom struct {
	Id      int64   `json:"id,omitempty"`
	Name    string  `json:"name,omitempty"`
	Cameras []Ipcam `json:"cameras,omitempty"`
}

type CameraList struct {
	Type  string       `json:"type,omitempty"`
	Rooms []CameraRoom `json:"rooms,omitempty"`
}

func (conn *ManyControlConn) SendCameraList(h *Hub) error {
	list := CameraList{
		Type:  "CameraList",
		Rooms: make([]CameraRoom, len(conn.Account.Ones)),
	}
	for i, one := range conn.Account.Ones {
		if room, ok := h.rooms[one.Id]; ok {
			list.Rooms[i] = CameraRoom{
				Id:      one.Id,
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
	conn.send <- cameraList
	return nil
}

// TODO next add manage api
func manyControlling(ws *websocket.Conn, c *gin.Context, h *Hub) {
	glog.Infoln("oneControlling start")
	// one is set in prev handler
	iuser, err := c.Get(GinKeyUser)
	if err != nil {
		glog.Errorln(err)
		return
	}
	user, ok := iuser.(*Account)
	if !ok {
		glog.Errorln("Account error")
		return
	}

	send := make(chan []byte, 64)
	many := &ManyControlConn{
		Conn:    ws,
		Account: user,
		send:    send,
	}

	h.join <- many
	defer func() { h.leave <- many }()

	go writingWithPing(ws, send)

	if err = many.SendCameraList(h); err != nil {
		glog.Errorln(err)
		return
	}

	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		glog.Infoln("From many client:", string(b))
		if !bytes.HasPrefix(b, []byte("many:")) {
			glog.Errorln("Wrong message from many")
			continue
		}
		raws := bytes.SplitN(b, []byte{':'}, 3)

		switch string(raws[1]) {
		case "Chat":
			OnChat(h, raws[2])
		case "Command":
			OnCommand(h, raws[2])
		default:
			glog.Errorln("Unknow command json:", string(b))
		}
	}
}

func OnChat(h *Hub, bmsg []byte) {
	msg := EmptyMessage()
	if err := json.Unmarshal(bmsg, msg); err != nil {
		glog.Errorln(err)
		msg.Free()
		return
	}
	h.msg <- msg
}

func OnCommand(h *Hub, bcmd []byte) {
	cmd := EmptyCommand()
	if err := json.Unmarshal(bcmd, cmd); err != nil {
		glog.Errorln(err)
		cmd.Free()
		return
	}
	h.cmd <- cmd
}

type CreateSignalingConnectionCommand struct {
	Name     string `json:"name"`
	Reciever string `json:"reciever"`
	Camera   string `json:"camera"`
}

func manySignaling(h *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		glog.Infoln("many signaling coming")
		roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
		if err != nil {
			panic("No room set in context")
		}
		room, ok := h.rooms[roomId]
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
		room.SendCtrlToOne <- cmdStr
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

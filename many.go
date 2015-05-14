package main

import (
	"bytes"
	"encoding/json"

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
	rooms []int64
	send  chan []byte
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

// send and refresh cameras to web client
// TODO next add manage api
func manyControlling(ws *websocket.Conn, c *gin.Context, h *Hub) {
	glog.Infoln("oneControlling start")
	// one is set in prev handler
	iuser, err := c.Get(GinKeyUser)
	if err != nil {
		glog.Errorln(err)
		return
	}
	ones := iuser.(*Account).Ones
	roomIds := make([]int64, len(ones))
	list := CameraList{
		Type:  "CameraList",
		Rooms: make([]CameraRoom, len(ones)),
	}
	for i, one := range ones {
		if room, ok := h.rooms[one.Id]; ok {
			roomIds[i] = one.Id
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

	send := make(chan []byte, 64)
	many := &ManyControlConn{
		Conn:    ws,
		Account: iuser.(*Account),
		rooms:   roomIds,
		send:    send,
	}

	h.join <- many
	defer func() { h.leave <- many }()

	go writingWithPing(ws, send)

	cameraList, err := json.Marshal(list)
	if err != nil {
		glog.Errorln(err)
		return
	}
	send <- cameraList

	for {
		_, b, err := ws.ReadMessage()
		if err != nil {
			glog.Errorln(err)
			return
		}
		glog.Infoln("From many client:", string(b))
		if !bytes.HasPrefix(b, []byte("many:")) {
			glog.Errorln("Wrong message from many")
			return
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

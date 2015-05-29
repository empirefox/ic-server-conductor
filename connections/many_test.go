package connections

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	. "github.com/smartystreets/goconvey/convey"
)

type fakeManyConn struct {
	ManyControlConn
	OnSend func(msg []byte)
	quit   chan bool
}

func NewFakeManyConn(msg string, onSend func([]byte)) *fakeManyConn {
	h := NewHub()
	return &fakeManyConn{
		ManyControlConn: ManyControlConn{
			Connection: newFakeWsConn(websocket.TextMessage, msg),
			Oauth:      &account.Oauth{},
			Send:       make(chan []byte, 64),
			Hub:        h,
		},
		OnSend: onSend,
		quit:   make(chan bool),
	}
}

func (c *fakeManyConn) Quit() {
	close(c.quit)
}

func (c *fakeManyConn) writePump() {
	select {
	case msg := <-c.ManyControlConn.Send:
		c.OnSend(msg)
	case <-c.quit:
		return
	}
}

type msgGetter struct {
	ok     chan bool
	Result []byte
}

func (g *msgGetter) GenOnSend() func([]byte) {
	g.ok = make(chan bool)
	return func(msg []byte) {
		g.Result = msg
		g.ok <- true
	}
}

func (g *msgGetter) Wait() {
	<-g.ok
}

func Test__ProcessCtrlMessage(t *testing.T) {
	Convey("ProcessCtrlMessage", t, func() {
		Convey("onManyChat should send msg to hub.msg", func() {
			conn := NewFakeManyConn(`many:Chat:{"to":2}`, nil)
			conn.Account.Name = "bob"
			go conn.readPump()
			msg := <-conn.Hub.msg
			So(msg, ShouldResemble, &Message{From: "bob", Room: 2})
			msg.Free()
		})
		Convey("onManyCommand should send cmd to hub.cmd", func() {
			service := fakeService{addr: "a-128"}
			account.SetService(service)
			defer account.SetService(nil)

			conn := NewFakeManyConn(`many:Command:{"room":12,"name":"ManageGetIpcam","content":"123"}`, nil)
			conn.Oauth = &account.Oauth{}
			conn.ID = 11
			room := &ControlRoom{Send: make(chan []byte)}
			conn.Hub.rooms[12] = room
			go conn.readPump()
			cmd := <-room.Send
			So(string(cmd), ShouldEqual, `{"from":11,"name":"ManageGetIpcam","content":"123"}`)
		})
		Convey("onManyGetData", func() {
			Convey("should proccess Username", func() {
				var g msgGetter
				conn := NewFakeManyConn(`many:GetManyData:Username`, g.GenOnSend())
				conn.Account.Name = "bob"
				go conn.readPump()
				go conn.writePump()
				g.Wait()
				So(string(g.Result), ShouldResemble, `{"type":"Username","content":"bob"}`)
			})
			Convey("should proccess CameraList", func() {
				one1 := account.One{SecretAddress: "addr1"}
				one1.ID = 1
				one1.Name = "one1"
				room1 := &ControlRoom{One: &one1}
				one2 := account.One{SecretAddress: "addr2"}
				one2.ID = 2
				one2.Name = "one2"
				var g msgGetter
				conn := NewFakeManyConn(`many:GetManyData:CameraList`, g.GenOnSend())
				conn.Account.Ones = []account.One{one1, one2}
				conn.Hub.rooms = map[uint]*ControlRoom{1: room1}
				go conn.readPump()
				go conn.writePump()
				g.Wait()
				var list CameraList
				err := json.Unmarshal(g.Result, &list)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, CameraList{
					Type: "CameraList",
					Rooms: []CameraRoom{
						{
							Id:   1,
							Name: "one1",
						},
					},
				})
			})
		})
	})
}

func Test_preProccessSignaling(t *testing.T) {
	Convey("preProccessSignaling", t, func() {
		Convey("should return a ws chan", func() {
			c := &gin.Context{
				Params: httprouter.Params{
					{"room", "1"},
					{"camera", "2"},
					{"reciever", "3"},
				},
			}
			one1 := account.One{SecretAddress: "addr1"}
			one1.ID = 1
			one1.Name = "one1"
			room1 := &ControlRoom{
				One:     &one1,
				Cameras: Ipcams{"2": Ipcam{Id: "2"}},
				Send:    make(chan []byte, 1),
			}
			h := &Hub{
				sigResWaitMap: make(map[string]chan *websocket.Conn),
				rooms: map[uint]*ControlRoom{
					1: room1,
				},
			}
			res, reciever := preProccessSignaling(h, c)
			So(reciever, ShouldEqual, "3")
			ws := &websocket.Conn{}
			go func() {
				resWait, _ := h.processFromWait("3")
				resWait <- ws
			}()
			var resWs *websocket.Conn
			select {
			case resWs = <-res:
			case <-time.After(time.Second * 3):
				t.Errorf("res timeout")
			}
			So(resWs, ShouldResemble, ws)
		})
	})
}

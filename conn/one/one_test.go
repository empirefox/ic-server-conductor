package one

import (
	"errors"
	"testing"

	. "github.com/empirefox/ic-server-ws-signal/account"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	. "github.com/smartystreets/goconvey/convey"
)

type fakeRoom struct {
	ControlRoom
	OnSend func(msg []byte)
	quit   chan bool
}

func NewFakeRoom(msg string, onSend func([]byte)) *fakeRoom {
	h := NewHub()
	return &fakeRoom{
		ControlRoom: ControlRoom{
			Connection: newFakeWsConn(websocket.TextMessage, msg),
			Send:       make(chan []byte, 64),
			Hub:        h,
		},
		OnSend: onSend,
	}
}

type fakeService struct {
	addr string
}

func (fakeService) CreateTables() error                          { return nil }
func (fakeService) DropTables() error                            { return nil }
func (fakeService) FindOauthProviders(ops *OauthProviders) error { return nil }
func (fakeService) SaveOauthProvider(ops *OauthProvider) error   { return nil }
func (fakeService) OnOid(o *Oauth, provider, oid string) error   { return nil }
func (fakeService) Permitted(o *Oauth, c *gin.Context) bool      { return true }
func (fakeService) Valid(o *Oauth) bool                          { return true }
func (fakeService) GetOnes(a *Account) error                     { return nil }
func (fakeService) RegOne(a *Account, o *One) error              { return nil }
func (fakeService) ViewOne(a *Account, o *One) error             { return nil }
func (fakeService) RemoveOne(a *Account, o *One) error           { return nil }
func (fakeService) Logoff(a *Account) error                      { return nil }
func (s fakeService) FindOne(o *One, addr []byte) error {
	if string(addr) == s.addr {
		return nil
	}
	return errors.New("addr not found")
}

func (s fakeService) FindOneIfOwner(o *One, id, ownerId uint) error { return nil }
func (s fakeService) Save(o *One) error                             { return nil }
func (s fakeService) Viewers(o *One) error                          { return nil }

func Test_readPump(t *testing.T) {
	Convey("readPump", t, func() {
		Convey("onOneIpcamsInfo should get ipcams from msg", func() {
			room := NewFakeRoom(`one:Ipcams:{"id1":{"id":"id1"},"id2":{"id":"id2"}}`, nil)
			room.readPump()
			So(room.Cameras, ShouldResemble, Ipcams{"id1": Ipcam{Id: "id1"}, "id2": Ipcam{Id: "id2"}})
		})
	})
}

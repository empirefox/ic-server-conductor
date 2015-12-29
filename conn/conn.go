package conn

import (
	"encoding/json"

	"github.com/empirefox/ic-server-conductor/account"
	"github.com/gorilla/websocket"
)

// Wrap websocket.Conn and common fn
type Ws interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

type Connection interface {
	Ws
	Id() uint
	Send(msg []byte)
}

type ControlUser interface {
	Connection
	Tag() string
	T2M(oneId uint, k []byte, part *json.RawMessage)
	RoomOnes() ([]account.One, error)
	GetOauth() *account.Oauth
}

type ControlRoom interface {
	Connection
	Tag() string
	Broadcast(msg []byte)
	BroadcastT2M(k []byte, part json.RawMessage)
	Friends() ([]account.Account, error)
	AddOnline(id uint, cu ControlUser, tag string)
	GetOnline(id uint) (ControlUser, bool)
	RemoveOnline(id uint)
	GetOne() *account.One
	Remove()
}

type Hub interface {
	Run()
	GetRoom(id uint) (ControlRoom, bool)

	OnReg(room ControlRoom)
	OnUnreg(room ControlRoom)
	OnCmd(cmd *Command)
	OnMsg(msg *Message)
	OnJoin(many ControlUser)
	OnLeave(many ControlUser)

	WaitForProcess(reciever string) (chan *websocket.Conn, error)
	ProcessFromWait(reciever string) (chan *websocket.Conn, error)
	NewInviteCode(room uint) string
	ValidateInviteCode(room uint, code string) bool
}

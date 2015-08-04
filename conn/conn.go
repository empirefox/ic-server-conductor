package conn

import (
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
	SendIpcams()
	RoomOnes() ([]account.One, error)
	GetOauth() *account.Oauth
}

type ControlRoom interface {
	Connection
	Broadcast(msg []byte)
	Ipcams() Ipcams
	Friends() ([]account.Account, error)
	AddOnline(id uint, cu ControlUser)
	GetOnline(id uint) (ControlUser, bool)
	RemoveOnline(id uint)
	GetOne() *account.One
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

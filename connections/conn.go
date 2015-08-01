package connections

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

var (
	ErrInvalidToken = errors.New("Token is not valid")
)

// Wrap websocket.Conn to interface
// Useful for test
type Connection interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

func AuthWs(conn Connection, secret interface{}) (*jwt.Token, error) {
	_, p, err := conn.ReadMessage()
	if err != nil {
		glog.Infoln("Read message err:", err)
		return nil, err
	}

	token, err := jwt.Parse(string(p), func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		glog.Infoln("Parse token:", err)
		return nil, err
	}
	if !token.Valid {
		glog.Infoln("Token is not valid")
		return nil, ErrInvalidToken
	}
	return token, error
}

package conn

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

const (
	UserKey = "user"
)

var (
	ErrInvalidToken = errors.New("Token is not valid")
)

func AuthWs(ws Ws, secret interface{}) (*jwt.Token, error) {
	_, p, err := ws.ReadMessage()
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
	return token, nil
}

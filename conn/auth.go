package conn

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/golang/glog"
)

var (
	ErrBadToken     = errors.New("Token should include oauth")
	ErrBadOauth     = errors.New("Token should include string oauth")
	ErrInvalidToken = errors.New("Token is not valid")
)

type VerifyFunc func(o *account.Oauth, token []byte) error

// Deprecated
func AuthWs(ws Ws, kf jwt.Keyfunc) (*jwt.Token, error) {
	_, p, err := ws.ReadMessage()
	if err != nil {
		glog.Infoln("Read message err:", err)
		return nil, err
	}

	token, err := jwt.Parse(string(p), kf)
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

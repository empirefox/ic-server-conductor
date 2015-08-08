package conn

import (
	"encoding/json"
	"errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/golang/glog"
)

var (
	UserKey = "user"
)

var (
	ErrBadToken     = errors.New("Token should include oauth")
	ErrBadOauth     = errors.New("Token should include string oauth")
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

func GetTokenOauth(token *jwt.Token, o *account.Oauth) error {
	oi, ok := token.Claims[UserKey]
	if !ok {
		return ErrBadToken
	}
	oa, ok := oi.(string)
	if !ok {
		return ErrBadOauth
	}
	return json.Unmarshal([]byte(oa), o)
}

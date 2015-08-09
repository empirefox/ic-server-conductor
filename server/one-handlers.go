package server

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func (s *Server) WsOneSignaling(c *gin.Context) {
	res, err := s.Hub.ProcessFromWait(c.Params.ByName("reciever"))
	if err != nil {
		glog.Errorln(err)
		return
	}
	ws, err := utils.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()
	res <- ws
	<-res
}

type regRoomData struct {
	Name string `json:"name"`
}

func (s *Server) PostRegRoom(c *gin.Context) {
	var data regRoomData
	if err := c.Bind(&data); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	claims := c.Keys[s.ClaimsKey].(map[string]interface{})
	var o *account.Oauth
	if err := o.FindOauth(claims["provider"].(string), claims["oid"].(string)); err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	one := &account.One{Addr: utils.NewRandom()}
	one.Name = data.Name
	if err := o.Account.RegOne(one); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := one.Find([]byte(one.Addr)); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	token := jwt.New(s.Alg)
	token.Header["kid"] = SK_ONE
	token.Claims["addr"] = one.Addr
	token.Claims["id"] = one.ID
	tokenString, err := token.SignedString(s.Keys[SK_ONE])
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.String(http.StatusOK, tokenString)
}

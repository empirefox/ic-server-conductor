package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/empirefox/gotool/paas"
	. "github.com/empirefox/ic-server-ws-signal/account"
	"github.com/empirefox/ic-server-ws-signal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

var ErrWrongKid = errors.New("Wrong kid")

func CheckIsSystemMode(c *gin.Context) {
	if paas.IsSystemMode() {
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": 1, "content": "system running mode changed"})
	c.Abort()
}

func (s *Server) SaveOauth(c *gin.Context) {
	var op OauthProvider
	if err := c.Bind(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": 1, "content": err})
		return
	}

	if err := op.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": 1, "content": "Cannot save"})
		return
	}

	s.OauthConfig.Providers, s.OauthJson = NewGoauthConf()
	c.JSON(http.StatusOK, gin.H{"error": 0, "content": "No need restart now!"})
}

func (s *Server) Auth(kid string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := jwt.ParseFromRequest(c.Request, func(token *jwt.Token) (interface{}, error) {
			reqKid, ok := token.Header["kid"]
			if !ok {
				return nil, ErrWrongKid
			}
			if req, ok := reqKid.(string); !ok || req != kid {
				return nil, ErrWrongKid
			}
			return s.Keys[kid], nil
		})

		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		c.Set(s.ClaimsKey, token.Claims)
	}
}

func (s *Server) CheckManyUser(c *gin.Context) {
	userBs, ok := c.Keys[s.ClaimsKey][s.UserKey]
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, utils.Err("User not found"))
		return
	}
	user, ok := userBs.(string)
	if !ok {
		c.AbortWithError(http.StatusUnauthorized, utils.Err("User format err"))
		return
	}
	o := &Oauth{}
	if err := json.Unmarshal([]byte(user), o); err != nil {
		glog.Infoln("Unmarshal user err:", err)
		c.AbortWithError(http.StatusUnauthorized, utils.Err("User unmarshal err"))
		return
	}
	c.Set(s.UserKey, o)
}

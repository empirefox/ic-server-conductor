package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/empirefox/gotool/paas"
	. "github.com/empirefox/ic-server-conductor/account"
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

func (s *Server) SecureWs(c *gin.Context) {
	if strings.EqualFold(c.Request.URL.Scheme, "ws") {
		glog.Infoln("insecure:", *c.Request.URL)
		c.Abort()
		return
	}
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

func (s *Server) Verify(o *account.Oauth, token []byte) error {
	return s.goauthConfig.Verify(o, token)
}

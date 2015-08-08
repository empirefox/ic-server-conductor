package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	"github.com/empirefox/ic-server-conductor/account"
)

func (s *Server) GetApiProviders(c *gin.Context) {
	var ops account.OauthProviders
	if err := ops.All(); err != nil {
		glog.Errorln(err)
		c.AbortWithStatus(http.StatusNotImplemented)
		return
	}
	c.JSON(http.StatusOK, ops)
}

func (s *Server) newManyToken(c *gin.Context, ouser interface{}) {
	u, err := json.Marshal(ouser)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	token := jwt.New(s.Alg)
	token.Header["kid"] = SK_MANY
	token.Claims[s.UserKey] = string(u)
	token.Claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
	tokenString, err := token.SignedString(s.Keys[SK_MANY])
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, tokenString)
}

func (s *Server) newOneToken(c *gin.Context, u *account.Oauth) {
	token := jwt.New(s.Alg)
	token.Header["kid"] = SK_ONE
	token.Claims["provider"] = u.Provider
	token.Claims["oid"] = u.Oid
	token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	tokenString, err := token.SignedString(s.Keys[SK_ONE])
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, tokenString)
}

func (s *Server) GetLogin(c *gin.Context) {
	claims := c.Keys[s.ClaimsKey].(map[string]interface{})
	ouser := &account.Oauth{}
	if err := ouser.OnOid(claims["provider"].(string), claims["oid"].(string)); err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if target, ok := claims["target"]; ok {
		if tar, ok := target.(string); ok {
			switch tar {
			case "one":
				s.newOneToken(c, ouser)
			case "many":
				s.newManyToken(c, ouser)
			default:
				c.AbortWithStatus(http.StatusBadRequest)
			}
			return
		}
	}
	s.newManyToken(c, ouser)
}

package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/empirefox/gotool/paas"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/proxy"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/itsjamie/gin-cors"
)

var ErrWrongKid = errors.New("Wrong kid")

func CheckIsSystemMode(c *gin.Context) {
	if paas.IsSystemMode() {
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": 1, "content": "system running mode changed"})
	c.Abort()
}

func (s *Server) Cors(method string) gin.HandlerFunc {
	return cors.Middleware(cors.Config{
		Origins:         s.Origins,
		Methods:         method,
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          48 * time.Hour,
		Credentials:     false,
		ValidateHeaders: false,
	})
}

func (s *Server) SecureWs(c *gin.Context) {
	if strings.EqualFold(c.Request.URL.Scheme, "ws") {
		glog.Infoln("insecure:", *c.Request.URL)
		c.Abort()
		return
	}
}

func (s *Server) PostNewToken(c *gin.Context) {
	tokenObj, err := s.goauthConfig.NewToken(c.Keys[s.UserKey].(*account.Oauth))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, tokenObj)
}

func (s *Server) PostProxyToken(c *gin.Context) {
	var data proxy.PostProxyTokenData
	if err := c.BindJSON(&data); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	reqToken, err := jwt.Parse(data.Token, s.goauthConfig.FindVerifyKey)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	c.Set(s.ClaimsKey, reqToken.Claims)
	s.goauthConfig.BindUser(c)
	if _, ok := c.Get("invalide-user"); ok {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	tokenObj, err := s.goauthConfig.HandleUserInfo(c, &data.Info)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, tokenObj)
}

func (s *Server) findProxyProvider(name string) (*proxy.Provider, bool) {
	for k, v := range s.goauthConfig.Providers {
		if v.Name == name {
			return &proxy.Provider{
				Name:         name,
				Path:         k,
				ClientID:     v.ClientID,
				ClientSecret: v.ClientSecret,
			}, true
		}
	}
	return nil, false
}

func (s *Server) GetProxiedProviders(c *gin.Context) {
	var ps []proxy.Provider
	for _, name := range s.Proxied {
		if p, ok := s.findProxyProvider(name); ok {
			ps = append(ps, *p)
		}
	}
	c.JSON(http.StatusOK, ps)
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

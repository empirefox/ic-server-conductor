package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	"github.com/empirefox/gin-oauth2"
	"github.com/empirefox/gotool/paas"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/conn/many"
	"github.com/empirefox/ic-server-conductor/conn/one"
	"github.com/empirefox/ic-server-conductor/invite"
)

const (
	SK_SYS = "system"
)

type Server struct {
	Origins         string
	ClaimsKey       string
	UserKey         string
	OneAlg          string
	Keys            map[string][]byte
	Hub             conn.Hub
	IsDevMode       bool
	OnEngineCreated func(*gin.Engine)
	OauthGroupName  string
	goauthConfig    *goauth.Config
}

func (s *Server) Ok(c *gin.Context)       { c.AbortWithStatus(http.StatusOK) }
func (s *Server) NotFound(c *gin.Context) { c.AbortWithStatus(http.StatusNotFound) }

func (s *Server) Run() error {
	corsMiddleWare := s.Cors("GET, PUT, POST, DELETE")

	s.goauthConfig = &goauth.Config{
		Origin:      strings.TrimSpace(strings.Split(s.Origins, ",")[0]),
		NewUserFunc: func() goauth.OauthUser { return &account.Oauth{} },
	}
	if err := account.AddGoauthProviders(s.goauthConfig, s.OauthGroupName); err != nil {
		glog.Errorln(err)
	}
	authMiddleWare := goauth.Middleware(s.goauthConfig)

	router := gin.Default()
	if s.OnEngineCreated != nil {
		s.OnEngineCreated(router)
	}

	router.Use(secure.Secure(secure.Options{
		SSLRedirect: true,
		SSLProxyHeaders: map[string]string{
			"X-Forwarded-Proto": "https",
			"Upgrade":           "websocket",
		},
		IsDevelopment: s.IsDevMode,
	}))

	router.Use(s.SecureWs)

	// peer from MANY client
	router.GET("/sys-data.js", s.GetSystemData)
	router.GET("/oauth/oauths", corsMiddleWare, s.GetOauths)
	router.OPTIONS("/oauth/oauths", corsMiddleWare, s.Ok)

	sys := router.Group("/sys", s.Auth(SK_SYS))
	sys.POST("/clear-tables", s.PostClearTables)
	sys.POST("/create-tables", s.PostCreateTables)
	sys.POST("/oauth", s.PostSaveOauth)

	// peer from ONE client
	ro := router.Group("/one")
	ro.GET("/ctrl", one.HandleOneCtrl(s.Hub, s.OneAlg, s.Verify))
	ro.GET("/signaling/:reciever", s.WsOneSignaling)

	// websocket
	// peer from MANY client
	manyws := router.Group("/mws")
	manyws.GET("/ctrl", many.HandleManyCtrl(s.Hub, s.Verify))
	manyws.GET("/signaling", s.WsManySignaling)

	// many rest
	rm := router.Group("/many", corsMiddleWare, authMiddleWare, s.goauthConfig.MustBindUser)
	rm.OPTIONS("/unlink", s.Ok)
	rm.DELETE("/unlink", s.goauthConfig.Unlink)
	rm.OPTIONS("/logoff", s.Ok)
	rm.DELETE("/logoff", s.goauthConfig.Logoff)
	rm.OPTIONS("/new-token", s.Ok)
	rm.POST("/new-token", s.PostNewToken)
	rm.OPTIONS("/myproviders", s.Ok)
	rm.GET("/myproviders", s.GetAccountProviders)
	rm.OPTIONS("/invite-code", s.Ok)
	rm.POST("/invite-code", invite.HandleManyGetInviteCode(s.Hub, s.UserKey))
	rm.OPTIONS("/invite-join", s.Ok)
	rm.POST("/invite-join", invite.HandleManyOnInvite(s.Hub, s.UserKey))

	// many and one login rest api
	// compatible with Satellizer
	for path := range s.goauthConfig.Providers {
		router.POST(path, corsMiddleWare, authMiddleWare, s.Ok)
		router.OPTIONS(path, corsMiddleWare, s.Ok)
	}

	return router.Run(paas.BindAddr)
}

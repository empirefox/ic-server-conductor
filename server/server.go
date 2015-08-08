package server

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/itsjamie/gin-cors"

	"github.com/empirefox/gotool/dp"
	"github.com/empirefox/gotool/paas"
	"github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/conn/many"
	"github.com/empirefox/ic-server-conductor/conn/one"
	"github.com/empirefox/ic-server-conductor/invite"
)

const (
	SK_CALL = "call"
	SK_MANY = "many"
	SK_ONE  = "one"
	SK_SYS  = "system"
)

type Server struct {
	ClaimsKey       string
	UserKey         string
	Alg             jwt.SigningMethod
	Keys            map[string][]byte
	Hub             conn.Hub
	OauthJson       []byte
	IsDevMode       bool
	OnEngineCreated func(*gin.Engine)
}

func (s *Server) Run() error {
	dp.SetDevMode(paas.IsDevMode)
	conn.UserKey = s.UserKey
	corsMiddleWare := cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          48 * time.Hour,
		Credentials:     false,
		ValidateHeaders: false,
	})

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
	router.GET("auth/oauths", corsMiddleWare, func(c *gin.Context) { c.Writer.Write(s.OauthJson) })

	sys := router.Group("/sys", s.Auth(SK_SYS))
	sys.POST("/clear-tables", s.PostClearTables)
	sys.POST("/oauth", s.PostSaveOauth)

	call := router.Group("/call", s.Auth(SK_CALL))
	call.GET("/login", s.GetLogin)
	call.GET("/providers", s.GetApiProviders)

	// peer from ONE client
	ro := router.Group("/one")
	ro.GET("/ctrl", one.HandleOneCtrl(s.Hub, s.Keys[SK_ONE]))
	ro.GET("/signaling/:reciever", s.WsOneSignaling)

	// one rest
	ror := router.Group("/one-rest", s.Auth(SK_ONE))
	// remove action will be performed among ctrl conn
	ror.POST("/reg-room", s.PostRegRoom)

	// websocket
	// peer from MANY client
	manyws := router.Group("/mws")
	manyws.GET("/ctrl", many.HandleManyCtrl(s.Hub, s.Keys[SK_MANY]))
	manyws.GET("/signaling", s.WsManySignaling)

	// many rest
	rm := router.Group("/many", corsMiddleWare, s.Auth(SK_MANY), s.CheckManyUser)
	rm.DELETE("/logoff", DeleteManyLogoff)
	rm.GET("/invite-code/:room", invite.HandleManyGetInviteCode(s.Hub))
	rm.GET("/invite/:room/:code", invite.HandleManyOnInvite(s.Hub))
	rm.POST("/refresh-token", func(c *gin.Context) { c.JSON(http.StatusOK, "") })

	return router.Run(paas.GetBindAddr())
}

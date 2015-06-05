package server

import (
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/empirefox/gin-oauth2"
	"github.com/empirefox/gotool/paas"
	. "github.com/empirefox/ic-server-ws-signal/connections"
)

type Server struct {
	Hub         *Hub
	OauthConfig *goauth.Config
	OauthJson   []byte
	IsDevMode   bool
}

func (s *Server) Run() error {
	router := gin.Default()

	router.Use(secure.Secure(secure.Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   s.IsDevMode,
	}))

	// peer from MANY client
	router.Use(static.Serve("/", static.LocalFile("./public", false)))

	// login page will be find in static serve
	// logout will proccess some logic
	router.GET(s.OauthConfig.PathLogout, s.OauthConfig.DefaultLogout)
	router.GET("/auth/oauths", func(c *gin.Context) {
		c.Writer.Write(s.OauthJson)
	})

	// peer from ONE client
	one := router.Group("/one")
	one.GET("/ctrl", HandleOneCtrl(s.Hub))
	one.GET("/signaling/:reciever", HandleOneSignaling(s.Hub))

	// websocket
	// peer from MANY client
	many := router.Group("/many")
	many.Use(goauth.Setup(s.OauthConfig))
	many.GET("/ctrl", HandleManyCtrl(s.Hub))
	many.GET("/signaling/:room/:camera/:reciever", HandleManySignaling(s.Hub))
	many.GET("/checklogin", HandleManyCheckLogin(s.OauthConfig))
	many.POST("/reg-room", s.OauthConfig.Check(goauth.Permitted), HandleManyRegRoom(s.Hub, s.OauthConfig))

	return router.Run(paas.GetBindAddr())
}

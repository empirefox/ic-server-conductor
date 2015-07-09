package server

import (
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/empirefox/gin-oauth2"
	"github.com/empirefox/gotool/dp"
	"github.com/empirefox/gotool/ng"
	"github.com/empirefox/gotool/paas"
	. "github.com/empirefox/ic-server-ws-signal/connections"
	"github.com/empirefox/ic-server-ws-signal/invite"
)

type Server struct {
	Hub          *Hub
	OauthConfig  *goauth.Config
	OauthJson    []byte
	IsDevMode    bool
	ValidateGets map[string]string
}

func (s *Server) Run() error {
	dp.SetDevMode(paas.IsDevMode)
	router := gin.Default()

	router.Use(secure.Secure(secure.Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   s.IsDevMode,
	}))

	// peer from MANY client
	router.Use(static.Serve("/", static.LocalFile("./public", false)))
	router.GET("/ng/sys-data.js", func(c *gin.Context) {
		ng.Write(c.Writer, ng.Module{
			Type:       "constant",
			ModuleName: "app.constants.system",
			Name:       "SystemData",
			Instance: map[string]interface{}{
				"DevProd": dp.Mode,
			},
		})
	})
	for k, v := range s.ValidateGets {
		router.GET(k, func(c *gin.Context) {
			c.Writer.Write([]byte(v))
		})
	}

	router.Use(goauth.Setup(s.OauthConfig))

	// login page will be find in static serve
	// logout will proccess some logic
	router.GET(s.OauthConfig.PathLogout, s.OauthConfig.DefaultLogout)
	router.GET("/auth/oauths", func(c *gin.Context) {
		c.Writer.Write(s.OauthJson)
	})

	sys := router.Group("/sys", CheckIsSystemMode)
	sys.POST("/oauth", SaveOauth)

	// peer from ONE client
	one := router.Group("/one")
	one.GET("/ctrl", HandleOneCtrl(s.Hub))
	one.GET("/signaling/:reciever", HandleOneSignaling(s.Hub))

	// websocket
	// peer from MANY client
	many := router.Group("/many")
	many.GET("/ctrl", s.OauthConfig.Check(goauth.Permitted), HandleManyCtrl(s.Hub))
	many.GET("/signaling/:room/:camera/:reciever", HandleManySignaling(s.Hub))

	// rest
	many.GET("/checklogin", HandleManyCheckLogin(s.OauthConfig))
	many.POST("/reg-room", s.OauthConfig.Check(goauth.Permitted), HandleManyRegRoom(s.Hub, s.OauthConfig))
	many.GET("/invite-code/:room", s.OauthConfig.Check(goauth.Permitted), invite.HandleManyGetInviteCode(s.Hub, s.OauthConfig))
	many.GET("/invite/:room/:code", s.OauthConfig.Check(goauth.Permitted), invite.HandleManyOnInvite(s.Hub, s.OauthConfig))

	return router.Run(paas.GetBindAddr())
}

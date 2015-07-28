package server

import (
	"strings"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	"github.com/empirefox/gin-oauth2"
	"github.com/empirefox/gotool/dp"
	"github.com/empirefox/gotool/ng"
	"github.com/empirefox/gotool/paas"
	. "github.com/empirefox/ic-server-ws-signal/connections"
	"github.com/empirefox/ic-server-ws-signal/invite"
	"github.com/empirefox/ic-server-ws-signal/utils"
)

type Server struct {
	Hub          *Hub
	OauthConfig  *goauth.Config
	OauthJson    []byte
	IsDevMode    bool
	ValidateGets map[string]string
}

func (s *Server) SecureWs(c *gin.Context) {
	if strings.EqualFold(c.Request.URL.Scheme, "ws") {
		glog.Infoln("insecure:", *c.Request.URL)
		c.Abort()
		return
	}
}

func (s *Server) Run() error {
	dp.SetDevMode(paas.IsDevMode)
	router := gin.Default()

	router.Use(PlainGets(s.ValidateGets))
	router.Use(goauth.Setup(s.OauthConfig))

	router.Use(secure.Secure(secure.Options{
		SSLRedirect: true,
		SSLProxyHeaders: map[string]string{
			"X-Forwarded-Proto": "https",
			"Upgrade":           "websocket",
		},
		IsDevelopment: s.IsDevMode,
	}))

	router.Use(s.SecureWs)

	router.Use(static.Serve("/", static.LocalFile(utils.GetStaticDir("public"), true)))
	// peer from MANY client
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
	many.GET("/logoff", s.OauthConfig.Check(goauth.Permitted), HandleManyLogoff(s.Hub, s.OauthConfig))
	many.POST("/reg-room", s.OauthConfig.Check(goauth.Permitted), HandleManyRegRoom(s.Hub, s.OauthConfig))
	many.GET("/invite-code/:room", s.OauthConfig.Check(goauth.Permitted), invite.HandleManyGetInviteCode(s.Hub, s.OauthConfig))
	many.GET("/invite/:room/:code", s.OauthConfig.Check(goauth.Permitted),
		invite.HandleManyOnInvite(s.Hub, s.OauthConfig, "/many/invite"))

	return router.Run(paas.GetBindAddr())
}

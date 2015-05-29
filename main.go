package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/empirefox/gin-oauth2"
	"github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/connections"
	"github.com/empirefox/ic-server-ws-signal/utils"
)

var (
	// set in cmd flag
	addr          = flag.String("addr", fmt.Sprintf(":%v", utils.GetEnv("PORT", "8080")), "http service address")
	names         []string
	isDevelopment = !utils.IsProduction()
)

func init() {
	if isDevelopment {
		flag.Set("stderrthreshold", "INFO")
	}
	flag.Parse()
	if isDevelopment {
		*addr = "0.0.0.0:9999"
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	h := NewHub()
	go h.Run()

	conf, oauthBs := account.NewGoauthConf()

	router := gin.Default()

	router.Use(secure.Secure(secure.Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   isDevelopment,
	}))

	// peer from MANY client
	router.Use(static.Serve("/", static.LocalFile("./public", false)))

	// login page will be find in static serve
	// logout will proccess some logic
	router.GET(conf.PathLogout, conf.DefaultLogout)
	router.GET("/auth/oauths", func(c *gin.Context) {
		c.Writer.Write(oauthBs)
	})

	// peer from ONE client
	one := router.Group("/one")
	one.GET("/ctrl", HandleOneCtrl(h))
	one.GET("/signaling/:reciever", HandleOneSignaling(h))

	// websocket
	// peer from MANY client
	many := router.Group("/many")
	many.Use(goauth.Setup(conf))
	many.GET("/ctrl", HandleManyCtrl(h))
	many.GET("/signaling/:room/:camera/:reciever", HandleManySignaling(h))

	router.Run(*addr)
}

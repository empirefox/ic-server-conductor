package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
)

var (
	// set in cmd flag
	addr          = flag.String("addr", fmt.Sprintf(":%v", getEnv("PORT", "8080")), "http service address")
	names         []string
	isDevelopment = !isProduction()
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

	h := New()
	go h.Run()

	router := gin.Default()

	router.Use(secure.Secure(secure.Options{
		SSLRedirect:     true,
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:   isDevelopment,
	}))

	// html
	// peer from MANY client
	router.Use(static.Serve("/", static.LocalFile("./public", false)))

	//	router.GET("/auth/login", Login)
	//	router.GET("/auth/logout", Logout)

	// websocket
	// peer from ONE client
	one := router.Group("/one")
	one.GET("/ctrl", fakeOneLogin(), handleWs(h, oneControlling))
	one.GET("/signaling/:reciever", oneSignaling(h))

	// websocket
	// peer from MANY client
	many := router.Group("/many", fakeManyLogin())
	many.GET("/ctrl", handleWs(h, manyControlling))
	many.GET("/signaling/:room/:camera/:reciever", manySignaling(h))

	router.Run(*addr)
}

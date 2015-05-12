package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
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
		*addr = "0.0.0.0:9998"
	}
}

type NewSignalingConnParams struct {
	Room     int64  `json:"room"`
	Camera   string `json:"camera"`
	Reciever string `json:"reciever"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	h := New()

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
	one.GET("/ctrl", handleWs(h, oneControlling))
	one.GET("/signaling/:reciever", func(c *gin.Context) {
		reciever := c.Params.ByName("reciever")
		res, ok := h.sigResWaitMap[reciever]
		if !ok {
			glog.Errorln("No reciever here")
			return
		}
		delete(h.sigResWaitMap, reciever)
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		res <- ws
		<-res
	})

	// websocket
	// peer from MANY client
	many := router.Group("/many")
	many.POST("/ctrl", handleWs(h, manyControlling))
	many.POST("/signaling", func(c *gin.Context) {
		var params NewSignalingConnParams
		if !c.Bind(&params) {
			glog.Errorln("Cannot bind params")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot bind params"})
			return
		}
		res := make(chan *websocket.Conn)
		h.sigResWaitMap[params.Reciever] = res
		h.sigReq <- &params
		select {
		case resWs := <-res:
			ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				glog.Errorln(err)
				return
			}
			defer ws.Close()
			Pipe(ws, resWs)
			res <- nil
		case <-time.After(time.Second * 15):
			delete(h.sigResWaitMap, params.Reciever)
			return
		}
	})

	router.Run(*addr)
}

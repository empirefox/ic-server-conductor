package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
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
	one.GET("/signaling/:reciever", func(c *gin.Context) {
		glog.Infoln("one signaling coming")
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
	many := router.Group("/many", fakeManyLogin())
	many.GET("/ctrl", handleWs(h, manyControlling))
	many.GET("/signaling/:room/:camera/:reciever", func(c *gin.Context) {
		glog.Infoln("many signaling coming")
		roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no room set"})
			return
		}
		room, ok := h.rooms[roomId]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Room not found in request"})
			return
		}
		cameras := room.Cameras
		if cameras == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cameras not found in room"})
			return
		}
		cmd := CreateSignalingConnectionCommand{
			Name:     "CreateSignalingConnection",
			Camera:   c.Params.ByName("camera"),
			Reciever: c.Params.ByName("reciever"),
		}
		_, ok = cameras[cmd.Camera]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Camera not found in room"})
			return
		}
		cmdStr, err := json.Marshal(cmd)
		if err != nil {
			glog.Errorln(err)
			return
		}
		res := make(chan *websocket.Conn)
		h.sigResWaitMap[cmd.Reciever] = res
		room.SendCtrlToOne <- cmdStr
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
			delete(h.sigResWaitMap, cmd.Reciever)
			return
		}
	})

	router.Run(*addr)
}

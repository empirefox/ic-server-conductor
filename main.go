package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

var (
	paasVendors = map[string]map[string]bool{
		"PAAS_VENDOR": {
			"cloudControl": true,
		},
	}

	upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
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
	flag.Set("stderrthreshold", "INFO")
	flag.Parse()
	if isDevelopment {
		*addr = "0.0.0.0:9999"
	}
}

func isProduction() bool {
	for envName, values := range paasVendors {
		if actual := os.Getenv(envName); values[actual] {
			return true
		}
	}
	return false
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 30 * time.Second
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
	var srcMutex sync.Mutex
	router.GET("/one", func(c *gin.Context) {
		defer srcMutex.Unlock()
		srcMutex.Lock()
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		serveSrc(ws)
	})

	// websocket
	// peer from MANY client
	var destMutex sync.Mutex
	router.GET("/many", func(c *gin.Context) {
		if !hasSrc {
			return
		}
		defer destMutex.Unlock()
		destMutex.Lock()
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		serveDest(ws)
	})

	router.Run(*addr)
}

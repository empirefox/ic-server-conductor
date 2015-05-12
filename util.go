package main

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 30 * time.Second
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

	dailer = websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
)

type WsHandler func(ws *websocket.Conn, c *gin.Context, h *Hub)

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

func handleWs(h *Hub, handler WsHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			glog.Errorln(err)
			return
		}
		defer ws.Close()
		handler(ws, c, h)
	}
}

func writing(ws *websocket.Conn, send chan []byte) {
	defer func() {
		ws.Close()
	}()
	for {
		select {
		case msg, ok := <-send:
			if !ok {
				ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		}
	}
}

func writingWithPing(ws *websocket.Conn, send chan []byte) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		ws.Close()
	}()
	for {
		select {
		case msg, ok := <-send:
			if !ok {
				ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				return
			}
		case <-ticker.C:
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

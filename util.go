package main

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"time"

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

func GetTypedMsg(t string, m interface{}) ([]byte, error) {
	msg := map[string]interface{}{"type": t, "content": m}
	return json.Marshal(msg)
}

package utils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/uniuri"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	WriteWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	PingPeriod = 30 * time.Second
)

var (
	Upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}

	Dailer = websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
)

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

func GetStaticDir(dir string) string {
	pwd, err := os.Getwd()
	if err != nil {
		glog.Errorln(err)
		return dir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(pwd, dir)
	}
	return dir
}

func GetTypedMsg(t string, m interface{}) ([]byte, error) {
	msg := map[string]interface{}{"type": t, "content": m}
	return json.Marshal(msg)
}

func GetTypedMsgStr(t, m string) []byte {
	return []byte(fmt.Sprintf(`{"type":"%s","content":"%s"}`, t, m))
}

func GetTypedInfo(info string) []byte {
	return GetTypedMsgStr("Info", info)
}

func GetNamedCmd(from uint, name, cmd string) []byte {
	return []byte(fmt.Sprintf(`{"from":%d,"name":"%s","content":"%s"}`, from, name, cmd))
}

func NewUUID() string {
	return uniuri.NewLen(36)
}

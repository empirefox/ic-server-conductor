package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"github.com/empirefox/gotool/paas"
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
	Origin string

	Upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			// https://xxx.xxx.com
			origin := r.Header["Origin"]
			if len(origin) == 0 || origin[0] == Origin {
				return true
			}
			u, err := url.Parse(origin[0])
			if err != nil {
				return false
			}
			o := strings.Split(u.Host, ":")[0]
			return o == paas.Info.HttpDomain || o == strings.Split(r.Host, ":")[0]
		},
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

var ErrNoSegContent = errors.New("No seg(:) found in bs")

func ReadO2MSeg(bs []byte) (uint, []byte, []byte, error) {
	raws := bytes.SplitN(bs, []byte{':'}, 3)
	if len(raws) < 3 {
		return 0, nil, nil, ErrNoSegContent
	}
	i, err := strconv.Atoi(string(raws[1]))
	if err != nil {
		return 0, nil, nil, err
	}
	return uint(i), raws[0], raws[2], nil
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

func GetNamedCmd(from uint, name, cmd []byte) []byte {
	return []byte(fmt.Sprintf(`{"from":%d,"name":"%s","content":%s}`, from, name, cmd))
}

func NewRandom() string {
	return uniuri.NewLen(36)
}

func OK(m string) []byte    { return []byte(fmt.Sprintf(`{"content":"%s"}`, m)) }
func Err(err string) []byte { return []byte(fmt.Sprintf(`{"error":1,"content":"%s"}`, err)) }

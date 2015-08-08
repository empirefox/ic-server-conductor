package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/empirefox/gotool/dp"
	"github.com/empirefox/gotool/paas"
	gws "github.com/empirefox/gotool/ws"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/conn/many"
	"github.com/empirefox/ic-server-conductor/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

func (s *Server) GetSystemData(c *gin.Context) {
	e := `sys-data`
	c.Header("Etag", e)
	c.Header("Cache-Control", "max-age=2592000") // 30 days

	if match := c.Request.Header.Get("If-None-Match"); strings.Contains(match, e) {
		c.Writer.WriteHeader(http.StatusNotModified)
		return
	}

	data, _ := json.Marshal(gin.H{
		"DevProd":   dp.Mode,
		"ApiDomain": paas.SubDomain,
	})
	c.String(http.StatusOK, fmt.Sprintf(`var ApiData=%s`, data))
}

type StartSignalingInfo struct {
	Room     uint   `json:"room"`
	Camera   string `json:"camera"`
	Reciever string `json:"reciever"`
}

// many signaling
func (s *Server) WsManySignaling(c *gin.Context) {
	ws, err := utils.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		glog.Infoln("Upgrade failed:", err)
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	defer ws.Close()
	o, err := many.AuthMws(ws, s.Keys[SK_MANY])
	if err != nil {
		return
	}

	_, startInfo, err := ws.ReadMessage()
	if err != nil {
		glog.Infoln("Read start info err:", err)
		return
	}

	var info StartSignalingInfo
	if err := json.Unmarshal(startInfo, &info); err != nil {
		glog.Infoln("Unmarshal info err:", err)
		return
	}

	res := preProccessSignaling(s.Hub, &info, o)
	if res == nil {
		return
	}
	var resWs *websocket.Conn
	select {
	case resWs = <-res:
	case <-time.After(time.Second * 15):
		s.Hub.ProcessFromWait(info.Reciever)
		glog.Infoln("Wait for one signaling timeout")
		return
	}
	gws.Pipe(ws, resWs)
	res <- nil
}

func preProccessSignaling(h conn.Hub, info *StartSignalingInfo, o *account.Oauth) chan *websocket.Conn {
	room, ok := h.GetRoom(info.Room)
	if !ok {
		glog.Infoln("Room not found in request")
		return nil
	}
	if !o.CanView(room.GetOne()) {
		glog.Infoln("Not permited to view this room")
		return nil
	}
	cameras := room.Ipcams()
	if cameras == nil {
		glog.Infoln("Cameras not found in room")
		return nil
	}
	_, ok = cameras[info.Camera]
	if !ok {
		glog.Infoln("Camera not found in room")
		return nil
	}
	res, err := h.WaitForProcess(info.Reciever)
	if err != nil {
		glog.Infoln("Wait for process:", err)
		return nil
	}
	cmd := fmt.Sprintf(`{
		"name":"CreateSignalingConnection",
		"from":%d,
		"content":{
			"camera":"%s", "reciever":"%s"
		}
	}`, info.Room, info.Camera, info.Reciever)
	room.Send([]byte(cmd))
	return res
}

func (s *Server) GetCheckToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.Keys[s.ClaimsKey].(map[string]interface{})
		exp := claims["exp"].(int64)
		update := time.Now().Add(time.Minute * 30).Unix()
		if exp > update {
			c.JSON(http.StatusOK, "{}")
			return
		}
		s.newManyToken(c, claims[s.UserKey])
	}
}

func DeleteManyLogoff(c *gin.Context) {
	iuser, ok := c.Get(conn.UserKey)
	if !ok {
		c.JSON(http.StatusForbidden, `{"error":1,"content":"user not authed"}`)
		return
	}
	if err := iuser.(*account.Oauth).Account.Logoff(); err != nil {
		c.JSON(http.StatusInternalServerError, `{"error":1,"content":"cannot del user"}`)
		return
	}
	c.JSON(http.StatusOK, `{"error":0,"content":"log off ok"}`)
}

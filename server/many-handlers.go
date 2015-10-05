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
	err = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"Accepted"}`))
	if err != nil {
		s.Hub.ProcessFromWait(info.Reciever)
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
		b1, _ := json.MarshalIndent(o, "", "\t")
		b2, _ := json.MarshalIndent(room.GetOne(), "", "\t")
		glog.Infoln(string(b1))
		glog.Infoln(string(b2))
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

func (s *Server) PostNewToken(c *gin.Context) {
	tokenObj, err := s.goauthConfig.NewToken(c.Keys[config.GinUserKey])
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, tokenObj)
}

// Deprecated
func (s *Server) PostOauthLogin(c *gin.Context) {
	o := &account.Oauth{}
	if err := c.BindJSON(o); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := c.Keys[s.UserKey].(*account.Oauth).Associate(o); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.AbortWithStatus(http.StatusOK)
}

// Deprecated
func (s *Server) DeleteOauthUnlink(c *gin.Context) {
	var obj map[string]interface{}
	if c.BindJSON(&obj) != nil {
		return
	}
	if prd, ok := obj["provider"].(string); !ok {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := c.Keys[s.UserKey].(*account.Oauth).UnAssociate(prd); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) GetAccountProviders(c *gin.Context) {
	o := c.Keys[s.UserKey].(*account.Oauth)
	ps := []string{}
	if err := o.GetProviders(&ps); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"current": o.Provider, "providers": ps})
}

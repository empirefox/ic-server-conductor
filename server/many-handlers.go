package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/empirefox/gotool/paas"
	gws "github.com/empirefox/gotool/ws"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

func (s *Server) GetOauths(c *gin.Context) {
	if data, err := account.SatellizerProviders(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	} else {
		c.Data(http.StatusOK, "application/json; charset=utf-8", data)
	}
}

func (s *Server) GetSystemData(c *gin.Context) {
	e := `sys-data`
	c.Header("Etag", e)
	c.Header("Cache-Control", "max-age=2592000") // 30 days

	if match := c.Request.Header.Get("If-None-Match"); strings.Contains(match, e) {
		c.Writer.WriteHeader(http.StatusNotModified)
		return
	}

	data, _ := json.Marshal(paas.Info)
	c.String(http.StatusOK, fmt.Sprintf(`var ApiData=%s`, data))
}

type StartSignalingInfo struct {
	Room     uint   `json:"room"`
	Reciever string `json:"reciever"`
	Token    string `json:"token"`
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
	_, startInfo, err := ws.ReadMessage()
	if err != nil {
		glog.Infoln("Read start info err:", err)
		return
	}

	var info StartSignalingInfo
	if err = json.Unmarshal(startInfo, &info); err != nil {
		glog.Infoln("Unmarshal info err:", err)
		return
	}

	o := &account.Oauth{}
	if err = s.Verify(o, []byte(info.Token)); err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"AuthFailed"}`))
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
	res, err := h.WaitForProcess(info.Reciever)
	if err != nil {
		glog.Infoln("Wait for process:", err)
		return nil
	}
	cmd := fmt.Sprintf(`{
		"name":"CreateSignalingConnection",
		"from":%d,
		"content":"%s"
	}`, o.AccountId, info.Reciever)
	room.Send([]byte(cmd))
	return res
}

func (s *Server) PostNewToken(c *gin.Context) {
	tokenObj, err := s.goauthConfig.NewToken(c.Keys[s.UserKey].(*account.Oauth))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, tokenObj)
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

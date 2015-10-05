package server

import (
	"github.com/empirefox/ic-server-conductor/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func (s *Server) WsOneSignaling(c *gin.Context) {
	res, err := s.Hub.ProcessFromWait(c.Params.ByName("reciever"))
	if err != nil {
		glog.Errorln(err)
		return
	}
	ws, err := utils.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		glog.Errorln(err)
		return
	}
	defer ws.Close()
	res <- ws
	<-res
}

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func PlainGets(gets map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			return
		}
		if c.Request.URL.Path == "/many/ctrl" {
			glog.Infoln(*c.Request.URL)
			for _, cookie := range c.Request.Cookies() {
				glog.Infoln(*cookie)
			}
		}

		if get, ok := gets[c.Request.URL.Path]; ok {
			c.String(http.StatusOK, get)
			c.Abort()
		}
	}
}

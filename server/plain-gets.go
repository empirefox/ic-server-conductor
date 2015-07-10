package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func PlainGets(gets map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			return
		}

		if get, ok := gets[c.Request.URL.Path]; ok {
			c.String(http.StatusOK, get)
			c.Abort()
		}
	}
}

package server

import (
	"net/http"

	"github.com/empirefox/gotool/paas"
	. "github.com/empirefox/ic-server-ws-signal/account"
	"github.com/gin-gonic/gin"
)

func CheckIsSystemMode(c *gin.Context) {
	if paas.IsSystemMode() {
		return
	}
	c.JSON(http.StatusNotFound, gin.H{"error": 1, "content": "system running mode changed"})
	c.Abort()
}

func SaveOauth(c *gin.Context) {
	var op OauthProvider
	if err := c.Bind(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": 1, "content": err})
		return
	}

	if err := op.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": 1, "content": "Cannot save"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"error": 0, "content": "Need restart server!"})
}

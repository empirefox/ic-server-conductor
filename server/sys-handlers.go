package server

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/empirefox/ic-server-conductor/account"
)

func (s *Server) PostSaveOauth(c *gin.Context) {
	var op account.OauthProvider
	if err := c.Bind(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": 1, "content": err})
		return
	}

	if err := op.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": 1, "content": "Cannot save"})
		return
	}

	s.OauthJson = account.PageOauthsBytes()
	c.JSON(http.StatusOK, gin.H{"error": 0, "content": "No need restart now!"})
}

func (s *Server) PostClearTables(c *gin.Context) {
	allow, _ := strconv.ParseBool(os.Getenv("ALLOW_CLEAR_TABLES"))
	if !allow {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err := account.ClearTables(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, "")
}

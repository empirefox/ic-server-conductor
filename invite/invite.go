package invite

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	. "github.com/empirefox/ic-server-conductor/account"
	. "github.com/empirefox/ic-server-conductor/conn"
)

type getInviteCodeData struct {
	Room uint `json:"room"`
}

func HandleManyGetInviteCode(h Hub, userKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data getInviteCodeData
		if err := c.BindJSON(&data); err != nil {
			glog.Infoln("No room set in context:", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		user := c.Keys[userKey].(*Oauth).Account
		one := &One{}
		if err := one.FindIfOwner(data.Room, user.ID); err != nil {
			glog.Infoln("Not the owner of the room:", err)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"room": data.Room,
			"code": h.NewInviteCode(data.Room),
		})
	}
}

type onInviteData struct {
	Room uint   `json:"room"`
	Code string `json:"code"`
}

func onManyInvite(h Hub, c *gin.Context, userKey string) (ok bool) {
	var data onInviteData
	if err := c.BindJSON(&data); err != nil {
		glog.Infoln("Get on-invite data:", err)
		return

	}
	if !h.ValidateInviteCode(data.Room, data.Code) {
		glog.Infoln("Invalid invite code")
		return
	}
	one := &One{}
	if err := one.Find(data.Room); err != nil {
		glog.Infoln("Room not found:", err)
		return
	}
	user := c.Keys[userKey].(*Oauth).Account
	if one.OwnerId == user.ID {
		glog.Infoln("Cannot invite to your own room")
		return
	}
	if err := user.ViewOne(one); err != nil {
		glog.Infoln("Cannot be invited to the room:", err)
		return
	}
	return true
}

func HandleManyOnInvite(h Hub, userKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !onManyInvite(h, c, userKey) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.AbortWithStatus(http.StatusOK)
	}
}

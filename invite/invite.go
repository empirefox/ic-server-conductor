package invite

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	. "github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/conn"
)

func HandleManyGetInviteCode(h Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
		if err != nil {
			glog.Infoln("No room set in context:", err)
			return
		}
		user := c.Keys[UserKey].(*Oauth).Account
		one := &One{}
		if err := one.FindIfOwner(uint(roomId), user.ID); err != nil {
			glog.Infoln("Not the owner of the room:", err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"room": roomId,
			"code": h.NewInviteCode(uint(roomId)),
		})
	}
}

func onManyInvite(h Hub, c *gin.Context) (ok bool) {
	roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
	if err != nil {
		glog.Infoln("No room set in context:", err)
		return
	}
	room := uint(roomId)
	if !h.ValidateInviteCode(room, c.Params.ByName("code")) {
		glog.Infoln("Invalid invite code:", err)
		return
	}
	one := &One{}
	if err := one.FindIfOwner(room, 0); err != nil {
		glog.Infoln("Room not found:", err)
		return
	}
	user := c.Keys[UserKey].(*Oauth).Account
	if one.OwnerId == user.ID {
		glog.Infoln("Cannot invite to your own room:", err)
		return
	}
	if err := user.ViewOne(one); err != nil {
		glog.Infoln("Cannot be invited to the room:", err)
		return
	}
	return true
}

func HandleManyOnInvite(h Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		if onManyInvite(h, c) {
			c.JSON(http.StatusOK, "")
			return
		}
		c.JSON(http.StatusBadRequest, "")
	}
}

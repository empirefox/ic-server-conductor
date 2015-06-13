package invite

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/empirefox/gin-oauth2"
	. "github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/connections"
)

func HandleManyGetInviteCode(h *Hub, conf *goauth.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
		if err != nil {
			panic("No room set in context")
		}
		user := c.Keys[conf.UserGinKey].(*Oauth).Account
		one := &One{}
		if err := one.FindIfOwner(uint(roomId), user.ID); err != nil {
			panic("Not the owner of the room")
		}
		c.JSON(http.StatusOK, gin.H{
			"room": roomId,
			"code": h.NewInviteCode(uint(roomId)),
		})
	}
}

func HandleManyOnInvite(h *Hub, conf *goauth.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomId, err := strconv.ParseInt(c.Params.ByName("room"), 10, 0)
		if err != nil {
			panic("No room set in context")
		}
		room := uint(roomId)
		if !h.ValidateInviteCode(room, c.Params.ByName("code")) {
			panic("Invalid invite code")
		}
		one := &One{}
		if err := one.FindIfOwner(room, 0); err != nil {
			panic("Room not found")
		}
		user := c.Keys[conf.UserGinKey].(*Oauth).Account
		if one.OwnerId == user.ID {
			panic("Cannot invite to your own room")
		}
		if err := user.ViewOne(one); err != nil {
			panic("Cannot be invited to the room")
		}
		c.JSON(http.StatusOK, "")
	}
}

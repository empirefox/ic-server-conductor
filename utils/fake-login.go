package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	. "github.com/empirefox/ic-server-ws-signal/account"
)

func newFakeAccount() Account {
	a := Account{
		Ones: []One{newFakeOne()},
	}
	a.ID = 20
	a.Name = "admin"
	return a
}

func newFakeOne() One {
	one := One{
		Owner: newFakeAccount(),
	}
	one.ID = 100
	one.Name = "room1"
	return one
}

func FakeOneLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		one := newFakeOne()
		c.Set("one", &one)
		glog.Infoln("Fake one login ok")
	}
}

func FakeManyLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user", &Oauth{
			Account: newFakeAccount(),
		})
		glog.Infoln("Fake many login ok")
	}
}

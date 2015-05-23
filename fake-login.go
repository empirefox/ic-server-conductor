package main

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func newFakeAccount() Account {
	a := Account{
		Ones: []One{newFakeOne()},
	}
	a.ID = 20
	a.Name = "管理员"
	return a
}

func newFakeOne() One {
	one := One{
		Owner: newFakeAccount(),
	}
	one.ID = 100
	one.Name = "监控室1"
	return one
}

func fakeOneLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		one := newFakeOne()
		c.Set(GinKeyOne, &one)
		glog.Infoln("Fake one login ok")
	}
}

func fakeManyLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(GinKeyUser, &Oauth{
			Account: newFakeAccount(),
		})
		glog.Infoln("Fake many login ok")
	}
}

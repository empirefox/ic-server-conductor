package main

import (
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
)

func newFakeAccount() Account {
	return Account{
		Id:   20,
		Name: "管理员",
		Ones: []One{
			{
				Id:   100,
				Name: "监控室1",
			},
		},
	}
}

func fakeOneLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		one := &One{
			Id:    100,
			Name:  "监控室1",
			Owner: newFakeAccount(),
		}
		c.Set(GinKeyOne, one)
		glog.Infoln("Fake one login ok")
	}
}

func fakeManyLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		a := newFakeAccount()
		c.Set(GinKeyUser, &a)
		glog.Infoln("Fake many login ok")
	}
}

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type BaseModel struct {
	Name        string `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Description string `json:",omitempty"                    sql:"type:varchar(128);default:''"`
}

//Provider:Google,Github,Qq,Weibo,Baidu,Souhu,Netease,Douban
type Oauth struct {
	gorm.Model
	Account   Account `json:",omitempty"`
	AccountId uint    `json:"-"                             sql:"not null"`
	Oid       string  `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Provider  string  `json:",omitempty" binding:"required" sql:"type:varchar(32);not null"`
	Validated bool    `json:",omitempty"                    sql:"default:false"`
	Enabled   bool    `json:",omitempty"                    sql:"default:false"`
}

func (o *Oauth) OnOid(provider, oid string) error {
	return DB.Where(Oauth{Provider: provider, Oid: oid}).
		Attrs(Oauth{Account: Account{}}).
		Preload("Account").Preload("Account.Ones").
		FirstOrCreate(o).Error
}

func (o Oauth) Permitted(c *gin.Context) bool {
	return o.Validated
}

func (o Oauth) Valid() bool {
	return o.Enabled && o.Account.Enabled
}

type Account struct {
	gorm.Model
	BaseModel
	Oauths  []Oauth `json:",omitempty"`
	Ones    []One   `json:",omitempty"`
	Enabled bool    `json:",omitempty" sql:"default:false"`
}

type One struct {
	gorm.Model
	BaseModel
	SecretAddress string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(128);unique"`
	Enabled       bool      `json:",omitempty"                    sql:"default:true"`
	Owner         Account   `json:",omitempty"`
	OwnerId       uint      `json:"-"                             sql:"not null"`
	Accounts      []Account `json:",omitempty"                    gorm:"many2many:account_ones;"`
}

func FindOne(addrb []byte) (*One, error) {
	var one One
	err := DB.Where(One{SecretAddress: string(addrb)}).
		Preload("Accounts").Preload("Owner").First(&one).Error
	return &one, err
}

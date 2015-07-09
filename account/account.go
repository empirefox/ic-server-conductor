package account

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type BaseModel struct {
	Name        string `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Description string `json:",omitempty"                    sql:"type:varchar(128);default:''"`
}

/////////////////////////////////////////
//                Oauth
/////////////////////////////////////////

//Provider:Google,Github,Qq,Weibo,Baidu,Souhu,Netease,Douban
type Oauth struct {
	gorm.Model
	Account   Account `json:",omitempty"`
	AccountId uint    `json:"-"`
	Oid       string  `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Provider  string  `json:",omitempty" binding:"required" sql:"type:varchar(32);not null"`
	Validated bool    `json:",omitempty"                    sql:"default:true"`
	Enabled   bool    `json:",omitempty"                    sql:"default:true"`
}

// Find Oauth, preload Account and Account.Ones
// Create when not found
// Account will be created when a user do first login
func (o *Oauth) OnOid(provider, oid string) error { return aservice.OnOid(o, provider, oid) }
func (o *Oauth) Permitted(c *gin.Context) bool    { return aservice.Permitted(o, c) }
func (o *Oauth) Valid() bool                      { return aservice.Valid(o) }

/////////////////////////////////////////
//                Account
/////////////////////////////////////////

type Account struct {
	gorm.Model
	BaseModel
	Oauths  []Oauth `json:",omitempty"`
	Ones    []One   `json:",omitempty" gorm:"many2many:account_ones;"`
	Enabled bool    `json:",omitempty" sql:"default:true"`
}

// one must be non-exist record
// a   must be from Oauth.OnOid
func (a *Account) RegOne(o *One) error { return aservice.RegOne(a, o) }

// one must be exist record
// a   must be from Oauth.OnOid
func (a *Account) ViewOne(o *One) error { return aservice.ViewOne(a, o) }

// one must be exist record
// a   must be from Oauth.OnOid
func (a *Account) RemoveOne(o *One) error { return aservice.RemoveOne(a, o) }

/////////////////////////////////////////
//                One
/////////////////////////////////////////

// One must be added under an exist Account
type One struct {
	gorm.Model
	BaseModel
	SecretAddress string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(128);unique"`
	Enabled       bool      `json:",omitempty"                    sql:"default:true"`
	Owner         Account   `json:",omitempty"`
	OwnerId       uint      `json:",omitempty"`
	Accounts      []Account `json:",omitempty"                    gorm:"many2many:account_ones;"`
}

func (o *One) Find(addr []byte) error             { return aservice.FindOne(o, addr) }
func (o *One) FindIfOwner(id, ownerId uint) error { return aservice.FindOneIfOwner(o, id, ownerId) }
func (o *One) Save() error                        { return aservice.Save(o) }

/////////////////////////////////////////
//              AccountOne
/////////////////////////////////////////

// Used for adding cascade
type AccountOne struct {
	ID        uint `gorm:"primary_key"`
	AccountId uint
	OneId     uint
}

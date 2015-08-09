package account

import (
	"time"

	"github.com/gin-gonic/gin"
)

type BaseModel struct {
	ID          uint `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Description string `json:",omitempty"                    sql:"type:varchar(128);default:''"`
}

/////////////////////////////////////////
//                Oauth
/////////////////////////////////////////

//Provider:Google,Github,Qq,Weibo,Baidu,Souhu,Netease,Douban
type Oauth struct {
	BaseModel
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
func (o *Oauth) OnOid(provider, oid string) error     { return aservice.OnOid(o, provider, oid) }
func (o *Oauth) FindOauth(provider, oid string) error { return aservice.FindOauth(o, provider, oid) }
func (o *Oauth) Permitted(c *gin.Context) bool        { return aservice.Permitted(o, c) }
func (o *Oauth) Valid() bool                          { return aservice.Valid(o) }
func (o *Oauth) GetOnes() error                       { return o.Account.GetOnes() }
func (o *Oauth) CanView(one *One) bool                { return aservice.CanView(o, one) }

/////////////////////////////////////////
//                Account
/////////////////////////////////////////

type Account struct {
	BaseModel
	Oauths  []Oauth `json:",omitempty"`
	Ones    []One   `json:"-"          gorm:"many2many:account_ones;"`
	Enabled bool    `json:",omitempty" sql:"default:true"`
}

func (a *Account) GetOnes() error { return aservice.GetOnes(a) }

// one must be non-exist record
// a   must be from Oauth.OnOid
// will not save id to o instance
func (a *Account) RegOne(o *One) error { return aservice.RegOne(a, o) }

// one must be exist record
// a   must be from Oauth.OnOid
func (a *Account) ViewOne(o *One) error { return aservice.ViewOne(a, o) }

// one must be exist record
// a   must be from Oauth.OnOid
func (a *Account) RemoveOne(o *One) error { return aservice.RemoveOne(a, o) }

func (a *Account) Logoff() error { return aservice.Logoff(a) }

/////////////////////////////////////////
//                One
/////////////////////////////////////////

// One must be added under an exist Account
type One struct {
	BaseModel
	Addr     string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(128);unique"`
	Enabled  bool      `json:",omitempty"                    sql:"default:true"`
	Owner    Account   `json:",omitempty"`
	OwnerId  uint      `json:",omitempty"`
	Accounts []Account `json:"-"                             gorm:"many2many:account_ones;"`
}

func (o *One) Find(addr []byte) error             { return aservice.FindOne(o, addr) }
func (o *One) FindIfOwner(id, ownerId uint) error { return aservice.FindOneIfOwner(o, id, ownerId) }
func (o *One) Save() error                        { return aservice.Save(o) }
func (o *One) Viewers() error                     { return aservice.Viewers(o) }
func (o *One) Delete() error                      { return aservice.Delete(o) }

/////////////////////////////////////////
//              AccountOne
/////////////////////////////////////////

// Used for adding cascade
type AccountOne struct {
	AccountId uint `gorm:"primary_key"`
	OneId     uint `gorm:"primary_key"`
}

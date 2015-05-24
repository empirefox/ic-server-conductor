package main

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

var (
	AccountNotAuthedErr = errors.New("Account is not authed")
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
	Validated bool    `json:",omitempty"                    sql:"default:false"`
	Enabled   bool    `json:",omitempty"                    sql:"default:false"`
}

func (o *Oauth) AddForeignKey() {
	DB.Model(o).AddForeignKey("account_id", "accounts", "CASCADE", "CASCADE")
}

// Find Oauth, preload Account and Account.Ones
// Create when not found
// Account will be created when a user do first login
func (o *Oauth) OnOid(provider, oid string) error {
	err := DB.Where(Oauth{Provider: provider, Oid: oid}).
		Attrs(Oauth{Account: Account{BaseModel: BaseModel{Name: provider + oid}}}).
		Preload("Account").FirstOrCreate(o).Error
	if err != nil {
		return err
	}
	return DB.Model(&o.Account).Related(&o.Account.Ones, "Ones").Error
}

func (o Oauth) Permitted(c *gin.Context) bool {
	return o.Validated
}

func (o Oauth) Valid() bool {
	return o.Enabled && o.Account.Enabled
}

/////////////////////////////////////////
//                Account
/////////////////////////////////////////

type Account struct {
	gorm.Model
	BaseModel
	Oauths  []Oauth `json:",omitempty"`
	Ones    []One   `json:",omitempty" gorm:"many2many:account_ones;"`
	Enabled bool    `json:",omitempty" sql:"default:false"`
}

// one must be non-exist record
// a   must be from Oauth.OnOid
func (a *Account) RegOne(one *One) error {
	one.Owner = *a
	return a.ViewOne(one)
}

// one must be exist record
// a   must be from Oauth.OnOid
func (a *Account) ViewOne(one *One) error {
	return DB.Model(a).Association("Ones").Append(one).Error
}

func (a *Account) indexOf(one *One) int {
	for i := range a.Ones {
		if a.Ones[i].ID == one.ID {
			return i
		}
	}
	return -1
}

// Owner remove one
func (a *Account) unregOne(one *One) error {
	err := DB.Delete(one).Error
	if err != nil {
		return err
	}
	if i := a.indexOf(one); i != -1 {
		a.Ones = append(a.Ones[:i], a.Ones[i+1:]...)
	}
	return nil
}

// Viewer remove one
func (a *Account) unviewOne(one *One) error {
	return DB.Model(a).Association("Ones").Delete(one).Error
}

// one must be exist record
// a   must be from Oauth.OnOid
func (a *Account) RemoveOne(one *One) error {
	if one.OwnerId != a.ID {
		return a.unviewOne(one)
	}
	return a.unregOne(one)
}

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
	OwnerId       uint      `json:"-"`
	Accounts      []Account `json:",omitempty"                    gorm:"many2many:account_ones;"`
}

func (o *One) AddForeignKey() {
	DB.Model(o).AddForeignKey("owner_id", "accounts", "CASCADE", "CASCADE")
}

func FindOne(addr []byte) (*One, error) {
	var one One
	err := DB.Where(One{SecretAddress: string(addr)}).Preload("Owner").First(&one).Error
	if err != nil {
		return &one, err
	}
	return &one, DB.Model(&one).Related(&one.Accounts, "Accounts").Error
}

/////////////////////////////////////////
//              AccountOne
/////////////////////////////////////////

// Used for adding cascade
type AccountOne struct {
	ID        uint `gorm:"primary_key"`
	AccountId uint
	OneId     uint
}

func (ao *AccountOne) AddForeignKey() {
	DB.Model(ao).AddForeignKey("account_id", "accounts", "CASCADE", "CASCADE")
	DB.Model(ao).AddForeignKey("one_id", "ones", "CASCADE", "CASCADE")
}

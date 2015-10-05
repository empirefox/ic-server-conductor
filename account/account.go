package account

import (
	"errors"
	"time"

	"github.com/empirefox/tagsjson/tagjson"
)

var (
	ErrLinkSelf      = errors.New("Cannot link account with self")
	ErrUnLinkSelf    = errors.New("Cannot unlink account with self")
	ErrOauthType     = errors.New("Wrong Oauth type")
	ErrUnauthedOauth = errors.New("Oauth not authed")
	ErrMultiLink     = errors.New("Oauth cannot link to multi account")
)

type BaseModel struct {
	tagjson.Tag
	ID        uint      `gorm:"primary_key" info:",omitempty"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	Name      string    `json:",omitempty"  info:",omitempty" sql:"type:varchar(128);not null"`
}

/////////////////////////////////////////
//                Oauth
/////////////////////////////////////////

type Oauth struct {
	BaseModel
	Account   Account `json:"-"`
	AccountId uint    `json:"-"`
	Oid       string  `json:",omitempty"                   sql:"type:varchar(128);not null"`
	Provider  string  `json:",omitempty"                   sql:"type:varchar(32);not null"`
	Picture   string  `json:",omitempty" info:",omitempty" sql:"type:varchar(128)"`
	Enabled   bool    `json:",omitempty"                   sql:"default:true"`
}

// Find Oauth, preload Account and Account.Ones
// Create when not found
// Account will be created when a user do first login
func (o *Oauth) OnLogin(provider, oid, name, pic string) error {
	return aservice.OnLogin(o, provider, oid, name, pic)
}
func (o *Oauth) OnLink(existed interface{}, provider, oid, name, pic string) error {
	if o.AccountId != 0 {
		return ErrMultiLink
	}
	o1, ok := existed.(*Oauth)
	if !ok {
		return ErrOauthType
	}
	if o1.AccountId == 0 {
		return ErrUnauthedOauth
	}
	if provider == o1.Provider {
		return ErrLinkSelf
	}
	o.Provider = provider
	o.Oid = oid
	o.Name = name
	o.Picture = pic
	o.AccountId = o1.AccountId
	return aservice.SaveOauth(o)
}
func (o *Oauth) Unlink(prd string) error {
	if o.Provider == prd {
		return ErrUnLinkSelf
	}
	return aservice.UnlinkOauth(o.AccountId, prd)
}
func (o *Oauth) GetOid() (provider, oid string)        { return o.Provider, o.Oid }
func (o *Oauth) Logoff() error                         { return o.Account.Logoff() }
func (o *Oauth) Find(provider, oid string) error       { return aservice.FindOauth(o, provider, oid) }
func (o *Oauth) Info() interface{}                     { return aservice.Info(o) }
func (o *Oauth) Valid() bool                           { return aservice.Valid(o) }
func (o *Oauth) GetOnes() error                        { return o.Account.GetOnes() }
func (o *Oauth) CanView(one *One) bool                 { return aservice.CanView(o, one) }
func (o *Oauth) GetProviders(ps *[]string) error       { return o.Account.GetProviders(ps) }
func (o *Oauth) ViewsByViewer(aos *[]AccountOne) error { return o.Account.ViewsByViewer(aos) }

/////////////////////////////////////////
//                Account
/////////////////////////////////////////

type Account struct {
	BaseModel
	Desc    string  `json:",omitempty" sql:"type:varchar(128);default:''" info:",omitempty" `
	Oauths  []Oauth `json:"-"`
	Ones    []One   `json:"-"          gorm:"many2many:account_ones;"`
	Enabled bool    `json:",omitempty" sql:"default:true"`
}

func (a *Account) GetOnes() error { return aservice.GetOnes(a) }

// one must be non-exist record
// a   must be from Oauth.OnLogin
// will not save id to o instance
func (a *Account) RegOne(o *One) error { return aservice.RegOne(a, o) }

// one must be exist record
// a   must be from Oauth.OnLogin
func (a *Account) ViewOne(o *One) error { return aservice.ViewOne(a, o) }

// one must be exist record
// a   must be from Oauth.OnLogin
func (a *Account) RemoveOne(o *One) error { return aservice.RemoveOne(a, o) }

// a   must be from Oauth.OnLogin
func (a *Account) GetProviders(ps *[]string) error { return aservice.AccountProviders(a, ps) }

func (a *Account) Logoff() error { return aservice.Logoff(a) }

func (a *Account) ViewsByViewer(aos *[]AccountOne) error { return aservice.ViewsByViewer(a, aos) }

/////////////////////////////////////////
//                One
/////////////////////////////////////////

// tagjson: {"upd":"dv"}
type One struct {
	tagjson.Tag
	ID        uint      `gorm:"primary_key"                 upd:",-o,+i"`
	CreatedAt time.Time `                                   upd:"-"`
	UpdatedAt time.Time `                                   upd:"-"`
	Name      string    `sql:"type:varchar(32);not null"    upd:",-o,+i;lmax(32)"`
	Desc      string    `sql:"type:varchar(128);default:''" upd:",-o;lmax(128)"`
	Addr      string    `sql:"not null;type:varchar(128)"   upd:"-"`
	Enabled   bool      `sql:"default:true"                 upd:",-o"`
	Owner     Account   `                                   upd:"-"`
	OwnerId   uint      `                                   upd:"-"`
	Accounts  []Account `gorm:"many2many:account_ones;"     upd:"-"`
	Ver       string    `sql:"-"                            upd:"-"`
}

func (o *One) Find(id uint) error                   { return aservice.FindOne(o, id) }
func (o *One) FindIfOwner(id, ownerId uint) error   { return aservice.FindOneIfOwner(o, id, ownerId) }
func (o *One) Save() error                          { return aservice.Save(o) }
func (o *One) Viewers() error                       { return aservice.Viewers(o) }
func (o *One) Delete() error                        { return aservice.Delete(o) }
func (o *One) ViewsByShare(aos *[]AccountOne) error { return aservice.ViewsByShare(o, aos) }

/////////////////////////////////////////
//              AccountOne
/////////////////////////////////////////

// Used for adding cascade
type AccountOne struct {
	AccountId    uint      `json:",omitempty" share:",omitempty"  gorm:"primary_key"`
	OneId        uint      `json:",omitempty" viewer:",omitempty" gorm:"primary_key"`
	ViewByShare  string    `json:",omitempty" share:",omitempty"  sql:"type:varchar(128)"`
	ViewByViewer string    `json:",omitempty" viewer:",omitempty" sql:"type:varchar(128)"`
	CreatedAt    time.Time `json:",omitempty"`
}

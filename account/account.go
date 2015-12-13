package account

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/empirefox/tagsjson/tagjson"
	"github.com/golang/glog"
)

const (
	UserInfo = iota
	UserRooms
	ViewByViewer
	ViewByShare
)

var (
	ErrLinkSelf      = errors.New("Cannot link account with self")
	ErrUnLinkSelf    = errors.New("Cannot unlink account with self")
	ErrOauthType     = errors.New("Wrong Oauth type")
	ErrUnauthedOauth = errors.New("Oauth not authed")
	ErrMultiLink     = errors.New("Oauth cannot link to multi account")
)

/////////////////////////////////////////
//                Oauth
/////////////////////////////////////////

// tagjson: {"UserInfo":"e"}
type Oauth struct {
	ID        uint      `gorm:"primary_key"               UserInfo:""`
	CreatedAt time.Time `                                 UserInfo:"-"`
	UpdatedAt time.Time `                                 UserInfo:"-"`
	Name      string    `sql:"type:varchar(128)"          UserInfo:""`
	Account   Account   `                                 UserInfo:""`
	AccountId uint      `                                 UserInfo:""`
	Oid       string    `sql:"type:varchar(128);not null" UserInfo:"-"`
	Provider  string    `sql:"type:varchar(32);not null"  UserInfo:"-"`
	Picture   string    `sql:"type:varchar(128)"          UserInfo:""`
	Enabled   bool      `sql:"default:true"               UserInfo:"-"`
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
func (o *Oauth) Info() interface{} {
	info, err := tagjson.MarshalR(o, UserInfo)
	if err != nil {
		glog.Errorln(err)
		return "error"
	}
	return info
}
func (o *Oauth) RawRooms() (*json.RawMessage, error) {
	if err := o.GetOnes(); err != nil {
		return nil, err
	}
	return tagjson.MarshalR(o.Account.Ones, UserRooms)
}
func (o *Oauth) RawViewsByViewer() (*json.RawMessage, error) {
	var aos AccountOnes
	if err := o.Account.ViewsByViewer(&aos); err != nil {
		return nil, err
	}
	return tagjson.MarshalR(aos, ViewByViewer)
}
func (o *Oauth) GetOid() (provider, oid string)  { return o.Provider, o.Oid }
func (o *Oauth) Logoff() error                   { return o.Account.Logoff() }
func (o *Oauth) Find(provider, oid string) error { return aservice.FindOauth(o, provider, oid) }
func (o *Oauth) Valid() bool                     { return aservice.Valid(o) }
func (o *Oauth) GetOnes() error                  { return o.Account.GetOnes() }
func (o *Oauth) CanView(one *One) bool           { return aservice.CanView(o, one) }
func (o *Oauth) GetProviders(ps *[]string) error { return o.Account.GetProviders(ps) }

/////////////////////////////////////////
//                Account
/////////////////////////////////////////

// tagjson: {"UserInfo":"e"}
type Account struct {
	ID        uint      `gorm:"primary_key"                 UserInfo:""`
	CreatedAt time.Time `                                   UserInfo:"-"`
	UpdatedAt time.Time `                                   UserInfo:"-"`
	Name      string    `sql:"type:varchar(128);not null"   UserInfo:""`
	Desc      string    `sql:"type:varchar(128);default:''" UserInfo:""`
	Oauths    []Oauth   `                                   UserInfo:"-"`
	Ones      Ones      `gorm:"many2many:account_ones;"     UserInfo:"-"`
	Enabled   bool      `sql:"default:true"                 UserInfo:"-"`
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

func (a *Account) ViewsByViewer(aos *AccountOnes) error { return aservice.ViewsByViewer(a, aos) }

/////////////////////////////////////////
//                One
/////////////////////////////////////////

// tagjson: {"UserRooms":"e"}
type One struct {
	ID        uint      `gorm:"primary_key"                 upd:",-o,+i"          UserRooms:""`
	CreatedAt time.Time `                                   upd:"-"               UserRooms:""`
	UpdatedAt time.Time `                                   upd:"-"               UserRooms:""`
	Name      string    `sql:"type:varchar(32);not null"    upd:",-o,+i;lmax(32)" UserRooms:""`
	Desc      string    `sql:"type:varchar(128);default:''" upd:",-o;lmax(128)"   UserRooms:""`
	Addr      string    `sql:"not null;type:varchar(128)"   upd:"-"               UserRooms:"-"`
	Enabled   bool      `sql:"default:true"                 upd:",-o"             UserRooms:""`
	Owner     Account   `                                   upd:"-"               UserRooms:"-"`
	OwnerId   uint      `                                   upd:"-"               UserRooms:""`
	Accounts  []Account `gorm:"many2many:account_ones;"     upd:"-"               UserRooms:"-"`
	Ver       string    `sql:"-"                            upd:"-"               UserRooms:""`
}

// tagjson: include
type Ones []One

func (o *One) Find(id uint) error                 { return aservice.FindOne(o, id) }
func (o *One) FindIfOwner(id, ownerId uint) error { return aservice.FindOneIfOwner(o, id, ownerId) }
func (o *One) Save() error                        { return aservice.Save(o) }
func (o *One) Viewers() error                     { return aservice.Viewers(o) }
func (o *One) Delete() error                      { return aservice.Delete(o) }
func (o *One) RawViewsByShare() (*json.RawMessage, error) {
	var aos AccountOnes
	if err := aservice.ViewsByShare(o, &aos); err != nil {
		return nil, err
	}
	return tagjson.MarshalR(aos, ViewByShare)
}

// Fix bug for mysql. See https://github.com/yanfali/gorm/commit/8545f588249455f922d415a699e0526c779a1639
// For mysql add:
// CREATE TRIGGER `account_del_oauths` BEFORE DELETE ON `accounts`
// FOR EACH ROW DELETE FROM oauths WHERE oauths.account_id =  OLD.id
/////////////////////////////////////////
//              AccountOne
/////////////////////////////////////////

// tagjson: {"ViewByViewer":"e","ViewByShare":"e"}
type AccountOne struct {
	AccountId    uint      `gorm:"primary_key;foreignkey"      ViewByViewer:"-" ViewByShare:""`
	OneId        uint      `gorm:"primary_key;foreignkey"      ViewByViewer:""  ViewByShare:"-"`
	ViewByShare  string    `sql:"type:varchar(128)" ViewByViewer:"-" ViewByShare:""`
	ViewByViewer string    `sql:"type:varchar(128)" ViewByViewer:""  ViewByShare:"-"`
	CreatedAt    time.Time `                        ViewByViewer:""  ViewByShare:""`
}

// tagjson: include
type AccountOnes []AccountOne

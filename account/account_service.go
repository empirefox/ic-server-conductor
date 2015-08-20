package account

import (
	"errors"

	"github.com/gin-gonic/gin"

	. "github.com/empirefox/ic-server-conductor/gorm"
)

var (
	aservice            = NewAccountService()
	AccountNotAuthedErr = errors.New("Account is not authed")
	ErrParamsRequired   = errors.New("Query param required")
)

func SetService(a AccountService) {
	if a == nil {
		aservice = NewAccountService()
	} else {
		aservice = a
	}
}

func ClearTables() error {
	err := aservice.DropTables()
	if err != nil {
		return err
	}
	return aservice.CreateTables()
}

// Used by application
type AccountService interface {
	CreateTables() error
	DropTables() error

	FindOauthProviders(ops *OauthProviders) error
	SaveOauthProvider(op *OauthProvider) error

	OnOid(o *Oauth, provider, oid string) error
	FindOauth(o *Oauth, provider, oid string) error
	Permitted(o *Oauth, c *gin.Context) bool
	Valid(o *Oauth) bool
	CanView(o *Oauth, one *One) bool

	GetOnes(a *Account) error
	RegOne(a *Account, o *One) error
	ViewOne(a *Account, o *One) error
	RemoveOne(a *Account, o *One) error
	Associate(a *Account, o *Oauth) error
	UnAssociate(a *Account, p string) error
	AccountProviders(a *Account, ps *[]string) error
	Logoff(a *Account) error

	FindOne(o *One, addr []byte) error
	FindOneIfOwner(o *One, id, ownerId uint) error
	Save(o *One) error
	Viewers(o *One) error
	Delete(o *One) error
}

func NewAccountService() AccountService {
	return accountService{}
}

type accountService struct{}

func (accountService) CreateTables() error {
	ao := &AccountOne{}
	one := &One{}
	oauth := &Oauth{}
	return DB.CreateTable(ao).CreateTable(&Account{}).CreateTable(one).
		CreateTable(oauth).CreateTable(&OauthProvider{}).
		Model(ao).AddForeignKey("account_id", "accounts", "CASCADE", "CASCADE").
		Model(ao).AddForeignKey("one_id", "ones", "CASCADE", "CASCADE").
		Model(one).AddForeignKey("owner_id", "accounts", "CASCADE", "CASCADE").
		Model(oauth).AddForeignKey("account_id", "accounts", "CASCADE", "CASCADE").Error
}

func (accountService) DropTables() error {
	return DB.DropTableIfExists(&AccountOne{}).DropTableIfExists(&Oauth{}).DropTableIfExists(&One{}).
		DropTableIfExists(&Account{}).DropTableIfExists(&OauthProvider{}).Error
}

func (accountService) FindOauthProviders(ops *OauthProviders) error {
	return DB.Where(&OauthProvider{Enabled: true}).Find(ops).Error
}

func (accountService) SaveOauthProvider(op *OauthProvider) error {
	return DB.Save(op).Error
}

func (accountService) Associate(a *Account, o *Oauth) error {
	return DB.Model(a).Association("Oauths").Append(o).Error
}

func (accountService) UnAssociate(a *Account, p string) error {
	w := &Oauth{AccountId: a.ID}
	return DB.Delete(w, w).Error
}

func (accountService) AccountProviders(a *Account, ps *[]string) error {
	w := &Oauth{AccountId: a.ID}
	return DB.Model(w).Where(w).Pluck("providers", ps).Error
}

func (accountService) Logoff(a *Account) error {
	return DB.Unscoped().Delete(a).Error
}

func (accountService) GetOnes(a *Account) error {
	ones := []One{}
	err := DB.Model(a).Association("Ones").Find(&ones).Error
	a.Ones = ones
	return err
}

// one must be non-exist record
// a   must be from Oauth.OnOid
func (accountService) RegOne(a *Account, one *One) error {
	one.Owner = *a
	return a.ViewOne(one)
}

// one must be exist record
// a   must be from Oauth.OnOid
func (accountService) ViewOne(a *Account, one *One) error {
	return DB.Model(a).Association("Ones").Append(one).Error
}

func indexOf(a *Account, one *One) int {
	for i := range a.Ones {
		if a.Ones[i].ID == one.ID {
			return i
		}
	}
	return -1
}

// one must be exist record
// a   must be from Oauth.OnOid
func (accountService) RemoveOne(a *Account, one *One) error {
	if one.OwnerId == a.ID {
		err := DB.Delete(one).Error
		if err != nil {
			return err
		}
		if i := indexOf(a, one); i != -1 {
			a.Ones = append(a.Ones[:i], a.Ones[i+1:]...)
		}
		return nil
	}
	return DB.Model(a).Association("Ones").Delete(one).Error
}

func (accountService) FindOne(o *One, addr []byte) error {
	var w One
	w.Addr = string(addr)
	return DB.Where(w).Preload("Owner").First(o).Error
}

func (accountService) FindOneIfOwner(o *One, id, ownerId uint) error {
	var w One
	w.ID = id
	w.OwnerId = ownerId
	return DB.Where(w).First(o).Error
}

func (accountService) Save(o *One) error {
	return DB.Save(o).Error
}

func (accountService) Viewers(o *One) error {
	viewers := []Account{}
	err := DB.Model(o).Association("Accounts").Find(&viewers).Error
	o.Accounts = viewers
	return err
}

func (accountService) Delete(o *One) error {
	return DB.Delete(o).Error
}

func (accountService) OnOid(o *Oauth, provider, oid string) error {
	if provider == "" || oid == "" {
		return ErrParamsRequired
	}
	return DB.Where(&Oauth{Provider: provider, Oid: oid, Validated: true, Enabled: true}).
		Attrs(&Oauth{Account: Account{BaseModel: BaseModel{Name: provider + oid}, Enabled: true}}).
		Preload("Account").FirstOrCreate(o).Error
}

func (accountService) FindOauth(o *Oauth, provider, oid string) error {
	if provider == "" || oid == "" {
		return ErrParamsRequired
	}
	return DB.Debug().Where(&Oauth{Provider: provider, Oid: oid, Validated: true, Enabled: true}).
		Preload("Account").First(o).Error
}

func (accountService) Permitted(o *Oauth, c *gin.Context) bool { return o.Validated }

func (accountService) Valid(o *Oauth) bool { return o.Enabled && o.Account.Enabled }

func (accountService) CanView(o *Oauth, one *One) bool {
	r := &AccountOne{
		AccountId: o.Account.ID,
		OneId:     one.ID,
	}
	var count uint
	DB.Model(r).Where(r).Count(&count)
	return count == 1
}

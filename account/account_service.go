package account

import (
	"errors"

	. "github.com/empirefox/ic-server-conductor/gorm"
)

var (
	aservice            = NewAccountService()
	AccountNotAuthedErr = errors.New("As Account is not authed")
	ErrParamsRequired   = errors.New("As Query param required")
)

func SetService(a AccountService) {
	if a == nil {
		aservice = NewAccountService()
	} else {
		aservice = a
	}
}

func ClearTables() error {
	return aservice.DropTables()
}

func CreateTables() error {
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

	OnLogin(o *Oauth, provider, oid, name, pic string) error
	SaveOauth(o *Oauth) error
	UnlinkOauth(accountId uint, prd string) error
	FindOauth(o *Oauth, provider, oid string) error
	Valid(o *Oauth) bool
	CanView(o *Oauth, one *One) bool

	GetOnes(a *Account) error
	RegOne(a *Account, o *One) error
	ViewOne(a *Account, o *One) error
	RemoveOne(a *Account, o *One) error
	AccountProviders(a *Account, ps *[]string) error
	Logoff(a *Account) error
	ViewsByViewer(a *Account, aos *AccountOnes) error

	FindOne(o *One, id uint) error
	FindOneIfOwner(o *One, id, ownerId uint) error
	Save(o *One) error
	Viewers(o *One) error
	Delete(o *One) error
	ViewsByShare(o *One, aos *AccountOnes) error
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
	return DB.Find(ops).Error
}

func (accountService) SaveOauthProvider(op *OauthProvider) error {
	return DB.Save(op).Error
}

func (accountService) AccountProviders(a *Account, ps *[]string) error {
	w := &Oauth{AccountId: a.ID}
	return DB.Model(w).Where(w).Pluck("provider", ps).Error
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

func (accountService) ViewsByViewer(a *Account, aos *AccountOnes) error {
	return DB.Where(AccountOne{AccountId: a.ID}).Select([]string{"one_id", "view_by_viewer"}).Find(aos).Error
}

// one must be non-exist record
// a   must be from Oauth.OnLogin
func (as accountService) RegOne(a *Account, one *One) error {
	one.OwnerId = a.ID
	return as.ViewOne(a, one)
}

// one must be exist record
// a   must be from Oauth.OnLogin
func (accountService) ViewOne(a *Account, one *One) error {
	tx := DB.Debug().Begin()
	if err := tx.Save(one).Error; err != nil {
		tx.Rollback()
		return err
	}
	ao := &AccountOne{AccountId: a.ID, ViewByShare: a.Name, OneId: one.ID, ViewByViewer: one.Name}
	if err := tx.Create(ao).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// one must be exist record
// a   must be from Oauth.OnLogin
func (accountService) RemoveOne(a *Account, one *One) error {
	if one.OwnerId == a.ID {
		return DB.Delete(one).Error
	}
	return DB.Model(a).Association("Ones").Delete(one).Error
}

func (accountService) FindOne(o *One, id uint) error {
	var w One
	w.ID = id
	return DB.Where(w).Preload("Owner").First(o).Error
}

func (accountService) FindOneIfOwner(o *One, id, ownerId uint) error {
	return DB.Where("id = ? and owner_id = ?", id, ownerId).Preload("Owner").First(o).Error
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

func (accountService) ViewsByShare(o *One, aos *AccountOnes) error {
	return DB.Where(AccountOne{OneId: o.ID}).Select([]string{"account_id", "view_by_share"}).Find(aos).Error
}

func (accountService) OnLogin(o *Oauth, provider, oid, name, pic string) error {
	if provider == "" || oid == "" || name == "" {
		return ErrParamsRequired
	}
	w := Oauth{Provider: provider, Oid: oid, Picture: pic, Enabled: true}
	w.Name = name
	attr := Oauth{}
	attr.Account.Name = name
	attr.Account.Enabled = true
	return DB.Where(&w).Attrs(&attr).Preload("Account").FirstOrCreate(o).Error
}

func (accountService) SaveOauth(o *Oauth) error {
	return DB.Save(o).Error
}

func (accountService) UnlinkOauth(accountId uint, prd string) error {
	if accountId == 0 || prd == "" {
		return ErrParamsRequired
	}
	return DB.Where(&Oauth{AccountId: accountId, Provider: prd}).Delete(Oauth{}).Error
}

func (accountService) FindOauth(o *Oauth, provider, oid string) error {
	if provider == "" || oid == "" {
		return ErrParamsRequired
	}
	return DB.Where(&Oauth{Provider: provider, Oid: oid, Enabled: true}).
		Preload("Account").First(o).Error
}

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

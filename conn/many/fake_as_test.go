package many

import (
	. "github.com/empirefox/ic-server-ws-signal/account"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	dataOnOid   Oauth
	dataGetOnes []One
	dataFindOne One
	dataViewers []Account
}

func (s fakeService) CreateTables() error { return nil }
func (s fakeService) DropTables() error   { return nil }

func (s fakeService) FindOauthProviders(ops *OauthProviders) error { return nil }
func (s fakeService) SaveOauthProvider(ops *OauthProvider) error   { return nil }

func (s fakeService) OnOid(o *Oauth, provider, oid string) error { *o = s.dataOnOid; return nil }
func (s fakeService) Permitted(o *Oauth, c *gin.Context) bool    { return true }
func (s fakeService) Valid(o *Oauth) bool                        { return true }
func (s fakeService) CanView(o *Oauth, one *One) bool            { return true }

func (s fakeService) GetOnes(a *Account) error           { a.Ones = s.dataGetOnes; return nil }
func (s fakeService) RegOne(a *Account, o *One) error    { return nil }
func (s fakeService) ViewOne(a *Account, o *One) error   { return nil }
func (s fakeService) RemoveOne(a *Account, o *One) error { return nil }
func (s fakeService) Logoff(a *Account) error            { return nil }

func (s fakeService) FindOne(o *One, addr []byte) error             { *o = s.dataFindOne; return nil }
func (s fakeService) FindOneIfOwner(o *One, id, ownerId uint) error { return nil }
func (s fakeService) Save(o *One) error                             { return nil }
func (s fakeService) Viewers(o *One) error                          { o.Accounts = s.dataViewers; return nil }

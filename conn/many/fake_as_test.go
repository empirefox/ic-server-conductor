package many

import . "github.com/empirefox/ic-server-conductor/account"

type fakeService struct {
	dataOnOid   Oauth
	dataGetOnes []One
	dataFindOne One
	dataViewers []Account
	dataPrds    []string
	dataViews   []AccountOne
}

func (s fakeService) CreateTables() error { return nil }
func (s fakeService) DropTables() error   { return nil }

func (s fakeService) FindOauthProviders(ops *OauthProviders) error { return nil }
func (s fakeService) SaveOauthProvider(ops *OauthProvider) error   { return nil }

func (s fakeService) OnLogin(o *Oauth, provider, oid, name, pic string) error {
	*o = s.dataOnOid
	return nil
}
func (s fakeService) SaveOauth(o *Oauth) error                       { return nil }
func (s fakeService) UnlinkOauth(accountId uint, prd string) error   { return nil }
func (s fakeService) FindOauth(o *Oauth, provider, oid string) error { return nil }
func (s fakeService) Info(o *Oauth) interface{}                      { return o }
func (s fakeService) Valid(o *Oauth) bool                            { return true }
func (s fakeService) CanView(o *Oauth, one *One) bool                { return true }

func (s fakeService) GetOnes(a *Account) error                        { a.Ones = s.dataGetOnes; return nil }
func (s fakeService) RegOne(a *Account, o *One) error                 { return nil }
func (s fakeService) ViewOne(a *Account, o *One) error                { return nil }
func (s fakeService) RemoveOne(a *Account, o *One) error              { return nil }
func (s fakeService) AccountProviders(a *Account, ps *[]string) error { *ps = s.dataPrds; return nil }
func (s fakeService) Logoff(a *Account) error                         { return nil }
func (s fakeService) ViewsByViewer(a *Account, aos *[]AccountOne) error {
	*aos = s.dataViews
	return nil
}

func (s fakeService) FindOne(o *One, id uint) error                 { *o = s.dataFindOne; return nil }
func (s fakeService) FindOneIfOwner(o *One, id, ownerId uint) error { return nil }
func (s fakeService) Save(o *One) error                             { return nil }
func (s fakeService) Viewers(o *One) error                          { o.Accounts = s.dataViewers; return nil }
func (s fakeService) Delete(o *One) error                           { return nil }
func (s fakeService) ViewsByShare(o *One, aos *[]AccountOne) error  { *aos = s.dataViews; return nil }

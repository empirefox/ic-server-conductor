package account

import (
	"encoding/json"

	"golang.org/x/oauth2"

	"github.com/empirefox/gin-oauth2"
	"github.com/fatih/structs"
)

type OauthProvider struct {
	ID           uint   `gorm:"primary_key"                                                      satellizer:"-"`
	Name         string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:",omitempty"`
	ClientID     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:",omitempty"`
	ClientSecret string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
	TokenURL     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
	RedirectURL  string `json:",omitempty"                    sql:"type:varchar(255);not null"   satellizer:",omitempty"`
	UserEndpoint string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
	JsonPathOid  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
	JsonPathName string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
	JsonPathPic  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
	Disabled     bool   `json:",omitempty"                    sql:"default:false"                satellizer:",omitempty"`
}

type SatellizerProvider map[string]interface{}

func (op *OauthProvider) Save() error {
	return aservice.SaveOauthProvider(op)
}

func (op *OauthProvider) ToSatellizer() SatellizerProvider {
	p := structs.New(op)
	p.TagName = "satellizer"
	return p.Map()
}

func (o *OauthProvider) ToGoauth() goauth.Provider {
	return goauth.Provider{
		Config: oauth2.Config{
			ClientID:     o.ClientID,
			ClientSecret: o.ClientSecret,
			RedirectURL:  o.RedirectURL,
			Endpoint: oauth2.Endpoint{
				TokenURL: o.TokenURL,
			},
		},
		Name:         o.Name,
		UserEndpoint: o.UserEndpoint,
		JsonPathOid:  o.JsonPathOid,
		JsonPathName: o.JsonPathName,
		JsonPathPic:  o.JsonPathPic,
	}
}

// Can be use by satellizer
func SatellizerProviders() ([]byte, error) {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		return nil, err
	}
	return json.Marshal(ops.ToSatellizer())
}

// Can be use by goauth
func GoauthProviders(grp string) (map[string]goauth.Provider, error) {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		return nil, err
	}
	return ops.ToGoauth(grp), nil
}

type OauthProviders []OauthProvider

func (ops *OauthProviders) All() error {
	return aservice.FindOauthProviders(ops)
}

func (ops OauthProviders) ToSatellizer() []SatellizerProvider {
	ps := make([]SatellizerProvider, 0)
	for _, v := range ops {
		ps = append(ps, v.ToSatellizer())
	}
	return ps
}

func (ops OauthProviders) ToGoauth(grp string) map[string]goauth.Provider {
	prefix := grp + "/"
	ps := make(map[string]goauth.Provider, 0)
	for _, op := range ops {
		if !op.Disabled {
			ps[prefix+op.Name] = op.ToGoauth()
		}
	}
	return ps
}

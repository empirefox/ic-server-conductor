package account

import (
	"encoding/json"
	"strings"

	"golang.org/x/oauth2"

	"github.com/empirefox/gin-oauth2"
	"github.com/fatih/structs"
)

type OauthProvider struct {
	BaseModel
	Path         string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"path,omitempty"`
	ClientID     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"client_id,omitempty"`
	ClientSecret string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	AuthURL      string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"auth_url,omitempty"`
	TokenURL     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	RedirectURL  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"redirect_url,omitempty"`
	Scope        string `json:",omitempty"                    sql:"type:varchar(255);default:''" satellizer:"scope,omitempty"`
	UserEndpoint string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	JsonPathOid  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	JsonPathName string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	JsonPathPic  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	Disabled     bool   `json:",omitempty"                    sql:"default:false"`
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
			Scopes:       strings.Split(o.Scope, ","),
			Endpoint: oauth2.Endpoint{
				AuthURL:  o.AuthURL,
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
func GoauthProviders() (map[string]goauth.Provider, error) {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		return nil, err
	}
	return ops.ToGoauth(), nil
}

type OauthProviders []OauthProvider

func (ops OauthProviders) All() error {
	return aservice.FindOauthProviders(&ops)
}

func (ops OauthProviders) ToSatellizer() []SatellizerProvider {
	ps := make([]SatellizerProvider, len(ops))
	for _, v := range ops {
		ps = append(ps, v.ToSatellizer())
	}
	return ps
}

func (ops OauthProviders) ToGoauth() map[string]goauth.Provider {
	ps := make(map[string]goauth.Provider, len(ops))
	for _, op := range ops {
		ps[op.Path] = op.ToGoauth()
	}
	return ps
}

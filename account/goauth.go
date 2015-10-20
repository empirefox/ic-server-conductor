package account

import (
	"encoding/json"
	"strings"

	"github.com/empirefox/gin-oauth2"
	"github.com/fatih/structs"
)

type OauthProvider struct {
	ID           uint   `gorm:"primary_key"                                                      satellizer:"-"`
	Name         string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:",omitempty"`
	ClientID     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:",omitempty"`
	ClientSecret string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"   satellizer:"-"`
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

// Can be use by satellizer
func SatellizerProviders() ([]byte, error) {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		return nil, err
	}
	return json.Marshal(ops.ToSatellizer())
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

// Can be use by goauth
func AddGoauthProviders(config *goauth.Config, grp string) error {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		return err
	}
	if !strings.HasSuffix(grp, "/") {
		grp = grp + "/"
	}
	for _, op := range ops {
		if !op.Disabled {
			if err := config.AddProvider(op.Name, grp+op.Name, op.ClientID, op.ClientSecret); err != nil {
				return err
			}
		}
	}
	return nil
}

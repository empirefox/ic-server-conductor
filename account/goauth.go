package account

import (
	"strings"

	"github.com/empirefox/gin-oauth2"
)

// tagjson: {                                            "PrdSave":"dv",         "PrdSatellizer":"e"}
type OauthProvider struct {
	ID           uint   `gorm:"primary_key"               PrdSave:""              PrdSatellizer:"-"`
	Name         string `sql:"type:varchar(255);not null" PrdSave:",+i;lmax(255)" PrdSatellizer:""`
	ClientID     string `sql:"type:varchar(255);not null" PrdSave:",+i;lmax(255)" PrdSatellizer:""`
	ClientSecret string `sql:"type:varchar(255);not null" PrdSave:",+i;lmax(255)" PrdSatellizer:"-"`
	Disabled     bool   `sql:"default:false"              PrdSave:""              PrdSatellizer:""`
}

// tagjson: include
type OauthProviders []OauthProvider

func (op *OauthProvider) Save() error {
	return aservice.SaveOauthProvider(op)
}

// Can be use by Satellizer
func SatellizerProviders() ([]byte, error) {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		return nil, err
	}
	return ops.MarshalTagJSON(PrdSatellizer)
}

func (ops *OauthProviders) All() error {
	return aservice.FindOauthProviders(ops)
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

package main

import (
	"encoding/json"
	"strings"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"golang.org/x/oauth2"

	. "test.com/empirefox/gin-oauth2"
)

type OauthProvider struct {
	gorm.Model
	BaseModel
	Path         string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	ClientID     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	ClientSecret string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	AuthURL      string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	TokenURL     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	RedirectURL  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	Scopes       string `json:",omitempty" binding:"required" sql:"type:varchar(255);default:''"`
	UserEndpoint string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	OidJsonPath  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	Enabled      bool   `json:",omitempty"                    sql:"default:true"`

	Css string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
}

func (c *OauthProvider) ConfigProvider() Provider {
	return Provider{
		Config: oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			RedirectURL:  c.RedirectURL,
			Scopes:       strings.Split(c.Scopes, ",|"),
			Endpoint: oauth2.Endpoint{
				AuthURL:  c.AuthURL,
				TokenURL: c.TokenURL,
			},
		},
		Name:         c.Name,
		UserEndpoint: c.UserEndpoint,
	}
}

type PageOauth struct {
	Path string `json:"path,omitempty"`
	Text string `json:"text,omitempty"`
	Css  string `json:"css,omitempty"`
}

func NewGoauthConf() (*Config, []byte) {
	var ops []OauthProvider
	if err := DB.Where(map[string]interface{}{"enabled": true}).Find(&ops).Error; err != nil {
		glog.Errorln(err)
	}
	ps := make(map[string]Provider, len(ops))
	pos := make([]PageOauth, len(ops))
	for _, op := range ops {
		if op.Enabled {
			ps[op.Path] = op.ConfigProvider()
			pos = append(pos, PageOauth{op.Path, op.Name, op.Css})
		}
	}
	config := &Config{
		Providers: ps,
		NewUserFunc: func() OauthUser {
			return &Oauth{}
		},
	}
	oauthBs, err := json.Marshal(pos)
	if err != nil {
		glog.Errorln(err)
	}
	return config, oauthBs
}

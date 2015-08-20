package account

import (
	"encoding/json"

	"github.com/golang/glog"
)

type OauthProvider struct {
	BaseModel
	Path         string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	ClientID     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	ClientSecret string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	AuthURL      string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	TokenURL     string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	RedirectURL  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	Scopes       string `json:",omitempty"                    sql:"type:varchar(255);default:''"`
	UserEndpoint string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	OidJsonPath  string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
	Enabled      bool   `json:",omitempty"                    sql:"default:false"`

	Css string `json:",omitempty" binding:"required" sql:"type:varchar(255);not null"`
}

func (op *OauthProvider) Save() error {
	return aservice.SaveOauthProvider(op)
}

type PageOauth struct {
	Path string `json:"path,omitempty"`
	Text string `json:"text,omitempty"`
	Css  string `json:"css,omitempty"`
}

type OauthProviders []OauthProvider

func (ops *OauthProviders) All() error {
	return aservice.FindOauthProviders(ops)
}

func PageOauthsBytes() []byte {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		glog.Errorln(err)
		return nil
	}
	pos := make([]PageOauth, 0, len(ops))
	for _, op := range ops {
		pos = append(pos, PageOauth{op.Path, op.Name, op.Css})
	}
	oauthBs, err := json.Marshal(pos)
	if err != nil {
		glog.Errorln(err)
	}
	return oauthBs
}

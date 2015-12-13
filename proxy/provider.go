package proxy

import "github.com/empirefox/gin-oauth2"

type Provider struct {
	Name, Path, ClientID, ClientSecret string
}

var Providers []Provider

func Add(Name, Path, ClientID, ClientSecret string) {
	Providers = append(Providers, Provider{Name, Path, ClientID, ClientSecret})
}

type PostProxyTokenData struct {
	Token string
	Info  goauth.UserInfo
}

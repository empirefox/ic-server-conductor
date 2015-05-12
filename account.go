package main

import "time"

//Provider:Google,Github,Qq,Weibo,Baidu,Souhu,Netease,Douban
type Oauth struct {
	Id          int64     `json:",omitempty"`
	AccountId   int64     `json:",omitempty"                    sql:"not null"`
	Oid         string    `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Provider    string    `json:",omitempty" binding:"required" sql:"type:varchar(32);not null"`
	Name        string    `json:",omitempty" binding:"required" sql:"type:varchar(128);not null"`
	Description string    `json:",omitempty"                    sql:"type:varchar(128);default:''"`
	Validated   bool      `json:",omitempty"                    sql:"default:false"`
	Enabled     bool      `json:",omitempty"                    sql:"default:false"`
	LogedAt     time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
	CreatedAt   time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
}

type One struct {
	Id            int64     `json:",omitempty"`
	Name          string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(128);unique"`
	Description   string    `json:",omitempty"                    sql:"type:varchar(128);default:''"`
	SecretAddress string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(128);unique"`
	Enabled       bool      `json:",omitempty"                    sql:"default:true"`
	Owner         Account   `json:",omitempty"`
	Accounts      []Account `json:",omitempty"`
	CreatedAt     time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
}

type Account struct {
	Id          int64     `json:",omitempty"`
	Name        string    `json:",omitempty" binding:"required" sql:"not null;type:varchar(128);unique"`
	Description string    `json:",omitempty"                    sql:"type:varchar(128);default:''"`
	Oauths      []Oauth   `json:",omitempty"`
	Ones        []One     `json:",omitempty"`
	Enabled     bool      `json:",omitempty"                    sql:"default:false"`
	CreatedAt   time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `json:",omitempty"                    sql:"default:CURRENT_TIMESTAMP"`
}

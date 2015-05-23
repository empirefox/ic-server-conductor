//GORM_DIALECT=mysql DB_URL="gorm:gorm@/gorm?charset=utf8&parseTime=True"
//GORM_DIALECT=postgres DB_URL="user=gorm dbname=gorm sslmode=disable" PORT=8080 goconvey
//GORM_DIALECT=sqlite3 DB_URL=/tmp/gorm.DB go testpackage db
package main

import (
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var (
	DB gorm.DB
)

func init() {
	var err error
	vendor := os.Getenv("GORM_DIALECT")
	url := os.Getenv("DB_URL")

	if vendor == "" || url == "" {
		panic("数据库环境变量没有正确设置")
	}

	DB, err = gorm.Open(vendor, url)

	if err != nil {
		panic(fmt.Sprintf("No error should happen when connect database, but got %+v", err))
	}

	DB.DB().SetMaxIdleConns(5)
	DB.DB().SetMaxOpenConns(10)
}

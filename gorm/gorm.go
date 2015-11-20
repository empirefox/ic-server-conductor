//DB_URL="gorm:gorm@/gorm?charset=utf8&parseTime=True"
//DB_URL="postgres://postgres:6Vno3r3gH9sZHSxo@localhost/ic_signal_test?sslmode=disable"
//Notsupported: GORM_DIALECT=sqlite3 DB_URL=/tmp/gorm.DB
package gorm

import (
	"fmt"
	"os"
	"strconv"

	"github.com/empirefox/gotool/paas"
	"github.com/jinzhu/gorm"
	//	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"
	//	_ "github.com/mattn/go-sqlite3"
)

var (
	DB gorm.DB
)

func init() {
	if os.Getenv("TEST_NO_DB") == "true" {
		return
	}

	if paas.Gorm.Url == "" {
		panic("'DB_URL' must be set, or set TEST_NO_DB=true for test.")
	}

	var err error
	DB, err = gorm.Open(paas.Gorm.Dialect, paas.Gorm.Url)
	if err != nil {
		panic(fmt.Sprintf("No error should happen when connect database, but got %+v", err))
	}

	debug, _ := strconv.ParseBool(os.Getenv("GORM_DEBUG"))
	DB.LogMode(debug)

	DB.DB().SetMaxIdleConns(paas.Gorm.MaxIdle)
	DB.DB().SetMaxOpenConns(paas.Gorm.MaxOpen)
}

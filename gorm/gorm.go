//DB_URL="gorm:gorm@/gorm?charset=utf8&parseTime=True"
//DB_URL="postgres://postgres:6Vno3r3gH9sZHSxo@localhost/ic_signal_test?sslmode=disable"
//Notsupported: GORM_DIALECT=sqlite3 DB_URL=/tmp/gorm.DB
package gorm

import (
	"fmt"
	"os"

	"github.com/empirefox/gotool/paas"
	"github.com/jinzhu/gorm"
	//	_ "github.com/lib/pq"
	//	_ "github.com/go-sql-driver/mysql"
	//	_ "github.com/mattn/go-sqlite3"
)

var (
	DB gorm.DB
)

func init() {
	if os.Getenv("TEST_NO_DB") == "true" {
		return
	}

	paasGorm := paas.GetGorm()
	if paasGorm.Url == "" {
		panic("Now in test mode, but 'DB_URL' must be set")
	}

	var err error
	DB, err = gorm.Open(paasGorm.Dialect, paasGorm.Url)
	if err != nil {
		panic(fmt.Sprintf("No error should happen when connect database, but got %+v", err))
	}

	DB.DB().SetMaxIdleConns(paasGorm.MaxIdle)
	DB.DB().SetMaxOpenConns(paasGorm.MaxOpen)
}

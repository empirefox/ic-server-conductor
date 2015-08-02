//GORM_DIALECT=mysql DB_URL="gorm:gorm@/gorm?charset=utf8&parseTime=True"
//GORM_DIALECT=postgres DB_URL="postgres://postgres:6Vno3r3gH9sZHSxo@localhost/ic_signal_test?sslmode=disable"
//GORM_DIALECT=sqlite3 DB_URL=/tmp/gorm.DB go test
package account

import (
	"encoding/json"
	"strconv"
	"testing"

	_ "github.com/lib/pq"
	. "github.com/smartystreets/goconvey/convey"

	. "github.com/empirefox/ic-server-ws-signal/gorm"
)

func init() {
	DB.LogMode(true)
}

func initProvider(i int) (*OauthProvider, error) {
	suffix := strconv.Itoa(i)
	p := &OauthProvider{
		BaseModel:    BaseModel{Name: "provider" + suffix},
		Path:         "path" + suffix,
		ClientID:     "client_id" + suffix,
		ClientSecret: "client_secret" + suffix,
		AuthURL:      "auth_url" + suffix,
		TokenURL:     "token_url" + suffix,
		RedirectURL:  "redirect_url" + suffix,
		UserEndpoint: "user_endpoint" + suffix,
		OidJsonPath:  "oid_json_path" + suffix,
		Css:          "css" + suffix,
	}
	return p, DB.Save(p).Error
}

func Test_PageOauthsBytes(t *testing.T) {
	Convey("findProviders", t, func() {
		recoveryAccount()
		Convey("should gen PageOauths bytes", func() {
			_, err := initProvider(1)
			So(err, ShouldBeNil)
			_, err = initProvider(2)
			So(err, ShouldBeNil)

			var pos []PageOauth
			err = json.Unmarshal(PageOauthsBytes(), &pos)
			So(err, ShouldBeNil)

			posResult := []PageOauth{
				{Path: "path1", Text: "provider1", Css: "css1"},
				{Path: "path2", Text: "provider2", Css: "css2"},
			}
			So(pos, ShouldResemble, posResult)
		})
	})
}

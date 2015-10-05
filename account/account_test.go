//GORM_DIALECT=mysql DB_URL="gorm:gorm@/gorm?charset=utf8&parseTime=True"
//GORM_DIALECT=postgres DB_URL="postgres://postgres:6Vno3r3gH9sZHSxo@localhost/ic_signal_test?sslmode=disable"
//GORM_DIALECT=sqlite3 DB_URL=/tmp/gorm.DB go test
package account

import (
	"flag"
	"testing"

	_ "github.com/lib/pq"
	. "github.com/smartystreets/goconvey/convey"

	. "github.com/empirefox/ic-server-conductor/gorm"
)

func init() {
	//	DB.LogMode(true)
	flag.Set("stderrthreshold", "INFO")
}

func recoveryAccount() {
	aservice.DropTables()
	aservice.CreateTables()
}

func TestOauth(t *testing.T) {
	Convey("Oauth", t, func() {
		recoveryAccount()
		Convey("should create new Oauth and Account, then Link new Oauth, Unlink", func() {
			// Create new
			o := &Oauth{}
			So(o.OnLogin("L2m", "oauth-oid", "oname", ""), ShouldBeNil)
			So(o.Account.Name, ShouldEqual, "Unknown")
			So(len(o.Account.Ones), ShouldEqual, 0)

			a := &Account{}
			So(DB.First(a).Error, ShouldBeNil)
			So(a.Name, ShouldEqual, "Unknown")

			// OnLink
			o2 := &Oauth{}
			So(o2.OnLink(o, "gogogo", "goid", "goname", ""), ShouldBeNil)
			So(DB.NewRecord(o2), ShouldBeFalse)
			var ps []string
			So(o.Account.GetProviders(&ps), ShouldBeNil)
			So(ps, ShouldContain, "L2m")
			So(ps, ShouldContain, "gogogo")

			// Unlink
			So(o2.Unlink("L2m"), ShouldBeNil)
			var ps2 []string
			So(o.Account.GetProviders(&ps2), ShouldBeNil)
			So(ps2, ShouldNotContain, "L2m")
			So(ps2, ShouldContain, "gogogo")
		})
		Convey("should find Oauth with full Account#Account.GetOnes", func() {
			// init
			o := &Oauth{}
			So(o.OnLogin("L2m", "oauth-oid2", "oname2", ""), ShouldBeNil)
			o.Account.Name = "account_name"
			So(DB.Save(&o.Account).Error, ShouldBeNil)

			// Find exist
			o2 := &Oauth{}
			So(o2.OnLogin("L2m", "oauth-oid2", "oname2", ""), ShouldBeNil)
			So(o2.ID, ShouldEqual, o.ID)
			So(o2.Account.Name, ShouldEqual, "account_name")

			// Logoff
			So(o2.Logoff(), ShouldBeNil)
			var count int
			So(DB.Model(&Account{}).Count(&count).Error, ShouldBeNil)
			So(count, ShouldEqual, 0)
			So(DB.Model(&Oauth{}).Count(&count).Error, ShouldBeNil)
			So(count, ShouldEqual, 0)
		})
	})
}

func TestAccount(t *testing.T) {
	Convey("Account", t, func() {
		recoveryAccount()
		Convey("should reg an One", func() {
			addr := "ssssssssss"
			a := &Account{}
			a.Name = "ExistAccount"
			So(DB.Save(a).Error, ShouldBeNil)

			So(a.RegOne(&One{Addr: addr, BaseModel: BaseModel{Name: "NewOne1"}}), ShouldBeNil)

			var one One
			So(DB.Where("addr=? and name=?", addr, "NewOne1").Preload("Owner").First(&one).Error, ShouldBeNil)
			So(one.Owner.ID, ShouldEqual, a.ID)
		})
		Convey("should unreg an One", func() {
			addr := "ssssssssss"
			// init owner
			owner := &Account{}
			owner.Name = "OwnerAccount"
			So(DB.Save(owner).Error, ShouldBeNil)
			// init One
			one0 := &One{Addr: addr, BaseModel: BaseModel{Name: "NewOne2"}}
			So(owner.RegOne(one0), ShouldBeNil)
			// init viewer
			viewer := &Account{}
			viewer.Name = "ViewerAccount"
			So(DB.Save(viewer).Error, ShouldBeNil)
			// viewer view the one
			one := &One{}
			So(one.FindIfOwner(one0.ID, one0.OwnerId), ShouldBeNil)
			So(viewer.ViewOne(one), ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			viewer.Ones = []One{}
			So(DB.Model(viewer).Related(&viewer.Ones, "Ones").Error, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			// onwer unreg the one
			So(owner.RemoveOne(one), ShouldBeNil)
			// validate One
			var result One
			notfound := DB.Where("addr=?", addr).First(&result).RecordNotFound()
			So(notfound, ShouldBeTrue)
			// validate viewer
			viewer.Ones = []One{}
			So(DB.Model(viewer).Related(&viewer.Ones, "Ones").Error, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 0)
			// validate owner
			owner.Ones = []One{}
			So(DB.Model(owner).Related(&owner.Ones, "Ones").Error, ShouldBeNil)
			So(len(owner.Ones), ShouldEqual, 0)
		})
		Convey("should unview an One", func() {
			addr := "ssssssssss"
			// init owner
			owner := &Account{}
			owner.Name = "OwnerAccount"
			So(DB.Save(owner).Error, ShouldBeNil)
			// init One
			one0 := &One{Addr: addr, BaseModel: BaseModel{Name: "NewOne5"}}
			So(owner.RegOne(one0), ShouldBeNil)
			// init viewer
			viewer := &Account{}
			viewer.Name = "ViewerAccount"
			So(DB.Save(viewer).Error, ShouldBeNil)
			// viewer view the one
			one := &One{}
			So(one.Find(one0.ID), ShouldBeNil)
			So(viewer.ViewOne(one), ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			viewer.Ones = []One{}
			So(DB.Model(viewer).Related(&viewer.Ones, "Ones").Error, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			// onwer unreg the one
			So(viewer.RemoveOne(one), ShouldBeNil)
			// validate One
			var result One
			notfound := DB.Where("addr=?", addr).First(&result).RecordNotFound()
			So(notfound, ShouldBeFalse)
			// validate viewer
			viewer.Ones = []One{}
			So(DB.Model(viewer).Related(&viewer.Ones, "Ones").Error, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 0)
			// validate owner
			owner.Ones = []One{}
			So(DB.Model(owner).Related(&owner.Ones, "Ones").Error, ShouldBeNil)
			So(len(owner.Ones), ShouldEqual, 1)
		})
	})
}

func TestOauth_View(t *testing.T) {
	Convey("Oauth_View", t, func() {
		recoveryAccount()
		Convey("should view special room", func() {
			// init oauth
			oauth := &Oauth{}
			So(oauth.OnLogin("p", "id", "pn", ""), ShouldBeNil)

			// reg one then find one will ok
			one := &One{Addr: "addr"}
			So(oauth.Account.RegOne(one), ShouldBeNil)
			var count int
			DB.Model(&One{}).Count(&count)
			So(count, ShouldEqual, 1)
			So(oauth.CanView(one), ShouldBeTrue)
			var aos []AccountOne
			So(one.ViewsByShare(&aos), ShouldBeNil)
			So(len(aos), ShouldEqual, 1)

			// init another oauth, will fail
			oauth = &Oauth{}
			So(oauth.OnLogin("p", "id2", "pn2", ""), ShouldBeNil)
			So(oauth.CanView(one), ShouldBeFalse)
			var aos2 []AccountOne
			So(one.ViewsByShare(&aos2), ShouldBeNil)
			So(len(aos2), ShouldEqual, 1)

			// view the one, will ok
			So(oauth.Account.ViewOne(one), ShouldBeNil)
			So(oauth.CanView(one), ShouldBeTrue)
			var aos3 []AccountOne
			So(one.ViewsByShare(&aos3), ShouldBeNil)
			So(len(aos3), ShouldEqual, 2)
		})
	})
}

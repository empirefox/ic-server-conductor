//GORM_DIALECT=mysql DB_URL="gorm:gorm@/gorm?charset=utf8&parseTime=True"
//GORM_DIALECT=postgres DB_URL="postgres://postgres:6Vno3r3gH9sZHSxo@localhost/ic_signal_test?sslmode=disable"
//GORM_DIALECT=sqlite3 DB_URL=/tmp/gorm.DB go test
package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	DB.LogMode(true)
}

func recoveryAccount() {
	ao := &AccountOne{}
	one := &One{}
	oauth := &Oauth{}

	DB.DropTableIfExists(ao)
	DB.DropTableIfExists(oauth)
	DB.DropTableIfExists(one)
	DB.DropTableIfExists(&Account{})

	DB.CreateTable(ao)
	DB.CreateTable(&Account{})
	DB.CreateTable(one)
	DB.CreateTable(oauth)

	ao.AddForeignKey()
	one.AddForeignKey()
	oauth.AddForeignKey()
}

func TestOauth_OnOid(t *testing.T) {
	Convey("Oauth", t, func() {
		recoveryAccount()
		Convey("should create new Oauth and Account", func() {
			var o Oauth
			err := o.OnOid("L2m", "oauth-oid")
			So(err, ShouldBeNil)
			So(o.Account.Name, ShouldEqual, "L2moauth-oid")
			So(len(o.Account.Ones), ShouldEqual, 0)

			var a Account
			err = DB.First(&a).Error
			So(err, ShouldBeNil)
			So(a.Name, ShouldEqual, "L2moauth-oid")
		})
		Convey("should find Oauth with full Account", func() {
			var o Oauth
			err := o.OnOid("L2m", "oauth-oid")
			So(err, ShouldBeNil)
			So(o.Account.Name, ShouldEqual, "L2moauth-oid")
			So(len(o.Account.Ones), ShouldEqual, 0)

			var o2 Oauth
			err = o2.OnOid("L2m", "oauth-oid")
			So(err, ShouldBeNil)
			So(o2, ShouldResemble, o)
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
			err := DB.Save(a).Error
			So(err, ShouldBeNil)

			err = a.RegOne(&One{SecretAddress: addr, BaseModel: BaseModel{Name: "NewOne1"}})
			So(err, ShouldBeNil)

			var one One
			err = DB.Where("secret_address=? and name=?", addr, "NewOne1").Preload("Owner").First(&one).Error
			So(err, ShouldBeNil)
			So(one.Owner.ID, ShouldEqual, a.ID)
		})
		Convey("should unreg an One", func() {
			addr := "ssssssssss"
			// init owner
			owner := &Account{}
			owner.Name = "OwnerAccount"
			err := DB.Save(owner).Error
			So(err, ShouldBeNil)
			// init One
			err = owner.RegOne(&One{SecretAddress: addr, BaseModel: BaseModel{Name: "NewOne2"}})
			So(err, ShouldBeNil)
			// init viewer
			viewer := &Account{}
			viewer.Name = "ViewerAccount"
			err = DB.Save(viewer).Error
			So(err, ShouldBeNil)
			// viewer view the one
			one, err := FindOne([]byte(addr))
			So(err, ShouldBeNil)
			err = viewer.ViewOne(one)
			So(err, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			viewer.Ones = []One{}
			err = DB.Model(viewer).Related(&viewer.Ones, "Ones").Error
			So(err, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			// onwer unreg the one
			err = owner.RemoveOne(one)
			So(err, ShouldBeNil)
			// validate One
			var result One
			notfound := DB.Where("secret_address=?", addr).First(&result).RecordNotFound()
			So(notfound, ShouldBeTrue)
			// validate viewer
			viewer.Ones = []One{}
			err = DB.Model(viewer).Related(&viewer.Ones, "Ones").Error
			So(err, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 0)
			// validate owner
			owner.Ones = []One{}
			err = DB.Model(owner).Related(&owner.Ones, "Ones").Error
			So(err, ShouldBeNil)
			So(len(owner.Ones), ShouldEqual, 0)
		})
		Convey("should unview an One", func() {
			addr := "ssssssssss"
			// init owner
			owner := &Account{}
			owner.Name = "OwnerAccount"
			err := DB.Save(owner).Error
			So(err, ShouldBeNil)
			// init One
			err = owner.RegOne(&One{SecretAddress: addr, BaseModel: BaseModel{Name: "NewOne5"}})
			So(err, ShouldBeNil)
			// init viewer
			viewer := &Account{}
			viewer.Name = "ViewerAccount"
			err = DB.Save(viewer).Error
			So(err, ShouldBeNil)
			// viewer view the one
			one, err := FindOne([]byte(addr))
			So(err, ShouldBeNil)
			err = viewer.ViewOne(one)
			So(err, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			viewer.Ones = []One{}
			err = DB.Model(viewer).Related(&viewer.Ones, "Ones").Error
			So(err, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 1)
			// onwer unreg the one
			err = viewer.RemoveOne(one)
			So(err, ShouldBeNil)
			// validate One
			var result One
			notfound := DB.Where("secret_address=?", addr).First(&result).RecordNotFound()
			So(notfound, ShouldBeFalse)
			// validate viewer
			viewer.Ones = []One{}
			err = DB.Model(viewer).Related(&viewer.Ones, "Ones").Error
			So(err, ShouldBeNil)
			So(len(viewer.Ones), ShouldEqual, 0)
			// validate owner
			owner.Ones = []One{}
			err = DB.Model(owner).Related(&owner.Ones, "Ones").Error
			So(err, ShouldBeNil)
			So(len(owner.Ones), ShouldEqual, 1)
		})
	})
}

func TestFindOne(t *testing.T) {
	Convey("FindOne", t, func() {
		recoveryAccount()
		Convey("should find an Oauth", func() {
			addr := "ssssssssss"
			a := &Account{}
			a.Name = "ExistAccount"
			err := DB.Save(a).Error
			So(err, ShouldBeNil)

			err = a.RegOne(&One{SecretAddress: addr, BaseModel: BaseModel{Name: "NewOne3"}})
			So(err, ShouldBeNil)

			one, err := FindOne([]byte(addr))
			So(err, ShouldBeNil)
			So(one.OwnerId, ShouldEqual, a.ID)
			So(one.SecretAddress, ShouldEqual, addr)
			So(one.Name, ShouldEqual, "NewOne3")
			So(one.Accounts[0].ID, ShouldEqual, a.ID)
		})
	})
}

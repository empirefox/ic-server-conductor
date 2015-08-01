package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"

	"github.com/empirefox/gin-oauth2"
	"github.com/empirefox/gotool/dp"
	"github.com/empirefox/gotool/paas"
	"github.com/empirefox/ic-server-ws-signal/account"
	. "github.com/empirefox/ic-server-ws-signal/connections"
	"github.com/empirefox/ic-server-ws-signal/invite"
	"github.com/empirefox/ic-server-ws-signal/utils"
)

const (
	SK_CALL = "call"
	SK_MANY = "many"
	SK_SYS  = "system"
)

type Server struct {
	ClaimsKey    string
	UserKey      string
	Keys         map[string][]byte
	Hub          *Hub
	OauthConfig  *goauth.Config
	OauthJson    []byte
	IsDevMode    bool
	ValidateGets map[string]string
}

func (s *Server) SecureWs(c *gin.Context) {
	if strings.EqualFold(c.Request.URL.Scheme, "ws") {
		glog.Infoln("insecure:", *c.Request.URL)
		c.Abort()
		return
	}
}

func (s *Server) GetApiProviders(c *gin.Context) {
	var ops OauthProviders
	if err := ops.All(); err != nil {
		glog.Errorln(err)
		c.AbortWithStatus(http.StatusNotImplemented)
		return
	}
	c.JSON(http.StatusOK, ops)
}

func (s *Server) GetSystemData(c *gin.Context) {
	e := `sys-data`
	c.Header("Etag", e)
	c.Header("Cache-Control", "max-age=2592000") // 30 days

	if match := c.Request.Header.Get("If-None-Match"); strings.Contains(match, e) {
		c.Writer.WriteHeader(http.StatusNotModified)
		return
	}

	data, _ := json.Marshal(gin.H{
		"DevProd":   dp.Mode,
		"ApiDomain": paas.SubDomain,
	})
	c.String(http.StatusOK, fmt.Sprintf(`var ApiData=%s`, data))
}

func (s *Server) newManyToken(c *gin.Context, ouser interface{}) {
	u, err := json.Marshal(ouser)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	token.Header["kid"] = SK_MANY
	token.Claims[s.UserKey] = string(u)
	token.Claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
	tokenString, err := token.SignedString(s.Keys[SK_MANY])
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return err
	}

	c.JSON(http.StatusOK, utils.OK(tokenString))
}

func (s *Server) GetLogin(c *gin.Context) {
	claims := c.Keys[s.ClaimsKey]
	ouser := &account.Oauth{}
	if err := ouser.OnOid(claims["provider"], claims["oid"]); err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	s.newManyToken(c, ouser)
}

func (s *Server) GetCheckToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.Keys[s.ClaimsKey]
		exp := claims["exp"].(int64)
		update := time.Now().Add(time.Minute * 30).Unix()
		if exp > update {
			c.JSON(http.StatusOK, "{}")
			return
		}
		s.newManyToken(c, claims[s.UserKey])
	}
}

func (s *Server) Run() error {
	dp.SetDevMode(paas.IsDevMode)
	router := gin.Default()

	router.Use(secure.Secure(secure.Options{
		SSLRedirect: true,
		SSLProxyHeaders: map[string]string{
			"X-Forwarded-Proto": "https",
			"Upgrade":           "websocket",
		},
		IsDevelopment: s.IsDevMode,
	}))

	router.Use(s.SecureWs)

	router.Use(static.Serve("/", static.LocalFile("public", true)))
	// peer from MANY client
	router.GET("/sys-data.js", s.GetSystemData)

	// login page will be find in static serve
	// logout will proccess some logic
	router.GET("/auth/oauths", func(c *gin.Context) { c.Writer.Write(s.OauthJson) })

	sys := router.Group("/sys", s.Auth(SK_SYS))
	sys.POST("/oauth", SaveOauth)

	call := router.Group("/call", s.Auth(SK_CALL))
	call.GET("/login", s.GetLogin)
	call.GET("/providers", s.GetApiProviders)

	// peer from ONE client
	one := router.Group("/one")
	one.GET("/ctrl", HandleOneCtrl(s.Hub))
	one.GET("/signaling/:reciever", HandleOneSignaling(s.Hub))

	// websocket
	// peer from MANY client
	manyws := router.Group("/mws")
	manyws.GET("/ctrl", HandleManyCtrl(s.Hub))
	manyws.GET("/signaling", HandleManySignaling(s.Hub))

	// rest
	many := router.Group("/many", s.Auth(SK_MANY), s.CheckManyUser)
	many.GET("/logoff", HandleManyLogoff(s.Hub))
	many.POST("/reg-room", HandleManyRegRoom(s.Hub))
	many.GET("/invite-code/:room", invite.HandleManyGetInviteCode(s.Hub))
	many.GET("/invite/:room/:code", invite.HandleManyOnInvite(s.Hub))
	many.POST("/refresh-token", func(c *gin.Context) { c.JSON(http.StatusOK, "") })

	return router.Run(paas.GetBindAddr())
}

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/itsjamie/gin-cors"

	"github.com/empirefox/gotool/dp"
	"github.com/empirefox/gotool/paas"
	"github.com/empirefox/ic-server-conductor/account"
	"github.com/empirefox/ic-server-conductor/conn"
	"github.com/empirefox/ic-server-conductor/conn/many"
	"github.com/empirefox/ic-server-conductor/conn/one"
	"github.com/empirefox/ic-server-conductor/invite"
)

const (
	SK_CALL = "call"
	SK_MANY = "many"
	SK_SYS  = "system"
)

type Server struct {
	ClaimsKey       string
	UserKey         string
	Keys            map[string][]byte
	Hub             conn.Hub
	OauthJson       []byte
	IsDevMode       bool
	OnEngineCreated func(*gin.Engine)
}

func (s *Server) SecureWs(c *gin.Context) {
	if strings.EqualFold(c.Request.URL.Scheme, "ws") {
		glog.Infoln("insecure:", *c.Request.URL)
		c.Abort()
		return
	}
}

func (s *Server) GetApiProviders(c *gin.Context) {
	var ops account.OauthProviders
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
		return
	}

	c.String(http.StatusOK, tokenString)
}

func (s *Server) GetLogin(c *gin.Context) {
	claims := c.Keys[s.ClaimsKey].(map[string]interface{})
	ouser := &account.Oauth{}
	if err := ouser.OnOid(claims["provider"].(string), claims["oid"].(string)); err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	s.newManyToken(c, ouser)
}

func (s *Server) GetCheckToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.Keys[s.ClaimsKey].(map[string]interface{})
		exp := claims["exp"].(int64)
		update := time.Now().Add(time.Minute * 30).Unix()
		if exp > update {
			c.JSON(http.StatusOK, "{}")
			return
		}
		s.newManyToken(c, claims[s.UserKey])
	}
}

func (s *Server) PostClearTables(c *gin.Context) {
	allow, _ := strconv.ParseBool(os.Getenv("ALLOW_CLEAR_TABLES"))
	if !allow {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err := account.ClearTables(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, "")
}

func (s *Server) Run() error {
	dp.SetDevMode(paas.IsDevMode)
	corsMiddleWare := cors.Middleware(cors.Config{
		Origins:         "*",
		Methods:         "GET, PUT, POST, DELETE",
		RequestHeaders:  "Origin, Authorization, Content-Type",
		ExposedHeaders:  "",
		MaxAge:          48 * time.Hour,
		Credentials:     false,
		ValidateHeaders: false,
	})

	router := gin.Default()
	if s.OnEngineCreated != nil {
		s.OnEngineCreated(router)
	}

	router.Use(secure.Secure(secure.Options{
		SSLRedirect: true,
		SSLProxyHeaders: map[string]string{
			"X-Forwarded-Proto": "https",
			"Upgrade":           "websocket",
		},
		IsDevelopment: s.IsDevMode,
	}))

	router.Use(s.SecureWs)

	// peer from MANY client
	router.GET("/sys-data.js", s.GetSystemData)
	router.GET("auth/oauths", corsMiddleWare, func(c *gin.Context) { c.Writer.Write(s.OauthJson) })

	sys := router.Group("/sys", s.Auth(SK_SYS))
	sys.POST("/clear-tables", s.PostClearTables)
	sys.POST("/oauth", s.SaveOauth)

	call := router.Group("/call", s.Auth(SK_CALL))
	call.GET("/login", s.GetLogin)
	call.GET("/providers", s.GetApiProviders)

	// peer from ONE client
	ro := router.Group("/one")
	ro.GET("/ctrl", one.HandleOneCtrl(s.Hub))
	ro.GET("/signaling/:reciever", one.HandleOneSignaling(s.Hub))

	// websocket
	// peer from MANY client
	manyws := router.Group("/mws")
	manyws.GET("/ctrl", many.HandleManyCtrl(s.Hub, s.Keys[SK_MANY]))
	manyws.GET("/signaling", many.HandleManySignaling(s.Hub, s.Keys[SK_MANY]))

	// rest
	rm := router.Group("/many", corsMiddleWare, s.Auth(SK_MANY), s.CheckManyUser)

	rm.DELETE("/logoff", many.HandleManyLogoff(s.Hub))
	rm.POST("/reg-room", many.HandleManyRegRoom(s.Hub))
	rm.GET("/invite-code/:room", invite.HandleManyGetInviteCode(s.Hub))
	rm.GET("/invite/:room/:code", invite.HandleManyOnInvite(s.Hub))
	rm.POST("/refresh-token", func(c *gin.Context) { c.JSON(http.StatusOK, "") })

	return router.Run(paas.GetBindAddr())
}

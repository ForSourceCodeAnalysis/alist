package server

import (
	"github.com/alist-org/alist/v3/cmd/flags"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/message"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/alist-org/alist/v3/server/handles"
	"github.com/alist-org/alist/v3/server/middlewares"
	"github.com/alist-org/alist/v3/server/static"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Init(e *gin.Engine) {
	if !utils.SliceContains([]string{"", "/"}, conf.URL.Path) {
		e.GET("/", func(c *gin.Context) {
			c.Redirect(302, conf.URL.Path)
		})
	}
	Cors(e)
	//创建分组路由
	g := e.Group(conf.URL.Path)
	if conf.Conf.Scheme.HttpPort != -1 && conf.Conf.Scheme.HttpsPort != -1 && conf.Conf.Scheme.ForceHttps {
		g.Use(middlewares.ForceHttps) //强制启用https
	}
	g.Any("/ping", func(c *gin.Context) { //用于测试网站是否可以正常访问，类似于ping命令
		c.String(200, "pong")
	})
	g.GET("/favicon.ico", handles.Favicon) //网站图标
	g.GET("/robots.txt", handles.Robots)   //爬虫访问控制
	g.GET("/i/:link_name", handles.Plist)  //
	common.SecretKey = []byte(conf.Conf.JwtSecret)
	g.Use(middlewares.StoragesLoaded)
	if conf.Conf.MaxConnections > 0 { //并发数限制
		g.Use(middlewares.MaxAllowed(conf.Conf.MaxConnections))
	}
	WebDav(g.Group("/dav")) //webdav服务

	//下载，代理下载
	//中间件会校验要下载的文件是否配置了元数据（默认是没有的），如果没有，就可以下载
	//注意：由于metas配置是针对游客的，所以这个路由并没有校验权限；如果知道文件路径，而又没有配置元数据，
	//     游客是可以通过此路由直接下载文件的
	g.GET("/d/*path", middlewares.Down, handles.Down)
	g.GET("/p/*path", middlewares.Down, handles.Proxy)
	g.HEAD("/d/*path", middlewares.Down, handles.Down)
	g.HEAD("/p/*path", middlewares.Down, handles.Proxy)

	api := g.Group("/api") //api
	auth := api.Group("", middlewares.Auth)
	webauthn := api.Group("/authn", middlewares.Authn)

	api.POST("/auth/login", handles.Login)
	api.POST("/auth/login/hash", handles.LoginHash)
	auth.GET("/me", handles.CurrentUser)
	auth.POST("/me/update", handles.UpdateCurrent)
	auth.POST("/auth/2fa/generate", handles.Generate2FA)
	auth.POST("/auth/2fa/verify", handles.Verify2FA)

	// auth
	api.GET("/auth/sso", handles.SSOLoginRedirect)
	api.GET("/auth/sso_callback", handles.SSOLoginCallback)

	//webauthn
	webauthn.GET("/webauthn_begin_registration", handles.BeginAuthnRegistration)
	webauthn.POST("/webauthn_finish_registration", handles.FinishAuthnRegistration)
	webauthn.GET("/webauthn_begin_login", handles.BeginAuthnLogin)
	webauthn.POST("/webauthn_finish_login", handles.FinishAuthnLogin)
	webauthn.POST("/delete_authn", handles.DeleteAuthnLogin)
	webauthn.GET("/getcredentials", handles.GetAuthnCredentials)

	// no need auth
	public := api.Group("/public")
	public.Any("/settings", handles.PublicSettings)

	_fs(auth.Group("/fs"))
	admin(auth.Group("/admin", middlewares.AuthAdmin))
	if flags.Dev {
		dev(g.Group("/dev"))
	}
	static.Static(g, func(handlers ...gin.HandlerFunc) {
		e.NoRoute(handlers...)
	})
}

func admin(g *gin.RouterGroup) {
	meta := g.Group("/meta")
	meta.GET("/list", handles.ListMetas)
	meta.GET("/get", handles.GetMeta)
	meta.POST("/create", handles.CreateMeta)
	meta.POST("/update", handles.UpdateMeta)
	meta.POST("/delete", handles.DeleteMeta)

	user := g.Group("/user")
	user.GET("/list", handles.ListUsers)
	user.GET("/get", handles.GetUser)
	user.POST("/create", handles.CreateUser)
	user.POST("/update", handles.UpdateUser)
	user.POST("/cancel_2fa", handles.Cancel2FAById)
	user.POST("/delete", handles.DeleteUser)
	user.POST("/del_cache", handles.DelUserCache)

	storage := g.Group("/storage")
	storage.GET("/list", handles.ListStorages)
	storage.GET("/get", handles.GetStorage)
	storage.POST("/create", handles.CreateStorage)
	storage.POST("/update", handles.UpdateStorage)
	storage.POST("/delete", handles.DeleteStorage)
	storage.POST("/enable", handles.EnableStorage)
	storage.POST("/disable", handles.DisableStorage)
	storage.POST("/load_all", handles.LoadAllStorages)

	driver := g.Group("/driver")
	driver.GET("/list", handles.ListDriverInfo)
	driver.GET("/names", handles.ListDriverNames)
	driver.GET("/info", handles.GetDriverInfo)

	setting := g.Group("/setting")
	setting.GET("/get", handles.GetSetting)
	setting.GET("/list", handles.ListSettings)
	setting.POST("/save", handles.SaveSettings)
	setting.POST("/delete", handles.DeleteSetting)
	setting.POST("/reset_token", handles.ResetToken)
	setting.POST("/set_aria2", handles.SetAria2)
	setting.POST("/set_qbit", handles.SetQbittorrent)

	task := g.Group("/task")
	handles.SetupTaskRoute(task)

	ms := g.Group("/message")
	ms.POST("/get", message.HttpInstance.GetHandle)
	ms.POST("/send", message.HttpInstance.SendHandle)

	index := g.Group("/index")
	index.POST("/build", middlewares.SearchIndex, handles.BuildIndex)
	index.POST("/update", middlewares.SearchIndex, handles.UpdateIndex)
	index.POST("/stop", middlewares.SearchIndex, handles.StopIndex)
	index.POST("/clear", middlewares.SearchIndex, handles.ClearIndex)
	index.GET("/progress", middlewares.SearchIndex, handles.GetProgress)
}

func _fs(g *gin.RouterGroup) {
	g.Any("/list", handles.FsList)
	g.Any("/search", middlewares.SearchIndex, handles.Search)
	g.Any("/get", handles.FsGet)
	g.Any("/other", handles.FsOther)
	g.Any("/dirs", handles.FsDirs)
	g.POST("/mkdir", handles.FsMkdir)
	g.POST("/rename", handles.FsRename)
	g.POST("/batch_rename", handles.FsBatchRename)
	g.POST("/regex_rename", handles.FsRegexRename)
	g.POST("/move", handles.FsMove)
	g.POST("/recursive_move", handles.FsRecursiveMove)
	g.POST("/copy", handles.FsCopy)
	g.POST("/remove", handles.FsRemove)
	g.POST("/remove_empty_directory", handles.FsRemoveEmptyDirectory)
	g.PUT("/put", middlewares.FsUp, handles.FsStream)
	g.PUT("/form", middlewares.FsUp, handles.FsForm)
	g.POST("/link", middlewares.AuthAdmin, handles.Link)
	g.POST("/add_aria2", handles.AddAria2)
	g.POST("/add_qbit", handles.AddQbittorrent)
}

// 中间件
func Cors(r *gin.Engine) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"*"}
	config.AllowMethods = []string{"*"}
	r.Use(cors.New(config))
}

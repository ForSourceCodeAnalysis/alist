package middlewares

import (
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

// 中间件
func StoragesLoaded(c *gin.Context) {
	if conf.StoragesLoaded { //挂载的存储已经加载完成了，继续
		c.Next()
	} else {
		//如果是请求网站首页或图标，继续
		if utils.SliceContains([]string{"", "/", "/favicon.ico"}, c.Request.URL.Path) {
			c.Next()
			return
		}
		//如果是请求的静态资源，继续
		paths := []string{"/assets", "/images", "/streamer", "/static"}
		for _, path := range paths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}
		//请求的其他路由，在存储未加载完成的情况下，进行阻止，提示用户稍后访问
		common.ErrorStrResp(c, "Loading storage, please wait", 500)
		c.Abort()
	}
}

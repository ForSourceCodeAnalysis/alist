package middlewares

import (
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"

	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/sign"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func Down(c *gin.Context) {
	rawPath := parsePath(c.Param("path"))
	c.Set("path", rawPath)
	meta, err := op.GetNearestMeta(rawPath) //获取元数据
	if err != nil {
		if !errors.Is(errors.Cause(err), errs.MetaNotFound) {
			common.ErrorResp(c, err, 500, true)
			return
		}
	}
	c.Set("meta", meta)
	// verify sign
	if needSign(meta, rawPath) { //是否需要验证签名
		s := c.Query("sign")
		err = sign.Verify(rawPath, strings.TrimSuffix(s, "/"))
		if err != nil {
			common.ErrorResp(c, err, 401)
			c.Abort()
			return
		}
	}
	c.Next()
}

// TODO: implement
// path maybe contains # ? etc.
func parsePath(path string) string {
	return utils.FixAndCleanPath(path)
}

func needSign(meta *model.Meta, path string) bool {
	if setting.GetBool(conf.SignAll) { //配置了签名所有
		return true
	}
	if common.IsStorageSignEnabled(path) { //当前存储需要签名
		return true
	}
	if meta == nil || meta.Password == "" { //如果元数据为空或元数据没有配置密码就不会验证签名
		return false
	}
	if !meta.PSub && path != meta.Path { //如果密码没有应用到子文件夹，且当前路径与元数据路径不一致则不验证签名
		return false
	}
	return true
}

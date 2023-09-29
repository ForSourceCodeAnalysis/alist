package handles

import (
	"net/url"
	stdpath "path"
	"strconv"
	"time"

	"github.com/alist-org/alist/v3/internal/fs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

func FsStream(c *gin.Context) {
	//目的目录，非源目录 例如挂载了/baidu，上传文件filename，则path=/baidu/filename
	path := c.GetHeader("File-Path")
	path, err := url.PathUnescape(path) //
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	asTask := c.GetHeader("As-Task") == "true" //是否按异步任务处理
	user := c.MustGet("user").(*model.User)    //获取用户，中间件会自动解析token并设置user
	path, err = user.JoinPath(path)
	if err != nil {
		common.ErrorResp(c, err, 403)
		return
	}

	dir, name := stdpath.Split(path)
	sizeStr := c.GetHeader("Content-Length")
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	//创建文件流
	stream := &model.FileStream{
		Obj: &model.Object{
			Name:     name,
			Size:     size,
			Modified: time.Now(), //这里把修改时间设置为了当前时间，而非源文件最后一次修改时间
		},
		ReadCloser:   c.Request.Body,
		Mimetype:     c.GetHeader("Content-Type"),
		WebPutAsTask: asTask,
	}
	if asTask {
		err = fs.PutAsTask(dir, stream)
	} else {
		err = fs.PutDirectly(c, dir, stream, true)
	}
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

func FsForm(c *gin.Context) {
	path := c.GetHeader("File-Path")
	path, err := url.PathUnescape(path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	asTask := c.GetHeader("As-Task") == "true"
	user := c.MustGet("user").(*model.User)
	path, err = user.JoinPath(path)
	if err != nil {
		common.ErrorResp(c, err, 403)
		return
	}
	storage, err := fs.GetStorage(path, &fs.GetStoragesArgs{})
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if storage.Config().NoUpload {
		common.ErrorStrResp(c, "Current storage doesn't support upload", 405)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	f, err := file.Open()
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	dir, name := stdpath.Split(path)
	stream := &model.FileStream{
		Obj: &model.Object{
			Name:     name,
			Size:     file.Size,
			Modified: time.Now(),
		},
		ReadCloser:   f,
		Mimetype:     file.Header.Get("Content-Type"),
		WebPutAsTask: false,
	}
	if asTask {
		err = fs.PutAsTask(dir, stream)
	} else {
		err = fs.PutDirectly(c, dir, stream, true)
	}
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

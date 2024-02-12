package handles

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

func Favicon(c *gin.Context) {
	c.Redirect(302, setting.GetStr(conf.Favicon))
}

func Robots(c *gin.Context) {
	c.String(200, setting.GetStr(conf.RobotsTxt))
}

// 这个路由不知道是干什么的，也不知道用在什么场景下
// 根据内容分析就是展示特性列表的
func Plist(c *gin.Context) {
	//参数link_name必须是base64加密格式
	linkNameB64 := strings.TrimSuffix(c.Param("link_name"), ".plist")
	linkName, err := utils.SafeAtob(linkNameB64)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	//且解密后，必须有且仅有一个/分割
	linkNameSplit := strings.Split(linkName, "/")
	if len(linkNameSplit) != 2 {
		common.ErrorStrResp(c, "malformed link", 400)
		return
	}
	linkEncode := linkNameSplit[0]
	linkStr, err := url.PathUnescape(linkEncode)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	link, err := url.Parse(linkStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	//这句话应该是误打了，下面在fullName使用前又重新赋值了
	fullName := c.Param("name")
	Url := link.String()
	Url = strings.ReplaceAll(Url, "<", "[")
	Url = strings.ReplaceAll(Url, ">", "]")
	nameEncode := linkNameSplit[1]
	fullName, err = url.PathUnescape(nameEncode)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	name := fullName
	identifier := fmt.Sprintf("ci.nn.%s", url.PathEscape(fullName))
	sep := "@"
	if strings.Contains(fullName, sep) {
		ss := strings.Split(fullName, sep)
		name = strings.Join(ss[:len(ss)-1], sep)
		identifier = ss[len(ss)-1]
	}

	name = strings.ReplaceAll(name, "<", "[")
	name = strings.ReplaceAll(name, ">", "]")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>items</key>
        <array>
            <dict>
                <key>assets</key>
                <array>
                    <dict>
                        <key>kind</key>
                        <string>software-package</string>
                        <key>url</key>
                        <string><![CDATA[%s]]></string>
                    </dict>
                </array>
                <key>metadata</key>
                <dict>
                    <key>bundle-identifier</key>
					<string>%s</string>
					<key>bundle-version</key>
                    <string>4.4</string>
                    <key>kind</key>
                    <string>software</string>
                    <key>title</key>
                    <string>%s</string>
                </dict>
            </dict>
        </array>
    </dict>
</plist>`, Url, identifier, name)
	c.Header("Content-Type", "application/xml;charset=utf-8")
	c.Status(200)
	_, _ = c.Writer.WriteString(plist)
}

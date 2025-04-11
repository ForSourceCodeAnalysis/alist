package extensions

import (
	"github.com/alist-org/alist/v3/extensions/backup"
	"github.com/alist-org/alist/v3/extensions/queue"
	"github.com/gin-gonic/gin"
)

// RegisterRoute register extension routes
func RegisterRoute(g map[string]*gin.RouterGroup) {
	backup.Route(g["backup"])
}

// Init extension
func Init() {
	queue.Init()
	backup.BackupInit()

	queue.Start()
}

package backup

import (
	"github.com/gin-gonic/gin"
)

// Route register route
func Route(g *gin.RouterGroup) {
	b := g.Group("/backup")
	b.GET("/list", listBackup)
	b.GET("/get", getBackup)
	b.POST("/create", createBackup)
	b.POST("/update", updateBackup)
	b.POST("/delete", deleteBackup)
	b.GET("/files/:id", getBackupFiles)
}

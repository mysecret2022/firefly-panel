package panel

import (
	"sun-panel/api/api_v1"
	"sun-panel/api/api_v1/middleware"

	"github.com/gin-gonic/gin"
)

func InitBackup(router *gin.RouterGroup) {
	api := api_v1.ApiGroupApp.ApiPanel.BackupApi
	r := router.Group("", middleware.LoginInterceptor)
	{
		r.GET("/panel/backup/export", api.ExportBackup)
		r.GET("/panel/backup/exportJson", api.ExportJson)
		r.POST("/panel/backup/restore", api.RestoreBackup)
	}
}

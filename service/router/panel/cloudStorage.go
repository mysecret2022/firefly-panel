package panel

import (
	"sun-panel/api/api_v1"
	"sun-panel/api/api_v1/middleware"

	"github.com/gin-gonic/gin"
)

func InitCloudStorage(router *gin.RouterGroup) {
	api := api_v1.ApiGroupApp.ApiPanel.CloudStorageApi
	r := router.Group("", middleware.LoginInterceptor)
	{
		r.GET("/panel/storage/getConfig", api.GetConfig)
		r.POST("/panel/storage/saveConfig", api.SaveConfig)
		r.POST("/panel/storage/test", api.TestConnection)
		r.POST("/panel/storage/uploadBackup", api.UploadEncryptedBackup)
		r.POST("/panel/storage/restoreBackup", api.RestoreEncryptedBackup)
		r.GET("/panel/storage/listBackups", api.ListCloudBackups)
		r.GET("/panel/storage/listImages", api.ListCloudImages)
		r.POST("/panel/storage/syncImage", api.SyncImageToLocal)
		r.POST("/panel/storage/syncAllImages", api.SyncAllImagesToLocal)
		r.POST("/panel/storage/deleteImages", api.DeleteCloudImages)
		r.POST("/panel/storage/clearImages", api.ClearAllCloudImages)
	}
}

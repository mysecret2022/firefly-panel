package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sun-panel/api/api_v1/common/apiData/commonApiStructs"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/cloudClient"
	"sun-panel/lib/cmn"
	"sun-panel/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
)

type FileApi struct{}

// 获取已启用的云存储客户端
func getCloudClient() (cloudClient.CloudStorageClient, error) {
	m := models.SystemSetting{}
	val, err := m.Get("cloud_storage_config")
	if err != nil || val == "" {
		return nil, fmt.Errorf("云存储未配置或未启用")
	}
	var cfg cloudClient.CloudStorageConfig
	if err := json.Unmarshal([]byte(val), &cfg); err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("云存储已停用，请在管理员面板开启")
	}
	return cloudClient.CreateCloudClient(cfg)
}

// 将本地文件推送到云端图床 (支持分别存放到 wallpapers 与 icons 目录)
func uploadLocalFileToCloud(client cloudClient.CloudStorageClient, localPath, fileName, ext string, fileType int) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	mime := "image/png"
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	case ".svg":
		mime = "image/svg+xml"
	case ".ico":
		mime = "image/x-icon"
	}

	subDir := "images/wallpapers/"
	if fileType == 1 {
		subDir = "images/icons/"
	}

	_, err = client.UploadFile(subDir+fileName, data, mime)
	return err
}

func (a *FileApi) UploadImg(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	configUpload := global.Config.GetValueString("base", "source_path")
	f, err := c.FormFile("imgfile")
	if err != nil {
		apiReturn.ErrorByCode(c, 1300)
		return
	}

	fileExt := strings.ToLower(path.Ext(f.Filename))
	agreeExts := []string{
		".png",
		".jpg",
		".gif",
		".jpeg",
		".webp",
		".svg",
		".ico",
	}

	if !cmn.InArray(agreeExts, fileExt) {
		apiReturn.ErrorByCode(c, 1301)
		return
	}

	fileType := 0
	if ftStr := c.PostForm("fileType"); ftStr != "" {
		if ft, err := strconv.Atoi(ftStr); err == nil {
			fileType = ft
		}
	}

	fileName := cmn.Md5(fmt.Sprintf("%s%s", f.Filename, time.Now().String()))
	fildDir := fmt.Sprintf("%s/%d/%d/%d/", configUpload, time.Now().Year(), time.Now().Month(), time.Now().Day())
	isExist, _ := cmn.PathExists(fildDir)
	if !isExist {
		os.MkdirAll(fildDir, os.ModePerm)
	}
	filepath := fmt.Sprintf("%s%s%s", fildDir, fileName, fileExt)
	c.SaveUploadedFile(f, filepath)

	// 异步双写到云端图床 (若配置了开启双写)
	go func(srcFile string, imgName string, ext string, fType int) {
		client, err := getCloudClient()
		if err != nil {
			return
		}
		_ = uploadLocalFileToCloud(client, srcFile, imgName, ext, fType)
	}(filepath, fileName+fileExt, fileExt, fileType)

	// 向数据库添加记录
	mFile := models.File{}
	mFile.AddFileWithCategory(userInfo.ID, f.Filename, fileExt, filepath, fileType)
	apiReturn.SuccessData(c, gin.H{
		"imageUrl": filepath[1:],
	})
}

func (a *FileApi) UploadFiles(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	configUpload := global.Config.GetValueString("base", "source_path")

	form, err := c.MultipartForm()
	if err != nil {
		apiReturn.ErrorByCode(c, 1300)
		return
	}
	files := form.File["files[]"]
	errFiles := []string{}
	succMap := map[string]string{}

	fileType := 0
	if fts, ok := form.Value["fileType"]; ok && len(fts) > 0 {
		if ft, err := strconv.Atoi(fts[0]); err == nil {
			fileType = ft
		}
	}

	for _, f := range files {
		fileExt := strings.ToLower(path.Ext(f.Filename))
		fileName := cmn.Md5(fmt.Sprintf("%s%s", f.Filename, time.Now().String()))
		fildDir := fmt.Sprintf("%s/%d/%d/%d/", configUpload, time.Now().Year(), time.Now().Month(), time.Now().Day())
		isExist, _ := cmn.PathExists(fildDir)
		if !isExist {
			os.MkdirAll(fildDir, os.ModePerm)
		}
		filepath := fmt.Sprintf("%s%s%s", fildDir, fileName, fileExt)
		if c.SaveUploadedFile(f, filepath) != nil {
			errFiles = append(errFiles, f.Filename)
		} else {
			mFile := models.File{}
			mFile.AddFileWithCategory(userInfo.ID, f.Filename, fileExt, filepath, fileType)
			succMap[f.Filename] = filepath[1:]
		}
	}

	apiReturn.SuccessData(c, gin.H{
		"succMap":  succMap,
		"errFiles": errFiles,
	})
}

func (a *FileApi) GetList(c *gin.Context) {
	list := []models.File{}
	userInfo, _ := base.GetCurrentUserInfo(c)
	var count int64

	dbQuery := global.Db.Model(&models.File{}).Order("created_at desc")

	// 普通用户必须且只能看自己的图片
	if userInfo.Role != 1 {
		dbQuery = dbQuery.Where("user_id = ?", userInfo.ID)
	}

	// 支持根据分类筛选 (0: 壁纸/大图, 1: 应用图标, -1 或不传为全部)
	if ftStr := c.Query("fileType"); ftStr != "" {
		if ft, err := strconv.Atoi(ftStr); err == nil && ft >= 0 {
			dbQuery = dbQuery.Where("file_type = ?", ft)
		}
	}

	if err := dbQuery.Find(&list).Count(&count).Error; err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}

	data := []map[string]interface{}{}
	for _, v := range list {
		src := v.Src
		if strings.HasPrefix(src, ".") {
			src = src[1:]
		}
		data = append(data, map[string]interface{}{
			"src":        src,
			"fileName":   v.FileName,
			"id":         v.ID,
			"fileType":   v.FileType,
			"syncStatus": v.SyncStatus,
			"userId":     v.UserId,
			"createTime": v.CreatedAt,
			"updateTime": v.UpdatedAt,
			"path":       v.Src,
		})
	}
	apiReturn.SuccessListData(c, data, count)
}

func (a *FileApi) Deletes(c *gin.Context) {
	req := commonApiStructs.RequestDeleteIds[uint]{}
	userInfo, _ := base.GetCurrentUserInfo(c)
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		apiReturn.ErrorParamFomat(c, err.Error())
		return
	}

	if len(req.Ids) == 0 {
		apiReturn.Success(c)
		return
	}

	global.Db.Transaction(func(tx *gorm.DB) error {
		files := []models.File{}
		query := tx.Order("created_at desc").Where("id IN ?", req.Ids)

		// 严格防水平越权：普通用户只能查出并删除自己上传的图片
		if userInfo.Role != 1 {
			query = query.Where("user_id = ?", userInfo.ID)
		}

		if err := query.Find(&files).Error; err != nil {
			return err
		}

		if len(files) == 0 {
			return nil
		}

		matchedIds := make([]uint, 0, len(files))
		for _, v := range files {
			matchedIds = append(matchedIds, v.ID)
			_ = os.Remove(v.Src)
		}

		delQuery := tx.Where("id IN ?", matchedIds)
		if userInfo.Role != 1 {
			delQuery = delQuery.Where("user_id = ?", userInfo.ID)
		}

		if err := delQuery.Delete(&models.File{}).Error; err != nil {
			return err
		}

		return nil
	})

	apiReturn.Success(c)
}

// ApplySyncToCloud 普通用户申请将选中的图片同步到云端图床
func (a *FileApi) ApplySyncToCloud(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		apiReturn.ErrorParamFomat(c, "未指定申请同步的图片")
		return
	}

	query := global.Db.Model(&models.File{}).Where("id IN ?", req.Ids)
	if userInfo.Role != 1 {
		query = query.Where("user_id = ?", userInfo.ID)
	}

	if err := query.Update("sync_status", 1).Error; err != nil {
		apiReturn.Error(c, "提交申请失败: "+err.Error())
		return
	}

	apiReturn.SuccessData(c, gin.H{
		"message": "已提交同步申请，等待管理员审核批准！",
	})
}

// UploadToCloud 管理员直接将本地图片同步上传到云端
func (a *FileApi) UploadToCloud(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "权限不足，仅管理员可直接推送到云端图床")
		return
	}

	var req struct {
		Ids []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		apiReturn.ErrorParamFomat(c, "未指定上传的图片")
		return
	}

	client, err := getCloudClient()
	if err != nil {
		apiReturn.Error(c, err.Error())
		return
	}

	files := []models.File{}
	if err := global.Db.Where("id IN ?", req.Ids).Find(&files).Error; err != nil {
		apiReturn.Error(c, "查询图片记录失败")
		return
	}

	successCount := 0
	for _, f := range files {
		cleanName := path.Base(f.Src)
		if err := uploadLocalFileToCloud(client, f.Src, cleanName, f.Ext, f.FileType); err == nil {
			global.Db.Model(&models.File{}).Where("id = ?", f.ID).Update("sync_status", 2)
			successCount++
		}
	}

	apiReturn.SuccessData(c, gin.H{
		"successCount": successCount,
		"message":      fmt.Sprintf("上传完成，已成功将 %d 张图片推送到云端图床", successCount),
	})
}

// AuditList 管理员查看待审核的图片列表
func (a *FileApi) AuditList(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "权限不足")
		return
	}

	files := []models.File{}
	global.Db.Order("created_at desc").Where("sync_status = 1").Find(&files)

	userIds := make([]uint, 0, len(files))
	for _, f := range files {
		userIds = append(userIds, f.UserId)
	}
	userMap := map[uint]string{}
	if len(userIds) > 0 {
		users := []models.User{}
		global.Db.Where("id IN ?", userIds).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u.Username
		}
	}

	data := []map[string]interface{}{}
	for _, v := range files {
		src := v.Src
		if strings.HasPrefix(src, ".") {
			src = src[1:]
		}
		data = append(data, map[string]interface{}{
			"id":         v.ID,
			"src":        src,
			"fileName":   v.FileName,
			"fileType":   v.FileType,
			"userId":     v.UserId,
			"userName":   userMap[v.UserId],
			"syncStatus": v.SyncStatus,
			"createTime": v.CreatedAt,
		})
	}

	apiReturn.SuccessListData(c, data, int64(len(data)))
}

// AuditAction 管理员审核用户申请 (一键全部批准 / 逐张批准 / 驳回)
func (a *FileApi) AuditAction(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "权限不足")
		return
	}

	var req struct {
		Ids    []uint `json:"ids"`
		Action string `json:"action"` // "approve" or "reject"
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		apiReturn.ErrorParamFomat(c, "参数错误")
		return
	}

	if req.Action == "reject" {
		global.Db.Model(&models.File{}).Where("id IN ?", req.Ids).Update("sync_status", 3)
		apiReturn.SuccessData(c, gin.H{"message": "已驳回选中的图片申请"})
		return
	}

	client, err := getCloudClient()
	if err != nil {
		apiReturn.Error(c, "云存储未准备就绪: "+err.Error())
		return
	}

	files := []models.File{}
	global.Db.Where("id IN ?", req.Ids).Find(&files)

	successCount := 0
	for _, f := range files {
		cleanName := path.Base(f.Src)
		if err := uploadLocalFileToCloud(client, f.Src, cleanName, f.Ext, f.FileType); err == nil {
			global.Db.Model(&models.File{}).Where("id = ?", f.ID).Update("sync_status", 2)
			successCount++
		}
	}

	apiReturn.SuccessData(c, gin.H{
		"successCount": successCount,
		"message":      fmt.Sprintf("审批完成！已批准并将 %d 张图片成功推送到云端图库", successCount),
	})
}

// GetPublicIcons 获取公共图库图标列表 (供卡片在线图标选择器使用)
func (a *FileApi) GetPublicIcons(c *gin.Context) {
	files := []models.File{}
	// 管理员上传的，或者已经通过审核并同步到云端的图片作为公共图库
	global.Db.Order("created_at desc").
		Where("file_type = 1 OR sync_status = 2 OR user_id = 1").
		Limit(120).
		Find(&files)

	data := []map[string]interface{}{}
	for _, v := range files {
		src := v.Src
		if strings.HasPrefix(src, ".") {
			src = src[1:]
		}
		data = append(data, map[string]interface{}{
			"id":       v.ID,
			"src":      src,
			"fileName": v.FileName,
			"fileType": v.FileType,
		})
	}

	apiReturn.SuccessListData(c, data, int64(len(data)))
}

// ChangeCategory 移动图片分类 (0: 壁纸图库, 1: 应用图标)
func (a *FileApi) ChangeCategory(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	var req struct {
		Ids      []uint `json:"ids"`
		FileType int    `json:"fileType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		apiReturn.ErrorParamFomat(c, "未指定修改的图片")
		return
	}

	query := global.Db.Model(&models.File{}).Where("id IN ?", req.Ids)
	if userInfo.Role != 1 {
		query = query.Where("user_id = ?", userInfo.ID)
	}

	if err := query.Update("file_type", req.FileType).Error; err != nil {
		apiReturn.Error(c, "移动分类失败: "+err.Error())
		return
	}

	apiReturn.SuccessData(c, gin.H{
		"message": "已成功移动图片分类",
	})
}

// RenameFile 重命名图片文件
func (a *FileApi) RenameFile(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	var req struct {
		Id       uint   `json:"id"`
		FileName string `json:"fileName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 || strings.TrimSpace(req.FileName) == "" {
		apiReturn.ErrorParamFomat(c, "文件名不能为空")
		return
	}

	query := global.Db.Model(&models.File{}).Where("id = ?", req.Id)
	if userInfo.Role != 1 {
		query = query.Where("user_id = ?", userInfo.ID)
	}

	if err := query.Update("file_name", strings.TrimSpace(req.FileName)).Error; err != nil {
		apiReturn.Error(c, "重命名失败: "+err.Error())
		return
	}

	apiReturn.SuccessData(c, gin.H{
		"message": "重命名成功",
	})
}


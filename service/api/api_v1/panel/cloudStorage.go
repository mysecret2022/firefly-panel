package panel

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/cloudClient"
	"sun-panel/lib/cryptoTool"
	"sun-panel/models"
	"time"

	"github.com/gin-gonic/gin"
)

type CloudStorageApi struct{}

const systemSettingCloudStorageKey = "cloud_storage_config"

// getStorageConfig 从数据库读取云存储配置
func getStorageConfig() (cloudClient.CloudStorageConfig, error) {
	var cfg cloudClient.CloudStorageConfig
	m := models.SystemSetting{}
	val, err := m.Get(systemSettingCloudStorageKey)
	if err != nil || val == "" {
		// 返回默认初始配置
		return cloudClient.CloudStorageConfig{
			Type:    "github",
			Enabled: false,
			GitHub: cloudClient.GitHubConfig{
				Branch:   "main",
				BasePath: "panel-data",
				UseCDN:   false,
			},
		}, nil
	}
	if err := json.Unmarshal([]byte(val), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// GetConfig 获取当前云存储配置
func (a *CloudStorageApi) GetConfig(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "无权访问系统存储配置")
		return
	}

	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "获取存储配置失败: "+err.Error())
		return
	}
	apiReturn.SuccessData(c, cfg)
}

// SaveConfig 保存云存储配置
func (a *CloudStorageApi) SaveConfig(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "只有管理员可修改系统存储配置")
		return
	}

	var cfg cloudClient.CloudStorageConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		apiReturn.Error(c, "解析请求参数失败: "+err.Error())
		return
	}

	m := models.SystemSetting{}
	if err := m.Set(systemSettingCloudStorageKey, cfg); err != nil {
		apiReturn.Error(c, "保存配置失败: "+err.Error())
		return
	}

	apiReturn.Success(c)
}

// TestConnection 测试与云存储的连接
func (a *CloudStorageApi) TestConnection(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "无权进行连接测试")
		return
	}

	var cfg cloudClient.CloudStorageConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		// 如果请求体为空，尝试使用数据库已保存的配置
		var loadErr error
		cfg, loadErr = getStorageConfig()
		if loadErr != nil {
			apiReturn.Error(c, "解析配置失败: "+err.Error())
			return
		}
	}

	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "初始化存储客户端失败: "+err.Error())
		return
	}

	isPrivate, err := client.TestConnection()
	if err != nil {
		apiReturn.Error(c, "连接测试未通过: "+err.Error())
		return
	}

	repoTypeDesc := "私有安全仓库 (Private)"
	if !isPrivate {
		repoTypeDesc = "公开仓库 (Public) - 外部可公开访问"
	}

	apiReturn.SuccessData(c, gin.H{
		"isPrivate":   isPrivate,
		"description": repoTypeDesc,
		"message":     "连接成功！仓库状态: " + repoTypeDesc,
	})
}

type uploadBackupReq struct {
	Password string `json:"password" binding:"required"`
}

// UploadEncryptedBackup 打包、AES 加密并上传到云端备份存储
func (a *CloudStorageApi) UploadEncryptedBackup(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "仅管理员可执行全站备份上传")
		return
	}

	var req uploadBackupReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Password) == "" {
		apiReturn.Error(c, "云端备份必须提供加密密码，确保数据隐私安全")
		return
	}

	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "读取存储配置失败")
		return
	}
	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "创建云端存储客户端失败: "+err.Error())
		return
	}

	// 1. 生成全站数据包 (ZIP)
	var backup SiteBackupData
	backup.Version = "v1.3.1-backup"
	backup.ExportTime = time.Now().Format("2006-01-02 15:04:05")
	backup.Operator = userInfo.Username

	global.Db.Find(&backup.Users)
	global.Db.Find(&backup.ItemIconGroups)
	global.Db.Find(&backup.ItemIcons)
	global.Db.Find(&backup.UserConfigs)
	global.Db.Find(&backup.SystemSettings)
	global.Db.Find(&backup.ModuleConfigs)

	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		apiReturn.Error(c, "序列化备份数据失败: "+err.Error())
		return
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// 写入 json 描述
	jsonFileWriter, err := zipWriter.Create("backup_data.json")
	if err != nil {
		apiReturn.Error(c, "创建 ZIP 索引失败: "+err.Error())
		return
	}
	_, _ = jsonFileWriter.Write(jsonData)

	// 打包 files / source_path
	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}
	addDirToZip(zipWriter, sourcePath, "files")

	// 尝试打包 uploads
	if _, err := os.Stat("./uploads"); err == nil {
		addDirToZip(zipWriter, "./uploads", "uploads")
	}

	if err := zipWriter.Close(); err != nil {
		apiReturn.Error(c, "打包 ZIP 失败: "+err.Error())
		return
	}

	// 2. 使用 AES-256-GCM 加密 ZIP
	zipBytes := zipBuf.Bytes()
	encBytes, err := cryptoTool.EncryptAES256GCM(zipBytes, req.Password)
	if err != nil {
		apiReturn.Error(c, "加密备份数据失败: "+err.Error())
		return
	}

	// 3. 上传到云端
	filename := fmt.Sprintf("sun-panel-backup_%s.enc.zip", time.Now().Format("20060102_150405"))
	cloudRelPath := path.Join("backups", filename)

	accessUrl, err := client.UploadFile(cloudRelPath, encBytes, "application/octet-stream")
	if err != nil {
		apiReturn.Error(c, "上传云端失败: "+err.Error())
		return
	}

	apiReturn.SuccessData(c, gin.H{
		"filename":    filename,
		"downloadUrl": accessUrl,
		"size":        len(encBytes),
		"message":     "加密备份上传成功！",
	})
}

type restoreBackupReq struct {
	Filename string `json:"filename" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RestoreEncryptedBackup 从云端拉取加密备份、解密并恢复
func (a *CloudStorageApi) RestoreEncryptedBackup(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "仅管理员可执行全站备份还原")
		return
	}

	var req restoreBackupReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Filename) == "" {
		apiReturn.Error(c, "备份文件名和解密密码均不能为空")
		return
	}

	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "读取存储配置失败")
		return
	}
	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "创建云端存储客户端失败: "+err.Error())
		return
	}

	// 1. 从云端下载加密文件
	cloudRelPath := path.Join("backups", req.Filename)
	encBytes, err := client.DownloadFile(cloudRelPath)
	if err != nil {
		apiReturn.Error(c, "从云端下载备份文件失败: "+err.Error())
		return
	}

	// 2. 解密
	zipBytes, err := cryptoTool.DecryptAES256GCM(encBytes, req.Password)
	if err != nil {
		apiReturn.Error(c, "解密失败，密码错误或备份文件已被篡改损坏")
		return
	}

	// 3. 写入临时 ZIP 并解压还原
	tempZipFile, err := os.CreateTemp("", "sun-panel-cloud-restore-*.zip")
	if err != nil {
		apiReturn.Error(c, "创建临时文件失败: "+err.Error())
		return
	}
	defer os.Remove(tempZipFile.Name())
	defer tempZipFile.Close()

	if _, err := tempZipFile.Write(zipBytes); err != nil {
		apiReturn.Error(c, "写入临时解密包失败: "+err.Error())
		return
	}

	zipReader, err := zip.OpenReader(tempZipFile.Name())
	if err != nil {
		apiReturn.Error(c, "打开解密 ZIP 失败: "+err.Error())
		return
	}
	defer zipReader.Close()

	var backupData SiteBackupData
	hasJson := false

	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}

	for _, zf := range zipReader.File {
		if zf.Name == "backup_data.json" {
			rc, err := zf.Open()
			if err != nil {
				continue
			}
			content, _ := io.ReadAll(rc)
			rc.Close()
			if err := json.Unmarshal(content, &backupData); err == nil {
				hasJson = true
			}
		} else if strings.HasPrefix(zf.Name, "files/") {
			relPath := strings.TrimPrefix(zf.Name, "files/")
			destPath := filepath.Join(sourcePath, relPath)
			extractZipFile(zf, destPath)
		} else if strings.HasPrefix(zf.Name, "uploads/") {
			destPath := filepath.Join(".", zf.Name)
			extractZipFile(zf, destPath)
		}
	}

	if !hasJson {
		apiReturn.Error(c, "备份文件中未找到有效的 backup_data.json")
		return
	}

	if err := applyDatabaseRestore(backupData, userInfo); err != nil {
		apiReturn.Error(c, "恢复数据库失败: "+err.Error())
		return
	}

	apiReturn.SuccessData(c, gin.H{
		"message": "解密还原成功！数据已更新",
	})
}

// ListCloudBackups 获取云端备份列表
func (a *CloudStorageApi) ListCloudBackups(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "无权查看备份列表")
		return
	}

	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "读取存储配置失败")
		return
	}
	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "创建云端存储客户端失败: "+err.Error())
		return
	}

	items, err := client.ListFiles("backups")
	if err != nil {
		apiReturn.Error(c, "获取备份列表失败: "+err.Error())
		return
	}

	var backups []cloudClient.FileItem
	for _, it := range items {
		if strings.HasSuffix(it.Name, ".zip") || strings.HasSuffix(it.Name, ".enc.zip") {
			backups = append(backups, it)
		}
	}
	apiReturn.SuccessData(c, backups)
}

// ListCloudImages 获取云端图床图片列表
func (a *CloudStorageApi) ListCloudImages(c *gin.Context) {
	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "读取存储配置失败")
		return
	}
	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "创建云端存储客户端失败: "+err.Error())
		return
	}

	items, err := client.ListFiles("images")
	if err != nil {
		apiReturn.Error(c, "获取云端图片列表失败: "+err.Error())
		return
	}

	var images []cloudClient.FileItem
	imageExts := []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".ico"}
	for _, it := range items {
		ext := strings.ToLower(path.Ext(it.Name))
		for _, e := range imageExts {
			if ext == e {
				images = append(images, it)
				break
			}
		}
	}
	apiReturn.SuccessData(c, images)
}

type syncImageReq struct {
	Filename string `json:"filename" binding:"required"`
}

// SyncImageToLocal 单个选择下载云端图片到本地存储
func (a *CloudStorageApi) SyncImageToLocal(c *gin.Context) {
	var req syncImageReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Filename == "" {
		apiReturn.Error(c, "未指定要下载的文件名")
		return
	}

	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "读取存储配置失败")
		return
	}
	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "创建云端存储客户端失败: "+err.Error())
		return
	}

	cleanName := path.Base(req.Filename)
	data, err := client.DownloadFile(path.Join("images", cleanName))
	if err != nil {
		apiReturn.Error(c, "下载文件失败: "+err.Error())
		return
	}

	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}

	targetDir := filepath.Join(sourcePath, "cloud_synced")
	_ = os.MkdirAll(targetDir, 0755)
	destPath := filepath.Join(targetDir, cleanName)

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		apiReturn.Error(c, "写入本地磁盘失败: "+err.Error())
		return
	}

	apiReturn.SuccessData(c, gin.H{
		"localPath": destPath,
		"filename":  cleanName,
		"message":   "已成功下载并同步到本地！",
	})
}

// SyncAllImagesToLocal 一键将云端所有图片全部下载回本地存储
func (a *CloudStorageApi) SyncAllImagesToLocal(c *gin.Context) {
	cfg, err := getStorageConfig()
	if err != nil {
		apiReturn.Error(c, "读取存储配置失败")
		return
	}
	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		apiReturn.Error(c, "创建云端存储客户端失败: "+err.Error())
		return
	}

	items, err := client.ListFiles("images")
	if err != nil {
		apiReturn.Error(c, "获取云端图片列表失败: "+err.Error())
		return
	}

	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}
	targetDir := filepath.Join(sourcePath, "cloud_synced")
	_ = os.MkdirAll(targetDir, 0755)

	syncedCount := 0
	for _, it := range items {
		if it.IsDirectory {
			continue
		}
		cleanName := path.Base(it.Name)
		data, err := client.DownloadFile(path.Join("images", cleanName))
		if err != nil {
			continue
		}
		destPath := filepath.Join(targetDir, cleanName)
		if err := os.WriteFile(destPath, data, 0644); err == nil {
			syncedCount++
		}
	}

	apiReturn.SuccessData(c, gin.H{
		"total":       len(items),
		"syncedCount": syncedCount,
		"message":     fmt.Sprintf("同步完成！共发现 %d 张云端图片，已成功同步 %d 张到本地存储", len(items), syncedCount),
	})
}

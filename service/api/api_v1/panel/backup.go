package panel

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/models"
	"time"

	"github.com/gin-gonic/gin"
)

type BackupApi struct{}

// 备份数据结构定义
type SiteBackupData struct {
	Version        string                 `json:"version"`
	ExportTime     string                 `json:"exportTime"`
	Operator       string                 `json:"operator"`
	Users          []models.User          `json:"users,omitempty"`
	ItemIconGroups []models.ItemIconGroup `json:"itemIconGroups"`
	ItemIcons      []models.ItemIcon      `json:"itemIcons"`
	UserConfigs    []models.UserConfig    `json:"userConfigs"`
	SystemSettings []models.SystemSetting `json:"systemSettings,omitempty"`
	ModuleConfigs  []models.ModuleConfig  `json:"moduleConfigs,omitempty"`
}

// ExportBackup 导出全站备份 (ZIP 文件流下载)
func (b *BackupApi) ExportBackup(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "权限不足，仅系统管理员可导出整站备份")
		return
	}

	var backup SiteBackupData
	backup.Version = "v1.3.1-backup"
	backup.ExportTime = time.Now().Format("2006-01-02 15:04:05")
	backup.Operator = userInfo.Username

	// 仅管理员权限备份全站数据
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

	// 设置响应头为 zip 下载
	filename := fmt.Sprintf("sun-panel-backup_%s.zip", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/zip")
	c.Header("Transfer-Encoding", "chunked")

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// 1. 写入 json 数据
	jsonFileWriter, err := zipWriter.Create("backup_data.json")
	if err != nil {
		return
	}
	_, _ = jsonFileWriter.Write(jsonData)

	// 2. 打包上传文件目录 (source_path)
	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}
	addDirToZip(zipWriter, sourcePath, "files")

	// 尝试打包 uploads 目录（如果存在）
	if _, err := os.Stat("./uploads"); err == nil {
		addDirToZip(zipWriter, "./uploads", "uploads")
	}
}

// ExportJson 导出纯 JSON 格式备份数据
func (b *BackupApi) ExportJson(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)

	var backup SiteBackupData
	backup.Version = "v1.3.1-backup"
	backup.ExportTime = time.Now().Format("2006-01-02 15:04:05")
	backup.Operator = userInfo.Username

	if userInfo.Role == 1 {
		global.Db.Find(&backup.Users)
		global.Db.Find(&backup.ItemIconGroups)
		global.Db.Find(&backup.ItemIcons)
		global.Db.Find(&backup.UserConfigs)
		global.Db.Find(&backup.SystemSettings)
		global.Db.Find(&backup.ModuleConfigs)
	} else {
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.ItemIconGroups)
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.ItemIcons)
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.UserConfigs)
	}

	apiReturn.SuccessData(c, backup)
}

// RestoreBackup 从上传的 ZIP 或 JSON 恢复全站数据
func (b *BackupApi) RestoreBackup(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)
	if userInfo.Role != 1 {
		apiReturn.Error(c, "权限不足，仅系统管理员可执行全站备份恢复")
		return
	}

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		apiReturn.Error(c, "获取上传文件失败")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == ".zip" {
		// 处理 ZIP 文件
		tempZipFile, err := os.CreateTemp("", "sun-panel-restore-*.zip")
		if err != nil {
			apiReturn.Error(c, "创建临时文件失败: "+err.Error())
			return
		}
		defer os.Remove(tempZipFile.Name())
		defer tempZipFile.Close()

		if _, err := io.Copy(tempZipFile, file); err != nil {
			apiReturn.Error(c, "写入临时文件失败: "+err.Error())
			return
		}

		zipReader, err := zip.OpenReader(tempZipFile.Name())
		if err != nil {
			apiReturn.Error(c, "无法解压 ZIP 文件: "+err.Error())
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
				// 恢复文件到 sourcePath
				relPath := strings.TrimPrefix(zf.Name, "files/")
				destPath := filepath.Join(sourcePath, relPath)
				extractZipFile(zf, destPath)
			} else if strings.HasPrefix(zf.Name, "uploads/") {
				// 恢复 uploads 文件
				destPath := filepath.Join(".", zf.Name)
				extractZipFile(zf, destPath)
			}
		}

		if !hasJson {
			apiReturn.Error(c, "压缩包内未找到有效的 backup_data.json 描述文件")
			return
		}

		// 写入数据库
		if err := applyDatabaseRestore(backupData, userInfo); err != nil {
			apiReturn.Error(c, "恢复数据库失败: "+err.Error())
			return
		}

		apiReturn.Success(c)
	} else if ext == ".json" {
		// 处理纯 JSON 文件
		content, err := io.ReadAll(file)
		if err != nil {
			apiReturn.Error(c, "读取 JSON 文件失败: "+err.Error())
			return
		}
		var backupData SiteBackupData
		if err := json.Unmarshal(content, &backupData); err != nil {
			apiReturn.Error(c, "解析 JSON 格式错误: "+err.Error())
			return
		}

		if err := applyDatabaseRestore(backupData, userInfo); err != nil {
			apiReturn.Error(c, "恢复数据库失败: "+err.Error())
			return
		}

		apiReturn.Success(c)
	} else {
		apiReturn.Error(c, "仅支持 .zip 或 .json 格式的备份文件")
	}
}

// 将指定目录打入 zip
func addDirToZip(zw *zip.Writer, srcDir, zipPrefix string) {
	_ = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return nil
		}
		zipPath := filepath.ToSlash(filepath.Join(zipPrefix, relPath))
		w, err := zw.Create(zipPath)
		if err != nil {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		_, _ = io.Copy(w, f)
		return nil
	})
}

// 解压单个文件
func extractZipFile(zf *zip.File, destPath string) {
	if zf.FileInfo().IsDir() {
		_ = os.MkdirAll(destPath, 0755)
		return
	}
	_ = os.MkdirAll(filepath.Dir(destPath), 0755)
	rc, err := zf.Open()
	if err != nil {
		return
	}
	defer rc.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zf.Mode())
	if err != nil {
		return
	}
	defer destFile.Close()
	_, _ = io.Copy(destFile, rc)
}

// 执行数据库恢复更新
func applyDatabaseRestore(data SiteBackupData, currentUser models.User) error {
	tx := global.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	isAdmin := currentUser.Role == 1

	if isAdmin {
		// 管理员执行全量恢复
		if len(data.ItemIconGroups) > 0 {
			tx.Exec("DELETE FROM item_icon")
			tx.Exec("DELETE FROM item_icon_group")

			for _, g := range data.ItemIconGroups {
				tx.Create(&g)
			}
			for _, item := range data.ItemIcons {
				tx.Create(&item)
			}
		}

		if len(data.UserConfigs) > 0 {
			tx.Exec("DELETE FROM user_config")
			for _, uc := range data.UserConfigs {
				tx.Create(&uc)
			}
		}

		if len(data.SystemSettings) > 0 {
			tx.Exec("DELETE FROM system_setting")
			for _, ss := range data.SystemSettings {
				tx.Create(&ss)
			}
		}

		if len(data.ModuleConfigs) > 0 {
			tx.Exec("DELETE FROM module_config")
			for _, mc := range data.ModuleConfigs {
				tx.Create(&mc)
			}
		}
	} else {
		// 普通用户仅覆盖恢复自己的卡片与分组
		tx.Where("user_id = ?", currentUser.ID).Delete(&models.ItemIcon{})
		tx.Where("user_id = ?", currentUser.ID).Delete(&models.ItemIconGroup{})

		for _, g := range data.ItemIconGroups {
			g.UserId = currentUser.ID
			tx.Create(&g)
		}
		for _, item := range data.ItemIcons {
			item.UserId = currentUser.ID
			tx.Create(&item)
		}

		for _, uc := range data.UserConfigs {
			uc.UserId = currentUser.ID
			var exists models.UserConfig
			if tx.Where("user_id = ?", currentUser.ID).First(&exists).Error == nil {
				tx.Model(&exists).Updates(&uc)
			} else {
				tx.Create(&uc)
			}
		}
	}

	return tx.Commit().Error
}

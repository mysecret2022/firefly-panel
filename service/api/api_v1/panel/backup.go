package panel

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/cryptoTool"
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
	Files          []models.File          `json:"files,omitempty"` // 完整包含图库记录
}

// ExportBackup 导出全站备份 (ZIP 完整封包后下载)
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

	// 1. 查询数据库全量数据
	global.Db.Find(&backup.Users)
	global.Db.Find(&backup.ItemIconGroups)
	global.Db.Find(&backup.ItemIcons)
	global.Db.Find(&backup.UserConfigs)
	global.Db.Find(&backup.SystemSettings)
	global.Db.Find(&backup.ModuleConfigs)
	global.Db.Find(&backup.Files) // 完整包含图库记录

	// 补全 GORM 忽略字段，防止导出空数据
	for i := range backup.ItemIcons {
		if backup.ItemIcons[i].IconJson != "" {
			_ = json.Unmarshal([]byte(backup.ItemIcons[i].IconJson), &backup.ItemIcons[i].Icon)
		}
	}
	for i := range backup.UserConfigs {
		if backup.UserConfigs[i].PanelJson != "" {
			_ = json.Unmarshal([]byte(backup.UserConfigs[i].PanelJson), &backup.UserConfigs[i].Panel)
		}
		if backup.UserConfigs[i].SearchEngineJson != "" {
			_ = json.Unmarshal([]byte(backup.UserConfigs[i].SearchEngineJson), &backup.UserConfigs[i].SearchEngine)
		}
	}
	for i := range backup.ModuleConfigs {
		if backup.ModuleConfigs[i].ValueJson != "" {
			_ = json.Unmarshal([]byte(backup.ModuleConfigs[i].ValueJson), &backup.ModuleConfigs[i].Value)
		}
	}

	jsonData, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		apiReturn.Error(c, "序列化备份数据失败: "+err.Error())
		return
	}

	// 2. 使用内存 Buffer 打包 ZIP，确保 ZIP 中央目录完全闭合无损
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// 写入 backup_data.json
	jsonFileWriter, err := zipWriter.Create("backup_data.json")
	if err != nil {
		apiReturn.Error(c, "创建 ZIP 描述文件失败: "+err.Error())
		return
	}
	if _, err := jsonFileWriter.Write(jsonData); err != nil {
		apiReturn.Error(c, "写入 JSON 描述失败: "+err.Error())
		return
	}

	// 打包上传文件目录 (source_path)
	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}
	addDirToZip(zipWriter, sourcePath, "files")

	// 尝试打包 uploads 目录（若存在）
	if _, err := os.Stat("./uploads"); err == nil {
		addDirToZip(zipWriter, "./uploads", "uploads")
	}

	// 显式 Close，封包 ZIP 并写入完整的中央索引结构
	if err := zipWriter.Close(); err != nil {
		apiReturn.Error(c, "封包 ZIP 文件失败: "+err.Error())
		return
	}

	// 3. 响应二进制文件流下载
	filename := fmt.Sprintf("sun-panel-backup_%s.zip", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Length", strconv.Itoa(zipBuf.Len()))
	c.Writer.Write(zipBuf.Bytes())
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
		global.Db.Find(&backup.Files)
	} else {
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.ItemIconGroups)
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.ItemIcons)
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.UserConfigs)
		global.Db.Where("user_id = ?", userInfo.ID).Find(&backup.Files)
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

	password := strings.TrimSpace(c.PostForm("password"))
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

	if ext == ".zip" || ext == ".enc" || ext == ".bin" || password != "" {
		tempZipFile, err := os.CreateTemp("", "sun-panel-restore-*.zip")
		if err != nil {
			apiReturn.Error(c, "创建临时文件失败: "+err.Error())
			return
		}
		defer os.Remove(tempZipFile.Name())
		defer tempZipFile.Close()

		if password != "" {
			rawBytes, err := io.ReadAll(file)
			if err != nil {
				apiReturn.Error(c, "读取上传数据失败: "+err.Error())
				return
			}
			decrypted, err := cryptoTool.DecryptAES256GCM(rawBytes, password)
			if err != nil {
				apiReturn.Error(c, "解密失败：密码错误或文件已损坏")
				return
			}
			if _, err := tempZipFile.Write(decrypted); err != nil {
				apiReturn.Error(c, "写入临时解密文件失败: "+err.Error())
				return
			}
		} else {
			if _, err := io.Copy(tempZipFile, file); err != nil {
				apiReturn.Error(c, "写入临时文件失败: "+err.Error())
				return
			}
		}

		zipReader, err := zip.OpenReader(tempZipFile.Name())
		if err != nil {
			if password == "" && (ext == ".enc" || ext == ".bin") {
				apiReturn.Error(c, "检测到该文件为加密备份，请输入 AES-256 解密密码后再试")
				return
			}
			apiReturn.Error(c, "无法解压 ZIP 文件 (可能文件已损坏): "+err.Error())
			return
		}
		defer zipReader.Close()

		var backupData SiteBackupData
		hasFoundFile := false
		var jsonUnmarshalErr error

		sourcePath := global.Config.GetValueString("base", "source_path")
		if sourcePath == "" {
			sourcePath = "./files"
		}

		// 遍历解压并定位 backup_data.json
		for _, zf := range zipReader.File {
			cleanName := strings.ReplaceAll(zf.Name, "\\", "/")
			baseName := filepath.Base(cleanName)

			// 只要文件名是 backup_data.json 且不是目录即解析
			if baseName == "backup_data.json" && !zf.FileInfo().IsDir() {
				hasFoundFile = true
				rc, err := zf.Open()
				if err != nil {
					jsonUnmarshalErr = err
					continue
				}
				content, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					jsonUnmarshalErr = err
					continue
				}

				if err := json.Unmarshal(content, &backupData); err != nil {
					jsonUnmarshalErr = err
				} else {
					jsonUnmarshalErr = nil
				}
			} else if strings.Contains(cleanName, "files/") {
				idx := strings.Index(cleanName, "files/")
				relPath := cleanName[idx+len("files/"):]
				destPath := filepath.Join(sourcePath, relPath)
				extractZipFile(zf, destPath)
			} else if strings.Contains(cleanName, "uploads/") {
				idx := strings.Index(cleanName, "uploads/")
				relPath := cleanName[idx:]
				destPath := filepath.Join(".", relPath)
				extractZipFile(zf, destPath)
			}
		}

		if !hasFoundFile {
			apiReturn.Error(c, "压缩包内未找到 backup_data.json 描述文件，请检查压缩包结构")
			return
		}
		if jsonUnmarshalErr != nil {
			apiReturn.Error(c, "解析 backup_data.json 失败: "+jsonUnmarshalErr.Error())
			return
		}

		// 执行数据库恢复更新
		if err := applyDatabaseRestore(backupData, userInfo); err != nil {
			apiReturn.Error(c, "恢复数据库失败: "+err.Error())
			return
		}

		apiReturn.Success(c)
	} else if ext == ".json" {
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
		if len(data.ItemIconGroups) > 0 {
			tx.Exec("DELETE FROM item_icon")
			tx.Exec("DELETE FROM item_icon_group")

			for _, g := range data.ItemIconGroups {
				tx.Create(&g)
			}
			for _, item := range data.ItemIcons {
				if item.IconJson == "" && (item.Icon.Src != "" || item.Icon.Text != "") {
					b, _ := json.Marshal(item.Icon)
					item.IconJson = string(b)
				}
				tx.Create(&item)
			}
		}

		if len(data.UserConfigs) > 0 {
			tx.Exec("DELETE FROM user_config")
			for _, uc := range data.UserConfigs {
				if uc.PanelJson == "" && uc.Panel != nil {
					b, _ := json.Marshal(uc.Panel)
					uc.PanelJson = string(b)
				}
				if uc.SearchEngineJson == "" && uc.SearchEngine != nil {
					b, _ := json.Marshal(uc.SearchEngine)
					uc.SearchEngineJson = string(b)
				}
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
				if mc.ValueJson == "" && mc.Value != nil {
					b, _ := json.Marshal(mc.Value)
					mc.ValueJson = string(b)
				}
				tx.Create(&mc)
			}
		}

		if len(data.Files) > 0 {
			tx.Exec("DELETE FROM file")
			for _, f := range data.Files {
				tx.Create(&f)
			}
		}
	} else {
		tx.Where("user_id = ?", currentUser.ID).Delete(&models.ItemIcon{})
		tx.Where("user_id = ?", currentUser.ID).Delete(&models.ItemIconGroup{})

		for _, g := range data.ItemIconGroups {
			g.UserId = currentUser.ID
			tx.Create(&g)
		}
		for _, item := range data.ItemIcons {
			item.UserId = currentUser.ID
			if item.IconJson == "" && (item.Icon.Src != "" || item.Icon.Text != "") {
				b, _ := json.Marshal(item.Icon)
				item.IconJson = string(b)
			}
			tx.Create(&item)
		}

		for _, uc := range data.UserConfigs {
			uc.UserId = currentUser.ID
			if uc.PanelJson == "" && uc.Panel != nil {
				b, _ := json.Marshal(uc.Panel)
				uc.PanelJson = string(b)
			}
			if uc.SearchEngineJson == "" && uc.SearchEngine != nil {
				b, _ := json.Marshal(uc.SearchEngine)
				uc.SearchEngineJson = string(b)
			}
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

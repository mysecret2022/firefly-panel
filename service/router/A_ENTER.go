package router

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sun-panel/global"
	"sun-panel/lib/cloudClient"
	"sun-panel/models"
	"sun-panel/router/openness"
	"sun-panel/router/panel"
	"sun-panel/router/system"

	"github.com/gin-gonic/gin"
)

// 初始化总路由
func InitRouters(addr string) error {
	router := gin.Default()
	rootRouter := router.Group("/")
	routerGroup := rootRouter.Group("api")

	// 接口
	system.Init(routerGroup)
	panel.Init(routerGroup)
	openness.Init(routerGroup)

	// WEB文件服务
	{
		webPath := "./web"
		router.StaticFile("/", webPath+"/index.html")
		router.Static("/assets", webPath+"/assets")
		router.Static("/custom", webPath+"/custom")
		router.StaticFile("/favicon.ico", webPath+"/favicon.ico")
		router.StaticFile("/favicon.svg", webPath+"/favicon.svg")
	}

	// 挂载支持【本地优先 + 云端按需回源自愈】的增强文件服务
	sourcePath := global.Config.GetValueString("base", "source_path")
	if sourcePath == "" {
		sourcePath = "./files"
	}
	setupFileServiceWithCloudFallback(router, sourcePath)

	global.Logger.Info("Sun-Panel is Started. Listening and serving HTTP on ", addr)
	return router.Run(addr)
}

// 增强文件服务：本地存在直接返回；本地缺失则自动从 GitHub/R2 回源并自愈缓存
func setupFileServiceWithCloudFallback(router *gin.Engine, sourcePath string) {
	prefix := sourcePath
	if strings.HasPrefix(prefix, ".") {
		prefix = prefix[1:]
	}
	prefix = "/" + strings.Trim(prefix, "/")

	handler := func(c *gin.Context) {
		reqRelPath := c.Param("filepath")
		cleanRel := strings.TrimPrefix(reqRelPath, "/")
		localFilePath := filepath.Join(sourcePath, cleanRel)

		// 1. 本地磁盘存在，直接使用原生高性能静态文件输出
		if info, err := os.Stat(localFilePath); err == nil && !info.IsDir() {
			c.File(localFilePath)
			return
		}

		// 备选本地路径（兼容 ./files 与 ./uploads 互相跨目录检查）
		altLocalFilePath := ""
		if strings.HasPrefix(sourcePath, "./uploads") || strings.HasPrefix(sourcePath, "uploads") {
			altLocalFilePath = filepath.Join("./files", cleanRel)
		} else {
			altLocalFilePath = filepath.Join("./uploads", cleanRel)
		}
		if info, err := os.Stat(altLocalFilePath); err == nil && !info.IsDir() {
			c.File(altLocalFilePath)
			return
		}

		// 2. 本地不存在，尝试从已配置的云端图床透明拉取
		data, mimeType, err := fetchImageFromCloudFallback(cleanRel)
		if err == nil && len(data) > 0 {
			// 自动自愈：异步/同步写入本地对应目录，下次访问直接走本地
			_ = os.MkdirAll(filepath.Dir(localFilePath), 0755)
			_ = os.WriteFile(localFilePath, data, 0644)

			// 设置图片缓存头并返回二进制流
			c.Header("Cache-Control", "public, max-age=86400")
			c.Data(http.StatusOK, mimeType, data)
			return
		}

		// 3. 云端亦未命中，返回 404
		c.Status(http.StatusNotFound)
	}

	// 注册动态路由（同时支持 GET 和 HEAD，并兼顾 /uploads 与 /files 两种常见前缀）
	router.GET(prefix+"/*filepath", handler)
	router.HEAD(prefix+"/*filepath", handler)
	if prefix != "/uploads" {
		router.GET("/uploads/*filepath", handler)
		router.HEAD("/uploads/*filepath", handler)
	}
	if prefix != "/files" {
		router.GET("/files/*filepath", handler)
		router.HEAD("/files/*filepath", handler)
	}
}

// 从云端探测并拉取指定图片
func fetchImageFromCloudFallback(relPath string) ([]byte, string, error) {
	// 读取系统云存储配置
	m := models.SystemSetting{}
	val, err := m.Get("cloud_storage_config")
	if err != nil || val == "" {
		return nil, "", fmt.Errorf("云存储未配置")
	}
	var cfg cloudClient.CloudStorageConfig
	if err := json.Unmarshal([]byte(val), &cfg); err != nil || !cfg.Enabled {
		return nil, "", fmt.Errorf("云存储未启用")
	}

	client, err := cloudClient.CreateCloudClient(cfg)
	if err != nil {
		return nil, "", err
	}

	fileName := path.Base(relPath)
	ext := strings.ToLower(path.Ext(fileName))

	// 根据扩展名确定 MIME
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "image/png"
	}

	// 候选云端相对路径（按优先级探查）
	candidates := []string{
		path.Join("images/wallpapers", fileName),
		path.Join("images/icons", fileName),
		path.Join("images", fileName),
		relPath,
	}

	// 尝试向云端按路径探查并拉取
	for _, cloudPath := range candidates {
		data, err := client.DownloadFile(cloudPath)
		if err == nil && len(data) > 0 {
			return data, mimeType, nil
		}
	}

	return nil, "", fmt.Errorf("云端未找到对应图片: %s", fileName)
}

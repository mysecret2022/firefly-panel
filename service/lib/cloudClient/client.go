package cloudClient

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

type FileItem struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	DownloadUrl string    `json:"downloadUrl"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsDirectory bool      `json:"isDirectory"`
}

type CloudStorageConfig struct {
	Type         string       `json:"type"` // "github" 或 "r2"
	Enabled      bool         `json:"enabled"`
	GitHub       GitHubConfig `json:"github"`
	CloudflareR2 R2Config     `json:"cloudflareR2"`
}

type GitHubConfig struct {
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch"`
	Token    string `json:"token"`
	BasePath string `json:"basePath"` // 默认 "panel-data"
	UseCDN   bool   `json:"useCDN"`   // 使用 jsdelivr CDN (仅对公开仓库生效)
}

type R2Config struct {
	AccountID       string `json:"accountId"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	BucketName      string `json:"bucketName"`
	PublicDomain    string `json:"publicDomain"` // 如 "https://img.yourdomain.com"
}

// CloudStorageClient 统一云存储接口
type CloudStorageClient interface {
	TestConnection() (isPrivate bool, err error)
	UploadFile(relativePath string, data []byte, contentType string) (string, error)
	DownloadFile(relativePath string) ([]byte, error)
	ListFiles(subDir string) ([]FileItem, error)
}

func CreateCloudClient(cfg CloudStorageConfig) (CloudStorageClient, error) {
	switch cfg.Type {
	case "github":
		return NewGitHubClient(cfg.GitHub), nil
	case "r2":
		return NewR2Client(cfg.CloudflareR2), nil
	default:
		return nil, errors.New("不支持的云存储类型: " + cfg.Type)
	}
}

// ======================== GitHub 实现 ========================

type GitHubClient struct {
	Config GitHubConfig
	client *http.Client
}

func NewGitHubClient(cfg GitHubConfig) *GitHubClient {
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.BasePath == "" {
		cfg.BasePath = "panel-data"
	}
	return &GitHubClient{
		Config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// TestConnection 测试 GitHub 连通性并检查是否为私有仓库
func (g *GitHubClient) TestConnection() (isPrivate bool, err error) {
	if g.Config.Owner == "" || g.Config.Repo == "" || g.Config.Token == "" {
		return false, errors.New("GitHub 用户名、仓库名和 Token 均不能为空")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", g.Config.Owner, g.Config.Repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+g.Config.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("网络连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return false, errors.New("仓库不存在，或 Token 权限不足以访问该私有仓库")
	}
	if resp.StatusCode == 401 {
		return false, errors.New("GitHub Token 无效或已过期")
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("GitHub API 响应异常 (%d): %s", resp.StatusCode, string(b))
	}

	var repoInfo struct {
		Private bool `json:"private"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return false, err
	}

	return repoInfo.Private, nil
}

// UploadFile 上传文件到 GitHub 仓库
func (g *GitHubClient) UploadFile(relativePath string, data []byte, contentType string) (string, error) {
	fullPath := strings.TrimPrefix(path.Join(g.Config.BasePath, relativePath), "/")
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", g.Config.Owner, g.Config.Repo, fullPath)

	// 1. 查询是否已存在以获取 SHA (覆盖模式)
	var sha string
	getReq, _ := http.NewRequest("GET", apiUrl+"?ref="+g.Config.Branch, nil)
	getReq.Header.Set("Authorization", "Bearer "+g.Config.Token)
	getReq.Header.Set("Accept", "application/vnd.github.v3+json")
	if getResp, err := g.client.Do(getReq); err == nil {
		if getResp.StatusCode == 200 {
			var fileInfo struct {
				Sha string `json:"sha"`
			}
			_ = json.NewDecoder(getResp.Body).Decode(&fileInfo)
			sha = fileInfo.Sha
		}
		getResp.Body.Close()
	}

	// 2. PUT 写入
	commitMsg := "Sync " + path.Base(relativePath)
	bodyData := map[string]interface{}{
		"message": commitMsg,
		"content": base64.StdEncoding.EncodeToString(data),
		"branch":  g.Config.Branch,
	}
	if sha != "" {
		bodyData["sha"] = sha
	}

	jsonBytes, _ := json.Marshal(bodyData)
	putReq, err := http.NewRequest("PUT", apiUrl, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	putReq.Header.Set("Authorization", "Bearer "+g.Config.Token)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Accept", "application/vnd.github.v3+json")

	putResp, err := g.client.Do(putReq)
	if err != nil {
		return "", err
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != 200 && putResp.StatusCode != 201 {
		b, _ := io.ReadAll(putResp.Body)
		return "", fmt.Errorf("上传失败 (%d): %s", putResp.StatusCode, string(b))
	}

	// 计算访问 URL
	if g.Config.UseCDN {
		return fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s/%s@%s/%s", g.Config.Owner, g.Config.Repo, g.Config.Branch, fullPath), nil
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", g.Config.Owner, g.Config.Repo, g.Config.Branch, fullPath), nil
}

// DownloadFile 从 GitHub 获取文件内容
func (g *GitHubClient) DownloadFile(relativePath string) ([]byte, error) {
	fullPath := strings.TrimPrefix(path.Join(g.Config.BasePath, relativePath), "/")
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", g.Config.Owner, g.Config.Repo, fullPath, g.Config.Branch)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.Config.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("获取文件失败 (%d)", resp.StatusCode)
	}

	var contentResp struct {
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
		return nil, err
	}

	cleanContent := strings.ReplaceAll(contentResp.Content, "\n", "")
	return base64.StdEncoding.DecodeString(cleanContent)
}

// ListFiles 列出指定目录下的文件列表
func (g *GitHubClient) ListFiles(subDir string) ([]FileItem, error) {
	fullPath := strings.TrimPrefix(path.Join(g.Config.BasePath, subDir), "/")
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", g.Config.Owner, g.Config.Repo, fullPath, g.Config.Branch)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.Config.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []FileItem{}, nil
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("列出文件失败 (%d)", resp.StatusCode)
	}

	var items []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Size        int64  `json:"size"`
		Type        string `json:"type"`
		DownloadUrl string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	var result []FileItem
	for _, it := range items {
		url := it.DownloadUrl
		if g.Config.UseCDN && it.Type == "file" {
			url = fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s/%s@%s/%s", g.Config.Owner, g.Config.Repo, g.Config.Branch, it.Path)
		}
		result = append(result, FileItem{
			Name:        it.Name,
			Path:        it.Path,
			Size:        it.Size,
			DownloadUrl: url,
			IsDirectory: it.Type == "dir",
			UpdatedAt:   time.Now(),
		})
	}
	return result, nil
}

// ======================== Cloudflare R2 / S3 兼容实现 ========================

type R2Client struct {
	Config R2Config
	client *http.Client
}

func NewR2Client(cfg R2Config) *R2Client {
	return &R2Client{
		Config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// TestConnection 测试 R2 连接并检查状态
func (r *R2Client) TestConnection() (isPrivate bool, err error) {
	if r.Config.AccountID == "" || r.Config.AccessKeyID == "" || r.Config.SecretAccessKey == "" || r.Config.BucketName == "" {
		return false, errors.New("Cloudflare R2 的 AccountID、AccessKey、SecretKey 和 BucketName 不能为空")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s?list-type=2&max-keys=1", r.Config.AccountID, r.Config.BucketName)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	signS3Request(req, r.Config.AccessKeyID, r.Config.SecretAccessKey, "auto", "s3")

	resp, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("连接 Cloudflare R2 失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, errors.New("Cloudflare R2 认证失败，请检查 AccountID、AccessKeyID 或 SecretAccessKey")
	}
	if resp.StatusCode == 404 {
		return false, errors.New("存储桶 (Bucket) 不存在，请检查存储桶名称")
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("R2 返回错误 (%d): %s", resp.StatusCode, string(b))
	}

	isPrivate = r.Config.PublicDomain == ""
	return isPrivate, nil
}

// UploadFile 上传文件到 R2
func (r *R2Client) UploadFile(relativePath string, data []byte, contentType string) (string, error) {
	cleanRelPath := strings.TrimPrefix(relativePath, "/")
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", r.Config.AccountID, r.Config.BucketName, cleanRelPath)
	req, err := http.NewRequest("PUT", endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)

	signS3Request(req, r.Config.AccessKeyID, r.Config.SecretAccessKey, "auto", "s3")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("R2 写入失败 (%d): %s", resp.StatusCode, string(b))
	}

	if r.Config.PublicDomain != "" {
		domain := strings.TrimSuffix(r.Config.PublicDomain, "/")
		return fmt.Sprintf("%s/%s", domain, cleanRelPath), nil
	}
	return endpoint, nil
}

// DownloadFile 从 R2 下载文件
func (r *R2Client) DownloadFile(relativePath string) ([]byte, error) {
	cleanRelPath := strings.TrimPrefix(relativePath, "/")
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", r.Config.AccountID, r.Config.BucketName, cleanRelPath)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	signS3Request(req, r.Config.AccessKeyID, r.Config.SecretAccessKey, "auto", "s3")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("R2 下载失败 (%d)", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type s3ListBucketResult struct {
	XMLName  xml.Name `xml:"ListBucketResult"`
	Contents []struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		Size         int64  `xml:"Size"`
	} `xml:"Contents"`
}

// ListFiles 列出 R2 中指定前缀的文件
func (r *R2Client) ListFiles(prefix string) ([]FileItem, error) {
	cleanPrefix := strings.TrimPrefix(prefix, "/")
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s?list-type=2&prefix=%s",
		r.Config.AccountID, r.Config.BucketName, url.QueryEscape(cleanPrefix))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	signS3Request(req, r.Config.AccessKeyID, r.Config.SecretAccessKey, "auto", "s3")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("R2 查询列表失败 (%d): %s", resp.StatusCode, string(b))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed s3ListBucketResult
	if err := xml.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("解析 R2 返回结果失败: %v", err)
	}

	var result []FileItem
	for _, it := range parsed.Contents {
		dlUrl := ""
		if r.Config.PublicDomain != "" {
			domain := strings.TrimSuffix(r.Config.PublicDomain, "/")
			dlUrl = fmt.Sprintf("%s/%s", domain, it.Key)
		} else {
			dlUrl = fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", r.Config.AccountID, r.Config.BucketName, it.Key)
		}

		t, _ := time.Parse(time.RFC3339, it.LastModified)
		result = append(result, FileItem{
			Name:        path.Base(it.Key),
			Path:        it.Key,
			Size:        it.Size,
			DownloadUrl: dlUrl,
			UpdatedAt:   t,
			IsDirectory: strings.HasSuffix(it.Key, "/"),
		})
	}
	return result, nil
}

// AWS V4 签名计算辅助函数
func signS3Request(req *http.Request, accessKey, secretKey, region, service string) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	// 1. Canonical Query String
	canonicalQuery := ""
	if req.URL.RawQuery != "" {
		params := req.URL.Query()
		var keys []string
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var queryParts []string
		for _, k := range keys {
			vals := params[k]
			for _, v := range vals {
				queryParts = append(queryParts, url.QueryEscape(k)+"="+url.QueryEscape(v))
			}
		}
		canonicalQuery = strings.Join(queryParts, "&")
	}

	// 2. Canonical Headers
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", req.URL.Host, amzDate)
	signedHeaders := "host;x-amz-date"

	payloadHash := "UNSIGNED-PAYLOAD"
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	canonicalReqHash := sha256.Sum256([]byte(canonicalRequest))

	// 3. String to sign
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/%s/aws4_request\n%s",
		amzDate,
		dateStamp,
		region,
		service,
		hex.EncodeToString(canonicalReqHash[:]),
	)

	// 4. Signing key
	h := hmac.New(sha256.New, []byte("AWS4"+secretKey))
	h.Write([]byte(dateStamp))
	kDate := h.Sum(nil)

	h = hmac.New(sha256.New, kDate)
	h.Write([]byte(region))
	kRegion := h.Sum(nil)

	h = hmac.New(sha256.New, kRegion)
	h.Write([]byte(service))
	kService := h.Sum(nil)

	h = hmac.New(sha256.New, kService)
	h.Write([]byte("aws4_request"))
	kSigning := h.Sum(nil)

	h = hmac.New(sha256.New, kSigning)
	h.Write([]byte(stringToSign))
	signature := hex.EncodeToString(h.Sum(nil))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/%s/aws4_request, SignedHeaders=%s, Signature=%s",
		accessKey, dateStamp, region, service, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

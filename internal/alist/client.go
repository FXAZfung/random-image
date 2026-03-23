// /home/user/random-image/internal/alist/client.go
package alist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".svg": true, ".ico": true,
	".avif": true,
}

// Client Alist API 客户端
type Client struct {
	baseURL        string
	token          string
	username       string
	password       string
	apiClient      *http.Client
	downloadClient *http.Client
	userAgent      string
	maxBodySize    int64

	mu         sync.RWMutex
	tokenCache string
	tokenExp   time.Time
}

// NewClient 创建 Alist 客户端
func NewClient(baseURL, token, username, password string, apiClient, downloadClient *http.Client, userAgent string, maxBodySizeMB int) *Client {
	if apiClient == nil {
		apiClient = &http.Client{Timeout: 15 * time.Second}
	}
	if downloadClient == nil {
		downloadClient = apiClient
	}
	if userAgent == "" {
		userAgent = "RandomImage/2.0"
	}

	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		token:          token,
		username:       username,
		password:       password,
		apiClient:      apiClient,
		downloadClient: downloadClient,
		userAgent:      userAgent,
		maxBodySize:    int64(maxBodySizeMB) * 1024 * 1024,
	}
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}

	c.mu.RLock()
	if c.tokenCache != "" && time.Now().Before(c.tokenExp) {
		token := c.tokenCache
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	return c.login(ctx)
}

func (c *Client) login(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenCache != "" && time.Now().Before(c.tokenExp) {
		return c.tokenCache, nil
	}

	req := LoginRequest{
		Username: c.username,
		Password: c.password,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal login request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create login request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.apiClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}

	if loginResp.Code != 200 {
		return "", fmt.Errorf("login failed: %s", loginResp.Message)
	}

	c.tokenCache = loginResp.Data.Token
	c.tokenExp = time.Now().Add(47 * time.Hour)
	return c.tokenCache, nil
}

func (c *Client) doAPI(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.apiClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return data, nil
}

// ListImages 列出指定路径下的所有图片（递归）
func (c *Client) ListImages(ctx context.Context, dirPath string) ([]string, error) {
	return c.listImagesRecursive(ctx, dirPath)
}

func (c *Client) listImagesRecursive(ctx context.Context, dirPath string) ([]string, error) {
	req := ListRequest{
		Path:    dirPath,
		Page:    1,
		PerPage: 0,
	}

	data, err := c.doAPI(ctx, http.MethodPost, "/api/fs/list", req)
	if err != nil {
		return nil, err
	}

	var listResp ListResponse
	if err := json.Unmarshal(data, &listResp); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}

	if listResp.Code != 200 {
		return nil, fmt.Errorf("list failed: %s", listResp.Message)
	}

	var images []string
	for _, item := range listResp.Data.Content {
		fullPath := path.Join(dirPath, item.Name)
		if item.IsDir {
			subImages, err := c.listImagesRecursive(ctx, fullPath)
			if err != nil {
				continue
			}
			images = append(images, subImages...)
		} else if isImageFile(item.Name) {
			images = append(images, fullPath)
		}
	}

	return images, nil
}

// GetFileURL 获取文件直链
func (c *Client) GetFileURL(ctx context.Context, filePath string) (string, error) {
	req := GetRequest{Path: filePath}

	data, err := c.doAPI(ctx, http.MethodPost, "/api/fs/get", req)
	if err != nil {
		return "", err
	}

	var getResp GetResponse
	if err := json.Unmarshal(data, &getResp); err != nil {
		return "", fmt.Errorf("decode get response: %w", err)
	}

	if getResp.Code != 200 {
		return "", fmt.Errorf("get file failed: %s", getResp.Message)
	}

	if getResp.Data.RawURL != "" {
		return getResp.Data.RawURL, nil
	}

	return fmt.Sprintf("%s/d%s", c.baseURL, filePath), nil
}

// DownloadFile 通过服务器中转下载图片
func (c *Client) DownloadFile(ctx context.Context, filePath string) ([]byte, string, error) {
	rawURL, err := c.GetFileURL(ctx, filePath)
	if err != nil {
		return nil, "", fmt.Errorf("get file url: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create download request: %w", err)
	}
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.downloadClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	reader := io.LimitReader(resp.Body, c.maxBodySize)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("read file content: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(filePath, data)
	}

	return data, contentType, nil
}

func isImageFile(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	return imageExtensions[ext]
}

func detectContentType(filePath string, data []byte) string {
	ext := strings.ToLower(path.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".avif":
		return "image/avif"
	case ".ico":
		return "image/x-icon"
	default:
		return http.DetectContentType(data)
	}
}

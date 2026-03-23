// /home/user/random-image/internal/alist/types.go
package alist

import "time"

// ==================== 认证 ====================

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

// ==================== 文件列表 ====================

// ListRequest 列出目录内容请求
type ListRequest struct {
	Path     string `json:"path"`
	Password string `json:"password,omitempty"`
	Page     int    `json:"page,omitempty"`
	PerPage  int    `json:"per_page,omitempty"`
	Refresh  bool   `json:"refresh,omitempty"`
}

// ListResponse 列出目录内容响应
type ListResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Content  []FileItem `json:"content"`
		Total    int        `json:"total"`
		Provider string     `json:"provider"`
	} `json:"data"`
}

// FileItem 单个文件/目录信息
type FileItem struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"is_dir"`
	Modified time.Time `json:"modified"`
	Sign     string    `json:"sign"`
	Thumb    string    `json:"thumb"`
	Type     int       `json:"type"`
}

// ==================== 文件详情 ====================

// GetRequest 获取文件详情请求
type GetRequest struct {
	Path     string `json:"path"`
	Password string `json:"password,omitempty"`
}

// GetResponse 获取文件详情响应
type GetResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Name     string    `json:"name"`
		Size     int64     `json:"size"`
		IsDir    bool      `json:"is_dir"`
		Modified time.Time `json:"modified"`
		Sign     string    `json:"sign"`
		Thumb    string    `json:"thumb"`
		Type     int       `json:"type"`
		RawURL   string    `json:"raw_url"`
		Provider string    `json:"provider"`
	} `json:"data"`
}

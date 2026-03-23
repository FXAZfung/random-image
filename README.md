# Random Image

一个轻量的随机图片 API 服务，使用 Go 编写。

它可以从本地目录或 Alist 中读取图片，并通过统一接口对外提供随机图片访问能力，适合用于壁纸站、图床接口、个人主页背景图、Bot 随机图片源等场景。

## 功能特性

- 支持本地目录和 Alist 两种图片来源
- 支持按分类随机返回图片
- 支持 `proxy`、`redirect`、`json` 三种返回模式
- 内置图片缓存与预取机制
- 支持限流和 IP 封禁
- 支持环境变量覆盖敏感配置
- 支持 Docker 部署
- 单二进制运行，无数据库依赖

## 适用场景

- 给前端页面提供随机背景图
- 给 Bot、插件或脚本提供图片接口
- 把 Alist 中的图片目录快速封装成 API
- 自建一个简单、稳定的随机图片服务

## 工作方式

服务启动后会扫描你配置的分类目录，并为每个分类建立图片列表。

当客户端访问 `/api/{category}` 时，程序会从对应分类中随机选择一张图片，然后按配置或请求参数决定返回方式：

- `proxy`：由服务端直接返回图片内容
- `redirect`：返回 `302` 跳转到图片原地址
- `json`：返回图片路径、分类、存储类型等信息

## 快速开始

### 1. 准备配置文件

复制示例配置：

```bash
cp config.example.yaml config.yaml
```

Windows PowerShell：

```powershell
Copy-Item config.example.yaml config.yaml
```

然后根据你的实际情况修改 `config.yaml`。

### 2. 准备图片

如果你使用本地存储，假设配置如下：

```yaml
local:
  enabled: true
  base_path: "./images"

categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
```

那么你的目录结构应类似：

```text
images/
└── wallpaper/
    ├── a.jpg
    ├── b.png
    └── c.webp
```

### 3. 启动服务

如果你已经有编译好的二进制：

```bash
./random-image
```

指定配置文件：

```bash
./random-image -config config.yaml
```

调试模式：

```bash
./random-image -debug
```

默认监听地址：

```text
http://127.0.0.1:8080
```

`-debug` 模式下还会开启 pprof，监听：

```text
http://127.0.0.1:6060
```

## 自行构建

要求：Go `1.26`

```bash
go build -o random-image .
```

如果你需要在发布时写入版本号，可使用：

```bash
go build -ldflags="-X main.version=v0.4.5" -o random-image .
```

## 最小可用配置

### 仅使用本地目录

```yaml
server:
  address: ":8080"

local:
  enabled: true
  base_path: "./images"

alist:
  enabled: false

categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
    description: "本地壁纸"
```

### 仅使用 Alist

```yaml
server:
  address: ":8080"

alist:
  enabled: true
  url: "https://alist.example.com"
  token: "your-token"
  timeout: 15s

local:
  enabled: false

categories:
  - name: "anime"
    storage: "alist"
    path: "/images/anime"
    description: "Alist 动漫图片"
```

## API

### 1. 获取随机图片

```http
GET /api/{category}
```

示例：

```bash
curl http://127.0.0.1:8080/api/wallpaper
curl -L "http://127.0.0.1:8080/api/wallpaper?type=redirect"
curl "http://127.0.0.1:8080/api/wallpaper?type=json"
```

查询参数：

| 参数 | 说明 |
| --- | --- |
| `type=proxy` | 返回图片二进制内容 |
| `type=redirect` | 返回 `302` 跳转 |
| `type=json` | 返回图片信息 JSON |

说明：

- 如果不传 `type`，默认使用配置文件中的 `relay.mode`
- 某些存储后端如果不支持跳转，`redirect` 会自动回退到 `proxy`

#### `type=json` 返回示例

```json
{
  "category": "anime",
  "path": "/images/anime/demo.jpg",
  "storage": "alist",
  "url": "https://alist.example.com/d/images/anime/demo.jpg"
}
```

### 2. 获取分类列表

```http
GET /api/categories
```

返回内容包含分类名、描述、存储类型和当前扫描到的图片数量。

### 3. 健康检查

```http
GET /health
```

返回服务状态、缓存占用、限流器统计和当前默认返回模式。

### 4. 服务信息

```http
GET /
```

返回版本号、接口列表和分类信息，适合用作服务探活或简单自检。

## 配置说明

下面只解释对用户最重要的配置项。

### `server`

```yaml
server:
  address: ":8080"
  read_timeout: 10s
  write_timeout: 30s
```

- `address`：监听地址
- `read_timeout`：读取请求超时
- `write_timeout`：写响应超时

### `local`

```yaml
local:
  enabled: true
  base_path: "./images"
```

- `enabled`：是否启用本地存储
- `base_path`：本地图片根目录

### `alist`

```yaml
alist:
  enabled: true
  url: "https://alist.example.com"
  token: ""
  username: ""
  password: ""
  timeout: 15s
```

- `url`：Alist 地址
- `token`：推荐优先使用 Token
- `username`、`password`：也可使用账号密码登录
- `timeout`：访问 Alist API 的超时时间

### `relay`

```yaml
relay:
  mode: "proxy"
  max_body_size_mb: 20
  user_agent: "RandomImage/dev"
  cache_control_max_age: 30s
```

- `mode`：默认返回模式，可选 `proxy`、`redirect`、`json`
- `max_body_size_mb`：中继下载图片时的大小限制
- `user_agent`：访问上游时使用的 UA
- `cache_control_max_age`：图片响应头的缓存时间

### `cache`

```yaml
cache:
  max_size: 256
  max_memory_mb: 512
  prefetch_count: 5
  prefetch_interval: 60s
  ttl: 30m
```

- 控制内存缓存数量、内存上限、预取数量、预取周期和缓存 TTL

### `limiter`

```yaml
limiter:
  enabled: true
  rate: 30
  burst: 10
  cleanup_interval: 5m
  ban_threshold: 100
  ban_duration: 30m
```

- `rate`：基础限流速率
- `burst`：突发请求容量
- `ban_threshold`：超过阈值后触发封禁
- `ban_duration`：封禁持续时间

### `selection`

```yaml
selection:
  avoid_repeats: 5
```

- 尽量避免连续重复命中同一张图
- 当分类图片数量较少时，程序会自动缩小这个窗口

### `startup`

```yaml
startup:
  require_ready_category: true
```

- 启动时是否要求至少有一个分类可用
- 设为 `true` 时，所有分类都扫描失败或为空会直接退出

### `categories`

```yaml
categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
    description: "本地壁纸"
```

- `name`：接口中的分类名，对应 `/api/{name}`
- `storage`：存储后端，必须是 `local` 或 `alist`
- `path`：该分类在存储中的实际路径
- `description`：分类描述，会出现在分类列表接口中

## 环境变量覆盖

敏感信息或部署差异可以通过环境变量覆盖，常用项如下：

- `RI_SERVER_ADDRESS`
- `RI_ALIST_ENABLED`
- `RI_ALIST_URL`
- `RI_ALIST_TOKEN`
- `RI_ALIST_USERNAME`
- `RI_ALIST_PASSWORD`
- `RI_ALIST_TIMEOUT`
- `RI_LOCAL_ENABLED`
- `RI_LOCAL_BASE_PATH`
- `RI_OUTBOUND_PROXY_ENABLED`
- `RI_OUTBOUND_PROXY_URL`
- `RI_RELAY_MODE`
- `RI_RELAY_USER_AGENT`
- `RI_RELAY_MAX_BODY_SIZE_MB`
- `RI_RELAY_CACHE_CONTROL_MAX_AGE`
- `RI_CACHE_MAX_SIZE`
- `RI_CACHE_MAX_MEMORY_MB`
- `RI_CACHE_PREFETCH_COUNT`
- `RI_CACHE_PREFETCH_INTERVAL`
- `RI_CACHE_TTL`
- `RI_LIMITER_ENABLED`
- `RI_LIMITER_RATE`
- `RI_LIMITER_BURST`
- `RI_LIMITER_CLEANUP_INTERVAL`
- `RI_LIMITER_BAN_THRESHOLD`
- `RI_LIMITER_BAN_DURATION`
- `RI_STARTUP_REQUIRE_READY_CATEGORY`
- `RI_SELECTION_AVOID_REPEATS`

示例：

```bash
RI_ALIST_TOKEN=your-token RI_RELAY_MODE=json ./random-image
```

PowerShell：

```powershell
$env:RI_ALIST_TOKEN = "your-token"
$env:RI_RELAY_MODE = "json"
.\random-image.exe
```

## Docker

### 构建镜像

```bash
docker build -t random-image .
```

### 运行容器

```bash
docker run --rm -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/images:/app/images \
  random-image
```

如果你在 Windows PowerShell 中运行：

```powershell
docker run --rm -p 8080:8080 `
  -v ${PWD}\config.yaml:/app/config.yaml `
  -v ${PWD}\images:/app/images `
  random-image
```

容器默认启动命令等价于：

```bash
/app/random-image -config /app/config.yaml
```

## 常见问题

### 启动后直接退出

优先检查以下几点：

- 是否至少启用了一个存储后端
- `categories` 是否至少配置了一项
- 本地目录或 Alist 路径下是否真的有图片
- `startup.require_ready_category` 是否为 `true`

### 访问分类返回 404

通常是以下原因：

- 请求的分类名没有在 `categories` 中配置
- 接口路径写错了，正确格式是 `/api/{category}`

### `redirect` 没有跳转

这是预期行为之一。如果当前存储不支持直接返回原图 URL，程序会自动退回 `proxy` 模式。

## 许可证

MIT License

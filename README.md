# Random Image

一个轻量的随机图片 API 服务，使用 Go 编写。

它可以从本地目录或 Alist 中读取图片，并通过统一接口对外提供随机图片访问能力，适合用于壁纸站、图床接口、个人主页背景图、Bot 随机图片源等场景。

## 特性

- 支持本地目录和 Alist 两种图片来源
- 支持按分类随机返回图片
- 支持 `proxy`、`redirect`、`json` 三种返回模式
- 内置图片缓存与预取机制
- 支持限流和 IP 封禁
- 支持使用环境变量覆盖敏感配置
- 支持 Docker 部署
- 单二进制运行，无数据库依赖

## 适用场景

- 给网页或应用提供随机背景图
- 为 Bot、脚本、插件提供图片接口
- 将 Alist 中的图片目录快速封装成 API
- 自建一个简单、稳定、易部署的随机图片服务

## 工作方式

服务启动后会扫描配置中的分类目录，并为每个分类建立图片列表。

当客户端访问 `/api/{category}` 时，程序会从对应分类中随机选择一张图片，再根据配置或查询参数决定返回方式：

- `proxy`：由服务端直接返回图片二进制内容
- `redirect`：返回 `302` 跳转到图片原始地址
- `json`：返回图片路径、分类、存储类型和原图地址等信息

## 快速启动

### 方式一：使用 GitHub Release 二进制文件

项目已经发布可直接运行的编译产物，适合不想自己编译的用户。

Release 页面：

- [GitHub Releases](https://github.com/FXAZfung/random-image/releases)

当前发布流程会构建以下平台：

- Linux `amd64`
- Linux `arm64`
- macOS `amd64`
- macOS `arm64`
- Windows `amd64`

下载对应平台压缩包后，解压并准备配置文件：

```bash
cp config.example.yaml config.yaml
```

Windows PowerShell：

```powershell
Copy-Item config.example.yaml config.yaml
```

如果你使用本地图片存储，准备一个类似下面的目录结构：

```text
images/
└── wallpaper/
    ├── a.jpg
    ├── b.png
    └── c.webp
```

然后启动程序：

Linux / macOS：

```bash
./random-image -config config.yaml
```

Windows PowerShell：

```powershell
.\random-image.exe -config config.yaml
```

默认监听地址：

```text
http://127.0.0.1:8080
```

### 方式二：使用 Docker

如果你更希望通过容器部署，可以直接构建镜像：

```bash
docker build -t random-image .
```

然后挂载配置文件和图片目录运行：

```bash
docker run --rm -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/images:/app/images \
  random-image
```

Windows PowerShell：

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

如果你使用 GitHub Actions 自动发布的容器镜像，也可以直接拉取：

```bash
docker pull ghcr.io/fxazfung/random-image:latest
```

```bash
docker run --rm -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/images:/app/images \
  ghcr.io/fxazfung/random-image:latest
```

### 方式三：从源码构建

要求：Go `1.26`

```bash
go build -o random-image .
```

如果你希望构建时写入版本号：

```bash
go build -ldflags="-X main.version=v0.4.5" -o random-image .
```

调试模式启动：

```bash
./random-image -config config.yaml -debug
```

启用 `-debug` 后还会开启 pprof：

```text
http://127.0.0.1:6060
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

## 配置文件说明

完整示例见 [config.example.yaml](./config.example.yaml)。

下面按配置块说明每个字段的作用、典型用途和注意点。

### `server`

```yaml
server:
  address: ":8080"
  read_timeout: 10s
  write_timeout: 30s
```

- `address`：HTTP 服务监听地址，默认 `:8080`
- `read_timeout`：读取请求超时时间
- `write_timeout`：写响应超时时间

说明：

- 如果只想监听本机，也可以改成 `127.0.0.1:8080`
- 部署到 Docker 或服务器时通常保留 `:8080` 即可

### `local`

```yaml
local:
  enabled: true
  base_path: "./images"
```

- `enabled`：是否启用本地图片存储
- `base_path`：本地图片根目录

说明：

- 分类中的 `path` 会基于这个根目录继续拼接
- 例如 `base_path` 是 `./images`，某个分类 `path` 是 `wallpaper`，实际扫描目录就是 `./images/wallpaper`

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

- `enabled`：是否启用 Alist 存储
- `url`：Alist 服务地址
- `token`：访问 Alist API 的 Token，推荐优先使用
- `username`：如果不使用 Token，也可以用账号登录
- `password`：与 `username` 配套使用
- `timeout`：访问 Alist API 的超时时间

说明：

- 启用 `alist` 时，`url` 不能为空
- 推荐优先使用 `token`，部署更方便，也更适合容器环境
- `username` / `password` 适合作为兼容方案，不建议和 `token` 混用

### `outbound_proxy`

```yaml
outbound_proxy:
  enabled: false
  url: "socks5://127.0.0.1:1080"
```

- `enabled`：是否启用出站代理
- `url`：代理地址，当前示例为 SOCKS5

说明：

- 这个配置会影响程序访问上游资源的方式，例如访问 Alist API 或下载图片
- 如果你的服务器访问 Alist 需要代理，可以启用这里

### `relay`

```yaml
relay:
  mode: "proxy"
  max_body_size_mb: 20
  user_agent: "RandomImage/dev"
  cache_control_max_age: 30s
```

- `mode`：默认返回模式，可选 `proxy`、`redirect`、`json`
- `max_body_size_mb`：代理下载图片时的单张大小限制
- `user_agent`：访问上游时使用的 User-Agent
- `cache_control_max_age`：图片响应头中的缓存时长

说明：

- `proxy` 最通用，客户端直接拿到图片内容
- `redirect` 更省服务端流量，但依赖后端存储是否支持返回原始图片地址
- `json` 适合你自己在前端或脚本里再做二次处理
- 当 `cache_control_max_age` 为 `0` 时，服务会返回 `private, no-cache, must-revalidate`

### `cache`

```yaml
cache:
  max_size: 256
  max_memory_mb: 512
  prefetch_count: 5
  prefetch_interval: 60s
  ttl: 30m
```

- `max_size`：最多缓存多少张图片
- `max_memory_mb`：缓存允许占用的最大内存
- `prefetch_count`：每次预取的图片数量
- `prefetch_interval`：预取任务执行周期
- `ttl`：缓存条目的存活时间

说明：

- 图片尺寸较大时，优先关注 `max_memory_mb`
- 请求量较高时，适当提高 `prefetch_count` 可以降低首个请求延迟
- 如果你机器内存较小，可以把 `max_size` 和 `max_memory_mb` 调低

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

- `enabled`：是否启用限流
- `rate`：基础限流速率
- `burst`：突发请求容量
- `cleanup_interval`：清理访问记录的周期
- `ban_threshold`：超过阈值后触发封禁
- `ban_duration`：封禁持续时间

说明：

- 这个配置适合直接暴露在公网时使用
- 如果你只在内网或本机使用，也可以关闭限流以减少干预
- `rate` 和 `burst` 需要结合你的实际访问量调整

### `selection`

```yaml
selection:
  avoid_repeats: 5
```

- `avoid_repeats`：尽量避免连续命中同一张图的窗口大小

说明：

- 设为 `0` 表示不做重复规避
- 如果分类内图片数量较少，程序会自动缩小这个窗口，避免无图可选

### `startup`

```yaml
startup:
  require_ready_category: true
```

- `require_ready_category`：启动时是否要求至少有一个分类可用

说明：

- 设为 `true` 时，如果所有分类扫描失败或都为空，程序会直接退出
- 如果你希望服务先启动、后续再补图片，也可以改为 `false`

### `categories`

```yaml
categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
    description: "本地壁纸"
```

- `name`：分类名称，对应接口路径 `/api/{name}`
- `storage`：该分类使用的存储后端，可选 `local` 或 `alist`
- `path`：该分类在存储中的实际路径
- `description`：分类说明，会出现在接口返回中

说明：

- `name` 建议使用稳定、简短、适合放进 URL 的名称
- `storage` 必须对应一个已启用的存储后端
- 如果分类来自本地目录，`path` 通常写相对 `base_path` 的子目录名
- 如果分类来自 Alist，`path` 通常写 Alist 中的完整目录路径

## API

### `GET /api/{category}`

获取指定分类的一张随机图片。

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

`type=json` 返回示例：

```json
{
  "category": "anime",
  "path": "/images/anime/demo.jpg",
  "storage": "alist",
  "url": "https://alist.example.com/d/images/anime/demo.jpg"
}
```

### `GET /api/categories`

获取分类列表，返回内容包含：

- 分类名称
- 描述
- 存储类型
- 当前扫描到的图片数量

### `GET /health`

健康检查接口，返回内容包含：

- 服务状态
- 缓存条目数和内存占用
- 限流器状态
- 当前默认返回模式

### `GET /`

服务信息接口，返回内容包含：

- 服务名称
- 版本号
- 可用接口列表
- 当前分类信息

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

Windows PowerShell：

```powershell
$env:RI_ALIST_TOKEN = "your-token"
$env:RI_RELAY_MODE = "json"
.\random-image.exe
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

这是预期行为之一。如果当前存储后端不支持直接返回原图 URL，程序会自动回退到 `proxy` 模式。

## License

[MIT](./LICENSE)

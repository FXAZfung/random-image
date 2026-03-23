# Random Image

# random-image (v2)

⚠️ This project has been completely rewritten.

- Legacy version: v1 branch
- This version is NOT backward compatible

一个使用 Go 编写的随机图片接口服务，支持从本地目录或 Alist 后端读取图片，并以代理、跳转或 JSON 元信息的方式返回。

## 特性

- 单二进制部署，无数据库依赖
- 同时支持 `local` 和 `alist` 两种存储后端
- 支持图片预取和内存缓存
- 支持按分类随机返回图片
- 支持限流、封禁、健康检查
- 支持 Docker 部署和 GitHub Actions 自动发布

## 项目结构

```text
.
├─ internal/
│  ├─ alist/      # Alist 客户端
│  ├─ cache/      # 图片缓存
│  ├─ config/     # 配置加载与校验
│  ├─ limiter/    # 限流与封禁
│  ├─ picker/     # 分类扫描与随机选图
│  ├─ proxy/      # 出站代理 HTTP 客户端
│  ├─ server/     # HTTP 接口
│  └─ storage/    # 存储抽象与实现
├─ .github/workflows/
├─ config.example.yaml
├─ Dockerfile
└─ main.go
```

## 快速开始

### 1. 克隆和构建

```bash
git clone https://github.com/<your-name>/random-image.git
cd random-image
go build -o random-image .
```

### 2. 准备配置

项目仓库只提交 `config.example.yaml`。实际运行时请复制为 `config.yaml`：

```bash
cp config.example.yaml config.yaml
```

然后按你的环境修改 `config.yaml`。

### 3. 运行

```bash
./random-image
```

调试模式会开启更详细的日志，并暴露本地 `pprof` 端口：

```bash
./random-image -debug
```

## 配置说明

### 最小本地存储示例

```yaml
server:
  address: ":8080"

local:
  enabled: true
  base_path: "./images"

alist:
  enabled: false

relay:
  mode: "proxy"

categories:
  - name: "wallpaper"
    storage: "local"
    path: "wallpaper"
    description: "本地壁纸"
```

### 最小 Alist 存储示例

```yaml
alist:
  enabled: true
  url: "https://alist.example.com"
  token: "your-token"

local:
  enabled: false

categories:
  - name: "anime"
    storage: "alist"
    path: "/images/anime"
    description: "动漫图片"
```

### 关键字段

- `server.address`: HTTP 监听地址，默认 `:8080`
- `alist.enabled`: 是否启用 Alist 后端
- `local.enabled`: 是否启用本地目录后端
- `outbound_proxy`: 服务端访问 Alist 或云盘时使用的代理
- `relay.mode`: 默认返回模式，可选 `proxy`、`redirect`、`json`
- `relay.cache_control_max_age`: 图片代理响应的缓存时间，配合 `ETag` / `Last-Modified` 使用
- `cache`: 缓存大小、预取数量、过期时间
- `limiter`: 限流和 IP 封禁配置
- `startup.require_ready_category`: 启动后若没有可用分类则直接失败退出
- `selection.avoid_repeats`: 每个分类短时间内尽量避免重复返回最近图片
- `categories`: 图片分类列表，`storage` 必须是 `alist` 或 `local`

敏感信息也可以通过环境变量覆盖，例如 `RI_ALIST_TOKEN`、`RI_ALIST_PASSWORD`、`RI_LOCAL_BASE_PATH`。

完整可提交示例见 `config.example.yaml`。

## API

### `GET /api/{category}`

随机返回指定分类的图片。

查询参数：

- `type=proxy`: 由服务端读取图片并直接返回
- `type=redirect`: 返回 302 跳转，只有支持直链的存储后端可用
- `type=json`: 返回图片路径和存储信息；Alist 后端可额外返回直链

示例：

```bash
curl http://localhost:8080/api/anime
curl -L "http://localhost:8080/api/anime?type=redirect"
curl "http://localhost:8080/api/anime?type=json"
```

### `GET /api/categories`

返回所有分类信息，包括名称、描述、存储类型和当前已扫描图片数量。

### `GET /health`

返回服务状态、缓存占用、限流器统计和当前默认 `relay_mode`。

### `GET /`

返回服务版本、接口入口和分类列表。

## Docker

### 本地构建

```bash
docker build -t random-image .
docker run --rm -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml random-image
```

## License

[MIT](LICENSE)

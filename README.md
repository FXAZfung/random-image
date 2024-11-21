# 随机图片服务

## 使用方法

1. 下载二进制文件
2. 修改目录下的config.yml文件

```yaml
app:
  name: "fxaz-random-image" # 应用名称 可自行修改
  version: "0.1.0" # 版本号
server:
  port: ":11451" # 端口默认开放在11451中 可自行修改 不要忘记":"！
  host: "localhost" # 主机名默认为localhost 请自行修改为你的ip地址或域名
file:
  path: "./images" # 图片存放路径 默认在当前目录下的"images"文件夹
```
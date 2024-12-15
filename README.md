# 随机图片服务

## 使用方法

1. 下载二进制文件
2. 修改目录下的config.yaml文件
3. 启动程序
    1. 在Linux环境下给予可执行权限`chmod +x random-image`
    2. 运行`./random-image`

```yaml
app:
   name: "random-image" # 应用名称 可自行修改
   version: "0.3.0" # 版本号
server:
   port: ":11451" # 端口默认开放在11451中 可自行修改 不要忘记":"！
   host: "localhost" # 主机名默认为localhost
   path: "/random" # 路径默认为"/random" 可自行修改 不要忘记"/"！
file:
   path: "./images" # 图片存放路径 默认在当前目录下的"images"文件夹
```

## 注意事项

- 此服务流量消耗为图片大小
- 如果图片目录图片很多，启动服务的时候会比较慢


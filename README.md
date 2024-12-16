# 随机图片服务

## 使用方法

1. 下载二进制文件
2. 修改目录下的config.yaml文件
3. 选择要随机的图片目录放入images文件夹
4. 启动程序
    1. 在Linux环境下给予可执行权限`chmod +x random-image`
    2. 运行`./random-image`

## 图片分类

- 请将图片放入images文件夹下的对应分类文件夹中
- 例如：images文件夹下有一个分类文件夹叫做"cat"，那么访问`http://localhost:11451/random?category=cat`就会返回cat文件夹下的图片
- 如果没有指定分类，那么会随机返回images文件夹下的图片

## 配置文件

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
  cache: 5 # 缓存的图片数量 默认为5 | 0代表不缓存
limit: # 此限流配置是针对每个IP的
  required: true # 是否开启限流 默认为true
  rate: 10 # 限制每秒请求次数 默认为10次
  bucket: 5 # 限制最多同时请求 默认为5次
```


## 注意事项

- 此服务流量消耗为图片大小
- 如果图片目录图片很多，启动服务的时候会比较慢


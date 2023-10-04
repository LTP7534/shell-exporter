# 功能说明
识别指定目录下的python 和 shell脚本，将脚本输出内容转变为prometheus metrics格式并提供metrics接口

# 使用说明
```shell
# 通过设置以下变量来配置
LISTEN_PORT  监听端口
SCRIPTS_PATH 脚本路径
INTERVAL     脚本执行的时间间隔
```

# 部署
> 修改deploy下kubernetes yaml的配置，deployment的变量 以及 servicemonitor等。
```
kubectl apply -f deploy/
```

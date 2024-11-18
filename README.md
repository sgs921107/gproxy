# 用golang启动一个http代理服务器(github.com/elazarl/goproxy)
## 通过自定义middleware对特定条件的请求进行拦截
> 由于现在大多数请求是https请求, 强制开启了对https的拦截, 使用前需提前安装  
> 通过浏览器访问$addr/ssl(如http://localhost:8080/ssl)下载证书并安装
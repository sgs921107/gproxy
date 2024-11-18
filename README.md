# 用golang启动一个http代理服务器(github.com/elazarl/goproxy)
## 通过自定义middleware对特定条件的请求进行拦截
> 默认未开启对https的拦截 若开启请配置ProxyOptions.HttpsMitm=true 开启后客户端需下载并安装证书  
> 通过浏览器访问http://yourAddr/ssl(如http://localhost:8080/ssl)下载证书并安装
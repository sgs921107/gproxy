/*
************************************************************************

	> File Name: gproxy.go
	> Author: xiangcai
	> Mail: xiangcai@gmail.com
	> Created Time: Sat 02 Nov 2024 07:42:00 PM CST

************************************************************************
*/
package gproxy

import (
	"net/http"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/sgs921107/glogging"
)

type (
	LogFields = glogging.LogrusFields
)

var (
	defaultAddr         = "0.0.0.0:8080"
	_, callerFile, _, _ = runtime.Caller(0)
	curDir              = path.Dir(callerFile)
	indexHtml           = path.Join(curDir, "./html/index.html")
	nonProxyHtml        = path.Join(curDir, "./html/nonProxy.html")
)

// 定义代理服务器接口
type ProxyServer interface {
	// 启动
	ListenAndServe()
	// 添加勾子  需要在启动前添加
	AddMiddleware(Middleware)
	GetLogger() *glogging.LogrusLogger
	// 代理服务器实例
	Proxy() *goproxy.ProxyHttpServer
}

type ProxyOptions struct {
	// 地址
	Addr      string
	Logger    *glogging.LogrusLogger
	HttpsMitm bool
}

type SimpleProxyServer struct {
	ProxyOptions
	proxy       *goproxy.ProxyHttpServer
	middlewares []Middleware
}

// 下载证书
func (p *SimpleProxyServer) certHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment;filename=gproxyCA.crt")
	w.WriteHeader(http.StatusOK)
	w.Write(goproxy.CA_CERT)
}

func (p *SimpleProxyServer) index(w http.ResponseWriter, _ *http.Request) {
	body, err := os.ReadFile(indexHtml)
	if err != nil {
		p.Logger.WithField("err", err.Error()).Error("Failed To Read Index Html")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// 健康检查
func (p *SimpleProxyServer) healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// 直接使用http访问时
// 如果不是访问下载证书的请求 则直接返回错误
func (p *SimpleProxyServer) nonProxyHandler(w http.ResponseWriter, r *http.Request) {
	p.Logger.WithFields(LogFields{
		"url":        r.URL.String(),
		"remoteAddr": r.RemoteAddr,
		"headers":    r.Header,
	}).Info("Received non-proxy request")
	if r.Method == http.MethodGet {
		switch r.URL.Path {
		case "/":
			p.index(w, r)
			return
		case "/index.html":
			p.index(w, r)
			return
		case "/ssl":
			p.certHandler(w, r)
			return
		case "/health":
			p.healthHandler(w, r)
			return
		default:
			goto nonProxy
		}
	} else {
		goto nonProxy
	}
nonProxy:
	body, err := os.ReadFile(nonProxyHtml)
	if err != nil {
		p.Logger.WithField("err", err.Error()).Error("Failed To Read Non Proxy Html")
		http.Error(w, "This server only responds to proxy requests.", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(body)
	}
}

func (p *SimpleProxyServer) GetLogger() *glogging.LogrusLogger {
	return p.Logger
}

func (p *SimpleProxyServer) Proxy() *goproxy.ProxyHttpServer {
	return p.proxy
}

// 添加中间件 对请求进行拦截操作
func (p *SimpleProxyServer) AddMiddleware(m Middleware) {
	p.middlewares = append(p.middlewares, m)
}

// 实例化并启动一个代理服务器
func (p *SimpleProxyServer) ListenAndServe() {
	proxy := goproxy.NewProxyHttpServer()
	// 格式化goproxy库中的调试日志
	proxy.Logger = p.Logger
	proxy.NonproxyHandler = http.HandlerFunc(p.nonProxyHandler)
	// 调试模式
	if p.Logger.Level.String() == "debug" {
		proxy.Verbose = true
	}
	// 加载中间件
	for _, m := range p.middlewares {
		proxy.OnRequest(goproxy.ReqConditionFunc(m.RequestCondition)).DoFunc(m.OnRequest)
		proxy.OnResponse(goproxy.RespConditionFunc(m.ResponseCondition)).DoFunc(m.OnResponse)
	}
	// 开启对https的拦截 开启后需下载并安装证书
	if p.HttpsMitm {
		// 启用 HTTPS 的 MITM 拦截
		proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	}
	server := &http.Server{
		Addr:         p.Addr,
		Handler:      proxy,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	p.Logger.Infof("Starting Proxy On %s", p.Addr)
	if err := server.ListenAndServe(); err != nil {
		p.Logger.WithField("err", err.Error()).Error("Start Proxy Failed!")
	}
}

func NewSimpleProxy(opt *ProxyOptions) ProxyServer {
	if opt.Addr == "" {
		opt.Addr = defaultAddr
	}
	if opt.Logger == nil {
		opt.Logger = glogging.NewLogrusLogging(glogging.Options{}).GetLogger()
	}
	return ProxyServer(&SimpleProxyServer{ProxyOptions: *opt})
}

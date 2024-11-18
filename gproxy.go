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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/sgs921107/glogging"
)

type (
	LogFields = glogging.LogrusFields
)

var (
	_, callerFile, _, _ = runtime.Caller(0)
	curDir              = path.Dir(callerFile)
	defaultAddr         = "0.0.0.0:8080"
	defaultCertPath     = path.Join(curDir, "./cert/ca.crt")
	defaultKeyPath      = path.Join(curDir, "./cert/ca.key")
	indexHtml           = path.Join(curDir, "./html/index.html")
	nonProxyHtml        = path.Join(curDir, "./html/nonProxy.html")
)

// 定义代理服务器接口
type ProxyServer interface {
	// 启动
	ListenAndServe()
	// 添加勾子  需要在启动前添加
	AddMiddleware(Middleware)
	Logger() *glogging.LogrusLogger
	// 代理服务器实例
	Proxy() *goproxy.ProxyHttpServer
}

type ProxyOptions struct {
	// 地址
	Addr     string `default:"0.0.0.0:8080"`
	CertPath string
	KeyPath  string
	LogLevel string `default:"INFO"`
}

type SimpleProxyServer struct {
	ProxyOptions
	logger      *glogging.LogrusLogger
	proxy       *goproxy.ProxyHttpServer
	middlewares []Middleware
}

// 验证证书公钥是否过期
func (p *SimpleProxyServer) validateCertificate(cert *x509.Certificate) error {
	now := time.Now().UTC()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("certificate is not valid. Current time: %v, Valid from: %v to %v", now, cert.NotBefore, cert.NotAfter)
	}
	return nil
}

func (p *SimpleProxyServer) setCA() error {
	if p.CertPath == "" || p.KeyPath == "" {
		p.logger.WithFields(LogFields{"certPath": p.CertPath, "keyPath": p.KeyPath}).Info("MITM For HTTPS Not Started")
		return nil
	}
	ca, err := tls.LoadX509KeyPair(p.CertPath, p.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to load x509 key pair: %w", err)
	}
	cert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	} else {
		ca.Leaf = cert
	}
	if err := p.validateCertificate(cert); err != nil {
		return err

	}
	tlsConfig := goproxy.TLSConfigFromCA(&ca)
	// 配置全局代理
	goproxy.GoproxyCa = ca
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: tlsConfig}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: tlsConfig}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: tlsConfig}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: tlsConfig}
	p.logger.WithFields(LogFields{"certPath": p.CertPath, "keyPath": p.KeyPath}).Info("Succeed To Started HTTPS MITM")
	return nil
}

// 下载证书
func (p *SimpleProxyServer) certHandler(w http.ResponseWriter, _ *http.Request) {
	if caBytes, err := os.ReadFile(p.CertPath); err != nil {
		p.logger.WithField("err", err.Error()).Error("Failed To Read Certificate!")
		http.Error(w, "Certificate Not Found!", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment;filename=gproxyCA.crt")
		w.WriteHeader(http.StatusOK)
		w.Write(caBytes)
	}
}

func (p *SimpleProxyServer) index(w http.ResponseWriter, _ *http.Request) {
	body, err := os.ReadFile(indexHtml)
	if err != nil {
		p.logger.WithField("err", err.Error()).Error("Failed To Read Index Html")
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
	p.logger.WithFields(LogFields{
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
		p.logger.WithField("err", err.Error()).Error("Failed To Read Non Proxy Html")
		http.Error(w, "This server only responds to proxy requests.", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(body)
	}
}

func (p *SimpleProxyServer) Logger() *glogging.LogrusLogger {
	return p.logger
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
	p.logger = glogging.NewLogrusLogging(glogging.Options{Level: p.LogLevel}).GetLogger()
	if err := p.setCA(); err != nil {
		p.logger.WithField("err", err.Error()).Panic("Error To Set CA!")
	}
	proxy := goproxy.NewProxyHttpServer()
	proxy.NonproxyHandler = http.HandlerFunc(p.nonProxyHandler)
	// 调试模式
	if p.LogLevel == "DEBUG" {
		proxy.Verbose = true
	}
	proxy.Logger = p.logger
	// 加载中间件
	for _, m := range p.middlewares {
		proxy.OnRequest(goproxy.ReqConditionFunc(m.RequestCondition)).DoFunc(m.OnRequest)
		proxy.OnResponse(goproxy.RespConditionFunc(m.ResponseCondition)).DoFunc(m.OnResponse)
	}
	// 启用 HTTPS 的 MITM 拦截
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	server := &http.Server{
		Addr:         p.Addr,
		Handler:      proxy,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	p.logger.Infof("Starting Proxy On %s", p.Addr)
	if err := server.ListenAndServe(); err != nil {
		p.logger.WithField("err", err.Error()).Error("Start Proxy Failed!")
	}
}

func NewSimpleProxy(opt *ProxyOptions) ProxyServer {
	if opt.Addr == "" {
		opt.Addr = defaultAddr
	}
	if opt.CertPath == "" || opt.KeyPath == "" {
		opt.CertPath = defaultCertPath
		opt.KeyPath = defaultKeyPath
	}
	opt.LogLevel = strings.ToUpper(opt.LogLevel)
	return ProxyServer(&SimpleProxyServer{ProxyOptions: *opt})
}

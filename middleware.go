/*************************************************************************
> File Name: middleware.go
> Author: sgs921107
> Mail: 757513128@gmail.com
> Created Time: 2024-11-18 11:05:40 星期一
> Content: This is a desc
*************************************************************************/

package gproxy

import (
	"github.com/elazarl/goproxy"
	"net/http"
)

// 中间件  用于对请求进行处理
type Middleware interface {
	// 请求前的勾子
	OnRequest(*http.Request, *goproxy.ProxyCtx) (*http.Request, *http.Response)
	// 请求后的勾子
	OnResponse(*http.Response, *goproxy.ProxyCtx) *http.Response
	// 执行勾子的条件
	RequestCondition(*http.Request, *goproxy.ProxyCtx) bool
	ResponseCondition(*http.Response, *goproxy.ProxyCtx) bool
}

// 基础的中间件结构体 未对请求作任何处理
type BaseMiddleware struct{}

func (m *BaseMiddleware) OnRequest(req *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return req, nil
}

func (m *BaseMiddleware) OnResponse(resp *http.Response, _ *goproxy.ProxyCtx) *http.Response {
	return resp
}

func (m *BaseMiddleware) RequestCondition(req *http.Request, _ *goproxy.ProxyCtx) bool {
	return false
}

func (m *BaseMiddleware) ResponseCondition(resp *http.Response, _ *goproxy.ProxyCtx) bool {
	return false
}

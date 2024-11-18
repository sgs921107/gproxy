/*************************************************************************
> File Name: main.go
> Author: sgs921107
> Mail: 757513128@gmail.com
> Created Time: 2024-11-18 13:45:16 星期一
> Content: This is a desc
*************************************************************************/

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/sgs921107/gproxy"
)

type EchoRespMiddleware struct {
	gproxy.BaseMiddleware
}

func (m *EchoRespMiddleware) ResponseCondition(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
	return resp != nil && resp.Body != nil
}

// 一个简单的输出resp的示例
func (m *EchoRespMiddleware) OnResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	scheme := resp.Request.URL.Scheme
	ctx.Proxy.Logger.Printf("Intercepted a %s request: %s", scheme, resp.Request.URL.String())
	if scheme == "https" {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Print(string(respBody))
		resp.Body.Close()
		//恢复响应体供后续流程使用
		resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}
	return resp
}

func main() {
	proxyServer := gproxy.NewSimpleProxy(&gproxy.ProxyOptions{})
	proxyServer.AddMiddleware(gproxy.Middleware(&EchoRespMiddleware{}))
	proxyServer.ListenAndServe()
}

package platform

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPDoer 是「只用 IPv4」的 HTTP 客户端（api-docs/001-timetable §2、§6.3）。
//
// 教务 aic 服务器 DNS 返回 AAAA 记录但没有 IPv6 监听：双栈拨号会先试 IPv6，
// 连不上只能等超时（约 10s）再回退 —— 每次请求都白等 10 秒。
// 所以这里用 `tcp4` 网络**禁用** v6，而不是「优选 v4」。
//
// 该行为只在适配器里，core 不感知（AGENTS §5.3）。
type HTTPDoer struct {
	client  *http.Client
	noRedir *http.Client
	// base 保留未包装的 Transport：安装 / 卸载请求记录时在此之上重新套一层，
	// 不让「包装」与「拆包」互相嵌套（否则重复开关会套出多层记录器）。
	base   *http.Transport
	reqLog *RequestLog
	mu     sync.Mutex
}

// NewHTTPDoer 构造适配器；timeout<=0 时用 30s。
func NewHTTPDoer(timeout time.Duration) *HTTPDoer {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 强制 IPv4：不是优选，是禁用 v6。
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   2,
		ForceAttemptHTTP2:     true,
	}
	return &HTTPDoer{client: &http.Client{Timeout: timeout, Transport: transport}, base: transport}
}

// SetRequestLog 安装请求记录器（nil 表示不记录）。
//
// 记录器只在启用时读取正文，关闭时 RoundTrip 直通，正常查询不受影响。
func (h *HTTPDoer) SetRequestLog(l *RequestLog) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reqLog = l
	if h.base != nil {
		h.client.Transport = l.Wrap(h.base)
	}
	// noRedir 是从 client 复制出来的：旧副本仍指向旧 Transport，必须重建。
	h.noRedir = nil
}

// Do 满足 core.HTTPDoer。
func (h *HTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return h.client.Do(req)
}

// noRedirectClient 用于需要逐跳读取 Set-Cookie 的场景（CAS 跨主机登录链）。
// 懒初始化，避免每个进程都多带一份连接池。
func (h *HTTPDoer) noRedirect() *http.Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.noRedir == nil {
		c := *h.client
		c.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
		h.noRedir = &c
	}
	return h.noRedir
}

// DoNoRedirect 满足 core.NoRedirectDoer：发一次请求，不跟随重定向。
func (h *HTTPDoer) DoNoRedirect(req *http.Request) (*http.Response, error) {
	return h.noRedirect().Do(req)
}

// CloseIdleConnections 供长驻进程（Windows）在空闲时释放连接；CLI 秒级退出无需调用。
func (h *HTTPDoer) CloseIdleConnections() { h.client.CloseIdleConnections() }

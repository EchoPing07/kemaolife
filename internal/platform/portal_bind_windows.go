//go:build windows

package platform

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"
)

func init() { localSourceDial = dialWithLocalSource }

var errNoLocalSource = errors.New("无可用本机源地址")

var (
	localIPMu sync.Mutex
	localIPs  []net.IP
	localIPAt time.Time
)

// localSourceIPv4s 枚举本机可绑定的 IPv4 源地址，缓存 30 秒。
//
// 代理 TUN（Clash / Mihomo 等）以 metric 0 接管默认路由后，探测地址
// 5.5.5.5 会被送去代理节点而不是被 BRAS 拦截，判态永远 offline。
// 实测把套接字绑定到物理网卡源地址后数据包不再经过 TUN，BRAS 正常
// 下发 302。这里收集候选源地址供 dialWithLocalSource 逐个尝试。
func localSourceIPv4s() []net.IP {
	localIPMu.Lock()
	defer localIPMu.Unlock()
	if localIPs != nil && time.Since(localIPAt) < 30*time.Second {
		return localIPs
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var candidates []net.IP
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			candidates = append(candidates, ipNet.IP)
		}
	}
	localIPs = filterSourceIPs(candidates)
	localIPAt = time.Now()
	return localIPs
}

// filterSourceIPs 只保留可直连校园网的 IPv4 源地址：
//   - 排除回环与链路本地地址（绑了也到不了 BRAS）；
//   - 排除 198.18.0.0/15：基准测试保留段，代理 TUN 普遍拿它做虚拟网卡
//     地址（如 198.18.0.1）。绑到它等于继续走 TUN，且 TCP 层会被代理
//     本地接住、无法靠拨号失败触发回落，必须事先剔除。
func filterSourceIPs(ips []net.IP) []net.IP {
	var out []net.IP
	seen := map[string]bool{}
	for _, raw := range ips {
		ip := raw.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip[0] == 198 && (ip[1] == 18 || ip[1] == 19) {
			continue
		}
		key := ip.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ip)
	}
	return out
}

// dialWithLocalSource 逐个绑定本机源地址拨号；全部失败才返回错误，
// 由 sourceAwareDial 回落到不绑定的普通拨号（保持无代理环境的老行为）。
// 每个源地址限时 2s：整体预算仍由 http.Client.Timeout 约束。
func dialWithLocalSource(ctx context.Context, network, address string) (net.Conn, error) {
	ips := localSourceIPv4s()
	if len(ips) == 0 {
		return nil, errNoLocalSource
	}
	var lastErr error = errNoLocalSource
	for _, ip := range ips {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// UDP（DNS 回落查询）与 TCP 的本地地址类型不同，混用会拨号失败。
		var local net.Addr = &net.TCPAddr{IP: ip}
		if strings.HasPrefix(network, "udp") {
			local = &net.UDPAddr{IP: ip}
		}
		d := net.Dialer{
			LocalAddr: local,
			Timeout:   2 * time.Second,
		}
		conn, err := d.DialContext(ctx, network, address)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

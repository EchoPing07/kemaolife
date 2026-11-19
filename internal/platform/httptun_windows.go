//go:build windows

package platform

import (
	"context"
	"net"
)

func init() { resolveHostIPv4 = resolveRealIPv4 }

// fakeIPDNS 是回落用的公共 DNS（阿里 / 腾讯）：仅当系统 DNS 被代理 TUN
// 劫持成 fake-ip 时使用，查询的域名是学校公开主机，不含个人信息。
var fakeIPDNS = []string{"223.5.5.5:53", "119.29.29.29:53"}

// systemLookupIPAddr 可注入，测试里用它模拟「系统 DNS 被劫持成 fake-ip」。
var systemLookupIPAddr = func(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

// resolveRealIPv4 先走系统解析；结果落在 fake-ip 段（198.18.0.0/15）时
// 改用绑定物理网卡源地址的 UDP 查询公共 DNS —— 绑定源地址的数据包不经过
// TUN，既绕开了 fake-ip 应答，也绕开了 TUN 对 53 端口的劫持。
func resolveRealIPv4(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := systemLookupIPAddr(ctx, host)
	if err == nil {
		if real := realIPv4s(addrs); len(real) > 0 {
			return real, nil
		}
	}
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			for _, dns := range fakeIPDNS {
				if conn, e := localSourceDial(ctx, "udp", dns); e == nil {
					return conn, nil
				}
			}
			return nil, errNoLocalSource
		},
	}
	addrs, err = r.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	return realIPv4s(addrs), nil
}

// realIPv4s 过滤出可直连的 IPv4：去掉 fake-ip 段、回环与链路本地地址。
func realIPv4s(addrs []net.IPAddr) []net.IP {
	var out []net.IP
	for _, a := range addrs {
		ip := a.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip[0] == 198 && (ip[1] == 18 || ip[1] == 19) {
			continue
		}
		out = append(out, ip)
	}
	return out
}

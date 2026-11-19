//go:build windows

package platform

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// fake-ip 段与回环 / IPv6 必须被滤掉，只留可直连的真实 IPv4。
func TestRealIPv4sFiltersFakeIP(t *testing.T) {
	addrs := []net.IPAddr{
		{IP: net.ParseIP("198.18.0.79")},
		{IP: net.ParseIP("125.216.194.67")},
		{IP: net.ParseIP("2001:250:301f::467")},
		{IP: net.ParseIP("127.0.0.1")},
		{IP: net.ParseIP("169.254.9.9")},
	}
	out := realIPv4s(addrs)
	if len(out) != 1 || out[0].String() != "125.216.194.67" {
		t.Fatalf("realIPv4s = %v, want [125.216.194.67]", out)
	}
}

// fakeDNS 是环回 UDP 上的最小 DNS 应答器：A 查询回固定 IPv4，其余类型空应答。
func fakeDNS(t *testing.T, ip net.IP) string {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if resp := fakeDNSResponse(buf[:n], ip); resp != nil {
				_, _ = conn.WriteToUDP(resp, addr)
			}
		}
	}()
	return conn.LocalAddr().String()
}

func fakeDNSResponse(query []byte, ip net.IP) []byte {
	if len(query) < 12 {
		return nil
	}
	i := 12
	for i < len(query) && query[i] != 0 {
		i += int(query[i]) + 1
	}
	if i+5 > len(query) {
		return nil
	}
	qtype := uint16(query[i+1])<<8 | uint16(query[i+2])
	resp := []byte{query[0], query[1], 0x81, 0x80, 0, 1, 0, 0, 0, 0, 0, 0}
	resp = append(resp, query[12:i+5]...)
	if qtype != 1 {
		return resp
	}
	resp[7] = 1 // ANCOUNT（头部位移 6~7，不是 QDCOUNT 的 4~5）
	resp = append(resp, 0xC0, 0x0C, 0, 1, 0, 1, 0, 0, 0, 60, 0, 4)
	return append(resp, ip.To4()...)
}

// withFakeResolver 把系统解析与源地址拨号都换成本地 fake：
// 绑定物理网卡源地址到不了环回 DNS，测试里用普通拨号代替。
func withFakeResolver(t *testing.T, lookup func(context.Context, string) ([]net.IPAddr, error), dns string) {
	t.Helper()
	oldLookup, oldDNS, oldDial := systemLookupIPAddr, fakeIPDNS, localSourceDial
	systemLookupIPAddr = lookup
	fakeIPDNS = []string{dns}
	localSourceDial = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, network, address)
	}
	t.Cleanup(func() {
		systemLookupIPAddr, fakeIPDNS, localSourceDial = oldLookup, oldDNS, oldDial
	})
}

// 系统解析被劫持成 fake-ip 时，必须经公共 DNS 复核出真实 IPv4。
func TestResolveRealIPv4FakeIPFallsBack(t *testing.T) {
	dns := fakeDNS(t, net.ParseIP("203.0.113.7"))
	withFakeResolver(t, func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("198.18.0.79")}}, nil
	}, dns)

	ips, err := resolveRealIPv4(context.Background(), "jw.example.edu.cn")
	if err != nil {
		t.Fatalf("resolveRealIPv4: %v", err)
	}
	if len(ips) != 1 || ips[0].String() != "203.0.113.7" {
		t.Fatalf("fake-ip 应回落真实 IP, got %v", ips)
	}
}

// 系统解析失败（DNS 整体被劫持 / 超时）同样走公共 DNS 回落。
func TestResolveRealIPv4SystemErrorFallsBack(t *testing.T) {
	dns := fakeDNS(t, net.ParseIP("203.0.113.7"))
	withFakeResolver(t, func(context.Context, string) ([]net.IPAddr, error) {
		return nil, errors.New("dns 超时")
	}, dns)

	ips, err := resolveRealIPv4(context.Background(), "jw.example.edu.cn")
	if err != nil || len(ips) != 1 || ips[0].String() != "203.0.113.7" {
		t.Fatalf("系统解析失败应回落真实 IP, got %v, %v", ips, err)
	}
}

// 系统解析已是真实 IP 时不得触发回落（回落 DNS 指向不可达地址也不受影响）。
func TestResolveRealIPv4KeepsRealSystemAnswer(t *testing.T) {
	withFakeResolver(t, func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("203.0.113.9")}}, nil
	}, "127.0.0.1:1")

	ips, err := resolveRealIPv4(context.Background(), "jw.example.edu.cn")
	if err != nil || len(ips) != 1 || ips[0].String() != "203.0.113.9" {
		t.Fatalf("真实系统解析应直接采用, got %v, %v", ips, err)
	}
}

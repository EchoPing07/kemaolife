//go:build windows

package platform

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestFilterSourceIPs(t *testing.T) {
	in := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("169.254.1.5"),
		net.ParseIP("198.18.0.1"),   // Clash / Mihomo TUN 常用地址
		net.ParseIP("198.19.255.2"), // 同属 198.18.0.0/15
		net.ParseIP("::1"),
		net.ParseIP("fe80::1"),
		net.ParseIP("192.0.2.10"),
		net.ParseIP("198.51.100.5"),
		net.ParseIP("192.0.2.10"), // 重复
	}
	out := filterSourceIPs(in)
	want := []string{"192.0.2.10", "198.51.100.5"}
	if len(out) != len(want) {
		t.Fatalf("候选数 = %d, want %d (%v)", len(out), len(want), out)
	}
	for i, ip := range out {
		if ip.String() != want[i] {
			t.Errorf("候选[%d] = %s, want %s", i, ip, want[i])
		}
	}
}

// 绑定拨号失败时必须回落普通拨号（无代理 / 无可用源地址环境保持老行为）。
func TestSourceAwareDialFallsBack(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	old := localSourceDial
	localSourceDial = func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("无可用源地址")
	}
	defer func() { localSourceDial = old }()

	dial := sourceAwareDial(time.Second)
	conn, err := dial(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("回落拨号失败: %v", err)
	}
	conn.Close()
}

// 绑定拨号成功时直接使用其连接，不再普通拨号。
func TestSourceAwareDialPrefersBound(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	old := localSourceDial
	localSourceDial = func(context.Context, string, string) (net.Conn, error) {
		return c1, nil
	}
	defer func() { localSourceDial = old }()

	dial := sourceAwareDial(time.Second)
	conn, err := dial(context.Background(), "tcp", "192.0.2.1:80")
	if err != nil {
		t.Fatalf("绑定拨号应成功: %v", err)
	}
	if conn != c1 {
		t.Error("应返回绑定拨号的连接")
	}
}

// dialWithLocalSource 必须按网络类型选对本地地址（TCPAddr / UDPAddr 混用会失败）。
// 把候选源地址缓存注入为环回地址，对环回监听验证真实绑定拨号。
func TestDialWithLocalSourceBindsTCPAndUDP(t *testing.T) {
	oldIPs, oldAt := localIPs, localIPAt
	localIPs, localIPAt = []net.IP{net.ParseIP("127.0.0.1")}, time.Now()
	defer func() { localIPs, localIPAt = oldIPs, oldAt }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	conn, err := dialWithLocalSource(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("TCP 绑定拨号失败: %v", err)
	}
	if conn.LocalAddr().(*net.TCPAddr).IP.String() != "127.0.0.1" {
		t.Errorf("TCP 未绑定源地址: %v", conn.LocalAddr())
	}
	conn.Close()

	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	uconn, err := dialWithLocalSource(context.Background(), "udp", udp.LocalAddr().String())
	if err != nil {
		t.Fatalf("UDP 绑定拨号失败: %v", err)
	}
	if uconn.LocalAddr().(*net.UDPAddr).IP.String() != "127.0.0.1" {
		t.Errorf("UDP 未绑定源地址: %v", uconn.LocalAddr())
	}
	uconn.Close()
}

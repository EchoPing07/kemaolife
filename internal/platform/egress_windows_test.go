//go:build windows

package platform

import (
	"net"
	"testing"
	"unsafe"
)

// 主出口源 IP 命中有线网卡才放行；无线 / 未知类型 / 未命中一律跳过。
func TestWiredEgressFor(t *testing.T) {
	ip := net.ParseIP("10.0.0.2")
	ifaces := []ifaceAddr{
		{index: 3, ips: []net.IP{net.ParseIP("192.168.1.2")}}, // Wi-Fi
		{index: 7, ips: []net.IP{net.ParseIP("10.0.0.2")}},    // 以太网
	}
	types := map[int]uint32{3: 71, 7: ifTypeEthernetCSMACD}

	if !wiredEgressFor(ip, ifaces, types) {
		t.Error("源 IP 命中以太网卡应判定为有线")
	}
	if wiredEgressFor(net.ParseIP("192.168.1.2"), ifaces, types) {
		t.Error("源 IP 命中无线网卡应判定为非有线")
	}
	if wiredEgressFor(net.ParseIP("10.0.0.99"), ifaces, types) {
		t.Error("源 IP 未命中任何网卡应判定为非有线")
	}
	if wiredEgressFor(ip, ifaces, map[int]uint32{3: 71}) {
		t.Error("缺少 IfType 记录时应保守判定为非有线")
	}
	if wiredEgressFor(ip, ifaces, nil) {
		t.Error("IfType 查询失败时应保守判定为非有线")
	}
	if wiredEgressFor(nil, ifaces, types) {
		t.Error("无源地址应判定为非有线")
	}
	// TUN 虚拟网卡即便序号命中，也不是以太网类型。
	tun := []ifaceAddr{{index: 9, ips: []net.IP{net.ParseIP("198.18.0.1")}}}
	if wiredEgressFor(net.ParseIP("198.18.0.1"), tun, map[int]uint32{9: 53}) {
		t.Error("TUN 出口不应判定为有线")
	}
}

// 头部布局必须与 Win32 一致：IfIndex 在偏移 4，Next 紧随其后，
// IfType 在 9 个指针 + 8 字节 MAC + 3 个 ULONG 之后（64 位为 100）。
// 布局错了不会编译报错，只会把无线判成有线，必须显式断言。
func TestAdapterAddressesLayout(t *testing.T) {
	var aa adapterAddresses
	base := uintptr(unsafe.Pointer(&aa))
	ptrSize := unsafe.Sizeof(uintptr(0))
	if off := uintptr(unsafe.Pointer(&aa.ifIndex)) - base; off != 4 {
		t.Errorf("IfIndex 偏移应为 4, got %d", off)
	}
	if off := uintptr(unsafe.Pointer(&aa.next)) - base; off != 8 {
		t.Errorf("Next 偏移应为 8, got %d", off)
	}
	wantIfType := uintptr(28 + 9*ptrSize)
	if off := uintptr(unsafe.Pointer(&aa.ifType)) - base; off != wantIfType {
		t.Errorf("IfType 偏移应为 %d, got %d", wantIfType, off)
	}
}

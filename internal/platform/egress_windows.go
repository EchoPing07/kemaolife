//go:build windows

package platform

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/EchoPing07/kemaolife/internal/core"
)

// LocalEgress 实现 core.WiredEgress：判断当前主出口是否为有线网卡。
//
// 不猜网卡名（「以太网 / WLAN」可被用户改名）：先用 UDP connect 取路由表
// 选定的源 IP（不发任何报文），再按 IfType 区分有线 / 无线。
// 任一步失败都返回 false —— 该端口服务于「避免顶掉宿舍路由器账号」，
// 无法确认有线时保守跳过自动认证（core.WiredEgress 注释）。
type LocalEgress struct{}

// IsWired 仅当主出口源地址落在 IfType 为以太网的网卡上时返回 true。
func (LocalEgress) IsWired() bool {
	ip := egressSourceIPv4()
	if ip == nil {
		return false
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	return wiredEgressFor(ip, collectIfaceAddrs(ifaces), adapterIfTypes())
}

// egressSourceIPv4 用 UDP connect 取主出口源地址：connect 只查路由表，
// 不发送任何数据包。目标取学校探测地址（学校协议事实，api-docs/003），
// 与判态探测同一路由口径；强制 udp4，校园网链路没有 IPv6。
func egressSourceIPv4() net.IP {
	conn, err := net.Dial("udp4", "5.5.5.5:80")
	if err != nil {
		return nil
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP
	}
	return nil
}

// ifaceAddr 是「网卡序号 → 本机 IPv4 地址」的最小记录，
// 与 Win32 查询解耦，便于纯函数单测。
type ifaceAddr struct {
	index int
	ips   []net.IP
}

func collectIfaceAddrs(ifaces []net.Interface) []ifaceAddr {
	var out []ifaceAddr
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		rec := ifaceAddr{index: ifi.Index}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipNet.IP.To4(); ip4 != nil {
					rec.ips = append(rec.ips, ip4)
				}
			}
		}
		if len(rec.ips) > 0 {
			out = append(out, rec)
		}
	}
	return out
}

// wiredEgressFor 是纯判定：源 IP 命中哪块网卡，就看那块网卡的 IfType。
// 只有明确识别为以太网（IF_TYPE_ETHERNET_CSMACD）才放行；
// 无线（IF_TYPE_IEEE80211 = 71）、TUN、未知类型一律按非有线处理。
func wiredEgressFor(ip net.IP, ifaces []ifaceAddr, ifTypes map[int]uint32) bool {
	if ip == nil {
		return false
	}
	for _, ifi := range ifaces {
		for _, addr := range ifi.ips {
			if addr.Equal(ip) {
				return ifTypes[ifi.index] == ifTypeEthernetCSMACD
			}
		}
	}
	return false
}

var (
	iphlpapi                 = syscall.NewLazyDLL("iphlpapi.dll")
	procGetAdaptersAddresses = iphlpapi.NewProc("GetAdaptersAddresses")
)

const (
	afInet           = 2
	gaaSkipAnycast   = 0x0002
	gaaSkipMulticast = 0x0004
	gaaSkipDNSServer = 0x0008

	ifTypeEthernetCSMACD = 6

	errorBufferOverflow = 111
)

// adapterAddresses 只布局解析需要的头部字段（IP_ADAPTER_ADDRESSES_LH）。
// 实际结构更长：缓冲区按 API 回写的尺寸分配，遍历靠 Next 指针，
// 不依赖本结构的 Sizeof。字段顺序与对齐必须与 Win32 头文件一致。
type adapterAddresses struct {
	length                uint32
	ifIndex               uint32
	next                  *adapterAddresses
	adapterName           *byte
	firstUnicastAddress   uintptr
	firstAnycastAddress   uintptr
	firstMulticastAddress uintptr
	firstDnsServerAddress uintptr
	dnsSuffix             *uint16
	description           *uint16
	friendlyName          *uint16
	physicalAddress       [8]byte
	physicalAddressLength uint32
	flags                 uint32
	mtu                   uint32
	ifType                uint32
	operStatus            uint32
}

// adapterIfTypes 枚举网卡 IfIndex → IfType。Go 的 net.Interface.Index
// 在 Windows 上就是 IfIndex，两边据此对齐。失败返回 nil（判定侧保守跳过）。
func adapterIfTypes() map[int]uint32 {
	size := uint32(16 << 10)
	for attempt := 0; attempt < 3; attempt++ {
		// 用 uint64 切片保证 8 字节对齐：结构首部含指针与 ULONGLONG。
		buf := make([]uint64, int(size)/8+1)
		r, _, _ := procGetAdaptersAddresses.Call(
			uintptr(afInet),
			uintptr(gaaSkipAnycast|gaaSkipMulticast|gaaSkipDNSServer),
			0,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
		)
		if r == errorBufferOverflow {
			continue // size 已被回写为所需尺寸，扩容重试
		}
		if r != 0 {
			return nil
		}
		out := make(map[int]uint32)
		for aa := (*adapterAddresses)(unsafe.Pointer(&buf[0])); aa != nil; aa = aa.next {
			out[int(aa.ifIndex)] = aa.ifType
		}
		return out
	}
	return nil
}

var _ core.WiredEgress = LocalEgress{}

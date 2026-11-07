//go:build !linux && !darwin

package platform

import "fmt"

// disableEcho 在其余平台不实现终端回显控制。
//
// CLI 版已支持 linux / darwin，分别由 termios_linux.go / termios_darwin.go 提供；
// 其余平台（含 Windows CLI 后续）暂不实现，ReadSecret 会降级（000 §5 不静默降级：
// 交互录口令时直接报错，echoOff=false 的管道输入路径不受影响）。
// 这里保留可编译的桩，让桌面两端 / Android 交叉编译不必为终端细节加分支。
func disableEcho(uintptr) (func(), error) {
	return nil, fmt.Errorf("当前平台不支持关闭终端回显")
}

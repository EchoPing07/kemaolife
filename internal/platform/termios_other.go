//go:build !linux

package platform

import "fmt"

// disableEcho 在非 Linux 平台不实现终端回显控制。
//
// Linux CLI 是唯一交付目标（000 §3.1）；这里保留可编译的降级实现，
// 让「桌面两端交叉编译」的自检不必为终端细节加 build tag 分支。
func disableEcho(uintptr) (func(), error) {
	return nil, fmt.Errorf("当前平台不支持关闭终端回显")
}

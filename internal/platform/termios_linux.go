//go:build linux

package platform

import (
	"syscall"
	"unsafe"
)

// ioctlTermios 是 Linux 的终端属性控制号；禁用 / 恢复回显都走这里。
//
// 用 syscall + unsafe 直接调 ioctl，避免为此引入 golang.org/x/term
// （AGENTS A5：标准库能解决的不新增依赖）。
const (
	ioctlReadTermios  = 0x5401 // TCGETS
	ioctlWriteTermios = 0x5402 // TCSETS
	echoFlag          = 0x00000008
)

// disableEcho 关闭 fd 上的回显，返回恢复函数。
// 失败时返回错误，由调用方决定降级（不 panic）。
func disableEcho(fd uintptr) (restore func(), err error) {
	var oldState syscall.Termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd,
		ioctlReadTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	newState := oldState
	newState.Lflag &^= echoFlag
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd,
		ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	return func() {
		syscall.Syscall6(syscall.SYS_IOCTL, fd,
			ioctlWriteTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0)
	}, nil
}

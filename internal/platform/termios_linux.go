//go:build linux

package platform

import (
	"syscall"
	"unsafe"
)

// Linux 的终端属性控制号是 TCGETS / TCSETS，与 macOS 的 BSD termios
// （TIOCGETA / TIOCSETA）不同，因此本文件与 termios_darwin.go 各自独立。
//
// 用 syscall + unsafe 直接调 ioctl，避免为此引入 golang.org/x/term
// （AGENTS A5：标准库能解决的不新增依赖）。
const (
	ioctlReadTermios  = 0x5401 // TCGETS
	ioctlWriteTermios = 0x5402 // TCSETS
	echoFlag          = 0x00000008
)

// disableEcho 关闭 fd 上的回显，返回恢复函数。
// 失败时返回错误，由 ReadSecret 决定是否警告降级（不 panic）。
func disableEcho(fd uintptr) (restore func(), err error) {
	var oldState syscall.Termios
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd,
		ioctlReadTermios, uintptr(unsafe.Pointer(&oldState)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	newState, restoreState := echoOffTermios(oldState)
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd,
		ioctlWriteTermios, uintptr(unsafe.Pointer(&newState)), 0, 0, 0); errno != 0 {
		return nil, errno
	}

	return func() {
		// 恢复失败只忽略 errno：口令已经读完，不能让恢复错误变成一次带凭据的重试。
		_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, fd,
			ioctlWriteTermios, uintptr(unsafe.Pointer(&restoreState)), 0, 0, 0)
	}, nil
}

// echoOffTermios 是纯逻辑：清掉 ECHO 位，并给出恢复要写回的原始状态。
// 抽出来是为了让位运算不用真实 tty / ioctl 就能测（AGENTS §7）。
func echoOffTermios(old syscall.Termios) (newState, restoreState syscall.Termios) {
	newState = old
	newState.Lflag &^= echoFlag
	return newState, old
}

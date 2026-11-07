//go:build darwin

package platform

import (
	"syscall"
	"unsafe"
)

// macOS 用 BSD termios：读写终端属性走 TIOCGETA / TIOCSETA，ioctl 号与
// Linux 的 TCGETS / TCSETS 不同，因此不能复用 termios_linux.go 的常量。
// 常量一律取标准库 syscall，不引 golang.org/x/term
// （AGENTS A5：标准库能解决的不新增第三方依赖）；凭据目录也沿用
// ~/.config/kemaolife / 0600 口径，不引 Keychain。
//
// disableEcho 关闭 fd 上的回显，返回恢复函数。
// 失败时返回错误，由 ReadSecret 决定警告降级（不 panic）。
func disableEcho(fd uintptr) (restore func(), err error) {
	var oldState syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd,
		syscall.TIOCGETA, uintptr(unsafe.Pointer(&oldState))); errno != 0 {
		return nil, errno
	}

	newState, restoreState := echoOffTermios(oldState)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd,
		syscall.TIOCSETA, uintptr(unsafe.Pointer(&newState))); errno != 0 {
		return nil, errno
	}

	return func() {
		// 恢复失败只忽略 errno，与 Linux 版一致：口令已经读完，
		// 不能让恢复错误变成一次带凭据的重试。
		_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, fd,
			syscall.TIOCSETA, uintptr(unsafe.Pointer(&restoreState)))
	}, nil
}

// echoOffTermios 是纯逻辑：清掉 ECHO 位，并给出恢复要写回的原始状态。
// 抽出来是为了让位运算不用真实 tty / ioctl 就能测（AGENTS §7）；
// 真正的 ioctl 行为仍需 macOS 真机验证。
func echoOffTermios(old syscall.Termios) (newState, restoreState syscall.Termios) {
	newState = old
	newState.Lflag &^= syscall.ECHO
	return newState, old
}

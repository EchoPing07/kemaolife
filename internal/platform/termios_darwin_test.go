//go:build darwin

package platform

import (
	"syscall"
	"testing"
)

// 纯逻辑测试：不碰真实 tty、不调 ioctl（AGENTS §7）。
// 真正的 TIOCGETA / TIOCSETA 行为仍需 macOS 真机验证。
func TestEchoOffTermiosDarwin(t *testing.T) {
	old := syscall.Termios{
		Iflag:  0x1,
		Oflag:  0x2,
		Cflag:  0x4,
		Lflag:  syscall.ECHO | 0x2, // ECHO 与一个无关位同时置位
		Ispeed: 9600,
		Ospeed: 9600,
	}
	old.Cc[0] = 0x11

	newState, restoreState := echoOffTermios(old)
	if newState.Lflag&syscall.ECHO != 0 {
		t.Error("ECHO 位应被清除")
	}
	if newState.Lflag != old.Lflag&^syscall.ECHO {
		t.Errorf("Lflag = %#x, want %#x", newState.Lflag, old.Lflag&^syscall.ECHO)
	}
	if newState.Iflag != old.Iflag || newState.Oflag != old.Oflag || newState.Cflag != old.Cflag ||
		newState.Cc != old.Cc || newState.Ispeed != old.Ispeed || newState.Ospeed != old.Ospeed {
		t.Error("除 Lflag 外不应改动任何字段")
	}
	if restoreState != old {
		t.Errorf("恢复状态必须完整写回 oldState: %+v", restoreState)
	}
}

// 常量本身取自标准库；此处断言它们在 darwin/amd64 与 darwin/arm64
// 上都是已编码好的 ioctl 请求字，且 ECHO 是单个位。
// 依据：GOROOT/src/syscall/zerrors_darwin_{amd64,arm64}.go、
// zsysnum_darwin_{amd64,arm64}.go（go1.24.0 与本机 go1.26 一致）。
func TestDarwinTermiosConstants(t *testing.T) {
	if syscall.SYS_IOCTL != 54 {
		t.Errorf("SYS_IOCTL = %d, want 54", syscall.SYS_IOCTL)
	}
	if syscall.TIOCGETA == 0 || syscall.TIOCSETA == 0 || syscall.TIOCGETA == syscall.TIOCSETA {
		t.Errorf("TIOCGETA/TIOCSETA = %#x/%#x", syscall.TIOCGETA, syscall.TIOCSETA)
	}
	if syscall.ECHO == 0 || syscall.ECHO&(syscall.ECHO-1) != 0 {
		t.Errorf("ECHO 必须是单个位: %#x", syscall.ECHO)
	}
	// &^= syscall.ECHO 在 uint64 Lflag 上必须能编译并按位生效。
	var state syscall.Termios
	state.Lflag = ^uint64(0)
	state.Lflag &^= syscall.ECHO
	if state.Lflag == ^uint64(0) || state.Lflag&syscall.ECHO != 0 {
		t.Errorf("Lflag = %#x", state.Lflag)
	}
}

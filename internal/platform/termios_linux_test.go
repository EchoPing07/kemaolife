//go:build linux

package platform

import (
	"syscall"
	"testing"
)

// 与 darwin 侧同构的纯逻辑测试：Linux 用 TCGETS/TCSETS + 本地 echoFlag，
// 位运算与完整恢复状态不依赖真实 tty（AGENTS §7）。
func TestEchoOffTermiosLinux(t *testing.T) {
	old := syscall.Termios{
		Iflag: 0x1,
		Oflag: 0x2,
		Cflag: 0x4,
		Lflag: echoFlag | 0x2,
	}
	old.Cc[0] = 0x11

	newState, restoreState := echoOffTermios(old)
	if newState.Lflag&echoFlag != 0 || newState.Lflag != old.Lflag&^echoFlag {
		t.Errorf("Lflag = %#x, want %#x", newState.Lflag, old.Lflag&^echoFlag)
	}
	if newState.Iflag != old.Iflag || newState.Oflag != old.Oflag || newState.Cflag != old.Cflag ||
		newState.Cc != old.Cc {
		t.Error("除 Lflag 外不应改动任何字段")
	}
	if restoreState != old {
		t.Errorf("恢复状态必须完整写回 oldState: %+v", restoreState)
	}
}

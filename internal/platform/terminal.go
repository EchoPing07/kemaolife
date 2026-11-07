package platform

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// disableEchoFn / echoWarningSink 可注入，是为了在不碰真实 tty、真实 stderr 的
// 前提下测「关闭回显失败」的降级路径（AGENTS §7：测试不依赖真实终端）。
var (
	disableEchoFn             = disableEcho
	echoWarningSink io.Writer = os.Stderr
)

// 回显关不掉时不能静默继续，否则提示说「不回显」而屏幕却明文显示口令。
// 只警告「可能被看到」，不写口令内容、不带上游 errno 细节（P2）。
const echoWarning = "警告：无法关闭终端回显，接下来输入的口令可能被看到；请确认周围环境安全\n"

// IsTerminal 判断 stdin 是否为终端。
//
// 交给 platform 是因为它要碰平台细节；app / core 不感知（AGENTS A2）。
func IsTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ReadSecret 从 r 读一行口令。
//
// 支持传入共享的 *bufio.Reader：config init 会先读学号再读密码，
// 各自新建 reader 会把下一行提前缓冲掉，导致「密码读不到」。
// echoOff 表示是否关闭终端回显（只有 stdin 真的是终端时才该为 true）。
//
// 关闭失败时的取舍：macOS 的 ioctl 本期没有真机验证，若直接拒绝读取，
// 某些受限 pty / 虚拟终端会让 config init 完全无法录入凭据；因此改为
// 在 stderr 给出不带个人信息的明确警告后继续读，同时把交互提示改成中性文案，
// 不再承诺「不回显」（000 §5 不静默降级）。agent 的管道输入 echoOff=false，
// 完全不走这条分支。密码用完即弃，不落盘、不进日志。
func ReadSecret(r io.Reader, echoOff bool) (string, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	if echoOff {
		restore, err := disableEchoFn(os.Stdin.Fd())
		if err != nil {
			_, _ = fmt.Fprint(echoWarningSink, echoWarning)
		} else {
			defer restore()
		}
	}
	line, err := br.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

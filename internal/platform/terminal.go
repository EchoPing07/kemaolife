package platform

import (
	"bufio"
	"io"
	"os"
	"strings"
)

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
// 密码用完即弃，不落盘、不进日志。
func ReadSecret(r io.Reader, echoOff bool) (string, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	if echoOff {
		if restore, err := disableEcho(os.Stdin.Fd()); err == nil {
			defer restore()
		}
	}
	line, err := br.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

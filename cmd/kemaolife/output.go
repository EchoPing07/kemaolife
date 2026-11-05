package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// stdout / stderr 在测试里被替换，避免直接用 fmt.Print*（AGENTS A2 的精神：
// 输出只出现在最外层，且可被断言）。
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

// writeJSON 输出 §2 信封到 stdout，不换行美化（便于 agent 流式解析）。
func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(stderr, "输出编码失败:", err)
	}
}

func writeStdout(s string) { fmt.Fprint(stdout, s) }
func writeStderr(s string) { fmt.Fprintln(stderr, s) }

// 表格输出只做「对齐 + 不省略字段」，不做富文本 / TUI（AGENTS §10）。

// renderTable 用等宽填充输出表格；列宽按显示宽度算（中文按 2 计）。
func renderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = displayWidth(h)
	}
	for _, row := range rows {
		for i := 0; i < len(row) && i < len(widths); i++ {
			if w := displayWidth(row[i]); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	writeRow := func(cells []string) {
		for i := range headers {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			b.WriteString(cell)
			if pad := widths[i] - displayWidth(cell); pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
			if i < len(headers)-1 {
				b.WriteString("  ")
			}
		}
		b.WriteString("\n")
	}
	writeRow(headers)
	b.WriteString(strings.Repeat("-", totalWidth(widths)) + "\n")
	for _, row := range rows {
		writeRow(row)
	}
	return b.String()
}

func totalWidth(widths []int) int {
	sum := 0
	for _, w := range widths {
		sum += w
	}
	if len(widths) > 1 {
		sum += 2 * (len(widths) - 1)
	}
	return sum
}

// displayWidth 估算终端显示宽度：CJK 与全角字符记 2。
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r >= 0x2E80 && r <= 0xA4CF,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF00 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6:
		return true
	}
	return false
}

// envelope 是子命令输出成功结果的统一入口：
// --json 走 §2 信封，否则走人类可读文本（字段与 JSON 同源，不加额外信息）。
func envelope(opts options, env model.Envelope, plain func()) error {
	if opts.json {
		writeJSON(stdout, env)
		return nil
	}
	plain()
	return nil
}

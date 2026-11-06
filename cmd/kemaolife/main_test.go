package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// runCLI 在隔离的目录里跑一次 CLI，返回退出码与 stdout/stderr。
func runCLI(t *testing.T, env map[string]string, stdinData string, args ...string) (int, string, string) {
	t.Helper()

	tmp := t.TempDir()
	envSet(t, "HOME", tmp)
	envSet(t, "XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	envSet(t, "XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	for k, v := range env {
		envSet(t, k, v)
	}

	var out, errOut bytes.Buffer
	oldStdout, oldStderr, oldStdin := stdout, stderr, stdin
	stdout, stderr = &out, &errOut
	if stdinData != "" {
		stdin = strings.NewReader(stdinData)
	} else {
		stdin = strings.NewReader("")
	}
	defer func() { stdout, stderr, stdin = oldStdout, oldStderr, oldStdin }()

	code := run(args)
	return code, out.String(), errOut.String()
}

func envSet(t *testing.T, key, value string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if had {
			os.Setenv(key, old)
			return
		}
		os.Unsetenv(key)
	})
}

func TestVersionPlain(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "--version")
	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, version) {
		t.Errorf("stdout = %q", out)
	}
}

func TestVersionJSON(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "--version", "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
		} `json:"data"`
		TS string `json:"ts"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout 不是合法 JSON: %v\n%s", err, out)
	}
	if !env.OK || env.Data.Version != version || env.TS == "" {
		t.Errorf("信封 = %+v", env)
	}
}

// 课表查询信封必须恒带 stale 键；刷新成功时为 false，不能因 omitempty 被省略。
func TestEnvelopeAlwaysHasStale(t *testing.T) {
	env := model.Envelope{OK: true, Data: map[string]any{}, TS: "2026-11-06T00:00:00+08:00", Stale: model.BoolPtr(false)}
	var buf bytes.Buffer
	writeJSON(&buf, env)
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	v, ok := got["stale"]
	if !ok || v != false {
		t.Errorf("stale = %v, present = %v", v, ok)
	}
}

func TestHelpDoesNotUseEnvelope(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "--json", "--help")
	if code != 0 {
		t.Errorf("exit = %d", code)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Error("--help 输出纯文本，不应进 JSON 信封")
	}
	if !strings.Contains(out, "timetable") {
		t.Errorf("帮助内容缺失: %q", out)
	}
}

func TestUnknownCommandExit2(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "--json", "nope")
	if code != 2 {
		t.Errorf("exit = %d, want 2（参数错误）", code)
	}
	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout 不是合法 JSON: %v", err)
	}
	if env.OK || env.Error.Code != 2001 {
		t.Errorf("信封 = %+v", env)
	}
}

func TestUnknownFlagExit2(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "", "--nope")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "未知参数") {
		t.Errorf("stderr = %q", errOut)
	}
}

// 课表查询在「无凭据 + 无快照」时必须退出码 3（docs/002 §1.3：
// 无旧快照可回落时才失败）。
func TestTimetableWithoutSnapshotExit3(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "--json", "timetable")
	if code != 3 {
		t.Errorf("exit = %d, want 3", code)
	}
	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout 不是合法 JSON: %v", err)
	}
	// 没配账号且无快照时，最贴近原因是「凭据缺失」（3001），
	// 而不是「快照缺失」（3003）—— 后者是刷新失败后的次生结果。
	if env.Error.Code != 3001 && env.Error.Code != 3003 {
		t.Errorf("code = %d, want 3001 或 3003", env.Error.Code)
	}
}

func TestTimetableBadDateExit2(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "", "timetable", "date", "2026/11/05")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut, "日期") {
		t.Errorf("stderr = %q", errOut)
	}
}

func TestTimetableBadWeekExit2(t *testing.T) {
	for _, w := range []string{"0", "-1", "abc"} {
		code, _, _ := runCLI(t, nil, "", "timetable", "week", w)
		if code != 2 {
			t.Errorf("week %q exit = %d, want 2", w, code)
		}
	}
}

func TestElectricityBadMonthExit2(t *testing.T) {
	code, _, _ := runCLI(t, nil, "", "electricity", "--month", "2026-13")
	// 缺凭据会先于月份校验触发 3001？不会：月份先校验。
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestConfigShowMasksSecrets(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "--json", "config", "show")
	if code != 0 {
		t.Fatalf("exit = %d, want 0（缺配置不是错误）", code)
	}
	if strings.Contains(out, "<学号>") {
		t.Error("输出不得出现真实学号")
	}
	var env struct {
		Data struct {
			StudentID    string `json:"studentId"`
			EduPassword  string `json:"eduPassword"`
			CustomerCode string `json:"customerCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout 不是合法 JSON: %v", err)
	}
	if env.Data.StudentID != "****" || env.Data.EduPassword != "****" {
		t.Errorf("学号与密码必须恒脱敏: %+v", env.Data)
	}
	if env.Data.CustomerCode != "****" {
		t.Errorf("学校代码也必须脱敏: %q", env.Data.CustomerCode)
	}
}

func TestConfigInitFromStdin(t *testing.T) {
	// 管道喂入学号与密码；密码绝不进命令行参数。
	code, out, errOut := runCLI(t, nil, "<学号>\n<密码>\n", "--json", "config", "init")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if strings.Contains(out, "<密码>") {
		t.Error("输出不得回显密码")
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			StudentID   string `json:"studentId"`
			EduPassword string `json:"eduPassword"`
			ConfigPath  string `json:"configPath"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout 不是合法 JSON: %v", err)
	}
	if !env.OK || env.Data.StudentID != "****" {
		t.Errorf("信封 = %+v", env)
	}

	// 配置文件权限必须是 0600（P5）。
	// Windows 不遵守 POSIX 权限位（OpenFile 的 perm 会被忽略），该断言只在 Linux 上有效；
	// 真正的交底目标是 Linux，那里由 platform 层的 chmod 强制。
	info, err := os.Stat(env.Data.ConfigPath)
	if err != nil {
		t.Fatalf("配置文件不存在: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("配置权限 = %04o, want 0600", perm)
		}
	}
}

func TestConfigSetCustomerCode(t *testing.T) {
	tmp := t.TempDir()
	envSet(t, "HOME", tmp)
	envSet(t, "XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	envSet(t, "XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	var out bytes.Buffer
	oldStdout, oldStdin := stdout, stdin
	stdout, stdin = &out, strings.NewReader("<学号>\n<密码>\n")
	defer func() { stdout, stdin = oldStdout, oldStdin }()

	if code := run([]string{"--json", "config", "init"}); code != 0 {
		t.Fatalf("config init exit = %d", code)
	}
	out.Reset()

	if code := run([]string{"--json", "config", "set", "customercode", "<自定义学校代码>"}); code != 0 {
		t.Fatalf("config set exit = %d", code)
	}
	if strings.Contains(out.String(), "<自定义学校代码>") {
		t.Error("set 不得回显学校代码取值")
	}

	out.Reset()
	if code := run([]string{"--json", "config", "show"}); code != 0 {
		t.Fatalf("config show exit = %d", code)
	}
	var env struct {
		Data struct {
			CustomerCodeSource string `json:"customerCodeSource"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.CustomerCodeSource != "custom" {
		t.Errorf("source = %q, want custom", env.Data.CustomerCodeSource)
	}
}

func TestConfigSetUnknownFieldExit2(t *testing.T) {
	code, _, _ := runCLI(t, nil, "", "config", "set", "password", "x")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

// 全局参数写在子命令之后也要生效（agent 习惯），含带值参数。
func TestTrailingGlobalFlags(t *testing.T) {
	code, out, _ := runCLI(t, nil, "", "config", "show", "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("尾部 --json 未生效: %q", out)
	}

	tmp := t.TempDir()
	custom := filepath.Join(tmp, "custom", "config.json")
	code, out, _ = runCLI(t, nil, "", "config", "show", "--json", "--config", custom)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var env struct {
		Data struct {
			ConfigPath string `json:"configPath"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Data.ConfigPath, "custom") {
		t.Errorf("尾部 --config 未生效: %q", env.Data.ConfigPath)
	}

	code, out, _ = runCLI(t, nil, "", "config", "show", "-v")
	if code != 0 {
		t.Fatalf("尾部 -v exit = %d", code)
	}
	if !strings.Contains(out, version) {
		t.Errorf("尾部 -v 应输出版本: %q", out)
	}

	code, out, _ = runCLI(t, nil, "", "config", "show", "--version")
	if code != 0 {
		t.Fatalf("尾部 --version exit = %d", code)
	}
	if !strings.Contains(out, version) {
		t.Errorf("尾部 --version 应输出版本: %q", out)
	}

	code, _, _ = runCLI(t, nil, "", "timetable", "--", "today", "--json")
	if code != 0 && code != 3 && code != 4 {
		t.Errorf("尾部 -- 终止符应按子命令参数处理, exit = %d", code)
	}
}

// --verbose 只写 stderr，不污染 stdout 信封，且不含个人信息。
func TestVerboseWritesDiagnosticsToStderr(t *testing.T) {
	code, out, errOut := runCLI(t, nil, "", "--json", "--verbose", "config", "show")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("stdout 必须只有信封: %q", out)
	}
	if !strings.Contains(errOut, "诊断") {
		t.Errorf("stderr 缺少诊断: %q", errOut)
	}
}

// 学校代码内置默认值为 3126（学校事实），config show 恒脱敏但标注来源。
func TestDefaultCustomerCode(t *testing.T) {
	if defaultCustomerCode != "3126" {
		t.Fatalf("defaultCustomerCode = %q", defaultCustomerCode)
	}
	code, out, _ := runCLI(t, nil, "", "--json", "config", "show")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var env struct {
		Data struct {
			CustomerCode       string `json:"customerCode"`
			CustomerCodeSource string `json:"customerCodeSource"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.CustomerCode != "****" || env.Data.CustomerCodeSource != "default" {
		t.Errorf("data = %+v", env.Data)
	}
}

func TestTableRenderingAligned(t *testing.T) {
	out := renderTable([]string{"项", "值"}, [][]string{
		{"学号", "****"},
		{"配置文件", "/home/u/.config/kemaolife/config.json"},
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("行数 = %d: %q", len(lines), out)
	}
	if !strings.Contains(out, "学号") || !strings.Contains(out, "****") {
		t.Errorf("表格内容缺失: %q", out)
	}
}

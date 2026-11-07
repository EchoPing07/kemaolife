package platform

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 注入桩而不是真实 tty：关闭回显失败时不能静默读口令，但也不能泄露口令内容。
// 这些用例不调 ioctl、不碰终端（AGENTS §7）。
func TestReadSecretEchoOffFailureWarnsAndReads(t *testing.T) {
	oldFn, oldSink := disableEchoFn, echoWarningSink
	t.Cleanup(func() { disableEchoFn, echoWarningSink = oldFn, oldSink })

	var warned bytes.Buffer
	echoWarningSink = &warned
	disableEchoFn = func(uintptr) (func(), error) {
		return nil, errors.New("ioctl 不可用")
	}

	got, err := ReadSecret(strings.NewReader("<密码>\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<密码>" {
		t.Errorf("ReadSecret = %q", got)
	}
	if !strings.Contains(warned.String(), "可能被看到") {
		t.Errorf("缺少明确警告: %q", warned.String())
	}
	if strings.Contains(warned.String(), "<密码>") || strings.Contains(warned.String(), "ioctl 不可用") {
		t.Errorf("警告不得含口令或上游细节: %q", warned.String())
	}
}

func TestReadSecretEchoOffSuccessRestores(t *testing.T) {
	oldFn, oldSink := disableEchoFn, echoWarningSink
	t.Cleanup(func() { disableEchoFn, echoWarningSink = oldFn, oldSink })

	var warned bytes.Buffer
	restored := false
	echoWarningSink = &warned
	disableEchoFn = func(uintptr) (func(), error) {
		return func() { restored = true }, nil
	}

	got, err := ReadSecret(strings.NewReader("<密码>\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<密码>" || !restored {
		t.Errorf("secret = %q, restored = %v", got, restored)
	}
	if warned.Len() != 0 {
		t.Errorf("成功时不应警告: %q", warned.String())
	}
}

// agent 的管道输入路径：echoOff=false 时完全不碰 disableEcho，也不写警告。
func TestReadSecretPipeInputNeverCallsDisableEcho(t *testing.T) {
	oldFn, oldSink := disableEchoFn, echoWarningSink
	t.Cleanup(func() { disableEchoFn, echoWarningSink = oldFn, oldSink })

	var warned bytes.Buffer
	echoWarningSink = &warned
	disableEchoFn = func(uintptr) (func(), error) {
		t.Fatal("echoOff=false 不应调用 disableEcho")
		return nil, nil
	}

	got, err := ReadSecret(strings.NewReader("<密码>\r\n"), false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<密码>" || warned.Len() != 0 {
		t.Errorf("secret = %q, warn = %q", got, warned.String())
	}
}

// 共用 bufio.Reader 的既有约束不能被注入改动破坏。
func TestReadSecretSharedReader(t *testing.T) {
	oldFn, oldSink := disableEchoFn, echoWarningSink
	t.Cleanup(func() { disableEchoFn, echoWarningSink = oldFn, oldSink })

	echoWarningSink = &bytes.Buffer{}
	disableEchoFn = func(uintptr) (func(), error) { return func() {}, nil }

	br := bufio.NewReader(strings.NewReader("<学号>\n<密码>\n"))
	first, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadSecret(br, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(first, "\r\n") != "<学号>" || second != "<密码>" {
		t.Errorf("first = %q, second = %q", first, second)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(filepath.Join(dir, "store"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("timetable/snapshot", []byte(`{"version":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("timetable/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"version":1}` {
		t.Errorf("Get = %q", got)
	}
	if err := store.Delete("timetable/snapshot"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("timetable/snapshot"); err == nil {
		t.Error("删除后应读不到")
	}
}

// 快照与配置都含凭据，权限必须是 0600（P5）。Windows 不遵守 POSIX 权限位。
func TestFileStorePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不支持 POSIX 权限位")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "store")
	store, err := NewFileStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("snapshot", []byte("x")); err != nil {
		t.Fatal(err)
	}

	rootInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if perm := rootInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("存储目录权限 = %04o, want 0700", perm)
	}
	perm, _, err := store.Stat("snapshot")
	if err != nil {
		t.Fatal(err)
	}
	if perm != 0o600 {
		t.Errorf("文件权限 = %04o, want 0600", perm)
	}
}

// 键不能逃出根目录。
func TestFileStoreRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(filepath.Join(dir, "store"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape", "a/../../escape", ""} {
		if err := store.Put(key, []byte("x")); err == nil {
			t.Errorf("键 %q 应被拒绝", key)
		}
	}
}

// 覆盖写必须是原子的：不能留下半份数据。
func TestFileStoreOverwrite(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(filepath.Join(dir, "store"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put("k", []byte("v1-long-content")); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("k", []byte("v2")); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Errorf("Get = %q, want v2", got)
	}
	// 临时文件不能残留。
	entries, err := os.ReadDir(filepath.Join(dir, "store"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("残留临时文件: %s", e.Name())
		}
	}
}

func TestSingleFileStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config", "config.json")
	store, err := NewSingleFileStore(path, "config")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("config"); err == nil {
		t.Error("文件不存在时应报错")
	}
	if err := store.Put("config", []byte(`{"studentId":"****"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("config")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"studentId":"****"}` {
		t.Errorf("Get = %q", got)
	}
	if err := store.Put("other", []byte("x")); err == nil {
		t.Error("只应接受约定的键")
	}
	if store.Path() != path {
		t.Errorf("Path = %q", store.Path())
	}
}

func TestDefaultDirsHonorXDG(t *testing.T) {
	getenv := func(k string) string {
		switch k {
		case "XDG_CONFIG_HOME":
			return "/xdg/config"
		case "XDG_CACHE_HOME":
			return "/xdg/cache"
		}
		return ""
	}
	cfgDir, err := DefaultConfigDir(getenv, "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if cfgDir != filepath.Join("/xdg/config", "kemaolife") {
		t.Errorf("configDir = %q", cfgDir)
	}
	cacheDir, err := DefaultCacheDir(getenv, "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if cacheDir != filepath.Join("/xdg/cache", "kemaolife") {
		t.Errorf("cacheDir = %q", cacheDir)
	}
}

func TestDefaultDirsFallbackToHome(t *testing.T) {
	none := func(string) string { return "" }
	cfgDir, err := DefaultConfigDir(none, "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if cfgDir != filepath.Join("/home/u", ".config", "kemaolife") {
		t.Errorf("configDir = %q", cfgDir)
	}
	if _, err := DefaultConfigDir(none, ""); err == nil {
		t.Error("没有 home 也没有 XDG 时应报错，不能猜路径")
	}
}

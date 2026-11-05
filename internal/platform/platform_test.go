package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

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

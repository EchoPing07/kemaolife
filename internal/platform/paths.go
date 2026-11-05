package platform

import (
	"fmt"
	"path/filepath"
	"strings"
)

// 目录布局（docs/002 §0）：
//
//	配置  ~/.config/kemaolife/config.json  （0600）
//	缓存  ~/.cache/kemaolife/              （遵循 XDG_CACHE_HOME）
//
// 只接受显式传入的 home / 环境变量值，不在本函数内判断 GOOS。

// DefaultConfigDir 返回配置目录（XDG_CONFIG_HOME 优先）。
func DefaultConfigDir(getenv func(string) string, home string) (string, error) {
	if dir := strings.TrimSpace(getenv("XDG_CONFIG_HOME")); dir != "" {
		return filepath.Join(dir, "kemaolife"), nil
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("无法确定用户主目录")
	}
	return filepath.Join(home, ".config", "kemaolife"), nil
}

// DefaultCacheDir 返回缓存目录（XDG_CACHE_HOME 优先）。
func DefaultCacheDir(getenv func(string) string, home string) (string, error) {
	if dir := strings.TrimSpace(getenv("XDG_CACHE_HOME")); dir != "" {
		return filepath.Join(dir, "kemaolife"), nil
	}
	if strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("无法确定用户主目录")
	}
	return filepath.Join(home, ".cache", "kemaolife"), nil
}

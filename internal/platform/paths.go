package platform

import (
	"fmt"
	"os"
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

// EnsureConfigFile 确保配置文件存在且权限为 0600。
//
// 只做「存在 + 权限」检查：内容由 app 层序列化后写回，platform 不解析配置结构。
func EnsureConfigFile(path string, seed []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	info, err := os.Stat(path)
	switch {
	case err == nil:
		// 历史文件可能是 0644，必须收紧；凭据明文落盘是红线（P5）。
		if info.Mode().Perm() != 0o600 {
			if err := os.Chmod(path, 0o600); err != nil {
				return fmt.Errorf("收紧配置文件权限失败: %w", err)
			}
		}
		return nil
	case os.IsNotExist(err):
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("创建配置文件失败: %w", err)
		}
		defer f.Close()
		if len(seed) > 0 {
			if _, err := f.Write(seed); err != nil {
				return fmt.Errorf("写入配置文件失败: %w", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("检查配置文件失败: %w", err)
	}
}

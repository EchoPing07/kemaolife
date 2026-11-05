//go:build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// WindowsDataDir 返回数据目录 %LOCALAPPDATA%\Kemaolife\（docs/002 §0）。
//
// 用 LOCALAPPDATA 而不是 APPDATA：配置含加密凭据，属于本机数据，
// 不应随漫游配置文件同步到别的机器（换来的是「换机后 DPAPI 也解不开」的一致性）。
func WindowsDataDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		// 极少见（服务账户 / 精简环境）：退到用户主目录，仍然限定在当前用户下。
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法确定数据目录: %w", err)
		}
		base = filepath.Join(home, "AppData", "Local")
	}
	dir := filepath.Join(base, "Kemaolife")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("创建数据目录失败: %w", err)
	}
	return dir, nil
}

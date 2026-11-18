//go:build windows

package ui

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

// 应用图标随二进制内嵌，不依赖外部文件（单二进制交付）。
//
// 为什么不用 CreateIconFromResourceEx 直接解析 ico 字节：
// 那个 API 需要的是 RT_ICON 资源（带 DIB 头）或 PNG 帧，而 .ico 文件里的
// 多帧目录格式并不被它接受 —— 实测所有帧都返回 0（图标不显示）。
// 改用 PrivateExtractIconsW：它原生理解 .ico 文件格式并会按尺寸挑帧，
// 唯一要求是给一个文件路径，所以这里把内嵌字节落到临时文件再加载。
//
//go:embed assets/icon.ico
var iconICO []byte

var (
	iconOnce, trayOnce sync.Once
	iconHandle         uintptr
	iconErr            error
	trayHandle         uintptr
	trayErr            error
	iconPath           string
)

var procPrivateExtractIcons = user32.NewProc("PrivateExtractIconsW")

const (
	// 托盘图标偏好 32px（高分屏由系统再缩放）。
	trayIconSize = 32
	// 窗口图标偏好 48px：任务栏与 Alt+Tab 需要更清晰的边缘。
	windowIconSize = 48
)

// AppIcon 返回窗口 / 托盘用的 HICON（进程内复用）。
func AppIcon() (uintptr, error) {
	iconOnce.Do(func() {
		iconHandle, iconErr = loadIcon(windowIconSize)
	})
	if iconErr != nil {
		return 0, fmt.Errorf("加载应用图标失败: %w", iconErr)
	}
	return iconHandle, nil
}

// TrayIcon 返回托盘专用的小尺寸 HICON。
func TrayIcon() (uintptr, error) {
	trayOnce.Do(func() {
		trayHandle, trayErr = loadIcon(trayIconSize)
	})
	if trayErr != nil {
		// 小图标失败就退回窗口图标，不要让托盘整块消失。
		return AppIcon()
	}
	return trayHandle, nil
}

// loadIcon 把内嵌 ico 落到临时文件，再用 PrivateExtractIconsW 按尺寸提取。
func loadIcon(size int) (uintptr, error) {
	path, err := materializeIcon()
	if err != nil {
		return 0, err
	}
	return extractIcon(path, size)
}

// materializeIcon 把内嵌 ico 写到 %TEMP%，只在首次调用时写一次。
func materializeIcon() (string, error) {
	if iconPath != "" {
		return iconPath, nil
	}
	dir, err := os.MkdirTemp("", "kemaolife-icon-")
	if err != nil {
		return "", fmt.Errorf("创建图标临时目录失败: %w", err)
	}
	path := filepath.Join(dir, "icon.ico")
	if err := os.WriteFile(path, iconICO, 0o600); err != nil {
		return "", fmt.Errorf("写入图标临时文件失败: %w", err)
	}
	iconPath = path
	return path, nil
}

// CleanupIcon 删除图标临时文件（进程退出前调用；失败可忽略）。
func CleanupIcon() {
	if iconPath == "" {
		return
	}
	_ = os.RemoveAll(filepath.Dir(iconPath))
	iconPath = ""
}

// extractIcon 调用 PrivateExtractIconsW 取一帧 HICON。
//
// 索引 0 + 宽高会让系统选最接近的那帧；返回 r==0 说明没有匹配帧。
func extractIcon(path string, size int) (uintptr, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var (
		hIcon  uintptr
		iconID uint32
	)
	r, _, e := procPrivateExtractIcons.Call(
		uintptr(unsafe.Pointer(p)),
		0, // 从第 0 帧开始按尺寸匹配
		uintptr(size), uintptr(size),
		uintptr(unsafe.Pointer(&hIcon)), uintptr(unsafe.Pointer(&iconID)),
		1, 0,
	)
	if r == 0 || hIcon == 0 {
		return 0, fmt.Errorf("提取 %dpx 图标失败: %w", size, e)
	}
	return hIcon, nil
}

// IconBytes 返回内嵌的 .ico 原始字节（构建脚本把它嵌入 exe 资源段时用）。
func IconBytes() []byte { return iconICO }

//go:build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"

	"syscall"
	"unsafe"
)

// 自启动走 HKCU\...\Run（AGENTS §5 / 001 §6.4：登录时自启一次，不保活）。
//
// 不用「任务计划程序」也不用服务：后者要求管理员权限，且属于守护语义，
// 与「被用户退出或杀掉后不重启」的约定冲突。

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	// AutoStartValueName 是自启项的注册表值名（docs/002 §0：KemaolifeAgent）。
	AutoStartValueName = "KemaolifeAgent"
)

var (
	advapi32          = syscall.NewLazyDLL("advapi32.dll")
	procRegCreateKey  = advapi32.NewProc("RegCreateKeyExW")
	procRegOpenKey    = advapi32.NewProc("RegOpenKeyExW")
	procRegSetValue   = advapi32.NewProc("RegSetValueExW")
	procRegQueryValue = advapi32.NewProc("RegQueryValueExW")
	procRegDelete     = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey   = advapi32.NewProc("RegCloseKey")
)

const (
	hkeyCurrentUser = 0x80000001
	keySetValue     = 0x0002
	keyQueryValue   = 0x0001
	regSZ           = 1
)

// AutoStartSupported 说明本平台是否支持开机自启（测试里用来跳过）。
func AutoStartSupported() bool { return true }

// SetAutoStart 打开 / 关闭自启。enable=false 时删除值，不留空项。
func SetAutoStart(enable bool) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法确定程序路径: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("无法确定程序路径: %w", err)
	}

	key, err := regCreateRunKey()
	if err != nil {
		return err
	}
	defer procRegCloseKey.Call(key)

	if !enable {
		procRegDelete.Call(key, uintptr(unsafe.Pointer(utf16Ptr(AutoStartValueName))))
		return nil
	}

	// 路径可能含空格（Program Files），必须整体加引号，否则 Windows 会截断命令行。
	value := `"` + exe + `"`
	val := utf16(value)
	r, _, e := procRegSetValue.Call(
		key,
		uintptr(unsafe.Pointer(utf16Ptr(AutoStartValueName))),
		0, regSZ,
		uintptr(unsafe.Pointer(&val[0])),
		uintptr(len(val)*2),
	)
	if r != 0 {
		return fmt.Errorf("写入自启项失败: %w", e)
	}
	return nil
}

// AutoStartEnabled 读回自启状态；值存在即视为开启（不比对路径，
// 程序被移动后仍会显示「已开启」，用户可在设置里重新开关一次修正）。
func AutoStartEnabled() bool {
	key, err := regOpenRunKey()
	if err != nil {
		return false
	}
	defer procRegCloseKey.Call(key)

	r, _, _ := procRegQueryValue.Call(
		key,
		uintptr(unsafe.Pointer(utf16Ptr(AutoStartValueName))),
		0, 0, 0, 0,
	)
	return r == 0
}

// regCreateRunKey 打开（必要时创建）HKCU 下的 Run 键。
func regCreateRunKey() (uintptr, error) {
	var key uintptr
	var disposition uint32
	r, _, e := procRegCreateKey.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16Ptr(runKeyPath))),
		0, 0,
		0, keySetValue|keyQueryValue,
		0,
		uintptr(unsafe.Pointer(&key)),
		uintptr(unsafe.Pointer(&disposition)),
	)
	if r != 0 {
		return 0, fmt.Errorf("打开注册表失败: %w", e)
	}
	return key, nil
}

// regOpenRunKey 只打开已存在的 Run 键（读状态用，不创建）。
func regOpenRunKey() (uintptr, error) {
	var key uintptr
	r, _, e := procRegOpenKey.Call(
		hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16Ptr(runKeyPath))),
		uintptr(unsafe.Pointer(&key)),
	)
	if r != 0 {
		return 0, fmt.Errorf("打开注册表失败: %w", e)
	}
	return key, nil
}

// utf16 转 UTF-16 并以 NUL 结尾（Win32 宽字符 API 要求）。
func utf16(s string) []uint16 {
	p, err := syscall.UTF16FromString(s)
	if err != nil {
		return []uint16{0}
	}
	return p
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		z := uint16(0)
		return &z
	}
	return p
}

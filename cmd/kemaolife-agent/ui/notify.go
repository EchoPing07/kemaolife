//go:build windows

package ui

import (
	"fmt"
	"syscall"
	"unsafe"
)

// 托盘图标与气泡通知。
//
// 为什么把通知放在 UI 包而不是 internal/platform：
// Shell_NotifyIcon 的「加图标 / 改提示 / 弹气泡 / 删图标」共用同一份
// NOTIFYICONDATAW 结构，拆到两个包里必然重复定义 168 字节的 Win32 结构，
// 一旦字段错位就是「图标不显示 / 气泡不弹出」这类难查的问题。
// 通知属于呈现，本就是 cmd 层的职责（AGENTS §3：cmd/** 允许原生 UI）。

var procShellNotifyIcon = syscall.NewLazyDLL("shell32.dll").NewProc("Shell_NotifyIconW")

const (
	nimAdd    = 0
	nimModify = 1
	nimDelete = 2

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nifInfo    = 0x00000010

	niifNone = 0x0
	niifInfo = 0x1
)

// 托盘回调消息号：与窗口过程里的分发常量必须一致。
const TrayCallbackMessage = WM_APP + 100

// notifyIconData 对应 Win32 NOTIFYICONDATAW（Vista+ 布局）。
//
// 字段顺序和宽度必须与 SDK 完全一致：Win32 按偏移读结构，错一个字段
// 就会读成垃圾值。
type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

// Tray 是常驻托盘图标；同时满足 core.Notifier。
type Tray struct {
	hwnd uintptr
	icon uintptr
	uid  uint32
	// registered 记录图标是否已加入托盘，避免重复 NimAdd 造成多个图标。
	registered bool
}

// NewTray 绑定一个隐藏窗口与图标（窗口只用来接收托盘消息）。
func NewTray(hwnd, icon uintptr) *Tray {
	return &Tray{hwnd: hwnd, icon: icon, uid: 1}
}

// Icon 返回托盘图标句柄（窗口类注册要用同一个）。
func (t *Tray) Icon() uintptr { return t.icon }

// Add 注册托盘图标。tip 是悬浮提示。
func (t *Tray) Add(tip string) error {
	if t.registered {
		return nil
	}
	d := notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             t.hwnd,
		uID:              t.uid,
		uFlags:           nifMessage | nifIcon | nifTip,
		uCallbackMessage: TrayCallbackMessage,
		hIcon:            t.icon,
	}
	copyUTF16(d.szTip[:], tip)
	if r, _, err := procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&d))); r == 0 {
		return fmt.Errorf("注册托盘图标失败: %w", err)
	}
	t.registered = true
	return nil
}

// SetTip 更新悬浮提示（状态变化时调用）。
func (t *Tray) SetTip(tip string) error {
	if !t.registered {
		return nil
	}
	d := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   t.hwnd,
		uID:    t.uid,
		uFlags: nifTip,
	}
	copyUTF16(d.szTip[:], tip)
	if r, _, err := procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&d))); r == 0 {
		return fmt.Errorf("更新托盘提示失败: %w", err)
	}
	return nil
}

// Notify 弹一条气泡通知（满足 core.Notifier）。
//
// 用气泡而不是 WinRT Toast：Toast 需要注册 AUMID 与开始菜单快捷方式，
// 否则在部分系统上静默不显示；气泡零注册，托盘程序本来就有图标。
// TODO(D4): 若最终定为系统 Toast，只换本方法的实现，调用点不变。
func (t *Tray) Notify(title, body string) error {
	if !t.registered {
		return fmt.Errorf("托盘图标未注册，无法通知")
	}
	d := notifyIconData{
		cbSize:      uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:        t.hwnd,
		uID:         t.uid,
		uFlags:      nifInfo,
		dwInfoFlags: niifInfo,
	}
	copyUTF16(d.szInfoTitle[:], title)
	copyUTF16(d.szInfo[:], body)
	if r, _, err := procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&d))); r == 0 {
		return fmt.Errorf("弹出通知失败: %w", err)
	}
	return nil
}

// Remove 注销图标。退出前必须调用，否则托盘里会留一个点不动的死图标。
func (t *Tray) Remove() {
	if !t.registered {
		return
	}
	d := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   t.hwnd,
		uID:    t.uid,
	}
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&d)))
	t.registered = false
}

// copyUTF16 写入定长 UTF-16 数组并保证 NUL 结尾，超长按 rune 截断。
func copyUTF16(dst []uint16, s string) {
	if len(dst) == 0 {
		return
	}
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			if n >= len(dst)-2 {
				break
			}
			r -= 0x10000
			dst[n] = uint16(0xD800 + (r >> 10))
			dst[n+1] = uint16(0xDC00 + (r & 0x3FF))
			n += 2
			continue
		}
		if n >= len(dst)-1 {
			break
		}
		dst[n] = uint16(r)
		n++
	}
	dst[n] = 0
}

// TipSetter 是「在 UI 线程上更新托盘提示」的唯一入口。
//
// 后台 goroutine 不能直接操作 USER 对象（跨线程改托盘提示会出现
// 「提示不刷新」或卡顿），所以状态文本先经 PostStatus 回到 UI 线程，
// 由这里落地。
type TipSetter struct {
	tray *Tray
}

// NewTipSetter 绑定托盘对象。
func NewTipSetter(t *Tray) *TipSetter { return &TipSetter{tray: t} }

// Set 更新托盘提示。
//
// 只保留很短的一句话：Win11 的托盘悬停提示是大气泡，内容一长就会盖住
// 刚弹出的右键菜单（实测会挡住「登录校园网 / 下线 / 重登」）。详细信息
// 已经在菜单前两行里，提示只需要「这是哪个程序」。
func (t *TipSetter) Set(text string) {
	if t == nil || t.tray == nil {
		return
	}
	_ = t.tray.SetTip(clipRunes(text, 60))
}

// clipRunes 按 rune 截断，避免把多字节字符切成乱码。
func clipRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

//go:build windows

package ui

import (
	"fmt"
	"runtime"
	"unsafe"
)

// 托盘右键菜单的菜单项 ID。数值与 cmd 层的命令分派一一对应。
const (
	IDStatusCampus = 1001 // 只读：校园网状态
	IDStatusPower  = 1002 // 只读：剩余电费（含数据截至）
	IDSep1         = 1003

	IDLogin   = 1101 // 手动模式：登录
	IDLogout  = 1102 // 手动模式：下线
	IDRelogin = 1103 // 两种模式都有：重登

	IDSep2     = 1200
	IDRefresh  = 1201 // 立即查询一次电费
	IDSettings = 1301
	IDSep3     = 1302
	IDExit     = 1401
)

// MenuItem 是一个菜单项：标题 + 命令 ID；Grayed 表示只读展示项。
type MenuItem struct {
	Title  string
	ID     int
	Grayed bool
	Sep    bool
	Check  *bool // 非 nil 时按布尔值显示勾选态（用于模式切换）
}

// Menu 是一次构建出来的弹出菜单。
type Menu struct {
	handle uintptr
	items  []MenuItem
}

// BuildMenu 创建弹出菜单。
func BuildMenu(items []MenuItem) (*Menu, error) {
	h, _, err := procCreatePopupMenu.Call()
	if h == 0 {
		return nil, fmt.Errorf("创建菜单失败: %w", err)
	}
	m := &Menu{handle: h, items: items}
	for _, it := range items {
		var flags uintptr = MF_STRING
		if it.Sep {
			// MF_SEPARATOR 忽略文本；用空串传更安全。
			procAppendMenu.Call(h, MF_SEPARATOR, 0, uintptr(unsafe.Pointer(strPtr(""))))
			continue
		}
		if it.Grayed {
			flags |= MF_GRAYED
		}
		if it.Check != nil {
			if *it.Check {
				flags |= MF_CHECKED
			} else {
				flags |= MF_UNCHECKED
			}
		}
		t := strPtr(it.Title)
		procAppendMenu.Call(h, flags, uintptr(it.ID), uintptr(unsafe.Pointer(t)))
		runtime.KeepAlive(t)
	}
	return m, nil
}

// Destroy 释放菜单句柄（不释放会持续泄漏 GDI/USER 对象）。
func (m *Menu) Destroy() {
	if m != nil && m.handle != 0 {
		procDestroyMenu.Call(m.handle)
		m.handle = 0
	}
}

// PopupAtCursor 在鼠标位置弹出菜单，返回用户选择的命令 ID（0 表示取消）。
//
// 必须先 SetForegroundWindow：否则点击菜单外部时菜单不会关闭，
// 表现为「菜单卡在屏幕上」（Win32 托盘的经典坑）。
func (m *Menu) PopupAtCursor(owner uintptr) int {
	if m == nil || m.handle == 0 {
		return 0
	}
	procSetForeground.Call(owner)
	p := cursorPos()
	r, _, _ := procTrackPopupMenu.Call(
		m.handle,
		TPM_RETURNCMD|TPM_RIGHTBUTTON,
		uintptr(p.X), uintptr(p.Y),
		0, owner, 0,
	)
	return int(r)
}

// LoadIconByID 加载系统预定义图标（窗口类注册用，失败时兜底）。
func LoadIconByID(id uintptr) uintptr {
	h, _, _ := procLoadIconChecked.Call(0, id)
	return h
}

//go:build windows

package ui

import (
	"sync"
	"syscall"
	"unsafe"
)

// 输入框占位提示：EM_SETCUEBANNER 需要 comctl32 v6 manifest，本程序不带
// manifest（纯 syscall 直调），系统会静默忽略该消息。改为在原生绘制之后
// 叠一层灰字：空且未聚焦时显示占位文案，聚焦或有内容时让位给真实输入。

const wmChar = 0x0102
const wmGetFont = 0x0031

var (
	cuemu          sync.Mutex
	cueTexts       = map[uintptr]string{}
	cueProcs       = map[uintptr]uintptr{}
	cueCallback    uintptr
	procGetDC      = user32.NewProc("GetDC")
	procReleaseDC  = user32.NewProc("ReleaseDC")
	cuePlaceholder = rgb(148, 158, 168)
)

// registerCueEdit 给输入框登记占位文案并换自绘子类（含滚轮转发）。
func registerCueEdit(hwnd uintptr, cue string) {
	if hwnd == 0 {
		return
	}
	cuemu.Lock()
	defer cuemu.Unlock()
	cueTexts[hwnd] = cue
	if cueCallback == 0 {
		cueCallback = syscall.NewCallback(cueEditProc)
	}
	if _, ok := cueProcs[hwnd]; ok {
		return
	}
	orig, _, _ := procSetWindowLongPtr.Call(hwnd, gwlpWndProc, cueCallback)
	cueProcs[hwnd] = orig
}

func cueEditProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	callOrig := func() uintptr {
		cuemu.Lock()
		orig := cueProcs[hwnd]
		cuemu.Unlock()
		if orig != 0 {
			r, _, _ := procCallWindowProc.Call(orig, hwnd, uintptr(msg), wParam, lParam)
			return r
		}
		r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	}
	switch msg {
	case wmPaint:
		r := callOrig()
		cuemu.Lock()
		cue := cueTexts[hwnd]
		cuemu.Unlock()
		focused, _, _ := procGetFocus.Call(hwnd)
		if cue != "" && focused != hwnd && Text(hwnd) == "" {
			dc, _, _ := procGetDC.Call(hwnd)
			if dc != 0 {
				var rect RECT
				procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
				font, _, _ := procSendMessage.Call(hwnd, wmGetFont, 0, 0)
				if font != 0 {
					procSelectObject.Call(dc, font)
				}
				SetBkMode(dc, transparentMode)
				SetTextColor(dc, cuePlaceholder)
				p := strPtr(cue)
				inner := mainRect(rect.Left+Scale(hwnd, 6), rect.Top, rect.Right-Scale(hwnd, 6), rect.Bottom)
				procDrawText.Call(dc, uintptr(unsafe.Pointer(p)), ^uintptr(0),
					uintptr(unsafe.Pointer(&inner)), uintptr(dtLeft|dtVcenter|dtSingleLine|dtEndEllipsis|dtNoPrefix))
				procReleaseDC.Call(hwnd, dc)
			}
		}
		return r
	case wmSetFocus, wmKillFocus, wmSetText, wmChar:
		procInvalidateRect.Call(hwnd, 0, 1)
		return callOrig()
	case wmMouseWheel:
		if parent, _, _ := procGetParent.Call(hwnd); parent != 0 {
			r, _, _ := procSendMessage.Call(parent, uintptr(msg), wParam, lParam)
			return r
		}
		return 0
	}
	return callOrig()
}

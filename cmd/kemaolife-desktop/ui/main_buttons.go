//go:build windows

package ui

import (
	"sync"
	"syscall"
	"unsafe"
)

// 自绘按钮：保留原生 BUTTON 的点击 / 键盘 / 焦点语义（BN_CLICKED 照常
// 发给父窗口），只接管 WM_PAINT 与鼠标悬停态，画出与页面一体的配色。
// 导航、主按钮、次按钮、危险按钮四种形态共用一套子类过程。

const (
	btnNav = iota
	btnPrimary
	btnGhost
	btnDanger
)

const (
	wmMouseMove      = 0x0200
	wmMouseLeave     = 0x02A3
	wmLButtonDown    = 0x0201
	wmSetFocus       = 0x0007
	wmKillFocus      = 0x0008
	wmEnable         = 0x000A
	wmSetText        = 0x000C
	wmCaptureChanged = 0x0215

	bmGetState = 0x00F2
	bstPushed  = 0x0004
	tmeLeave   = 0x0002
)

type mainTrackMouseEvent struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

var (
	procGetFocus        = user32.NewProc("GetFocus")
	procIsWindowEnabled = user32.NewProc("IsWindowEnabled")
	procTrackMouseEvent = user32.NewProc("TrackMouseEvent")
	procRoundRect       = gdi32.NewProc("RoundRect")
)

type mainButtonStyle struct {
	kind    int
	active  bool
	hot     bool
	pressed bool
}

var (
	mainButtonMu       sync.Mutex
	mainButtonStyles   = map[uintptr]*mainButtonStyle{}
	mainButtonCallback uintptr
	mainButtonProcs    = map[uintptr]uintptr{}
)

// registerMainButton 给按钮换自绘子类并登记形态。
func registerMainButton(hwnd uintptr, kind int) {
	if hwnd == 0 {
		return
	}
	mainButtonMu.Lock()
	defer mainButtonMu.Unlock()
	mainButtonStyles[hwnd] = &mainButtonStyle{kind: kind}
	if mainButtonCallback == 0 {
		mainButtonCallback = syscall.NewCallback(mainButtonProc)
	}
	if _, ok := mainButtonProcs[hwnd]; ok {
		return
	}
	orig, _, _ := procSetWindowLongPtr.Call(hwnd, gwlpWndProc, mainButtonCallback)
	mainButtonProcs[hwnd] = orig
}

func setMainButtonActive(hwnd uintptr, active bool) {
	if hwnd == 0 {
		return
	}
	mainButtonMu.Lock()
	st := mainButtonStyles[hwnd]
	if st != nil {
		st.active = active
	}
	mainButtonMu.Unlock()
	procInvalidateRect.Call(hwnd, 0, 1)
}

func mainButtonProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	callOrig := func() uintptr {
		mainButtonMu.Lock()
		orig := mainButtonProcs[hwnd]
		mainButtonMu.Unlock()
		if orig != 0 {
			r, _, _ := procCallWindowProc.Call(orig, hwnd, uintptr(msg), wParam, lParam)
			return r
		}
		r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	}
	setHot := func(hot bool) {
		mainButtonMu.Lock()
		st := mainButtonStyles[hwnd]
		changed := st != nil && st.hot != hot
		if st != nil {
			st.hot = hot
		}
		mainButtonMu.Unlock()
		if changed {
			procInvalidateRect.Call(hwnd, 0, 1)
		}
	}
	setPressed := func(pressed bool) {
		mainButtonMu.Lock()
		st := mainButtonStyles[hwnd]
		if st != nil {
			st.pressed = pressed
		}
		mainButtonMu.Unlock()
		procInvalidateRect.Call(hwnd, 0, 1)
	}

	switch msg {
	case wmPaint:
		paintMainButton(hwnd)
		return 0
	case WM_ERASEBKGND:
		return 1
	case wmMouseMove:
		var r RECT
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		x, y := int32(int16(lParam&0xFFFF)), int32(int16((lParam>>16)&0xFFFF))
		inside := x >= 0 && y >= 0 && x < r.Right && y < r.Bottom
		mainButtonMu.Lock()
		st := mainButtonStyles[hwnd]
		needTrack := st != nil && !st.hot
		mainButtonMu.Unlock()
		if needTrack {
			ev := mainTrackMouseEvent{
				CbSize:  uint32(unsafe.Sizeof(mainTrackMouseEvent{})),
				DwFlags: tmeLeave, HwndTrack: hwnd,
			}
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&ev)))
		}
		setHot(inside)
		return 0
	case wmMouseLeave:
		setHot(false)
		return 0
	case wmLButtonDown:
		setPressed(true)
		return callOrig()
	case WM_LBUTTONUP:
		setPressed(false)
		return callOrig()
	case wmCaptureChanged:
		setPressed(false)
		return callOrig()
	case wmSetFocus, wmKillFocus, wmEnable:
		procInvalidateRect.Call(hwnd, 0, 1)
		return callOrig()
	case wmSetText:
		r := callOrig()
		procInvalidateRect.Call(hwnd, 0, 1)
		return r
	case wmMouseWheel:
		// 与输入框同一套滚轮转发：焦点在按钮上时页面仍要能滚。
		if parent, _, _ := procGetParent.Call(hwnd); parent != 0 {
			r, _, _ := procSendMessage.Call(parent, uintptr(msg), wParam, lParam)
			return r
		}
		return 0
	}
	return callOrig()
}

// paintMainButton 按形态与交互态画圆角按钮。
func paintMainButton(hwnd uintptr) {
	var ps mainPaintStruct
	dc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var r RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if r.Right <= r.Left || r.Bottom <= r.Top {
		return
	}

	mainButtonMu.Lock()
	st := mainButtonStyle{kind: btnGhost}
	if s := mainButtonStyles[hwnd]; s != nil {
		st = *s
	}
	mainButtonMu.Unlock()

	enabled, _, _ := procIsWindowEnabled.Call(hwnd)
	state, _, _ := procSendMessage.Call(hwnd, bmGetState, 0, 0)
	pressed := enabled != 0 && (st.pressed || state&bstPushed != 0)
	hot := enabled != 0 && st.hot && !pressed
	focused, _, _ := procGetFocus.Call(hwnd)

	var surround, fill, border, textColor uintptr
	weight := int32(400)
	switch st.kind {
	case btnNav:
		surround, fill, textColor = mainSidebar, mainSidebar, mainSidebarText
		border = fill
		if st.active {
			fill, textColor, weight = mainAccent, rgb(255, 255, 255), 700
			if pressed {
				fill = mainAccentLo
			} else if hot {
				fill = mainAccentHi
			}
		} else if pressed {
			fill = mainSidebarHi
		} else if hot {
			fill = mainSidebarHi
			textColor = rgb(235, 241, 247)
		}
		border = fill
	case btnPrimary:
		surround, fill, textColor, weight = mainBackground, mainAccent, rgb(255, 255, 255), 600
		if enabled == 0 {
			fill, textColor = rgb(198, 207, 216), rgb(244, 246, 248)
		} else if pressed {
			fill = mainAccentLo
		} else if hot {
			fill = mainAccentHi
		}
		border = fill
	default: // btnGhost / btnDanger
		surround, fill, border, textColor = mainBackground, mainCard, mainLine, mainTextColor
		if st.kind == btnDanger {
			textColor = mainDanger
		}
		if enabled == 0 {
			fill, border, textColor = mainBackground, rgb(232, 236, 240), rgb(163, 172, 181)
		} else if pressed {
			fill = mainAccentSoft
			if st.kind == btnDanger {
				fill = rgb(250, 235, 235)
			}
		} else if hot {
			fill, border = rgb(237, 243, 247), rgb(146, 178, 196)
			if st.kind == btnDanger {
				fill, border = rgb(252, 243, 243), rgb(222, 154, 154)
			}
		}
	}

	c := mainCanvas{hwnd: hwnd, dc: dc}
	c.fill(r, surround)
	rad := Scale(hwnd, 8)
	brush, pen := mainBrush(fill), mainPen(border)
	if brush != 0 && pen != 0 {
		oldBrush, _, _ := procSelectObject.Call(dc, brush)
		oldPen, _, _ := procSelectObject.Call(dc, pen)
		procRoundRect.Call(dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom),
			uintptr(rad), uintptr(rad))
		procSelectObject.Call(dc, oldBrush)
		procSelectObject.Call(dc, oldPen)
	}
	if focused == hwnd && enabled != 0 {
		focusColor := mainAccent
		if st.kind == btnNav || st.kind == btnPrimary {
			focusColor = rgb(255, 255, 255)
		}
		if pen := mainPen(focusColor); pen != 0 {
			oldPen, _, _ := procSelectObject.Call(dc, pen)
			oldBrush, _, _ := procSelectObject.Call(dc, mainBrush(fill))
			inner := RECT{r.Left + Scale(hwnd, 3), r.Top + Scale(hwnd, 3),
				r.Right - Scale(hwnd, 3), r.Bottom - Scale(hwnd, 3)}
			procRoundRect.Call(dc, uintptr(inner.Left), uintptr(inner.Top),
				uintptr(inner.Right), uintptr(inner.Bottom), uintptr(rad), uintptr(rad))
			procSelectObject.Call(dc, oldBrush)
			procSelectObject.Call(dc, oldPen)
		}
	}
	text := Text(hwnd)
	c.useFont(14, weight).text(text,
		mainRect(r.Left+Scale(hwnd, 8), r.Top, r.Right-Scale(hwnd, 8), r.Bottom),
		dtCenter|dtVcenter|dtSingleLine|dtEndEllipsis, textColor)
}

//go:build windows

package ui

import "testing"

func TestShowMainWaitsForTrayHandler(t *testing.T) {
	trayMu.Lock()
	trayHandler = TrayHandler{}
	trayHwnd = 0
	trayShowPending = false
	trayMu.Unlock()
	t.Cleanup(func() {
		trayMu.Lock()
		trayHandler = TrayHandler{}
		trayHwnd = 0
		trayShowPending = false
		trayMu.Unlock()
	})

	if _, ok := handleTrayMessage(0, WM_SHOW_MAIN, 0, 0); !ok {
		t.Fatal("唤起消息应由托盘处理")
	}
	trayMu.Lock()
	pending := trayShowPending
	trayMu.Unlock()
	if !pending {
		t.Fatal("处理器未绑定时的唤起请求应被保留")
	}

	calls := 0
	BindTray(0, TrayHandler{OnTrayClick: func() { calls++ }})
	if calls != 1 {
		t.Fatalf("绑定处理器后应补发一次唤起，got %d", calls)
	}
	trayMu.Lock()
	pending = trayShowPending
	trayMu.Unlock()
	if pending {
		t.Fatal("补发后不得继续保留唤起请求")
	}
}

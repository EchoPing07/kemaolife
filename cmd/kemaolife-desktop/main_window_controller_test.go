//go:build windows

package main

import (
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
	"github.com/EchoPing07/kemaolife/internal/desktop"
)

// 设置页打开时必须回显当前生效配置（否则每次打开都是默认值），
// 且两类密码恒为空，不回显已保存凭据。
func TestMainStateWiresSettingsForm(t *testing.T) {
	settings := desktop.DefaultSettings()
	settings.StudentID = "<学号>"
	settings.EduPassword = "<教务密码>"
	settings.CampusPassword = "<校园网密码>"
	settings.PollSeconds = 900

	a := &desktopApp{settings: settings}
	a.engine = desktop.NewEngine(app.Deps{
		Clock: porttest.NewFakeClock(time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)),
		Store: porttest.NewMemStore(),
	}, app.CampusDeps{}, nil, a.currentSettings, nil)

	state := a.mainState()
	form := state.SettingsForm
	if form.StudentID != settings.StudentID || form.PollSeconds != settings.PollSeconds {
		t.Errorf("设置页未回显生效配置: %+v", form)
	}
	if form.EduPassword != "" || form.CampusPassword != "" {
		t.Errorf("设置表单不得回显明文密码: edu=%q campus=%q", form.EduPassword, form.CampusPassword)
	}
}

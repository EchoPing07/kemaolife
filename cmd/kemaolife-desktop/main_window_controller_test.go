//go:build windows

package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/cmd/kemaolife-desktop/ui"
	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
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
	settings.CampusWiredOnly = true

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
	if !form.WiredOnly {
		t.Error("设置页应回显「仅有线自动认证」开关")
	}
	if form.EduPassword != "" || form.CampusPassword != "" {
		t.Errorf("设置表单不得回显明文密码: edu=%q campus=%q", form.EduPassword, form.CampusPassword)
	}
}

// blockingDoer 在 release 关闭前阻塞，模拟「查询在途时用户换了账号」。
type blockingDoer struct{ release chan struct{} }

func (d blockingDoer) Do(*http.Request) (*http.Response, error) {
	<-d.release
	return nil, errors.New("上游失败")
}

// blockingCampus 同上，用于校园网动作。
type blockingCampus struct{ release chan struct{} }

func (c blockingCampus) Status(context.Context) (model.CampusStatus, error) {
	<-c.release
	return model.CampusStatus{}, errors.New("判态失败")
}
func (c blockingCampus) Login(context.Context) error   { return nil }
func (c blockingCampus) Logout(context.Context) error  { return nil }
func (c blockingCampus) Relogin(context.Context) error { return nil }
func (c blockingCampus) EnsureCampusNetwork(context.Context) error {
	return nil
}

// 手动任务在途时换账号（代次推进 + 视图重置）：旧任务的收尾不得把
// 旧账号的错误写进新视图，也不得清掉新视图的加载态（同 refreshAll 的守卫）。
func TestManualJobsGuardGeneration(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(a *desktopApp)
	}{
		{"立即查询电费", func(a *desktopApp) { a.startElectricityPoll() }},
		{"月份查询", func(a *desktopApp) { a.startElectricityQuery("2026-10") }},
		{"校园网判态", func(a *desktopApp) { a.startCampusAction("refresh") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			settings := desktop.DefaultSettings()
			settings.StudentID = "<学号>"
			a := &desktopApp{settings: settings, activeJobs: map[string]bool{}}
			clock := porttest.NewFakeClock(time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local))
			campus := blockingCampus{release}
			a.engine = desktop.NewEngine(app.Deps{
				Clock:               clock,
				Store:               porttest.NewMemStore(),
				ConfigStore:         porttest.NewMemStore(),
				HTTP:                blockingDoer{release},
				DefaultCustomerCode: "3126",
				ElectricityEndpoint: "http://127.0.0.1:1",
			}, app.CampusDeps{Clock: clock, Net: campus, Ready: campus}, nil, a.currentSettings, nil)

			tc.start(a)
			// 模拟换账号：推进代次并重置视图，新视图进入加载态。
			a.snapshotGenerations.next()
			a.viewMu.Lock()
			a.view = ui.MainWindowState{ElectricityLoading: true, CampusLoading: true}
			a.viewMu.Unlock()
			close(release)

			deadline := time.Now().Add(2 * time.Second)
			for {
				a.viewMu.RLock()
				done := len(a.activeJobs) == 0
				a.viewMu.RUnlock()
				if done {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("手动任务未在期限内收尾")
				}
				time.Sleep(5 * time.Millisecond)
			}
			a.viewMu.RLock()
			defer a.viewMu.RUnlock()
			if a.electricityErr != "" || a.campusErr != "" {
				t.Errorf("旧任务错误污染了新视图: electricity=%q campus=%q", a.electricityErr, a.campusErr)
			}
			if !a.view.ElectricityLoading || !a.view.CampusLoading {
				t.Error("旧任务收尾不得清掉新视图的加载态")
			}
		})
	}
}

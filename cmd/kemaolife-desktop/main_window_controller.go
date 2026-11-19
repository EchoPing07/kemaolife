//go:build windows

package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/EchoPing07/kemaolife/cmd/kemaolife-desktop/ui"
	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/desktop"
)

// openMain 只在 UI 线程创建或唤起窗口；所有业务回调只启动后台任务。
func (a *desktopApp) openMain() {
	state := a.mainState()
	icon, _ := ui.AppIcon()
	if err := ui.OpenMainWindow(a.wnd, icon, state, ui.MainWindowHandler{
		OnRefreshAll:       a.refreshAll,
		OnTimetableWeek:    a.selectTimetableWeek,
		OnTimetableRefresh: func() { a.startTimetableSync() },
		OnElectricityMonth: a.startElectricityQuery,
		OnElectricityPoll:  a.startElectricityPoll,
		OnCampusAction:     a.startCampusAction,
		OnSaveSettings:     a.saveSettingsFromPage,
	}); err != nil {
		ui.MessageBox(desktopAppName, "打开主窗口失败："+err.Error(), ui.MB_OK|ui.MB_ICONERROR)
		return
	}
	ui.OpenMainPage(ui.MainPageToday)
}

const desktopAppName = "科贸生活"

// mainState 聚合三个功能域的只读状态。各域错误互不覆盖。
func (a *desktopApp) mainState() ui.MainWindowState {
	a.viewMu.Lock()
	state := a.view
	state.Version = version + " (" + commit + ")"
	state.Days = append([]model.DaySchedule(nil), a.view.Days...)
	electricityErr, campusErr := a.electricityErr, a.campusErr
	a.viewMu.Unlock()

	status := a.engine.Status()
	state.Report = status.Report
	state.HasReport = status.HasReport
	state.Campus = status.CampusStatus
	state.CampusAutoSkipped = status.CampusAutoSkipped
	if status.ElectricityError != "" {
		state.ElectricityError = status.ElectricityError
	} else if electricityErr != "" {
		state.ElectricityError = electricityErr
	}
	if status.CampusError != "" {
		state.CampusError = status.CampusError
	} else if campusErr != "" {
		state.CampusError = campusErr
	}
	settings := a.currentSettings()
	state.AutoMode = settings.AutoMode
	state.SettingsForm = desktop.ToSettingsForm(settings)
	// 今天的课表还未回来时，用本周缓存的对应天兜底，避免首帧显示「暂无」。
	if !state.HasToday && len(state.Days) > 0 {
		today := a.engine.Deps().Clock.Now()
		for _, day := range state.Days {
			if day.Date.String() == today.Format("2006-01-02") {
				state.Today = day
				state.HasToday = true
				break
			}
		}
	}
	if state.TodayDate == "" {
		state.TodayDate = a.engine.Deps().Clock.Now().Format("2006-01-02")
	}
	return state
}

// refreshMainWindow 是后台任务回到 UI 线程的唯一更新入口。
func (a *desktopApp) refreshMainWindow() {
	a.refreshTrayTip()
	ui.UpdateMainWindow(a.mainState())
}

func (a *desktopApp) postMainUpdate() { ui.PostTrigger(a.wnd, ui.WM_QUERY_DONE) }

// bootstrapTimetable 先读 DPAPI 快照，再尝试一次独立同步；失败不丢缓存。
func (a *desktopApp) bootstrapTimetable() {
	ctx, cancel, generation, ok := a.beginTimetable()
	if !ok {
		return
	}
	defer a.finishTimetable(cancel, generation)

	cfg := a.currentSettings().TimetableConfig()
	if err := a.loadCachedTimetable(ctx, cfg, generation); err != nil {
		a.viewMu.Lock()
		if generation == a.snapshotGenerations.current() {
			a.view.TimetableError = model.MessageOf(err)
		}
		a.viewMu.Unlock()
	}
	if strings.TrimSpace(cfg.StudentID) == "" || cfg.EduPassword == "" {
		return
	}
	a.syncTimetable(ctx, cfg, generation)
}

func (a *desktopApp) startTimetableSync() {
	ctx, cancel, generation, ok := a.beginTimetable()
	if !ok {
		return
	}
	defer a.finishTimetable(cancel, generation)
	a.syncTimetable(ctx, a.currentSettings().TimetableConfig(), generation)
}

func (a *desktopApp) beginTimetable() (context.Context, context.CancelFunc, uint64, bool) {
	a.viewMu.Lock()
	if a.timetableInFlight {
		a.viewMu.Unlock()
		return nil, nil, 0, false
	}
	ctx, cancel := a.commandContext()
	a.timetableInFlight = true
	a.timetableCancel = cancel
	generation := a.snapshotGenerations.next()
	a.view.TimetableLoading = true
	a.view.TimetableError = ""
	a.viewMu.Unlock()
	a.postMainUpdate()
	return ctx, cancel, generation, true
}

func (a *desktopApp) finishTimetable(cancel context.CancelFunc, generation uint64) {
	cancel()
	a.viewMu.Lock()
	if generation == a.snapshotGenerations.current() {
		a.timetableInFlight = false
		a.timetableCancel = nil
		a.view.TimetableLoading = false
	}
	a.viewMu.Unlock()
	a.postMainUpdate()
}

func (a *desktopApp) timetableDeps(generation uint64) app.Deps {
	d := a.engine.Deps()
	d.Store = generationSnapshotStore{inner: a.secure, gens: &a.snapshotGenerations, gen: generation}
	return d
}

func (a *desktopApp) syncTimetable(ctx context.Context, cfg model.Config, generation uint64) {
	_, err := app.SyncTimetable(ctx, a.timetableDeps(generation), cfg)
	if err != nil {
		a.viewMu.Lock()
		if generation == a.snapshotGenerations.current() {
			a.view.TimetableOffline = a.view.HasToday || len(a.view.Days) != 0
			a.view.TimetableError = model.MessageOf(err)
		}
		a.viewMu.Unlock()
		return
	}
	if err := a.loadCachedTimetable(ctx, cfg, generation); err != nil {
		a.viewMu.Lock()
		if generation == a.snapshotGenerations.current() {
			a.view.TimetableOffline = a.view.HasToday || len(a.view.Days) != 0
			a.view.TimetableError = model.MessageOf(err)
		}
		a.viewMu.Unlock()
		return
	}
	a.viewMu.Lock()
	if generation == a.snapshotGenerations.current() {
		a.view.TimetableOffline = false
		a.view.TimetableError = ""
	}
	a.viewMu.Unlock()
}

func (a *desktopApp) loadCachedTimetable(ctx context.Context, cfg model.Config, generation uint64) error {
	d := a.timetableDeps(generation)
	today, err := app.GetSchedule(ctx, d, cfg, model.NewDate(d.Clock.Now()), app.CacheOnly)
	if err != nil {
		return err
	}
	a.viewMu.RLock()
	week := a.view.Week
	a.viewMu.RUnlock()
	current, err := app.GetWeekSchedule(ctx, d, cfg, week, app.CacheOnly)
	if err != nil {
		return err
	}
	a.viewMu.Lock()
	if generation != a.snapshotGenerations.current() {
		a.viewMu.Unlock()
		return nil
	}
	a.view.HasToday = true
	a.view.Today = today.Day
	a.view.TodayDate = today.Day.Date.String()
	a.view.Week = current.Week
	a.view.LastWeek = current.LastWeek
	a.view.Days = append([]model.DaySchedule(nil), current.Days...)
	a.view.TimetableSyncedAt = current.SyncedAt.Local().Format("2006-01-02 15:04")
	a.viewMu.Unlock()
	a.postMainUpdate()
	return nil
}

// selectTimetableWeek 只读缓存，即使反复切周也不触发 SyncTimetable。
func (a *desktopApp) selectTimetableWeek(week int) {
	a.viewMu.RLock()
	lastWeek := a.view.LastWeek
	a.viewMu.RUnlock()
	if lastWeek > 0 && week > lastWeek {
		return
	}
	generation := a.snapshotGenerations.current()
	if !a.beginJob("week") {
		return
	}
	a.viewMu.Lock()
	a.view.TimetableLoading = true
	a.viewMu.Unlock()
	a.postMainUpdate()

	go func() {
		defer a.finishJob("week")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		current, err := app.GetWeekSchedule(ctx, a.timetableDeps(generation),
			a.currentSettings().TimetableConfig(), week, app.CacheOnly)
		a.viewMu.Lock()
		if generation != a.snapshotGenerations.current() {
			a.viewMu.Unlock()
			return
		}
		if err != nil {
			a.view.TimetableError = model.MessageOf(err)
		} else {
			a.view.Week = current.Week
			a.view.LastWeek = current.LastWeek
			a.view.Days = append([]model.DaySchedule(nil), current.Days...)
			a.view.TimetableSyncedAt = current.SyncedAt.Local().Format("2006-01-02 15:04")
			a.view.TimetableError = ""
		}
		a.view.TimetableLoading = false
		a.viewMu.Unlock()
	}()
}

func (a *desktopApp) startElectricityQuery(month string) {
	if !a.beginJob("electricity") {
		return
	}
	generation := a.snapshotGenerations.current()
	a.viewMu.Lock()
	a.view.ElectricityLoading = true
	a.view.ElectricityError = ""
	a.viewMu.Unlock()
	a.postMainUpdate()
	go func() {
		defer a.finishJob("electricity")
		ctx, cancel := a.commandContext()
		defer cancel()
		err := a.engine.QueryElectricity(ctx, month)
		a.viewMu.Lock()
		// 换账号会推进代次并重置视图：旧查询的收尾不得污染新视图（同 refreshAll）。
		if generation == a.snapshotGenerations.current() {
			a.view.ElectricityLoading = false
			if err != nil {
				a.electricityErr = model.MessageOf(err)
			} else {
				a.electricityErr = ""
			}
		}
		a.viewMu.Unlock()
	}()
}

// startElectricityPoll 是电费页「立即查询电费」：走 PollOnce，
// 保留轮询告警与退避语义（docs/windows.md §3.3）；月份查询才走 QueryElectricity。
func (a *desktopApp) startElectricityPoll() {
	if !a.beginJob("electricity") {
		return
	}
	generation := a.snapshotGenerations.current()
	a.viewMu.Lock()
	a.view.ElectricityLoading = true
	a.view.ElectricityError = ""
	a.viewMu.Unlock()
	a.postMainUpdate()
	go func() {
		defer a.finishJob("electricity")
		ctx, cancel := a.commandContext()
		defer cancel()
		err := a.engine.PollOnce(ctx)
		a.viewMu.Lock()
		if generation == a.snapshotGenerations.current() {
			a.view.ElectricityLoading = false
			if err != nil {
				a.electricityErr = model.MessageOf(err)
			} else {
				a.electricityErr = ""
			}
		}
		a.viewMu.Unlock()
	}()
}

func (a *desktopApp) startCampusAction(action string) {
	if !a.beginJob("campus") {
		return
	}
	generation := a.snapshotGenerations.current()
	a.viewMu.Lock()
	a.view.CampusLoading = true
	a.view.CampusError = ""
	a.viewMu.Unlock()
	a.postMainUpdate()
	go func() {
		defer a.finishJob("campus")
		ctx, cancel := a.commandContext()
		defer cancel()
		var err error
		switch action {
		case "refresh":
			_, err = a.engine.ProbeCampus(ctx)
		case "login":
			err = a.engine.DoCampus(ctx, desktop.ActionLogin)
		case "logout":
			err = a.engine.DoCampus(ctx, desktop.ActionLogout)
		case "relogin":
			err = a.engine.DoCampus(ctx, desktop.ActionRelogin)
		default:
			err = model.NewError(model.CodeInvalidArgument, "未知校园网动作")
		}
		a.viewMu.Lock()
		if generation == a.snapshotGenerations.current() {
			a.view.CampusLoading = false
			if err != nil {
				a.campusErr = model.MessageOf(err)
			} else {
				a.campusErr = ""
			}
		}
		a.viewMu.Unlock()
	}()
}

// refreshAll 并行刷新三个独立域，不串行等待课表登录完成才查电费。
func (a *desktopApp) refreshAll() {
	a.viewMu.Lock()
	if a.view.Refreshing || a.timetableInFlight {
		a.viewMu.Unlock()
		return
	}
	a.view.Refreshing = true
	a.view.TimetableLoading = true
	a.view.ElectricityLoading = true
	a.view.CampusLoading = true
	a.view.TimetableError = ""
	a.view.ElectricityError = ""
	a.view.CampusError = ""
	a.electricityErr = ""
	a.campusErr = ""
	ctx, cancel := a.commandContext()
	a.timetableInFlight = true
	a.timetableCancel = cancel
	generation := a.snapshotGenerations.next()
	a.viewMu.Unlock()
	a.postMainUpdate()

	go func() {
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			a.syncTimetable(ctx, a.currentSettings().TimetableConfig(), generation)
		}()
		go func() {
			defer wg.Done()
			err := a.engine.PollOnce(ctx)
			a.viewMu.Lock()
			if generation == a.snapshotGenerations.current() {
				if err != nil {
					a.electricityErr = model.MessageOf(err)
				} else {
					a.electricityErr = ""
				}
			}
			a.viewMu.Unlock()
		}()
		go func() {
			defer wg.Done()
			_, err := a.engine.ProbeCampus(ctx)
			a.viewMu.Lock()
			if generation == a.snapshotGenerations.current() {
				if err != nil {
					a.campusErr = model.MessageOf(err)
				} else {
					a.campusErr = ""
				}
			}
			a.viewMu.Unlock()
		}()
		wg.Wait()
		a.finishTimetable(cancel, generation)
		a.viewMu.Lock()
		// 换账号会重置视图并推进代次：旧刷新的收尾不得清掉新视图的加载态。
		if generation == a.snapshotGenerations.current() {
			a.view.Refreshing = false
			a.view.ElectricityLoading = false
			a.view.CampusLoading = false
		}
		a.viewMu.Unlock()
		a.postMainUpdate()
	}()
}

func (a *desktopApp) beginJob(name string) bool {
	a.viewMu.Lock()
	defer a.viewMu.Unlock()
	if a.view.Refreshing || a.activeJobs[name] {
		return false
	}
	a.activeJobs[name] = true
	return true
}

func (a *desktopApp) finishJob(name string) {
	a.viewMu.Lock()
	delete(a.activeJobs, name)
	a.viewMu.Unlock()
	a.postMainUpdate()
}

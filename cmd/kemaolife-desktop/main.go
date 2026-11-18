//go:build windows

// kemaolife-desktop 是 Windows 原生主窗口与常驻托盘程序（docs/windows.md）。
//
// 职责边界（AGENTS §3）：
//   - 本包只做「触发器 + 呈现 + 平台组装」：主窗口、托盘菜单、设置窗口、轮询循环；
//   - 业务逻辑来自 internal/app / internal/core / internal/desktop；
//   - 平台能力（DPAPI / 注册表自启 / Portal 认证）来自 internal/platform。
//
// 明确不做：保活 / 守护、多账号、云端同步、浏览器内核。
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/EchoPing07/kemaolife/cmd/kemaolife-desktop/ui"
	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
	"github.com/EchoPing07/kemaolife/internal/desktop"
	"github.com/EchoPing07/kemaolife/internal/platform"
	buildversion "github.com/EchoPing07/kemaolife/internal/version"
)

// 版本号取自仓库唯一来源 internal/version.Version，与 CLI 版同源（docs/contract.md §2）。
const version = buildversion.Version

var commit = "unknown"

// defaultCustomerCode 是学校代码内置默认值（学校事实，非个人隐私）。
var defaultCustomerCode = "3126"

// app 聚合常驻进程的全部运行状态。
type desktopApp struct {
	tray *ui.Tray
	tip  *ui.TipSetter
	wnd  uintptr

	settingsMu sync.RWMutex
	settings   desktop.Settings
	store      *desktop.SettingsStore
	secure     core.KVStore

	engine *desktop.Engine

	viewMu              sync.RWMutex
	view                ui.MainWindowState
	electricityErr      string
	campusErr           string
	timetableInFlight   bool
	timetableCancel     context.CancelFunc
	snapshotGenerations snapshotGeneration
	activeJobs          map[string]bool

	stopOnce sync.Once
	stop     chan struct{}
	rootCtx  context.Context
	cancel   context.CancelFunc
}

func main() {
	// 托盘程序不占控制台：构建成 GUI 子系统时这里是空操作，
	// 万一按 console 子系统构建，也能摘掉黑框而不是被黑框一起带走（见 console_windows.go）。
	detachConsole()

	// Win32 消息队列属于 OS 线程：必须锁住主 goroutine，
	// 否则 goroutine 迁移后 GetMessage 收不到自己的窗口消息（界面「假死」）。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ui.EnablePerMonitorDPI()
	release, ok := ui.AcquireSingleInstance()
	if !ok {
		ui.ShowExistingMainWindow()
		return
	}
	defer release()

	// 系统通知渠道：进程级 AUMID 必须在托盘图标创建前设置一次，
	// 否则系统通知不会归到 Kemaolife.Desktop。具体呈现样式随 Windows 版本变化，
	// 不承诺 WinRT Toast 或操作中心归档；失败只降级提示，不阻断轮询。
	if err := ui.SetAppUserModelID(); err != nil {
		ui.MessageBox(desktop.AppName, "系统通知初始化失败，将使用降级提示："+err.Error(),
			ui.MB_OK|ui.MB_ICONWARNING)
	}

	if err := run(); err != nil {
		ui.MessageBox(desktop.AppName, "启动失败："+err.Error(), ui.MB_OK|ui.MB_ICONERROR)
		os.Exit(1)
	}
}

func run() (err error) {
	// Win32 回调里发生 Go panic 会直接杀掉进程（syscall.NewCallback 不传播 panic），
	// 启动期把 panic 转成可见的错误提示，而不是静默消失。
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("启动异常: %v", r)
		}
	}()

	icon, err := ui.AppIcon()
	if err != nil {
		return err
	}
	trayIcon, err := ui.TrayIcon()
	if err != nil {
		return err
	}

	if err := ui.RegisterClass(ui.TrayClass, ui.TrayProc, icon); err != nil {
		return err
	}
	if err := ui.RegisterClass(ui.MainCls, ui.MainProc, icon); err != nil {
		return err
	}

	// 托盘窗口是隐藏窗：不显示界面，只接收托盘图标回调与自定义消息。
	trayWnd, err := ui.CreateWindow(ui.TrayClass, desktop.AppName, 0, 0, 0, 0)
	if err != nil {
		return err
	}

	a := &desktopApp{
		tray:       ui.NewTray(trayWnd, trayIcon),
		wnd:        trayWnd,
		stop:       make(chan struct{}),
		activeJobs: make(map[string]bool),
	}
	a.tip = ui.NewTipSetter(a.tray)

	// —— 平台组装 ——
	dataDir, err := platform.WindowsDataDir()
	if err != nil {
		return err
	}
	rawStore, err := platform.NewFileStore(dataDir)
	if err != nil {
		return err
	}
	// 配置含校园网凭据与学号，写盘前一律 DPAPI 加密（P5）。
	secure := platform.NewEncryptedStore(rawStore)
	a.secure = secure
	a.store = desktop.NewSettingsStore(secure)

	loaded, loadErr := a.store.Load()
	a.settings = loaded
	if loadErr != nil {
		// 配置损坏不阻断启动：用默认值继续，用户重新填一次即可。
		a.tip.Set("配置损坏，已使用默认值：" + loadErr.Error())
	}

	doer := platform.NewHTTPDoer(30 * time.Second)
	defer doer.CloseIdleConnections()

	deps := app.Deps{
		Clock:               platform.RealClock{},
		HTTP:                doer,
		Store:               secure,
		ConfigStore:         secure,
		DefaultCustomerCode: defaultCustomerCode,
	}

	// 校园网凭据每轮现读：设置窗口保存后立即生效，不需要重启。
	portal := platform.NewPortalNetwork(func() platform.PortalCredential {
		s := a.currentSettings()
		return platform.PortalCredential{
			UserID:      s.CampusUser,
			Password:    s.CampusPassword,
			Fingerprint: s.CampusFingerprint,
		}
	}, "")
	campusDeps := app.CampusDeps{Clock: platform.RealClock{}, Net: portal, Ready: portal}

	ctx, cancel := context.WithCancel(context.Background())
	a.rootCtx = ctx
	a.cancel = cancel
	a.view.AutoMode = a.currentSettings().AutoMode

	a.engine = desktop.NewEngine(deps, campusDeps, a.tray, a.currentSettings, func(st desktop.Status) {
		// 回调来自引擎 goroutine：只把托盘文本与一次刷新请求投递回 UI 线程，
		// 主窗口的状态一律由 UI 线程从 Engine.Status() 重新合成。
		ui.PostStatus(trayWnd, desktop.FormatStatus(st))
		ui.PostTrigger(trayWnd, ui.WM_QUERY_DONE)
	})

	ui.BindTray(trayWnd, ui.TrayHandler{
		BuildMenu:    a.buildMenu,
		OnCommand:    a.handleCommand,
		OnTrayClick:  a.openMain,
		OnStatusText: a.tip.Set,
		OnQueryDone:  a.refreshMainWindow,
	})

	if err := a.tray.Add(desktop.AppName); err != nil {
		return err
	}
	defer a.tray.Remove()

	background := len(os.Args) > 1 && os.Args[1] == "--background"
	if !background {
		// 主窗口在首个网络动作前创建，后台同步时显示局部加载态。
		a.openMain()
		if strings.TrimSpace(a.currentSettings().StudentID) == "" {
			ui.OpenMainPage(ui.MainPageSettings)
			a.openSettings()
		}
	}

	// 首轮立即查一次：开机自启后用户马上能看到数据，不必等一个轮询周期。
	go a.engine.Run(ctx, a.stop, func(error) {
		// 失败只更新托盘状态，不弹余额通知（docs/windows.md §6 规则 5）。
		ui.PostTrigger(trayWnd, ui.WM_QUERY_DONE)
	})

	go a.bootstrapTimetable()

	ui.Run()
	return nil
}

// currentSettings 返回当前设置的副本。
func (a *desktopApp) currentSettings() desktop.Settings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings
}

// applySettings 校验、落盘并生效一次设置保存（设置窗口的回调）。
func (a *desktopApp) applySettings(res desktop.SettingsResult) error {
	previous := a.currentSettings()
	next, err := desktop.ParseSettingsResult(res, previous)
	if err != nil {
		return err
	}

	if _, err := a.store.Save(next); err != nil {
		return err
	}

	// 自启开关立刻写注册表；失败要明确告知，不能「提示已保存但实际没生效」。
	if err := platform.SetAutoStart(next.AutoStart); err != nil {
		return fmt.Errorf("配置已保存，但写入开机自启失败：%w", err)
	}

	a.settingsMu.Lock()
	a.settings = next
	a.settingsMu.Unlock()

	a.refreshTrayTip()
	if strings.TrimSpace(previous.StudentID) != strings.TrimSpace(next.StudentID) {
		// 先推进代次再取消与删除：即使旧同步已收到响应，也不能写回新账号快照。
		a.snapshotGenerations.next()
		a.viewMu.Lock()
		cancel := a.timetableCancel
		a.timetableCancel = nil
		a.timetableInFlight = false
		a.view = ui.MainWindowState{TimetableError: "教务账号已更改，请重新同步课表"}
		a.viewMu.Unlock()
		if cancel != nil {
			cancel()
		}
		_ = a.secure.Delete(timetable.SnapshotStoreKey)
	}
	// 模式切换会改变按钮组，立即重建一次菜单状态。
	ui.PostTrigger(a.wnd, ui.WM_QUERY_DONE)

	if desktop.SettingsWereClamped(res, next) {
		return desktop.ErrSettingsClamped
	}
	return nil
}

// openSettings 切到主窗口「设置」页（托盘菜单项与校园网页按钮都走这里）。
func (a *desktopApp) openSettings() {
	a.openMain()
	ui.OpenMainPage(ui.MainPageSettings)
}

// saveSettingsFromPage 在主窗口设置页保存，返回提示文本；
// 钳制与保存失败都留在页内，只有完全成功才提示「已保存」。
func (a *desktopApp) saveSettingsFromPage(res desktop.SettingsResult) (hint string) {
	defer func() {
		a.refreshMainWindow()
	}()
	if err := a.applySettings(res); err != nil {
		if errors.Is(err, desktop.ErrSettingsClamped) {
			return desktop.ClampedSettingsHint(a.currentSettings())
		}
		return err.Error()
	}
	return "已保存"
}

// buildMenu 按当前状态与模式生成托盘右键菜单。
//
// 两行只读状态在上，按钮组按模式切换（docs/windows.md）：
// 手动模式给 登录 / 下线 / 重登；自动模式只给 重登。
func (a *desktopApp) buildMenu() []ui.MenuItem {
	s := a.currentSettings()
	st := a.engine.Status()
	campusLine, powerLine := desktop.TrayLine(st)

	items := []ui.MenuItem{
		{Title: "打开科贸生活", ID: ui.IDOpenMain},
		{Sep: true},
		{Title: campusLine, ID: ui.IDStatusCampus, Grayed: true},
		{Title: powerLine, ID: ui.IDStatusPower, Grayed: true},
		{Sep: true},
	}

	if !s.AutoMode {
		// 手动模式：三个动作都交给用户。
		items = append(items,
			ui.MenuItem{Title: "登录校园网", ID: ui.IDLogin},
			ui.MenuItem{Title: "下线", ID: ui.IDLogout},
		)
	}
	// 自动模式仍保留重登：自动登录只在未认证时触发，
	// 顶号 / 换账号这类情况必须由用户主动发起（docs/windows.md §10）。
	items = append(items,
		ui.MenuItem{Title: "重登", ID: ui.IDRelogin},
		ui.MenuItem{Title: "立即探测校园网", ID: ui.IDProbe},
		ui.MenuItem{Sep: true},
		ui.MenuItem{Title: "立即查询电费", ID: ui.IDRefresh},
		ui.MenuItem{Title: "设置…", ID: ui.IDSettings},
		ui.MenuItem{Sep: true},
		ui.MenuItem{Title: "退出", ID: ui.IDExit},
	)
	return items
}

// handleCommand 分派托盘菜单命令。
//
// 分两条路：⚠️ 所有网络动作必须离开 UI 线程。窗口过程在消息循环里同步执行，
// 一个 5s 探测 / 3s 认证生效等待 / 3 个电费请求就能让右键菜单卡成「无响应」。
// 网络动作在 goroutine 里跑，结果经 PostStatus / PostTrigger 回到 UI 线程。
type campusJob func(context.Context, app.CampusDeps) error

var campusJobs = map[int]campusJob{
	ui.IDLogin:   app.LoginCampusNetwork,
	ui.IDLogout:  app.LogoutCampusNetwork,
	ui.IDRelogin: app.ReloginCampusNetwork,
}

func (a *desktopApp) handleCommand(id int) {
	if job, ok := campusJobs[id]; ok {
		a.runAsync(func(ctx context.Context) {
			a.reportCampusErr(ctx, job(ctx, a.engine.CampusDeps()))
		})
		return
	}

	switch id {
	case ui.IDOpenMain:
		a.openMain()
	case ui.IDProbe:
		a.runAsync(func(ctx context.Context) { a.engine.ProbeCampus(ctx) })
	case ui.IDRefresh:
		a.runAsync(func(ctx context.Context) {
			if err := a.engine.PollOnce(ctx); err != nil {
				a.engine.ReportError(err)
			}
		})
	case ui.IDSettings:
		a.openSettings()
	case ui.IDExit:
		a.shutdown()
	}
}

// runAsync 在后台执行网络动作，完成后把状态刷新推回 UI 线程。
func (a *desktopApp) runAsync(fn func(context.Context)) {
	go func() {
		ctx, cancel := a.commandContext()
		defer cancel()
		fn(ctx)
		ui.PostTrigger(a.wnd, ui.WM_QUERY_DONE)
	}()
}

// reportCampusErr 把校园网动作结果反映到托盘（不弹窗打断用户）。
func (a *desktopApp) reportCampusErr(ctx context.Context, err error) {
	a.engine.ReportError(err)
	a.engine.RefreshCampusStatus(ctx)
}

// commandContext 给用户点击触发的动作 60s：Portal 有 3s 生效等待 + 复探，
// 电费要发 3 个请求，30s 太紧。
func (a *desktopApp) commandContext() (context.Context, context.CancelFunc) {
	parent := a.rootCtx
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, 60*time.Second)
}

// refreshTrayTip 用引擎当前状态刷新托盘提示。
func (a *desktopApp) refreshTrayTip() {
	a.tip.Set(desktop.FormatStatus(a.engine.Status()))
}

// shutdown 退出流程：先停轮询，再注销托盘，最后结束消息循环。
//
// 顺序很重要：先停轮询能避免退出时还在打学校接口，
// 先摘图标能避免「进程已退出但托盘还留着死图标」。
func (a *desktopApp) shutdown() {
	a.stopOnce.Do(func() { close(a.stop) })
	if a.cancel != nil {
		a.cancel()
	}
	a.tray.Remove()
	ui.Quit()
}

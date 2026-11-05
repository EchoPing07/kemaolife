//go:build windows

// kemaolife-agent 是 Windows 常驻托盘程序（docs/000 §3.2、docs/002 §5.2）。
//
// 职责边界（AGENTS §3）：
//   - 本包只做「触发器 + 呈现 + 平台组装」：托盘菜单、设置窗口、轮询循环；
//   - 业务逻辑来自 internal/app / internal/core / internal/agent；
//   - 平台能力（DPAPI / 注册表自启 / Portal 认证）来自 internal/platform。
//
// 明确不做（000 §3.2）：课表主界面、保活 / 守护、多账号、云端同步、浏览器内核。
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/EchoPing07/kemaolife/cmd/kemaolife-agent/ui"
	"github.com/EchoPing07/kemaolife/internal/agent"
	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/platform"
	buildversion "github.com/EchoPing07/kemaolife/internal/version"
)

// 版本号取自仓库唯一来源 internal/version.Version，与 Linux 端同源（docs/002 §1.1）。
const version = buildversion.Version

var commit = "unknown"

// defaultCustomerCode 是学校代码内置默认值（学校事实，非个人隐私）。
var defaultCustomerCode = "3126"

// app 聚合常驻进程的全部运行状态。
type agentApp struct {
	tray *ui.Tray
	tip  *ui.TipSetter
	wnd  uintptr

	settingsMu sync.RWMutex
	settings   agent.Settings
	store      *agent.SettingsStore

	engine *agent.Engine

	stopOnce sync.Once
	stop     chan struct{}
	cancel   context.CancelFunc
}

func main() {
	// Win32 消息队列属于 OS 线程：必须锁住主 goroutine，
	// 否则 goroutine 迁移后 GetMessage 收不到自己的窗口消息（界面「假死」）。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ui.EnablePerMonitorDPI()

	release, ok := ui.AcquireSingleInstance()
	if !ok {
		ui.MessageBox(agent.AppName, "科贸生活已在运行，请查看系统托盘。", ui.MB_OK|ui.MB_ICONINFO)
		return
	}
	defer release()

	if err := run(); err != nil {
		ui.MessageBox(agent.AppName, "启动失败："+err.Error(), ui.MB_OK|ui.MB_ICONERROR)
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
	if err := ui.RegisterClass(ui.SettingsCls, ui.SettingsProc, icon); err != nil {
		return err
	}

	// 托盘窗口是隐藏窗：不显示界面，只接收托盘图标回调与自定义消息。
	trayWnd, err := ui.CreateWindow(ui.TrayClass, agent.AppName, 0, 0, 0, 0)
	if err != nil {
		return err
	}

	a := &agentApp{
		tray: ui.NewTray(trayWnd, trayIcon),
		wnd:  trayWnd,
		stop: make(chan struct{}),
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
	a.store = agent.NewSettingsStore(secure)

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
	})
	campusDeps := app.CampusDeps{Clock: platform.RealClock{}, Net: portal, Ready: portal}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel

	a.engine = agent.NewEngine(deps, campusDeps, a.tray, a.currentSettings, func(st agent.Status) {
		// 回调来自轮询 goroutine：必须转回 UI 线程再更新托盘。
		ui.PostStatus(trayWnd, agent.FormatStatus(st))
	})

	ui.BindTray(trayWnd, ui.TrayHandler{
		BuildMenu:    a.buildMenu,
		OnCommand:    a.handleCommand,
		OnTrayClick:  a.openSettings,
		OnStatusText: a.tip.Set,
		OnQueryDone:  a.refreshTrayTip,
	})

	if err := a.tray.Add(agent.AppName); err != nil {
		return err
	}
	defer a.tray.Remove()

	// 首次启动（没有学号）直接打开设置：不填配置什么也查不到，
	// 让用户去翻托盘菜单是没必要的绕路。
	if strings.TrimSpace(a.currentSettings().StudentID) == "" {
		ui.PostTrigger(trayWnd, ui.WM_OPEN_SETTINGS)
	}

	// 首轮立即查一次：开机自启后用户马上能看到数据，不必等一个轮询周期。
	go a.engine.Run(ctx, a.stop, func(error) {
		// 失败只更新托盘状态，不弹余额通知（docs/002 §6 规则 5）。
		ui.PostTrigger(trayWnd, ui.WM_QUERY_DONE)
	})

	ui.Run()
	return nil
}

// currentSettings 返回当前设置的副本。
func (a *agentApp) currentSettings() agent.Settings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings
}

// applySettings 校验、落盘并生效一次设置保存（设置窗口的回调）。
func (a *agentApp) applySettings(res agent.SettingsResult) error {
	next, err := agent.ParseSettingsResult(res, a.currentSettings())
	if err != nil {
		return err
	}

	clamped, err := a.store.Save(next)
	if err != nil {
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
	// 模式切换会改变按钮组，立即重建一次菜单状态。
	ui.PostTrigger(a.wnd, ui.WM_QUERY_DONE)

	if clamped {
		return fmt.Errorf("轮询间隔低于 600 秒下限，已按 %d 秒执行",
			int(next.PollInterval()/time.Second))
	}
	return nil
}

// openSettings 打开设置窗口（托盘左键或菜单项）。
func (a *agentApp) openSettings() {
	icon, _ := ui.AppIcon()
	ui.OpenSettings(a.wnd, icon, agent.ToSettingsForm(a.currentSettings()), a.applySettings)
}

// buildMenu 按当前状态与模式生成托盘右键菜单。
//
// 两行只读状态在上，按钮组按模式切换（docs/002 §5.2）：
// 手动模式给 登录 / 下线 / 重登；自动模式只给 重登。
func (a *agentApp) buildMenu() []ui.MenuItem {
	s := a.currentSettings()
	st := a.engine.Status()
	campusLine, powerLine := agent.TrayLine(st)

	items := []ui.MenuItem{
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
	// 顶号 / 换账号这类情况必须由用户主动发起（000 §3.2）。
	items = append(items,
		ui.MenuItem{Title: "重登", ID: ui.IDRelogin},
		ui.MenuItem{Sep: true},
		ui.MenuItem{Title: "立即查询电费", ID: ui.IDRefresh},
		ui.MenuItem{Title: "设置…", ID: ui.IDSettings},
		ui.MenuItem{Sep: true},
		ui.MenuItem{Title: "退出", ID: ui.IDExit},
	)
	return items
}

// handleCommand 分派托盘菜单命令。
func (a *agentApp) handleCommand(id int) {
	ctx, cancel := a.commandContext()
	defer cancel()

	switch id {
	case ui.IDLogin:
		a.reportCampusErr(ctx, app.LoginCampusNetwork(ctx, a.engine.CampusDeps()))
	case ui.IDLogout:
		a.reportCampusErr(ctx, app.LogoutCampusNetwork(ctx, a.engine.CampusDeps()))
	case ui.IDRelogin:
		a.reportCampusErr(ctx, app.ReloginCampusNetwork(ctx, a.engine.CampusDeps()))
	case ui.IDRefresh:
		a.engine.PollOnce(ctx)
	case ui.IDSettings:
		a.openSettings()
	case ui.IDExit:
		a.shutdown()
	}
	// 任何动作之后都刷新一次提示，让用户立刻看到结果。
	a.refreshTrayTip()
}

// reportCampusErr 把校园网动作结果反映到托盘（不弹窗打断用户）。
func (a *agentApp) reportCampusErr(ctx context.Context, err error) {
	a.engine.ReportError(err)
	a.engine.RefreshCampusStatus(ctx)
}

// commandContext 给用户点击触发的动作 60s：Portal 有 3s 生效等待 + 复探，
// 电费要发 3 个请求，30s 太紧。
func (a *agentApp) commandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

// refreshTrayTip 用引擎当前状态刷新托盘提示。
func (a *agentApp) refreshTrayTip() {
	a.tip.Set(agent.FormatStatus(a.engine.Status()))
}

// shutdown 退出流程：先停轮询，再注销托盘，最后结束消息循环。
//
// 顺序很重要：先停轮询能避免退出时还在打学校接口，
// 先摘图标能避免「进程已退出但托盘还留着死图标」。
func (a *agentApp) shutdown() {
	a.stopOnce.Do(func() { close(a.stop) })
	if a.cancel != nil {
		a.cancel()
	}
	a.tray.Remove()
	ui.Quit()
}

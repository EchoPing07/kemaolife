package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/alert"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Status 是托盘要展示的两行只读状态（docs/002 §5.2「托盘右键」）。
type Status struct {
	Campus        model.CampusState
	CampusMsg     string
	HasReport     bool
	Balance       float64
	Unit          string
	CollectedAt   string
	EstimatedDays *int
	LastError     string
	LastCheckedAt time.Time
}

// Engine 是常驻轮询的编排层：把 app 用例、告警纯函数、通知端口串起来。
//
// 它不含任何平台调用（不 import platform），所有能力经端口注入，
// 因此可以在任何平台上用 fake 跑单测 —— 但只有 Windows 端会实例化它。
type Engine struct {
	mu sync.Mutex

	deps     app.Deps
	campus   app.CampusDeps
	notifier core.Notifier
	settings func() Settings // 每轮重新读，设置窗口保存后立即生效
	sink     func(Status)    // 状态变化回调（UI 用 PostMessage 转回 UI 线程）

	alertState alert.State
	status     Status
	failures   int
}

// NewEngine 构造引擎。settings 与 sink 都可为空（后者空表示只更新内部状态）。
func NewEngine(deps app.Deps, campus app.CampusDeps, notifier core.Notifier,
	settings func() Settings, sink func(Status)) *Engine {
	e := &Engine{
		deps: deps, campus: campus, notifier: notifier,
		settings: settings, sink: sink,
	}
	e.status.Campus = model.CampusOffline
	return e
}

// Status 返回当前状态的副本（UI 只读，避免数据竞争）。
func (e *Engine) Status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status
}

// SetClock 仅供测试替换时钟（正式路径由 Deps.Clock 注入）。
func (e *Engine) SetClock(c core.Clock) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deps.Clock = c
	e.campus.Clock = c
}

func (e *Engine) updateStatus(fn func(*Status)) {
	e.mu.Lock()
	fn(&e.status)
	s := e.status
	e.mu.Unlock()
	if e.sink != nil {
		e.sink(s)
	}
}

// CampusAction 是托盘按钮触发的校园网动作。
type CampusAction int

const (
	ActionLogin CampusAction = iota
	ActionLogout
	ActionRelogin
)

// DoCampus 执行一次手动校园网动作（按钮回调）。
//
// 无论成败都立刻刷新一次托盘状态：用户点了按钮必须看到反馈，
// 不能等到下一次轮询（默认 4 小时后）。
func (e *Engine) DoCampus(ctx context.Context, action CampusAction) error {
	var err error
	switch action {
	case ActionLogin:
		err = app.LoginCampusNetwork(ctx, e.campus)
	case ActionLogout:
		err = app.LogoutCampusNetwork(ctx, e.campus)
	case ActionRelogin:
		err = app.ReloginCampusNetwork(ctx, e.campus)
	default:
		err = model.NewError(model.CodeInvalidArgument, "未知校园网动作")
	}

	if err != nil {
		e.updateStatus(func(s *Status) { s.LastError = model.MessageOf(err) })
	} else {
		e.updateStatus(func(s *Status) { s.LastError = "" })
	}
	e.RefreshCampusStatus(ctx)
	return err
}

// RefreshCampusStatus 只刷新判态（不登录），用于托盘打开与按钮动作之后。
func (e *Engine) RefreshCampusStatus(ctx context.Context) {
	status, err := app.GetCampusNetworkStatus(ctx, e.campus)
	e.updateStatus(func(s *Status) {
		if err != nil {
			// 判态本身失败不覆盖上一次已知状态，只记错误。
			s.LastError = model.MessageOf(err)
			return
		}
		s.Campus = status.State
		s.CampusMsg = status.ErrorMsg
	})
}

// PollOnce 完整执行一轮：自动模式先恢复网络 → 查电费 → 判告警 → 通知。
//
// 失败只走退避与状态更新，不弹余额通知（docs/002 §6 规则 5）。
func (e *Engine) PollOnce(ctx context.Context) error {
	cfg := e.settings()

	// 自动模式的前置恢复；失败不阻断（也许只是没连校园网 WiFi），
	// 但要让用户知道，所以刷新一次状态。
	if cfg.AutoMode {
		if err := app.EnsureCampusNetwork(ctx, e.campus); err != nil {
			e.updateStatus(func(s *Status) { s.LastError = model.MessageOf(err) })
		}
		e.RefreshCampusStatus(ctx)
	}

	report, err := app.CheckElectricity(ctx, e.deps, cfg.ElectricityConfig(e.deps.DefaultCustomerCode), "")
	if err != nil {
		err = withSettingsHint(err)
		e.mu.Lock()
		e.failures++
		e.mu.Unlock()
		e.updateStatus(func(s *Status) {
			s.LastError = model.MessageOf(err)
			s.LastCheckedAt = e.deps.Clock.Now()
		})
		return err
	}

	e.mu.Lock()
	e.failures = 0
	e.mu.Unlock()

	now := e.deps.Clock.Now()
	e.updateStatus(func(s *Status) {
		s.HasReport = true
		s.Balance = report.Status.Balance
		s.Unit = report.Status.Unit
		s.CollectedAt = report.Status.CollectedAt
		s.EstimatedDays = report.Status.EstimatedDays
		s.LastError = ""
		s.LastCheckedAt = now
	})

	dec, nextState := alert.Evaluate(e.alertState, report.Status, now, cfg.AlertConfig())
	e.alertState = nextState
	if dec.Notify && e.notifier != nil {
		if err := e.notifier.Notify(dec.Title, dec.Body); err != nil {
			// 通知失败不改变告警状态：状态已置位，本轮不再重复弹，
			// 但也不该因为这个错误把电费数据丢掉。
			e.updateStatus(func(s *Status) { s.LastError = "通知发送失败: " + err.Error() })
		}
	}
	return nil
}

// Run 是常驻循环：成功按轮询间隔，连续失败按指数退避（docs/002 §6 规则 6）。
//
// stop 关闭后立即退出；ctx 用于随进程退出取消在途请求。
func (e *Engine) Run(ctx context.Context, stop <-chan struct{}, onErr func(error)) {
	for {
		err := e.PollOnce(ctx)
		if err != nil && onErr != nil {
			onErr(err)
		}

		var wait time.Duration
		e.mu.Lock()
		failures := e.failures
		e.mu.Unlock()
		if failures == 0 {
			wait = e.settings().PollInterval()
		} else {
			wait = alert.NextBackoff(e.settings().PollInterval(), failures)
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-stop:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// TrayLine 返回托盘右键的两行只读状态文本（docs/002 §5.2）。
//
// 这里是纯格式化，不含 IO；数字与「数据截至」必须同时出现，
// 避免用户把 1 天前的数据当实时值。
func TrayLine(s Status) (campusLine, powerLine string) {
	switch s.Campus {
	case model.CampusOnline:
		campusLine = "校园网：已在线"
	case model.CampusUnauth:
		campusLine = "校园网：未认证"
	default:
		campusLine = "校园网：不在校园网"
	}
	if s.CampusMsg != "" {
		campusLine += "（" + s.CampusMsg + "）"
	}

	switch {
	case !s.HasReport:
		powerLine = "剩余电费：未获取"
		if s.LastError != "" {
			powerLine += "（" + s.LastError + "）"
		}
	case s.EstimatedDays != nil:
		powerLine = fmt.Sprintf("剩余电费：%.2f %s（约可用 %d 天）",
			s.Balance, unitOr(s.Unit), *s.EstimatedDays)
	default:
		powerLine = fmt.Sprintf("剩余电费：%.2f %s", s.Balance, unitOr(s.Unit))
	}
	if s.HasReport {
		if until := formatCollected(s.CollectedAt); until != "" {
			powerLine += "｜数据截至 " + until
		}
	}
	return campusLine, powerLine
}

func unitOr(u string) string {
	if strings.TrimSpace(u) == "" {
		return "度"
	}
	return u
}

// formatCollected 把 RFC3339 转成本地「MM-DD HH:MM」；解析失败原样返回。
func formatCollected(raw string) string {
	if raw == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.In(time.Local).Format("01-02 15:04")
	}
	return raw
}

// withSettingsHint 给凭据类错误补上「在托盘程序里怎么修」的指引。
//
// 与 CLI 的 withCLIHint 对称：core / app 只报中性原因，
// 各端各自说明修复入口（这里的入口是托盘「设置…」）。
func withSettingsHint(err error) error {
	if model.CodeOf(err) != model.CodeCredentialMissing {
		return err
	}
	if strings.Contains(model.MessageOf(err), "设置") {
		return err
	}
	return model.WrapError(model.CodeCredentialMissing,
		model.MessageOf(err)+"，请在托盘菜单打开「设置…」填写", err)
}

// CampusDeps 返回校园网用例需要的端口（cmd 层手动动作直接用）。
func (e *Engine) CampusDeps() app.CampusDeps { return e.campus }

// ReportError 记录一次错误到状态（供 cmd 层在动作失败后提示）。
func (e *Engine) ReportError(err error) {
	if err == nil {
		e.updateStatus(func(s *Status) { s.LastError = "" })
		return
	}
	e.updateStatus(func(s *Status) { s.LastError = model.MessageOf(err) })
}

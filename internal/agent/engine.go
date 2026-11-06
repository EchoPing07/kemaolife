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
	Campus          model.CampusState
	CampusMsg       string
	CampusCheckedAt time.Time
	HasReport       bool
	Balance         float64
	Unit            string
	CollectedAt     string
	EstimatedDays   *int
	LastError       string
	LastCheckedAt   time.Time
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
//
// 应用层已经把「网络探测失败」归为 offline；这里再对 4001 做一次兜底，
// 让 Windows 的判态轮询在临时网络故障时只显示「不在校园网」，不把失败原因
// 当异常塞进悬浮提示（docs/002 §5.2：探测失败归 offline，不退避）。
func (e *Engine) RefreshCampusStatus(ctx context.Context) {
	status, err := app.GetCampusNetworkStatus(ctx, e.campus)
	if model.CodeOf(err) == model.CodeNetworkDown {
		e.updateStatus(func(s *Status) {
			s.Campus = model.CampusOffline
			s.CampusMsg = ""
			s.CampusCheckedAt = e.deps.Clock.Now()
			s.LastError = ""
		})
		return
	}
	e.updateStatus(func(s *Status) {
		if err != nil {
			// 内部故障仍保留提示，但不覆盖上一次已知状态。
			s.LastError = model.MessageOf(err)
			return
		}
		s.Campus = status.State
		s.CampusMsg = status.ErrorMsg
		s.CampusCheckedAt = e.deps.Clock.Now()
	})
}

// ProbeCampus 是「手动触发探测」：立即探一次判态，不登录、不改任何凭据。
//
// 与 RefreshCampusStatus 分开的原因是返回值：手动探测要把结果/错误直接告诉
// 调用方（托盘要反馈本次动作），而 RefreshCampusStatus 是内部串联调用。
func (e *Engine) ProbeCampus(ctx context.Context) (model.CampusState, error) {
	status, err := app.GetCampusNetworkStatus(ctx, e.campus)
	if model.CodeOf(err) == model.CodeNetworkDown {
		e.updateStatus(func(s *Status) {
			s.Campus = model.CampusOffline
			s.CampusMsg = ""
			s.CampusCheckedAt = e.deps.Clock.Now()
			s.LastError = ""
		})
		return model.CampusOffline, nil
	}
	if err != nil {
		e.updateStatus(func(s *Status) { s.LastError = model.MessageOf(err) })
		return model.CampusOffline, err
	}
	e.updateStatus(func(s *Status) {
		s.Campus = status.State
		s.CampusMsg = status.ErrorMsg
		s.CampusCheckedAt = e.deps.Clock.Now()
	})
	return status.State, nil
}

// PollOnce 完整执行一轮：自动模式先恢复网络 → 查电费 → 判告警 → 通知。
//
// 失败只走退避与状态更新，不弹余额通知（docs/002 §6 规则 5）。
func (e *Engine) PollOnce(ctx context.Context) error {
	cfg := e.settings()
	var preflightErr error

	// 自动模式的前置恢复；失败不阻断（也许只是没连校园网 WiFi），
	// 但要让用户知道，所以刷新一次状态。
	if cfg.AutoMode {
		if err := app.EnsureCampusNetwork(ctx, e.campus); err != nil {
			preflightErr = err
		}
		e.RefreshCampusStatus(ctx)
		// 判态轮询把 4001 归为 offline 时会清 LastError；前置失败要留在提示里，
		// 否则用户只看到「不在校园网」，不知道自动恢复为什么没做。
		if preflightErr != nil {
			e.updateStatus(func(s *Status) { s.LastError = model.MessageOf(preflightErr) })
		}
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
		// 前置认证失败与电费查询是独立链路：电费成功不能把它从托盘提示里抹掉，
		// 但下一轮前置成功时要清掉旧提示（docs/002 §6 规则 5）。
		if preflightErr != nil {
			s.LastError = model.MessageOf(preflightErr)
		} else {
			s.LastError = ""
		}
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
	// 校园网判态轮询与电费轮询是两个独立周期（docs/002 §5.2）：
	// 判态只是 1 个探测请求，可以密得多，单独跑才不会把电费接口拉上。
	go e.runCampusProbe(ctx, stop)

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

// runCampusProbe 按设置的探测间隔反复判态（默认 120s）。
//
// 只调 Status（1 个探测请求）：不发登录、不改凭据，也不因失败退避 ——
// 网络探测失败（4001）由 ProbeCampus 归一为 offline，是正常结果而不是故障；
// 内部故障仍会进 LastError 供悬浮提示显示，但不改变探测周期
// （docs/002 §5.2 / §3.3）。间隔每轮重读设置，设置窗口保存后立即生效。
func (e *Engine) runCampusProbe(ctx context.Context, stop <-chan struct{}) {
	// 首轮立即探：否则托盘上的校园网状态要等一个完整周期才有值。
	for {
		// ProbeCampus 自己会经 sink 回报状态（成功与失败都报），这里不重复推。
		e.ProbeCampus(ctx)

		timer := time.NewTimer(e.settings().CampusProbeInterval())
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
// 这里是纯格式化，不含 IO。每行必须单行放得下（菜单项不换行，过长会被吃掉），
// 因此只留状态与余额：
//   - 探测时刻不展示（轮询默认 120s，刷新反而刷屏，时刻也不能说明网络是否可用）；
//   - 上游长文案（如「正在进行外网拨号请稍候…」）不展示，一律收敛为「已在线」；
//   - 可用天数不展示：它是本地估算，容易被误读为承诺。
func TrayLine(s Status) (campusLine, powerLine string) {
	campusLine = "校园网：" + campusStateText(s.Campus)
	if !s.HasReport {
		powerLine = "剩余电费：未获取"
		return campusLine, powerLine
	}
	powerLine = fmt.Sprintf("剩余电费：%.2f %s", s.Balance, unitOr(s.Unit))
	return campusLine, powerLine
}

// errorLine 返回附加在提示里的错误文案（失败原因必须可在本机看到）。
func errorLine(s Status) string {
	return strings.TrimSpace(s.LastError)
}

func unitOr(u string) string {
	if strings.TrimSpace(u) == "" {
		return "度"
	}
	return u
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

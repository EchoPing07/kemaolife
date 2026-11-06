package agent

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/alert"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// AppName 是三端共用的展示名（窗口标题 / 通知标题 / 托盘提示）。
const AppName = "科贸生活"

// FormatStatus 把引擎状态渲染成托盘提示的三行文本。
//
// 只放用户能行动的信息：校园网状态、剩余电费（含数据截至）、最近一次错误。
// 不放 roomName / roomVerify（P2）。
func FormatStatus(s Status) string {
	campus, power := TrayLine(s)
	line := AppName + "\n" + campus + "\n" + power
	if s.LastError != "" {
		line += "\n" + s.LastError
	}
	return line
}

// toSettingsForm 把持久化设置转成 UI 表单初始值。
func ToSettingsForm(s Settings) SettingsForm {
	return SettingsForm{
		StudentID:         s.StudentID,
		CustomerCode:      s.CustomerCode,
		CampusUser:        s.CampusUser,
		CampusPassword:    s.CampusPassword,
		CampusFingerprint: s.CampusFingerprint,
		AutoMode:          s.AutoMode,
		AutoStart:         s.AutoStart,
		PollSeconds:       s.PollSeconds,
		CampusProbeSecs:   s.CampusProbeSecs,
		BalanceThreshold:  strconv.FormatFloat(s.BalanceThreshold, 'f', -1, 64),
		DaysThreshold:     s.DaysThreshold,
		QuietFrom:         s.QuietStartHour,
		QuietTo:           s.QuietEndHour,
	}
}

// parseSettingsResult 校验设置窗口回传的原文，转成持久化设置。
//
// 校验集中在这一处：UI 层只管收字符串，业务规则（下限、范围）只在这里定，
// 避免「设置窗口一套规则、命令行另一套」的漂移。
func ParseSettingsResult(res SettingsResult, base Settings) (Settings, error) {
	next := base
	next.StudentID = strings.TrimSpace(res.StudentID)
	next.CustomerCode = strings.TrimSpace(res.CustomerCode)
	next.CampusUser = strings.TrimSpace(res.CampusUser)
	next.CampusPassword = res.CampusPassword
	next.CampusFingerprint = strings.TrimSpace(res.CampusFingerprint)
	next.AutoMode = res.AutoMode
	next.AutoStart = res.AutoStart

	poll, err := strconv.Atoi(strings.TrimSpace(res.PollSeconds))
	if err != nil || poll <= 0 {
		return base, fmt.Errorf("轮询间隔必须是正整数（秒）")
	}
	// 低于 600 秒一律钳制，不拒绝保存（AGENTS §5.5：钳制到 600s 并提示）。
	if d, clamped := alert.NormalizePollInterval(secondsDuration(poll)); clamped {
		poll = int(d.Seconds())
	}
	next.PollSeconds = poll

	probe, err := strconv.Atoi(strings.TrimSpace(res.CampusProbeSecs))
	if err != nil || probe <= 0 {
		return base, fmt.Errorf("校园网探测间隔必须是正整数（秒）")
	}
	// 低于下限钳制而非拒绝，与轮询间隔同一套口径（钳制要提示，不能静默改）。
	if d, clamped := NormalizeCampusProbeInterval(secondsDuration(probe)); clamped {
		probe = int(d.Seconds())
	}
	next.CampusProbeSecs = probe

	balance, err := strconv.ParseFloat(strings.TrimSpace(res.BalanceThreshold), 64)
	if err != nil || balance <= 0 {
		return base, fmt.Errorf("余额阈值必须是正数（度）")
	}
	next.BalanceThreshold = balance

	days, err := strconv.Atoi(strings.TrimSpace(res.DaysThreshold))
	if err != nil || days <= 0 {
		return base, fmt.Errorf("可用天数阈值必须是正整数")
	}
	next.DaysThreshold = days

	from, err := strconv.Atoi(strings.TrimSpace(res.QuietFrom))
	if err != nil || from < 0 || from > 23 {
		return base, fmt.Errorf("静默起点必须是 0–23 的整数（时）")
	}
	to, err := strconv.Atoi(strings.TrimSpace(res.QuietTo))
	if err != nil || to < 0 || to > 23 {
		return base, fmt.Errorf("静默终点必须是 0–23 的整数（时）")
	}
	next.QuietStartHour = from
	next.QuietEndHour = to

	return next, nil
}

// campusStateText 把三态转成中文短语（托盘菜单与提示共用）。
func campusStateText(s model.CampusState) string {
	switch s {
	case model.CampusOnline:
		return "已在线"
	case model.CampusUnauth:
		return "未认证"
	default:
		return "不在校园网"
	}
}

// secondsDuration 把秒数转成 Duration（避免在多个文件里重复写 time.Second）。
func secondsDuration(secs int) time.Duration {
	return time.Duration(secs) * time.Second
}

// SettingsForm 是设置窗口的初始值（与 internal/agent.Settings 同构，
// 数值以字符串传，便于窗口控件直接填充）。
type SettingsForm struct {
	StudentID         string
	CustomerCode      string
	CampusUser        string
	CampusPassword    string
	CampusFingerprint string
	AutoMode          bool
	AutoStart         bool
	PollSeconds       int
	CampusProbeSecs   int
	BalanceThreshold  string
	DaysThreshold     int
	QuietFrom         int
	QuietTo           int
}

// SettingsResult 是设置窗口保存时回传的原文，校验由 ParseSettingsResult 负责。
type SettingsResult struct {
	StudentID         string
	CustomerCode      string
	CampusUser        string
	CampusPassword    string
	CampusFingerprint string
	AutoMode          bool
	AutoStart         bool
	PollSeconds       string
	CampusProbeSecs   string
	BalanceThreshold  string
	DaysThreshold     string
	QuietFrom         string
	QuietTo           string
}

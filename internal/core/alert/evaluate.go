// Package alert 实现 Windows 端电费告警判定（docs/002 §6）。
//
// 这里只有纯函数：不读时钟、不发通知、不写文件。调用方（cmd/kemaolife-agent）
// 负责把 Clock 取到的时间传进来、把通知交给 Notifier、把状态留在内存。
package alert

import (
	"fmt"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// 默认值（docs/002 §6「默认值」表）。轮询间隔已定稿；阈值与静默时段待 D3 定稿。
const (
	DefaultBalanceThreshold = 10.0
	DefaultDaysThreshold    = 3
	DefaultQuietStartHour   = 23
	DefaultQuietEndHour     = 7

	// PollIntervalFloor 是轮询间隔硬下限（AGENTS §5.5：≥ 600 秒，不可放行）。
	PollIntervalFloor = 10 * time.Minute
	// DefaultPollInterval 是默认轮询间隔（docs/002 §6：4 小时）。
	DefaultPollInterval = 4 * time.Hour
	// MaxBackoff 是失败退避上限；TODO(D3): 上限数值待定稿，此处取「大于轮询间隔」的保守值。
	MaxBackoff = 24 * time.Hour
)

// Config 是告警判定需要的配置（来自设置窗口）。
type Config struct {
	BalanceThreshold float64 // 余额 ≤ 该值触发（度）
	DaysThreshold    int     // 可用天数 ≤ 该值触发
	QuietStartHour   int     // 静默起点（含），与 QuietEndHour 相同表示无静默时段
	QuietEndHour     int     // 静默终点（不含）
}

// State 是需要跨轮次保存的告警状态。
//
// 用两个独立标记而不是一个「已提醒」布尔：余额与天数阈值互相独立，
// 余额回升（充值）只重置余额标记，不该把天数标记一起清掉。
type State struct {
	BalanceAlerted bool `json:"balanceAlerted"`
	DaysAlerted    bool `json:"daysAlerted"`
}

// Reason 是本次触发的原因。
type Reason int

const (
	ReasonNone    Reason = iota
	ReasonBalance        // 余额跌到阈值
	ReasonDays           // 可用天数跌到阈值
)

// Decision 是一次判定结果。
type Decision struct {
	Notify bool
	Title  string
	Body   string
	Reason Reason
}

// Evaluate 判定本轮是否要弹通知，并返回下一轮要保存的状态。
//
// 规则（docs/002 §6）：
//  1. 余额 ≤ 阈值 或 可用天数 ≤ 阈值 触发；
//  2. 同一触发原因只提醒一次，余额回升到阈值以上后重置；
//  3. 静默时段内不发通知，但满足条件时**不置位**标记 —— 退出静默后补发一次；
//  4. 正文必须带「数据截至」；
//  5. 失败不告警（由调用方只在成功路径调用本函数）。
func Evaluate(prev State, cur model.ElectricityStatus, now time.Time, cfg Config) (Decision, State) {
	next := prev
	cfg = cfg.withDefaults()

	// 回升即重置：这是「充值后能再次提醒」的唯一入口。
	if cur.Balance > cfg.BalanceThreshold {
		next.BalanceAlerted = false
	}
	if cur.EstimatedDays == nil || *cur.EstimatedDays > cfg.DaysThreshold {
		next.DaysAlerted = false
	}

	reason := ReasonNone
	switch {
	case cur.Balance <= cfg.BalanceThreshold && !next.BalanceAlerted:
		reason = ReasonBalance
	case cur.EstimatedDays != nil && *cur.EstimatedDays <= cfg.DaysThreshold && !next.DaysAlerted:
		reason = ReasonDays
	}
	if reason == ReasonNone {
		return Decision{}, next
	}

	// 静默时段：不置位，退出静默后仍满足条件就补发（规则 3）。
	if InQuietHours(now, cfg.QuietStartHour, cfg.QuietEndHour) {
		return Decision{Reason: reason}, next
	}

	switch reason {
	case ReasonBalance:
		next.BalanceAlerted = true
	case ReasonDays:
		next.DaysAlerted = true
	}

	return Decision{
		Notify: true,
		Title:  "宿舍电费提醒",
		Body:   body(cur, cfg, reason),
		Reason: reason,
	}, next
}

// InQuietHours 判断 now 是否落在静默时段内。
//
// 支持跨午夜（默认 23:00–07:00）；start == end 表示没有静默时段。
// 用本地时区的钟点比较，避免把「晚上 23 点」算成 UTC 的 15 点。
func InQuietHours(now time.Time, startHour, endHour int) bool {
	startHour = ((startHour % 24) + 24) % 24
	endHour = ((endHour % 24) + 24) % 24
	if startHour == endHour {
		return false
	}
	hour := now.In(time.Local).Hour()
	if startHour < endHour {
		return hour >= startHour && hour < endHour
	}
	return hour >= startHour || hour < endHour
}

// body 组装通知正文：只放用户能行动的信息，不放 roomName（P2）。
func body(cur model.ElectricityStatus, cfg Config, reason Reason) string {
	unit := cur.Unit
	if unit == "" {
		unit = "度"
	}
	var head string
	switch reason {
	case ReasonDays:
		days := 0
		if cur.EstimatedDays != nil {
			days = *cur.EstimatedDays
		}
		head = fmt.Sprintf("剩余 %.2f %s，约可用 %d 天，已低于 %d 天阈值",
			cur.Balance, unit, days, cfg.DaysThreshold)
	default:
		head = fmt.Sprintf("剩余 %.2f %s，已低于 %.0f %s阈值",
			cur.Balance, unit, cfg.BalanceThreshold, unit)
	}
	return head + "\n数据截至 " + displayCollectedAt(cur.CollectedAt) + "（电表约 1 天延迟）"
}

// displayCollectedAt 把 RFC3339 转成人类可读的本地时间；解析不了就原样给出，
// 绝不因为格式问题把「数据截至」整行吞掉。
func displayCollectedAt(raw string) string {
	if raw == "" {
		return "未知"
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.In(time.Local).Format("2006-01-02 15:04")
	}
	return raw
}

// withDefaults 给零值配置补默认值：设置窗口首次保存前就可能进入轮询。
func (c Config) withDefaults() Config {
	if c.BalanceThreshold <= 0 {
		c.BalanceThreshold = DefaultBalanceThreshold
	}
	if c.DaysThreshold <= 0 {
		c.DaysThreshold = DefaultDaysThreshold
	}
	return c
}

// NormalizePollInterval 把用户配置的间隔钳制进 [600s, ∞)，并告知是否被钳制
// （docs/002 §5.2：低于下限钳制到 600s 并提示，不可放行）。
func NormalizePollInterval(d time.Duration) (time.Duration, bool) {
	if d <= 0 {
		return DefaultPollInterval, false
	}
	if d < PollIntervalFloor {
		return PollIntervalFloor, true
	}
	return d, false
}

// NextBackoff 计算连续失败后的下次重试间隔（docs/002 §6 规则 6）。
//
// 起点是轮询间隔，按次翻倍，上限 MaxBackoff；永远不小于成功路径的间隔，
// 避免失败时反而更频繁地打学校接口。
func NextBackoff(pollInterval time.Duration, failures int) time.Duration {
	base, _ := NormalizePollInterval(pollInterval)
	if failures <= 1 {
		return base
	}
	d := base
	for i := 1; i < failures; i++ {
		if d >= MaxBackoff/2 {
			return MaxBackoff
		}
		d *= 2
	}
	if d > MaxBackoff {
		d = MaxBackoff
	}
	return d
}

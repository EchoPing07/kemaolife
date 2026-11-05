package alert_test

import (
	"strings"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/alert"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// at 构造本地时区的测试时刻；用 time.Local 避免 CI（UTC）与开发机时区不同导致断言漂移。
func at(hour, minute int) time.Time {
	return time.Date(2026, 11, 5, hour, minute, 0, 0, time.Local)
}

// noon 是白天、非静默时段的默认测试时刻。
func noon() time.Time { return at(12, 0) }

func status(balance float64, days *int) model.ElectricityStatus {
	return model.ElectricityStatus{
		Balance:       balance,
		Unit:          "度",
		CollectedAt:   at(22, 41).Format(time.RFC3339),
		EstimatedDays: days,
	}
}

func days(n int) *int { return &n }

func defaultCfg() alert.Config {
	return alert.Config{
		BalanceThreshold: 10,
		DaysThreshold:    3,
		QuietStartHour:   23,
		QuietEndHour:     7,
	}
}

// 跌破阈值只提醒一次（docs/002 §6 规则 2）。
func TestEvaluateNotifiesOncePerThreshold(t *testing.T) {
	cfg := defaultCfg()
	var state alert.State

	dec, state := alert.Evaluate(state, status(9.5, nil), noon(), cfg)
	if !dec.Notify || dec.Reason != alert.ReasonBalance {
		t.Fatalf("首次跌破应提醒: %+v", dec)
	}
	if !strings.Contains(dec.Body, "数据截至") {
		t.Errorf("通知正文必须包含「数据截至」: %q", dec.Body)
	}

	// 同一阈值区间内再查，不重复提醒。
	dec, state = alert.Evaluate(state, status(8.0, nil), noon(), cfg)
	if dec.Notify {
		t.Error("同一阈值只应提醒一次")
	}

	// 充值回升到阈值以上后重置。
	dec, state = alert.Evaluate(state, status(50, nil), noon(), cfg)
	if dec.Notify {
		t.Error("回升到阈值以上不应提醒")
	}
	if state.BalanceAlerted {
		t.Error("回升后标记应被重置")
	}

	// 再次跌破可以再提醒一次。
	dec, _ = alert.Evaluate(state, status(9.9, nil), noon(), cfg)
	if !dec.Notify {
		t.Error("充值后再次跌破应可再提醒")
	}
}

// 余额与可用天数是两条独立通道，互不清除。
func TestEvaluateBalanceAndDaysAreIndependent(t *testing.T) {
	cfg := defaultCfg()
	cfg.BalanceThreshold = 1 // 余额远高于阈值，确保只走天数通道
	var state alert.State

	dec, state := alert.Evaluate(state, status(500, days(2)), noon(), cfg)
	if !dec.Notify || dec.Reason != alert.ReasonDays {
		t.Fatalf("应因可用天数触发: %+v", dec)
	}
	if !state.DaysAlerted || state.BalanceAlerted {
		t.Errorf("只应置位天数标记: %+v", state)
	}

	dec, state = alert.Evaluate(state, status(500, days(2)), noon(), cfg)
	if dec.Notify {
		t.Error("天数阈值也只应提醒一次")
	}

	// 天数回到阈值以上后，余额这条仍应正常触发（证明两条通道互不清除）。
	cfg.BalanceThreshold = 600
	dec, state = alert.Evaluate(state, status(500, nil), noon(), cfg)
	if !dec.Notify || dec.Reason != alert.ReasonBalance {
		t.Errorf("余额通道应独立触发: %+v", dec)
	}
	if !state.BalanceAlerted || !state.DaysAlerted {
		// 天数标记此时已被重置（EstimatedDays 为 nil），余额标记应新置位。
		t.Logf("state = %+v", state)
	}
	if !state.BalanceAlerted {
		t.Error("余额通道应已置位")
	}
}

// 静默时段内不通知，但状态不置位；退出静默后补发一次（规则 3）。
func TestEvaluateQuietHoursDefersNotification(t *testing.T) {
	cfg := defaultCfg()
	var state alert.State

	// 默认静默时段 23:00–07:00。
	dec, state := alert.Evaluate(state, status(5, nil), at(23, 30), cfg)
	if dec.Notify {
		t.Error("静默时段内不应发通知")
	}
	if state.BalanceAlerted {
		t.Error("静默时段内不应置位，否则退出静默后不会补发")
	}

	dec, state = alert.Evaluate(state, status(5, nil), at(7, 0), cfg)
	if !dec.Notify {
		t.Error("退出静默后仍满足条件应补发一次")
	}
	if !state.BalanceAlerted {
		t.Error("补发后应置位")
	}
}

// 静默时段跨午夜与「无静默」两种边界。
func TestInQuietHours(t *testing.T) {
	cases := []struct {
		start, end, hour int
		want             bool
	}{
		{23, 7, 23, true},
		{23, 7, 0, true},
		{23, 7, 6, true},
		{23, 7, 7, false},
		{23, 7, 22, false},
		{23, 7, 12, false},
		{13, 15, 14, true},  // 不跨午夜
		{13, 15, 16, false}, //
		{0, 0, 3, false},    // 起止相同 = 无静默
		{24, 7, 2, true},    // 越界钟点按 24 小时归一
	}
	for _, c := range cases {
		got := alert.InQuietHours(at(c.hour, 0), c.start, c.end)
		if got != c.want {
			t.Errorf("InQuietHours(hour=%d, %d-%d) = %v, want %v", c.hour, c.start, c.end, got, c.want)
		}
	}
}

// 余额充足且天数充足：不提醒。
func TestEvaluateNoAlertWhenHealthy(t *testing.T) {
	dec, state := alert.Evaluate(alert.State{}, status(100, days(30)), noon(), defaultCfg())
	if dec.Notify || state.BalanceAlerted || state.DaysAlerted {
		t.Errorf("健康状态不应提醒: %+v", dec)
	}
}

// estimatedDays 为 nil（数据不足）不能当 0 天触发提醒。
func TestEvaluateNilEstimatedDaysDoesNotAlert(t *testing.T) {
	cfg := defaultCfg()
	cfg.BalanceThreshold = 1 // 排除余额通道
	dec, _ := alert.Evaluate(alert.State{}, status(500, nil), noon(), cfg)
	if dec.Notify {
		t.Error("estimatedDays 为 nil 不应触发天数通道")
	}
}

// 轮询间隔下限：低于 600s 钳制到 600s 并返回「被钳制」标记（AGENTS §5.5）。
func TestNormalizePollInterval(t *testing.T) {
	if d, clamped := alert.NormalizePollInterval(60 * time.Second); d != alert.PollIntervalFloor || !clamped {
		t.Errorf("60s 应钳制到 %v 并标记, got %v/%v", alert.PollIntervalFloor, d, clamped)
	}
	if d, clamped := alert.NormalizePollInterval(0); d != alert.DefaultPollInterval || clamped {
		t.Errorf("0 应回落默认 %v, got %v", alert.DefaultPollInterval, d)
	}
	if d, clamped := alert.NormalizePollInterval(6 * time.Hour); d != 6*time.Hour || clamped {
		t.Errorf("6h 应原样保留, got %v", d)
	}
}

// 失败退避：起点是轮询间隔，单调不减，且有上限。
func TestNextBackoff(t *testing.T) {
	base := alert.DefaultPollInterval
	var prev time.Duration
	for i := 1; i <= 10; i++ {
		d := alert.NextBackoff(base, i)
		if d < base {
			t.Errorf("failures=%d 退避 %v 不应小于轮询间隔 %v", i, d, base)
		}
		if d < prev {
			t.Errorf("failures=%d 退避 %v 比上一次 %v 小", i, d, prev)
		}
		if d > alert.MaxBackoff {
			t.Errorf("failures=%d 退避 %v 超过上限 %v", i, d, alert.MaxBackoff)
		}
		prev = d
	}
}

// 用户传入低于下限的间隔时，退避也以钳制后的值为起点。
func TestNextBackoffUsesClampedBase(t *testing.T) {
	if d := alert.NextBackoff(time.Second, 1); d != alert.PollIntervalFloor {
		t.Errorf("退避起点应为钳制值 %v, got %v", alert.PollIntervalFloor, d)
	}
}

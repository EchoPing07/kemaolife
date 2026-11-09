package app

import (
	"context"
	"errors"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// ScheduleResult 是一次课表查询的完整结果：数据 + 快照状态。
//
// 把 stale / syncedAt 和 DTO 绑在一起，而不是让调用方自己拼：
// 「是否用了旧快照」是查询语义的一部分，不该由展示层猜。
type ScheduleResult struct {
	Day      model.DaySchedule
	Stale    bool      // true = 本次刷新失败，用的是旧快照
	SyncedAt time.Time // 数据实际来自哪次同步
}

// CacheOption 控制查询是否先联网刷新快照。
//
// 为什么需要它：CLI 的查询动作是「先刷新再计算」（docs/002 §1.3，D15 已定稿），
// 而 Android 每次打开、切周、选日期只要**直接读缓存**——联网刷新由 UI 另起的
// 独立 `timetable.sync` 负责，成功后再重新读缓存（docs/002 §5.3）。
// 两者是同一份快照、两条链路，不能混成一个「查询即刷新」。
type CacheOption int

const (
	// RefreshFirst 是 CLI 口径：先刷新一次，失败回落旧快照并标 Stale。
	RefreshFirst CacheOption = iota
	// CacheOnly 是 Android 口径：只读本地快照，0 请求，不要求 HTTP 端口。
	// 只有「从没同步过」才报 3001，快照损坏 / 版本不匹配仍保留 3003。
	CacheOnly
)

// validCacheOption 拒绝非法枚举值：裸 int 传入时 if mode == RefreshFirst 与
// if mode != CacheOnly 会给出不一致行为（docs/002 §2.1 规则 7）。
func validCacheOption(mode CacheOption) error {
	if mode != RefreshFirst && mode != CacheOnly {
		return model.NewError(model.CodeInvalidArgument, "非法的课表查询模式")
	}
	return nil
}

// GetSchedule 查询某一天课表。
//
// RefreshFirst 语义（docs/002 §1.3、D15）：
//   - 刷新成功 → 用最新快照计算，Stale=false，SyncedAt=本次同步时间；
//   - 刷新失败但有旧快照 → 用旧快照计算，Stale=true，SyncedAt=旧快照时间，
//     不返回错误（查询照常可用，让调用方通过 Stale 告知用户数据时间）；
//   - 刷新失败且无旧快照 → 返回错误，由调用方映射退出码。
//
// CacheOnly 语义（docs/002 §5.3）：不刷新，Stale 恒 false —
// 「旧快照」这个概念属于刷新失败，纯读缓存没有「旧」可言。
func GetSchedule(ctx context.Context, d Deps, cfg model.Config, date model.Date, mode CacheOption) (ScheduleResult, error) {
	if err := validCacheOption(mode); err != nil {
		return ScheduleResult{}, err
	}
	if err := ensureDeps(d, mode == RefreshFirst, true); err != nil {
		return ScheduleResult{}, err
	}

	var refreshErr error
	if mode == RefreshFirst {
		refreshErr = refreshSnapshot(ctx, d, cfg)
	}

	snap, loadErr := timetable.LoadSnapshot(d.Store)
	if loadErr != nil {
		// 没有快照可用：刷新成功却没快照是不可能的，按刷新错误报（更贴近原因）。
		if refreshErr != nil {
			return ScheduleResult{}, refreshErr
		}
		return ScheduleResult{}, cacheMiss(mode, loadErr)
	}

	return ScheduleResult{
		Day:      timetable.Query(snap, date),
		Stale:    refreshErr != nil,
		SyncedAt: snap.SyncedAt,
	}, nil
}

// cacheMiss 决定「没有快照」时报什么错（docs/002 §2.1 规则 7）。
//
// 只有哨兵 ErrSnapshotAbsent（存储里从来没写入过快照）才在 CacheOnly 下映射为
// 3001「请刷新一次」；损坏 / 版本不匹配 / 内容不完整 / IO 错误一律保留 3003，
// 让用户重新同步而不是被误导去改凭据。
func cacheMiss(mode CacheOption, loadErr error) error {
	if mode == CacheOnly && errors.Is(loadErr, timetable.ErrSnapshotAbsent) {
		return model.NewError(model.CodeCredentialMissing, "本地还没有课表，请先刷新一次")
	}
	return loadErr
}

// refreshSnapshot 尝试刷新快照，返回刷新是否失败。
//
// 缺教务凭据时不抛错：有旧快照时仍可查并标 stale（docs/002 §1.3）；
// 无旧快照时由调用方拿到 refreshErr 并映射为退出码 3。
func refreshSnapshot(ctx context.Context, d Deps, cfg model.Config) error {
	if cfg.StudentID == "" || cfg.EduPassword == "" {
		return model.NewError(model.CodeCredentialMissing, "未配置教务账号")
	}
	_, err := SyncTimetable(ctx, d, cfg)
	return err
}

// WeekResult 是某周查询结果。
type WeekResult struct {
	Week     int
	Days     []model.DaySchedule
	Stale    bool
	SyncedAt time.Time
}

// GetWeekSchedule 查询第 week 周（语义同 GetSchedule，含 CacheOnly）。
// week == 0 表示当周；其余值（含负数，用于学期开始前的周）都是显式周次。
// 当前日期不在学期内时返回「今天所在周」的 7 天 InSemester:false 成功结果（不产生 1002）。
func GetWeekSchedule(ctx context.Context, d Deps, cfg model.Config, week int, mode CacheOption) (WeekResult, error) {
	if err := validCacheOption(mode); err != nil {
		return WeekResult{}, err
	}
	if err := ensureDeps(d, mode == RefreshFirst, true); err != nil {
		return WeekResult{}, err
	}

	var refreshErr error
	if mode == RefreshFirst {
		refreshErr = refreshSnapshot(ctx, d, cfg)
	}

	snap, loadErr := timetable.LoadSnapshot(d.Store)
	if loadErr != nil {
		if refreshErr != nil {
			return WeekResult{}, refreshErr
		}
		return WeekResult{}, cacheMiss(mode, loadErr)
	}

	resolved := week
	outsideSemester := false
	if resolved == 0 {
		if w := timetable.CurrentWeek(snap, model.NewDate(d.Clock.Now())); w > 0 {
			resolved = w
		} else {
			// 假期 / 学期外：按「今天所在周」取 7 天并统一标 InSemester:false。
			// 必须用向下取整：Go 的 / 对负数向零截断，学期前会算出第 1 周而不是
			// 今天所在的那一周，把用户直接扔到开学第一周（docs/002 §1.1）。
			outsideSemester = true
			today := model.NewDate(d.Clock.Now())
			resolved = floorDiv(today.DaysSince(snap.Week1Monday), 7) + 1
		}
	}

	days := timetable.QueryWeek(snap, resolved)
	if outsideSemester {
		for i := range days {
			days[i].InSemester = false
			days[i].Entries = []model.Entry{}
		}
	}

	return WeekResult{
		Week:     resolved,
		Days:     days,
		Stale:    refreshErr != nil,
		SyncedAt: snap.SyncedAt,
	}, nil
}

// floorDiv 是数学意义上的整除（向负无穷取整），Go 的 / 对负数向零截断。
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

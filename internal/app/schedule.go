package app

import (
	"context"
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

// GetSchedule 查询某一天课表：**先刷新，再计算**（docs/002 §1.3、D15）。
//
// 语义：
//   - 刷新成功 → 用最新快照计算，Stale=false，SyncedAt=本次同步时间；
//   - 刷新失败但有旧快照 → 用旧快照计算，Stale=true，SyncedAt=旧快照时间，
//     不返回错误（查询照常可用，让调用方通过 Stale 告知用户数据时间）；
//   - 刷新失败且无旧快照 → 返回错误，由调用方映射退出码。
func GetSchedule(ctx context.Context, d Deps, cfg model.Config, date model.Date) (ScheduleResult, error) {
	if err := ensureDeps(d, true, true); err != nil {
		return ScheduleResult{}, err
	}

	refreshErr := refreshSnapshot(ctx, d, cfg)

	snap, loadErr := timetable.LoadSnapshot(d.Store)
	if loadErr != nil {
		// 没有快照可用：刷新成功却没快照是不可能的，按刷新错误报（更贴近原因）。
		if refreshErr != nil {
			return ScheduleResult{}, refreshErr
		}
		return ScheduleResult{}, loadErr
	}

	return ScheduleResult{
		Day:      timetable.Query(snap, date),
		Stale:    refreshErr != nil,
		SyncedAt: snap.SyncedAt,
	}, nil
}

// refreshSnapshot 尝试刷新快照，返回刷新是否失败。
//
// 缺教务凭据时不抛错：某些环境只需要读快照（例如只配了学号但密码过期），
// 让查询仍然可用并标 stale，而不是把「没配密码」变成硬失败。
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

// GetWeekSchedule 查询第 week 周（先刷新，语义同 GetSchedule）。
// week <= 0 表示当周；当周解析不出来时返回 1002。
func GetWeekSchedule(ctx context.Context, d Deps, cfg model.Config, week int) (WeekResult, error) {
	if err := ensureDeps(d, true, true); err != nil {
		return WeekResult{}, err
	}

	refreshErr := refreshSnapshot(ctx, d, cfg)

	snap, loadErr := timetable.LoadSnapshot(d.Store)
	if loadErr != nil {
		if refreshErr != nil {
			return WeekResult{}, refreshErr
		}
		return WeekResult{}, loadErr
	}

	resolved := week
	if resolved <= 0 {
		resolved = timetable.CurrentWeek(snap, model.NewDate(d.Clock.Now()))
		if resolved == 0 {
			return WeekResult{}, model.NewError(model.CodeNotInSemester,
				"当前日期不在本学期范围（"+timetable.SemesterRange(snap)+"）")
		}
	}

	return WeekResult{
		Week:     resolved,
		Days:     timetable.QueryWeek(snap, resolved),
		Stale:    refreshErr != nil,
		SyncedAt: snap.SyncedAt,
	}, nil
}

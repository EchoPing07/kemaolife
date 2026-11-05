package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// cmdTimetable 实现 timetable / sync / today / week / date（docs/002 §1.1）。
//
// 查询类子命令（today / week / date）每次都会先刷新快照：
// 刷新成功用最新数据，失败则回落旧快照并标 stale:true（docs/002 §1.3）。
func cmdTimetable(ctx context.Context, opts options, rt *session, args []string) error {
	sub := "today"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "sync":
		return timetableSync(ctx, opts, rt)
	case "today":
		return timetableDay(ctx, opts, rt, nowDate(rt))
	case "week":
		week := 0
		if len(args) > 1 {
			n, err := parseWeek(args[1])
			if err != nil {
				return err
			}
			week = n
		}
		return timetableWeek(ctx, opts, rt, week)
	case "date":
		if len(args) < 2 {
			return model.NewError(model.CodeInvalidArgument,
				"缺少日期：kemaolife timetable date <YYYY-MM-DD>")
		}
		date, err := model.ParseDate(args[1])
		if err != nil {
			return model.WrapError(model.CodeInvalidArgument, "日期无法解析", err)
		}
		return timetableDay(ctx, opts, rt, date)
	}
	return model.NewError(model.CodeInvalidArgument,
		"未知 timetable 子命令 "+quote(sub)+"，可用：sync、today、week、date")
}

// syncSchema 是 timetable sync 的 data（不含 Cookie）。
type syncSchema struct {
	SyncedAt    string `json:"syncedAt"`
	Week1Monday string `json:"week1Monday"`
	LastWeek    int    `json:"lastWeek"`
	LessonCount int    `json:"lessonCount"`
}

// timetableSync 只刷新、不查询；失败直接报错（不回落旧快照）。
func timetableSync(ctx context.Context, opts options, rt *session) error {
	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	result, err := app.SyncTimetable(ctx, rt.deps, cfg)
	if err != nil {
		return err
	}
	data := syncSchema(result)
	return envelope(opts, model.Envelope{
		OK:       true,
		Data:     data,
		TS:       formatNow(rt),
		SyncedAt: data.SyncedAt,
	}, func() {
		writeStdout("同步完成：" + data.SyncedAt + "\n")
		writeStdout(fmt.Sprintf("第 1 周周一 %s，最后一周 %d，课节 %d 条\n",
			data.Week1Monday, data.LastWeek, data.LessonCount))
	})
}

// timetableDay 输出单日课表；非学期内返回成功信封 + inSemester:false（退出码 0）。
//
// 查询前会先刷新快照；刷新失败但有旧快照时照常返回，信封带 stale:true。
func timetableDay(ctx context.Context, opts options, rt *session, date model.Date) error {
	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	result, err := app.GetSchedule(ctx, rt.deps, cfg, date)
	if err != nil {
		return err
	}

	env := model.Envelope{
		OK:       true,
		Data:     result.Day,
		TS:       formatNow(rt),
		Stale:    model.BoolPtr(result.Stale),
		SyncedAt: formatSyncedAt(result.SyncedAt),
	}
	return envelope(opts, env, func() {
		writeStdout(renderDay(result.Day, result.Stale, env.SyncedAt))
	})
}

// weekSchema 是 timetable week 的 data。
type weekSchema struct {
	Week int                 `json:"week"`
	Days []model.DaySchedule `json:"days"`
}

func timetableWeek(ctx context.Context, opts options, rt *session, week int) error {
	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	result, err := app.GetWeekSchedule(ctx, rt.deps, cfg, week)
	if err != nil {
		return err
	}
	data := weekSchema{Week: result.Week, Days: result.Days}
	env := model.Envelope{
		OK:       true,
		Data:     data,
		TS:       formatNow(rt),
		Stale:    model.BoolPtr(result.Stale),
		SyncedAt: formatSyncedAt(result.SyncedAt),
	}
	return envelope(opts, env, func() {
		writeStdout(renderWeek(data.Week, result.Days, result.Stale, result.SyncedAt))
	})
}

func renderDay(day model.DaySchedule, stale bool, syncedAt string) string {
	var b strings.Builder
	b.WriteString(day.Date.String() + "  星期" + day.WeekdayCn)
	if day.InSemester {
		b.WriteString(fmt.Sprintf("  第 %d 周", day.Week))
	} else {
		b.WriteString("  不在本学期范围")
	}
	b.WriteString("\n")
	b.WriteString(dataTimeLine(stale, syncedAt))

	if len(day.Entries) == 0 {
		b.WriteString("当天没课\n")
		return b.String()
	}
	rows := make([][]string, 0, len(day.Entries))
	for _, e := range day.Entries {
		rows = append(rows, []string{e.Periods, e.Time, e.Name, e.Location, e.Teacher})
	}
	b.WriteString(renderTable([]string{"节次", "时间", "课程", "地点", "教师"}, rows))
	return b.String()
}

func renderWeek(week int, days []model.DaySchedule, stale bool, syncedAt time.Time) string {
	var b strings.Builder
	semesterDays := 0
	for _, d := range days {
		if d.InSemester {
			semesterDays++
		}
	}
	b.WriteString(dataTimeLine(stale, formatSyncedAt(syncedAt)))
	if semesterDays == 0 {
		b.WriteString(fmt.Sprintf("第 %d 周不在本学期范围\n", week))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("第 %d 周\n", week))
	b.WriteString("\n")

	rows := make([][]string, 0, len(days))
	for _, d := range days {
		names := make([]string, 0, len(d.Entries))
		for _, e := range d.Entries {
			names = append(names, fmt.Sprintf("%s节 %s", e.Periods, e.Name))
		}
		summary := "没课"
		if len(names) > 0 {
			summary = strings.Join(names, "；")
		}
		rows = append(rows, []string{fmt.Sprintf("周%s", d.WeekdayCn), d.Date.String(), summary})
	}
	b.WriteString(renderTable([]string{"星期", "日期", "课程"}, rows))
	return b.String()
}

// parseWeek 解析周次参数；越界不报错（返回 inSemester:false），只校验正整数。
func parseWeek(raw string) (int, error) {
	n := 0
	if _, err := fmt.Sscan(strings.TrimSpace(raw), &n); err != nil || n < 1 {
		return 0, model.NewError(model.CodeInvalidArgument,
			"周次必须是大于 0 的整数，例如 kemaolife timetable week 3")
	}
	return n, nil
}

// dataTimeLine 输出数据时间提示。
//
// stale 时必须显式说明「刷新失败、用的是旧快照」：这是 agent 唯一能判断
// 「要不要告诉用户数据不是最新的」的依据（docs/002 §1.3）。
func dataTimeLine(stale bool, syncedAt string) string {
	if syncedAt == "" {
		return ""
	}
	if stale {
		return "（刷新失败，显示的是 " + syncedAt + " 的快照数据）\n"
	}
	return "（快照同步于 " + syncedAt + "）\n"
}

func nowDate(rt *session) model.Date { return model.NewDate(rt.deps.Clock.Now()) }

func formatSyncedAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format(time.RFC3339)
}

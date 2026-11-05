package timetable

import (
	"fmt"
	"sort"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Query 是纯函数：由快照与日期算出当天课表，0 请求、0 副作用（R1 验收）。
//
// 周次公式（api-docs/001 §1）：week = floor((目标日 − 第 1 周周一) / 7) + 1。
// 越界周次不报错，返回 InSemester:false 与空 entries（docs/002 §1.1）。
func Query(snap *Snapshot, date model.Date) model.DaySchedule {
	out := model.DaySchedule{
		Date:    date,
		Weekday: date.Weekday(),
		Entries: []model.Entry{},
	}
	if snap == nil || snap.Week1Monday.IsZero() {
		return out
	}

	out.WeekdayCn = WeekdayCN(out.Weekday)

	days := date.DaysSince(snap.Week1Monday)
	if days < 0 {
		out.Week = 1
		out.InSemester = false
		return out
	}
	out.Week = days/7 + 1

	lastWeek := snap.LastWeek
	if lastWeek < 1 {
		lastWeek = 1
	}
	out.InSemester = out.Week <= lastWeek
	if !out.InSemester {
		return out
	}

	entries := make([]model.Entry, 0, 4)
	for _, lesson := range snap.Lessons {
		if lesson.Weekday != out.Weekday || !containsInt(lesson.Weeks, out.Week) {
			continue
		}
		entries = append(entries, model.Entry{
			Name:      lesson.Name,
			Periods:   lesson.Periods,
			Time:      slotTimeOf(lesson.Periods),
			Teacher:   lesson.Teacher,
			Location:  lesson.Location,
			WeeksText: lesson.WeeksText,
		})
	}

	// 输出稳定可 diff（AGENTS §6）：先按起始节次，再按课程名，最后按地点。
	sort.SliceStable(entries, func(i, j int) bool {
		pi, pj := SlotIndex(entries[i].Periods), SlotIndex(entries[j].Periods)
		if pi != pj {
			return pi < pj
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Location < entries[j].Location
	})
	out.Entries = entries
	return out
}

// QueryRange 返回 [from, to] 的逐日课表（含首含尾）。
//
// 只供 core 内部「当周 7 日」计算使用，不暴露为 CLI 命令（docs/002 §1.1）。
func QueryRange(snap *Snapshot, from, to model.Date) []model.DaySchedule {
	if from.After(to) {
		return nil
	}
	n := -from.DaysSince(to)
	out := make([]model.DaySchedule, 0, n+1)
	for i := 0; i <= n; i++ {
		out = append(out, Query(snap, from.AddDays(i)))
	}
	return out
}

// QueryWeek 返回第 week 周的 7 天课表；week 越界时 7 天全部 InSemester:false。
func QueryWeek(snap *Snapshot, week int) []model.DaySchedule {
	if snap == nil || snap.Week1Monday.IsZero() {
		return nil
	}
	monday := snap.Week1Monday.AddDays(7 * (week - 1))
	return QueryRange(snap, monday, monday.AddDays(6))
}

// CurrentWeek 由快照与「今天」算出当前周次；非学期内返回 0。
func CurrentWeek(snap *Snapshot, today model.Date) int {
	if snap == nil || snap.Week1Monday.IsZero() {
		return 0
	}
	days := today.DaysSince(snap.Week1Monday)
	if days < 0 {
		return 0
	}
	week := days/7 + 1
	if snap.LastWeek < 1 || week > snap.LastWeek {
		return 0
	}
	return week
}

// SemesterRange 返回学期起止日（第 1 周周一 ~ 最后一周周日），用于错误提示。
func SemesterRange(snap *Snapshot) string {
	if snap == nil || snap.Week1Monday.IsZero() {
		return ""
	}
	last := snap.LastWeek
	if last < 1 {
		last = 1
	}
	end := snap.Week1Monday.AddDays(last*7 - 1)
	return fmt.Sprintf("%s ~ %s", snap.Week1Monday, end)
}

func containsInt(list []int, want int) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

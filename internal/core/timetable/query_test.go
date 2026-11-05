package timetable

import (
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// 固定锚点：第 1 周周一 = 2026-08-31。
func testSnapshot() *Snapshot {
	return &Snapshot{
		Version:     SnapshotVersion,
		Week1Monday: model.Date{Year: 2026, Month: 8, Day: 31},
		Week1Dates: []model.Date{
			{Year: 2026, Month: 8, Day: 31},
			{Year: 2026, Month: 9, Day: 1},
			{Year: 2026, Month: 9, Day: 2},
			{Year: 2026, Month: 9, Day: 3},
			{Year: 2026, Month: 9, Day: 4},
			{Year: 2026, Month: 9, Day: 5},
			{Year: 2026, Month: 9, Day: 6},
		},
		SlotTimes: []string{
			"08:30~10:00", "10:20~11:50", "14:00~15:20",
			"15:30~16:50", "18:30~19:50", "20:00~21:20",
		},
		LastWeek: 16,
		Lessons: []Lesson{
			{Name: "高等数学", Periods: "1,2", Weekday: 4, Weeks: seq(1, 16),
				Teacher: "教师甲", Location: "教室A", WeeksText: "1-16周"},
			{Name: "大学英语", Periods: "3,4", Weekday: 4, Weeks: seq(1, 8),
				Teacher: "教师乙", Location: "教室B", WeeksText: "1-8周"},
			{Name: "程序设计", Periods: "1,2", Weekday: 1, Weeks: seq(1, 16),
				Teacher: "教师丙", Location: "机房C", WeeksText: "1-16周"},
		},
	}
}

func TestQueryWeekCalculation(t *testing.T) {
	snap := testSnapshot()
	// 2026-11-05 是第 10 周的周四。
	day := Query(snap, model.Date{Year: 2026, Month: 11, Day: 5})
	if day.Week != 10 {
		t.Errorf("week = %d, want 10", day.Week)
	}
	if day.Weekday != 4 {
		t.Errorf("weekday = %d, want 4", day.Weekday)
	}
	if day.WeekdayCn != "四" {
		t.Errorf("weekdayCn = %q", day.WeekdayCn)
	}
	if !day.InSemester {
		t.Error("应在学期内")
	}
	// 第 10 周只有「高等数学」（英语只到第 8 周）。
	if len(day.Entries) != 1 || day.Entries[0].Name != "高等数学" {
		t.Fatalf("entries = %+v", day.Entries)
	}
	if day.Entries[0].Time != "08:30~10:00" {
		t.Errorf("time = %q", day.Entries[0].Time)
	}
}

func TestQueryWeek1MondayIsWeekOne(t *testing.T) {
	snap := testSnapshot()
	day := Query(snap, snap.Week1Monday)
	if day.Week != 1 {
		t.Errorf("第 1 周周一 week = %d, want 1", day.Week)
	}
	if day.Weekday != 1 {
		t.Errorf("weekday = %d, want 1", day.Weekday)
	}
	if len(day.Entries) != 1 || day.Entries[0].Name != "程序设计" {
		t.Errorf("entries = %+v", day.Entries)
	}
}

// 早于第 1 周周一：不在学期内，不能算出 0 或负周次。
func TestQueryBeforeSemester(t *testing.T) {
	snap := testSnapshot()
	day := Query(snap, model.Date{Year: 2026, Month: 8, Day: 24})
	if day.InSemester {
		t.Error("不应在学期内")
	}
	if day.Week != 1 {
		t.Errorf("week = %d, want 1（归一化）", day.Week)
	}
	if len(day.Entries) != 0 {
		t.Errorf("entries 应为空: %+v", day.Entries)
	}
}

// 越界周次返回 inSemester:false 而非报错（docs/002 §1.1）。
func TestQueryBeyondLastWeek(t *testing.T) {
	snap := testSnapshot()
	monday := snap.Week1Monday.AddDays(7 * 20) // 第 21 周
	day := Query(snap, monday)
	if day.InSemester {
		t.Error("第 21 周不应在学期内")
	}
	if day.Week != 21 {
		t.Errorf("week = %d, want 21", day.Week)
	}
	if len(day.Entries) != 0 {
		t.Errorf("越界周次 entries 应为空: %+v", day.Entries)
	}
}

// 输出稳定：同一节次多门课按课程名排序。
func TestQueryEntriesSorted(t *testing.T) {
	snap := testSnapshot()
	snap.Lessons = []Lesson{
		{Name: "B课程", Periods: "3,4", Weekday: 2, Weeks: seq(1, 16), WeeksText: "1-16周"},
		{Name: "A课程", Periods: "3,4", Weekday: 2, Weeks: seq(1, 16), WeeksText: "1-16周"},
		{Name: "早课", Periods: "1,2", Weekday: 2, Weeks: seq(1, 16), WeeksText: "1-16周"},
	}
	day := Query(snap, snap.Week1Monday.AddDays(1)) // 周二
	if len(day.Entries) != 3 {
		t.Fatalf("entries = %+v", day.Entries)
	}
	if day.Entries[0].Name != "早课" {
		t.Errorf("起始节次升序失败: %v", day.Entries)
	}
	if day.Entries[1].Name != "A课程" || day.Entries[2].Name != "B课程" {
		t.Errorf("同节次未按课程名排序: %v", day.Entries)
	}
}

func TestQueryEmptyDayReturnsEmptySlice(t *testing.T) {
	snap := testSnapshot()
	day := Query(snap, snap.Week1Monday.AddDays(5)) // 周六没课
	if day.Entries == nil {
		t.Fatal("entries 必须是空数组而不是 nil（JSON 要输出 []）")
	}
	if len(day.Entries) != 0 {
		t.Errorf("entries = %+v", day.Entries)
	}
}

func TestQueryNilSnapshot(t *testing.T) {
	day := Query(nil, model.Date{Year: 2026, Month: 11, Day: 5})
	if day.InSemester || len(day.Entries) != 0 {
		t.Errorf("nil 快照应返回空结果: %+v", day)
	}
}

func TestQueryRangeInclusive(t *testing.T) {
	snap := testSnapshot()
	from := snap.Week1Monday
	to := from.AddDays(6)
	days := QueryRange(snap, from, to)
	if len(days) != 7 {
		t.Fatalf("QueryRange 返回 %d 天，want 7", len(days))
	}
	if days[0].Date != from || days[6].Date != to {
		t.Errorf("范围端点错误: %s .. %s", days[0].Date, days[6].Date)
	}
	// 反向区间返回 nil，不 panic。
	if got := QueryRange(snap, to, from); got != nil {
		t.Errorf("反向区间应返回 nil, got %v", got)
	}
}

func TestQueryWeek(t *testing.T) {
	snap := testSnapshot()
	days := QueryWeek(snap, 3)
	if len(days) != 7 {
		t.Fatalf("QueryWeek 返回 %d 天", len(days))
	}
	wantStart := snap.Week1Monday.AddDays(14)
	if days[0].Date != wantStart {
		t.Errorf("第 3 周周一 = %s, want %s", days[0].Date, wantStart)
	}
	if days[0].Week != 3 {
		t.Errorf("week = %d, want 3", days[0].Week)
	}
}

func TestCurrentWeek(t *testing.T) {
	snap := testSnapshot()
	if got := CurrentWeek(snap, model.Date{Year: 2026, Month: 11, Day: 5}); got != 10 {
		t.Errorf("CurrentWeek = %d, want 10", got)
	}
	// 假期 / 开学前返回 0。
	if got := CurrentWeek(snap, model.Date{Year: 2026, Month: 7, Day: 1}); got != 0 {
		t.Errorf("开学前 CurrentWeek = %d, want 0", got)
	}
	if got := CurrentWeek(snap, model.Date{Year: 2027, Month: 3, Day: 1}); got != 0 {
		t.Errorf("学期后 CurrentWeek = %d, want 0", got)
	}
	if got := CurrentWeek(nil, model.Date{Year: 2026, Month: 11, Day: 5}); got != 0 {
		t.Errorf("nil 快照 CurrentWeek = %d, want 0", got)
	}
}

// 跨学期也要能算：只要锚点换了，周次就从新锚点重算。
func TestQueryCrossSemester(t *testing.T) {
	snap := testSnapshot()
	snap.Week1Monday = model.Date{Year: 2027, Month: 2, Day: 22}
	snap.LastWeek = 18
	day := Query(snap, model.Date{Year: 2027, Month: 3, Day: 1})
	if day.Week != 2 {
		t.Errorf("新学期第 2 周 = %d, want 2", day.Week)
	}
	if !day.InSemester {
		t.Error("应在学期内")
	}
}

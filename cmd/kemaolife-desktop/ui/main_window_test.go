//go:build windows

package ui

import (
	"reflect"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

func TestMainEntryRuns(t *testing.T) {
	got := mainEntryRuns(model.Entry{Periods: "1-2、5、7—9"})
	want := [][2]int{{1, 2}, {5, 5}, {7, 9}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mainEntryRuns() = %v, want %v", got, want)
	}
}

func TestCourseBlocksSpanAndSplitCollisions(t *testing.T) {
	w := &mainWindow{state: MainWindowState{Days: []model.DaySchedule{{Entries: []model.Entry{
		{Name: "A", Periods: "1-3"},
		{Name: "B", Periods: "2-4"},
		{Name: "C", Periods: "4"},
		{Name: "D", Periods: "5-6"},
	}}}}}
	got := w.courseBlocks()
	if len(got) != 4 {
		t.Fatalf("course count = %d, want 4", len(got))
	}
	for i := 0; i < 3; i++ {
		if got[i].columns != 2 {
			t.Errorf("block %s columns = %d, want 2", got[i].entry.Name, got[i].columns)
		}
	}
	if got[0].column == got[1].column {
		t.Error("overlapping A and B must use different columns")
	}
	if got[0].column != got[2].column {
		t.Error("non-overlapping A and C should reuse a column")
	}
	if got[3].columns != 1 || got[3].column != 0 {
		t.Errorf("independent D must use the full column: %+v", got[3])
	}
}

func TestCloneMainStateDoesNotAliasSlices(t *testing.T) {
	avg, days := 2.5, 4
	s := MainWindowState{
		Today: model.DaySchedule{Entries: []model.Entry{{Name: "A"}}},
		Days:  []model.DaySchedule{{Entries: []model.Entry{{Name: "B"}}}},
		Report: model.ElectricityReport{Status: model.ElectricityStatus{
			DailyAvg: &avg, EstimatedDays: &days,
		}, Usage: model.ElectricityUsage{Days: []model.DayUsage{{Use: 1}}}},
	}
	clone := cloneMainState(s)
	s.Today.Entries[0].Name = "changed"
	s.Days[0].Entries[0].Name = "changed"
	s.Report.Usage.Days[0].Use = 99
	avg, days = 0, 0
	if clone.Today.Entries[0].Name != "A" || clone.Days[0].Entries[0].Name != "B" ||
		clone.Report.Usage.Days[0].Use != 1 || *clone.Report.Status.DailyAvg != 2.5 ||
		*clone.Report.Status.EstimatedDays != 4 {
		t.Fatal("MainWindowState must own its DTO slices and optional values")
	}
}

// 「上一周」在周次 ≤1 时不得触发回调：0 是「当周」语义，传下去会跳回当前周。
func TestTimetablePrevClampsAtWeekOne(t *testing.T) {
	for _, tc := range []struct {
		week int
		want []int
	}{
		{week: 0, want: nil},
		{week: 1, want: nil},
		{week: 8, want: []int{7}},
	} {
		var calls []int
		w := &mainWindow{
			state:   MainWindowState{Week: tc.week},
			handler: MainWindowHandler{OnTimetableWeek: func(n int) { calls = append(calls, n) }},
		}
		w.command(IDMainTimetablePrev)
		if len(calls) != len(tc.want) {
			t.Fatalf("week=%d: 回调次数 %d, want %d", tc.week, len(calls), len(tc.want))
		}
		for i, n := range calls {
			if n != tc.want[i] {
				t.Fatalf("week=%d: 回调周次 %d, want %d", tc.week, n, tc.want[i])
			}
		}
	}
}

//go:build windows

package ui

import (
	"reflect"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/desktop"
	myui "github.com/egoist/mygo/ui"
)

func testMainWindow(page int) *mainWindow {
	w := &mainWindow{
		page: page,
		state: MainWindowState{
			TodayDate: "2026-11-18",
			HasToday:  true,
			Today: model.DaySchedule{
				Date: model.Date{Year: 2026, Month: 11, Day: 18}, Week: 12, WeekdayCn: "三", InSemester: true,
				Entries: []model.Entry{{Name: "示例课程", Periods: "1-2", Time: "08:30~10:00", Location: "示例地点", Teacher: "示例教师"}},
			},
			Week: 12, LastWeek: 20,
			Days: []model.DaySchedule{{Date: model.Date{Year: 2026, Month: 11, Day: 16}, WeekdayCn: "一", InSemester: true,
				Entries: []model.Entry{{Name: "示例课程", Periods: "1-2", Time: "08:30~10:00", Location: "示例地点"}}}},
			HasReport: true,
			Report: model.ElectricityReport{
				Status: model.ElectricityStatus{Balance: 20, Unit: "度", CollectedAt: "2026-11-17T08:00:00+08:00"},
				Usage:  model.ElectricityUsage{YearMonth: "2026-11", MonthUse: 12.5, DataUntil: "2026-11-17", Days: []model.DayUsage{{Date: "2026-11-17", Use: 1.5}}},
			},
			Campus: model.CampusStatus{State: model.CampusOnline, CheckedAt: "2026-11-18T08:00:00+08:00"},
		},
		month: "2026-11",
	}
	w.state.SettingsForm = desktop.ToSettingsForm(desktop.DefaultSettings())
	w.state.SettingsForm.StudentID = "<学号>"
	w.setSettingsForm(w.state.SettingsForm)
	return w
}

func TestTodayPageRendersDTO(t *testing.T) {
	w := testMainWindow(mainPageToday)
	tt := myui.NewTester(w.view, 1180, 760)
	for _, text := range []string{"今天", "示例课程", "20.00", "已在线", "08:30~10:00"} {
		if !tt.HasText(text) {
			t.Errorf("今天页缺少 %q；当前文本：%q", text, tt.Texts())
		}
	}
}

func TestElectricityAndCampusPagesRender(t *testing.T) {
	for _, tc := range []struct {
		page int
		want []string
	}{
		{mainPageElectricity, []string{"电费摘要", "20.00", "12.50", "2026-11-17", "数据截至"}},
		{mainPageCampus, []string{"已在线", "登录校园网", "重登", "检查时间："}},
	} {
		w := testMainWindow(tc.page)
		tt := myui.NewTester(w.view, 1180, 760)
		for _, text := range tc.want {
			if !tt.HasText(text) {
				t.Errorf("page=%d 缺少 %q；当前文本：%q", tc.page, text, tt.Texts())
			}
		}
	}
}

func TestSettingsFailedSaveKeepsDraft(t *testing.T) {
	w := testMainWindow(mainPageSettings)
	w.settings.StudentID = "<新学号>"
	w.settings.EduPassword = "<密码>"
	w.settingsDirty = true
	w.handler.OnSaveSettings = func(desktop.SettingsResult) (string, desktop.SettingsForm, bool) {
		return "输入无效", w.state.SettingsForm, false
	}
	tt := myui.NewTester(w.view, 1180, 760)
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if w.settings.StudentID != "<新学号>" || w.settings.EduPassword != "<密码>" {
		t.Fatal("保存失败不得覆盖当前编辑内容")
	}
	if w.settingsHint != "输入无效" || !w.settingsDirty {
		t.Fatalf("保存失败状态错误：hint=%q dirty=%v", w.settingsHint, w.settingsDirty)
	}
}

func TestSettingsBackgroundUpdateKeepsDraft(t *testing.T) {
	w := testMainWindow(mainPageSettings)
	w.settings.StudentID = "<编辑中的学号>"
	w.settingsDirty = true
	w.state.SettingsForm.StudentID = "<学号>"
	w.state.SettingsForm.PollSeconds = 7200
	w.syncStateForView()
	if w.settings.StudentID != "<编辑中的学号>" || w.settings.PollSeconds != "14400" {
		t.Fatalf("后台更新覆盖了编辑中的表单：%+v", w.settings)
	}
	w.settingsDirty = false
	w.syncStateForView()
	if w.settings.PollSeconds != "7200" {
		t.Fatalf("非编辑状态应回显生效配置：%q", w.settings.PollSeconds)
	}
}

func TestSettingsSuccessfulSaveClearsPasswords(t *testing.T) {
	w := testMainWindow(mainPageSettings)
	w.settings.EduPassword = "<密码>"
	w.settings.CampusPassword = "<校园网密码>"
	w.settingsDirty = true
	form := w.state.SettingsForm
	form.PollSeconds = 600
	w.handler.OnSaveSettings = func(desktop.SettingsResult) (string, desktop.SettingsForm, bool) {
		return "低于下限，已按 600 秒执行", form, true
	}
	tt := myui.NewTester(w.view, 1180, 760)
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if w.settings.EduPassword != "" || w.settings.CampusPassword != "" {
		t.Fatal("保存成功后不得继续回显密码")
	}
	if w.settings.PollSeconds != "600" || w.settingsDirty {
		t.Fatalf("未回显实际生效值：poll=%q dirty=%v", w.settings.PollSeconds, w.settingsDirty)
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
		t.Fatal("MainWindowState 必须持有 DTO 切片与可选值的副本")
	}
}

func TestDarkThemePagesRender(t *testing.T) {
	for _, page := range []int{mainPageToday, mainPageTimetable, mainPageElectricity, mainPageCampus, mainPageSettings} {
		w := testMainWindow(page)
		tt := myui.NewTester(w.view, 960, 640)
		tt.SetDark(true)
		if !mainPaletteForTest(tt) {
			t.Fatalf("page=%d 深色主题未完成渲染", page)
		}
	}
}

func mainPaletteForTest(tt *myui.Tester) bool {
	return tt.HasText("科贸生活") && tt.Image() != nil
}

func TestMainEntryRuns(t *testing.T) {
	got := mainEntryRuns(model.Entry{Periods: "1-2,5,7-9"})
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
		t.Error("重叠课程必须分列")
	}
	if got[0].column != got[2].column {
		t.Error("不重叠课程应复用列")
	}
	if got[3].columns != 1 || got[3].column != 0 {
		t.Errorf("独立课程应占满整列: %+v", got[3])
	}
}

func TestTimetableCourseKeysUniqueAcrossDays(t *testing.T) {
	w := testMainWindow(mainPageTimetable)
	w.state.Days = []model.DaySchedule{
		{WeekdayCn: "一", Entries: []model.Entry{{Name: "第十一节课程", Periods: "11-12"}}},
		{WeekdayCn: "二", Entries: []model.Entry{{Name: "第一节课程", Periods: "1-2"}}},
	}
	tt := myui.NewTester(w.view, 1180, 760)
	if !tt.HasText("第十一节课程") || !tt.HasText("第一节课程") {
		t.Fatalf("跨天课程未正确渲染：%q", tt.Texts())
	}
}

func TestTimetableWeekActions(t *testing.T) {
	w := testMainWindow(mainPageTimetable)
	var weeks []int
	w.handler.OnTimetableWeek = func(week int) { weeks = append(weeks, week) }
	tt := myui.NewTester(w.view, 1180, 760)
	if !tt.HasText("示例课程") || !tt.HasText("第 12 周 / 共 20 周") {
		t.Fatalf("课表页未展示周次与课程：%q", tt.Texts())
	}
	if err := tt.Click("上一周"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("当周"); err != nil {
		t.Fatal(err)
	}
	if len(weeks) != 2 || weeks[0] != 11 || weeks[1] != 0 {
		t.Fatalf("切周回调 = %v，want [11 0]", weeks)
	}
}

package timetable

import (
	"strings"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// exportSnapshot 构造一份覆盖各种周次 / 节次边界的快照，用于导出测试。
//
// 不复用 query_test.go 的 testSnapshot：那份固定锚点服务于周次计算，
// 导出要覆盖的是「周次写法 / 节次拆分 / 空值占位」这些格式边界。
func exportSnapshot() *Snapshot {
	return &Snapshot{
		Version:     SnapshotVersion,
		SyncedAt:    time.Date(2026, 11, 9, 21, 10, 2, 0, time.Local),
		Week1Monday: model.Date{Year: 2026, Month: 8, Day: 31},
		SlotTimes:   []string{"08:30~10:00", "10:20~11:50", "14:00~15:20"},
		LastWeek:    16,
		Lessons: []Lesson{
			// 连续周 + 连续节次。
			{Name: "高等数学", Periods: "1-2", Weekday: 1, Weeks: seq(1, 5),
				Teacher: "张三", Location: "逸夫楼101", WeeksText: "1-5周"},
			// 间断周（连续段 + 隔周段）：官方示范里的 `1-5、7-11单`。
			{Name: "大学计算机", Periods: "1-2", Weekday: 5, Weeks: append(seq(1, 5), 7, 9, 11),
				Teacher: "", Location: "", WeeksText: "1-5周"},
			// 不连续节次必须拆行，不能压成 1-3。
			{Name: "有机化学", Periods: "1,3", Weekday: 4, Weeks: seq(1, 5),
				Teacher: "王五", Location: "逸夫楼108", WeeksText: "1-5周"},
			// 含逗号的课程名 / 地点要转义，否则列会错位。
			{Name: "创新,实践", Periods: "3-4", Weekday: 2, Weeks: seq(2, 4),
				Teacher: "李四", Location: "实验楼,A101", WeeksText: "2-4周"},
			// 重复行（多时段解析出的同一节）只占一格。
			{Name: "高等数学", Periods: "1,2", Weekday: 1, Weeks: seq(1, 5),
				Teacher: "张三", Location: "逸夫楼101", WeeksText: "1-5周"},
		},
	}
}

func TestExportWakeUpCSVMatchOfficialTemplateShape(t *testing.T) {
	csvText, err := ExportWakeUpCSV(exportSnapshot())
	if err != nil {
		t.Fatalf("ExportWakeUpCSV: %v", err)
	}

	// 官方模板带 UTF-8 BOM + CRLF（WPS 打开才不乱码、不串行）。
	if !strings.HasPrefix(csvText, wakeUpCSVBOM+wakeUpCSVHeader+"\r\n") {
		t.Fatalf("表头不符合官方模板: %q", firstLine(csvText))
	}
	if !strings.HasSuffix(csvText, "\r\n") {
		t.Error("最后一行应以 CRLF 结束")
	}

	lines := splitCRLF(csvText)
	want := []string{
		"高等数学,1,1,2,张三,逸夫楼101,1-5",
		// 含逗号的课程名 / 地点要加引号，否则列会错位。
		`"创新,实践",2,3,4,李四,"实验楼,A101",2-4`,
		// 不连续节次（1,3）拆成两行，不能压成 1-3 凭空多出第 2 节。
		"有机化学,4,1,1,王五,逸夫楼108,1-5",
		"有机化学,4,3,3,王五,逸夫楼108,1-5",
		// 老师 / 地点为空必须写「无」，否则 WakeUp 拒绝导入；
		// 周次是官方示范里的 `1-5、7-11单` 写法。
		"大学计算机,5,1,2,无,无,1-5、7-11单",
	}
	got := lines[1:] // 去掉表头
	if len(got) != len(want) {
		t.Fatalf("行数 = %d, want %d\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 行 = %q, want %q", i+1, got[i], want[i])
		}
	}
}

// 间断周必须用官方顿号写法，单双周带「单 / 双」。
func TestWakeUpWeeksFormatting(t *testing.T) {
	cases := []struct {
		weeks []int
		want  string
	}{
		{seq(1, 16), "1-16"},
		{append(seq(1, 5), 7, 8, 9, 10, 11), "1-5、7-11"},
		{[]int{7, 9, 11}, "7-11单"},
		{[]int{12, 14, 16}, "12-16双"},
		{[]int{1, 3}, "1、3"},
		{[]int{8}, "8"},
		{[]int{1, 2, 3, 5, 7}, "1-3、5、7"},
	}
	for _, c := range cases {
		if got := wakeUpWeeks(c.weeks); got != c.want {
			t.Errorf("wakeUpWeeks(%v) = %q, want %q", c.weeks, got, c.want)
		}
	}
}

// 导出必须稳定可 diff：同一个快照两次导出逐字节一致，且不依赖 Lessons 原顺序。
func TestExportWakeUpCSVIsStable(t *testing.T) {
	snap := exportSnapshot()
	first, err := ExportWakeUpCSV(snap)
	if err != nil {
		t.Fatal(err)
	}

	// 打乱课时顺序后仍应得到同一份文件（排序是显式的）。
	reversed := make([]Lesson, 0, len(snap.Lessons))
	for i := len(snap.Lessons) - 1; i >= 0; i-- {
		reversed = append(reversed, snap.Lessons[i])
	}
	snap.Lessons = reversed
	second, err := ExportWakeUpCSV(snap)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("导出结果不稳定：\n--- first ---\n%q\n--- second ---\n%q", first, second)
	}
}

// 空快照 / 全空课表要明确报错，不能导出只有表头的文件骗用户「导出成功」。
func TestExportWakeUpCSVRejectsEmpty(t *testing.T) {
	if _, err := ExportWakeUpCSV(nil); err == nil {
		t.Error("nil 快照应报错")
	}
	snap := exportSnapshot()
	snap.Lessons = nil
	if _, err := ExportWakeUpCSV(snap); err == nil {
		t.Error("没有课程时应报错")
	}
	// 周次为空 / 星期非法的课节跳过；全部跳过同样报错。
	snap.Lessons = []Lesson{{Name: "幽灵课", Periods: "1-2", Weekday: 0, Weeks: seq(1, 5)}}
	if _, err := ExportWakeUpCSV(snap); err == nil {
		t.Error("无有效课节时应报错")
	}
}

func TestExportFilename(t *testing.T) {
	got := ExportFilename(time.Date(2026, 11, 9, 21, 10, 2, 0, time.Local))
	if got != "kemaolife-timetable-20261109-2110.csv" {
		t.Errorf("filename = %q", got)
	}
	if ExportFilename(time.Time{}) != "kemaolife-timetable.csv" {
		t.Errorf("零值时间应退化为无时间戳文件名")
	}
}

func firstLine(s string) string {
	if i := strings.Index(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func splitCRLF(s string) []string {
	s = strings.TrimSuffix(s, wakeUpCSVLineEnd)
	return strings.Split(strings.TrimPrefix(s, wakeUpCSVBOM), wakeUpCSVLineEnd)
}

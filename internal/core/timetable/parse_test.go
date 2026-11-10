package timetable

import (
	"fmt"
	"testing"
)

func TestParseSktimeSingle(t *testing.T) {
	slots, err := ParseSktime("周三第1、2节{2,4,7-10}全周")
	if err != nil {
		t.Fatalf("ParseSktime: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots = %+v", slots)
	}
	if slots[0].Weekday != 3 {
		t.Errorf("weekday = %d, want 3", slots[0].Weekday)
	}
	// 连续节号压缩成区间，便于阅读与 diff。
	if slots[0].Periods != "1-2" {
		t.Errorf("periods = %q, want 1-2", slots[0].Periods)
	}
	want := []int{2, 4, 7, 8, 9, 10}
	if !equalInts(slots[0].Weeks, want) {
		t.Errorf("weeks = %v, want %v", slots[0].Weeks, want)
	}
}

func TestParseSktimeMultipleSlots(t *testing.T) {
	raw := "周三第3、4节{19}全周;周三第3、4节{2-4,7-8,10}全周;周四第5、6节{9}全周"
	slots, err := ParseSktime(raw)
	if err != nil {
		t.Fatalf("ParseSktime: %v", err)
	}
	if len(slots) != 3 {
		t.Fatalf("slots = %d, want 3", len(slots))
	}
	if slots[2].Weekday != 4 {
		t.Errorf("第三段 weekday = %d, want 4", slots[2].Weekday)
	}
}

func TestParseSktimeRangePeriods(t *testing.T) {
	// `第5-8节` 这种连续写法要保留成 5-8。
	slots, err := ParseSktime("周二第5-8节{19}全周")
	if err != nil {
		t.Fatal(err)
	}
	if slots[0].Periods != "5-8" {
		t.Errorf("periods = %q, want 5-8", slots[0].Periods)
	}
	if got := SlotIndex(slots[0].Periods); got != 5 {
		t.Errorf("SlotIndex = %d, want 5", got)
	}
}

func TestParseSktimeRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"", "   ", "周三", "第1节{1}全周", "周三第1节", "周三第1节{}全周"} {
		if _, err := ParseSktime(raw); err == nil {
			t.Errorf("ParseSktime(%q) 应当报错", raw)
		}
	}
}

func TestExpandWeeks(t *testing.T) {
	cases := []struct {
		body   string
		parity string
		want   []int
	}{
		{"1-16", "", seq(1, 16)},
		{"2,4,7-10", "", []int{2, 4, 7, 8, 9, 10}},
		{"19", "", []int{19}},
		{"2-4,7-8,10", "", []int{2, 3, 4, 7, 8, 10}},
		{"1-8", "单", []int{1, 3, 5, 7}},
		{"1-8", "双", []int{2, 4, 6, 8}},
		{"1,3,5", "", []int{1, 3, 5}},
	}
	for _, c := range cases {
		got, err := ExpandWeeks(c.body, c.parity)
		if err != nil {
			t.Errorf("ExpandWeeks(%q,%q): %v", c.body, c.parity, err)
			continue
		}
		if !equalInts(got, c.want) {
			t.Errorf("ExpandWeeks(%q,%q) = %v, want %v", c.body, c.parity, got, c.want)
		}
	}
}

func TestExpandWeeksRejectsGarbage(t *testing.T) {
	for _, body := range []string{"", "  ", "abc", "0", "1-200"} {
		if _, err := ExpandWeeks(body, ""); err == nil {
			t.Errorf("ExpandWeeks(%q) 应当报错", body)
		}
	}
}

// 小节号 → 大节序号的换算（6 个大节 / 12 个小节）。
func TestSlotTimeOf(t *testing.T) {
	cases := map[string]string{
		"1-2":   "08:30~10:00", // 第 1 大节
		"3-4":   "10:20~11:50", // 第 2 大节
		"5-6":   "14:00~15:20", // 第 3 大节
		"7-8":   "15:30~16:50", // 第 4 大节
		"9-10":  "18:30~19:50", // 第 5 大节
		"11-12": "20:00~21:20", // 第 6 大节
	}
	for in, want := range cases {
		if got := slotTimeOf(in); got != want {
			t.Errorf("slotTimeOf(%q) = %q, want %q", in, got, want)
		}
	}
	// 越界不 panic、不返回错时间。
	if got := slotTimeOf("13,14"); got != "" {
		t.Errorf("越界 = %q, want empty", got)
	}
	if got := slotTimeOf(""); got != "" {
		t.Errorf("空 = %q, want empty", got)
	}
}

// 连续上 4 小节 = 跨两个大节；time 必须覆盖整段，而不是只给起始大节。
func TestSlotTimeOfSpansMultipleSections(t *testing.T) {
	cases := map[string]string{
		"5-8":  "14:00~16:50", // 第 3 + 第 4 大节（下午一下午）
		"1-4":  "08:30~11:50", // 第 1 + 第 2 大节
		"3-6":  "10:20~15:20",
		"7-10": "15:30~19:50",
		"9-12": "18:30~21:20", // 第 5 + 第 6 大节
		"1-8":  "08:30~16:50",
		"5-6":  "14:00~15:20", // 单个大节保持原样
		"1-12": "08:30~21:20",
		"1,3":  "08:30~11:50", // 不连续节次：取首尾覆盖区间
		"5,11": "14:00~21:20",
	}
	for in, want := range cases {
		if got := slotTimeOf(in); got != want {
			t.Errorf("slotTimeOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// 全量核对：1..12 小节的全部 78 个连续区间都必须覆盖首尾大节，
// 防止以后只修某一段（如 5-8）而把别的连堂组合退回「只给起始大节」。
// 期望值独立由协议实测的 6 大节时间推导，不调用被测实现的辅助函数。
func TestSlotTimeOfAllContinuousRanges(t *testing.T) {
	base := []struct{ start, end string }{
		{"08:30", "10:00"}, {"10:20", "11:50"}, {"14:00", "15:20"},
		{"15:30", "16:50"}, {"18:30", "19:50"}, {"20:00", "21:20"},
	}
	for a := 1; a <= 12; a++ {
		for b := a; b <= 12; b++ {
			first, last := (a+1)/2-1, (b+1)/2-1
			want := base[first].start + "~" + base[last].end
			got := slotTimeOf(fmt.Sprintf("%d-%d", a, b))
			if got != want {
				t.Errorf("slotTimeOf(%d-%d) = %q, want %q", a, b, got, want)
			}
		}
	}
}

// 作息表条目格式异常时不能产出半段时间。
func TestSplitSlotTimeRejectsMalformed(t *testing.T) {
	for _, raw := range []string{"", "bad", "08:30", "08:30~10:00~11:00"} {
		if _, _, ok := splitSlotTime(raw); ok {
			t.Errorf("splitSlotTime(%q) 应当失败", raw)
		}
	}
	if start, end, ok := splitSlotTime("14:00-15:20"); !ok || start != "14:00" || end != "15:20" {
		t.Errorf("连字符分隔解析失败: %q %q %v", start, end, ok)
	}
}

// 排序用的小节号必须保持原始粒度（不是大节序号）。
func TestSlotIndexIsSubPeriod(t *testing.T) {
	if got := SlotIndex("7-8"); got != 7 {
		t.Errorf("SlotIndex(7-8) = %d, want 7", got)
	}
	if got := SlotIndexToSection("7-8"); got != 4 {
		t.Errorf("SlotIndexToSection(7-8) = %d, want 4", got)
	}
}

// 连续节号要压缩成区间；不连续的保留逗号。
func TestNormalizePeriodsCompactsRuns(t *testing.T) {
	cases := map[string]string{
		"1、2":     "1-2",
		"1、2、3、4": "1-4",
		"3、4":     "3-4",
		"5-8":     "5-8",
		"1,3":     "1,3",
		"9":       "9",
	}
	for in, want := range cases {
		if got := NormalizePeriods(in); got != want {
			t.Errorf("NormalizePeriods(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpandPeriods(t *testing.T) {
	cases := map[string][]int{
		"1,2": {1, 2},
		"5-8": {5, 6, 7, 8},
		"3-4": {3, 4},
		"9":   {9},
	}
	for in, want := range cases {
		if got := ExpandPeriods(in); !equalInts(got, want) {
			t.Errorf("ExpandPeriods(%q) = %v, want %v", in, got, want)
		}
	}
}

// 锚点页只给 MM-DD，必须能推出年份并确认首个日期是周一。
func TestParseWeek1Anchor(t *testing.T) {
	html := `<table>
	  <th><span>周一<br /></span><span>08-31</span></th>
	  <th><span>周二<br /></span><span>09-01</span></th>
	  <th><span>周三<br /></span><span>09-02</span></th>
	  <th><span>周四<br /></span><span>09-03</span></th>
	  <th><span>周五<br /></span><span>09-04</span></th>
	  <th><span>周六<br /></span><span>09-05</span></th>
	  <th><span>周日<br /></span><span>09-06</span></th>
	</table>`
	dates, err := ParseWeek1Anchor(html, 2026)
	if err != nil {
		t.Fatalf("ParseWeek1Anchor: %v", err)
	}
	if len(dates) != 7 {
		t.Fatalf("dates = %v", dates)
	}
	if dates[0].String() != "2026-08-31" {
		t.Errorf("第 1 周周一 = %s, want 2026-08-31", dates[0])
	}
	if dates[0].Weekday() != 1 {
		t.Errorf("首日不是周一: %v", dates[0])
	}
	if dates[6].String() != "2026-09-06" {
		t.Errorf("周日 = %s", dates[6])
	}
}

// 跨年学期：第 1 周在 8 月底，最后一周在次年 1 月。
func TestParseWeek1AnchorAcrossYear(t *testing.T) {
	html := `08-31 09-01 09-02 09-03 09-04 09-05 09-06`
	dates, err := ParseWeek1Anchor(html, 2027)
	if err != nil {
		t.Fatalf("ParseWeek1Anchor: %v", err)
	}
	// 8 月属于「当年前一学年」的第 1 周；用 2027 传入时应回退到 2026。
	if dates[0].String() != "2026-08-31" {
		t.Errorf("跨年锚点 = %s, want 2026-08-31", dates[0])
	}
}

func TestParseWeek1AnchorRejectsIncomplete(t *testing.T) {
	for _, html := range []string{
		"",
		"08-31 09-01",
		// 7 个日期但不连续
		"08-31 09-02 09-05 09-09 09-11 09-15 09-20",
	} {
		if _, err := ParseWeek1Anchor(html, 2026); err == nil {
			t.Errorf("ParseWeek1Anchor(%q) 应当报错", html)
		}
	}
}

func TestSummarizeWeeks(t *testing.T) {
	slots := []Slot{
		{Weeks: []int{1, 2, 3}},
		{Weeks: []int{5, 6}},
	}
	if got := summarizeWeeks(slots); got != "1-3,5-6周" {
		t.Errorf("summarizeWeeks = %q", got)
	}
	if got := summarizeWeeks(nil); got != "" {
		t.Errorf("空输入 = %q", got)
	}
}

func seq(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

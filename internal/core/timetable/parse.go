package timetable

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// 本文件是「周次 / 星期 / 节次 / 时间 / 日期」的纯解析，不发请求、不读时钟（AGENTS §6）。
//
// 实测教务把「星期 + 节次 + 周次」压在一个字段里：
//
//	周三第1、2节{2,4,7-10}全周
//	周二第5-8节{19}全周
//	周四第5、6节{9}全周
//
// 一条课程可含多个这样的片段，用 `;` 分隔（多时段 / 多地点）。
// 解析失败必须报错，不得产出「空周次」让课表静默变短。

var (
	reWeekdayCN = regexp.MustCompile(`(?:星期|周)([一二三四五六日天])`)
	rePeriods   = regexp.MustCompile(`第([0-9、,，\-—~]+)节`)
	reWeeksBody = regexp.MustCompile(`\{([^}]*)\}`)
	// 单双周标记：`全周` / `单周` / `双周`。
	reParity = regexp.MustCompile(`(\{|\})(单|双)周`)
	reAnchor = regexp.MustCompile(`(\d{1,2})-(\d{1,2})`)
	reRange  = regexp.MustCompile(`^(\d+)\s*[-~—]\s*(\d+)$`)
	reNumber = regexp.MustCompile(`\d+`)
)

// Slot 是一条课的一个上课时段（同一条课程可能有多个）。
type Slot struct {
	Weekday int    // 周一 = 1 … 周日 = 7
	Periods string // 归一化节次，如 "1-2" 或 "1,2"
	Weeks   []int  // 展开后的周次（升序）
}

// ParseSktime 把 `sktime` 字段解析成若干上课时段。
//
// 例：`周三第1、2节{2,4,7-10}全周;周四第1、2节{9}全周` → 两个 Slot。
func ParseSktime(raw string) ([]Slot, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, model.NewError(model.CodeMalformedResponse, "上课安排字段为空")
	}

	segments := strings.Split(trimmed, ";")
	slots := make([]Slot, 0, len(segments))
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		slot, err := parseSlotSegment(seg)
		if err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	if len(slots) == 0 {
		return nil, model.NewError(model.CodeMalformedResponse,
			fmt.Sprintf("上课安排未解析出任何时段: %q", raw))
	}
	return slots, nil
}

func parseSlotSegment(seg string) (Slot, error) {
	weekday, err := parseWeekdayCN(seg)
	if err != nil {
		return Slot{}, err
	}

	pm := rePeriods.FindStringSubmatch(seg)
	if pm == nil {
		return Slot{}, model.NewError(model.CodeMalformedResponse,
			fmt.Sprintf("上课安排缺少节次: %q", seg))
	}
	periods := normalizePeriods(pm[1])
	if periods == "" {
		return Slot{}, model.NewError(model.CodeMalformedResponse,
			fmt.Sprintf("节次无法解析: %q", seg))
	}

	wm := reWeeksBody.FindStringSubmatch(seg)
	if wm == nil {
		return Slot{}, model.NewError(model.CodeMalformedResponse,
			fmt.Sprintf("上课安排缺少周次: %q", seg))
	}
	parity := ""
	if pm := reParity.FindStringSubmatch(seg); pm != nil {
		parity = pm[2]
	}
	weeks, err := ExpandWeeks(wm[1], parity)
	if err != nil {
		return Slot{}, err
	}

	return Slot{Weekday: weekday, Periods: periods, Weeks: weeks}, nil
}

func parseWeekdayCN(s string) (int, error) {
	m := reWeekdayCN.FindStringSubmatch(s)
	if m == nil {
		return 0, model.NewError(model.CodeMalformedResponse,
			fmt.Sprintf("上课安排缺少星期: %q", s))
	}
	switch m[1] {
	case "一":
		return 1, nil
	case "二":
		return 2, nil
	case "三":
		return 3, nil
	case "四":
		return 4, nil
	case "五":
		return 5, nil
	case "六":
		return 6, nil
	case "日", "天":
		return 7, nil
	}
	return 0, model.NewError(model.CodeMalformedResponse, fmt.Sprintf("无法解析星期: %q", s))
}

// ExpandWeeks 把周次体展开为升序列表，如 `2,4,7-10` → [2 4 7 8 9 10]。
//
// parity 为 `单` / `双` 时按奇偶过滤；为空不过滤。
func ExpandWeeks(body, parity string) ([]int, error) {
	if strings.TrimSpace(body) == "" {
		return nil, model.NewError(model.CodeMalformedResponse, "周次内容为空")
	}

	set := map[int]bool{}
	for _, part := range strings.FieldsFunc(body, func(r rune) bool {
		return r == ',' || r == '，' || r == '、' || r == ' '
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if m := reRange.FindStringSubmatch(part); m != nil {
			from, _ := strconv.Atoi(m[1])
			to, _ := strconv.Atoi(m[2])
			if to < from {
				from, to = to, from
			}
			if from < 1 || to > 60 {
				return nil, model.NewError(model.CodeMalformedResponse,
					fmt.Sprintf("周次区间超出合理范围: %q", body))
			}
			for w := from; w <= to; w++ {
				set[w] = true
			}
			continue
		}
		for _, num := range reNumber.FindAllString(part, -1) {
			w, _ := strconv.Atoi(num)
			if w < 1 || w > 60 {
				return nil, model.NewError(model.CodeMalformedResponse,
					fmt.Sprintf("周次超出合理范围: %q", body))
			}
			set[w] = true
		}
	}

	weeks := make([]int, 0, len(set))
	for w := range set {
		switch parity {
		case "单":
			if w%2 == 0 {
				continue
			}
		case "双":
			if w%2 == 1 {
				continue
			}
		}
		weeks = append(weeks, w)
	}
	if len(weeks) == 0 {
		return nil, model.NewError(model.CodeMalformedResponse,
			fmt.Sprintf("周次内容未解析出任何周次: %q", body))
	}
	sort.Ints(weeks)
	return weeks, nil
}

// WeekdayCN 返回 1..7 对应的「一 … 日」。
func WeekdayCN(weekday int) string {
	switch weekday {
	case 1:
		return "一"
	case 2:
		return "二"
	case 3:
		return "三"
	case 4:
		return "四"
	case 5:
		return "五"
	case 6:
		return "六"
	case 7:
		return "日"
	}
	return ""
}

// SlotIndex 从节次文本取出起始节次（1 起）；`1-2` / `1、2` / `5-8` 都取首个数。
func SlotIndex(periods string) int {
	m := reNumber.FindString(periods)
	if m == "" {
		return 0
	}
	n, _ := strconv.Atoi(m)
	return n
}

// normalizePeriods 归一化节次文本：全角连字符 / 顿号统一成半角，去掉空白，
// 并把连续节号压缩成区间（`1、2` → `1-2`，`1、2、3、4` → `1-4`）。
//
// 统一形态是为了让 `periods` 稳定可 diff，也便于人类读「第 1-2 节」。
func normalizePeriods(raw string) string {
	s := strings.TrimSpace(raw)
	replacer := strings.NewReplacer(
		"－", "-", "—", "-", "~", "-", "，", ",", "、", ",",
	)
	s = replacer.Replace(s)
	s = strings.Join(strings.Fields(s), "")

	nums := ExpandPeriods(s)
	if len(nums) == 0 {
		return s
	}
	// 连续 → 区间；不连续 → 用逗号连接（保持原始粒度）。
	var parts []string
	start, prev := nums[0], nums[0]
	flush := func() {
		if start == prev {
			parts = append(parts, strconv.Itoa(start))
			return
		}
		parts = append(parts, fmt.Sprintf("%d-%d", start, prev))
	}
	for _, n := range nums[1:] {
		if n == prev+1 {
			prev = n
			continue
		}
		flush()
		start, prev = n, n
	}
	flush()
	return strings.Join(parts, ",")
}

// NormalizePeriods 是对外暴露的节次归一化（协议层与测试共用）。
func NormalizePeriods(raw string) string { return normalizePeriods(raw) }

// slotTimeOf 由节次取大节时间；越界时返回空串（不影响课表正确性）。
//
// 注意换算是「小节号 → 大节序号」，不是直接下标：教务只有 6 个大节，
// 而 sktime 用的是小节号（1..12），映射为
//
//	1-2 → 第 1 大节, 3-4 → 第 2, 5-6 → 第 3, 7-8 → 第 4, 9-10 → 第 5, 11-12 → 第 6
//
// 直接用小节号当下标会让 7 节以后的课拿不到时间（实测存在的坑）。
func slotTimeOf(periods string) string {
	slot := slotIndex(periods)
	if slot < 1 || slot > len(SlotTimes) {
		return ""
	}
	return SlotTimes[slot-1]
}

// slotIndex 把起始小节号换算成大节序号（1 起）。
func slotIndex(periods string) int {
	first := SlotIndex(periods)
	if first < 1 {
		return 0
	}
	return (first + 1) / 2
}

// SlotIndex 从节次文本取出起始小节号（1 起）；`1-2` / `1,2` / `5-8` 都取首个数。
//
// 对外暴露的是小节号（教务 sktime 的口径）；大节序号用 SlotIndexToSection。
func SlotIndexToSection(periods string) int { return slotIndex(periods) }

// ExpandPeriods 把节次文本展开成具体节号，用于按大节逐节判断跨节。
//
// `5-8` → [5 6 7 8]；`1,2` → [1 2]。
func ExpandPeriods(periods string) []int {
	parts := strings.FieldsFunc(periods, func(r rune) bool { return r == ',' || r == '-' })
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 60 {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	// `5-8` 这种被 FieldsFunc 拆成两端，需要补齐中间节号。
	if strings.Contains(periods, "-") && len(out) == 2 {
		for n := out[0] + 1; n < out[1]; n++ {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

var reAnchorDate = regexp.MustCompile(`\b(\d{1,2})-(\d{1,2})\b`)

// monthDay 是锚点页给出的无年份月日。
type monthDay struct {
	month int
	day   int
}

// ParseWeek1Anchor 从 loadkb 页面表格文本里抽取第 1 周的 7 个日期。
//
// 教务页面只给 `MM-DD`（如 `08-31`），年份需要推断：
// 取「给定年份」中与这 7 个连续月日匹配的那一组，7 个日期必须连续且周一对齐。
// inferYear 由调用方用 Clock 的当前年份传入，避免 core 读时钟。
func ParseWeek1Anchor(html string, currentYear int) ([]model.Date, error) {
	matches := reAnchorDate.FindAllStringSubmatch(html, -1)
	if len(matches) < 7 {
		return nil, model.NewError(model.CodeMalformedResponse,
			"第 1 周锚点页未解析出 7 个日期，可能接口已变更")
	}

	candidates := make([]monthDay, 0, len(matches))
	for _, m := range matches {
		month, _ := strconv.Atoi(m[1])
		day, _ := strconv.Atoi(m[2])
		if month < 1 || month > 12 || day < 1 || day > 31 {
			continue
		}
		candidates = append(candidates, monthDay{month, day})
	}

	// 前 7 个连续且能构成「周一..周日」的日期就是第 1 周。
	// 学年跨年：第 1 周可能在当前年 8/9 月，也可能在「上一年」8/9 月
	// （例如 1 月时 currentYear 已是次年）。两个年份都试，避免整学期算错一年。
	for _, year := range []int{currentYear, currentYear - 1} {
		for start := 0; start+7 <= len(candidates); start++ {
			week := candidates[start : start+7]
			dates, ok := buildWeek(week, year)
			if ok {
				return dates, nil
			}
		}
	}
	return nil, model.NewError(model.CodeMalformedResponse,
		"第 1 周锚点日期无法拼成连续 7 天，可能接口已变更")
}

// buildWeek 尝试用候选月日与年份拼出连续 7 天、且首日为周一的日期序列。
func buildWeek(week []monthDay, year int) ([]model.Date, bool) {
	first := model.Date{Year: year, Month: week[0].month, Day: week[0].day}
	// 学期跨越 8 月到次年 1 月：第 1 周在 8/9 月，若首日月份小于 6 则属于上一年。
	if first.Month < 6 {
		first.Year = year - 1
	}

	out := make([]model.Date, 0, 7)
	for i := 0; i < 7; i++ {
		d := first.AddDays(i)
		if d.Month != week[i].month || d.Day != week[i].day {
			return nil, false
		}
		out = append(out, d)
	}
	// 第 1 周必须从周一开始。
	if out[0].Weekday() != 1 {
		return nil, false
	}
	return out, true
}

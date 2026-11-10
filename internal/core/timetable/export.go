package timetable

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// 本文件把整学期课节导出成 WakeUp 课程表的 CSV 导入格式。
//
// 格式来源（唯一依据）：https://wakeup.fun/doc/import_from_csv.html 「五、课表正确填写示范」。
// 七列表头固定、顺序不可变；「星期 / 开始节数 / 结束节数」只能是一个具体数字；
// 「老师 / 地点」可以为空但必须写「无」，留空会直接导致导入失败。
//
// 为什么放在 core：列序、周次写法、间断周顿号规则都是业务格式，
// 三端共用一份才不会出现「CLI 导出的能导入、Android 导出的不能」（AGENTS A3）。

const (
	// wakeUpCSVHeader 是模板表头；改动等于改协议，必须对照官方文档确认。
	wakeUpCSVHeader = "课程名称,星期,开始节数,结束节数,老师,地点,周数"
	// wakeUpCSVNone 是空老师 / 空地点的占位（官方注意事项第 2、4 条）。
	wakeUpCSVNone = "无"
	// wakeUpCSVLineEnd 与官方模板一致用 CRLF：Excel / WPS 在部分系统上只认它。
	wakeUpCSVLineEnd = "\r\n"
	// wakeUpCSVBOM 与官方模板一致带 UTF-8 BOM，否则 WPS 打开中文会乱码。
	wakeUpCSVBOM = "\uFEFF"
	// wakeUpCSVSep 是间断周的顿号（官方注意事项第 6 条）。
	wakeUpCSVSep = "、"
)

// ExportLineEnd 是导出文件的行尾，供调用方按行切分（含表头计数）。
const ExportLineEnd = wakeUpCSVLineEnd

// ExportFilename 生成导出文件名：带上快照同步时间，同名文件不会互相覆盖，
// 也能一眼看出这份 CSV 是哪次同步的课表。
//
// 用 ASCII 而不是中文：文件要经「分享到微信 / QQ」转一手，
// ASCII 名在跨应用、跨系统的文件传输里不会被转码或截断。
func ExportFilename(syncedAt time.Time) string {
	if syncedAt.IsZero() {
		return "kemaolife-timetable.csv"
	}
	return "kemaolife-timetable-" + syncedAt.In(time.Local).Format("20060102-1504") + ".csv"
}

// wakeUpRow 是一行待输出的课节。
//
// 排序键与展示文本分开存：先按结构化字段排序，再输出已拼好的行，
// 保证课表内容不变时导出的文件逐字节一致、可 diff（AGENTS §6）。
type wakeUpRow struct {
	weekday  int
	start    int
	end      int
	name     string
	location string
	line     string
}

// ExportWakeUpCSV 把快照里的整学期课节导出为 WakeUp 课程表可导入的 CSV。
//
// 纯函数：不发请求、不读时钟、不写文件（AGENTS §6）。
func ExportWakeUpCSV(snap *Snapshot) (string, error) {
	if snap == nil {
		return "", model.NewError(model.CodeSnapshotMissing, "本地课表没有课程可导出，请先同步一次")
	}

	rows := make([]wakeUpRow, 0, len(snap.Lessons))
	seen := make(map[string]bool, len(snap.Lessons))
	for _, lesson := range snap.Lessons {
		if lesson.Weekday < 1 || lesson.Weekday > 7 || strings.TrimSpace(lesson.Name) == "" {
			continue
		}
		weeks := wakeUpWeeks(lesson.Weeks)
		if weeks == "" {
			continue
		}
		// 「第 1、3 节」这种不连续节次不能压成 1-3（会凭空多出第 2 节），
		// 拆成多行，每行一个连续区间。
		for _, group := range contiguousPeriods(ExpandPeriods(lesson.Periods)) {
			row := wakeUpRow{
				weekday:  lesson.Weekday,
				start:    group[0],
				end:      group[1],
				name:     lesson.Name,
				location: lesson.Location,
			}
			row.line = strings.Join([]string{
				csvCell(lesson.Name),
				strconv.Itoa(row.weekday),
				strconv.Itoa(row.start),
				strconv.Itoa(row.end),
				csvCellOrNone(lesson.Teacher),
				csvCellOrNone(lesson.Location),
				weeks,
			}, ",")
			// 上游同一门课可能因多时段 / 重复行给出完全相同的课节，去重后不重复占格。
			if seen[row.line] {
				continue
			}
			seen[row.line] = true
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return "", model.NewError(model.CodeSnapshotMissing, "本地课表没有课程可导出，请先同步一次")
	}

	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.weekday != b.weekday {
			return a.weekday < b.weekday
		}
		if a.start != b.start {
			return a.start < b.start
		}
		if a.end != b.end {
			return a.end < b.end
		}
		if a.name != b.name {
			return a.name < b.name
		}
		return a.location < b.location
	})

	var b strings.Builder
	b.WriteString(wakeUpCSVBOM)
	b.WriteString(wakeUpCSVHeader)
	b.WriteString(wakeUpCSVLineEnd)
	for _, row := range rows {
		b.WriteString(row.line)
		b.WriteString(wakeUpCSVLineEnd)
	}
	return b.String(), nil
}

// wakeUpWeeks 把展开后的周次写成 WakeUp 的写法：连续 `1-5`、隔周 `7-11单`、
// 间断用顿号分隔 `1-5、7-11单`（官方注意事项第 5、6 条）。
//
// 隔周写法只在「同奇偶且连续 3 周以上」时采用：两个孤立的隔周（`1、3`）
// 用顿号表达更稳，不依赖导入端如何理解 `1-3单`。
func wakeUpWeeks(weeks []int) string {
	unique := make(map[int]bool, len(weeks))
	sorted := make([]int, 0, len(weeks))
	for _, w := range weeks {
		if w < 1 || w > 60 || unique[w] {
			continue
		}
		unique[w] = true
		sorted = append(sorted, w)
	}
	sort.Ints(sorted)
	if len(sorted) == 0 {
		return ""
	}

	parts := make([]string, 0, 4)
	for i := 0; i < len(sorted); {
		consecutive := 1
		for i+consecutive < len(sorted) && sorted[i+consecutive] == sorted[i+consecutive-1]+1 {
			consecutive++
		}
		alternate := 1
		for i+alternate < len(sorted) && sorted[i+alternate] == sorted[i+alternate-1]+2 {
			alternate++
		}
		// 取更长的那个解释：`1,2,3` 是一段连续周，`2,4,6` 是一段隔周。
		if alternate >= 3 && alternate > consecutive {
			parts = append(parts, fmt.Sprintf("%d-%d%s",
				sorted[i], sorted[i+alternate-1], parityLabel(sorted[i])))
			i += alternate
			continue
		}
		parts = append(parts, spanText(sorted[i], sorted[i+consecutive-1]))
		i += consecutive
	}
	return strings.Join(parts, wakeUpCSVSep)
}

// parityLabel 返回单双周标记：奇数周是「单」，偶数是「双」。
func parityLabel(week int) string {
	if week%2 == 0 {
		return "双"
	}
	return "单"
}

func spanText(from, to int) string {
	if from == to {
		return strconv.Itoa(from)
	}
	return fmt.Sprintf("%d-%d", from, to)
}

// contiguousPeriods 把展开后的节号压成连续区间：`5-8` → `[[5 8]]`，
// `1,3` → `[[1 1] [3 3]]`（官方要求开始 / 结束节数各是一个具体数字）。
func contiguousPeriods(periods []int) [][2]int {
	if len(periods) == 0 {
		return nil
	}
	out := make([][2]int, 0, len(periods))
	start, prev := periods[0], periods[0]
	for _, n := range periods[1:] {
		if n == prev+1 {
			prev = n
			continue
		}
		out = append(out, [2]int{start, prev})
		start, prev = n, n
	}
	out = append(out, [2]int{start, prev})
	return out
}

// csvCell 转义单元格：逗号 / 双引号会破坏列结构，用双引号包裹并把内部双引号翻倍；
// 换行替换成空格，避免一条记录被拆成两行。
func csvCell(raw string) string {
	s := strings.TrimSpace(strings.NewReplacer(
		"\r\n", " ", "\n", " ", "\r", " ",
	).Replace(raw))
	if strings.ContainsAny(s, ",\"") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// csvCellOrNone 写老师 / 地点：留空会让 WakeUp 导入失败，必须写「无」。
func csvCellOrNone(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return wakeUpCSVNone
	}
	return csvCell(raw)
}

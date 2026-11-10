package app

import (
	"context"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// ExportResult 是一次课表导出的产出：整学期 CSV + 数据时间。
//
// Filename 由用例给出而不是留给 UI 拼：文件名要带上快照同步时间，
// 而「同步于何时」只有快照知道；让三端各拼一次必然出现三种命名。
type ExportResult struct {
	CSV      string `json:"csv"`
	Filename string `json:"filename"`
	// SyncedAt 是快照同步时间（RFC3339，本地时区）：CSV 内容来自这一份快照，
	// 导出侧必须能说明数据时间（docs/002 §5.3）。
	SyncedAt string `json:"syncedAt"`
	// LessonCount 是写进 CSV 的课节行数（不含表头），供 UI 回报「导出多少条」。
	LessonCount int `json:"lessonCount"`
}

// ExportTimetableCSV 把本地快照导出成 WakeUp 课程表可导入的 CSV。
//
// 只读缓存（CacheOnly，0 请求）：导出是「把这台机器上已有的课表带走」，
// 不是一次查询；联网刷新仍由独立的 timetable.sync 承担（docs/002 §5.3）。
// 因此这里与课表查询一样不要求 HTTP 端口，也不需要 ctx / cfg。
func ExportTimetableCSV(_ context.Context, d Deps, _ model.Config) (ExportResult, error) {
	if err := ensureDeps(d, false, true); err != nil {
		return ExportResult{}, err
	}

	snap, err := timetable.LoadSnapshot(d.Store)
	if err != nil {
		// 与课表查询同一口径：从没同步过引导「先刷新一次」，快照损坏仍保留 3003。
		return ExportResult{}, cacheMiss(CacheOnly, err)
	}

	csvText, err := timetable.ExportWakeUpCSV(snap)
	if err != nil {
		return ExportResult{}, err
	}

	// 行数含表头，减 1 得课节数；csv 末尾有换行，最后一段空串不计。
	rows := 0
	for _, line := range strings.Split(csvText, timetable.ExportLineEnd) {
		if strings.TrimSpace(line) != "" {
			rows++
		}
	}
	if rows > 0 {
		rows--
	}

	return ExportResult{
		CSV:         csvText,
		Filename:    timetable.ExportFilename(snap.SyncedAt),
		SyncedAt:    formatRFC3339(snap.SyncedAt),
		LessonCount: rows,
	}, nil
}

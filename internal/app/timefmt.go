package app

import (
	"strings"
	"time"
)

// localZone 让时间格式化统一落在本地时区，避免 UTC 转日期差一天（AGENTS §6）。
func localZone() *time.Location { return time.Local }

// formatRFC3339 统一信封里的时间格式（docs/002 §2 `ts` / `syncedAt`）。
func formatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(localZone()).Format(time.RFC3339)
}

// normalizeTimestamp 把上游多种时间写法统一成 RFC3339（本地时区）。
//
// 电费的可选写法（api-docs/002）：
//   - `2026-11-04`（opdtdate，数据截止日）
//   - `2026-11-04 22:41:37`（collecdate，采集时刻）
//
// 第二种必须按本地时区解释：用 UTC 解释会把「数据截至」整体挪一天。
func normalizeTimestamp(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		var (
			t   time.Time
			err error
		)
		if layout == time.RFC3339 {
			t, err = time.Parse(layout, s)
		} else {
			t, err = time.ParseInLocation(layout, s, localZone())
		}
		if err != nil {
			continue
		}
		return t.In(localZone()).Format(time.RFC3339), true
	}
	return s, false
}

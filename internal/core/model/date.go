package model

import (
	"fmt"
	"time"
)

// Date 是本地时区意义上的「日历日」。
//
// 为什么不用 time.Time 直接对外：电费 / 课表的日期都是「哪一天」，不含时刻。
// 用 time.Time 会在 UTC 与本地时区之间换算时差一天（见 AGENTS §6「时间」）。
type Date struct {
	Year  int
	Month int
	Day   int
}

// NewDate 取 t 在本地时区下的日历日。
func NewDate(t time.Time) Date {
	t = t.In(time.Local)
	return Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}

// ParseDate 解析 YYYY-MM-DD。
func ParseDate(s string) (Date, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return Date{}, fmt.Errorf("日期格式应为 YYYY-MM-DD: %q: %w", s, err)
	}
	return NewDate(t), nil
}

// Time 返回该日本地时间 00:00:00。
func (d Date) Time() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.Local)
}

func (d Date) IsZero() bool { return d.Year == 0 && d.Month == 0 && d.Day == 0 }

func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// Weekday 返回 ISO 星期：周一 = 1 … 周日 = 7。
func (d Date) Weekday() int {
	wd := int(d.Time().Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// AddDays 加减天数（按日历日，不涉及时刻）。
func (d Date) AddDays(n int) Date {
	utc := time.Date(d.Year, time.Month(d.Month), d.Day+n, 0, 0, 0, 0, time.UTC)
	return Date{Year: utc.Year(), Month: int(utc.Month()), Day: utc.Day()}
}

// DaysSince 返回 d 与 other 相差的天数（d 晚于 other 为正）。
//
// 内部用 UTC 零点做差，避免本地时区夏令时把 24 小时变成 23/25 小时。
func (d Date) DaysSince(other Date) int {
	a := time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
	b := time.Date(other.Year, time.Month(other.Month), other.Day, 0, 0, 0, 0, time.UTC)
	return int(a.Sub(b).Hours() / 24)
}

// Before 判断 d 是否早于 other。
func (d Date) Before(other Date) bool { return d.DaysSince(other) < 0 }

// After 判断 d 是否晚于 other。
func (d Date) After(other Date) bool { return d.DaysSince(other) > 0 }

func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + d.String() + `"`), nil
}

func (d *Date) UnmarshalJSON(b []byte) error {
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		s := string(b[1 : len(b)-1])
		if s == "" {
			*d = Date{}
			return nil
		}
		parsed, err := ParseDate(s)
		if err != nil {
			return err
		}
		*d = parsed
		return nil
	}
	return fmt.Errorf("日期字段应为字符串: %s", string(b))
}

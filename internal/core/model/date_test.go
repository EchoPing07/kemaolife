package model

import (
	"errors"
	"testing"
	"time"
)

func TestDateWeekdayISO(t *testing.T) {
	cases := []struct {
		date Date
		want int
	}{
		{Date{2026, 11, 2}, 1}, // 周一
		{Date{2026, 11, 5}, 4}, // 周四
		{Date{2026, 11, 8}, 7}, // 周日
	}
	for _, c := range cases {
		if got := c.date.Weekday(); got != c.want {
			t.Errorf("%s weekday = %d, want %d", c.date, got, c.want)
		}
	}
}

func TestDateAddDaysAcrossMonth(t *testing.T) {
	d := Date{2026, 10, 29}
	if got := d.AddDays(7).String(); got != "2026-11-05" {
		t.Errorf("AddDays(7) = %s, want 2026-11-05", got)
	}
	if got := d.AddDays(-1).String(); got != "2026-10-28" {
		t.Errorf("AddDays(-1) = %s, want 2026-10-28", got)
	}
}

func TestDateDaysSince(t *testing.T) {
	a := Date{2026, 11, 5}
	b := Date{2026, 10, 29}
	if got := a.DaysSince(b); got != 7 {
		t.Errorf("DaysSince = %d, want 7", got)
	}
	if got := b.DaysSince(a); got != -7 {
		t.Errorf("reverse DaysSince = %d, want -7", got)
	}
}

// 跨年、跨夏令时边界都必须按日历日算，不能按 24 小时。
func TestDateDaysSinceAcrossYear(t *testing.T) {
	a := Date{2027, 1, 3}
	b := Date{2026, 12, 28}
	if got := a.DaysSince(b); got != 6 {
		t.Errorf("DaysSince = %d, want 6", got)
	}
}

func TestParseDateLocalNoShift(t *testing.T) {
	// 用 UTC 转换会让 2026-11-05 变成 11-04（AGENTS §6「时间」）。
	d, err := ParseDate("2026-11-05")
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	if d.String() != "2026-11-05" {
		t.Errorf("ParseDate = %s, want 2026-11-05", d)
	}
	if got := NewDate(d.Time()); got != d {
		t.Errorf("round trip = %v, want %v", got, d)
	}
}

func TestParseDateRejectsBadFormat(t *testing.T) {
	for _, s := range []string{"", "2026/11/05", "2026-13-01", "abc"} {
		if _, err := ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) 应当报错", s)
		}
	}
}

func TestDateJSONRoundTrip(t *testing.T) {
	d := Date{2026, 11, 5}
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2026-11-05"` {
		t.Errorf("MarshalJSON = %s", b)
	}
	var back Date
	if err := back.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if back != d {
		t.Errorf("UnmarshalJSON = %v, want %v", back, d)
	}
}

func TestErrorMessageOfDoesNotLeakWrapped(t *testing.T) {
	// 展示文案只取 Message，不含 Err 的原文，避免上游报文里的个人信息漏出去。
	err := WrapError(CodeMalformedResponse, "响应格式异常", errors.New("upstream detail"))
	if got := MessageOf(err); got != "响应格式异常" {
		t.Errorf("MessageOf = %q", got)
	}
}

var fixedTime = time.Date(2026, 11, 5, 0, 20, 43, 0, time.Local)

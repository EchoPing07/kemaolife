package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// cmdElectricity 实现 `electricity [--month YYYY-MM]`（docs/002 §1.1）。
//
// 一次调用内部 2 个主请求 + 1 个采集时刻请求；主请求任一失败整体报错（R2）。
// 任何输出都必须标注「数据截至」：电表约 1 天延迟，不能当实时值。
func cmdElectricity(ctx context.Context, opts options, rt *session, args []string) error {
	yearMonth := ""
	for i := 0; i < len(args); i++ {
		name, value, hasValue := splitFlag(args[i])
		switch name {
		case "--month":
			if hasValue {
				yearMonth = value
				continue
			}
			if i+1 >= len(args) {
				return model.NewError(model.CodeInvalidArgument, "--month 缺少取值，示例：--month 2026-11")
			}
			i++
			yearMonth = args[i]
		default:
			return model.NewError(model.CodeInvalidArgument,
				"未知参数 "+quote(name)+"，electricity 只接受 --month")
		}
	}

	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	report, err := app.CheckElectricity(ctx, rt.deps, cfg, yearMonth)
	if err != nil {
		return err
	}

	return envelope(opts, model.Envelope{
		OK:   true,
		Data: report,
		TS:   formatNow(rt),
	}, func() {
		writeStdout(renderElectricity(report))
	})
}

func renderElectricity(r model.ElectricityReport) string {
	var b strings.Builder
	s := r.Status

	b.WriteString(renderTable([]string{"项", "值"}, [][]string{
		{"剩余电量", fmt.Sprintf("%.2f %s", s.Balance, orDefault(s.Unit, "度"))},
		{"宿舍", orDefault(s.RoomName, "（未返回）")},
		{"电表状态", orDefault(s.MeterStatus, "（未返回）")},
		{"数据截至", orDefault(s.CollectedAt, "（未返回）")},
		{"近 7 日均量", formatFloatPtr(s.DailyAvg, "度/天")},
		{"预计可用", formatIntPtr(s.EstimatedDays, "天")},
	}))

	u := r.Usage
	b.WriteString(fmt.Sprintf("\n%s 每日用量（数据截至 %s，当月累计 %.2f 度）\n",
		u.YearMonth, orDefault(u.DataUntil, "未知"), u.MonthUse))
	if len(u.Days) == 0 {
		b.WriteString("暂无每日用量数据\n")
		return b.String()
	}
	rows := make([][]string, 0, len(u.Days))
	for _, d := range u.Days {
		rows = append(rows, []string{d.Date, fmt.Sprintf("%.2f", d.Use)})
	}
	b.WriteString(renderTable([]string{"日期", "用量（度）"}, rows))
	return b.String()
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func formatFloatPtr(v *float64, unit string) string {
	if v == nil {
		return "无法估算"
	}
	return fmt.Sprintf("%.2f %s", *v, unit)
}

func formatIntPtr(v *int, unit string) string {
	if v == nil {
		return "无法估算"
	}
	return fmt.Sprintf("%d %s", *v, unit)
}

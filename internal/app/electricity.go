package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/electricity"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// CheckElectricity 一次查询同时返回余额与每日用量（ElectricityReport）。
//
// 2 个主请求任一失败整体报错，不返回半份数据；第 3 个采集时刻请求失败静默降级
// （docs/002 §3.2）。
// yearMonth 为空时取 Clock 当月的 YYYY-MM；
// schoolCode 为空时回落配置值。
func CheckElectricity(ctx context.Context, d Deps, cfg model.Config, yearMonth string) (model.ElectricityReport, error) {
	if err := ensureDeps(d, true, false); err != nil {
		return model.ElectricityReport{}, err
	}

	// 参数校验先于凭据检查：月份写错是命令问题，与账号无关（docs/002 §4 2001）。
	if yearMonth == "" {
		yearMonth = d.Clock.Now().In(localZone()).Format("2006-01")
	}
	normalized, err := normalizeYearMonth(yearMonth)
	if err != nil {
		return model.ElectricityReport{}, err
	}

	if cfg.StudentID == "" {
		return model.ElectricityReport{}, model.NewError(model.CodeCredentialMissing,
			"未配置学号，请先填写学号")
	}
	schoolCode := cfg.CustomerCode
	if schoolCode == "" {
		schoolCode = d.DefaultCustomerCode
	}
	if schoolCode == "" {
		return model.ElectricityReport{}, model.NewError(model.CodeCredentialMissing,
			"缺少学校代码，请填写学校代码")
	}

	client := electricity.NewClient(d.HTTP, d.ElectricityEndpoint)

	status, roomVerify, err := client.GetBindRoom(ctx, cfg.StudentID, schoolCode)
	if err != nil {
		return model.ElectricityReport{}, err
	}

	usage, err := client.GetUseDetail(ctx, cfg.StudentID, schoolCode, roomVerify, normalized)
	if err != nil {
		// 主请求任一失败整体报错，不返回半份数据。
		return model.ElectricityReport{}, err
	}

	electricity.WithEstimate(&status, usage.Days)

	// 「数据截至」优先用 getstuindexpage 的 collecdate（真实采集时刻），
	// 拿不到再退到 getusedetail 的 opdtdate（只有截止日，无时刻）。
	//
	// 为什么值得多发这一条请求：`opdtdate` 只到日，补成 00:00:00 会让用户看到
	// 「数据截至 00:00」这种明显不是采集时刻的值；而 collecdate 是 22:24:52 这类真时刻。
	if collectedAt, err := client.GetIndexPage(ctx, cfg.StudentID, schoolCode); err == nil && collectedAt != "" {
		if normalizedAt, ok := normalizeTimestamp(collectedAt); ok {
			status.CollectedAt = normalizedAt
		}
	}
	if status.CollectedAt == "" && usage.DataUntil != "" {
		if normalizedAt, ok := normalizeTimestamp(usage.DataUntil); ok {
			status.CollectedAt = normalizedAt
		}
	}

	return model.ElectricityReport{Status: status, Usage: usage}, nil
}

// normalizeYearMonth 校验并补零 YYYY-MM（api-docs/002 §6.2）。
func normalizeYearMonth(s string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(s), "-", 2)
	if len(parts) != 2 || len(parts[0]) != 4 {
		return "", model.NewError(model.CodeInvalidArgument,
			fmt.Sprintf("月份格式应为 YYYY-MM: %q", s))
	}
	if len(parts[1]) > 2 || parts[1] == "" {
		return "", model.NewError(model.CodeInvalidArgument,
			fmt.Sprintf("月份格式应为 YYYY-MM: %q", s))
	}
	month := parts[1]
	if len(month) == 1 {
		month = "0" + month
	}
	if month < "01" || month > "12" {
		return "", model.NewError(model.CodeInvalidArgument,
			fmt.Sprintf("月份超出 01-12: %q", s))
	}
	return parts[0] + "-" + month, nil
}

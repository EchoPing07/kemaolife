package electricity

import (
	"math"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Estimate 由近 7 日平均日用量估算可用天数（docs/002 §3.2）。
//
// 数据不足 7 日时用实际天数求均值；无数据或均量为 0 时返回 nil，
// 让输出明确「算不出」，不要用 0 冒充。
func Estimate(days []model.DayUsage) (dailyAvg *float64, estimatedDays *int) {
	if len(days) == 0 {
		return nil, nil
	}
	window := days
	if len(window) > 7 {
		window = window[len(window)-7:]
	}
	var sum float64
	for _, d := range window {
		sum += d.Use
	}
	avg := sum / float64(len(window))
	if avg <= 0 {
		zero := 0.0
		return &zero, nil
	}
	avg = math.Round(avg*100) / 100
	return &avg, nil
}

// WithEstimate 把日用量估算填进 status；balance 为剩余量。
//
// 单独拆出来是因为 CLI 先拿到 status（getbindroom），再拿到 days（getusedetail），
// 而 estimatedDays 依赖两者。
func WithEstimate(status *model.ElectricityStatus, days []model.DayUsage) {
	if status == nil {
		return
	}
	avg, _ := Estimate(days)
	status.DailyAvg = avg
	if avg == nil || *avg <= 0 {
		status.EstimatedDays = nil
		return
	}
	n := int(math.Floor(status.Balance / *avg))
	if n < 0 {
		n = 0
	}
	status.EstimatedDays = &n
}

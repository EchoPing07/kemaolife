package desktop_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
	"github.com/EchoPing07/kemaolife/internal/desktop"
)

func validSettingsResult() desktop.SettingsResult {
	return desktop.SettingsResult{
		StudentID:        "<学号>",
		PollSeconds:      "14400",
		CampusProbeSecs:  "120",
		BalanceThreshold: "10",
		DaysThreshold:    "3",
		QuietFrom:        "23",
		QuietTo:          "7",
	}
}

func TestTimetableConfigAndSettingsFormIncludeEduPassword(t *testing.T) {
	settings := desktop.DefaultSettings()
	settings.StudentID = "<学号>"
	settings.CustomerCode = "<学校代码>"
	settings.EduPassword = "<教务密码>"

	cfg := settings.TimetableConfig()
	if cfg.StudentID != settings.StudentID || cfg.CustomerCode != settings.CustomerCode ||
		cfg.EduPassword != settings.EduPassword {
		t.Errorf("TimetableConfig 未完整映射教务配置: %+v", cfg)
	}
	if form := desktop.ToSettingsForm(settings); form.EduPassword != settings.EduPassword {
		t.Errorf("设置表单教务密码 = %q, want %q", form.EduPassword, settings.EduPassword)
	}
}

func TestParseSettingsResultBlankEduPasswordKeepsCurrent(t *testing.T) {
	base := desktop.DefaultSettings()
	base.EduPassword = "<旧教务密码>"
	base.CampusPassword = "<旧校园网密码>"

	// 旧版设置窗口不收集 EduPassword，因此其零值结果也必须保留密码。
	next, err := desktop.ParseSettingsResult(validSettingsResult(), base)
	if err != nil {
		t.Fatalf("ParseSettingsResult: %v", err)
	}
	if next.EduPassword != base.EduPassword {
		t.Errorf("空教务密码应保留旧值, got %q", next.EduPassword)
	}
	// 两类密码共用「留空不修改」语义，避免设置窗口回显已保存凭据。
	res := validSettingsResult()
	res.CampusPassword = " <新校园网密码> "
	next, err = desktop.ParseSettingsResult(res, base)
	if err != nil {
		t.Fatalf("ParseSettingsResult 新校园网密码: %v", err)
	}
	if next.CampusPassword != res.CampusPassword {
		t.Errorf("校园网密码应原样保存, got %q", next.CampusPassword)
	}

	res = validSettingsResult()
	res.EduPassword = " <新教务密码> "
	next, err = desktop.ParseSettingsResult(res, base)
	if err != nil {
		t.Fatalf("ParseSettingsResult 新密码: %v", err)
	}
	if next.EduPassword != res.EduPassword {
		t.Errorf("非空教务密码应原样保存, got %q", next.EduPassword)
	}
}

func TestSettingsStoreEduPasswordCompatibility(t *testing.T) {
	inner := porttest.NewMemStore()
	if err := inner.Put(desktop.ConfigStoreKey,
		[]byte(`{"studentId":"<学号>","campusPassword":"<校园网密码>"}`)); err != nil {
		t.Fatal(err)
	}
	store := desktop.NewSettingsStore(inner)
	old, err := store.Load()
	if err != nil {
		t.Fatalf("读取老配置: %v", err)
	}
	if old.EduPassword != "" {
		t.Errorf("老配置教务密码应为空, got %q", old.EduPassword)
	}

	old.EduPassword = "<教务密码>"
	if _, err := store.Save(old); err != nil {
		t.Fatalf("保存教务密码: %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.EduPassword != old.EduPassword {
		t.Errorf("教务密码往返失败: password=%q err=%v", loaded.EduPassword, err)
	}

	// 可选字段为零值时不得写入 JSON，保持老配置的字段集合兼容。
	loaded.EduPassword = ""
	if _, err := store.Save(loaded); err != nil {
		t.Fatalf("保存空教务密码: %v", err)
	}
	raw, err := inner.Get(desktop.ConfigStoreKey)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["eduPassword"]; exists {
		t.Errorf("空教务密码应省略: %s", raw)
	}
}

func TestPollKeepsFullElectricityReport(t *testing.T) {
	campus := newFakeCampus(model.CampusOnline)
	srv := electricityServer(t, 35.99, 2)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}

	st := eng.Status()
	if !st.HasReport {
		t.Fatal("成功查询后 HasReport 应为 true")
	}
	if st.Report.Status.RoomName == "" || st.Report.Status.RoomVerify == "" {
		t.Errorf("完整电费概况丢失: %+v", st.Report.Status)
	}
	if st.Report.Usage.YearMonth != "2026-11" || st.Report.Usage.MonthUse != 63.13 ||
		st.Report.Usage.DataUntil != "2026-11-04" {
		t.Errorf("完整月度用量丢失: %+v", st.Report.Usage)
	}
	if len(st.Report.Usage.Days) != 1 || st.Report.Usage.Days[0].Date != "2026-11-04" ||
		st.Report.Usage.Days[0].Use != 2 {
		t.Errorf("每日用量丢失: %+v", st.Report.Usage.Days)
	}
}

func TestEngineStatusReturnsDeepCopy(t *testing.T) {
	campus := newFakeCampus(model.CampusOnline)
	srv := electricityServer(t, 35.99, 2)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}

	first := eng.Status()
	if first.Report.Status.DailyAvg == nil || first.Report.Status.EstimatedDays == nil ||
		len(first.Report.Usage.Days) == 0 {
		t.Fatalf("测试需要包含指针和切片的报告: %+v", first.Report)
	}
	*first.Report.Status.DailyAvg = -1
	*first.Report.Status.EstimatedDays = -1
	first.Report.Usage.Days[0].Date = "已修改"

	second := eng.Status()
	if *second.Report.Status.DailyAvg == -1 || *second.Report.Status.EstimatedDays == -1 ||
		second.Report.Usage.Days[0].Date == "已修改" {
		t.Errorf("修改 Status 返回值影响了引擎内部状态: %+v", second.Report)
	}
}

func TestCampusProbeAndRefreshKeepCompleteStatus(t *testing.T) {
	const checkedAt = "2001-02-03T04:05:06+08:00"
	for _, tc := range []struct {
		name string
		run  func(context.Context, *desktop.Engine) error
	}{
		{"Probe", func(ctx context.Context, eng *desktop.Engine) error {
			_, err := eng.ProbeCampus(ctx)
			return err
		}},
		{"Refresh", func(ctx context.Context, eng *desktop.Engine) error {
			eng.RefreshCampusStatus(ctx)
			return nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			campus := newFakeCampus(model.CampusOnline)
			campus.status = model.CampusStatus{
				CheckedAt:   checkedAt,
				ErrorMsg:    "<上游长文案>",
				BrasName:    "<BRAS>",
				BrasAddress: "<BRAS地址>",
				Vlan:        "<VLAN>",
				DeviceIP:    "<设备IP>",
				Mac:         "<MAC>",
			}
			srv := electricityServer(t, 50, 1)
			eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)
			if err := tc.run(context.Background(), eng); err != nil {
				t.Fatalf("刷新校园网状态: %v", err)
			}

			want := campus.status
			want.State = model.CampusOnline
			got := eng.Status().CampusStatus
			if !reflect.DeepEqual(got, want) {
				t.Errorf("校园网完整状态未保留:\n got %+v\nwant %+v", got, want)
			}
			// 上游 ErrorMsg 存于 DTO 供需要者诊断，但不能进入主 UI 状态文案。
			if text := desktop.FormatStatus(eng.Status()); strings.Contains(text, "<上游长文案>") {
				t.Errorf("主 UI 不应展示 CampusStatus.ErrorMsg: %q", text)
			}
		})
	}
}

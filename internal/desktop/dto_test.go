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

func TestTimetableConfigIncludesEduPasswordAndFormHidesPasswords(t *testing.T) {
	settings := desktop.DefaultSettings()
	settings.StudentID = "<学号>"
	settings.CustomerCode = "<学校代码>"
	settings.EduPassword = "<教务密码>"
	settings.CampusPassword = "<校园网密码>"

	cfg := settings.TimetableConfig()
	if cfg.StudentID != settings.StudentID || cfg.CustomerCode != settings.CustomerCode ||
		cfg.EduPassword != settings.EduPassword {
		t.Errorf("TimetableConfig 未完整映射教务配置: %+v", cfg)
	}
	form := desktop.ToSettingsForm(settings)
	if form.EduPassword != "" || form.CampusPassword != "" {
		t.Errorf("设置表单不得回显明文密码: edu=%q campus=%q", form.EduPassword, form.CampusPassword)
	}
	if form.StudentID != settings.StudentID || form.CustomerCode != settings.CustomerCode ||
		form.PollSeconds != settings.PollSeconds {
		t.Errorf("设置表单未回显非敏感配置: %+v", form)
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

func TestSettingsStoreTimetableDisplayCompatibilityAndRoundTrip(t *testing.T) {
	inner := porttest.NewMemStore()
	// 老配置不含课表显示三项字段时，应沿用默认值（优化开、@ 开、过长隐藏关）。
	if err := inner.Put(desktop.ConfigStoreKey, []byte(`{"studentId":"<学号>"}`)); err != nil {
		t.Fatal(err)
	}
	store := desktop.NewSettingsStore(inner)
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("读取老配置: %v", err)
	}
	if !loaded.OptimizeLocationDisplay || !loaded.PrefixLocationAt || loaded.HideTimetableOverflow {
		t.Fatalf("老配置课表显示默认值异常: opt=%v at=%v hide=%v",
			loaded.OptimizeLocationDisplay, loaded.PrefixLocationAt, loaded.HideTimetableOverflow)
	}

	// 显式反转全部三项后保存，再次读取不得回退到默认值。
	res := validSettingsResult()
	res.OptimizeLocationDisplay = false
	res.PrefixLocationAt = false
	res.HideTimetableOverflow = true
	next, err := desktop.ParseSettingsResult(res, loaded)
	if err != nil {
		t.Fatalf("ParseSettingsResult: %v", err)
	}
	if _, err := store.Save(next); err != nil {
		t.Fatalf("保存课表显示配置: %v", err)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("重读课表显示配置: %v", err)
	}
	if reloaded.OptimizeLocationDisplay || reloaded.PrefixLocationAt || !reloaded.HideTimetableOverflow {
		t.Fatalf("课表显示开关往返异常: opt=%v at=%v hide=%v",
			reloaded.OptimizeLocationDisplay, reloaded.PrefixLocationAt, reloaded.HideTimetableOverflow)
	}
	form := desktop.ToSettingsForm(reloaded)
	if form.OptimizeLocationDisplay || form.PrefixLocationAt || !form.HideTimetableOverflow {
		t.Fatalf("ToSettingsForm 未映射课表显示开关: %+v", form)
	}
}

func TestSettingsStoreThemeModeCompatibilityAndNormalization(t *testing.T) {
	inner := porttest.NewMemStore()
	if err := inner.Put(desktop.ConfigStoreKey, []byte(`{"studentId":"<学号>"}`)); err != nil {
		t.Fatal(err)
	}
	store := desktop.NewSettingsStore(inner)
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("读取老配置: %v", err)
	}
	if loaded.ThemeMode != desktop.ThemeSystem {
		t.Fatalf("老配置缺省 themeMode 应回落 system, got %q", loaded.ThemeMode)
	}
	if got := desktop.NormalizeThemeMode("invalid"); got != desktop.ThemeSystem {
		t.Fatalf("非法 themeMode 应归一化为 system, got %q", got)
	}

	for _, mode := range []string{desktop.ThemeLight, desktop.ThemeDark, desktop.ThemeSystem} {
		res := validSettingsResult()
		res.ThemeMode = mode
		next, err := desktop.ParseSettingsResult(res, loaded)
		if err != nil {
			t.Fatalf("ParseSettingsResult(%s): %v", mode, err)
		}
		if _, err := store.Save(next); err != nil {
			t.Fatalf("Save(%s): %v", mode, err)
		}
		reloaded, err := store.Load()
		if err != nil {
			t.Fatalf("Load(%s): %v", mode, err)
		}
		if reloaded.ThemeMode != mode {
			t.Fatalf("ThemeMode 往返不一致: got %q, want %q", reloaded.ThemeMode, mode)
		}
		if form := desktop.ToSettingsForm(reloaded); form.ThemeMode != mode {
			t.Fatalf("ToSettingsForm.ThemeMode = %q, want %q", form.ThemeMode, mode)
		}
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

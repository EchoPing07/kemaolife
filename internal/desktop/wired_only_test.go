package desktop_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
	"github.com/EchoPing07/kemaolife/internal/desktop"
)

// fakeEgress 是 core.WiredEgress 的内存实现，可随用例切换有线 / 无线。
type fakeEgress struct {
	mu    sync.Mutex
	wired bool
	calls int
}

func (f *fakeEgress) IsWired() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.wired
}

func (f *fakeEgress) set(wired bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wired = wired
}

func buildEngineWithEgress(t *testing.T, campus *fakeCampus, egress *fakeEgress,
	settings desktop.Settings, srv *httptest.Server) *desktop.Engine {
	t.Helper()
	clock := testClock()
	deps := app.Deps{
		Clock:               clock,
		HTTP:                srv.Client(),
		Store:               porttest.NewMemStore(),
		ConfigStore:         porttest.NewMemStore(),
		DefaultCustomerCode: "3126",
		ElectricityEndpoint: srv.URL,
	}
	var wired core.WiredEgress
	if egress != nil {
		wired = egress
	}
	campusDeps := app.CampusDeps{Clock: clock, Net: campus, Ready: campus, Egress: wired}
	return desktop.NewEngine(deps, campusDeps, &porttest.Recorder{},
		func() desktop.Settings { return settings }, nil)
}

// 无线避让：开启「仅有线」后主出口为无线时，前置整体跳过，
// 不发任何认证动作，电费照常查，跳过提示可见且不算错误。
func TestPollWiredOnlySkipsAutoLoginOnWireless(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = true
	settings.CampusWiredOnly = true

	eng := buildEngineWithEgress(t, campus, &fakeEgress{wired: false}, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("跳过前置不应阻断电费查询: %v", err)
	}

	login, logout, relogin, ensure := campus.counts()
	if login != 0 || logout != 0 || relogin != 0 || ensure != 0 {
		t.Errorf("无线出口不应有任何自动校园网动作: login=%d logout=%d relogin=%d ensure=%d",
			login, logout, relogin, ensure)
	}
	st := eng.Status()
	if !st.CampusAutoSkipped {
		t.Error("应标记自动认证已跳过")
	}
	if !st.HasReport {
		t.Error("电费查询不应被跳过")
	}
	if st.LastError != "" || st.CampusError != "" {
		t.Errorf("跳过是正常避让不是错误: last=%q campus=%q", st.LastError, st.CampusError)
	}
	if !strings.Contains(desktop.FormatStatus(st), "自动认证已跳过") {
		t.Errorf("托盘提示应包含跳过说明: %q", desktop.FormatStatus(st))
	}
}

// 有线出口：开启「仅有线」不改变正常自动认证。
func TestPollWiredOnlyAllowsOnWired(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = true
	settings.CampusWiredOnly = true

	eng := buildEngineWithEgress(t, campus, &fakeEgress{wired: true}, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	if _, _, _, ensure := campus.counts(); ensure != 1 {
		t.Errorf("有线出口应正常前置认证, got %d", ensure)
	}
	if eng.Status().CampusAutoSkipped {
		t.Error("有线出口不应标记跳过")
	}
}

// 有线恢复：拔线期间跳过后插回网线，下一轮恢复自动认证并清掉跳过标记。
func TestPollWiredOnlyRecoveryAfterReplug(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = true
	settings.CampusWiredOnly = true
	egress := &fakeEgress{wired: false}

	eng := buildEngineWithEgress(t, campus, egress, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("无线轮次: %v", err)
	}
	egress.set(true)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("有线轮次: %v", err)
	}
	if _, _, _, ensure := campus.counts(); ensure != 1 {
		t.Errorf("插回网线后应恢复前置认证, got %d", ensure)
	}
	if eng.Status().CampusAutoSkipped {
		t.Error("恢复有线后应清掉跳过标记")
	}
}

// 手动模式不受「仅有线」影响：本来就不自动登录，也不显示跳过提示。
func TestPollWiredOnlyIgnoredInManualMode(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = false
	settings.CampusWiredOnly = true

	eng := buildEngineWithEgress(t, campus, &fakeEgress{wired: false}, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	if _, _, _, ensure := campus.counts(); ensure != 0 {
		t.Errorf("手动模式不应前置认证, got %d", ensure)
	}
	if eng.Status().CampusAutoSkipped {
		t.Error("手动模式不应显示跳过提示")
	}
}

// 未注入端口（Android / 老组装路径）：即使开了开关也保持老行为。
func TestPollWiredOnlyNilEgressKeepsOldBehavior(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = true
	settings.CampusWiredOnly = true

	eng := buildEngineWithEgress(t, campus, nil, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	if _, _, _, ensure := campus.counts(); ensure != 1 {
		t.Errorf("无端口时应不限制, got %d", ensure)
	}
}

// 老配置兼容：无 campusWiredOnly 字段的旧 JSON 读出不限制；新字段可往返。
func TestSettingsWiredOnlyBackwardCompatible(t *testing.T) {
	mem := porttest.NewMemStore()
	store := desktop.NewSettingsStore(mem)
	old := `{"studentId":"<学号>","autoMode":true,"pollSeconds":14400,` +
		`"balanceThreshold":10,"daysThreshold":3,"quietStartHour":23,"quietEndHour":7}`
	if err := mem.Put(desktop.ConfigStoreKey, []byte(old)); err != nil {
		t.Fatalf("写入旧配置: %v", err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CampusWiredOnly {
		t.Error("老配置默认不应限制网卡")
	}

	cfg.CampusWiredOnly = true
	if _, err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reloaded.CampusWiredOnly {
		t.Error("保存后应读回限制开关")
	}
}

// 表单往返：开关经 SettingsResult → ParseSettingsResult → SettingsForm 不丢失。
func TestSettingsWiredOnlyFormRoundTrip(t *testing.T) {
	base := defaultSettings()
	res := desktop.SettingsResult{
		StudentID:        "<学号>",
		AutoMode:         true,
		WiredOnly:        true,
		PollSeconds:      "14400",
		CampusProbeSecs:  "120",
		BalanceThreshold: "10",
		DaysThreshold:    "3",
		QuietFrom:        "23",
		QuietTo:          "7",
	}
	next, err := desktop.ParseSettingsResult(res, base)
	if err != nil {
		t.Fatalf("ParseSettingsResult: %v", err)
	}
	if !next.CampusWiredOnly {
		t.Fatal("保存应写入限制开关")
	}
	if form := desktop.ToSettingsForm(next); !form.WiredOnly {
		t.Error("表单回显应带上限制开关")
	}
}

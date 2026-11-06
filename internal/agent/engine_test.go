package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/agent"
	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
)

// fakeCampus 是 core.CampusNetwork + core.NetworkReady 的内存实现。
//
// 它同时记录调用次数：断言「手动模式不自动登录」必须靠计数，
// 只看状态是看不出有没有偷偷调登录的。
type fakeCampus struct {
	mu sync.Mutex

	state    model.CampusState
	distoken string

	loginCalls   int
	logoutCalls  int
	reloginCalls int
	ensureCalls  int

	loginErr  error
	statusErr error
	ensureErr error
}

func newFakeCampus(state model.CampusState) *fakeCampus {
	return &fakeCampus{state: state, distoken: "<distoken>"}
}

func (f *fakeCampus) Status(context.Context) (model.CampusStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.statusErr != nil {
		return model.CampusStatus{}, f.statusErr
	}
	return model.CampusStatus{State: f.state}, nil
}

func (f *fakeCampus) Login(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loginCalls++
	if f.loginErr != nil {
		return f.loginErr
	}
	f.state = model.CampusOnline
	return nil
}

func (f *fakeCampus) Logout(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logoutCalls++
	f.state = model.CampusUnauth
	return nil
}

func (f *fakeCampus) Relogin(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reloginCalls++
	f.state = model.CampusOnline
	return nil
}

func (f *fakeCampus) EnsureCampusNetwork(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureCalls++
	if f.ensureErr != nil {
		return f.ensureErr
	}
	if f.state == model.CampusUnauth {
		f.loginCalls++
		f.state = model.CampusOnline
	}
	return nil
}

func (f *fakeCampus) setEnsureErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureErr = err
}

func (f *fakeCampus) counts() (login, logout, relogin, ensure int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.loginCalls, f.logoutCalls, f.reloginCalls, f.ensureCalls
}

func testClock() *porttest.FakeClock {
	// 中午：既避开默认静默时段（23–07），也让「今天」有明确含义。
	return porttest.NewFakeClock(time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local))
}

// electricityServer 返回一个只会返回固定余额 / 用量的电费后端。
func electricityServer(t *testing.T, balance float64, dayUse float64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		var param struct {
			Cmd string `json:"cmd"`
		}
		_ = json.Unmarshal([]byte(r.Form.Get("param")), &param)
		switch param.Cmd {
		case "getbindroom":
			body := fmt.Sprintf(`{"result":"0","existflag":"1","roomfullname":"<校区>_<楼栋>_<楼层>_<房间>","roomverify":"<roomverify>","detaillist":[{"businesstype":"0","odd":"%.2f","status":"1"}]}`, balance)
			writeOuter(w, body)
		case "getusedetail":
			body := fmt.Sprintf(`{"result":"0","yearmonth":"2026-11","monthuse":"63.13","opdtdate":"2026-11-04","dayuselist":[{"date":"2026-11-04","dayuse":%.2f}]}`, dayUse)
			writeOuter(w, body)
		case "getstuindexpage":
			writeOuter(w, `{"result":"0","modlist":[{"collecdate":"2026-11-04 22:41:37"}]}`)
		default:
			writeOuter(w, `{"result":"0"}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeOuter(w http.ResponseWriter, body string) {
	b, _ := json.Marshal(body)
	fmt.Fprintf(w, `{"code_":0,"result_":"true","body":%s,"message_":"获取成功！"}`, b)
}

// buildEngine 组装一个「电费可查 + 校园网可控」的引擎。
func buildEngine(t *testing.T, campus *fakeCampus, settings agent.Settings, srv *httptest.Server) (*agent.Engine, *porttest.Recorder, *porttest.FakeClock) {
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
	campusDeps := app.CampusDeps{Clock: clock, Net: campus, Ready: campus}
	rec := &porttest.Recorder{}

	settingsMu := sync.Mutex{}
	current := settings
	get := func() agent.Settings {
		settingsMu.Lock()
		defer settingsMu.Unlock()
		return current
	}
	eng := agent.NewEngine(deps, campusDeps, rec, get, nil)
	return eng, rec, clock
}

func defaultSettings() agent.Settings {
	s := agent.DefaultSettings()
	s.StudentID = "<学号>"
	return s
}

// 手动模式：轮询不得触发任何自动登录（000 §3.2 / 验收 8）。
func TestPollManualModeDoesNotTouchCampusNetwork(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = false

	eng, _, _ := buildEngine(t, campus, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}

	login, logout, relogin, ensure := campus.counts()
	if login != 0 || logout != 0 || relogin != 0 || ensure != 0 {
		t.Errorf("手动模式不应有任何自动校园网动作: login=%d logout=%d relogin=%d ensure=%d",
			login, logout, relogin, ensure)
	}
}

// 自动模式：轮询前置会调 EnsureCampusNetwork，且未认证时由它完成登录。
func TestPollAutoModeEnsuresCampusNetwork(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = true

	eng, _, _ := buildEngine(t, campus, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}

	login, _, _, ensure := campus.counts()
	if ensure != 1 {
		t.Errorf("自动模式应调用一次 EnsureCampusNetwork, got %d", ensure)
	}
	if login != 1 {
		t.Errorf("未认证态应触发登录, got %d", login)
	}
	if st := eng.Status(); st.Campus != model.CampusOnline {
		t.Errorf("状态应被刷新为 online, got %q", st.Campus)
	}
	if !eng.Status().HasReport {
		t.Error("应拿到电费数据")
	}
}

// 自动模式前置失败：只记到托盘状态，不阻断电费查询与余额提醒（docs/002 §6 规则 5）。
func TestPollAutoModePreflightFailureDoesNotBlockElectricity(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	campus.setEnsureErr(model.NewError(model.CodeNetworkDown, "不在校园网环境"))
	srv := electricityServer(t, 5, 1) // 低余额：成功路径应照常告警
	settings := defaultSettings()
	settings.AutoMode = true

	eng, rec, _ := buildEngine(t, campus, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("前置失败不应阻断电费查询: %v", err)
	}

	st := eng.Status()
	if !st.HasReport {
		t.Error("应拿到电费数据")
	}
	if !strings.Contains(st.LastError, "不在校园网环境") ||
		!strings.Contains(agent.FormatStatus(st), "不在校园网环境") {
		t.Errorf("前置失败应留在托盘提示: %q / %q", st.LastError, agent.FormatStatus(st))
	}
	if rec.Count() != 1 {
		t.Errorf("前置失败不应阻断余额提醒, got %d 次", rec.Count())
	}
}

// 下一轮前置成功后清掉上一轮的前置失败提示，不把旧错误一直挂在托盘上。
func TestPollAutoModePreflightErrorClearsAfterRecovery(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	campus.setEnsureErr(model.NewError(model.CodeNetworkDown, "不在校园网环境"))
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.AutoMode = true

	eng, _, _ := buildEngine(t, campus, settings, srv)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("第一轮: %v", err)
	}
	if eng.Status().LastError == "" {
		t.Fatal("第一轮应记录前置失败")
	}

	campus.setEnsureErr(nil)
	if err := eng.PollOnce(context.Background()); err != nil {
		t.Fatalf("第二轮: %v", err)
	}
	if got := eng.Status().LastError; got != "" {
		t.Errorf("前置恢复后应清空旧提示, got %q", got)
	}
}

// 跌破阈值只提醒一次（验收 7）。
func TestPollAlertsOncePerThreshold(t *testing.T) {
	campus := newFakeCampus(model.CampusOnline)
	srv := electricityServer(t, 5, 1) // 余额 5 度 < 默认阈值 10
	eng, rec, _ := buildEngine(t, campus, defaultSettings(), srv)

	for i := 0; i < 3; i++ {
		if err := eng.PollOnce(context.Background()); err != nil {
			t.Fatalf("第 %d 轮: %v", i+1, err)
		}
	}
	if rec.Count() != 1 {
		t.Fatalf("同一阈值应只提醒一次, got %d 次", rec.Count())
	}
	last, _ := rec.Last()
	if !strings.Contains(last.Body, "数据截至") {
		t.Errorf("通知正文必须含「数据截至」: %q", last.Body)
	}
}

// 电费查询失败：不弹余额通知，只更新状态（规则 5）。
func TestPollFailureDoesNotNotify(t *testing.T) {
	campus := newFakeCampus(model.CampusOnline)
	// 指向一个必然失败的地址。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	eng, rec, _ := buildEngine(t, campus, defaultSettings(), srv)
	err := eng.PollOnce(context.Background())
	if err == nil {
		t.Fatal("该路径应返回错误")
	}
	if rec.Count() != 0 {
		t.Errorf("失败不应弹余额通知, got %d", rec.Count())
	}
	st := eng.Status()
	if st.LastError == "" {
		t.Error("失败应记录到 LastError")
	}
	if st.HasReport {
		t.Error("失败不应产生数据")
	}
}

// 托盘右键两行状态 + 悬浮提示：只留单行短状态，不含房间名（P2），
// 也不展示探测时刻、可用天数与「数据截至」。
func TestTrayLineText(t *testing.T) {
	days := 3
	st := agent.Status{
		Campus:        model.CampusOnline,
		HasReport:     true,
		Balance:       35.99,
		Unit:          "度",
		CollectedAt:   time.Date(2026, 11, 4, 22, 41, 37, 0, time.Local).Format(time.RFC3339),
		EstimatedDays: &days,
	}
	campus, power := agent.TrayLine(st)
	if campus != "校园网：已在线" {
		t.Errorf("campus = %q", campus)
	}
	if power != "剩余电费：35.99 度" {
		t.Errorf("power = %q, want 剩余电费：35.99 度", power)
	}
	if strings.Contains(power, "房间") || strings.Contains(power, "<校区>") {
		t.Errorf("托盘不得显示房间信息: %q", power)
	}

	// 菜单行是单行项目：不带换行 / 分隔符，也不带可用天数。
	for _, line := range []string{campus, power} {
		if strings.ContainsAny(line, "\n｜（）") {
			t.Errorf("菜单行必须单行简短: %q", line)
		}
	}
	if strings.Contains(power, "约可用") {
		t.Errorf("菜单行不应带可用天数: %q", power)
	}

	// 悬浮提示同样不展示「数据截至」（采集时刻不进托盘 UI）。
	tip := agent.FormatStatus(st)
	if strings.Contains(tip, "数据截至") || strings.Contains(tip, "11-04") {
		t.Errorf("悬浮提示不应展示数据截至: %q", tip)
	}
}

// 上游长文案（如「正在进行外网拨号请稍候…」）不得进单行的菜单状态行。
func TestTrayLineDropsUpstreamMessage(t *testing.T) {
	st := agent.Status{
		Campus:    model.CampusOnline,
		CampusMsg: "正在进行外网拨号请稍候…",
	}
	campus, _ := agent.TrayLine(st)
	if campus != "校园网：已在线" {
		t.Errorf("campus = %q, want 校园网：已在线", campus)
	}
}

// 三态文案。
func TestTrayLineStates(t *testing.T) {
	cases := map[model.CampusState]string{
		model.CampusOnline:  "已在线",
		model.CampusUnauth:  "未认证",
		model.CampusOffline: "不在校园网",
	}
	for state, want := range cases {
		campus, _ := agent.TrayLine(agent.Status{Campus: state})
		if !strings.Contains(campus, want) {
			t.Errorf("state %q → %q, want 含 %q", state, campus, want)
		}
	}
}

// 设置校验：轮询间隔低于 600 秒必须钳制（AGENTS §5.5）。
func TestParseSettingsResultClampsPollInterval(t *testing.T) {
	base := agent.DefaultSettings()
	res := agent.SettingsResult{
		StudentID:        "<学号>",
		PollSeconds:      "60",
		CampusProbeSecs:  "5",
		BalanceThreshold: "10",
		DaysThreshold:    "3",
		QuietFrom:        "23",
		QuietTo:          "7",
	}
	next, err := agent.ParseSettingsResult(res, base)
	if err != nil {
		t.Fatalf("ParseSettingsResult: %v", err)
	}
	if next.PollSeconds != 600 {
		t.Errorf("PollSeconds = %d, want 600（钳制到下限）", next.PollSeconds)
	}
	if next.CampusProbeSecs != agent.CampusProbeFloorSeconds {
		t.Errorf("CampusProbeSecs = %d, want %d（钳制到下限）",
			next.CampusProbeSecs, agent.CampusProbeFloorSeconds)
	}
	if !agent.SettingsWereClamped(res, next) {
		t.Error("低于下限应可生成设置窗口提示")
	}
	hint := agent.ClampedSettingsHint(next)
	if !strings.Contains(hint, "600") || !strings.Contains(hint, "30") {
		t.Errorf("钳制提示应含生效值: %q", hint)
	}
}

// 探测间隔：默认 120s，钳制下限 30s，自定义值原样保留。
func TestCampusProbeIntervalNormalization(t *testing.T) {
	cases := []struct {
		name  string
		input int
		want  int
	}{
		{"零值取默认", 0, agent.DefaultCampusProbeSeconds},
		{"默认值保持", agent.DefaultCampusProbeSeconds, agent.DefaultCampusProbeSeconds},
		{"低于下限钳制", 5, agent.CampusProbeFloorSeconds},
		{"自定义值保留", 300, 300},
	}
	for _, c := range cases {
		s := agent.DefaultSettings()
		s.CampusProbeSecs = c.input
		if got := int(s.CampusProbeInterval().Seconds()); got != c.want {
			t.Errorf("%s: CampusProbeInterval = %d, want %d", c.name, got, c.want)
		}
	}
}

// 老配置没有探测间隔字段（omitempty）时必须能读起来并跑默认 120s。
func TestSettingsStoreDefaultsProbeIntervalOnOldConfig(t *testing.T) {
	inner := porttest.NewMemStore()
	// 模拟升级前的配置：没有 campusProbeSeconds。
	if err := inner.Put(agent.ConfigStoreKey, []byte(`{"studentId":"<学号>","pollSeconds":14400}`)); err != nil {
		t.Fatal(err)
	}

	store := agent.NewSettingsStore(inner)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CampusProbeSecs != agent.DefaultCampusProbeSeconds {
		t.Errorf("老配置探测间隔 = %d, want %d", cfg.CampusProbeSecs, agent.DefaultCampusProbeSeconds)
	}
}

// 手动探测：只发判态、不触发登录，且把探测时刻记到状态里。
func TestProbeCampusDoesNotLogin(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)

	state, err := eng.ProbeCampus(context.Background())
	if err != nil {
		t.Fatalf("ProbeCampus: %v", err)
	}
	if state != model.CampusUnauth {
		t.Errorf("state = %q, want unauth", state)
	}
	login, logout, relogin, ensure := campus.counts()
	if login != 0 || logout != 0 || relogin != 0 || ensure != 0 {
		t.Errorf("探测不得触发任何认证动作: login=%d logout=%d relogin=%d ensure=%d",
			login, logout, relogin, ensure)
	}
	if st := eng.Status(); st.CampusCheckedAt.IsZero() {
		t.Error("探测后应记录 CampusCheckedAt")
	}
	// 探测时刻只存不展：菜单行就是简短状态。
	if campus, _ := agent.TrayLine(eng.Status()); campus != "校园网：未认证" {
		t.Errorf("菜单行不得带探测时刻: %q", campus)
	}
}

// 设置校验：探测间隔非数字 / 为零必须报错。
func TestParseSettingsResultRejectsBadProbeInterval(t *testing.T) {
	base := agent.DefaultSettings()
	valid := agent.SettingsResult{
		PollSeconds:      "14400",
		CampusProbeSecs:  "120",
		BalanceThreshold: "10",
		DaysThreshold:    "3",
		QuietFrom:        "23",
		QuietTo:          "7",
	}
	for _, bad := range []string{"abc", "0", "-5"} {
		r := valid
		r.CampusProbeSecs = bad
		if _, err := agent.ParseSettingsResult(r, base); err == nil {
			t.Errorf("探测间隔 %q: 应报错", bad)
		}
	}
}

// 设置校验：非法输入必须报错而不是静默接受。
func TestParseSettingsResultRejectsInvalid(t *testing.T) {
	base := agent.DefaultSettings()
	valid := agent.SettingsResult{
		PollSeconds:      "14400",
		CampusProbeSecs:  "120",
		BalanceThreshold: "10",
		DaysThreshold:    "3",
		QuietFrom:        "23",
		QuietTo:          "7",
	}

	bad := []struct {
		name   string
		mutate func(*agent.SettingsResult)
	}{
		{"轮询间隔非数字", func(r *agent.SettingsResult) { r.PollSeconds = "abc" }},
		{"轮询间隔为零", func(r *agent.SettingsResult) { r.PollSeconds = "0" }},
		{"余额非数字", func(r *agent.SettingsResult) { r.BalanceThreshold = "x" }},
		{"余额为负", func(r *agent.SettingsResult) { r.BalanceThreshold = "-1" }},
		{"天数非数字", func(r *agent.SettingsResult) { r.DaysThreshold = "3.5" }},
		{"静默起点越界", func(r *agent.SettingsResult) { r.QuietFrom = "24" }},
		{"静默终点越界", func(r *agent.SettingsResult) { r.QuietTo = "-1" }},
	}
	for _, c := range bad {
		r := valid
		c.mutate(&r)
		if _, err := agent.ParseSettingsResult(r, base); err == nil {
			t.Errorf("%s: 应报错", c.name)
		}
	}
}

// 设置往返：保存后读回字段一致，且非法轮询间隔被钳制。
func TestSettingsStoreRoundTrip(t *testing.T) {
	store := agent.NewSettingsStore(porttest.NewMemStore())

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("首次 Load: %v", err)
	}
	if cfg.PollSeconds != int(agent.DefaultSettings().PollSeconds) {
		t.Errorf("默认轮询间隔 = %d", cfg.PollSeconds)
	}
	if cfg.AutoMode {
		t.Error("默认必须是手动模式")
	}

	cfg.StudentID = "<学号>"
	cfg.CampusUser = "<学号>"
	cfg.CampusPassword = "<密码>"
	cfg.PollSeconds = 60 // 低于下限
	clamped, err := store.Save(cfg)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !clamped {
		t.Error("低于下限应返回钳制标记")
	}

	back, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if back.PollSeconds != 600 {
		t.Errorf("落盘后轮询间隔 = %d, want 600", back.PollSeconds)
	}
	if back.CustomerCode != "" {
		t.Errorf("学校代码应留空以使用默认值, got %q", back.CustomerCode)
	}
	if back.CampusPassword != "<密码>" {
		t.Error("校园网密码未正确保存")
	}
	if agent.SettingsWereClamped(agent.SettingsResult{
		PollSeconds:     "14400",
		CampusProbeSecs: "120",
	}, back) {
		t.Error("合法间隔不应被误判为钳制")
	}
}

// 手动动作会同步刷新判态，用户点完按钮立刻看到结果。
func TestDoCampusRefreshesStatus(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)

	if err := eng.DoCampus(context.Background(), agent.ActionLogin); err != nil {
		t.Fatalf("ActionLogin: %v", err)
	}
	if st := eng.Status(); st.Campus != model.CampusOnline {
		t.Errorf("登录后状态 = %q, want online", st.Campus)
	}

	if err := eng.DoCampus(context.Background(), agent.ActionLogout); err != nil {
		t.Fatalf("ActionLogout: %v", err)
	}
	if st := eng.Status(); st.Campus != model.CampusUnauth {
		t.Errorf("下线后状态 = %q, want unauth", st.Campus)
	}
}

// 校园网动作失败要落到状态里，而不是只返回给调用方后被忽略。
func TestDoCampusRecordsError(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	campus.loginErr = fmt.Errorf("账号或密码错误")
	srv := electricityServer(t, 50, 1)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)

	if err := eng.DoCampus(context.Background(), agent.ActionLogin); err == nil {
		t.Fatal("应返回错误")
	}
	if st := eng.Status(); st.LastError == "" {
		t.Error("错误应记录到状态")
	}
}

// 网络探测失败归 offline：不写 LastError、不触发认证动作（docs/002 §5.2）。
func TestProbeCampusNetworkFailureIsOffline(t *testing.T) {
	campus := newFakeCampus(model.CampusOnline)
	campus.statusErr = model.NewError(model.CodeNetworkDown, "探测超时")
	srv := electricityServer(t, 50, 1)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)

	state, err := eng.ProbeCampus(context.Background())
	if err != nil {
		t.Fatalf("网络探测失败应归一为 offline: %v", err)
	}
	if state != model.CampusOffline {
		t.Errorf("state = %q, want offline", state)
	}
	st := eng.Status()
	if st.Campus != model.CampusOffline || st.LastError != "" {
		t.Errorf("offline 不应写错误提示: state=%q LastError=%q", st.Campus, st.LastError)
	}
	login, logout, relogin, ensure := campus.counts()
	if login != 0 || logout != 0 || relogin != 0 || ensure != 0 {
		t.Errorf("offline 不得触发认证动作: login=%d logout=%d relogin=%d ensure=%d",
			login, logout, relogin, ensure)
	}
}

// 内部故障仍保留提示，供托盘悬浮提示显示。
func TestProbeCampusInternalFailureKeepsError(t *testing.T) {
	campus := newFakeCampus(model.CampusOnline)
	// 用非 4001 的领域错误模拟内部故障：CodeOf 会把普通 error 统一归 4001，
	// 那样就会被 ProbeCampus 按「网络探测失败」正常归一为 offline。
	campus.statusErr = model.NewError(model.CodeInvalidArgument, "内部判态故障")
	srv := electricityServer(t, 50, 1)
	eng, _, _ := buildEngine(t, campus, defaultSettings(), srv)

	if _, err := eng.ProbeCampus(context.Background()); err == nil {
		t.Fatal("内部故障应返回错误")
	}
	if st := eng.Status(); st.LastError == "" {
		t.Error("内部故障应记录 LastError")
	}
}

// 探测轮询：到点只发判态、不触发任何认证动作（默认 120s 周期靠设置控制）。
func TestRunCampusProbeOnlyProbes(t *testing.T) {
	campus := newFakeCampus(model.CampusUnauth)
	srv := electricityServer(t, 50, 1)
	settings := defaultSettings()
	settings.CampusProbeSecs = 30 // 下限，测试里不等默认的 120s

	eng, _, _ := buildEngine(t, campus, settings, srv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		eng.Run(ctx, stop, nil)
		close(done)
	}()

	// 首轮立即探测：等到状态被写入即可，不必等满一个周期。
	deadline := time.Now().Add(10 * time.Second)
	for eng.Status().CampusCheckedAt.IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("启动后应立刻探测一次")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run 未在取消后退出")
	}

	login, logout, relogin, ensure := campus.counts()
	if login != 0 || logout != 0 || relogin != 0 || ensure != 0 {
		t.Errorf("探测轮询不得触发认证动作: login=%d logout=%d relogin=%d ensure=%d",
			login, logout, relogin, ensure)
	}
}

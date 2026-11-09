package kemaolife

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// localCampusNet 让所有校园网测试只走内存，绝不碰真实 Portal 探测地址（AGENTS §7）。
type localCampusNet struct{}

func (localCampusNet) Status(context.Context) (model.CampusStatus, error) {
	return model.CampusStatus{State: model.CampusOffline, CheckedAt: "<探测时间>"}, nil
}
func (localCampusNet) Login(context.Context) error   { return nil }
func (localCampusNet) Logout(context.Context) error  { return nil }
func (localCampusNet) Relogin(context.Context) error { return nil }
func (localCampusNet) EnsureCampusNetwork(context.Context) error {
	return nil
}

// 门面是最窄的对外契约（docs/002 §2.1）。测试全程不得碰真实学校接口（AGENTS §7）：
// 只用「无凭据 / 无快照」的本地失败路径与可注入的 session 成功路径。

func testSession(t *testing.T, cfg initConfig) *session {
	t.Helper()
	resetTimeZoneForTest()
	if cfg.CacheDir == "" {
		cfg.CacheDir = filepath.ToSlash(t.TempDir())
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if env := decode(t, Init(string(b))); env["ok"] != true {
		t.Fatalf("Init 失败: %v", env)
	}
	t.Cleanup(func() { resetTimeZoneForTest() })
	s := currentSession()
	// 门面生产装配的是真实 Portal；测试把它替换成本地桩，
	// 避免 campusnet.probe / campusnet.status 真的去探测学校内网。
	s.campus.Net = localCampusNet{}
	s.campus.Ready = localCampusNet{}
	return s
}

func TestCallBeforeInitFails(t *testing.T) {
	mu.Lock()
	saved := cur
	cur = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		cur = saved
		mu.Unlock()
	})

	env := decode(t, Call(`{"op":"ping"}`))
	if env["ok"] != false {
		t.Fatalf("未初始化时不应成功: %v", env)
	}
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 2001 {
		t.Errorf("code = %v, want 2001", errObj["code"])
	}
}

func TestInitPingAndUnknownOp(t *testing.T) {
	testSession(t, initConfig{
		StudentID:    "<学号>",
		EduPassword:  "<密码>",
		CustomerCode: "3126",
	})

	env := decode(t, Call(`{"op":"ping"}`))
	if env["ok"] != true {
		t.Fatalf("ping 失败: %v", env)
	}
	if env["ts"] == "" {
		t.Error("信封必须带 ts（docs/002 §2）")
	}
	if _, has := env["error"]; has {
		t.Error("成功信封不应出现 error")
	}

	bad := decode(t, Call(`{"op":"timetable.nope"}`))
	if bad["ok"] != false {
		t.Fatalf("未知 op 必须失败: %v", bad)
	}
	if _, has := bad["data"]; has {
		t.Error("失败信封不应出现 data")
	}
	errObj, _ := bad["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 2001 {
		t.Errorf("未知 op code = %v, want 2001", errObj["code"])
	}
	if msg, _ := errObj["message"].(string); strings.Contains(msg, "timetable.nope") {
		t.Errorf("未知 op 文案不得回显用户输入: %q", msg)
	}
}

// timetable.sync 的信封层 syncedAt 必须与 CLI 同构，且必须走真实用例成功路径
// （本地 httptest，不碰真实学校接口；AGENTS §7）。
func TestTimetableSyncSuccessEnvelopeIncludesSyncedAt(t *testing.T) {
	var cas, jsb *httptest.Server
	jsb = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ticket") != "" {
			http.SetCookie(w, &http.Cookie{Name: timetable.SessionCookie, Value: "sess-1"})
			http.SetCookie(w, &http.Cookie{Name: timetable.ServerCookie, Value: "srv-1"})
			http.Redirect(w, r, "/jsxsd/framework/xsMainV.htmlx", http.StatusFound)
			return
		}
		if !hasLocalSession(r) {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		switch {
		case strings.Contains(r.URL.Path, "mainV_index_loadkb"):
			fmt.Fprint(w, localAnchorHTML)
		case strings.Contains(r.URL.Path, "xskb_list"):
			fmt.Fprint(w, `{"code":0,"data":[]}`)
		default:
			fmt.Fprint(w, `<html>教务首页 课表</html>`)
		}
	}))
	defer jsb.Close()
	cas = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `<title>统一身份认证平台</title><input name="signupInputUserName" />`)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get(timetable.CASFieldUser) != "<学号>" ||
			r.Form.Get(timetable.CASFieldPass) != "<密码>" {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		http.Redirect(w, r, jsb.URL+"/jsxsd/?ticket=ST-1", http.StatusFound)
	}))
	defer cas.Close()

	s := testSession(t, initConfig{StudentID: "<学号>", EduPassword: "<密码>"})
	s.deps.CASBaseURL = cas.URL
	s.deps.JSBBaseURL = jsb.URL
	s.deps.HTTP = localHTTPDoer{}

	env := decode(t, Call(`{"op":"timetable.sync"}`))
	if env["ok"] != true {
		t.Fatalf("timetable.sync 失败: %v", env)
	}
	if syncedAt, _ := env["syncedAt"].(string); syncedAt == "" {
		t.Errorf("信封缺 syncedAt: %v", env)
	}
	data, _ := env["data"].(map[string]any)
	if data["syncedAt"] == "" || data["week1Monday"] != "2026-08-31" {
		t.Errorf("data = %v", data)
	}
	if env["ts"] == "" {
		t.Error("信封缺 ts")
	}
}

const localAnchorHTML = `<table><thead><tr>
<th><span>周一</span><span>08-31</span></th><th><span>周二</span><span>09-01</span></th>
<th><span>周三</span><span>09-02</span></th><th><span>周四</span><span>09-03</span></th>
<th><span>周五</span><span>09-04</span></th><th><span>周六</span><span>09-05</span></th>
<th><span>周日</span><span>09-06</span></th></tr></thead></table>`

func hasLocalSession(r *http.Request) bool {
	a, e1 := r.Cookie(timetable.SessionCookie)
	b, e2 := r.Cookie(timetable.ServerCookie)
	return e1 == nil && e2 == nil && a.Value != "" && b.Value != ""
}

type localHTTPDoer struct{ client *http.Client }

func (d localHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if d.client == nil {
		d.client = &http.Client{}
	}
	return d.client.Do(req)
}

func (d localHTTPDoer) DoNoRedirect(req *http.Request) (*http.Response, error) {
	c := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	return c.Do(req)
}

func TestInitRejectsMissingCacheDir(t *testing.T) {
	env := decode(t, Init(`{"studentId":"<学号>"}`))
	if env["ok"] != false {
		t.Fatalf("缺少可写目录应失败: %v", env)
	}
}

func TestCallRejectsBrokenJSON(t *testing.T) {
	env := decode(t, Call(`{`))
	if env["ok"] != false {
		t.Fatalf("非法请求应失败: %v", env)
	}
}

// 课表查询在「从没同步过」时必须回 3001（引导去设置页 / 刷新）。
func TestTimetableDateWithoutCredentialsReturns3001(t *testing.T) {
	testSession(t, initConfig{CustomerCode: "3126"})

	env := decode(t, Call(`{"op":"timetable.date","date":"2026-11-05"}`))
	if env["ok"] != false {
		t.Fatalf("无快照必须失败: %v", env)
	}
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 3001 {
		t.Errorf("code = %v, want 3001", errObj["code"])
	}
	if _, has := env["stale"]; has {
		t.Error("失败信封不应出现 stale")
	}
}

func TestDateArgumentValidation(t *testing.T) {
	testSession(t, initConfig{StudentID: "<学号>", EduPassword: "<密码>"})

	env := decode(t, Call(`{"op":"timetable.date","date":"2026/11/05"}`))
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 2001 {
		t.Errorf("code = %v, want 2001", errObj["code"])
	}
}

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("响应不是合法 JSON: %v\n%s", err, s)
	}
	return out
}

// 时区只校准一次：重复 Init 不得再写 time.Local（避免与在途 Call 数据竞争）。
func TestApplyTimeZoneIsIdempotent(t *testing.T) {
	saved := time.Local
	resetTimeZoneForTest()
	t.Cleanup(func() {
		time.Local = saved
		resetTimeZoneForTest()
	})

	applyTimeZone("Asia/Shanghai", 0)
	if got := time.Local.String(); got != "Asia/Shanghai" {
		t.Fatalf("Local = %q", got)
	}
	ts := time.Date(2026, 11, 6, 22, 48, 42, 0, time.Local).Format(time.RFC3339)
	if ts != "2026-11-06T22:48:42+08:00" {
		t.Errorf("format = %q, want +08:00", ts)
	}

	applyTimeZone("Not/AZone", 0)
	if got := time.Local.String(); got != "Asia/Shanghai" {
		t.Errorf("重复 Init 改写了 time.Local: %q", got)
	}
}

func TestApplyTimeZoneFallbackToOffset(t *testing.T) {
	saved := time.Local
	resetTimeZoneForTest()
	t.Cleanup(func() {
		time.Local = saved
		resetTimeZoneForTest()
	})

	applyTimeZone("Not/AZone", 8*3600)
	if _, off := time.Date(2026, 11, 6, 22, 48, 42, 0, time.Local).Zone(); off != 8*3600 {
		t.Errorf("offset = %d, want %d", off, 8*3600)
	}
}

// Init 传时区后，信封 ts 必须是本地时区而不是 UTC。
func TestInitAppliesTimeZoneToEnvelope(t *testing.T) {
	saved := time.Local
	resetTimeZoneForTest()
	t.Cleanup(func() {
		time.Local = saved
		resetTimeZoneForTest()
	})

	testSession(t, initConfig{TimeZone: "Asia/Shanghai", TimeZoneOffsetSeconds: 8 * 3600})
	ts, _ := decode(t, Call(`{"op":"ping"}`))["ts"].(string)
	if len(ts) < 6 || ts[len(ts)-6:] != "+08:00" {
		t.Errorf("ts = %q, 应以 +08:00 结尾（本地时区）", ts)
	}
}

// 复现真机问题：时区未校准时，collectedAt 会输出 UTC（14:48:42Z）。
func TestElectricityTimestampUsesDeviceZone(t *testing.T) {
	saved := time.Local
	resetTimeZoneForTest()
	t.Cleanup(func() {
		time.Local = saved
		resetTimeZoneForTest()
	})
	applyTimeZone("Asia/Shanghai", 8*3600)

	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", "2026-11-06 22:48:42", time.Local)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.In(time.Local).Format(time.RFC3339)
	if got != "2026-11-06T22:48:42+08:00" {
		t.Errorf("collectedAt = %q, 应为 +08:00", got)
	}
	if d := model.NewDate(parsed); d.String() != "2026-11-06" {
		t.Errorf("date = %s, 不应差一天", d)
	}
}

// 测试按钮的 op：失败也要 ok:true + data.ok:false，才能把耗时与原因展示出来。
func TestProbeOpReportsFailureInData(t *testing.T) {
	testSession(t, initConfig{CustomerCode: "3126"})

	env := decode(t, Call(`{"op":"timetable.probe"}`))
	if env["ok"] != true {
		t.Fatalf("测试动作的结构化回报必须 ok:true: %v", env)
	}
	data, _ := env["data"].(map[string]any)
	if data["ok"] != false {
		t.Fatalf("无凭据时 data.ok 必须为 false: %v", data)
	}
	if code, _ := data["code"].(float64); int(code) != 3001 {
		t.Errorf("code = %v, want 3001", data["code"])
	}
	if data["message"] == "" {
		t.Error("失败也要给脱敏后的 message")
	}
	if _, has := data["elapsedMs"]; !has {
		t.Error("测试回报必须带 elapsedMs")
	}
	if data["message"] != model.CodeMessage(model.CodeCredentialMissing) {
		t.Errorf("probe message 必须按错误码映射固定文案, got %v", data["message"])
	}
}

// timed() 成功路径：detail 精确白名单、可选 report / campusStatus 用同一次请求带回。
func TestTimedProbeSuccessCarriesDetailAndOptionalPayload(t *testing.T) {
	s := testSession(t, initConfig{})
	report := model.ElectricityReport{
		Status: model.ElectricityStatus{Balance: 35.99, Unit: "度", RoomName: "<房间>"},
		Usage:  model.ElectricityUsage{YearMonth: "2026-11", Days: []model.DayUsage{}},
	}
	got := decode(t, s.timed(func() (probeOutcome, error) {
		return probeOutcome{
			Detail: map[string]any{"balance": 35.99, "estimatedDays": ""},
			Report: &report,
		}, nil
	}))
	if got["ok"] != true {
		t.Fatalf("外层信封必须 ok:true: %v", got)
	}
	data, _ := got["data"].(map[string]any)
	if data["ok"] != true {
		t.Fatalf("data.ok 必须为 true: %v", data)
	}
	if _, has := data["elapsedMs"]; !has {
		t.Error("成功回报也必须带 elapsedMs")
	}
	detail, _ := data["detail"].(map[string]any)
	if detail["balance"] != 35.99 || detail["estimatedDays"] != "" {
		t.Errorf("detail = %v", detail)
	}
	payload, _ := data["report"].(map[string]any)
	status, _ := payload["status"].(map[string]any)
	if status["roomName"] != "<房间>" {
		t.Errorf("report 应回带完整 DTO, got %v", status)
	}
}

// 失败 message 不得带上游原文 / 用户输入。
func TestTimedProbeFailureMessageIsSanitized(t *testing.T) {
	s := testSession(t, initConfig{})
	raw := errors.New("上游异常: 学号 <学号> 房间 <房间> secret")
	got := decode(t, s.timed(func() (probeOutcome, error) {
		return probeOutcome{}, model.WrapError(model.CodeMalformedResponse, "上游异常: 学号 <学号> 房间 <房间>", raw)
	}))
	data, _ := got["data"].(map[string]any)
	msg, _ := data["message"].(string)
	if strings.Contains(msg, "<学号>") || strings.Contains(msg, "<房间>") || strings.Contains(msg, "secret") {
		t.Errorf("probe 失败文案泄漏上游原文: %q", msg)
	}
	if msg != model.CodeMessage(model.CodeMalformedResponse) {
		t.Errorf("message = %q, want 固定映射文案", msg)
	}
	if code, _ := data["code"].(float64); int(code) != int(model.CodeMalformedResponse) {
		t.Errorf("code = %v", data["code"])
	}
}

// 三个 *.probe 都已登记，且都不触碰真实学校接口：无凭据时走本地失败分支。
func TestProbeOpsAreRegistered(t *testing.T) {
	testSession(t, initConfig{CustomerCode: "3126"})

	for _, op := range []string{"timetable.probe", "electricity.probe", "campusnet.probe"} {
		env := decode(t, Call(`{"op":"`+op+`"}`))
		if env["ok"] != true {
			t.Fatalf("%s 必须回结构化成功信封: %v", op, env)
		}
		data, _ := env["data"].(map[string]any)
		if data == nil {
			t.Fatalf("%s 缺 data: %v", op, env)
		}
		if _, has := data["elapsedMs"]; !has {
			t.Errorf("%s 缺 elapsedMs: %v", op, data)
		}
		errObj, _ := env["error"].(map[string]any)
		if code, _ := errObj["code"].(float64); int(code) == 2001 {
			t.Fatalf("%s 未登记，被当成未知 op: %v", op, env)
		}
	}
}

// 会话代次：新 Init 后旧会话不得再写快照。
func TestGenerationStoreRejectsStaleSession(t *testing.T) {
	old := testSession(t, initConfig{})
	_ = testSession(t, initConfig{})

	if err := old.deps.Store.Put(timetable.SnapshotStoreKey, []byte("stale")); err == nil {
		t.Fatal("旧会话仍能写快照，会污染新账号的本地状态")
	}
	if _, err := old.deps.Store.Get(timetable.SnapshotStoreKey); err == nil {
		t.Error("旧会话不应在测试里留下快照")
	}
}

// resetSnapshot 会删掉旧账号快照：CacheOnly 不能先显示别人的课表。
func TestInitResetSnapshotClearsOldData(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir())
	resetTimeZoneForTest()
	first, _ := json.Marshal(initConfig{CacheDir: dir})
	if env := decode(t, Init(string(first))); env["ok"] != true {
		t.Fatalf("Init 失败: %v", env)
	}
	s := currentSession()
	if err := timetable.SaveSnapshot(s.deps.Store, &timetable.Snapshot{
		Version:     timetable.SnapshotVersion,
		SyncedAt:    time.Date(2026, 11, 7, 8, 0, 0, 0, time.Local),
		Week1Monday: model.Date{Year: 2026, Month: 8, Day: 31},
		SlotTimes:   []string{"08:30~10:00"},
		LastWeek:    16,
	}); err != nil {
		t.Fatal(err)
	}
	second, _ := json.Marshal(initConfig{CacheDir: dir, ResetSnapshot: true})
	if env := decode(t, Init(string(second))); env["ok"] != true {
		t.Fatalf("带 resetSnapshot 的 Init 失败: %v", env)
	}
	env := decode(t, Call(`{"op":"timetable.week"}`))
	if env["ok"] != false {
		t.Fatalf("旧快照应已被清除: %v", env)
	}
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 3001 {
		t.Errorf("code = %v, want 3001", errObj["code"])
	}
}

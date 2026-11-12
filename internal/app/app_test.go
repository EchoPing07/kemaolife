package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

const (
	testAccount = "<学号>"
	testPass    = "<密码>"
	testSchool  = "3126" // 学校事实，非个人隐私；允许入仓
)

func testClock() *porttest.FakeClock {
	return porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))
}

func testDeps(clock *porttest.FakeClock) (app.Deps, *porttest.MemStore, *porttest.MemStore) {
	store := porttest.NewMemStore()
	cfgStore := porttest.NewMemStore()
	return app.Deps{
		Clock:               clock,
		Store:               store,
		ConfigStore:         cfgStore,
		DefaultCustomerCode: testSchool,
	}, store, cfgStore
}

// seedSnapshot 写入一份「上周」的快照，用于验证「刷新成功会替换」。
func seedSnapshot(t *testing.T, store *porttest.MemStore) {
	t.Helper()
	snap := &timetable.Snapshot{
		Version:     timetable.SnapshotVersion,
		SyncedAt:    time.Date(2026, 10, 29, 21, 10, 2, 0, time.Local),
		Week1Monday: model.Date{Year: 2026, Month: 8, Day: 31},
		SlotTimes:   []string{"08:30~10:00", "10:20~11:50", "14:00~15:20", "15:30~16:50", "18:30~19:50", "20:00~21:20"},
		LastWeek:    16,
		Lessons: []timetable.Lesson{
			{Name: "旧课程", Periods: "1-2", Weekday: 4, Weeks: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				Teacher: "教师甲", Location: "教室A", WeeksText: "1-16周"},
		},
		Cookies: map[string]string{timetable.SessionCookie: "x", timetable.ServerCookie: "y"},
	}
	if err := timetable.SaveSnapshot(store, snap); err != nil {
		t.Fatal(err)
	}
}

// failingHTTP 模拟网络不可用：任何请求都失败。
type failingHTTP struct{}

func (failingHTTP) Do(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network unreachable")
}

func testConfig() model.Config {
	return model.Config{StudentID: testAccount, EduPassword: testPass, CustomerCode: testSchool}
}

// 刷新失败但有旧快照：查询照常返回，Stale=true，SyncedAt 是旧快照时间。
func TestGetScheduleFallsBackToOldSnapshot(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}
	seedSnapshot(t, store)

	result, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 5}, app.RefreshFirst)
	if err != nil {
		t.Fatalf("有旧快照时不应报错: %v", err)
	}
	if !result.Stale {
		t.Error("刷新失败时 Stale 必须为 true")
	}
	wantSynced := time.Date(2026, 10, 29, 21, 10, 2, 0, time.Local)
	if !result.SyncedAt.Equal(wantSynced) {
		t.Errorf("SyncedAt = %v, want 旧快照时间 %v", result.SyncedAt, wantSynced)
	}
	if result.Day.Week != 10 || len(result.Day.Entries) != 1 {
		t.Errorf("day = %+v", result.Day)
	}
	if result.Day.Entries[0].Name != "旧课程" {
		t.Errorf("应返回旧快照里的课程, got %q", result.Day.Entries[0].Name)
	}
}

// 刷新失败且没有旧快照：才报错（唯一需要先 sync 的场景）。
func TestGetScheduleWithoutAnySnapshotFails(t *testing.T) {
	clock := testClock()
	deps, _, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}

	_, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 5}, app.RefreshFirst)
	if err == nil {
		t.Fatal("无快照且刷新失败应报错")
	}
	if model.CodeOf(err) != model.CodeNetworkDown {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeNetworkDown)
	}
}

// 缺教务凭据时，有旧快照仍可查（stale），不把「没配密码」变成硬失败。
func TestGetScheduleWithoutCredentialsUsesSnapshot(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}
	seedSnapshot(t, store)

	result, err := app.GetSchedule(context.Background(), deps, model.Config{},
		model.Date{Year: 2026, Month: 11, Day: 5}, app.RefreshFirst)
	if err != nil {
		t.Fatalf("有快照时应可查: %v", err)
	}
	if !result.Stale {
		t.Error("未刷新成功时 Stale 应为 true")
	}
}

// 非学期内日期：inSemester:false + 空 entries，不是错误。
func TestGetScheduleOutOfSemester(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}
	seedSnapshot(t, store)

	result, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 7, Day: 15}, app.RefreshFirst)
	if err != nil {
		t.Fatalf("非学期内不应报错: %v", err)
	}
	if result.Day.InSemester {
		t.Error("InSemester 应为 false")
	}
	if len(result.Day.Entries) != 0 {
		t.Errorf("entries 应为空: %+v", result.Day.Entries)
	}
}

// 周查询：越界不报错；当周解析不出来才报 1002。
func TestGetWeekSchedule(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}
	seedSnapshot(t, store)

	result, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0, app.RefreshFirst)
	if err != nil {
		t.Fatalf("当周查询: %v", err)
	}
	if result.Week != 10 {
		t.Errorf("week = %d, want 10", result.Week)
	}
	if len(result.Days) != 7 {
		t.Fatalf("days = %d, want 7", len(result.Days))
	}
	// LastWeek 是 Android 翻页容器的边界（docs/002 §3.1 可选字段）。
	if result.LastWeek != 16 {
		t.Errorf("lastWeek = %d, want 16（快照推断的学期最后一周）", result.LastWeek)
	}

	// 越界周次：7 天全部 inSemester:false，不报错。
	out, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 30, app.RefreshFirst)
	if err != nil {
		t.Fatalf("越界周次不应报错: %v", err)
	}
	for _, d := range out.Days {
		if d.InSemester {
			t.Errorf("%s 不应在学期内", d.Date)
		}
	}
}

// 假期里未指定周次：返回 7 天 inSemester:false 的成功结果，不产生 1002。
func TestGetWeekScheduleOutsideSemesterIsSuccess(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 7, 1, 8, 0, 0, 0, time.Local))
	deps, store, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}
	seedSnapshot(t, store)

	result, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0, app.RefreshFirst)
	if err != nil {
		t.Fatalf("非学期内不应报错: %v", err)
	}
	if len(result.Days) != 7 {
		t.Fatalf("days = %d, want 7", len(result.Days))
	}
	for _, d := range result.Days {
		if d.InSemester || len(d.Entries) != 0 {
			t.Errorf("day = %+v, want InSemester:false / 空 entries", d)
		}
	}
	// 学期开始前也必须按「今天所在周」返回 7 天；Go 整除向零截断会错给第 1 周。
	if got := result.Days[0].Date; got.String() != "2026-06-29" {
		t.Errorf("学期前的当周 = %s, want 2026-06-29（今天所在周）", got)
	}
}

// 刷新成功必须替换快照：查询用新数据、Stale=false、SyncedAt 是本次时间。
func TestGetScheduleRefreshSuccessReplacesSnapshot(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	seedSnapshot(t, store) // 先放一份旧快照

	fake := newFakeSchool(t)
	deps.HTTP = fake.client
	deps.CASBaseURL = fake.casURL
	deps.JSBBaseURL = fake.jsbURL

	result, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 5}, app.RefreshFirst)
	if err != nil {
		t.Fatalf("刷新成功应可查: %v", err)
	}
	if result.Stale {
		t.Error("刷新成功时 Stale 必须为 false")
	}
	if !result.SyncedAt.Equal(clock.Now()) {
		t.Errorf("SyncedAt = %v, want 本次同步时间 %v", result.SyncedAt, clock.Now())
	}
	// 新快照里的课程应替换掉旧的「旧课程」。
	if len(result.Day.Entries) != 0 {
		t.Errorf("第 10 周周四按新快照应没课, got %+v", result.Day.Entries)
	}

	// 快照确实被覆盖了。
	snap, err := timetable.LoadSnapshot(store)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.SyncedAt.Equal(clock.Now()) {
		t.Errorf("存储里的快照未刷新: %v", snap.SyncedAt)
	}
	if len(snap.Lessons) == 0 || snap.Lessons[0].Name == "旧课程" {
		t.Errorf("存储里的快照仍是旧数据: %+v", snap.Lessons)
	}
}

func TestSyncTimetableWithoutCredentials(t *testing.T) {
	clock := testClock()
	deps, _, _ := testDeps(clock)

	_, err := app.SyncTimetable(context.Background(), deps, model.Config{})
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
}

// CheckElectricity：两个请求任一失败都要整体报错，不返回半份数据。
func TestCheckElectricityFailsWholeReport(t *testing.T) {
	clock := testClock()

	useOK := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		var param struct {
			Cmd string `json:"cmd"`
		}
		_ = json.Unmarshal([]byte(r.Form.Get("param")), &param)
		switch param.Cmd {
		case "getbindroom":
			writeOuter(w, `{"result":"0","existflag":"1","roomfullname":"<校区>_<楼栋>_<楼层>_<房间>","roomverify":"<roomverify>","detaillist":[{"businesstype":"0","odd":"35.99","status":"1"}]}`)
		case "getusedetail":
			if !useOK {
				writeOuter(w, `{"result":"-10005","message":"查询失败"}`)
				return
			}
			writeOuter(w, `{"result":"0","yearmonth":"2026-11","monthuse":"63.13","opdtdate":"2026-11-04","dayuselist":[{"date":"2026-11-04","dayuse":11.75}]}`)
		case "getstuindexpage":
			writeOuter(w, `{"result":"0","modlist":[{"collecdate":"2026-11-04 22:41:37"}]}`)
		}
	}))
	defer srv.Close()

	deps, _, _ := testDeps(clock)
	deps.HTTP = srv.Client()
	deps.ElectricityEndpoint = srv.URL

	report, err := app.CheckElectricity(context.Background(), deps, testConfig(), "2026-11")
	if err != nil {
		t.Fatalf("成功路径: %v", err)
	}
	if report.Status.Balance != 35.99 || report.Usage.MonthUse != 63.13 {
		t.Errorf("report = %+v", report)
	}
	// collectedAt 应取 collecdate 的真实采集时刻，而不是 opdtdate 补的 00:00:00。
	// 期望值按 time.Local 生成：CI runner 是 UTC，写死 +08:00 会让断言依赖运行机器时区。
	want := time.Date(2026, 11, 4, 22, 41, 37, 0, time.Local).Format(time.RFC3339)
	if report.Status.CollectedAt != want {
		t.Errorf("collectedAt = %q, want %q", report.Status.CollectedAt, want)
	}

	useOK = false
	if _, err := app.CheckElectricity(context.Background(), deps, testConfig(), "2026-11"); err == nil {
		t.Fatal("用量请求失败时必须整体报错，不能返回半份数据")
	}
}

// collecdate 拿不到时降级到 opdtdate（只有截止日），不报错。
func TestCheckElectricityFallsBackToOpdtdate(t *testing.T) {
	clock := testClock()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		var param struct {
			Cmd string `json:"cmd"`
		}
		_ = json.Unmarshal([]byte(r.Form.Get("param")), &param)
		switch param.Cmd {
		case "getbindroom":
			writeOuter(w, `{"result":"0","existflag":"1","roomverify":"<roomverify>","detaillist":[{"businesstype":"0","odd":"10.00","status":"1"}]}`)
		case "getusedetail":
			writeOuter(w, `{"result":"0","yearmonth":"2026-11","monthuse":"5.00","opdtdate":"2026-11-04","dayuselist":[]}`)
		case "getstuindexpage":
			// 缺 collecdate
			writeOuter(w, `{"result":"0","modlist":[{"modstatus":"1路:一般送电"}]}`)
		}
	}))
	defer srv.Close()

	deps, _, _ := testDeps(clock)
	deps.HTTP = srv.Client()
	deps.ElectricityEndpoint = srv.URL

	report, err := app.CheckElectricity(context.Background(), deps, testConfig(), "2026-11")
	if err != nil {
		t.Fatalf("CheckElectricity: %v", err)
	}
	want := time.Date(2026, 11, 4, 0, 0, 0, 0, time.Local).Format(time.RFC3339)
	if report.Status.CollectedAt != want {
		t.Errorf("collectedAt = %q, want 降级到 opdtdate %q", report.Status.CollectedAt, want)
	}
}

func TestCheckElectricityMonthNormalization(t *testing.T) {
	clock := testClock()
	deps, _, _ := testDeps(clock)
	deps.HTTP = failingHTTP{}
	deps.ElectricityEndpoint = "http://example.invalid"

	for _, m := range []string{"2026-13", "2026", "2026-1-1", "abc", "2026-"} {
		_, err := app.CheckElectricity(context.Background(), deps, testConfig(), m)
		if model.CodeOf(err) != model.CodeInvalidArgument {
			t.Errorf("月份 %q code = %d, want %d", m, model.CodeOf(err), model.CodeInvalidArgument)
		}
	}
}

func TestConfigRoundTripAndMasking(t *testing.T) {
	clock := testClock()
	deps, _, _ := testDeps(clock)

	cfg, err := app.LoadConfig(deps)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.CustomerCode != testSchool {
		t.Errorf("应回落内置默认学校代码, got %q", cfg.CustomerCode)
	}
	if app.CustomerCodeSource(deps, cfg) != model.CustomerCodeSourceDefault {
		t.Errorf("source = %q", app.CustomerCodeSource(deps, cfg))
	}

	cfg.StudentID = testAccount
	cfg.EduPassword = testPass
	cfg.CustomerCode = "<自定义学校代码>"
	if err := app.SaveConfig(deps, cfg); err != nil {
		t.Fatal(err)
	}
	back, err := app.LoadConfig(deps)
	if err != nil {
		t.Fatal(err)
	}
	if back.StudentID != testAccount || back.EduPassword != testPass {
		t.Errorf("round trip 失败: %+v", back)
	}
	if app.CustomerCodeSource(deps, back) != model.CustomerCodeSourceCustom {
		t.Errorf("覆盖后来源应为 custom")
	}
	if app.MaskSecret(back.StudentID) != "****" {
		t.Errorf("MaskSecret = %q", app.MaskSecret(back.StudentID))
	}
}

func writeOuter(w http.ResponseWriter, body string) {
	b, _ := json.Marshal(body)
	fmt.Fprintf(w, `{"code_":0,"result_":"true","body":%s,"message_":"获取成功！"}`, b)
}

var _ = strings.TrimSpace

// 纯读缓存：即使 HTTP 完全不可用也照常出数据，且不发任何请求（docs/002 §5.3）。
func TestGetScheduleCacheOnlyReadsLocalSnapshot(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	spy := &countingFailHTTP{}
	deps.HTTP = spy
	seedSnapshot(t, store)

	result, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 5}, app.CacheOnly)
	if err != nil {
		t.Fatalf("纯读缓存不应触发网络: %v", err)
	}
	if result.Stale {
		t.Error("纯读缓存没有「刷新失败」这回事，Stale 必须为 false")
	}
	if spy.calls != 0 {
		t.Errorf("CacheOnly 不应发请求, got %d", spy.calls)
	}
	if len(result.Day.Entries) != 1 {
		t.Errorf("应返回快照里的课程, got %+v", result.Day)
	}
}

// CacheOnly 连 HTTP 端口都不需要：纯本地查询不该被缺端口拦住。
func TestGetScheduleCacheOnlyDoesNotRequireHTTP(t *testing.T) {
	deps, store, _ := testDeps(testClock())
	deps.HTTP = nil
	seedSnapshot(t, store)

	if _, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 5}, app.CacheOnly); err != nil {
		t.Fatalf("CacheOnly 不应要求 HTTP 端口: %v", err)
	}
	if _, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 10, app.CacheOnly); err != nil {
		t.Fatalf("CacheOnly 不应要求 HTTP 端口: %v", err)
	}
}

// 纯读缓存且没有快照：回 3001 引导「先刷新一次」，而不是无法处理的 3003。
func TestGetWeekScheduleCacheOnlyWithoutSnapshot(t *testing.T) {
	deps, _, _ := testDeps(testClock())
	spy := &countingFailHTTP{}
	deps.HTTP = spy

	_, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0, app.CacheOnly)
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
	if spy.calls != 0 {
		t.Errorf("CacheOnly 不应发请求, got %d", spy.calls)
	}
}

// 快照损坏 / 版本不匹配是「重新同步」而不是「没配置凭据」：不能压成 3001。
func TestCacheOnlyKeepsSnapshotErrors(t *testing.T) {
	deps, store, _ := testDeps(testClock())
	deps.HTTP = &countingFailHTTP{}
	if err := store.Put(timetable.SnapshotStoreKey, []byte("not json")); err != nil {
		t.Fatal(err)
	}
	_, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0, app.CacheOnly)
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("损坏快照 code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
	if !strings.Contains(model.MessageOf(err), "重新同步") {
		t.Errorf("应提示重新同步: %q", model.MessageOf(err))
	}

	snap := &timetable.Snapshot{
		Version:     timetable.SnapshotVersion + 1,
		Week1Monday: model.Date{Year: 2026, Month: 8, Day: 31},
		SlotTimes:   []string{"08:30~10:00"},
		LastWeek:    16,
	}
	if err := timetable.SaveSnapshot(store, snap); err != nil {
		t.Fatal(err)
	}
	_, err = app.GetWeekSchedule(context.Background(), deps, testConfig(), 0, app.CacheOnly)
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("版本不匹配 code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
}

// 非法 CacheOption 一律拒绝：裸 int 不得绕过刷新判断或错误映射。
func TestInvalidCacheOptionRejected(t *testing.T) {
	deps, store, _ := testDeps(testClock())
	seedSnapshot(t, store)
	if _, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 5}, app.CacheOption(42)); model.CodeOf(err) != model.CodeInvalidArgument {
		t.Errorf("GetSchedule code = %d, want 2001", model.CodeOf(err))
	}
	if _, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0, app.CacheOption(42)); model.CodeOf(err) != model.CodeInvalidArgument {
		t.Errorf("GetWeekSchedule code = %d, want 2001", model.CodeOf(err))
	}
}

// 学期开始前按负数周次取「上上周」时必须真的返回那一周，而不是跳回当周。
func TestGetWeekScheduleHonorsNegativeWeek(t *testing.T) {
	deps, store, _ := testDeps(testClock())
	deps.HTTP = &countingFailHTTP{}
	seedSnapshot(t, store)

	got, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), -2, app.CacheOnly)
	if err != nil {
		t.Fatal(err)
	}
	if got.Week != -2 {
		t.Errorf("week = %d, want -2", got.Week)
	}
	if len(got.Days) != 7 || got.Days[0].Date.String() != "2026-08-10" {
		t.Errorf("days = %+v, want 2026-08-10 起 7 天", got.Days)
	}
}

// 空白独立学号必须回落教务学号，不能把一串空格送进电费接口。
func TestElectricityStudentTrimsAndFallsBack(t *testing.T) {
	cfg := testConfig()
	cfg.ElectricityStudentID = "   "
	if got := cfg.ElectricityStudent(); got != testAccount {
		t.Errorf("空白独立学号应回落教务学号, got %q", got)
	}
	cfg.ElectricityStudentID = "  <电费学号>  "
	if got := cfg.ElectricityStudent(); got != "<电费学号>" {
		t.Errorf("独立学号应去空白, got %q", got)
	}
}

// 只配电费独立学号、不填教务凭据也必须能查电费（docs/002 §5.3 tab3）。
func TestCheckElectricityOnlyWithOwnStudentID(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		var param struct {
			Cmd     string `json:"cmd"`
			Account string `json:"account"`
		}
		_ = json.Unmarshal([]byte(r.Form.Get("param")), &param)
		seen = append(seen, param.Cmd+":"+param.Account)
		switch param.Cmd {
		case "getbindroom":
			writeOuter(w, `{"result":"0","existflag":"1","roomverify":"<roomverify>","detaillist":[{"businesstype":"0","odd":"10.00","status":"1"}]}`)
		case "getusedetail":
			writeOuter(w, `{"result":"0","yearmonth":"2026-11","monthuse":"5.00","opdtdate":"2026-11-04","dayuselist":[]}`)
		case "getstuindexpage":
			writeOuter(w, `{"result":"0","modlist":[{"collecdate":"2026-11-04 22:41:37"}]}`)
		}
	}))
	defer srv.Close()

	deps, _, _ := testDeps(testClock())
	deps.HTTP = srv.Client()
	deps.ElectricityEndpoint = srv.URL
	cfg := model.Config{ElectricityStudentID: "<电费学号>", CustomerCode: testSchool}
	if _, err := app.CheckElectricity(context.Background(), deps, cfg, "2026-11"); err != nil {
		t.Fatalf("只配电费学号也应能查询: %v", err)
	}
	if len(seen) != 3 {
		t.Fatalf("应发 3 个请求, got %v", seen)
	}
	for _, got := range seen {
		if !strings.HasSuffix(got, ":"+"<电费学号>") {
			t.Errorf("请求 %q 未使用电费独立学号", got)
		}
	}
}

// 电费独立学号：配置了就优先于教务学号（docs/002 §5.3 tab3 的「继承」开关）。
func TestCheckElectricityUsesOwnStudentID(t *testing.T) {
	clock := testClock()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		var param struct {
			Cmd     string `json:"cmd"`
			Account string `json:"account"`
		}
		_ = json.Unmarshal([]byte(r.Form.Get("param")), &param)
		seen = append(seen, param.Cmd+":"+param.Account)
		switch param.Cmd {
		case "getbindroom":
			writeOuter(w, `{"result":"0","existflag":"1","roomverify":"<roomverify>","detaillist":[{"businesstype":"0","odd":"10.00","status":"1"}]}`)
		case "getusedetail":
			writeOuter(w, `{"result":"0","yearmonth":"2026-11","monthuse":"5.00","opdtdate":"2026-11-04","dayuselist":[]}`)
		case "getstuindexpage":
			writeOuter(w, `{"result":"0","modlist":[{"collecdate":"2026-11-04 22:41:37"}]}`)
		}
	}))
	defer srv.Close()

	deps, _, _ := testDeps(clock)
	deps.HTTP = srv.Client()
	deps.ElectricityEndpoint = srv.URL

	cfg := testConfig()
	cfg.ElectricityStudentID = "<电费学号>"
	if _, err := app.CheckElectricity(context.Background(), deps, cfg, "2026-11"); err != nil {
		t.Fatalf("CheckElectricity: %v", err)
	}
	for _, got := range seen {
		if !strings.HasSuffix(got, ":"+"<电费学号>") {
			t.Errorf("请求 %q 未使用电费独立学号", got)
		}
	}
	if len(seen) != 3 {
		t.Errorf("应发 3 个请求, got %v", seen)
	}
}

// countingFailHTTP 记录调用次数并一律失败：用于断言「这条路径根本不联网」。
type countingFailHTTP struct{ calls int }

func (c *countingFailHTTP) Do(*http.Request) (*http.Response, error) {
	c.calls++
	return nil, fmt.Errorf("network unreachable")
}

// 导出只读缓存：即使 HTTP 完全不可用也照常出 CSV，且不发任何请求（docs/002 §5.3）。
// 真实同步链路必须把多时段地点分别写入快照，并按周查询返回当节真实地点。
func TestSyncTimetableMapsLocationsPerSlot(t *testing.T) {
	deps, _, _ := testDeps(testClock())
	fake := newFakeSchool(t)
	deps.HTTP = fake.client
	deps.CASBaseURL = fake.casURL
	deps.JSBBaseURL = fake.jsbURL

	result, err := app.SyncTimetable(context.Background(), deps, testConfig())
	if err != nil {
		t.Fatalf("SyncTimetable: %v", err)
	}
	if result.LessonCount != 2 {
		t.Errorf("lessonCount = %d, want 2", result.LessonCount)
	}

	// 第 10 周周三：第一个时段；第 9 周周四：第二个时段。
	wed, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 11, Day: 4}, app.CacheOnly)
	if err != nil {
		t.Fatalf("GetSchedule wed: %v", err)
	}
	if len(wed.Day.Entries) != 1 || wed.Day.Entries[0].Location != "<地点A>" {
		t.Errorf("wed entries = %+v, want <地点A>", wed.Day.Entries)
	}
	thu, err := app.GetSchedule(context.Background(), deps, testConfig(),
		model.Date{Year: 2026, Month: 10, Day: 29}, app.CacheOnly)
	if err != nil {
		t.Fatalf("GetSchedule thu: %v", err)
	}
	if len(thu.Day.Entries) != 1 || thu.Day.Entries[0].Location != "<地点B>" {
		t.Errorf("thu entries = %+v, want <地点B>", thu.Day.Entries)
	}
}

func TestExportTimetableCSVReadsLocalSnapshot(t *testing.T) {
	clock := testClock()
	deps, store, _ := testDeps(clock)
	spy := &countingFailHTTP{}
	deps.HTTP = spy
	seedSnapshot(t, store)

	result, err := app.ExportTimetableCSV(context.Background(), deps, testConfig())
	if err != nil {
		t.Fatalf("导出应只读缓存: %v", err)
	}
	if spy.calls != 0 {
		t.Errorf("导出不应发请求, got %d", spy.calls)
	}
	if !strings.HasSuffix(result.CSV, "\r\n") || !strings.Contains(result.CSV, "旧课程,4,1,2,教师甲,教室A,1-10") {
		t.Errorf("CSV 内容不符预期:\n%q", result.CSV)
	}
	// 数据时间取自快照，UI 必须一并展示。
	want := time.Date(2026, 10, 29, 21, 10, 2, 0, time.Local).Format(time.RFC3339)
	if result.SyncedAt != want {
		t.Errorf("syncedAt = %q, want %q", result.SyncedAt, want)
	}
	if result.Filename == "" || !strings.HasSuffix(result.Filename, ".csv") {
		t.Errorf("filename = %q", result.Filename)
	}
	if result.LessonCount != 1 {
		t.Errorf("lessonCount = %d, want 1", result.LessonCount)
	}
}

// 从没同步过：回 3001 引导「先刷新一次」，与课表查询同一口径。
func TestExportTimetableCSVWithoutSnapshot(t *testing.T) {
	deps, _, _ := testDeps(testClock())
	deps.HTTP = &countingFailHTTP{}

	_, err := app.ExportTimetableCSV(context.Background(), deps, testConfig())
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
}

// 快照损坏是「重新同步」而不是「没配置凭据」：保留 3003。
func TestExportTimetableCSVKeepsSnapshotErrors(t *testing.T) {
	deps, store, _ := testDeps(testClock())
	if err := store.Put(timetable.SnapshotStoreKey, []byte("not json")); err != nil {
		t.Fatal(err)
	}
	_, err := app.ExportTimetableCSV(context.Background(), deps, testConfig())
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
}

// 导出不需要 HTTP 端口：纯本地动作不该被缺端口拦住。
func TestExportTimetableCSVDoesNotRequireHTTP(t *testing.T) {
	deps, store, _ := testDeps(testClock())
	deps.HTTP = nil
	seedSnapshot(t, store)
	if _, err := app.ExportTimetableCSV(context.Background(), deps, testConfig()); err != nil {
		t.Fatalf("导出不应要求 HTTP 端口: %v", err)
	}
}

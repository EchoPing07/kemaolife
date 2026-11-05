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
		model.Date{Year: 2026, Month: 11, Day: 5})
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
		model.Date{Year: 2026, Month: 11, Day: 5})
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
		model.Date{Year: 2026, Month: 11, Day: 5})
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
		model.Date{Year: 2026, Month: 7, Day: 15})
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

	result, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0)
	if err != nil {
		t.Fatalf("当周查询: %v", err)
	}
	if result.Week != 10 {
		t.Errorf("week = %d, want 10", result.Week)
	}
	if len(result.Days) != 7 {
		t.Fatalf("days = %d, want 7", len(result.Days))
	}

	// 越界周次：7 天全部 inSemester:false，不报错。
	out, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 30)
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

	result, err := app.GetWeekSchedule(context.Background(), deps, testConfig(), 0)
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
		model.Date{Year: 2026, Month: 11, Day: 5})
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
	if report.Status.CollectedAt != "2026-11-04T22:41:37+08:00" {
		t.Errorf("collectedAt = %q, want 2026-11-04T22:41:37+08:00", report.Status.CollectedAt)
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
	if report.Status.CollectedAt != "2026-11-04T00:00:00+08:00" {
		t.Errorf("collectedAt = %q, want 降级到 opdtdate", report.Status.CollectedAt)
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

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
	testSchool  = "<学校代码>"
)

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

func seedSnapshot(t *testing.T, store *porttest.MemStore) {
	t.Helper()
	snap := &timetable.Snapshot{
		Version:     timetable.SnapshotVersion,
		SyncedAt:    time.Date(2026, 11, 4, 21, 10, 2, 0, time.Local),
		Week1Monday: model.Date{Year: 2026, Month: 8, Day: 31},
		SlotTimes:   []string{"08:30~10:00", "10:15~11:45", "14:00~15:30", "15:45~17:15", "19:00~20:30", "20:45~22:15"},
		LastWeek:    16,
		Lessons: []timetable.Lesson{
			{Name: "高等数学", Periods: "1-2", Weekday: 4, Weeks: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				Teacher: "教师甲", Location: "教室A", WeeksText: "1-16周"},
		},
		Cookies: map[string]string{"bzb_jsxsd": "x", "SERVERID_jsxsd": "y"},
	}
	if err := timetable.SaveSnapshot(store, snap); err != nil {
		t.Fatal(err)
	}
}

// 课表查询必须 0 请求：用一个会「炸」的 HTTP 端口来证明它真的没被调用。
type panicHTTP struct{ t *testing.T }

func (p panicHTTP) Do(*http.Request) (*http.Response, error) {
	p.t.Fatal("课表查询路径不得发起网络请求（R1：快照有效时 0 请求）")
	return nil, nil
}

func TestGetScheduleIsOffline(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))
	deps, store, _ := testDeps(clock)
	deps.HTTP = panicHTTP{t: t}
	seedSnapshot(t, store)

	env, err := app.ScheduleEnvelope(deps, model.Date{Year: 2026, Month: 11, Day: 5})
	if err != nil {
		t.Fatalf("ScheduleEnvelope: %v", err)
	}
	if !env.OK {
		t.Error("应成功")
	}
	if env.Stale {
		t.Error("快照有效的正常查询 stale 必须为 false")
	}
	if env.SyncedAt == "" {
		t.Error("课表数据来自快照时必须给出 syncedAt")
	}
	day, ok := env.Data.(model.DaySchedule)
	if !ok {
		t.Fatalf("data 类型 = %T", env.Data)
	}
	if day.Week != 10 || len(day.Entries) != 1 {
		t.Errorf("day = %+v", day)
	}
}

func TestGetScheduleMissingSnapshotIs3003(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))
	deps, _, _ := testDeps(clock)
	deps.HTTP = panicHTTP{t: t}

	_, err := app.ScheduleEnvelope(deps, model.Date{Year: 2026, Month: 11, Day: 5})
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
}

func TestWeekScheduleCurrentWeek(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))
	deps, store, _ := testDeps(clock)
	seedSnapshot(t, store)

	days, week, err := app.WeekSchedule(deps, 0)
	if err != nil {
		t.Fatalf("WeekSchedule: %v", err)
	}
	if week != 10 {
		t.Errorf("week = %d, want 10", week)
	}
	if len(days) != 7 {
		t.Fatalf("days = %d, want 7", len(days))
	}
	if days[0].Weekday != 1 || days[6].Weekday != 7 {
		t.Errorf("周内顺序错误: %s..%s", days[0].Date, days[6].Date)
	}
}

func TestWeekScheduleOutOfSemesterIsNotAnError(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 7, 1, 8, 0, 0, 0, time.Local))
	deps, store, _ := testDeps(clock)
	seedSnapshot(t, store)

	// 显式指定越界周次：返回 7 天、全部 inSemester:false，不报错。
	days, week, err := app.WeekSchedule(deps, 30)
	if err != nil {
		t.Fatalf("越界周次不应报错: %v", err)
	}
	if week != 30 {
		t.Errorf("week = %d", week)
	}
	for _, d := range days {
		if d.InSemester {
			t.Errorf("%s 不应在学期内", d.Date)
		}
	}
}

// 当前日期不在学期内且未指定周次：明确报错，不猜一个周次。
func TestWeekScheduleCurrentWeekOutsideSemester(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 7, 1, 8, 0, 0, 0, time.Local))
	deps, store, _ := testDeps(clock)
	seedSnapshot(t, store)

	_, _, err := app.WeekSchedule(deps, 0)
	if err == nil {
		t.Fatal("假期里未指定周次应报错")
	}
	if model.CodeOf(err) != model.CodeNotInSemester {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeNotInSemester)
	}
}

func TestSyncTimetableWithoutCredentials(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))
	deps, _, _ := testDeps(clock)

	_, err := app.SyncTimetable(context.Background(), deps, model.Config{})
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
}

// CheckElectricity：两个请求任一失败都要整体报错，不返回半份数据。
func TestCheckElectricityFailsWholeReport(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))

	var bindOK = true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		var param struct {
			Cmd string `json:"cmd"`
		}
		_ = json.Unmarshal([]byte(r.Form.Get("param")), &param)
		switch param.Cmd {
		case "getbindroom":
			body := `{"result":"0","existflag":"1","roomfullname":"<校区>_<楼栋>_<楼层>_<房间>","roomverify":"<roomverify>","detaillist":[{"businesstype":"0","odd":"35.99","status":"1"}]}`
			writeOuter(w, body)
		case "getusedetail":
			if !bindOK {
				writeOuter(w, `{"result":"-10005","message":"查询失败"}`)
				return
			}
			writeOuter(w, `{"result":"0","yearmonth":"2026-11","monthuse":"63.13","opdtdate":"2026-11-04","dayuselist":[{"date":"2026-11-04","dayuse":11.75}]}`)
		}
	}))
	defer srv.Close()

	deps, _, _ := testDeps(clock)
	deps.HTTP = srv.Client()
	deps.ElectricityEndpoint = srv.URL
	cfg := model.Config{StudentID: testAccount, CustomerCode: testSchool}

	report, err := app.CheckElectricity(context.Background(), deps, cfg, "2026-11")
	if err != nil {
		t.Fatalf("成功路径: %v", err)
	}
	if report.Status.Balance != 35.99 || report.Usage.MonthUse != 63.13 {
		t.Errorf("report = %+v", report)
	}
	if report.Status.CollectedAt == "" {
		t.Error("必须标注数据截至时间")
	}

	bindOK = false
	if _, err := app.CheckElectricity(context.Background(), deps, cfg, "2026-11"); err == nil {
		t.Fatal("用量请求失败时必须整体报错，不能返回半份数据")
	}
}

func TestCheckElectricityMonthNormalization(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 1, 5, 8, 0, 0, 0, time.Local))
	deps, _, _ := testDeps(clock)
	deps.HTTP = panicHTTP{t: t}
	deps.ElectricityEndpoint = "http://example.invalid"

	// 月格式非法应报参数错误，且不触网。
	for _, m := range []string{"2026-13", "2026", "2026-1-1", "abc", "2026-"} {
		_, err := app.CheckElectricity(context.Background(), deps,
			model.Config{StudentID: testAccount, CustomerCode: testSchool}, m)
		if model.CodeOf(err) != model.CodeInvalidArgument {
			t.Errorf("月份 %q code = %d, want %d", m, model.CodeOf(err), model.CodeInvalidArgument)
		}
	}
}

func TestConfigRoundTripAndMasking(t *testing.T) {
	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 8, 0, 0, 0, time.Local))
	deps, _, _ := testDeps(clock)

	// 缺失配置不是错误。
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
	cfg.CustomerCode = "999999"
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

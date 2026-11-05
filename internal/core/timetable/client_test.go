package timetable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
)

const (
	testUser = "<学号>"
	testPass = "<密码>"
)

func fixedClock() *porttest.FakeClock {
	return porttest.NewFakeClock(time.Date(2026, 11, 5, 20, 0, 0, 0, time.Local))
}

// 实测锚点页：7 个 MM-DD 表格单元格。
const anchorHTML = `<table id="timetable"><thead><tr>
<th><span>周一<br /></span><span>08-31</span></th>
<th><span>周二<br /></span><span>09-01</span></th>
<th><span>周三<br /></span><span>09-02</span></th>
<th><span>周四<br /></span><span>09-03</span></th>
<th><span>周五<br /></span><span>09-04</span></th>
<th><span>周六<br /></span><span>09-05</span></th>
<th><span>周日<br /></span><span>09-06</span></th>
</tr></thead></table>`

// 实测课表 JSON：单条课程含两个时段。
const coursesJSON = `{"msg":"","code":0,"count":1,"data":[
 {"kc_mc":"课程甲","kch":"<课程号>","jg0101mc":"教师甲",
  "sktime":"周三第1、2节{2,4,7-10}全周;周四第1、2节{9}全周",
  "skddmc":"<地点>","skxqmc":"<校区>","kt_mc":"<教学班>"}
]}`

// newFakeSchool 起两个 httptest：cas 与 jsb，模拟实测跨主机链路。
type fakeSchool struct {
	cas *httptest.Server
	jsb *httptest.Server
}

func newFakeSchool(t *testing.T, opts fakeOpts) *fakeSchool {
	t.Helper()

	mux := http.NewServeMux()

	// jsb 侧：登录后所有教务接口；未登录时一律回登录页（模拟假 200）。
	jsbMux := http.NewServeMux()
	jsbMux.HandleFunc("/jsxsd/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ticket") != "" {
			// 实测：jsxsd 下发两个会话 Cookie，缺一不可。
			http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "sess-1"})
			http.SetCookie(w, &http.Cookie{Name: ServerCookie, Value: "srv-1"})
			http.Redirect(w, r, "/jsxsd/framework/xsMainV.htmlx", http.StatusFound)
			return
		}
		if hasSession(r) {
			// 已登录后访问 /jsxsd/ 在实测里是 404（只有子页面有效）。
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if !hasSession(r) {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	jsbMux.HandleFunc("/jsxsd/framework/xsMainV.htmlx", func(w http.ResponseWriter, r *http.Request) {
		if !hasSession(r) {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		fmt.Fprint(w, `<html><body>教务首页 课表</body></html>`)
	})
	// 登录后补的那一跳：service 首页会 404，但也要能下发 Cookie。
	jsbMux.HandleFunc("/jsxsd/index.htmlx", func(w http.ResponseWriter, r *http.Request) {
		if hasSession(r) {
			fmt.Fprint(w, `<html><body>教务首页 课表</body></html>`)
			return
		}
		fmt.Fprint(w, `<title>统一身份认证平台</title>`)
	})
	jsbMux.HandleFunc("/jsxsd/framework/mainV_index_loadkb.htmlx", func(w http.ResponseWriter, r *http.Request) {
		if !hasSession(r) {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		if r.URL.Query().Get("zc") != "1" {
			t.Errorf("锚点请求缺少 zc=1: %s", r.URL.RawQuery)
		}
		if opts.anchorBody != "" {
			fmt.Fprint(w, opts.anchorBody)
			return
		}
		fmt.Fprint(w, anchorHTML)
	})
	jsbMux.HandleFunc("/jsxsd/xskb/xskb_list.do", func(w http.ResponseWriter, r *http.Request) {
		if !hasSession(r) {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("课表接口应为 POST，收到 %s", r.Method)
		}
		_ = r.ParseForm()
		if r.Form.Get("viweType") != "1" || r.Form.Get("needData") != "1" {
			t.Errorf("课表表单缺少 viweType/needData: %v", r.Form)
		}
		if opts.coursesBody != "" {
			fmt.Fprint(w, opts.coursesBody)
			return
		}
		w.Header().Set("Content-Type", "text/html;charset=utf-8")
		fmt.Fprint(w, coursesJSON)
	})

	jsb := httptest.NewServer(jsbMux)
	t.Cleanup(jsb.Close)

	// cas 侧：登录页 + 登录 POST。登录成功的判定只看凭据。
	mux.HandleFunc("/cas_server/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `<title>统一身份认证平台</title>
<form id="form1">
<input type="hidden" id="__VIEWSTATE" value="vs-token" />
<input type="hidden" id="__VIEWSTATEGENERATOR" value="gen" />
<input type="hidden" id="__EVENTVALIDATION" value="ev-token" />
<input name="signupInputUserName" />
<input name="signupInputPassword" type="password" />
<input type="submit" name="btn_login" />
</form>`)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get("__VIEWSTATE") != "vs-token" {
			t.Errorf("登录表单未回传 __VIEWSTATE: %v", r.Form)
		}
		if r.Form.Get(CASFieldUser) != testUser || r.Form.Get(CASFieldPass) != testPass {
			// 凭据错误：CAS 原样返回登录页，HTTP 200（必须靠内容判定）。
			fmt.Fprint(w, `<title>统一身份认证平台</title><input name="signupInputUserName" />`)
			return
		}
		// 成功：302 到 service（jsb 的 /jsxsd/ + ticket）。
		http.Redirect(w, r, jsb.URL+"/jsxsd/?ticket=ST-1", http.StatusFound)
	})

	cas := httptest.NewServer(mux)
	t.Cleanup(cas.Close)

	return &fakeSchool{cas: cas, jsb: jsb}
}

type fakeOpts struct {
	anchorBody  string
	coursesBody string
}

func (f *fakeSchool) client() *Client {
	// 实测链路跨两个主机：CAS 302 到 jsb 并带上 ticket，jsb 再下发会话 Cookie。
	// 所以测试里必须用带 jar 的 client，否则第二个主机的 Set-Cookie 会被丢掉。
	// 不用 jar：本包自己维护会话 Cookie（快照要保存它），并手动跟随跨主机重定向。
	// 必须用 noRedirDoer，模拟 platform.HTTPDoer 的真实行为。
	return NewClient(noRedirDoer{Client: &http.Client{}}, fixedClock(), f.cas.URL, f.jsb.URL)
}

// 会话失效也返回 200：必须判响应内容（api-docs/001 §6.1）。
func TestSyncSessionExpiredReturns3002(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<title>统一身份认证平台</title><input name="signupInputUserName" />`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), fixedClock(), srv.URL, srv.URL)
	c.SetCookies(map[string]string{SessionCookie: "stale"})

	if _, err := c.Sync(context.Background()); err == nil {
		t.Fatal("会话失效应返回错误")
	} else if model.CodeOf(err) != model.CodeSessionExpired {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeSessionExpired)
	}
}

func TestIsAliveDetectsLoginPage(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), fixedClock(), srv.URL, srv.URL)
	c.SetCookies(map[string]string{SessionCookie: "x", ServerCookie: "y"})

	body = `<title>统一身份认证平台</title>`
	if alive, err := c.IsAlive(context.Background()); err != nil || alive {
		t.Errorf("登录页应判为失效: alive=%v err=%v", alive, err)
	}

	body = `<html><body>教务首页 课表</body></html>`
	if alive, err := c.IsAlive(context.Background()); err != nil || !alive {
		t.Errorf("教务内页应判为有效: alive=%v err=%v", alive, err)
	}
}

func TestIsAliveWithoutCookiesIsFalse(t *testing.T) {
	c := NewClient(http.DefaultClient, fixedClock(), "http://example.invalid", "http://example.invalid")
	alive, err := c.IsAlive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if alive {
		t.Error("没有会话 Cookie 时不应判为有效")
	}
}

// 完整同步：CAS 登录 → 锚点 → 课表 JSON。
func TestSyncHappyPath(t *testing.T) {
	school := newFakeSchool(t, fakeOpts{})
	c := school.client()

	if err := c.Login(context.Background(), testUser, testPass); err != nil {
		t.Fatalf("Login: %v (cookies=%v)", err, c.sessionCookieNames())
	}
	if !c.HasSession() {
		t.Fatalf("登录后应拿到会话 Cookie, got %v", c.sessionCookieNames())
	}

	snap, err := c.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if snap.Version != SnapshotVersion {
		t.Errorf("Version = %d", snap.Version)
	}
	if snap.Week1Monday.String() != "2026-08-31" {
		t.Errorf("Week1Monday = %s", snap.Week1Monday)
	}
	if snap.LastWeek != 10 {
		t.Errorf("LastWeek = %d, want 10", snap.LastWeek)
	}
	// 一条课程两个时段 → 两条 Lesson。
	if len(snap.Lessons) != 2 {
		t.Fatalf("Lessons = %+v", snap.Lessons)
	}
	if len(snap.SlotTimes) != 6 || snap.SlotTimes[0] != "08:30~10:00" {
		t.Errorf("SlotTimes = %v", snap.SlotTimes)
	}
	if snap.Cookies[SessionCookie] != "sess-1" {
		t.Errorf("快照应保存会话 Cookie, got %v", snap.Cookies)
	}
	if !snap.SyncedAt.Equal(fixedClock().Now()) {
		t.Errorf("SyncedAt 应来自注入时钟: %v", snap.SyncedAt)
	}

	// 快照可直接用于查询，0 请求。
	day := Query(snap, model.Date{Year: 2026, Month: 11, Day: 5}) // 第 10 周周四
	if day.Week != 10 {
		t.Errorf("week = %d, want 10", day.Week)
	}
	// 周四第 1、2 节只上第 9 周 → 第 10 周周四没课。
	if len(day.Entries) != 0 {
		t.Errorf("第 10 周周四不应有课: %+v", day.Entries)
	}

	wed := Query(snap, model.Date{Year: 2026, Month: 11, Day: 4}) // 第 10 周周三
	if wed.Week != 10 {
		t.Fatalf("周三 week = %d", wed.Week)
	}
	if len(wed.Entries) != 1 || wed.Entries[0].Name != "课程甲" {
		t.Fatalf("第 10 周周三 entries = %+v", wed.Entries)
	}
	if wed.Entries[0].Time != "08:30~10:00" {
		t.Errorf("time = %q", wed.Entries[0].Time)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	school := newFakeSchool(t, fakeOpts{})
	c := school.client()

	err := c.Login(context.Background(), testUser, "wrong")
	if err == nil {
		t.Fatal("错误密码应失败")
	}
	if model.CodeOf(err) != model.CodeUpstreamRejected {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeUpstreamRejected)
	}
	if strings.Contains(err.Error(), testPass) {
		t.Error("错误信息不得包含密码")
	}
}

func TestLoginRequiresCredentials(t *testing.T) {
	c := NewClient(http.DefaultClient, fixedClock(), "http://example.invalid", "http://example.invalid")
	if err := c.Login(context.Background(), "", ""); model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
}

func TestLoginNetworkFailureIs4001(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	base := srv.URL
	srv.Close() // 立刻关掉，制造连接失败

	c := NewClient(http.DefaultClient, fixedClock(), base, base)
	err := c.Login(context.Background(), testUser, testPass)
	if model.CodeOf(err) != model.CodeNetworkDown {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeNetworkDown)
	}
}

// 锚点页缺少 7 个日期时必须报 4002，不能产出空课表。
func TestSyncMalformedAnchor(t *testing.T) {
	school := newFakeSchool(t, fakeOpts{anchorBody: `<html>只有节次没有日期</html>`})
	c := school.client()
	c.SetCookies(map[string]string{SessionCookie: "sess-1", ServerCookie: "srv-1"})

	_, err := c.Sync(context.Background())
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeMalformedResponse)
	}
}

func TestParseCourseListBusinessCode(t *testing.T) {
	_, err := ParseCourseList([]byte(`{"code":1,"msg":"无权限","data":[]}`))
	if model.CodeOf(err) != model.CodeBusinessFailed {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeBusinessFailed)
	}
}

// ParseCourseList 只负责结构：能读出数组就算通过，字段漂移由 Sync 判定。
// 这里断言的是「结构坏了必须报错」。
func TestParseCourseListStructureErrors(t *testing.T) {
	_, err := ParseCourseList([]byte(`not json`))
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("非法 JSON code = %d", model.CodeOf(err))
	}

	_, err = ParseCourseList([]byte(`{"code":0,"nothing":[]}`))
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("缺少课程数组 code = %d", model.CodeOf(err))
	}
}

// 字段名全部漂移时，Sync 必须报 4002，而不是产出一份空课表。
func TestSyncFieldDriftIsError(t *testing.T) {
	school := newFakeSchool(t, fakeOpts{
		coursesBody: `{"code":0,"data":[{"a":"1","b":"2"}]}`,
	})
	c := school.client()
	c.SetCookies(map[string]string{SessionCookie: "sess-1", ServerCookie: "srv-1"})

	_, err := c.Sync(context.Background())
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeMalformedResponse)
	}
}

func TestParseCourseListEmptyIsOK(t *testing.T) {
	rows, err := ParseCourseList([]byte(`{"code":0,"data":[]}`))
	if err != nil {
		t.Fatalf("空课程数组不应报错: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %+v", rows)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	store := porttest.NewMemStore()
	snap := testSnapshot()
	snap.SyncedAt = fixedClock().Now()
	snap.Cookies = map[string]string{SessionCookie: "a"}

	if err := SaveSnapshot(store, snap); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSnapshot(store)
	if err != nil {
		t.Fatal(err)
	}
	if got.Week1Monday != snap.Week1Monday || got.LastWeek != snap.LastWeek {
		t.Errorf("round trip 不一致: %+v", got)
	}
	if len(got.Lessons) != len(snap.Lessons) {
		t.Errorf("lessons = %d, want %d", len(got.Lessons), len(snap.Lessons))
	}
}

func TestLoadSnapshotMissing(t *testing.T) {
	_, err := LoadSnapshot(porttest.NewMemStore())
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
}

// 版本不匹配：丢弃重拉，不尝试迁移（AGENTS §4.3）。
func TestLoadSnapshotVersionMismatch(t *testing.T) {
	store := porttest.NewMemStore()
	snap := testSnapshot()
	snap.Version = SnapshotVersion + 1
	snap.SyncedAt = fixedClock().Now()
	if err := SaveSnapshot(store, snap); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSnapshot(store)
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
}

func TestLoadSnapshotCorrupt(t *testing.T) {
	store := porttest.NewMemStore()
	if err := store.Put(SnapshotStoreKey, []byte("not json")); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSnapshot(store)
	if model.CodeOf(err) != model.CodeSnapshotMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeSnapshotMissing)
	}
}

// Cookie 值里带 ";path=/" 尾巴时只取分号前的值。
func TestAbsorbCookiesStripsPathAttribute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", SessionCookie+"=abc123;path=/;HttpOnly")
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), fixedClock(), srv.URL, srv.URL)
	if _, err := c.get(context.Background(), srv.URL, "/"); err != nil {
		t.Fatal(err)
	}
	if got := c.cookies[SessionCookie]; got != "abc123" {
		t.Errorf("cookie = %q, want abc123", got)
	}
}

// noRedirDoer 模拟 platform.HTTPDoer：提供 Do 与 DoNoRedirect。
type noRedirDoer struct{ Client *http.Client }

func (d noRedirDoer) Do(req *http.Request) (*http.Response, error) { return d.Client.Do(req) }

func (d noRedirDoer) DoNoRedirect(req *http.Request) (*http.Response, error) {
	c := *d.Client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c.Do(req)
}

// hasSession 判断请求是否带齐两个会话 Cookie。
func hasSession(r *http.Request) bool {
	a, err1 := r.Cookie(SessionCookie)
	b, err2 := r.Cookie(ServerCookie)
	return err1 == nil && err2 == nil && a.Value != "" && b.Value != ""
}

var _ = json.Marshal
var _ = url.Values{}

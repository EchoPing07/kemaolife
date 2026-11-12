package platform

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// 请求记录是排查用能力：默认关闭、只存内存、内容原样保留。
// 以下用例全部走本地 httptest，不碰真实学校接口（AGENTS §7）。

func fixedClock() func() time.Time {
	base := time.Date(2026, 11, 11, 19, 11, 40, 490_000_000, time.Local)
	var mu sync.Mutex
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		base = base.Add(20 * time.Millisecond)
		return base
	}
}

func TestRequestLogDisabledByDefault(t *testing.T) {
	l := NewRequestLog(fixedClock())
	if l.Enabled() {
		t.Fatal("请求记录必须默认关闭（docs/002 §5.3）")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	// 关闭时包一层也不能记录：RoundTrip 直通。
	client := &http.Client{Transport: l.Wrap(http.DefaultTransport)}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := l.Count(); got != 0 {
		t.Fatalf("关闭状态下记录了 %d 条", got)
	}
}

func TestRequestLogCapturesFullExchange(t *testing.T) {
	l := NewRequestLog(fixedClock())
	l.SetEnabled(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"result":"0"}`)
	}))
	defer srv.Close()

	client := &http.Client{Transport: l.Wrap(http.DefaultTransport)}
	resp, err := client.Post(srv.URL+"/api/test", "application/x-www-form-urlencoded",
		strings.NewReader("cmd=getbindroom&page=1"))
	if err != nil {
		t.Fatal(err)
	}
	// 记录不能吃掉响应正文：调用方还得能正常读。
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"result":"0"}` {
		t.Fatalf("记录改动了响应正文: %q", body)
	}

	list := l.Summaries(0)
	if len(list) != 1 {
		t.Fatalf("条目数 = %d, want 1", len(list))
	}
	if list[0].Method != http.MethodPost || list[0].Status != 200 {
		t.Errorf("摘要 = %+v", list[0])
	}

	entry, found := l.Find(list[0].Seq)
	if !found {
		t.Fatal("找不到刚记录的条目")
	}
	for _, want := range []string{"cmd=getbindroom", `{"result":"0"}`, "Content-Type"} {
		if !strings.Contains(entry.RequestBody+entry.ResponseBody+entry.RequestHeaders, want) {
			t.Errorf("缺少 %q: %+v", want, entry)
		}
	}
	if !strings.Contains(entry.ResponseHeaders, "Content-Type: application/json") {
		t.Errorf("响应标头 = %q", entry.ResponseHeaders)
	}
}

// 记录的是完整原始报文：标头（含 Cookie）、正文（含表单 / JSON 字段）都原样保留。
// 隐私约束落在「不入仓」这一侧：代码与测试夹具一律用占位符（P1 / P3）。
func TestRequestLogKeepsRawPayload(t *testing.T) {
	l := NewRequestLog(fixedClock())
	l.SetEnabled(true)

	const (
		placeholderCookie = "session=<session-value>"
		placeholderBody   = "signupInputUserName=<学号>&signupInputPassword=<密码>"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cookie 值必须是 ASCII，否则 net/http 会丢弃它（与本用例意图无关）。
		http.SetCookie(w, &http.Cookie{Name: "bzb_jsxsd", Value: "session-value"})
		_, _ = io.WriteString(w, `{"result":"0","roomverify":"<roomverify>"}`)
	}))
	defer srv.Close()

	client := &http.Client{Transport: l.Wrap(http.DefaultTransport)}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/login?ticket=ST-1",
		strings.NewReader(placeholderBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Cookie", placeholderCookie)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	entry, found := l.Find(l.Summaries(1)[0].Seq)
	if !found {
		t.Fatal("找不到刚记录的条目")
	}
	// 原始值必须在：这是「完整请求数据」的全部意义。
	for _, want := range []string{
		placeholderBody, placeholderCookie, "<roomverify>", "ST-1", "bzb_jsxsd=session-value",
	} {
		all := entry.URL + entry.RequestHeaders + entry.RequestBody + entry.ResponseHeaders + entry.ResponseBody
		if !strings.Contains(all, want) {
			t.Errorf("记录缺少原始内容 %q:\n%s", want, all)
		}
	}
}

// 记录必须按「新的在前」返回，且环形窗口只留最近 max 条。
func TestRequestLogKeepsNewestFirst(t *testing.T) {
	l := NewRequestLog(fixedClock())
	l.SetEnabled(true)
	l.max = 3

	for i := 0; i < 5; i++ {
		l.append(LogEntry{At: "2026-11-11T19:11:40", Method: "GET", URL: "http://example.test/"})
	}
	list := l.Summaries(0)
	if len(list) != 3 {
		t.Fatalf("条目数 = %d, want 3", len(list))
	}
	if !(list[0].Seq > list[1].Seq && list[1].Seq > list[2].Seq) {
		t.Errorf("顺序不是新的在前: %+v", list)
	}
	if list[0].Seq != 5 {
		t.Errorf("最新 seq = %d, want 5", list[0].Seq)
	}
}

// 关闭记录要立刻清空：否则用户以为关掉就没了，实际正文还留在内存里。
func TestRequestLogDisableClears(t *testing.T) {
	l := NewRequestLog(fixedClock())
	l.SetEnabled(true)
	l.append(LogEntry{URL: "http://example.test/"})
	if l.Count() != 1 {
		t.Fatalf("条目数 = %d", l.Count())
	}
	l.SetEnabled(false)
	if l.Count() != 0 {
		t.Fatalf("关闭后仍留 %d 条", l.Count())
	}
}

// 超长正文要截断并标记，避免一次排查把几十 MB 的响应留在内存里。
func TestRequestLogTruncatesLongBodies(t *testing.T) {
	l := NewRequestLog(fixedClock())
	l.bodyCap = 16
	got := l.clip(strings.Repeat("a", 100))
	if !strings.HasSuffix(got, logTruncatedMark) {
		t.Fatalf("缺少截断标记: %q", got)
	}
	if len(got) > l.bodyCap+len(logTruncatedMark) {
		t.Fatalf("截断后仍过长: %d", len(got))
	}
}

// 网络层失败也要留一条记录：此时没有任何响应元数据，只有错误文本与耗时。
func TestRequestLogRecordsTransportError(t *testing.T) {
	l := NewRequestLog(fixedClock())
	l.SetEnabled(true)

	client := &http.Client{
		Transport: l.Wrap(&failingTransport{err: errString("dial tcp: 连接超时")}),
	}
	if _, err := client.Get("http://example.test/api/test"); err == nil {
		t.Fatal("预期失败")
	}
	list := l.Summaries(0)
	if len(list) != 1 || list[0].Error == "" {
		t.Fatalf("失败也要留档: %+v", list)
	}
}

type failingTransport struct{ err error }

func (f *failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

type errString string

func (e errString) Error() string { return string(e) }

// httpDoer 安装 / 卸载记录器时不能把 Transport 套成多层，也要重建 no-redirect 副本。
func TestHTTPDoerSetRequestLogRebuildsClients(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	doer := NewHTTPDoer(5 * time.Second)
	l := NewRequestLog(fixedClock())
	doer.SetRequestLog(l)
	l.SetEnabled(true)

	resp, err := doer.Do(mustRequest(t, srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if l.Count() != 1 {
		t.Fatalf("Do 未记录: %d", l.Count())
	}

	// no-redirect 也要记录（CAS 逐跳吸收 Cookie 走的是这条）。
	l.Clear()
	resp, err = doer.DoNoRedirect(mustRequest(t, srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if l.Count() != 1 {
		t.Fatalf("DoNoRedirect 未记录: %d", l.Count())
	}

	if doer.reqLog == nil {
		t.Error("SetRequestLog 应保留记录器引用")
	}
}

func mustRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

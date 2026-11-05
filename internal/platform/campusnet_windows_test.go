//go:build windows

package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// probeFixture 是一个模拟 BRAS 探测端点的 httptest 服务器。
//
// 真实探测地址是 http://5.5.5.5/（内网固定 IP），测试里用 httptest 替代，
// 通过覆盖 client 的 Transport 把请求重定向到本地。
type probeFixture struct {
	srv      *httptest.Server
	location string // 要返回的 302 Location
	status   int    // 要返回的状态码
	calls    int
}

func newProbeFixture(t *testing.T, status int, location string) *probeFixture {
	t.Helper()
	f := &probeFixture{status: status, location: location}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls++
		if f.location != "" {
			w.Header().Set("Location", f.location)
		}
		w.WriteHeader(f.status)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// newTestPortal 构造一个把探测 / Portal 请求都指向本地 fixture 的适配器。
func newTestPortal(t *testing.T, cred PortalCredential, probe *httptest.Server, portal *httptest.Server) *PortalNetwork {
	t.Helper()
	p := NewPortalNetwork(func() PortalCredential { return cred })
	// 立即生效的等待：真实实现要等 3s，测试里不能真的睡。
	p.sleep = func(context.Context, time.Duration) error { return nil }

	probeURL, err := url.Parse(probe.URL)
	if err != nil {
		t.Fatal(err)
	}
	portalURL, err := url.Parse(portal.URL)
	if err != nil {
		t.Fatal(err)
	}

	// 用自定义 Transport 把固定地址改写到 fixture。
	p.probe = &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: rewriteTransport{base: probeURL},
	}
	p.portal = &http.Client{
		Timeout:   2 * time.Second,
		Transport: rewriteTransport{base: portalURL},
	}
	// 把 Portal 基地址指到 fixture。
	p.base = portal.URL
	return p
}

// rewriteTransport 把所有请求改写到 fixture 主机（保留 path 与 body）。
type rewriteTransport struct {
	base *url.URL
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = r.base.Scheme
	clone.URL.Host = r.base.Host
	clone.Host = r.base.Host
	return http.DefaultTransport.RoundTrip(clone)
}

const (
	fxOnlineLoc = "https://10.0.128.3/portal/webauth?wlanacip=<BRAS地址>&wlanacname=km-bras" +
		"&wlanuserip=<内网IP>&mac=aa:bb:cc:dd:ee:ff&vlan=0&act=LOGINSUCC&distoken=<distoken>&url=http://5.5.5.5"
	fxUnauthLoc = "https://10.0.128.3/portal/webauth?wlanacip=<BRAS地址>&wlanacname=km-bras" +
		"&wlanuserip=<内网IP>&mac=aa:bb:cc:dd:ee:ff&vlan=0&url=http://5.5.5.5"
)

func testCred() PortalCredential {
	return PortalCredential{UserID: "<学号>", Password: "<密码>", Fingerprint: "<指纹>"}
}

// 探测失败（连不上）归 offline，不返回错误（docs/002 §3.3）。
func TestPortalStatusOfflineOnProbeFailure(t *testing.T) {
	// 关闭的 fixture：请求必然失败。
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	p := NewPortalNetwork(func() PortalCredential { return testCred() })
	p.sleep = func(context.Context, time.Duration) error { return nil }
	u, _ := url.Parse(deadURL)
	p.probe = &http.Client{Timeout: time.Second, Transport: rewriteTransport{base: u}}

	status, err := p.Status(context.Background())
	if err != nil {
		t.Fatalf("探测失败不应返回错误: %v", err)
	}
	if status.State != model.CampusOffline {
		t.Errorf("state = %q, want offline", status.State)
	}
}

// 三态判定。
func TestPortalStatusStates(t *testing.T) {
	portalSrv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer portalSrv.Close()

	cases := []struct {
		name   string
		status int
		loc    string
		want   model.CampusState
	}{
		{"已在线", 302, fxOnlineLoc, model.CampusOnline},
		{"未认证", 302, fxUnauthLoc, model.CampusUnauth},
		{"200 无重定向", 200, "", model.CampusOffline},
		{"302 但跳到别处", 302, "https://example.invalid/x", model.CampusOffline},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fx := newProbeFixture(t, c.status, c.loc)
			p := newTestPortal(t, testCred(), fx.srv, portalSrv)
			status, err := p.Status(context.Background())
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if status.State != c.want {
				t.Errorf("state = %q, want %q", status.State, c.want)
			}
		})
	}
}

// 已在线时 Login 直接返回，不发 Portal 请求（不重复认证）。
func TestPortalLoginSkipsWhenOnline(t *testing.T) {
	portalHits := 0
	portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		portalHits++
		w.WriteHeader(http.StatusOK)
	}))
	defer portalSrv.Close()

	fx := newProbeFixture(t, 302, fxOnlineLoc)
	p := newTestPortal(t, testCred(), fx.srv, portalSrv)

	if err := p.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if portalHits != 0 {
		t.Errorf("已在线时不应发认证请求, got %d", portalHits)
	}
}

// 不在校园网时动作直接返回 4001，不发认证请求。
func TestPortalLoginOfflineReturnsNetworkDown(t *testing.T) {
	portalHits := 0
	portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		portalHits++
	}))
	defer portalSrv.Close()

	fx := newProbeFixture(t, 200, "")
	p := newTestPortal(t, testCred(), fx.srv, portalSrv)

	err := p.Login(context.Background())
	if err == nil {
		t.Fatal("不在校园网应报错")
	}
	if model.CodeOf(err) != model.CodeNetworkDown {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeNetworkDown)
	}
	if portalHits != 0 {
		t.Errorf("不应发认证请求, got %d", portalHits)
	}
}

// 缺校园网凭据：报 3001，而不是拿空账号去认证。
func TestPortalLoginWithoutCredential(t *testing.T) {
	portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer portalSrv.Close()

	fx := newProbeFixture(t, 302, fxUnauthLoc)
	p := newTestPortal(t, PortalCredential{}, fx.srv, portalSrv)

	err := p.Login(context.Background())
	if err == nil {
		t.Fatal("缺凭据应报错")
	}
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
}

// 登录成功路径：发 11 字段表单，然后复探确认在线。
func TestPortalLoginSuccess(t *testing.T) {
	var gotForm url.Values
	portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		_, _ = w.Write([]byte(`{"success":true,"message":"正在进行外网拨号请稍候...","code":200}`))
	}))
	defer portalSrv.Close()

	// 探测第一次给未认证，登录后复探给在线。
	probeCalls := 0
	probeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probeCalls++
		if probeCalls == 1 {
			w.Header().Set("Location", fxUnauthLoc)
		} else {
			w.Header().Set("Location", fxOnlineLoc)
		}
		w.WriteHeader(302)
	}))
	defer probeSrv.Close()

	p := newTestPortal(t, testCred(), probeSrv, portalSrv)
	if err := p.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}

	if gotForm.Get("userId") != "<学号>" || gotForm.Get("passwd") != "<密码>" {
		t.Errorf("表单账号密码不对: %v", gotForm)
	}
	if gotForm.Get("loginMode") != "password" || gotForm.Get("recordTerminalMac") != "Y" {
		t.Errorf("常量字段不对: %v", gotForm)
	}
	if gotForm.Get("mac") != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("登录 mac 应保持小写原样, got %q", gotForm.Get("mac"))
	}
	if gotForm.Get("wlanacname") != "km-bras" {
		t.Errorf("六参数未回传: %v", gotForm)
	}
	if probeCalls < 2 {
		t.Errorf("登录后应复探确认, probeCalls = %d", probeCalls)
	}
}

// 上游拒绝：文案必须脱敏后进入 error.message。
func TestPortalLoginRejectedSanitizesMessage(t *testing.T) {
	portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 故意回显账号，验证脱敏。
		_, _ = w.Write([]byte(`{"success":false,"message":"账号 <学号> 或密码错误","code":500}`))
	}))
	defer portalSrv.Close()

	fx := newProbeFixture(t, 302, fxUnauthLoc)
	p := newTestPortal(t, testCred(), fx.srv, portalSrv)

	err := p.Login(context.Background())
	if err == nil {
		t.Fatal("应报错")
	}
	if model.CodeOf(err) != model.CodeUpstreamRejected {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeUpstreamRejected)
	}
	msg := model.MessageOf(err)
	if strings.Contains(msg, "<学号>") {
		t.Errorf("错误文案未脱敏: %q", msg)
	}
	// 回显了账号的文案整体不可信，必须退化成固定短语（P2）。
	if msg != "认证被拒绝" {
		t.Errorf("含敏感内容的文案应整体替换, got %q", msg)
	}
}

// 下线：mac 转大写、distoken 来自本次探测。
func TestPortalLogoutFormShape(t *testing.T) {
	var gotForm url.Values
	portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		_, _ = w.Write([]byte(`{"success":true,"message":"下线成功","code":200}`))
	}))
	defer portalSrv.Close()

	// 第一次探测给在线（供下线取 distoken），之后给未认证（说明已下线）。
	probeCalls := 0
	probeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probeCalls++
		if probeCalls == 1 {
			w.Header().Set("Location", fxOnlineLoc)
		} else {
			w.Header().Set("Location", fxUnauthLoc)
		}
		w.WriteHeader(302)
	}))
	defer probeSrv.Close()

	p := newTestPortal(t, testCred(), probeSrv, portalSrv)
	// 复探看到「未认证」正是下线生效的表现，因此 Logout 应返回成功。
	if err := p.Logout(context.Background()); err != nil {
		t.Fatalf("Logout 应成功（复探已变未认证）: %v", err)
	}
	if gotForm == nil {
		t.Fatal("未收到下线请求")
	}

	if gotForm.Get("mac") != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("下线 mac 应转大写, got %q", gotForm.Get("mac"))
	}
	if gotForm.Get("distoken") != "<distoken>" {
		t.Errorf("distoken = %q", gotForm.Get("distoken"))
	}
	for _, k := range []string{"pageid", "act", "ssid"} {
		if _, ok := gotForm[k]; !ok {
			t.Errorf("应包含空字段 %q", k)
		}
	}
}

// EnsureCampusNetwork：已在线不发请求；未认证则登录。
func TestPortalEnsureCampusNetwork(t *testing.T) {
	t.Run("已在线不动作", func(t *testing.T) {
		portalHits := 0
		portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			portalHits++
		}))
		defer portalSrv.Close()

		fx := newProbeFixture(t, 302, fxOnlineLoc)
		p := newTestPortal(t, testCred(), fx.srv, portalSrv)
		if err := p.EnsureCampusNetwork(context.Background()); err != nil {
			t.Fatalf("EnsureCampusNetwork: %v", err)
		}
		if portalHits != 0 {
			t.Errorf("已在线不应发认证请求, got %d", portalHits)
		}
	})

	t.Run("不在校园网返回4001", func(t *testing.T) {
		portalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer portalSrv.Close()

		fx := newProbeFixture(t, 200, "")
		p := newTestPortal(t, testCred(), fx.srv, portalSrv)
		err := p.EnsureCampusNetwork(context.Background())
		if model.CodeOf(err) != model.CodeNetworkDown {
			t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeNetworkDown)
		}
	})
}

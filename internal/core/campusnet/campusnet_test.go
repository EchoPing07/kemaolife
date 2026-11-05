package campusnet_test

import (
	"net/url"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/campusnet"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// 探测 302 的 Location（占位值：BRAS 名称与门户 IP 属学校事实，本机 IP / MAC 用占位符）。
const (
	portalHost = "10.0.128.3"
	// onlineLocation 是已在线：带 act=LOGINSUCC 与新鲜的 distoken。
	onlineLocation = "https://" + portalHost + "/portal/webauth" +
		"?wlanacip=<BRAS地址>&wlanacname=km-bras&wlanuserip=<内网IP>&mac=<小写MAC>" +
		"&vlan=0&act=LOGINSUCC&distoken=<distoken>&url=http://5.5.5.5"
	// unauthLocation 是未认证：跳 webauth 但没有 act，也没有 distoken。
	unauthLocation = "https://" + portalHost + "/portal/webauth" +
		"?wlanacip=<BRAS地址>&wlanacname=km-bras&wlanuserip=<内网IP>&mac=<小写MAC>" +
		"&vlan=0&url=http://5.5.5.5"
)

func TestParseProbeOnline(t *testing.T) {
	p := campusnet.ParseProbe(302, onlineLocation)
	if p.State != model.CampusOnline {
		t.Fatalf("state = %q, want online", p.State)
	}
	if p.Distoken != "<distoken>" {
		t.Errorf("distoken = %q，下线需要它", p.Distoken)
	}
	if p.Params.WlanAcName != "km-bras" || p.Params.Mac != "<小写MAC>" {
		t.Errorf("六参数未正确抽取: %+v", p.Params)
	}
}

func TestParseProbeUnauth(t *testing.T) {
	p := campusnet.ParseProbe(302, unauthLocation)
	if p.State != model.CampusUnauth {
		t.Fatalf("state = %q, want unauth", p.State)
	}
	if p.Distoken != "" {
		t.Errorf("未认证态不应有 distoken: %q", p.Distoken)
	}
}

// 判态失败的各种形态都归 offline，且不是错误（docs/002 §3.3）。
func TestParseProbeOffline(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		location string
	}{
		{"非 302", 200, onlineLocation},
		{"无 Location", 302, ""},
		{"Location 不是 Portal 路径", 302, "https://example.invalid/login"},
		{"Location 无法解析", 302, "://bad"},
		{"跳转到别的 Portal 路径", 302, "https://" + portalHost + "/portal/other?x=1"},
	}
	for _, c := range cases {
		if got := campusnet.ParseProbe(c.status, c.location).State; got != model.CampusOffline {
			t.Errorf("%s: state = %q, want offline", c.name, got)
		}
	}
}

// 已在线态的 errorMsg 只是历史文案，不参与判态。
func TestParseProbeKeepsHistoryMessageWithoutChangingState(t *testing.T) {
	loc := "https://" + portalHost + "/portal/webauth?wlanacip=x&wlanacname=km-bras" +
		"&wlanuserip=y&mac=aa:bb:cc:dd:ee:ff&vlan=0&act=LOGINSUCC&errorMsg=<历史提示>"
	p := campusnet.ParseProbe(302, loc)
	if p.State != model.CampusOnline {
		t.Fatalf("state = %q, want online", p.State)
	}
	if p.ErrorMsg != "<历史提示>" {
		t.Errorf("errorMsg = %q", p.ErrorMsg)
	}
}

// 登录表单：11 个字段，与 api-docs/003 §2 ② 一一对应。
func TestBuildLoginForm(t *testing.T) {
	p := campusnet.ParseProbe(302, unauthLocation).Params
	form := campusnet.BuildLoginForm(p, "<学号>", "<密码>", "<指纹>")

	wantKeys := []string{
		"wlanacip", "wlanacname", "wlanuserip", "mac", "vlan", "url",
		"userId", "passwd", "loginMode", "recordTerminalMac", "browserFingerprint",
	}
	if len(form) != len(wantKeys) {
		t.Errorf("字段数 = %d, want %d: %v", len(form), len(wantKeys), form)
	}
	for _, k := range wantKeys {
		if _, ok := form[k]; !ok {
			t.Errorf("缺少字段 %q", k)
		}
	}
	if form.Get("userId") != "<学号>" || form.Get("passwd") != "<密码>" {
		t.Error("账号密码未填入")
	}
	if form.Get("loginMode") != "password" || form.Get("recordTerminalMac") != "Y" {
		t.Error("常量字段不符合协议")
	}
	// 登录时 mac 保持下发的小写形态。
	if form.Get("mac") != "<小写MAC>" {
		t.Errorf("登录 mac = %q, 应为小写原样", form.Get("mac"))
	}
	// 六参数原样回传。
	if form.Get("wlanacname") != p.WlanAcName || form.Get("url") != p.URL {
		t.Error("六参数未原样回传")
	}
}

// 下线表单：mac 转大写、distoken 来自本次探测、pageid/act/ssid 为空串。
func TestBuildLogoutForm(t *testing.T) {
	p := campusnet.ParseProbe(302, onlineLocation).Params
	form := campusnet.BuildLogoutForm(p, "<distoken>")

	if got := form.Get("mac"); got != "<小写MAC>" {
		// 占位符本身没有字母大小写差异，这里改用真实形态的 MAC 再验一次。
		t.Logf("占位符 MAC 无大小写差异: %q", got)
	}
	if got := form.Get("distoken"); got != "<distoken>" {
		t.Errorf("distoken = %q", got)
	}
	for _, k := range []string{"pageid", "act", "ssid"} {
		if v, ok := form[k]; !ok || v[0] != "" {
			t.Errorf("%s 应为空串字段, got %v", k, v)
		}
	}

	// 用真实形态的 MAC 验证大写转换。
	p.Mac = "aa:bb:cc:dd:ee:ff"
	if got := campusnet.BuildLogoutForm(p, "<distoken>").Get("mac"); got != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("下线 mac 应转大写, got %q", got)
	}
}

// 表单必须能被 url.Values.Encode 编码（含空值字段）。
func TestFormsAreEncodable(t *testing.T) {
	p := campusnet.ParseProbe(302, onlineLocation).Params
	for name, form := range map[string]url.Values{
		"login":  campusnet.BuildLoginForm(p, "<学号>", "<密码>", "<指纹>"),
		"logout": campusnet.BuildLogoutForm(p, "<distoken>"),
	} {
		if form.Encode() == "" {
			t.Errorf("%s 表单编码为空", name)
		}
	}
}

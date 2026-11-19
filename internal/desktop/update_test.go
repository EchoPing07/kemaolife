package desktop_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/porttest"
	"github.com/EchoPing07/kemaolife/internal/desktop"
)

// updateDoer 是本地 fake：记录请求、返回罐头响应，不触网（AGENTS §7）。
type updateDoer struct {
	resp   *http.Response
	err    error
	gotReq *http.Request
}

func (d *updateDoer) Do(req *http.Request) (*http.Response, error) {
	d.gotReq = req
	return d.resp, d.err
}

func updateResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestCheckUpdateDetectsNewerRelease(t *testing.T) {
	d := &updateDoer{resp: updateResponse(200, `{"tag_name":"v0.6.0","name":"0.6.0"}`)}
	info, err := desktop.CheckUpdate(context.Background(), d, "0.5.5")
	if err != nil {
		t.Fatalf("CheckUpdate: %v", err)
	}
	if !info.HasUpdate || info.Latest != "0.6.0" {
		t.Errorf("info = %+v，期望 HasUpdate=true Latest=0.6.0", info)
	}
	req := d.gotReq
	if req == nil {
		t.Fatal("未发出请求")
	}
	if req.URL.String() != "https://api.github.com/repos/EchoPing07/kemaolife/releases/latest" {
		t.Errorf("请求地址异常: %s", req.URL)
	}
	if req.Method != http.MethodGet {
		t.Errorf("方法应为 GET, got %s", req.Method)
	}
	if req.Header.Get("Accept") != "application/vnd.github+json" ||
		req.Header.Get("User-Agent") != "kemaolife-desktop" {
		t.Errorf("请求头异常: %v", req.Header)
	}
	// 公开查询不得携带任何凭据 / 会话。
	if req.Header.Get("Cookie") != "" || req.Header.Get("Authorization") != "" {
		t.Errorf("更新检查不得携带凭据: %v", req.Header)
	}
}

func TestCheckUpdateNoUpdateWhenSameOrOlder(t *testing.T) {
	for _, tag := range []string{"v0.5.5", "0.5.4"} {
		d := &updateDoer{resp: updateResponse(200, `{"tag_name":"`+tag+`"}`)}
		info, err := desktop.CheckUpdate(context.Background(), d, "0.5.5")
		if err != nil {
			t.Fatalf("tag %s: %v", tag, err)
		}
		if info.HasUpdate {
			t.Errorf("tag %s 不应判为有更新", tag)
		}
	}
}

func TestCheckUpdateErrorsAreSanitized(t *testing.T) {
	cases := []struct {
		name   string
		doer   *updateDoer
		leak   string
		status string
	}{
		{
			name: "非200",
			doer: &updateDoer{resp: updateResponse(403, `{"message":"rate limit exceeded for <学号>"}`)},
			leak: "rate limit",
		},
		{
			name: "非法JSON",
			doer: &updateDoer{resp: updateResponse(200, `<html>上游异常正文</html>`)},
			leak: "上游异常正文",
		},
		{
			name: "非法tag",
			doer: &updateDoer{resp: updateResponse(200, `{"tag_name":"nightly-2026"}`)},
			leak: "nightly",
		},
		{
			name: "网络失败",
			doer: &updateDoer{err: errors.New("dial tcp 10.0.0.1:443: connect refused")},
			leak: "10.0.0.1",
		},
	}
	for _, tc := range cases {
		_, err := desktop.CheckUpdate(context.Background(), tc.doer, "0.5.5")
		if err == nil {
			t.Fatalf("%s: 期望错误", tc.name)
		}
		if strings.Contains(err.Error(), tc.leak) {
			t.Errorf("%s: 错误文案泄露上游内容: %q", tc.name, err)
		}
	}
}

func TestCheckUpdateInvalidLocalVersionFailsBeforeRequest(t *testing.T) {
	d := &updateDoer{resp: updateResponse(200, `{"tag_name":"v0.6.0"}`)}
	if _, err := desktop.CheckUpdate(context.Background(), d, "dev"); err == nil {
		t.Fatal("本机版本非法时应报错")
	}
	if d.gotReq != nil {
		t.Error("本机版本非法时不应发出请求")
	}
}

func TestAutoUpdateCheckDefaultsOffAndRoundTrips(t *testing.T) {
	if desktop.DefaultSettings().AutoUpdateCheck {
		t.Fatal("自动检查更新默认应关闭")
	}
	inner := porttest.NewMemStore()
	if err := inner.Put(desktop.ConfigStoreKey, []byte(`{"studentId":"<学号>"}`)); err != nil {
		t.Fatal(err)
	}
	store := desktop.NewSettingsStore(inner)
	old, err := store.Load()
	if err != nil || old.AutoUpdateCheck {
		t.Fatalf("老配置不应开启自动检查更新: %+v err=%v", old, err)
	}

	res := validSettingsResult()
	res.AutoUpdateCheck = true
	next, err := desktop.ParseSettingsResult(res, old)
	if err != nil || !next.AutoUpdateCheck {
		t.Fatalf("表单应能开启自动检查更新: %+v err=%v", next, err)
	}
	if !desktop.ToSettingsForm(next).AutoUpdateCheck {
		t.Fatal("表单应回显自动检查更新开关")
	}
	if _, err := store.Save(next); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || !loaded.AutoUpdateCheck {
		t.Fatalf("保存往返失败: %+v err=%v", loaded, err)
	}

	// 零值省略写入，保持老配置的字段集合兼容。
	next.AutoUpdateCheck = false
	if _, err := store.Save(next); err != nil {
		t.Fatal(err)
	}
	raw, err := inner.Get(desktop.ConfigStoreKey)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "autoUpdateCheck") {
		t.Errorf("零值应省略: %s", raw)
	}
}

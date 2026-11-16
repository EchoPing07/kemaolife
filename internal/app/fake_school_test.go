package app_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// fakeSchool 起两个 httptest，模拟实测的跨主机链路：
// CAS 在 casURL，教务 jsxsd 在 jsbURL。
type fakeSchool struct {
	casURL string
	jsbURL string
	client *fakeDoer
}

// 实测形状的锚点页与课表 JSON（占位值）。
const fakeAnchorHTML = `<table id="timetable"><thead><tr>
<th><span>周一<br /></span><span>08-31</span></th>
<th><span>周二<br /></span><span>09-01</span></th>
<th><span>周三<br /></span><span>09-02</span></th>
<th><span>周四<br /></span><span>09-03</span></th>
<th><span>周五<br /></span><span>09-04</span></th>
<th><span>周六<br /></span><span>09-05</span></th>
<th><span>周日<br /></span><span>09-06</span></th>
</tr></thead></table>`

const fakeCoursesJSON = `{"msg":"","code":0,"count":1,"data":[
 {"kc_mc":"新课程","kch":"<课程号>","jg0101mc":"教师乙",
  "sktime":"周三第1、2节{10}全周;周四第1、2节{9}全周",
  "skddmc":"<地点A>;<地点B>","skxqmc":"<校区>","kt_mc":"<教学班>"}
]}`

// 实测会话失效响应：HTTP 200 + 一层 JS 跳转到 CAS，既不含登录页标记也不含教务内容。
// 旧实现在这里会误判成「会话仍有效」，导致会话过期后永不重登（docs/contract.md §4）。
const expiredSessionStub = `<script languge='javascript'>window.location.href='https://aic.gdkm.edu.cn/cas_server/?service=http%3A%2F%2Fjw.gdkm.edu.cn%2Fjsxsd%2Fframework%2FxsMainV.htmlx'</script>`

func newFakeSchool(t *testing.T) *fakeSchool {
	t.Helper()

	jsbMux := http.NewServeMux()
	jsbMux.HandleFunc("/jsxsd/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ticket") != "" {
			http.SetCookie(w, &http.Cookie{Name: timetable.SessionCookie, Value: "sess-1"})
			http.SetCookie(w, &http.Cookie{Name: timetable.ServerCookie, Value: "srv-1"})
			http.Redirect(w, r, "/jsxsd/framework/xsMainV.htmlx", http.StatusFound)
			return
		}
		if !hasFakeSession(r) {
			fmt.Fprint(w, `<title>统一身份认证平台</title>`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	jsbMux.HandleFunc("/jsxsd/framework/xsMainV.htmlx", func(w http.ResponseWriter, r *http.Request) {
		if !hasFakeSession(r) {
			// 实测失效响应：HTTP 200 + JS 跳转桩页（不是登录页 HTML）。
			// 若这里回登录页，就测不出「桩页被误判成有效会话」的缺陷。
			_, _ = io.WriteString(w, expiredSessionStub)
			return
		}
		fmt.Fprint(w, `<html><body><div class="jsxsd">教务首页 课表</div></body></html>`)
	})
	jsbMux.HandleFunc("/jsxsd/framework/mainV_index_loadkb.htmlx", func(w http.ResponseWriter, r *http.Request) {
		if !hasFakeSession(r) {
			_, _ = io.WriteString(w, expiredSessionStub)
			return
		}
		fmt.Fprint(w, fakeAnchorHTML)
	})
	jsbMux.HandleFunc("/jsxsd/xskb/xskb_list.do", func(w http.ResponseWriter, r *http.Request) {
		if !hasFakeSession(r) {
			_, _ = io.WriteString(w, expiredSessionStub)
			return
		}
		w.Header().Set("Content-Type", "text/html;charset=utf-8")
		fmt.Fprint(w, fakeCoursesJSON)
	})

	jsb := httptest.NewServer(jsbMux)
	t.Cleanup(jsb.Close)

	casMux := http.NewServeMux()
	casMux.HandleFunc("/cas_server/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `<title>统一身份认证平台</title>
<form id="form1">
<input type="hidden" id="__VIEWSTATE" value="vs" />
<input type="hidden" id="__VIEWSTATEGENERATOR" value="gen" />
<input type="hidden" id="__EVENTVALIDATION" value="ev" />
<input name="signupInputUserName" />
<input name="signupInputPassword" type="password" />
<input type="submit" name="btn_login" />
</form>`)
			return
		}
		_ = r.ParseForm()
		if r.Form.Get(timetable.CASFieldUser) != testAccount ||
			r.Form.Get(timetable.CASFieldPass) != testPass {
			fmt.Fprint(w, `<title>统一身份认证平台</title><input name="signupInputUserName" />`)
			return
		}
		http.Redirect(w, r, jsb.URL+"/jsxsd/?ticket=ST-1", http.StatusFound)
	})

	cas := httptest.NewServer(casMux)
	t.Cleanup(cas.Close)

	return &fakeSchool{
		casURL: cas.URL,
		jsbURL: jsb.URL,
		client: &fakeDoer{client: &http.Client{}},
	}
}

// hasFakeSession 只把登录下发的 sess-1 / srv-1 当成有效会话。
//
// 不能只判「有 Cookie」：种子快照里的 Cookie 必须被当成已失效，
// 才能复现「旧会话过期 → 自动重登」链路（与真实 jsxsd 行为一致）。
func hasFakeSession(r *http.Request) bool {
	a, e1 := r.Cookie(timetable.SessionCookie)
	b, e2 := r.Cookie(timetable.ServerCookie)
	return e1 == nil && e2 == nil && a.Value == "sess-1" && b.Value == "srv-1"
}

// fakeDoer 同时实现 Do 与 DoNoRedirect，模拟 platform.HTTPDoer。
type fakeDoer struct{ client *http.Client }

func (d *fakeDoer) Do(req *http.Request) (*http.Response, error) { return d.client.Do(req) }

func (d *fakeDoer) DoNoRedirect(req *http.Request) (*http.Response, error) {
	c := *d.client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c.Do(req)
}

var _ = strings.TrimSpace

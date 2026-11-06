package electricity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

const (
	testAccount = "<学号>"
	testSchool  = "3126" // 学校事实，非个人隐私；允许入仓
	testRoom    = "<roomverify>"
)

// newServer 构造一个按 cmd 分发假响应的 SWAEServlet。
func newServer(t *testing.T, handler func(cmd string, form map[string]string) (outer string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("电费接口只接受 POST，收到 %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/x-www-form-urlencoded") {
			t.Errorf("Content-Type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		form := map[string]string{
			"param":        r.Form.Get("param"),
			"customercode": r.Form.Get("customercode"),
			"method":       r.Form.Get("method"),
		}

		var param struct {
			Cmd string `json:"cmd"`
		}
		if err := json.Unmarshal([]byte(form["param"]), &param); err != nil {
			t.Fatalf("param 不是合法 JSON: %v", err)
		}
		if form["method"] != param.Cmd {
			t.Errorf("method=%q 与 param.cmd=%q 不一致", form["method"], param.Cmd)
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		fmt.Fprint(w, handler(param.Cmd, form))
	}))
}

// wrap 拼出外层包装；body 为内层 JSON 字符串。
func wrap(body string) string {
	b, _ := json.Marshal(body)
	return fmt.Sprintf(`{"code_":0,"sign":"<签名>","result_":"true","body":%s,"message_":"获取成功！"}`, b)
}

const bindRoomBody = `{
  "result":"0","message":"操作成功","existflag":"1",
  "roomfullname":"<校区>_<楼栋>_<楼层>_<房间>",
  "roomverify":"<roomverify>",
  "detaillist":[
    {"businesstype":"0","odd":"35.99","use":"100","sumbuy":0,"sumsub":0,"status":"1"}
  ]
}`

const useDetailBody = `{
  "result":"0","message":"操作成功","yearmonth":"2026-11","monthuse":"63.13",
  "opdtdate":"2026-11-04",
  "dayuselist":[
    {"date":"2026-11-04","dayuse":11.75,"use":11.75},
    {"date":"2026-11-03","dayuse":11.64,"use":11.64},
    {"date":"2026-11-02","dayuse":10.50,"use":10.50}
  ]
}`

func TestGetBindRoomParsesBalance(t *testing.T) {
	srv := newServer(t, func(cmd string, form map[string]string) string {
		if form["customercode"] != testSchool {
			t.Errorf("customercode = %q", form["customercode"])
		}
		return wrap(bindRoomBody)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	status, roomVerify, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if err != nil {
		t.Fatalf("GetBindRoom: %v", err)
	}
	if status.Balance != 35.99 {
		t.Errorf("Balance = %v, want 35.99", status.Balance)
	}
	if status.Unit != "度" || status.BusinessType != 0 {
		t.Errorf("unit/businessType = %q/%d", status.Unit, status.BusinessType)
	}
	if status.MeterStatus != "1" {
		t.Errorf("MeterStatus = %q", status.MeterStatus)
	}
	if roomVerify != testRoom {
		t.Errorf("roomVerify = %q（内部需要原值）", roomVerify)
	}
	// 对外展示必须脱敏。
	if status.RoomVerify == testRoom {
		t.Error("对外 status.RoomVerify 必须脱敏")
	}
}

// 关键坑：缺 account 时外层仍是 result_:"true"，错误只在 body.result（-20003）。
func TestCallDetectsInnerFailureDespiteOuterSuccess(t *testing.T) {
	srv := newServer(t, func(string, map[string]string) string {
		return wrap(`{"result":"-20003","message":"参数异常：account不存在"}`)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if err == nil {
		t.Fatal("内层失败必须报错，不能因为外层 result_ 为 true 就当成功")
	}
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeCredentialMissing)
	}
	if !strings.Contains(err.Error(), "电费业务失败") || strings.Contains(err.Error(), "参数异常") {
		t.Errorf("上游文案必须脱敏: %v", err)
	}
}

func TestCallOuterFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code_":500,"result_":"false","body":"","message_":"服务异常"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if model.CodeOf(err) != model.CodeNetworkDown {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeNetworkDown)
	}
	if strings.Contains(err.Error(), "服务异常") {
		t.Errorf("上游文案必须脱敏: %v", err)
	}
}

func TestCallEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code_":0,"result_":"true","body":"","message_":"获取成功！"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeMalformedResponse)
	}
}

func TestCallInvalidInnerJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code_":0,"result_":"true","body":"not json","message_":"获取成功！"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeMalformedResponse)
	}
}

// 未绑定房间返回 5001。
func TestGetBindRoomNotBound(t *testing.T) {
	srv := newServer(t, func(string, map[string]string) string {
		return wrap(`{"result":"0","existflag":"2","message":"操作成功"}`)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if model.CodeOf(err) != model.CodeRoomNotBound {
		t.Errorf("code = %d, want %d", model.CodeOf(err), model.CodeRoomNotBound)
	}
}

// 没有电表记录（只有水表）不能返回 0 度冒充成功。
func TestGetBindRoomNoElectricMeter(t *testing.T) {
	srv := newServer(t, func(string, map[string]string) string {
		return wrap(`{"result":"0","existflag":"1","roomverify":"<roomverify>",
			"detaillist":[{"businesstype":"1","odd":"3.00","status":"1"}]}`)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if err == nil {
		t.Fatal("无水表外的电表记录时应报错")
	}
	if model.CodeOf(err) != model.CodeMalformedResponse {
		t.Errorf("code = %d", model.CodeOf(err))
	}
}

// 每日用量必须按日期升序：即使上游乱序返回也保证升序（AGENTS §6）。
func TestGetUseDetailAscendingOrder(t *testing.T) {
	const shuffled = `{
  "result":"0","yearmonth":"2026-11","monthuse":"63.13","opdtdate":"2026-11-04",
  "dayuselist":[
    {"date":"2026-11-03","dayuse":11.64},
    {"date":"2026-11-04","dayuse":11.75},
    {"date":"2026-11-02","dayuse":10.50}
  ]
}`
	srv := newServer(t, func(cmd string, form map[string]string) string {
		return wrap(shuffled)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	usage, err := c.GetUseDetail(context.Background(), testAccount, testSchool, testRoom, "2026-11")
	if err != nil {
		t.Fatalf("GetUseDetail: %v", err)
	}
	if len(usage.Days) != 3 {
		t.Fatalf("days = %+v", usage.Days)
	}
	if usage.Days[0].Date != "2026-11-02" || usage.Days[2].Date != "2026-11-04" {
		t.Errorf("未按日期升序: %+v", usage.Days)
	}
	if usage.DataUntil != "2026-11-04" {
		t.Errorf("DataUntil = %q", usage.DataUntil)
	}
	if usage.MonthUse != 63.13 {
		t.Errorf("MonthUse = %v", usage.MonthUse)
	}
}

func TestCallRequiresAccountAndSchoolCode(t *testing.T) {
	c := NewClient(http.DefaultClient, "http://example.invalid")
	if _, _, err := c.GetBindRoom(context.Background(), "", testSchool); model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("空学号 code = %d", model.CodeOf(err))
	}
	if _, _, err := c.GetBindRoom(context.Background(), testAccount, ""); model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("空学校代码 code = %d", model.CodeOf(err))
	}
}

func TestEstimate(t *testing.T) {
	days := []model.DayUsage{
		{Date: "2026-11-01", Use: 10},
		{Date: "2026-11-02", Use: 12},
		{Date: "2026-11-03", Use: 11},
		{Date: "2026-11-04", Use: 10},
		{Date: "2026-11-05", Use: 13},
		{Date: "2026-11-06", Use: 9},
		{Date: "2026-11-07", Use: 11},
	}
	avg, _ := Estimate(days)
	if avg == nil {
		t.Fatal("应有均值")
	}
	if want := 10.86; *avg != want {
		t.Errorf("dailyAvg = %v, want %v", *avg, want)
	}

	status := model.ElectricityStatus{Balance: 35.99}
	WithEstimate(&status, days)
	if status.EstimatedDays == nil || *status.EstimatedDays != 3 {
		t.Errorf("estimatedDays = %v, want 3", status.EstimatedDays)
	}
}

func TestEstimateFewerThan7Days(t *testing.T) {
	days := []model.DayUsage{{Date: "2026-11-03", Use: 4}, {Date: "2026-11-04", Use: 6}}
	avg, _ := Estimate(days)
	if avg == nil || *avg != 5 {
		t.Errorf("dailyAvg = %v, want 5", avg)
	}
}

func TestEstimateNoData(t *testing.T) {
	avg, _ := Estimate(nil)
	if avg != nil {
		t.Errorf("无数据时应为 nil, got %v", *avg)
	}
	status := model.ElectricityStatus{Balance: 20}
	WithEstimate(&status, nil)
	if status.DailyAvg != nil || status.EstimatedDays != nil {
		t.Error("无数据时均值与可用天数都应为 null，不能用 0 冒充")
	}
}

func TestEstimateZeroUsage(t *testing.T) {
	days := []model.DayUsage{{Date: "2026-11-03", Use: 0}, {Date: "2026-11-04", Use: 0}}
	status := model.ElectricityStatus{Balance: 20}
	WithEstimate(&status, days)
	if status.EstimatedDays != nil {
		t.Errorf("均量为 0 时可用天数应为 null, got %v", *status.EstimatedDays)
	}
}

// 实测形状：上游拒绝时可能只给 result_:false + exception_，完全没有 body 字段。
// 这必须映射成「配置/凭据问题」或「网络不可用」，而不是 4002 响应格式异常。
func TestCallOuterRejectedWithoutBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code_":8099,"result_":false,"exception_":"客户学校不存在","message_":"客户学校不存在"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if err == nil {
		t.Fatal("应报错")
	}
	if model.CodeOf(err) != model.CodeCredentialMissing {
		t.Errorf("code = %d, want %d（学校代码配错属配置问题）", model.CodeOf(err), model.CodeCredentialMissing)
	}
}

// 实测形状：body 直接内联对象、code_ 为字符串；flexjson 必须都兼容。
func TestCallAcceptsInlineBodyAndStringCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code_":"0","result_":"true","body":`+bindRoomBody+`,"message_":"获取成功！"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	status, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if err != nil {
		t.Fatalf("内联 body 与字符串 code_ 应可解析: %v", err)
	}
	if status.Balance != 35.99 {
		t.Errorf("Balance = %v", status.Balance)
	}
}

// result_ 是字符串 "true" 或布尔 true 都要能读。
func TestCallAcceptsBooleanResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(bindRoomBody)
		fmt.Fprintf(w, `{"code_":0,"result_":true,"body":%s,"message_":"获取成功！"}`, b)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	status, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if err != nil {
		t.Fatalf("布尔 result_ 应可解析: %v", err)
	}
	if status.Balance != 35.99 {
		t.Errorf("Balance = %v", status.Balance)
	}
}

// 「客户未配置完美校园前置地址」表示学校代码存在但接口未开通：
// 用户改代码永远改不好，必须报网络/不可用（4001），不能报成凭据问题（3001）。
func TestCallOuterSchoolNotProvisionedIsNotCredentialError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code_":8099,"result_":false,"exception_":"客户未配置完美校园前置地址","message_":"客户未配置完美校园前置地址"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	_, _, err := c.GetBindRoom(context.Background(), testAccount, testSchool)
	if model.CodeOf(err) != model.CodeNetworkDown {
		t.Errorf("code = %d, want %d（学校接口未开通属不可用，不是配置错误）",
			model.CodeOf(err), model.CodeNetworkDown)
	}
	if msg := model.MessageOf(err); strings.Contains(msg, "客户未配置") || strings.Contains(msg, "前置地址") {
		t.Errorf("exception_ 原文不得进入 error.message: %q", msg)
	}
}

// getstuindexpage 的 collecdate 是真实采集时刻；odd/todayuse 实测缺失，不得当数据源。
func TestGetIndexPageReturnsCollecDate(t *testing.T) {
	srv := newServer(t, func(cmd string, form map[string]string) string {
		if cmd != CmdGetStuIndexPage {
			t.Errorf("cmd = %q", cmd)
		}
		return wrap(`{"result":"0","modlist":[
			{"collecdate":"2026-11-05 22:24:52","modstatus":"1路:一般送电"}
		]}`)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	got, err := c.GetIndexPage(context.Background(), testAccount, testSchool)
	if err != nil {
		t.Fatalf("GetIndexPage: %v", err)
	}
	if got != "2026-11-05 22:24:52" {
		t.Errorf("collecdate = %q", got)
	}
}

// 没有 collecdate 时返回空串而不是报错，让调用方降级到 opdtdate。
func TestGetIndexPageWithoutCollecDate(t *testing.T) {
	srv := newServer(t, func(string, map[string]string) string {
		return wrap(`{"result":"0","modlist":[{"modstatus":"1路:一般送电"}]}`)
	})
	defer srv.Close()

	c := NewClient(srv.Client(), srv.URL)
	got, err := c.GetIndexPage(context.Background(), testAccount, testSchool)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if got != "" {
		t.Errorf("collecdate = %q, want 空", got)
	}
}

// Package electricity 访问智能水电 H5 后端（SWAEServlet），解析双层 JSON。
//
// 两个坑（AGENTS §5.3）必须逐条对照本包：
//  1. `body` 是字符串，缺参数时外层仍 `result_:"true"`，必须解 `body` 再判 `body.result`；
//  2. 一次查询 = 2 个请求（getbindroom + getusedetail），任一失败整体报错，不返回半份数据。
package electricity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Client 通过 HTTPDoer 访问水电后端；无鉴权，身份凭证只有学号（api-docs/002 §1）。
type Client struct {
	doer     core.HTTPDoer
	endpoint string
}

// NewClient 构造客户端；endpoint 为空时用 Endpoint（单测用它指向 httptest）。
func NewClient(doer core.HTTPDoer, endpoint string) *Client {
	if endpoint == "" {
		endpoint = Endpoint
	}
	return &Client{doer: doer, endpoint: endpoint}
}

// GetBindRoom 取已绑定房间与余额概况，返回 status 与 roomverify。
//
// roomverify 是后续查询必用的房间编码；属个人信息，不进日志（api-docs/002 §6.1）。
func (c *Client) GetBindRoom(ctx context.Context, account, schoolCode string) (model.ElectricityStatus, string, error) {
	body, err := c.call(ctx, CmdGetBindRoom, account, schoolCode, nil)
	if err != nil {
		return model.ElectricityStatus{}, "", err
	}

	existFlag := asString(body[bodyFieldExistFlag])
	if existFlag != "" && existFlag != "1" {
		return model.ElectricityStatus{}, "", model.NewError(model.CodeRoomNotBound,
			"该账号尚未绑定房间，请先在 H5 页面完成绑定（本工具不做绑定）")
	}

	roomVerify := asString(body[bodyFieldRoomVer])
	status := model.ElectricityStatus{
		RoomName:   asString(body[bodyFieldRoomName]),
		RoomVerify: maskRoomVerify(roomVerify),
	}

	list, _ := body[bodyFieldDetail].([]any)
	found := false
	for _, item := range list {
		d, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if asString(d[detailFieldBusinessType]) != BusinessTypeElectric {
			continue
		}
		odd, err := parseFloat(d[detailFieldOdd])
		if err != nil {
			return model.ElectricityStatus{}, "", model.WrapError(model.CodeMalformedResponse,
				"剩余电量字段无法解析", err)
		}
		status.Balance = odd
		status.Unit = "度"
		status.BusinessType = 0
		status.MeterStatus = asString(d[detailFieldStatus])
		found = true
		break
	}
	if !found {
		return model.ElectricityStatus{}, "", model.NewError(model.CodeMalformedResponse,
			"未找到电表记录（businesstype=0）")
	}
	return status, roomVerify, nil
}

// GetIndexPage 取首页聚合数据里的**数据采集时刻**（`modlist[0].collecdate`）。
//
// 为什么单独发这一个请求：电费 DTO 的 `collectedAt` 必须是「数据截至」的准确时刻。
// `getbindroom` 不给时间，`getusedetail` 只给 `opdtdate`（截止日，无时刻）。
// 实测 `getstuindexpage` 的 `collecdate` 形如 `2026-11-05 22:24:52`，才是真实采集时刻。
//
// 注意：`getstuindexpage` 的 `odd` / `todayuse` 实测**缺失**，不得用作余额或用量来源
// （api-docs/002 §6.3）。本方法只读时间字段，读不到就返回空串，由调用方降级。
func (c *Client) GetIndexPage(ctx context.Context, account, schoolCode string) (string, error) {
	body, err := c.call(ctx, CmdGetStuIndexPage, account, schoolCode, nil)
	if err != nil {
		return "", err
	}
	list, _ := body[bodyFieldModList].([]any)
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if ts := asString(m[bodyFieldCollecDate]); ts != "" {
			return ts, nil
		}
	}
	return "", nil
}

// GetUseDetail 取某月每日用量，days 按日期升序（接口返回倒序，这里反转）。
func (c *Client) GetUseDetail(ctx context.Context, account, schoolCode, roomVerify, yearMonth string) (model.ElectricityUsage, error) {
	body, err := c.call(ctx, CmdGetUseDetail, account, schoolCode, map[string]string{
		"roomverify":   roomVerify,
		"businesstype": BusinessTypeElectric,
		"yearmonth":    yearMonth,
	})
	if err != nil {
		return model.ElectricityUsage{}, err
	}

	usage := model.ElectricityUsage{
		YearMonth: asString(body[bodyFieldYearMonth]),
		DataUntil: asString(body[bodyFieldOpdtDate]),
		Days:      []model.DayUsage{},
	}
	if usage.YearMonth == "" {
		usage.YearMonth = yearMonth
	}
	if v := asString(body[bodyFieldMonthUse]); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			usage.MonthUse = f
		}
	}

	list, _ := body[bodyFieldDayUses].([]any)
	reversed := make([]model.DayUsage, 0, len(list))
	for _, item := range list {
		d, ok := item.(map[string]any)
		if !ok {
			continue
		}
		date := asString(d[dayFieldDate])
		if date == "" {
			continue
		}
		raw := d[dayFieldDayUse]
		if raw == nil {
			raw = d[dayFieldUse]
		}
		use, err := parseFloat(raw)
		if err != nil {
			continue // 单日数据异常不拖垮整月
		}
		reversed = append(reversed, model.DayUsage{Date: date, Use: use})
	}
	// 接口倒序 → 升序（docs/002 §3.2「days[] 按日期升序输出」）。
	for i := len(reversed) - 1; i >= 0; i-- {
		usage.Days = append(usage.Days, reversed[i])
	}
	return usage, nil
}

// call 发一次 SWAEServlet 请求，返回已解开双层的业务 body。
func (c *Client) call(ctx context.Context, cmd, account, schoolCode string, extra map[string]string) (map[string]any, error) {
	if strings.TrimSpace(account) == "" {
		return nil, model.NewError(model.CodeCredentialMissing, "学号为空，请先执行 kemaolife config init")
	}
	if strings.TrimSpace(schoolCode) == "" {
		return nil, model.NewError(model.CodeCredentialMissing,
			"学校代码为空，请检查配置或执行 kemaolife config set customercode")
	}

	// param 是 JSON 字符串，account 必须由客户端注入（api-docs/002 §3.2）。
	payload := map[string]any{"cmd": cmd, "account": account}
	for k, v := range extra {
		payload[k] = v
	}
	paramJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, model.WrapError(model.CodeInvalidArgument, "构造电费请求参数失败", err)
	}

	form := url.Values{
		"param":        {string(paramJSON)},
		"customercode": {schoolCode},
		"method":       {cmd}, // 必须与 param.cmd 一致（api-docs/002 §3.1）
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, model.WrapError(model.CodeNetworkDown, "构造电费请求失败", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, model.WrapError(model.CodeNetworkDown, "电费接口请求失败", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, model.NewError(model.CodeNetworkDown,
			fmt.Sprintf("电费接口返回 HTTP %d", resp.StatusCode))
	}

	var outer struct {
		Code      flexInt    `json:"code_"`
		Result    flexBool   `json:"result_"`
		Body      flexString `json:"body"`
		Message   string     `json:"message_"`
		Exception string     `json:"exception_"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&outer); err != nil {
		return nil, model.WrapError(model.CodeMalformedResponse, "电费外层 JSON 解析失败", err)
	}
	if !bool(outer.Result) {
		// 上游拒绝（如「客户学校不存在」）：属配置或凭据问题，不是网络问题。
		// message_ / exception_ 只用于定位，不代表个人信息，可以透传。
		msg := outer.Message
		if msg == "" {
			msg = outer.Exception
		}
		if msg == "" {
			msg = "上游返回失败"
		}
		return nil, model.NewError(codeForOuterFailure(int(outer.Code), msg), "电费接口请求失败: "+msg)
	}
	bodyText := string(outer.Body)
	if strings.TrimSpace(bodyText) == "" {
		return nil, model.NewError(model.CodeMalformedResponse, "电费响应 body 为空")
	}

	// 关键：外层成功不代表成功，必须解 body 再判 body.result（api-docs/002 §4）。
	var body map[string]any
	if err := json.Unmarshal([]byte(bodyText), &body); err != nil {
		return nil, model.WrapError(model.CodeMalformedResponse, "电费内层 JSON 解析失败", err)
	}
	if result := asString(body[bodyFieldResult]); result != "0" {
		msg := asString(body[bodyFieldMessage])
		if msg == "" {
			msg = "上游业务失败"
		}
		return nil, model.NewError(codeForResult(result), "电费业务失败: "+msg)
	}
	return body, nil
}

// codeForResult 把 body.result 映射到错误码域（api-docs/002 §8）。
func codeForResult(result string) model.ErrorCode {
	switch result {
	case "-20003", "-1":
		// 参数缺失或学号无效：属凭据问题，引导用户检查配置。
		return model.CodeCredentialMissing
	}
	return model.CodeBusinessFailed
}

// codeForOuterFailure 把外层 result_!=true 映射到错误码域。
//
// api-docs/002 §4 只描述了字段，未穷举错误码。已知 8099 的两种 message 语义不同：
//   - 「客户学校不存在」→ 学校代码配错，用户可自己改（3001）
//   - 「客户未配置完美校园前置地址」→ 代码存在但该学校未开通本接口，用户改不了（4001）
//
// 区分它们很关键：把后者报成 3001 会让用户反复改代码却永远失败。
func codeForOuterFailure(code int, msg string) model.ErrorCode {
	if strings.Contains(msg, "未配置") || strings.Contains(msg, "前置") {
		return model.CodeNetworkDown
	}
	switch code {
	case 8099:
		return model.CodeCredentialMissing
	}
	if strings.Contains(msg, "学校") || strings.Contains(msg, "参数") {
		return model.CodeCredentialMissing
	}
	return model.CodeNetworkDown
}

// maskRoomVerify 对房间编码脱敏：对外只保留长度信息（P1 / docs/002 §3.2）。
func maskRoomVerify(raw string) string {
	if raw == "" {
		return ""
	}
	return "已脱敏"
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func parseFloat(v any) (float64, error) {
	switch t := v.(type) {
	case nil:
		return 0, fmt.Errorf("值为空")
	case float64:
		return t, nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, fmt.Errorf("值为空字符串")
		}
		return strconv.ParseFloat(s, 64)
	default:
		return strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(t)), 64)
	}
}

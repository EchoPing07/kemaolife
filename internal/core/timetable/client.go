package timetable

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Client 通过 HTTPDoer 访问教务系统。传输层细节（强制 IPv4、超时、TLS）
// 由 platform 适配器负责，本包不感知（AGENTS §5.3）。
//
// 实测链路（2026-11）：
//
//	GET  aic/cas_server/login?service=http://jw/jsxsd/   → 取 __VIEWSTATE
//	POST aic/cas_server/login?service=...                → 302 → jw/jsxsd/?ticket=
//	                                                       → 302 → bzb_jsxsd Cookie
//	GET  jw/jsxsd/framework/mainV_index_loadkb.htmlx?zc=1 → 第 1 周 7 个 MM-DD
//	POST jw/jsxsd/xskb/xskb_list.do?viweType=1&needData=1 → 课程 JSON
type Client struct {
	doer    core.HTTPDoer
	clock   core.Clock
	casBase string
	jsbBase string
	cookies map[string]string
}

// NewClient 构造客户端；casBase / jsbBase 为空时用实测默认值（单测用它们指向 httptest）。
func NewClient(doer core.HTTPDoer, clock core.Clock, casBase, jsbBase string) *Client {
	if casBase == "" {
		casBase = CASBase
	}
	if jsbBase == "" {
		jsbBase = JSBBase
	}
	return &Client{
		doer:    doer,
		clock:   clock,
		casBase: strings.TrimRight(casBase, "/"),
		jsbBase: strings.TrimRight(jsbBase, "/"),
		cookies: map[string]string{},
	}
}

// SetCookies 注入已有会话 Cookie（来自快照）。
func (c *Client) SetCookies(cookies map[string]string) {
	c.cookies = make(map[string]string, len(cookies))
	for k, v := range cookies {
		c.cookies[k] = v
	}
}

// Cookies 返回当前会话 Cookie 的副本；调用方按凭据对待（P6）。
func (c *Client) Cookies() map[string]string {
	out := make(map[string]string, len(c.cookies))
	for k, v := range c.cookies {
		out[k] = v
	}
	return out
}

// HasSession 判断教务会话 Cookie 是否齐全。
//
// 实测需要两个：bzb_jsxsd（业务会话）与 SERVERID_jsxsd（后端粘性）。缺一个就会被
// 回退成登录页，表现为「同步拿到 179 字节的登录 HTML」。
func (c *Client) HasSession() bool {
	return c.cookies[SessionCookie] != "" && c.cookies[ServerCookie] != ""
}

var (
	reViewState = regexp.MustCompile(`id="__VIEWSTATE"\s+value="([^"]*)"`)
	reVSGen     = regexp.MustCompile(`id="__VIEWSTATEGENERATOR"\s+value="([^"]*)"`)
	reEventVal  = regexp.MustCompile(`id="__EVENTVALIDATION"\s+value="([^"]*)"`)
)

// Login 走统一 CAS 登录，成功后应拿到 jsxsd 会话 Cookie。
//
// 不能只看 HTTP 状态码：CAS 与 jsxsd 在凭据错误时也可能返回 200（api-docs/001 §6.1），
// 所以判据是「是否拿到 SessionCookie」。
func (c *Client) Login(ctx context.Context, user, password string) error {
	if strings.TrimSpace(user) == "" || password == "" {
		return model.NewError(model.CodeCredentialMissing, "学号或教务密码为空")
	}

	loginURL := c.casBase + CASLoginPath + "?service=" + url.QueryEscape(JSBTimetableService)

	page, err := c.get(ctx, c.casBase, CASLoginPath+"?service="+url.QueryEscape(JSBTimetableService))
	if err != nil {
		return err
	}
	if strings.Contains(page, "signupInputUserName") == false && strings.Contains(page, "统一身份认证") == false {
		return model.NewError(model.CodeMalformedResponse,
			"CAS 登录页结构不符合预期，可能接口已变更")
	}

	form := url.Values{
		CASFieldUser:            {user},
		CASFieldPass:            {password},
		CASFieldSubmit:          {"登录"},
		CASFieldViewState:       {firstMatch(reViewState, page)},
		CASFieldViewStateGen:    {firstMatch(reVSGen, page)},
		CASFieldEventValidation: {firstMatch(reEventVal, page)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return model.WrapError(model.CodeNetworkDown, "构造登录请求失败", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", loginURL)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	body, err := c.doFollow(ctx, req)
	if err != nil {
		return err
	}

	// 凭据错误时 CAS 会把登录页原样返回（HTTP 200）。
	if !c.HasSession() && looksLikeLoginPage(body) {
		return model.NewError(model.CodeUpstreamRejected, "教务账号或密码错误")
	}

	if !c.HasSession() {
		// HTTPDoer 不跟随重定向时，ticket 换 Cookie 的那一跳还没发生；
		// 主动请求一次主页让它下发会话 Cookie（只补一跳，不重复登录登录）。
		if _, err := c.getAllowError(ctx, c.jsbBase, HomePath); err != nil {
			return err
		}
	}
	if !c.HasSession() {
		return model.NewError(model.CodeUpstreamRejected,
			"登录未取得教务会话 Cookie，请确认账号状态")
	}
	return nil
}

// sessionCookieNames 仅用于诊断（不输出值）。
func (c *Client) sessionCookieNames() []string {
	out := make([]string, 0, len(c.cookies))
	for k := range c.cookies {
		out = append(out, k)
	}
	return out
}

// IsAlive 探测会话是否仍然有效：会话失效也返回 200，必须判响应内容。
func (c *Client) IsAlive(ctx context.Context) (bool, error) {
	if !c.HasSession() {
		return false, nil
	}
	page, err := c.get(ctx, c.jsbBase, HomePath)
	if err != nil {
		return false, err
	}
	if looksLikeLoginPage(page) {
		return false, nil
	}
	return true, nil
}

// looksLikeLoginPage 判断响应内容是否是登录页。
func looksLikeLoginPage(body string) bool {
	if strings.Contains(body, "jsxsd") && strings.Contains(body, "课表") {
		return false // 明显是教务内页
	}
	lower := strings.ToLower(body)
	for _, marker := range []string{
		"signupinputusername", "统一身份认证", "casloginform", "__viewstate",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// Sync 拉取第 1 周锚点与课表 JSON，产出一份快照。
func (c *Client) Sync(ctx context.Context) (*Snapshot, error) {
	now := c.clock.Now()

	anchorHTML, err := c.get(ctx, c.jsbBase, Week1AnchorPath)
	if err != nil {
		return nil, err
	}
	if looksLikeLoginPage(anchorHTML) {
		return nil, model.NewError(model.CodeSessionExpired, "会话失效，需要重新登录")
	}

	week1Dates, err := ParseWeek1Anchor(anchorHTML, now.Year())
	if err != nil {
		// 诊断信息只含长度与结构特征，不含任何个人信息（P2）。
		return nil, model.WrapError(model.CodeOf(err),
			fmt.Sprintf("第 1 周锚点页解析失败（响应 %d 字节，日期样式 %d 处）",
				len(anchorHTML), len(reAnchorDate.FindAllString(anchorHTML, -1))), err)
	}

	courses, err := c.fetchCourses(ctx)
	if err != nil {
		return nil, err
	}

	lessons := make([]Lesson, 0, len(courses))
	lastWeek := 0
	for _, course := range courses {
		expanded, maxWeek, err := lessonFromCourse(course)
		if err != nil {
			// 单条课程解析失败不拖垮整份课表，但不能整份静默变空。
			continue
		}
		lessons = append(lessons, expanded...)
		if maxWeek > lastWeek {
			lastWeek = maxWeek
		}
	}
	if len(lessons) == 0 && len(courses) > 0 {
		return nil, model.NewError(model.CodeMalformedResponse,
			"课程 JSON 全部条目解析失败，可能字段名已变更")
	}
	if lastWeek == 0 {
		lastWeek = 1
	}

	return &Snapshot{
		Version:     SnapshotVersion,
		SyncedAt:    now,
		Week1Monday: week1Dates[0],
		Week1Dates:  week1Dates,
		SlotTimes:   append([]string(nil), SlotTimes...),
		LastWeek:    lastWeek,
		Lessons:     lessons,
		Cookies:     c.Cookies(),
	}, nil
}

// courseRow 是课表 JSON 的一行（只取用到的字段）。
type courseRow map[string]any

// lessonFromCourse 把一条课程行展开成多个 Lesson（一个时段一个），并返回最大周次。
func lessonFromCourse(row courseRow) ([]Lesson, int, error) {
	name := pickString(row, fieldCourseName)
	sktime := pickString(row, fieldCourseTime)
	if name == "" || sktime == "" {
		return nil, 0, fmt.Errorf("课程行缺少必填字段")
	}
	slots, err := ParseSktime(sktime)
	if err != nil {
		return nil, 0, err
	}

	teacher := pickString(row, fieldCourseTeacher)
	location := dedupeList(pickString(row, fieldCourseRoom))
	weeksText := summarizeWeeks(slots)

	maxWeek := 0
	out := make([]Lesson, 0, len(slots))
	for _, slot := range slots {
		out = append(out, Lesson{
			Name:      name,
			Periods:   slot.Periods,
			Weekday:   slot.Weekday,
			Weeks:     slot.Weeks,
			Teacher:   teacher,
			Location:  location,
			WeeksText: weeksText,
		})
		for _, w := range slot.Weeks {
			if w > maxWeek {
				maxWeek = w
			}
		}
	}
	return out, maxWeek, nil
}

// summarizeWeeks 生成 `weeksText`（如 `1-16周`）；连续区间合并，保证输出稳定可 diff。
func summarizeWeeks(slots []Slot) string {
	set := map[int]bool{}
	for _, s := range slots {
		for _, w := range s.Weeks {
			set[w] = true
		}
	}
	weeks := make([]int, 0, len(set))
	for w := range set {
		weeks = append(weeks, w)
	}
	sortInts(weeks)
	if len(weeks) == 0 {
		return ""
	}

	var parts []string
	start, prev := weeks[0], weeks[0]
	flush := func() {
		if start == prev {
			parts = append(parts, fmt.Sprint(start))
			return
		}
		parts = append(parts, fmt.Sprintf("%d-%d", start, prev))
	}
	for _, w := range weeks[1:] {
		if w == prev+1 {
			prev = w
			continue
		}
		flush()
		start, prev = w, w
	}
	flush()
	return strings.Join(parts, ",") + "周"
}

func (c *Client) fetchCourses(ctx context.Context) ([]courseRow, error) {
	form := url.Values{
		formFieldViewType: {"1"},
		formFieldNeedData: {"1"},
		formFieldBaseURL:  {"/jsxsd"},
		formFieldSfykb:    {"2"},
		formFieldXSFL:     {"理论学时,实践学时,"},
		formFieldPageNum:  {"1"},
		formFieldPageSize: {"300"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.jsbBase+CourseListPath,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, model.WrapError(model.CodeNetworkDown, "构造课表请求失败", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", c.jsbBase+"/jsxsd/xskb/xskb_list.do?viweType=1")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	c.attachCookies(req)

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, model.WrapError(model.CodeNetworkDown, "课表请求失败", err)
	}
	raw, err := c.readAll(resp)
	if err != nil {
		return nil, model.WrapError(model.CodeNetworkDown, "读取课表响应失败", err)
	}
	if looksLikeLoginPage(string(raw)) {
		return nil, model.NewError(model.CodeSessionExpired, "会话失效，需要重新登录")
	}
	return ParseCourseList(raw)
}

// ParseCourseList 解析课表 JSON：`{"code":0,"data":[...]}`。
func ParseCourseList(raw []byte) ([]courseRow, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, model.WrapError(model.CodeMalformedResponse, "课表 JSON 解析失败", err)
	}

	if codeRaw, ok := top["code"]; ok {
		var code int
		if err := json.Unmarshal(codeRaw, &code); err == nil && code != 0 {
			return nil, model.NewError(model.CodeBusinessFailed,
				fmt.Sprintf("课表接口返回业务码 %d", code))
		}
	}

	var rows []courseRow
	found := false
	for _, key := range CourseListKeys {
		body, ok := top[key]
		if !ok {
			continue
		}
		if err := json.Unmarshal(body, &rows); err != nil {
			continue
		}
		found = true
		break
	}
	if !found {
		return nil, model.NewError(model.CodeMalformedResponse,
			"课表 JSON 结构不符合预期（未找到课程数组）")
	}
	return rows, nil
}

// getAllowError 与 get 相同，但容忍 4xx：用于「只为了收 Cookie」的补跳。
func (c *Client) getAllowError(ctx context.Context, base, path string) (string, error) {
	return c.getWith(ctx, base+path, true)
}

func (c *Client) get(ctx context.Context, base, path string) (string, error) {
	return c.getWith(ctx, base+path, false)
}

func (c *Client) getWith(ctx context.Context, target string, allowError bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", model.WrapError(model.CodeNetworkDown, "构造请求失败", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	status, body, err := c.doFollowStatus(ctx, req)
	if err != nil {
		return "", err
	}
	if !allowError && status >= 400 {
		return "", model.NewError(model.CodeNetworkDown,
			fmt.Sprintf("教务接口返回 HTTP %d", status))
	}
	return body, nil
}

// maxRedirects 是手动跟随的上限；CAS 正常链路是 3 跳。
const maxRedirects = 8

// doFollow 手动跟随重定向并逐跳吸收 Set-Cookie，返回最终响应体。
//
// 为什么不用 http.Client 的自动跟随：会话 Cookie 由本包自己持有（快照要存它），
// 而 http.Client 只有配了 Jar 才会保存中间跳的 Set-Cookie。手动走能保证
// 「每一跳的 Cookie 都被我们拿到」，也不依赖调用方的 HTTPDoer 是否跟随重定向。
func (c *Client) doFollow(ctx context.Context, req *http.Request) (string, error) {
	_, body, err := c.doFollowStatus(ctx, req)
	return body, err
}

func (c *Client) doFollowStatus(ctx context.Context, req *http.Request) (int, string, error) {
	current := req
	for hop := 0; hop <= maxRedirects; hop++ {
		current = current.Clone(ctx)
		c.attachCookies(current)

		resp, err := c.do(current)
		if err != nil {
			return 0, "", model.WrapError(model.CodeNetworkDown, "请求教务接口失败", err)
		}
		body, err := c.readAll(resp)
		if err != nil {
			return 0, "", model.WrapError(model.CodeNetworkDown, "读取教务响应失败", err)
		}

		if !isRedirect(resp.StatusCode) {
			return resp.StatusCode, string(body), nil
		}
		location := resp.Header.Get("Location")
		if location == "" {
			return resp.StatusCode, string(body), nil
		}
		// 实测中间跳的 Location 形如 `/jsxsd/;<Cookie 名>=<会话值>`：
		// Cookie 被塞在路径参数里，标准 url.Parse 不会当 Set-Cookie 处理，必须自己抽。
		// 文档与注释均不复述真实会话值（P1）。
		if strings.Contains(location, ";") {
			c.absorbPathCookies(location)
			location = stripPathParams(location)
		}

		next, err := current.URL.Parse(location)
		if err != nil {
			return 0, "", model.WrapError(model.CodeMalformedResponse, "重定向地址无法解析", err)
		}
		if next.Host != current.URL.Host {
			// 换主机（aic → jw）：不再带旧主机的 Cookie，由目标主机自己下发。
			current = &http.Request{URL: next, Method: http.MethodGet, Header: http.Header{}}
			current.Header.Set("User-Agent", "Mozilla/5.0")
			continue
		}
		current = &http.Request{URL: next, Method: http.MethodGet, Header: http.Header{}}
		current.Header.Set("User-Agent", "Mozilla/5.0")
	}
	return 0, "", model.NewError(model.CodeNetworkDown, "教务登录重定向次数过多")
}

// do 发一次请求；若适配器支持 NoRedirectDoer 就用不跟随重定向的版本，
// 这样中间跳的 Set-Cookie 不会丢（http.Client 自动跟随会丢弃它们）。
func (c *Client) do(req *http.Request) (*http.Response, error) {
	if nr, ok := c.doer.(core.NoRedirectDoer); ok {
		return nr.DoNoRedirect(req)
	}
	return c.doer.Do(req)
}

// absorbPathCookies 从 `;name=value` 形式的路径参数里抽取 Cookie。
//
// 这是实测行为：jsxsd 把会话 Cookie 拼在 302 的 Location 路径后面
// （`/jsxsd/;bzb_jsxsd=...`），而不是用标准 Set-Cookie 头。
func (c *Client) absorbPathCookies(location string) {
	for _, seg := range strings.Split(location, ";") {
		seg = strings.TrimSpace(seg)
		idx := strings.IndexByte(seg, '=')
		if idx <= 0 {
			continue
		}
		name := seg[:idx]
		value := seg[idx+1:]
		if cut := strings.IndexAny(value, "&?#/"); cut >= 0 {
			value = value[:cut]
		}
		// 只接受已知会话 Cookie 名，避免把奇怪的路径参数当凭据存下来。
		if name == SessionCookie || name == ServerCookie {
			c.cookies[name] = value
		}
	}
}

// stripPathParams 去掉 URL 路径里的 `;name=value` 段。
func stripPathParams(location string) string {
	parts := strings.Split(location, ";")
	if len(parts) == 0 {
		return location
	}
	return parts[0]
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

// readAll 读取响应体并把 Set-Cookie 并入会话。
func (c *Client) readAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	c.absorbCookies(resp)
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// absorbCookies 把响应里的 Cookie 写入客户端。
//
// 实测 jsxsd 会下发 `bzb_jsxsd=...;path=/`，值里带 `;path=/` 尾巴；
// 这里只取分号前的值，避免把路径写进 Cookie。
func (c *Client) absorbCookies(resp *http.Response) {
	for _, ck := range resp.Cookies() {
		if ck.Name == "" {
			continue
		}
		value := ck.Value
		if idx := strings.IndexByte(value, ';'); idx >= 0 {
			value = value[:idx]
		}
		c.cookies[ck.Name] = value
	}
}

func (c *Client) attachCookies(req *http.Request) {
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
}

func firstMatch(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

func pickString(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return trimFloat(t)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

// dedupeList 去掉 `;` 分隔列表里的重复项并保留首次出现顺序。
//
// 实测 skddmc 会把同一地点的多个周次段重复拼接
// （`A;A;A;B`），原样输出既难读又会让 diff 噪声很大。
func dedupeList(raw string) string {
	if !strings.Contains(raw, ";") {
		return raw
	}
	seen := map[string]bool{}
	parts := make([]string, 0, 4)
	for _, item := range strings.Split(raw, ";") {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		parts = append(parts, item)
	}
	return strings.Join(parts, ";")
}

func trimFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", f)
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

// DefaultTimeout 是单次网络请求的建议上限（docs/002 §1.1 --timeout 默认值）。
const DefaultTimeout = 30 * time.Second

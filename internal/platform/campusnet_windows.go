//go:build windows

package platform

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/campusnet"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Portal 端点（api-docs/003 §1）。属学校协议事实，允许入仓。
const (
	portalBase     = "https://10.0.128.3"
	portalProbeURL = "http://5.5.5.5/"
	portalLogin    = "/api/portal/webauth"
	portalDisconn  = "/api/portal/webdisconn"

	// probeTimeout 与 settleDelay 按实测设定：探测 5s 足够判「不在校园网」，
	// 登录 / 下线后 BRAS 需要约 2~3s 才生效（api-docs/003 §1「生效确认」）。
	probeTimeout  = 5 * time.Second
	portalTimeout = 15 * time.Second
	settleDelay   = 3 * time.Second
)

// PortalCredential 是校园网凭据：独立字段，不复用教务密码（D9 已定稿）。
//
// 只在内存里出现，落盘由调用方经 DPAPI 加密（P5）。
type PortalCredential struct {
	UserID      string `json:"userId"`
	Password    string `json:"password"`
	Fingerprint string `json:"fingerprint"` // browserFingerprint，配置项而非硬编码
}

// PortalNetwork 是 core.CampusNetwork + core.NetworkReady 的 Windows 实现。
//
// 平台细节集中在这里：探测用明文 HTTP、Portal 用自签证书（仅对这两个地址
// 跳过校验）、禁跟随重定向。core 只拿到纯函数构造的表单与判态结果。
type PortalNetwork struct {
	probe  *http.Client
	portal *http.Client
	cred   func() PortalCredential
	sleep  func(context.Context, time.Duration) error
	// base 是 Portal 基地址；正常为 portalBase，测试覆盖为 httptest 地址。
	base string
}

// NewPortalNetwork 构造适配器。cred 是凭据提供者（每轮重新读，支持设置窗口热更新）。
func NewPortalNetwork(cred func() PortalCredential) *PortalNetwork {
	probe := &http.Client{
		Timeout: probeTimeout,
		// 必须禁跟随：状态完全由 302 的 Location 决定（api-docs/003 §2 ①）。
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy:       nil, // 探测必须直连校园网，不能被系统代理劫持
			DialContext: (&net.Dialer{Timeout: probeTimeout}).DialContext,
			// 探测地址是内网，走代理会直接判 offline。
			DisableKeepAlives: true,
		},
	}

	portal := &http.Client{
		Timeout: portalTimeout,
		Transport: &http.Transport{
			Proxy:       nil,
			DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				// 自签证书：**只**作用于 Portal 客户端（AGENTS §5.4）。
				// 教务 / 电费的 HTTPDoer 是另一个 client，绝不共享此配置。
				InsecureSkipVerify: true,
			},
			MaxIdleConnsPerHost: 2,
		},
	}

	return &PortalNetwork{
		probe:  probe,
		portal: portal,
		cred:   cred,
		sleep:  sleepCtx,
		base:   portalBase,
	}
}

// Status 探测判态；探测失败归 offline，不返回错误（docs/002 §3.3）。
func (p *PortalNetwork) Status(ctx context.Context) (model.CampusStatus, error) {
	probe, err := p.probeOnce(ctx)
	if err != nil {
		return model.CampusStatus{State: model.CampusOffline}, nil
	}
	return model.CampusStatus{
		State:    probe.State,
		ErrorMsg: probe.ErrorMsg,
	}, nil
}

// Login 登录。已在线 / 不在校园网时直接返回（不误报失败）。
func (p *PortalNetwork) Login(ctx context.Context) error {
	probe, err := p.probeOnce(ctx)
	if err != nil || probe.State == model.CampusOffline {
		return model.NewError(model.CodeNetworkDown, "不在校园网环境，无法认证")
	}
	if probe.State == model.CampusOnline {
		return nil // 已在线：重复认证无意义
	}

	cred := p.cred()
	if strings.TrimSpace(cred.UserID) == "" || cred.Password == "" {
		return model.NewError(model.CodeCredentialMissing, "未配置校园网账号或密码")
	}

	form := campusnet.BuildLoginForm(probe.Params, cred.UserID, cred.Password, cred.Fingerprint)
	ok, msg, err := p.post(ctx, portalLogin, form)
	if err != nil {
		return err
	}
	if !ok {
		// 上游文案先脱敏再进错误（AGENTS §5、P2）。
		return model.NewError(model.CodeUpstreamRejected, sanitizePortalMessage(msg))
	}
	return p.confirm(ctx, model.CampusOnline)
}

// Logout 下线。distoken 必须来自本次探测（api-docs/003 §2 ③）。
func (p *PortalNetwork) Logout(ctx context.Context) error {
	probe, err := p.probeOnce(ctx)
	if err != nil || probe.State == model.CampusOffline {
		return model.NewError(model.CodeNetworkDown, "不在校园网环境，无法下线")
	}
	if probe.State != model.CampusOnline || probe.Distoken == "" {
		return model.NewError(model.CodeUpstreamRejected, "当前未在线，无法下线")
	}

	form := campusnet.BuildLogoutForm(probe.Params, probe.Distoken)
	ok, msg, err := p.post(ctx, portalDisconn, form)
	if err != nil {
		return err
	}
	if !ok {
		return model.NewError(model.CodeUpstreamRejected, sanitizePortalMessage(msg))
	}
	// 下线后复探的期望是「不再在线」：未认证或不在校园网都算下线成功。
	return p.confirm(ctx, model.CampusUnauth)
}

// Relogin 重登：下线 → 等生效 → 登录 → 等生效 → 复探确认。
func (p *PortalNetwork) Relogin(ctx context.Context) error {
	// 已在线才先下线；未认证态直接登录，避免无谓的下线请求。
	if probe, err := p.probeOnce(ctx); err == nil && probe.State == model.CampusOnline {
		if err := p.Logout(ctx); err != nil {
			return err
		}
	}
	// Login 内部已包含复探确认，这里不再重复。
	return p.Login(ctx)
}

// EnsureCampusNetwork 是自动模式的前置：已在线不动，未认证就登录，
// 不在校园网就返回错误但不重试（由调用方的指数退避接管）。
func (p *PortalNetwork) EnsureCampusNetwork(ctx context.Context) error {
	probe, err := p.probeOnce(ctx)
	if err != nil || probe.State == model.CampusOffline {
		return model.NewError(model.CodeNetworkDown, "不在校园网环境")
	}
	if probe.State == model.CampusOnline {
		return nil
	}
	return p.Login(ctx)
}

// probeOnce 发一次探测请求，返回判态结果；只在网络层失败时返回错误。
func (p *PortalNetwork) probeOnce(ctx context.Context) (campusnet.Probe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, portalProbeURL, nil)
	if err != nil {
		return campusnet.Probe{State: model.CampusOffline}, err
	}
	resp, err := p.probe.Do(req)
	if err != nil {
		return campusnet.Probe{State: model.CampusOffline}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	return campusnet.ParseProbe(resp.StatusCode, resp.Header.Get("Location")), nil
}

// post 发 Portal 表单；成败只看 body 的 success（HTTP 恒 200）。
func (p *PortalNetwork) post(ctx context.Context, path string, form url.Values) (bool, string, error) {
	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.base+path, strings.NewReader(body))
	if err != nil {
		return false, "", model.WrapError(model.CodeNetworkDown, "构造认证请求失败", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", p.base+"/portal/webauth")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := p.portal.Do(req)
	if err != nil {
		return false, "", model.WrapError(model.CodeNetworkDown, "认证请求失败", err)
	}
	defer resp.Body.Close()

	var out struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Code    int    `json:"code"`
		Result  struct {
			Message string `json:"message"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&out); err != nil {
		return false, "", model.WrapError(model.CodeMalformedResponse, "认证响应无法解析", err)
	}

	msg := out.Message
	if msg == "" {
		msg = out.Result.Message
	}
	return out.Success, msg, nil
}

// confirm 等 BRAS 生效后复探一次，把「受理成功」变成「状态已按预期改变」。
//
// want 是期望状态：登录后期望在线，下线后期望不在线（未认证）。
// 只信 HTTP 200 + success:true 是不够的 —— 认证受理后 BRAS 仍可能失败。
func (p *PortalNetwork) confirm(ctx context.Context, want model.CampusState) error {
	if err := p.sleep(ctx, settleDelay); err != nil {
		return err
	}
	probe, err := p.probeOnce(ctx)
	if err != nil {
		return model.WrapError(model.CodeNetworkDown, "认证后复探失败", err)
	}

	if want == model.CampusOnline {
		switch probe.State {
		case model.CampusOnline:
			return nil
		case model.CampusUnauth:
			return model.NewError(model.CodeUpstreamRejected, "认证未生效，请稍后重试")
		default:
			return model.NewError(model.CodeNetworkDown, "认证后不在校园网环境")
		}
	}

	// 下线：未认证即成功；已在线说明下线没生效。探测彻底失败时无法判断，
	// 按「不在校园网」处理（用户已经离开校园网，目标已达成）。
	if probe.State == model.CampusOnline {
		return model.NewError(model.CodeUpstreamRejected, "下线未生效，请稍后重试")
	}
	return nil
}

// sanitizePortalMessage 只放行已知的短文案形态，其余一律替换。
//
// 为什么需要：登录失败时上游可能回显账号等敏感内容（P2），
// 而这条文案会进用户可见的错误与日志。
func sanitizePortalMessage(msg string) string {
	m := strings.TrimSpace(msg)
	if m == "" {
		return "认证被拒绝"
	}
	known := []string{"账号或密码错误", "此IP已在线", "已在线", "正在进行外网拨号", "下线失败", "非法请求", "认证失败"}
	for _, k := range known {
		if strings.Contains(m, k) {
			return k
		}
	}
	return "认证被拒绝"
}

// sleepCtx 是可中断的等待（测试里替换成立即返回）。
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

var (
	_ core.CampusNetwork = (*PortalNetwork)(nil)
	_ core.NetworkReady  = (*PortalNetwork)(nil)
)

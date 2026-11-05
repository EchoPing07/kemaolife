// Package campusnet 是校园网 Portal 的**纯逻辑**部分（api-docs/003）。
//
// 这里只做两件事：把探测响应判成三态 + 构造登录 / 下线表单。
// 探测与 POST 的 HTTP I/O、自签证书策略、网关地址全部留在 internal/platform，
// core 不持有客户端、不知道服务器在哪（001 §4）。
package campusnet

import (
	"net/url"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// Params 是探测 302 下发、登录 / 下线必须原样回传的六参数（api-docs/003 §2）。
//
// 这些值是 BRAS 根据请求来源判定出来的，不能自己拼（例如本机内网 IP）。
type Params struct {
	WlanAcIP   string // wlanacip
	WlanAcName string // wlanacname
	WlanUserIP string // wlanuserip
	Mac        string // mac，小写冒号
	Vlan       string // vlan
	URL        string // url
}

// Probe 是探测结果：三态 + 登录 / 下线所需参数。
type Probe struct {
	State    model.CampusState
	Params   Params
	Distoken string // 仅 Online 态有值，下线必填且须新鲜
	ErrorMsg string // 历史提示文案，只用于展示，不参与判态
}

// ParseProbe 是判态纯函数：由状态码 + Location 得出三态与六参数。
//
// 判定依据只有 302 的 `act`：`LOGINSUCC` = 已在线；跳 `/portal/webauth`
// 且无 act = 未认证；其余（超时、非 302、无 Location、非 Portal 路径）
// 一律视为不在校园网，**不是错误**（docs/002 §3.3）。
func ParseProbe(statusCode int, location string) Probe {
	if statusCode != 302 || strings.TrimSpace(location) == "" {
		return Probe{State: model.CampusOffline}
	}
	u, err := url.Parse(location)
	if err != nil || !strings.Contains(u.Path, "/portal/webauth") {
		return Probe{State: model.CampusOffline}
	}

	q := u.Query()
	p := Probe{
		Params: Params{
			WlanAcIP:   q.Get("wlanacip"),
			WlanAcName: q.Get("wlanacname"),
			WlanUserIP: q.Get("wlanuserip"),
			Mac:        q.Get("mac"),
			Vlan:       q.Get("vlan"),
			URL:        q.Get("url"),
		},
		Distoken: q.Get("distoken"),
		ErrorMsg: q.Get("errorMsg"),
	}
	if q.Get("act") == "LOGINSUCC" {
		p.State = model.CampusOnline
	} else {
		p.State = model.CampusUnauth
	}
	return p
}

// 常量字段值（api-docs/003 §2 ②）。
const (
	LoginMode         = "password"
	RecordTerminalMac = "Y"
)

// BuildLoginForm 构造登录表单：11 个字段 = 六参数 + 账号密码 + 3 个常量。
//
// 只返回表单，不发请求；密码只在内存里经过一次。
func BuildLoginForm(p Params, userID, password, fingerprint string) url.Values {
	return url.Values{
		"wlanacip":           {p.WlanAcIP},
		"wlanacname":         {p.WlanAcName},
		"wlanuserip":         {p.WlanUserIP},
		"mac":                {p.Mac},
		"vlan":               {p.Vlan},
		"url":                {p.URL},
		"userId":             {userID},
		"passwd":             {password},
		"loginMode":          {LoginMode},
		"recordTerminalMac":  {RecordTerminalMac},
		"browserFingerprint": {fingerprint},
	}
}

// BuildLogoutForm 构造下线表单。
//
// 两个已实测的易错点（api-docs/003 §2 ③）：
//   - distoken 必填且须来自**本次探测**，空值会被拒（500 非法请求）；
//   - mac 提交前转大写（下发是小写）。
//
// pageid / act / ssid 填空串即可，旧文档要求填 pageid 已被证伪。
func BuildLogoutForm(p Params, distoken string) url.Values {
	return url.Values{
		"wlanacip":   {p.WlanAcIP},
		"wlanacname": {p.WlanAcName},
		"wlanuserip": {p.WlanUserIP},
		"mac":        {strings.ToUpper(p.Mac)},
		"vlan":       {p.Vlan},
		"url":        {p.URL},
		"distoken":   {distoken},
		"pageid":     {""},
		"act":        {""},
		"ssid":       {""},
	}
}

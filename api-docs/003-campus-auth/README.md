# 校园网认证（Campus Portal）

> **状态**：已验证（Portal 方式）
> **更新**：2026-11-05
> **适用**：广东科贸职业学院清远校区 Portal（`km-bras`）

**核心链路只有 3 个请求，全部无 Cookie、无 Token、无签名。**

---

## 1. 核心事实

| 项 | 值 |
|---|---|
| Portal 基地址 | `https://10.0.128.3`（**自签证书**，仅此地址跳过 TLS 校验） |
| 探测地址 | `http://5.5.5.5/`（**必须 http**，https 拿不到 302） |
| 重定向 | **一律禁跟随**，自己读 `Location` |
| 鉴权 | 无。凭据只有学号 + 密码（明文） |
| 请求格式 | `application/x-www-form-urlencoded` |
| 响应判定 | HTTP 恒为 **200**，成败只看 body 的 `success`（`code:500` 是业务错误，不是 HTTP 500） |
| 生效确认 | 登录 / 下线后约 2~3s 再探测一次即可 |

---

## 2. 三个核心请求

### ① 探测：`GET http://5.5.5.5/`

唯一的状态判定依据，同时产出登录所需的全部参数。

```
HTTP/1.1 302
Location: https://10.0.128.3/portal/webauth?wlanacip=<BRAS 地址>&wlanacname=km-bras
  &wlanuserip=<本机内网 IP>&mac=<本机 MAC>&vlan=0
  &act=LOGINSUCC&distoken=<下线凭据>&url=http://5.5.5.5
```

| 判定 | 条件 | 动作 |
|---|---|---|
| **已在线** | 302 且 Location 含 `act=LOGINSUCC` | 无需登录；`distoken` 是新鲜的下线凭据 |
| **未认证** | 302 且 Location 指向 `/portal/webauth`、**无 `act`** | 取 Location 六参数去登录 |
| **不在校园网** | 超时 / 非 302 / Location 不含 `/portal/webauth` | 不登录，下轮重试 |

**六参数**（由 BRAS 下发，不要自己拼）：

| 参数 | 说明 |
|---|---|
| `wlanacip` | BRAS 地址 |
| `wlanacname` | BRAS 名称（如 `km-bras`） |
| `wlanuserip` | 本机内网 IP（BRAS 已识别） |
| `mac` | 本机 MAC，**小写冒号** |
| `vlan` | 通常 `0` |
| `url` | 原访问目标，原样回传 |

> 已在线态的 `errorMsg` 只是历史文案（如顶号提示），仅用于展示，不参与判态。

### ② 登录：`POST https://10.0.128.3/api/portal/webauth`

**11 个字段 = 探测六参数 + 账号密码 + 3 个常量**：

| 字段 | 来源 |
|---|---|
| `wlanacip`, `wlanacname`, `wlanuserip`, `mac`, `vlan`, `url` | ① 的 `Location`，原样回传 |
| `userId` | 学号 |
| `passwd` | 密码（明文） |
| `loginMode` | 常量 `password` |
| `recordTerminalMac` | 常量 `Y` |
| `browserFingerprint` | 前端指纹 visitorId，固定值即可（用配置项，不硬编码抓包值） |

请求头：`Content-Type: application/x-www-form-urlencoded`、`Referer: https://10.0.128.3/portal/webauth`、常规 UA。
**不要带 `Authorization`，不要带 Cookie。**

响应（HTTP 200）：

| `success` | `result.message` | 含义 |
|---|---|---|
| `true` | `正在进行外网拨号请稍候...` | 受理成功，BRAS 拨号中 |
| `true` | `此IP已在线请勿重复认证` | 已在线，无害 |
| `false` | `账号或密码错误`（`code:500`） | 明确拒绝 |

响应中的 `result.token` / `result.distoken` / `redirect` **本链路全部忽略**：`token` 是自服务页用的 Bearer，下线不需要；`distoken` 每次从探测 302 现取更新鲜。

### ③ 下线：`POST https://10.0.128.3/api/portal/webdisconn`

```
探测六参数（原样） + mac 转大写 + distoken（① 现取） + pageid= + act= + ssid=
```

| 易错点 | 结论 |
|---|---|
| `distoken` | **必填且须新鲜**，每次下线前先探测取；空值 → `500 下线失败，非法请求` |
| `mac` | 提交时把 302 下发的小写**转大写** |
| `pageid` | **空串即可成功**（旧文档要求填，已证伪） |
| `wlanacip` | 用探测给的值即可，不必用认证后的另一个地址 |
| 鉴权 | 不需要 token，接口按请求来源 IP 鉴权 |

响应：`{"success":true,"message":"下线成功","code":200,...}`。

---

## 3. 数据结构

```go
// PortalParams 是探测 302 下发、登录/下线必须原样回传的六参数。
type PortalParams struct {
	WlanAcIP   string // wlanacip
	WlanAcName string // wlanacname
	WlanUserIP string // wlanuserip
	Mac        string // mac，小写冒号
	Vlan       string // vlan
	URL        string // url
}

// Probe 是探测结果，只保留判定所需字段。
type Probe struct {
	State    State  // Online / Unauth / Offline
	Params   PortalParams
	Distoken string // 仅 Online 态有值，用于下线
	ErrorMsg string // 历史提示文案，展示用
}

type State int

const (
	StateOffline State = iota // 不在校园网 / 探测失败
	StateUnauth               // 未认证
	StateOnline               // 已在线
)
```

判态是纯函数：`ParseProbe(statusCode int, location string) Probe`。

---

## 4. 示例代码（Go）

```go
package campusnet

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	portalBase    = "https://10.0.128.3"
	probeURL      = "http://5.5.5.5/"
	loginPath     = "/api/portal/webauth"
	disconnPath   = "/api/portal/webdisconn"
	webauthReferer = portalBase + "/portal/webauth"
)

// NewPortalClient 返回一个「探测不跟随重定向 + 仅对 Portal 跳过 TLS 校验」的客户端组。
func NewPortalClient() (probe *http.Client, portal *http.Client) {
	probe = &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // 禁跟随，自己读 Location
		},
	}

	portal = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 仅用于自签 Portal，勿用于其他域名
			},
		},
	}
	return probe, portal
}

// Probe 探测当前认证状态。
func Probe(ctx context.Context, c *http.Client) (Probe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return Probe{State: StateOffline}, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return Probe{State: StateOffline}, err // 超时 = 不在校园网
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return ParseProbe(resp.StatusCode, resp.Header.Get("Location")), nil
}

// ParseProbe 纯函数：由状态码 + Location 判定状态并抽取六参数。
func ParseProbe(status int, location string) Probe {
	if status != http.StatusFound || location == "" {
		return Probe{State: StateOffline}
	}
	u, err := url.Parse(location)
	if err != nil || !strings.Contains(u.Path, "/portal/webauth") {
		return Probe{State: StateOffline}
	}

	q := u.Query()
	p := Probe{
		Params: PortalParams{
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
		p.State = StateOnline
	} else {
		p.State = StateUnauth
	}
	return p
}

// Login 用六参数 + 账号密码认证。成败只看响应 body 的 success。
func Login(ctx context.Context, c *http.Client, p PortalParams, user, pass, fingerprint string) (bool, string, error) {
	form := url.Values{
		"wlanacip":            {p.WlanAcIP},
		"wlanacname":          {p.WlanAcName},
		"wlanuserip":          {p.WlanUserIP},
		"mac":                 {p.Mac},
		"vlan":                {p.Vlan},
		"url":                 {p.URL},
		"userId":              {user},
		"passwd":              {pass},
		"loginMode":           {"password"},
		"recordTerminalMac":   {"Y"},
		"browserFingerprint":  {fingerprint},
	}
	return postPortal(ctx, c, loginPath, form)
}

// Logout 下线：mac 转大写，distoken 必须来自本次探测。
func Logout(ctx context.Context, c *http.Client, p PortalParams, distoken string) (bool, string, error) {
	form := url.Values{
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
	return postPortal(ctx, c, disconnPath, form)
}

func postPortal(ctx context.Context, c *http.Client, path string, form url.Values) (bool, string, error) {
	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, portalBase+path, strings.NewReader(body))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", webauthReferer)
	req.Header.Set("Content-Length", fmt.Sprint(len(body)))

	resp, err := c.Do(req)
	if err != nil {
		return false, "", err
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
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "", err
	}

	msg := out.Message
	if msg == "" {
		msg = out.Result.Message
	}
	return out.Success, msg, nil
}
```

编排（与 `app.EnsureCampusNetwork` 对应）：

```go
probeClient, portalClient := NewPortalClient()

p, err := Probe(ctx, probeClient)
if err != nil || p.State == StateOffline {
	fmt.Println("不在校园网环境")
	return nil
}
if p.State == StateOnline {
	fmt.Println("已在线，无需登录")
	return nil
}

ok, msg, err := Login(ctx, portalClient, p.Params, user, pass, fingerprint)
if err != nil {
	return err
}
fmt.Println("认证受理:", ok, msg)

// 约 3s 后再探测一次确认生效
time.Sleep(3 * time.Second)
p2, _ := Probe(ctx, probeClient)
fmt.Println("复探状态:", p2.State)
```

---

## 5. 别绕远路

- **不需要**：`GET /portal/webauth` 页面、`/api/portal/config`、`Authorization: Bearer`、`/api/portal/user/lastAuthResult`、`/api/portal/user/logout`、`currentOnlineUser`。这些是自服务页 / 旧实现路径。
- **不要用** `/api/portal/user/login`：字段名不同（`accountId` / `password`）且要图形验证码。
- **判定在线 / 离线只看探测 302 的 `act`**，不要用 webauth 的 `success`（已在线时重复登录照样返回 `true`）。
- 自签证书跳过校验**仅限** Portal 与探测地址，不要全局关闭 TLS。
- 密码明文传输，且 `webdisconn` 仅凭来源 IP 鉴权。仅用于本人设备，凭据不得提交仓库。

---

## 6. 凭据与隐私

| 项 | 要求 |
|---|---|
| 账号密码 | 独立配置字段，不复用教务密码；DPAPI（Windows）或 Keystore（Android）加密存储 |
| 日志 | 不打印密码、`wlanuserip`、本机 `mac`、`distoken`、指纹值。Portal / 探测 IP 与 BRAS 名称属学校协议事实，可写入协议文档；用户本机内网 IP / MAC 一律用占位符 |
| 指纹 | `browserFingerprint` 作为配置项，不在仓库硬编码抓包值 |
| 频率 | 认证失败指数退避，起点 = 电费轮询间隔（默认 4 小时），逐步增大且上限大于起点；不重复弹通知。自动模式仅在 Windows 启用；Windows 手动与 Android 手动动作无退避轮询 |

---

## 7. 待验证

| 项 | 说明 |
|---|---|
| 其他校区 / 其他认证方式 | 本文仅覆盖清远校区 Portal；802.1X / 客户端认证需另行抓包 |
| `browserFingerprint` 是否强校验 | 固定值实测可用，是否需与 UA 一致未验证 |
| 探测地址可变性 | `5.5.5.5` 是当前实测值，若 BRAS 调整需改为可配置 |

# 电费接口（SWAEServlet）

> **状态**：已验证
> **更新**：2026-11-05
> **适用**：智能水电 H5 后端（`xqh5.17wanxiao.com`，完美校园 / 新开普体系）

查询学生宿舍的**剩余电量、每日用量、充值记录、订单详情**，以及剩余水量。

---

## 1. 概述

| 项 | 值 |
|---|---|
| 基地址 | `https://xqh5.17wanxiao.com/smartWaterAndElectricityService/` |
| 端点 | `SWAEServlet` |
| 方法 | `POST`（GET 不支持） |
| Content-Type | `application/x-www-form-urlencoded;charset=UTF-8` |
| 鉴权 | **无**。无 Cookie、无 Token、无签名、无密码 |
| 身份凭证 | 学号（放在 `param.account`） |
| 数据格式 | 双层 JSON：`body` 是**字符串**，需二次解析 |
| 编码 | UTF-8 |

> H5 页面 URL 上的 `token=`、`code=`、`command=`、`openid=`、`Bopenid=`、`userid=` **全部不参与接口调用**。
> H5 页面链接上的 `customerid`（学校代码）与 `outid`（学号）只是页面入口参数；实际接口调用使用表单字段 `customercode` 与 `param.account`（见 §3.1 / §3.2）。

---

## 2. 快速验证

```bash
curl -s -X POST 'https://xqh5.17wanxiao.com/smartWaterAndElectricityService/SWAEServlet' \
  -H 'Content-Type: application/x-www-form-urlencoded;charset=UTF-8' \
  --data-urlencode 'param={"cmd":"getbindroom","account":"<学号>"}' \
  --data-urlencode 'customercode=3126' \
  --data-urlencode 'method=getbindroom'
```

返回（敏感字段以占位符呈现）：

```json
{
  "code_": 0,
  "sign": "<服务端签名，客户端不校验>",
  "result_": "true",
  "body": "{\"result\":\"0\",\"roomfullname\":\"<校区>_<楼栋>_<楼层>_<房间>\",\"existflag\":\"1\",\"roomverify\":\"<房间编码>\",\"detaillist\":[{\"sumsub\":0,\"use\":\"<累计用量>\",\"sumbuy\":<本期购入>,\"odd\":\"<剩余量>\",\"businesstype\":\"0\",\"status\":\"1\"}],\"message\":\"操作成功\"}",
  "message_": "获取成功！"
}
```

`body.detaillist[businesstype="0"].odd` 即剩余电量（度）。

---

## 3. 请求约定

### 3.1 表单字段

请求体为 `application/x-www-form-urlencoded`，共 **3 个字段**：

| 字段 | 必填 | 说明 |
|---|---|---|
| `param` | 是 | JSON **字符串**，内容为命令名 + 业务参数 |
| `customercode` | 是 | 学校代码，因校而异；本项目内置默认 `3126`（`cmd` 层，可覆盖），不写成 core 通用常量 |
| `method` | 是 | 命令名，与 `param.cmd` **保持一致** |

### 3.2 `param` 结构

```json
{
  "cmd": "getbindroom",
  "account": "<学号>",
  "...": "该命令的业务参数"
}
```

- `cmd`：命令名，必填，与表单 `method` 相同。
- `account`：**必填**，值为学号。缺失返回 `参数异常：account不存在`。
- 其余字段按命令而定（见 §6）。
- `timestamp` 前端会带，实测**可省略**，服务端不校验。

> 客户端需自行把 `account` 注入 `param`；H5 是由前端封装函数自动注入的。

---

## 4. 响应结构

外层固定包装（最常见形态）：

```json
{
  "code_": 0,              // 0 = HTTP 层成功
  "sign": "<签名>",         // 服务端签名，客户端不校验
  "result_": "true",       // "true" = 后端处理成功
  "body": "{...}",         // 业务数据（JSON 字符串）
  "message_": "获取成功！"
}
```

实测存在三种兼容形状，客户端必须都接受（`internal/core/electricity/flexjson.go`）：

| 字段 | 常见 | 实测变体 |
|---|---|---|
| `code_` | 数字 `0` | 字符串（如拒绝分支的 `"8099"`） |
| `result_` | 字符串 `"true"` | 布尔 `true` / `false`；拒绝分支为 `false` 且可能无 `body` |
| `body` | JSON 字符串 | 直接内联的 JSON 对象 |
| 错误说明 | `message_` | `exception_` |

拒绝分支示例（学校代码 / 前置地址配置问题，不含任何个人信息）：

```json
{"code_":8099,"result_":false,"exception_":"<上游提示>"}
```

业务层还有一层：

```json
{ "result": "0", "message": "操作成功", "...": "业务字段" }
```

**成功判定**：`result_ == "true"` **且** `body.result == "0"`，两者缺一不可。

> 陷阱：缺 `account` 时外层仍然是 `result_:"true"`、`message_:"获取成功！"`，错误只体现在 `body.result`（实测 `-20003`）。必须先解析 `body` 再判成败。

---

## 5. 命令一览

| cmd | 作用 | 必填参数 |
|---|---|---|
| `login` | 校验身份，返回姓名 / 学校 | `account` |
| `getbindroom` | 已绑定房间 + 余额概况 | `account` |
| `getstuindexpage` | 首页数据：近 7 天、历史月份、供电状态；本项目两个主请求成功后固定发一次，仅取 `collecdate` | `account`（可选 `roomverify`） |
| `getusedetail` | 指定月份的每日用量 | `account`, `roomverify`, `businesstype`, `yearmonth` |
| `getrechargelist` | 充值记录（按月分组） | `account`, `roomnum`, `roomverify`, `businesstype`, `chargetype` |
| `getorderdetail` | 单笔订单详情（电价、状态流转） | `account`, `orderid`, `type`, `businesstype`, `chargetype` |
| `getzonelist` | 分区列表（绑定流程第 1 步） | `account` |
| `getbuildlist` | 楼栋列表（第 2 步） | `zoneid` |
| `getfloorlist` | 楼层列表（第 3 步） | `buildid` |
| `getroomnumlist` | 房间列表（第 4 步） | `floorid` |
| `bindroom` | 绑定房间 | `zoneid`, `buildid`, `floorid`, `roomnum` |
| `unbindroom` | 解绑房间 | — |

> 本工具主路径用 **`getbindroom`**（余额）与 **`getusedetail`**（每日用量），另用 **`getstuindexpage`** 的 `collecdate` 作为「数据截至」的准确采集时刻；该请求失败或缺失时降级到 `getusedetail.opdtdate`（只有截止日）。其余命令记录在此以便排查。

---

## 6. 常用命令详解

### 6.1 `getbindroom`

最常用的查询接口，一次拿到剩余电量与房间标识。

**请求**

```json
{"cmd": "getbindroom", "account": "<学号>"}
```

**响应 body**

```json
{
  "result": "0",
  "message": "操作成功",
  "ver": 2,
  "existflag": "1",
  "roomfullname": "<校区>_<楼栋>_<楼层>_<房间>",
  "roomverify": "<房间编码>",
  "isshowsubsidy": "0",
  "detaillist": [
    {
      "businesstype": "0",
      "odd": "<剩余量>",
      "use": "<累计用量>",
      "sumbuy": 0,
      "sumsub": 0,
      "status": "1"
    }
  ]
}
```

| 字段 | 含义 |
|---|---|
| `existflag` | `1` 已绑定，`2` 未绑定（需走绑定流程），`0` 无记录 |
| `roomfullname` | 宿舍全名，**个人信息**，不写日志 |
| `roomverify` | 房间唯一编码，后续查询必用 |
| `detaillist[].businesstype` | `0` 电，`1` 水 |
| `detaillist[].odd` | **剩余量**（度 / 吨），字符串 |
| `detaillist[].use` | 累计使用量 |
| `detaillist[].sumbuy` | 本期购入量 |
| `detaillist[].sumsub` | 补贴量 |
| `detaillist[].status` | `1` 正常送电 |

### 6.2 `getusedetail`

指定月份，返回该月每日用量。

**请求**

```json
{
  "cmd": "getusedetail",
  "account": "<学号>",
  "roomverify": "<房间编码>",
  "businesstype": "0",
  "yearmonth": "2026-11"
}
```

`yearmonth` 格式 `YYYY-MM`，月份需补零（如 `2026-01`）。

**响应 body**

```json
{
  "result": "0",
  "message": "操作成功",
  "yearmonth": "<YYYY-MM>",
  "monthuse": "<当月累计>",
  "opdtdate": "<YYYY-MM-DD>",
  "dayuselist": [
    {"date": "<YYYY-MM-DD>", "daydate": "<YYYY-MM-DD>", "status": "1", "dayuse": "<当日用量>", "use": "<当日用量>"}
  ]
}
```

- `dayuselist` 按日期**倒序**返回，展示前需反转。
- `monthuse` 为当月累计，`opdtdate` 为数据截止日（有延迟，属正常）。
- 单条里 `dayuse` 与 `use` 数值相同，取其一。

### 6.3 `getstuindexpage`（不用于取余额）

首页聚合数据，含近 7 天逐日用量、历史月份、供电状态。

**请求**

```json
{"cmd": "getstuindexpage", "account": "<学号>", "roomverify": "<房间编码>"}
```

`roomverify` 可省略（省略则用已绑定房间）。

响应节选（示例值与真实用电数据无关，仅示意字段形状）：

```json
{
  "result": "0",
  "collecdate": "<采集时间>",
  "status": "<状态>",
  "modlist": [
    {
      "collecdate": "<采集时间>",
      "modstatus": "<供电状态>",
      "isshowpay": 1,
      "weekuselist": [
        {"date": "<MM-DD>", "weekday": "<星期>", "use": "<当日用量>", "dayuse": "<当日用量>"}
      ],
      "monthuselist": [
        {"yearmonth": "<YYYY.MM>", "monthuse": "<当月累计>"}
      ]
    }
  ]
}
```

> ⚠️ 实测 `modlist[0]` **没有** `odd` / `todayuse`，顶层也没有。取余额请用 `getbindroom`，取每日用量请用 `getusedetail`，**不要依赖 `getstuindexpage.odd`**。

### 6.4 `getrechargelist` / `getorderdetail`（不实现）

充值记录与订单详情，含 `sumbuy`（购得电量）、`sumbuyfare`（支付金额）、`price`（电价 元/度）、`orderstatuslist`（状态流转）。本工具不做充值，仅此记录以备排查。

### 6.5 房间绑定流程（不实现）

未绑定时（`existflag != "1"`）H5 走四级联动：

```
getzonelist → getbuildlist(zoneid) → getfloorlist(buildid)
            → getroomnumlist(floorid) → bindroom(zoneid, buildid, floorid, roomnum)
```

`bindroom` / `unbindroom` 会改变账号绑定状态；本工具不实现，参数取自前端源码，未实测。

---

## 7. 字段字典

### businesstype

| 值 | 含义 |
|---|---|
| `0` | 电（单位：度） |
| `1` | 水（单位：吨） |

### existflag

| 值 | 含义 |
|---|---|
| `1` | 已绑定房间 |
| `2` | 未绑定，需走绑定流程 |
| `0` | 无绑定记录 |

### status（表状态）

| 值 | 含义 |
|---|---|
| `1` | 正常 |
| 其他 | 异常 / 欠费停机（取值未穷举） |

### type / chargetype

| 值 | 含义 |
|---|---|
| `1` | 购电 / 购水 |
| `3` | 退电 / 退水 |
| `4` | 清零 |

`getrechargelist` 入参 `chargetype=0` 表示不过滤。

### opresult

| 值 | 含义 |
|---|---|
| `100` | 成功 |

---

## 8. 错误码

业务错误在 `body.result`，`body.message` 给中文说明。

| result | 含义 |
|---|---|
| `0` | 成功 |
| `-1` | 必填参数缺失 |
| `-20003` | 参数缺失或学号无效（**外层仍 `result_:"true"`**） |
| `-10005` | body 内层失败 |
| `-90001` | 未获取到数据（`body` 为 `{}`） |
| `-90002` | `body` 非法 JSON |
| `-90003` | `body` 为空 |
| `-90004` | `result_ != "true"` |

> `-9000x` 是前端解析响应时生成的，服务端不返回。

---

## 9. 示例代码（Go）

零第三方依赖，标准库即可。

```go
package electricity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const endpoint = "https://xqh5.17wanxiao.com/smartWaterAndElectricityService/SWAEServlet"

type Client struct {
	HTTPDoer interface {
		Do(*http.Request) (*http.Response, error)
	}
}

// call 发一次 SWAEServlet 请求，返回已解开双层的业务 body。
func (c *Client) call(ctx context.Context, cmd, account, schoolCode string, extra map[string]string) (map[string]any, error) {
	payload := map[string]any{"cmd": cmd, "account": account}
	for k, v := range extra {
		payload[k] = v
	}
	paramJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"param":        {string(paramJSON)},
		"customercode": {schoolCode},
		"method":       {cmd},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	resp, err := c.HTTPDoer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var outer struct {
		Code      int    `json:"code_"`
		Result    string `json:"result_"`
		Body      string `json:"body"`
		Message   string `json:"message_"`
		Exception string `json:"exception_"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&outer); err != nil {
		return nil, fmt.Errorf("外层 JSON 解析失败: %w", err)
	}
	if outer.Result != "true" {
		// 上游文案只用于错误分类，不得原样进 error.message（P2）。
		_ = outer.Message
		_ = outer.Exception
		return nil, fmt.Errorf("电费接口请求失败")
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(outer.Body), &body); err != nil {
		return nil, fmt.Errorf("body 解析失败: %w", err)
	}
	if fmt.Sprint(body["result"]) != "0" {
		// body.message 同样可能夹带个人信息，不外传。
		_ = body["message"]
		return nil, fmt.Errorf("电费业务失败")
	}
	return body, nil
}

// Balance 取剩余电量（度）。只查本人。
func (c *Client) Balance(ctx context.Context, account, schoolCode string) (float64, string, error) {
	body, err := c.call(ctx, "getbindroom", account, schoolCode, nil)
	if err != nil {
		return 0, "", err
	}

	roomVerify, _ := body["roomverify"].(string)
	list, _ := body["detaillist"].([]any)
	for _, item := range list {
		d, ok := item.(map[string]any)
		if !ok || fmt.Sprint(d["businesstype"]) != "0" {
			continue
		}
		var odd float64
		if _, err := fmt.Sscan(fmt.Sprint(d["odd"]), &odd); err != nil {
			return 0, roomVerify, fmt.Errorf("odd 不是数字: %v", d["odd"])
		}
		return odd, roomVerify, nil
	}
	return 0, roomVerify, fmt.Errorf("未找到电表记录（businesstype=0）")
}

// DailyUsage 取某月每日用量，按日期升序返回。
func (c *Client) DailyUsage(ctx context.Context, account, schoolCode, roomVerify, yearMonth string) ([]DayUsage, error) {
	body, err := c.call(ctx, "getusedetail", account, schoolCode, map[string]string{
		"roomverify":  roomVerify,
		"businesstype": "0",
		"yearmonth":   yearMonth,
	})
	if err != nil {
		return nil, err
	}

	list, _ := body["dayuselist"].([]any)
	out := make([]DayUsage, 0, len(list))
	for _, item := range list { // 接口倒序，逐步前插
		d, ok := item.(map[string]any)
		if !ok {
			continue
		}
		use, _ := d["dayuse"].(float64)
		date, _ := d["date"].(string)
		out = append([]DayUsage{{Date: date, Use: use}}, out...)
	}
	return out, nil
}

type DayUsage struct {
	Date string
	Use  float64
}
```

调用：

```go
c := &electricity.Client{HTTPDoer: http.DefaultClient}

balance, roomVerify, err := c.Balance(ctx, account, schoolCode)
if err != nil {
	return err
}
fmt.Printf("剩余 %.2f 度（数据截至 %s）\n", balance, dataUntil)

usage, err := c.DailyUsage(ctx, account, schoolCode, roomVerify, "<YYYY-MM>")
if err != nil {
	return err
}
for _, d := range usage {
	fmt.Println(d.Date, d.Use)
}
```

> `ctx` 由 CLI 的 `--timeout` 与 `HTTPDoer` 统一控制（默认 30s）；轮询间隔 ≥ 600 秒（默认 4 小时）。`Balance` 与 `DailyUsage` 是 `core` 内部两个主请求；两个主请求成功后还会固定发一次 `getstuindexpage` 取采集时刻，失败静默降级到 `opdtdate`；对外一次 `ElectricityReport` 查询同时取余额与用量，主请求任一失败整体报错（见 `docs/002` §3.2）。

---

## 10. 注意事项

1. **接口无鉴权（IDOR）**。仅填学号即可查询对应学生的姓名、宿舍、每日用电曲线。
   仅可用于查询**本人**数据；批量遍历、公开服务化、售卖查询均属违法
   （《刑法》285 条、《个人信息保护法》）。上游返回的 `message_` / `exception_` / `body.message` 可能夹带个人信息，客户端必须先脱敏再写入错误输出。
2. **学校代码因校而异**（本项目实测值 `3126`）。作为 `cmd` 层内置默认值下发并支持用户覆盖，不要写成 `internal/core` 通用常量对外发布；获取方式为公众号菜单链接中的 `customerid=`。
3. **隐私字段**。响应含真实姓名（`login.empname`）、楼栋房间号、用电作息，属个人信息，勿记录、勿分享、勿写日志。
4. **调用频率**。服务端未见频控，但不代表无日志。轮询 ≥ 600 秒/次；Windows 默认 4 小时一次，设置低于下限钳制到 600 秒并提示。
5. **数据延迟**。电表约 1 天延迟（`opdtdate` 为截止日，`collecdate` 为采集时刻），不能当实时功率使用；CLI 输出、告警通知与 Android 电费页必须标注「数据截至」（Windows 托盘按 `docs/002 §5.2` 不展示采集时刻）。
6. **接口稳定性**。学校侧可能升级网关、加签名或鉴权，届时本文档失效。
7. **`sign` 字段**是响应签名，客户端不校验；请求侧无签名机制。
8. **字段以实测为准**。`getstuindexpage` 的 `odd` / `todayuse` 实测缺失，查余额用 `getbindroom`；外层 `result_` 的布尔写法、`body` 内联对象、`code_:8099` 与 `exception_` 均为实测形状。

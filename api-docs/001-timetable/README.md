# 课表接口（教务系统）

> **状态**：已验证（2026-11，Linux CLI M1 实测）
> **更新**：2026-11-05
> **适用**：广东科贸职业学院教务系统（jsxsd）
> **实现参考**：[001-架构设计.md](../../docs/001-架构设计.md)、`internal/core/timetable/protocol.go`

本文描述如何从教务系统拿到课表数据并在本地计算。**1 次 CAS 登录 + 2 个数据请求**产出快照；`Query` 本身是纯函数（0 请求），但 M1 起**每次课表查询会先刷新快照**（含 1 次可选的会话探活），刷新失败才回落旧快照。

---

## 1. 链路（实测）

| 步骤 | 动作 | 产出 |
|---|---|---|
| ① | `GET https://aic.gdkm.edu.cn/cas_server/login?service=<见下>` | 登录页 HTML（ASP.NET WebForms 隐藏域） |
| ② | `POST` 同一 URL（带隐藏域 + 账号密码） | 302 → `http://jw.gdkm.edu.cn/jsxsd/?ticket=...` |
| ③ | 跟随 302 → `/jsxsd/` | 302 → `/jsxsd/;bzb_jsxsd=...` 再 302 → 主页 |
| ④ | `GET /jsxsd/framework/mainV_index_loadkb.htmlx?zc=1` | 第 1 周 7 个日期（`MM-DD`）+ 6 个大节时间 |
| ⑤ | `POST /jsxsd/xskb/xskb_list.do?viweType=1&needData=1` | 课表 JSON（`{"code":0,"data":[...]}`） |
| ⑥ | 本地计算 | `week = floor((目标日 − 第 1 周周一) / 7) + 1`，按「星期 + 周次」过滤 |

**两个主机，不是同一个**：

| 主机 | 作用 |
|---|---|
| `aic.gdkm.edu.cn` | 统一身份认证（CAS）。登录页、`POST /cas_server/login` |
| `jw.gdkm.edu.cn` | 教务 jsxsd。所有课表数据接口 |

> `service` 参数必须是 **`http://jw.gdkm.edu.cn/jsxsd/`**：
> 用 `https` 或指向 `aic` 主机会被 CAS 直接跳到 `/cas_server/error?host=...`。

**会话 Cookie 有两个，缺一不可**：`bzb_jsxsd`（业务会话）、`SERVERID_jsxsd`（后端粘性）。
只有前者时，后续所有请求都会被回退成登录页（HTTP 200，约 179 字节）。

> ⚠️ 中间跳 `GET /jsxsd/` 的 `Location` 把 Cookie 拼在**路径参数**里：
> `/jsxsd/;<Cookie 名>=<会话值>`。标准 `Set-Cookie` 头里也有一份，但若 HTTP 客户端
> 自动跟随重定向，中间那一跳的头会被丢弃 —— 必须**手动跟随**并逐跳吸收 Cookie，
> 同时把 `;name=value` 路径参数也当 Cookie 解析。文档与代码注释不复述真实会话值。

---

## 2. 前置条件

- 学号 + 教务密码。密码**明文提交、无验证码**，不得写入仓库或日志。
- 客户端须**禁用 IPv6**：`aic` 与 `jw` 的 DNS 都返回 AAAA 记录，但服务器没有 IPv6 监听，走 v6 只能等连接超时（约 10s）再回退，每次请求都白等。
- **User-Agent 必须设置**（实测缺失会被拦）。
- 会话 Cookie 可复用；快照内含 Cookie，视同凭据。

---

## 3. 构造与方法

```go
client := timetable.NewClient(httpDoer, clock, casBase, jsbBase) // 后两个留空即用实测默认

err := client.Login(ctx, user, pass)   // 失败返回 4003
alive, err := client.IsAlive(ctx)      // 探测 Cookie 是否仍有效
snap, err := client.Sync(ctx)          // 2 个数据请求
day := timetable.Query(snap, date)     // 纯函数，0 请求
```

| 方法 | 输入 | 输出 |
|---|---|---|
| `Login(ctx, user, pass)` | 学号、密码 | 成功返回 nil；失败区分凭据错误（4003）与网络错误（4001） |
| `IsAlive(ctx)` | — | `bool`；会话失效也返回 200，判据是响应内容 |
| `Sync(ctx)` | — | `*Snapshot` |
| `Query(snap, date)` | 快照、`Date` | `DaySchedule` |
| `QueryRange(snap, from, to)` | 快照、起止日期 | `[]DaySchedule`（含首含尾） |
| `QueryWeek(snap, n)` | 快照、周次 | 7 天 `[]DaySchedule` |

---

## 4. 数据结构

### 4.1 `Snapshot`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Version` | `int` | 快照格式版本，当前 `1`；不匹配即丢弃重拉 |
| `SyncedAt` | `time.Time` | 同步时间 |
| `Week1Monday` | `Date` | 第 1 周周一（日期锚点） |
| `Week1Dates` | `[]Date` | 第 1 周 7 个日期 |
| `SlotTimes` | `[]string` | 6 个大节时间，如 `08:30~10:00` |
| `LastWeek` | `int` | 由课程推断的学期最后一周 |
| `Lessons` | `[]Lesson` | 展开后的课节（一条课程可能展开成多条） |
| `Cookies` | `map[string]string` | 会话 Cookie，**敏感** |

### 4.2 `Lesson`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Name` | `string` | 课程名 |
| `Periods` | `string` | **小节**节次，如 `1-2`、`7-8`、`11-12` |
| `Weekday` | `int` | 周一 = 1 … 周日 = 7 |
| `Weeks` | `[]int` | 上课周次（升序） |
| `Teacher` | `string` | 教师 |
| `Location` | `string` | 地点（多个用 `;` 分隔，已去重） |
| `WeeksText` | `string` | 周次文本，如 `1-16周`、`2-4,7-8,10-19周` |

### 4.3 `DaySchedule`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Date` | `Date` | `YYYY-MM-DD` |
| `Week` | `int` | 周次，从 1 起 |
| `Weekday` | `int` | 周一 = 1 … 周日 = 7 |
| `WeekdayCn` | `string` | 一 / 二 / … / 日 |
| `InSemester` | `bool` | 是否在本学期范围 |
| `Entries` | `[]Entry` | 当天课节，空数组表示没课 |

`Entry` 字段：`Name`、`Periods`、`Time`、`Teacher`、`Location`、`WeeksText`。

---

## 5. 协议字段（实测）

### 5.1 CAS 登录表单

ASP.NET WebForms，三组隐藏域必须原样回传：

| 字段 | 说明 |
|---|---|
| `__VIEWSTATE` | 从登录页 HTML 提取 |
| `__VIEWSTATEGENERATOR` | 同上 |
| `__EVENTVALIDATION` | 同上 |
| `signupInputUserName` | 学号 |
| `signupInputPassword` | 密码（明文） |
| `btn_login` | 常量 `登录` |

请求头需带 `Content-Type: application/x-www-form-urlencoded`、`Referer`（登录页 URL）、常规 `User-Agent`。

**成败判定**：不看状态码（凭据错误时也可能 200），看是否拿到两个会话 Cookie。

### 5.2 周锚点页 `mainV_index_loadkb.htmlx?zc=1`

教务只渲染表格，日期是 **`MM-DD`**（如 `08-31`），**没有年份**。

- 表格第一列的标签是「第一大节 … 第六大节」，共 **6 个大节**；
- 时间以 `HH:MM-HH:MM` 形式出现在表中。

年份推断：用当前年份与上一年各试一次，取能拼出「连续 7 天且首日为周一」的那组。

### 5.3 课表 JSON `xskb_list.do`（POST）

表单字段（固定）：

```
viweType=1&needData=1&baseUrl=/jsxsd&sfykb=2
&xsflMapListJsonStr=理论学时,实践学时,&pageNum=1&pageSize=300
```

响应：

```json
{"msg":"","code":0,"count":14,"data":[ ... ]}
```

`code != 0` 为业务失败。`data[]` 是课程行。

### 5.4 课程行字段

| 字段 | 含义 |
|---|---|
| `kc_mc` | 课程名 |
| `sktime` | 上课安排，见下 |
| `jg0101mc` | 教师 |
| `skddmc` | 地点（多个用 `;` 分隔，实测会重复拼接同一地点） |
| `skxqmc` | 校区 |
| `kch` | 课程号 |
| `kt_mc` | 教学班 |

### 5.5 `sktime` 语法

**星期 + 节次 + 周次压在一个字段里**，多条用 `;` 分隔：

```
周三第1、2节{2,4,7-10}全周
周二第5-8节{19}全周
周四第5、6节{9}全周
周三第3、4节{19}全周;周四第5、6节{9}全周
```

| 片段 | 含义 |
|---|---|
| `周三` | 星期（周一 … 周日） |
| `第1、2节` / `第5-8节` | **小节**节次，取值 1..12 |
| `{2,4,7-10}` | 周次集合，逗号分隔，支持区间 |
| `全周` / `单周` / `双周` | 单双周标记 |

> ⚠️ **小节 vs 大节**：教务只有 6 个大节，但 `sktime` 用的是**小节号 1..12**。
> 换算：`大节序号 = ceil(小节号 / 2)`，即
> `1-2 → 第1大节, 3-4 → 第2, 5-6 → 第3, 7-8 → 第4, 9-10 → 第5, 11-12 → 第6`。
> 直接拿小节号当大节下标，会让第 7 节以后的课拿不到时间。

### 5.6 大节时间

教务不下发「大节时间表」字段，只在表格里渲染。实测值：

| 大节 | 时间 |
|---|---|
| 第 1 大节 | `08:30~10:00` |
| 第 2 大节 | `10:20~11:50` |
| 第 3 大节 | `14:00~15:20` |
| 第 4 大节 | `15:30~16:50` |
| 第 5 大节 | `18:30~19:50` |
| 第 6 大节 | `20:00~21:20` |

若学校调整作息，需要更新 `internal/core/timetable/protocol.go` 的 `SlotTimes`。

---

## 6. 示例代码（Go）

### 6.1 登录并同步

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/timetable"
	"github.com/EchoPing07/kemaolife/internal/platform"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// casBase / jsbBase 传空即用实测默认值（aic + jw）。
	client := timetable.NewClient(
		platform.NewHTTPDoer(60*time.Second), // 强制 IPv4 + 超时
		platform.RealClock{},
		"", "",
	)

	if err := client.Login(ctx, "<学号>", "<密码>"); err != nil {
		log.Fatalf("登录失败: %v", err)
	}

	snap, err := client.Sync(ctx) // 2 个数据请求
	if err != nil {
		log.Fatalf("同步失败: %v", err)
	}
	log.Printf("快照已生成：第 1 周周一 %s，最后一周 %d，课节 %d 条",
		snap.Week1Monday, snap.LastWeek, len(snap.Lessons))
}
```

### 6.2 查询单日（`Query` 纯函数，0 请求）

```go
day := timetable.Query(snap, model.NewDate(time.Now()))
log.Printf("第 %d 周 星期%s", day.Week, day.WeekdayCn)
for _, e := range day.Entries {
	log.Printf("第%s节 %s | %s | %s | %s", e.Periods, e.Time, e.Name, e.Location, e.Teacher)
}
if len(day.Entries) == 0 {
	log.Println("当天没课")
}
if !day.InSemester {
	log.Println("该日期不在本学期范围")
}
```

### 6.3 查询指定周

```go
week := timetable.QueryWeek(snap, 10) // 7 天
for _, d := range week {
	for _, e := range d.Entries {
		log.Printf("%s 星期%s 第%s节 %s", d.Date, d.WeekdayCn, e.Periods, e.Name)
	}
}
```

### 6.4 复用快照（离线）

```go
snap, err := timetable.LoadSnapshot(store) // 从 KVStore 读并校验 Version
if err != nil {
	return err // 3003：快照缺失，提示用户执行 sync
}

// 无网络也可查询。
day := timetable.Query(snap, model.Date{Year: 2026, Month: 11, Day: 5})
```

### 6.5 会话失效后重登

```go
alive, _ := client.IsAlive(ctx)
if !alive {
	if err := client.Login(ctx, "<学号>", "<密码>"); err != nil {
		return err
	}
}
```

---

## 7. 必须记住的六条

1. **两个主机** —— CAS 在 `aic`，教务在 `jw`；`service` 必须是 `http://jw.gdkm.edu.cn/jsxsd/`（http，不是 https）。
2. **两个 Cookie** —— `bzb_jsxsd` + `SERVERID_jsxsd`，缺一不可。
3. **手动跟随重定向** —— 中间跳把 Cookie 塞在 `Location` 的路径参数里（`/jsxsd/;bzb_jsxsd=...`）；自动跟随会丢掉它。
4. **小节 ≠ 大节** —— `sktime` 用 1..12 小节号，教务只有 6 个大节，要 `ceil(n/2)` 换算。
5. **会话失效也返回 200** —— 判响应内容（是否含 `signupInputUserName` / `统一身份认证`），不能只看状态码。
6. **强制 IPv4 + 设 UA** —— 两台服务器都没监听 v6；缺 UA 会被拦。

---

## 8. 待验证

| 项 | 说明 |
|---|---|
| 学期切换判定 | 用 `Week1Monday` + `SyncedAt` 双重判断；校历变更时的容忍度需实测 |
| 作息时间表 | `SlotTimes` 目前是实测常量，学校调整作息需同步 |
| 校外可达性 | 教务网关是否必须校园网，需实测（影响 Android 刷新时机，不影响离线查询） |
| 会话探活 | 复用快照 Cookie 时先发 1 次 `IsAlive` 判断会话是否有效；有效可省掉完整 CAS 登录 |
| 其他校区 | 本文覆盖清远校区；其他校区若主机不同需另行实测 |

---

## 10. 变更记录

| 日期 | 内容 |
|---|---|
| 2026-11-05 | 重写为实测协议：CAS 在 aic、jsxsd 在 jw，`service` 必须指向 jw；两个会话 Cookie；手动跟随重定向并解析 `;name=value` 路径参数；课表改 `xskb_list.do` POST；`sktime` 语法与小节/大节换算；锚点页只有 `MM-DD`；补 6 个大节实测时间 |

# 课表接口（教务系统）

> **状态**：已验证
> **更新**：2026-11-05
> **适用**：广东科贸职业学院教务系统
> **实现参考**：[001-架构设计.md](../../docs/001-架构设计.md)

本文描述如何从教务系统拿到课表数据并在本地计算。**1 次登录 + 2 个数据请求**产出快照，此后查任意日期均为**纯本地计算，0 请求**。

---

## 1. 链路

| 步骤 | 动作 | 产出 |
|---|---|---|
| ① | CAS 登录 `https://aic.gdkm.edu.cn/cas_server/login?service=<教务主页>` | Cookie `bzb_jsxsd`、`SERVERID_jsxsd`（缺一不可） |
| ② | `GET /jsxsd/framework/mainV_index_loadkb.htmlx?zc=1` | 第 1 周 7 个日期（日期锚点）+ 6 个大节时间 |
| ③ | `GET /jsxsd/xskb/xskb_list.do?viweType=1&needData=1&xsflMapListJsonStr=理论学时,实践学时,&pageNum=1&pageSize=300` | 全学期课程 JSON（含周次、星期、节次，**不含日期**） |
| ④ | 本地计算 | `week = floor((目标日 − 第 1 周周一) / 7) + 1`，按「星期 + 周次」过滤 |

- `Sync()` = ② + ③，实测约 190ms，产出一份可缓存快照。
- `Query(snapshot, date)` = ④，0 请求、0 副作用。
- 同一学期课程不变，快照可长期复用。

> 不要解析 `loadkb` 的 `<tbody>`：rowspan 合并会错位。课程数据一律取 JSON。

---

## 2. 前置条件

- 学号 + 教务密码。密码**明文提交、无验证码**，不得写入仓库或日志。
- 客户端须 **只用 IPv4**：DNS 返回了 AAAA 记录，但 aic 服务器**并未监听 IPv6**，走 IPv6 只能等连接超时（约 10s）再回退，每次请求都白等。
- 会话 Cookie 可复用；快照内含 Cookie，视同凭据。

---

## 3. 构造与方法

```go
client := timetable.NewClient(httpDoer, clock) // httpDoer / clock 来自 core/ports

err := client.Login(ctx, user, pass)           // 失败返回 ErrAuthFailed
alive, err := client.IsAlive(ctx)              // 探测 Cookie 是否仍有效
snap, err := client.Sync(ctx)                  // 1 登录 + 2 请求
day := timetable.Query(snap, date)             // 纯函数
```

| 方法 | 输入 | 输出 |
|---|---|---|
| `Login(ctx, user, pass)` | 学号、密码 | 成功返回 nil；失败返回错误（区分密码错误与网络错误） |
| `IsAlive(ctx)` | — | `bool` |
| `Sync(ctx)` | — | `*Snapshot` |
| `Query(snap, date)` | 快照、`time.Time` | `DaySchedule` |
| `QueryRange(snap, from, to)` | 快照、起止日期 | `[]DaySchedule`（含首含尾） |

---

## 4. 数据结构

### 4.1 `Snapshot`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Version` | `int` | 快照格式版本，当前 `1`；不匹配即丢弃重拉 |
| `SyncedAt` | `time.Time` | 同步时间 |
| `Week1Monday` | `Date` | 第 1 周周一（日期锚点） |
| `Week1Dates` | `[]Date` | 第 1 周 7 个日期 |
| `SlotTimes` | `[6]string` | 6 个大节时间，如 `08:30~10:00` |
| `LastWeek` | `int` | 由课程推断的学期最后一周 |
| `Lessons` | `[]Lesson` | 展开后的课节 |
| `Cookies` | `map[string]string` | 会话 Cookie，**敏感** |

### 4.2 `Lesson`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Name` | `string` | 课程名 |
| `Periods` | `string` | 节次，如 `1-2` |
| `Weekday` | `int` | 周一 = 1 … 周日 = 7 |
| `Weeks` | `[]int` | 上课周次 |
| `Teacher` | `string` | 教师 |
| `Location` | `string` | 地点 |
| `WeeksText` | `string` | 周次文本，如 `1-16周` |

### 4.3 `DaySchedule`

| 字段 | 类型 | 说明 |
|---|---|---|
| `Date` | `Date` | `YYYY-MM-DD` |
| `Week` | `int` | 周次，从 1 起 |
| `Weekday` | `int` | 周一 = 1 … 周日 = 7 |
| `WeekdayCn` | `string` | 一 / 二 / … / 日 |
| `InSemester` | `bool` | 是否在本学期范围 |
| `Entries` | `[]Entry` | 当天课节，空表示没课 |

`Entry` 字段：`Name`、`Periods`、`Time`、`Teacher`、`Location`、`WeeksText`。

---

## 5. 示例代码（Go）

### 5.1 登录并同步

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/timetable"
	"github.com/EchoPing07/kemaolife/internal/platform"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := timetable.NewClient(
		platform.NewHTTPDoer(), // 强制 IPv4（aic 有 AAAA 但无 v6 监听）+ 超时
		platform.RealClock{},
	)

	if err := client.Login(ctx, os.Getenv("KM_USER"), os.Getenv("KM_PASS")); err != nil {
		log.Fatalf("登录失败: %v", err)
	}

	snap, err := client.Sync(ctx) // 2 个请求
	if err != nil {
		log.Fatalf("同步失败: %v", err)
	}
	fmt.Printf("快照已生成：第 1 周周一 %s，最后一周 %d，课节 %d 条\n",
		snap.Week1Monday, snap.LastWeek, len(snap.Lessons))
}
```

### 5.2 查询单日

```go
day := timetable.Query(snap, time.Now())
fmt.Printf("第 %d 周 星期%s\n", day.Week, day.WeekdayCn)
for _, e := range day.Entries {
	fmt.Printf("第%s节 %s | %s | %s | %s\n", e.Periods, e.Time, e.Name, e.Location, e.Teacher)
}
if len(day.Entries) == 0 {
	fmt.Println("当天没课")
}
if !day.InSemester {
	fmt.Println("该日期不在本学期范围")
}
```

### 5.3 查询指定周

```go
// 第 N 周周一 = 锚点 + 7*(N-1)
monday := snap.Week1Monday.AddDays(7 * (n - 1))
week := timetable.QueryRange(snap, monday, monday.AddDays(6))
for _, d := range week {
	for _, e := range d.Entries {
		fmt.Printf("%s 星期%s 第%s节 %s\n", d.Date, d.WeekdayCn, e.Periods, e.Name)
	}
}
```

### 5.4 复用快照（0 请求）

```go
snap, err := timetable.LoadSnapshot(ctx, store) // 从 KVStore 读并校验 Version
if err != nil {
	return err // 快照缺失 → 提示用户执行 sync
}

// 无网络也可查询
day := timetable.Query(snap, time.Date(2026, 11, 5, 0, 0, 0, 0, time.Local))
fmt.Println(day.Week, day.WeekdayCn, len(day.Entries))
```

### 5.5 覆盖假 200 的会话失效

```go
alive, _ := client.IsAlive(ctx)
if !alive {
	// 会话失效，重新登录一次
	if err := client.Login(ctx, user, pass); err != nil {
		return err
	}
}
```

---

## 6. 必须记住的四条

1. **会话失效也返回 200** —— 必须判响应内容（是否含课程数据 / 是否跳转登录页），不能只看 `StatusCode`。
2. **课程数据走 JSON** —— 不要解析 `loadkb` 的 `<tbody>`。
3. **强制 IPv4** —— DNS 有 AAAA 记录但服务器没监听 IPv6，走 v6 必然先超时（约 10s）。这不是「v6 比较慢」，是「v6 走不通」，因此要**主动禁用 v6 而非优选的 v6 回退**。在 `HTTPDoer` 实现里处理，不在 `core` 里判断。
4. **日期用本地时区** —— 不要用 UTC 转 `YYYY-MM-DD`，会差一天。构造 `time.Time` 时指定 `time.Local`。

---

## 7. 待验证

| 项 | 说明 |
|---|---|
| 学期切换判定 | 用 `Week1Monday` + `SyncedAt` 双重判断；校历变更时的容忍度需实测 |
| 越界周次 | 传入大于 `LastWeek` 的周次，应返回 `InSemester:false` 而非报错 |
| 校外可达性 | 教务网关是否必须校园网，需实测（影响 Android 刷新时机，不影响离线查询） |

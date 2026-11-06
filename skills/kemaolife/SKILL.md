---
name: kemaolife
description: 用 kemaolife CLI 查询广东科贸职业学院的课表与宿舍电费。当用户问「今天/某天有什么课」「第几周课表」「本周课表」「电费还剩多少」「宿舍电量」时使用。固定带 --json 只解析 stdout。
---

# kemaolife（Linux CLI）

供 agent 调用的单二进制命令。三个能力中的两个在 Linux 端可用：**课表查询**、**电费查询**。
**没有校园网能力** —— 不要回答「能不能上网」「校园网掉线了怎么办」，本端不判态、不认证、不存凭据。

## 1. 何时用

| 用户问 | 跑 |
|---|---|
| — | **每次课表查询都会先联网刷新**：连网时是最新数据，断网时自动回落旧快照并标 `stale:true` |
| 今天有什么课 / 现在上什么 | `kemaolife --json timetable`（等价于 `timetable today`） |
| 第 N 周课表 / 本周课表 | `kemaolife --json timetable week [N]`（省略 N = 当周） |
| 某天有没有课 | `kemaolife --json timetable date <YYYY-MM-DD>` |
| 电费还剩多少 / 还能撑几天 | `kemaolife --json electricity` |
| 某月用了多少电 | `kemaolife --json electricity --month <YYYY-MM>` |
| 版本 / 帮助 | `kemaolife --json --version` / `kemaolife help <命令>` |
| 首次使用 / 只想刷新 | `kemaolife --json config init` / `kemaolife --json timetable sync` |
| 配置状态 | `kemaolife --json config show` |

不适用：校园网状态、成绩、选课、考试安排、充值、替别人查。

## 2. 固定用法

- **所有命令带 `--json`**，只解析 `stdout`。`stderr` 只有诊断文本与交互提示，不是数据。全局参数可放在子命令前后；支持 `--flag=value`、`-h` / `-v` 与 `--`。`-v` / `--version` 与 `--` 在子命令前后均可生效，`--` 之后的内容按子命令原样参数处理；子命令之后也可用 `--json` / `--verbose` / `--config` / `--timeout` / `-h` / `--help`。
- 退出码表达结果类别，先看退出码再看 `error.code`：

| 退出码 | 含义 | 怎么办 |
|---|---|---|
| `0` | 成功 | 读 `data` |
| `1` | 业务失败 | 读 `error.message`，向用户说明 |
| `2` | 参数错误 | 修正命令后重试，**不要原样重跑** |
| `3` | 凭据缺失 / 会话失效 / 快照缺失 | 引导用户配置或同步 |
| `4` | 网络或接口不可用 | 课表通常已自动回落旧快照（看 `stale:true` + `syncedAt`，退出码仍 `0`）；电费明确报错 |

信封：`{"ok":true,"data":{...},"ts":"..."}`（课表查询另有 `stale` 与 `syncedAt`）或 `{"ok":false,"error":{"code":3002,"message":"..."},"ts":"..."}`。

## 3. 输出怎么读

### 课表（`DaySchedule`）

示例为字段示意；`<...>` 是占位符字符串，实际数值字段按 DTO 编码。

```json
{
  "date": "<YYYY-MM-DD>", "week": "<周次>", "weekday": "<1-7>", "weekdayCn": "<一/二/…/日>",
  "inSemester": true,
  "entries": [
    {"name": "<课程名>", "periods": "<起始小节>-<结束小节>", "time": "<上课时间>",
     "teacher": "<教师>", "location": "<地点>", "weeksText": "<周次文本>"}
  ]
}
```

- `entries: []` 表示**当天没课**，不是查询失败。
- **`stale: true` 表示本次刷新失败、返回的是旧快照**：必须告诉用户数据时间（`syncedAt`），不要说成「最新课表」。
- `inSemester: false` 表示该日期不在本学期范围（假期 / 越界周次），此时 `entries` 为空；这是**成功**（退出码 `0`），不是错误。
- `week` 从 1 起；`weekday` 周一 = 1、周日 = 7。
- `timetable week [N]` 返回 `{"week":N,"days":[7 天]}`，省略 `N` 为当前周；当前日期不在学期内且省略 `N` 时返回 7 天 `inSemester:false` 的成功结果，不产生 `1002`，请说明当前是假期 / 学期外。
- `periods` 是**小节**节次（1..12），`time` 已按大节换算好；一节大课约 90 分钟，教务共 6 个大节。多门课按起始节次、再按课程名排序。
- `weekday` 周一 = 1、周日 = 7，`weekdayCn` 为「一 / 二 / … / 日」；`entries: []` 表示当天没课，不是失败。
- `location` 可能有多个（`;` 分隔），表示该课程不同周次在不同地点。

### 电费（`ElectricityReport`）

示例为字段示意；`<...>` 是占位符字符串，实际数值字段按 DTO 编码（数值或 `null`）。

```json
{
  "status": {
    "balance": "<剩余量>", "unit": "度", "collectedAt": "<采集时间>",
    "dailyAvg": "<近七日均量或 null>", "estimatedDays": "<估算天数或 null>",
    "roomName": "<校区>_<楼栋>_<楼层>_<房间>", "meterStatus": "<状态>"
  },
  "usage": {
    "yearMonth": "<YYYY-MM>", "monthUse": "<当月累计>", "dataUntil": "<YYYY-MM-DD>",
    "days": [{"date": "<YYYY-MM-DD>", "use": "<当日用量>"}]
  }
}
```

- **`collectedAt` 是数据采集时刻，`dataUntil` 是数据截止日**，都不是实时值。电表约有 1 天延迟，回答时必须写明「数据截至 <日期或时刻>」。
- `days[]` 按日期**升序**。
- `balance` 单位是 `度`。
- `dailyAvg` / `estimatedDays` 为 `null` 表示数据不足，**不要编造可用天数**。
- 两个主请求任一失败会整体报错（退出码按错误码可为 `1` / `3` / `4`），此时**不要用上一次的数字冒充本次结果**。

## 4. 失败怎么办

| 场景 | `error.code` | 应对 |
|---|---|---|
| 快照缺失 / 版本不匹配（且刷新也失败） | `3003` | 让用户执行 `kemaolife --json timetable sync` |
| 刷新失败但有旧快照 | 无（退出码 `0`） | 看 `stale:true`，向用户说明用的是 `syncedAt` 的数据 |
| 会话失效 | `3002` | 刷新内部会重登一次；仍失败且有旧快照时按 `stale:true` 返回，无快照才报此错 |
| 凭据缺失 | `3001` | 让用户执行 `kemaolife config init`，或 `config set customercode <学校代码>` |
| 网络不可达 | `4001` | 课表：有旧快照时不报错，会以 `stale:true` 返回；无快照才报此错。电费：明确报错，不要用缓存 |
| 响应格式异常 | `4002` | 上游可能改版，如实告知，不要自己修正协议 |
| 账号密码错误 | `4003` | 让用户检查凭据 |
| 不在本学期范围 | 无 | 成功信封 + `inSemester:false`；说明是假期 / 越界周次，不要报成错误 |
| 未绑定房间 | `5001` | 说明需先在 H5 页面绑定房间，本工具不做绑定 |

**不要把「刷新失败」说成「查到最新课表」**：`stale:true` 就是沿用了旧快照，回答里必须带数据时间。

## 5. 隐私约束（硬性）

- 不回显、不复述密码；不把学号、姓名、宿舍号、`roomName`、`roomVerify`、Cookie、会话值拼进答复或日志。学校代码（`3126`）与协议 IP 属学校事实，不算个人隐私，但也不必主动复述。
- `config show` 的学号与密码恒为 `****`，不要试图还原。
- 不代查他人、不做批量查询、不把结果发布成服务。
- 用户把凭据贴在对话里时，提示他用 `kemaolife config init` 自行录入，不要把它写进命令参数（会进 shell 历史）。

## 6. 首次使用引导

1. `kemaolife config init` —— 交互录入学号与教务密码（权限 0600）。
2. `kemaolife --json config show` —— 确认学校代码内置默认值（`3126`）已生效；需要时 `config set customercode <学校代码>` 覆盖。
3. `kemaolife --json timetable sync` —— 先生成一次快照（含可选的会话探活 + 1 登录 + 2 数据请求）。之后每次查询会自动刷新，这一步只在首次 / 快照被清时需要。
4. `kemaolife --json timetable` / `kemaolife --json electricity` —— 日常查询。

> // TODO(D2): 待定 —— skill 存放位置仍是待确认项；当前按 `skills/kemaolife/SKILL.md`（仓库内）放置，安装说明见仓库 README「安装与构建」。

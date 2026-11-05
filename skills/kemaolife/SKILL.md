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
| 今天有什么课 / 现在上什么 | `kemaolife --json timetable` |
| 第 N 周课表 / 本周课表 | `kemaolife --json timetable week [N]` |
| 某天有没有课 | `kemaolife --json timetable date <YYYY-MM-DD>` |
| 电费还剩多少 / 还能撑几天 | `kemaolife --json electricity` |
| 某月用了多少电 | `kemaolife --json electricity --month <YYYY-MM>` |
| 版本 / 帮助 | `kemaolife --json --version` / `kemaolife help <命令>` |

不适用：校园网状态、成绩、选课、考试安排、充值、替别人查。

## 2. 固定用法

- **所有命令带 `--json`**，只解析 `stdout`。`stderr` 只有诊断文本，不是数据。
- 退出码表达结果类别，先看退出码再看 `error.code`：

| 退出码 | 含义 | 怎么办 |
|---|---|---|
| `0` | 成功 | 读 `data` |
| `1` | 业务失败 | 读 `error.message`，向用户说明 |
| `2` | 参数错误 | 修正命令后重试，**不要原样重跑** |
| `3` | 凭据缺失 / 会话失效 / 快照缺失 | 引导用户配置或同步 |
| `4` | 网络或接口不可用 | 课表可改查缓存结果（说明 `syncedAt`）；电费明确报错 |

信封：`{"ok":true,"data":{...},"ts":"...","syncedAt":"..."}` 或 `{"ok":false,"error":{"code":3002,"message":"..."},"ts":"..."}`。

## 3. 输出怎么读

### 课表（`DaySchedule`）

```json
{
  "date": "2026-11-05", "week": 10, "weekday": 4, "weekdayCn": "四",
  "inSemester": true,
  "entries": [
    {"name": "课程名", "periods": "1-2", "time": "08:30~10:00",
     "teacher": "教师", "location": "地点", "weeksText": "1-16周"}
  ]
}
```

- `entries: []` 表示**当天没课**，不是查询失败。
- `inSemester: false` 表示该日期不在本学期范围（假期 / 越界周次），此时 `entries` 为空；这是**成功**（退出码 `0`），不是错误。
- `week` 从 1 起；`weekday` 周一 = 1、周日 = 7。
- `periods` 是大节区间，`time` 是该大节的时间段。多门课按起始节次、再按课程名排序。
- `timetable week [N]` 返回 `{"week":N,"days":[7 天]}`，省略 `N` 为当前周；当前日期不在学期内且省略 `N` 时返回 `1002`（`error.message` 会说明学期范围），此时请让用户明确指定周次。
- `periods` 是**小节**节次（1..12），`time` 已按大节换算好；一节大课约 90 分钟，教务共 6 个大节。
- `location` 可能有多个（`;` 分隔），表示该课程不同周次在不同地点。

### 电费（`ElectricityReport`）

```json
{
  "status": {
    "balance": 35.99, "unit": "度", "collectedAt": "2026-11-04T22:41:37+08:00",
    "dailyAvg": 11.7, "estimatedDays": 3,
    "roomName": "校区_楼栋_楼层_房间", "meterStatus": "1"
  },
  "usage": {
    "yearMonth": "2026-11", "monthUse": 63.13, "dataUntil": "2026-11-04",
    "days": [{"date": "2026-11-03", "use": 11.64}, {"date": "2026-11-04", "use": 11.75}]
  }
}
```

- **`collectedAt` / `dataUntil` 是数据采集时间，不是实时值**。电表约有 1 天延迟，回答时必须写明「数据截至 <日期>」。
- `days[]` 按日期**升序**。
- `balance` 单位是 `度`。
- `dailyAvg` / `estimatedDays` 为 `null` 表示数据不足，**不要编造可用天数**。
- 两个请求任一失败会整体报错（退出码 `4`），此时**不要用上一次的数字冒充本次结果**。

## 4. 失败怎么办

| 场景 | `error.code` | 应对 |
|---|---|---|
| 快照缺失 / 版本不匹配 | `3003` | 让用户执行 `kemaolife timetable sync`（查询路径不会自动联网） |
| 会话失效 | `3002` | 让用户执行 `kemaolife timetable sync`（sync 内部会重登一次） |
| 凭据缺失 | `3001` | 让用户执行 `kemaolife config init` 或 `config set customercode` |
| 网络不可达 | `4001` | 课表：**已有快照时查询本不需要网络**，说明同步失败并给出 `syncedAt`；电费：明确报错，不要用缓存 |
| 响应格式异常 | `4002` | 上游可能改版，如实告知，不要自己修正协议 |
| 账号密码错误 | `4003` | 让用户检查凭据 |
| 不在本学期范围（且未指定周次） | `1002` | 让用户明确指定周次，或说明当前是假期 |
| 未绑定房间 | `5001` | 说明需先在 H5 页面绑定房间，本工具不做绑定 |

**不要把「同步失败」说成「查到最新课表」**：`stale` 字段为 `true` 时才表示沿用了旧快照。

## 5. 隐私约束（硬性）

- 不回显、不复述密码；不把学号、学校代码、`roomName`、`roomVerify` 拼进答复。
- `config show` 的学号与密码恒为 `****`，不要试图还原。
- 不代查他人、不做批量查询、不把结果发布成服务。
- 用户把凭据贴在对话里时，提示他用 `kemaolife config init` 自行录入，不要把它写进命令参数（会进 shell 历史）。

## 6. 首次使用引导

1. `kemaolife config init` —— 交互录入学号与教务密码（权限 0600）。
2. `kemaolife config set customercode <学校代码>` —— 电费需要；未设置时电费报 `3001`。
3. `kemaolife timetable sync` —— 生成课表快照（1 登录 + 2 请求）。之后查课表都是 0 请求。
4. `kemaolife --json timetable` / `kemaolife --json electricity` —— 日常查询。

> skill 存放位置仍是待确认项 `D2`；当前按 `skills/kemaolife/SKILL.md`（仓库内）放置，安装说明见仓库 README。

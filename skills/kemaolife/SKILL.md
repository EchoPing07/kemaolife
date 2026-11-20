---
name: kemaolife
description: 查询广东科贸职业学院课表（今日/某日/周课表）与宿舍电费（余额/每日用量/月统计）。
---

# kemaolife CLI 指南

供 agent 调用的单二进制工具。支持**课表查询**与**电费查询**（无校园网功能）。
**核心规范**：所有命令固定带 `--json`，只读取 `stdout`（退出码表达成败）。

## 1. 命令速查与映射

| 用户意图 | 执行命令 | 关键说明 |
|---|---|---|
| 今日课表 / 现在上什么 | `kemaolife --json timetable` | 默认 today；`entries: []` 表示无课 |
| 某日课表 | `kemaolife --json timetable date <YYYY-MM-DD>` | `inSemester: false` 为假期/非学期 |
| 指定周 / 本周课表 | `kemaolife --json timetable week [N]` | 省略 `N` 为当周；返回 `days` 7 天列表 |
| 电费余额 / 还能用多久 | `kemaolife --json electricity` | 当前余额与近 7 日每日用量 |
| 某月用电统计 | `kemaolife --json electricity --month <YYYY-MM>` | 指定月份累计用量与逐日明细 |
| 首次使用 / 重新录入 | `kemaolife config init` | 交互式录入学号密码（0600 权限） |
| 配置学校代码 | `kemaolife config set customercode <代码>` | 覆盖内置默认值（默认 3126） |
| 检查配置 | `kemaolife --json config show` | 确认凭据与学校代码状态（学号密码恒脱敏） |
| 同步快照 / 修复损坏 | `kemaolife --json timetable sync` | 向教务全量重拉并生成快照 |

## 2. 字段契约与业务语义

### 课表（`timetable`）
- **数据结构**：
  - 单日（`timetable` / `today` / `date`）：`DaySchedule { date, week, weekday (1-7), weekdayCn, inSemester, entries: [{ name, periods, time, teacher, location, weeksText }] }`
  - 周课表（`timetable week [N]`）：`{ week, days: [DaySchedule], lastWeek }`
- **刷新机制**：每次查询默认先联网刷新。若返回 `stale: true`，说明联网失败但回退至旧快照，**回复必须明确注明「数据来源于 <syncedAt> 缓存」**。
- **正常状态**：`entries: []` 为当天无课；`inSemester: false` 为假期或非学期，退出码均为 `0`，非错误。
- **节次说明**：`periods` 为小节次（1-12），`time` 为对应大节时间；多门课按起始节次升序。

### 电费（`electricity`）
- **数据结构**：`ElectricityReport { status: { balance, unit, dailyAvg, estimatedDays, collectedAt, roomName }, usage: { yearMonth, monthUse, dataUntil, days: [{ date, use }] } }`
- **延迟标注**：电表数据非实时（约 1 天延迟），**回复必须明确注明「数据截至 <collectedAt 或 dataUntil>」**。
- **严禁臆测**：`dailyAvg` 或 `estimatedDays` 为 `null` 时表示数据不足，严禁虚构可用天数；主请求任一失败即报错，严禁复用旧值冒充本次结果。
- **单位与排序**：`balance` 单位为度；`days[]` 按日期升序。

## 3. 错误决策与恢复矩阵

| 退出码 | `error.code` | 场景分类 | Agent 应对动作 |
|---|---|---|---|
| `0` | - | 成功 | 解析 `data`；若含 `stale: true` 须附带缓存时间说明。 |
| `1` | `1001` | 业务失败 | 上游业务错误，读取 `error.message` 如实向用户说明。 |
| `1` | `5001` | 未绑定房间 | 提示用户需先在学校智慧宿管 H5 页面绑定房间。 |
| `2` | `2001` | 参数错误 | 检查日期/周次格式修正后重试（禁止原样重试）。 |
| `3` | `3001` | 凭据缺失 | 引导用户执行 `kemaolife config init` 录入凭据。 |
| `3` | `3002` | 会话过期且无快照 | 提示用户重新登录或执行 `kemaolife --json timetable sync`。 |
| `3` | `3003` | 快照缺失或损坏 | 引导用户执行 `kemaolife --json timetable sync`。 |
| `4` | `4001` | 网络不可达 | 课表通常已回落旧快照；电费如实告知网络失败，不伪造数据。 |
| `4` | `4002` | 上游接口异常 | 如实告知上游可能改版，不要推测或建议修改协议。 |
| `4` | `4003` | 账号密码错误 | 提示密码错误，引导重新执行 `kemaolife config init`。 |

## 4. 隐私安全与回复风格（硬性）

1. **隐私禁入回复与日志**：严禁在回复、思考或日志中输出学号、密码、宿舍房间全名（`roomName`）、Cookie、Token。若用户在对话中发送密码，提示其通过 `kemaolife config init` 交互录入，严禁拼入命令行参数。
2. **高效回复风格（零冗余）**：
   - 禁止未经请求的客套、寒暄与结束语（如「好的，为您查询到...」、「祝您生活愉快」）。
   - 直接输出核心结果：
     - **课表**：列出上课时间、课程名、教室、节次；若无课直接说明「今日无课」。
     - **电费**：列出剩余度数与数据截至时间；若有预估天数一并列出，否则仅报余额。

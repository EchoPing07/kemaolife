# 接口协议文档

本目录是**协议事实的唯一来源**：实现前必读对应模块，禁止凭记忆写接口。示例代码一律 Go；`curl` 与 JSON 报文块仅作协议证据。所有示例值为占位符，真实凭据 / 会话值 / 响应正文不入仓（见 [隐私与合规](../docs/privacy.md)）。

| 模块 | 内容 | 状态 |
|---|---|---|
| [001-timetable](001-timetable/README.md) | 教务课表：CAS 登录（aic / jw 双主机）、周锚点 + 课表 JSON、快照与本地计算、会话失效的两种形状 | 已实测验证 |
| [002-electricity](002-electricity/README.md) | 宿舍电费：`SWAEServlet` 表单 POST、双层 JSON、余额 / 每日用量 / 采集时刻、字段字典与错误码 | 已实测验证 |
| [003-campus-auth](003-campus-auth/README.md) | 校园网 Portal：探测 302 判态、登录 / 下线表单、自签证书与生效复探 | 已实测验证（Portal 方式） |
| [004-pickup-deeplink](004-pickup-deeplink/README.md) | 取件码深链：淘宝 / 拼多多 `ACTION_VIEW` 跳转与多级回退（非查询协议，纯呈现层） | 整理自公开实现，未逐条实机验证 |

协议与实测不一致时，开 Issue 说明具体条目，不要自行纠偏。对外契约（DTO、错误码、门面 `op`）见 [docs/contract.md](../docs/contract.md)。
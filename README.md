<div align="center">

# 科贸生活 · KeMao Life

**上课前不用翻教务网页，电费一次看全，校园网掉线随手重连 —— 一份 Go 核心，三个原生端。**

[![QQ 交流群 1126474633](https://img.shields.io/badge/QQ%20交流群-1126474633-12B7F5?logo=qq&logoColor=white)](https://qm.qq.com/q/rf4PfFM5RA)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black)](#三端)
[![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white)](#三端)
[![Windows](https://img.shields.io/badge/Windows-0078D6?logo=windows&logoColor=white)](#三端)
[![Android](https://img.shields.io/badge/Android-3DDC84?logo=android&logoColor=white)](#三端)

</div>

围绕**课表、宿舍电费、校园网认证**三件事，按平台特性裁成三个形态，共用一份 Go 核心。三端均已落地并在真机验证（macOS CLI 除外，见下）。这是给自己用的工具，**不是学校官方产品**，只查本人数据。

> [!WARNING]
> macOS 预编译包未经 Apple 签名 / 公证，且 macOS CLI 未经过实机测试，按试验性支持使用。完整说明见 [CLI 手册 §6](./docs/cli.md#6-macos-注意事项试验性支持)。

## 三端

**CLI 被问、Windows 主动说、Android 随手看。** 差异集中在触发器、呈现与平台能力，业务逻辑一份。

| 端 | 形态 | 给谁用 | 能做什么 |
|---|---|---|---|
| **CLI**（linux / darwin） | 单二进制 `kemaolife` + 子命令 + Agent Skill | agent | 课表四维度查询（每次先刷新，断网回落旧快照）、电费余额与每日用量一次查 |
| **Windows** | `kemaolife-desktop.exe` 原生托盘常驻 | 人 | 电费轮询（默认 4 小时）+ 阈值提醒、校园网判态与手动 / 自动认证、开机自启一次 |
| **Android** | Kotlin / Compose 原生 App，三 tab | 人 | 单页周视图 + 横向翻页、WakeUp CSV 导出、电费与校园网手动动作、取件码跳转；缓存优先，无后台 |

三端一律原生 UI，不引入浏览器内核。Windows 不做课表主界面，Android 不加后台服务，CLI 不做 GUI。CLI 与桌面端不共享配置与凭据存储。

## 快速开始

从 [Releases](https://github.com/EchoPing07/kemaolife/releases) 下载对应产物（五类 12 个包，清单见 [构建与发布](./docs/release.md)）：

- **CLI**：解压 tar.gz 后 `install -m 0755 kemaolife ~/.local/bin/kemaolife`，然后 `kemaolife config init` → `kemaolife --json timetable`。详见 [CLI 手册](./docs/cli.md)。
- **Windows**：解压 zip 后双击 `install.bat`，无需管理员权限。详见 [Windows 托盘](./docs/windows.md)。
- **Android**：下载对应架构的 APK 直接安装（多数手机用 `arm64-v8a`），新版覆盖安装即可保留凭据。详见 [Android App](./docs/android.md)。

## 它怎么工作

```
教务系统 ──CAS 登录──► 快照 ──本地计算──► 课表（CLI 每次查询先刷新；Android 查询纯读缓存）
水电后端 ──余额 + 每日用量──► 可用天数估算 ──►（Windows）阈值提醒
Portal   ──探测 302──► 判态 ──► 登录 / 下线 / 重登（Windows 手动 / 自动；Android 全手动）
```

数据怎么取、怎么解析、周次怎么算、告警怎么判，只写一次（`internal/core`）；平台能力经端口注入，UI 只消费 DTO。几个关键的协议坑：教务 CAS 会话失效仍返回 200，必须判响应内容；电费是双层 JSON，外层成功不代表成功；教务服务器没有 IPv6 监听，HTTP 适配器强制 `tcp4`；Portal 禁跟随重定向。协议事实以 [`api-docs/`](./api-docs/) 为准。

## 文档

| 文档 | 内容 |
|---|---|
| [CLI 手册](./docs/cli.md) | 安装、配置、命令、输出解读、macOS 注意事项 |
| [Windows 托盘](./docs/windows.md) | 安装、托盘行为、轮询与告警规则、设置 |
| [Android App](./docs/android.md) | 安装、缓存优先与触发策略、三 tab 功能 |
| [对外契约参考](./docs/contract.md) | 命名、CLI 命令与退出码、JSON 信封、门面 `op`、DTO、错误码 |
| [架构设计](./docs/architecture.md) | 分层、端口、用例、gomobile 复用、依赖约束 |
| [构建与发布](./docs/release.md) | 版本号、产物清单、签名、CI |
| [隐私与合规](./docs/privacy.md) | 只查本人、占位符、凭据存储、脱敏 |
| [接口协议](./api-docs/) | 课表 CAS / 电费 SWAEServlet / 校园网 Portal / 取件码深链，示例一律 Go |
| [参与贡献](./CONTRIBUTING.md) | Issue / PR 规范、提交信息格式、自检清单 |

## 参与贡献

欢迎 Issue 和 PR：三端改进、文档契约审阅、协议证据补充。几条最容易踩的：

- **一个 PR 只做一件事**，标题格式 `类型(范围): 说明`。
- **改契约先改文档**：命令、退出码、信封字段、DTO、错误码、门面 `op` 任一变动，先改 [contract.md](./docs/contract.md) 再动代码，三端同步。
- **测试不碰真实学校接口**，全部 `httptest` + 本地夹具；个人隐私一律占位符。

完整规范见 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 交流

**QQ 交流群：1126474633** · [点此加群](https://qm.qq.com/q/rf4PfFM5RA)。聊想法、问用法都行，但群内讨论不代替 Issue 和 PR；请勿在群里发真实学号、密码、Cookie、课表快照或其他个人信息。

## 安全与隐私

只查本人数据，不做批量、代查与对外服务化；个人隐私不进仓库，凭据只进平台加密存储（CLI `0600` / Windows DPAPI / Android Keystore）。完整规则见 [隐私与合规](./docs/privacy.md)。

> [!WARNING]
> 发现敏感信息被误提交时，**不要公开引用原文**，按 [SECURITY.md](./SECURITY.md) 走私有渠道报告。学校系统自身的漏洞不属于本项目范围，请直接联系学校信息化部门。

## 许可证

[MIT](./LICENSE)。提交贡献即视为同意按 MIT 授权该贡献。
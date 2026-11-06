<div align="center">

# 科贸生活 · KeMao Life

**上课前不用翻教务网页，电费一次看全，校园网掉线随手重连 —— 一份 Go 核心，覆盖 Linux CLI、Windows 托盘与 Android App。三端均已落地。**

[![QQ 交流群 1126474633](https://img.shields.io/badge/QQ%20交流群-1126474633-12B7F5?logo=qq&logoColor=white)](https://qm.qq.com/q/rf4PfFM5RA)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black)](#三端)
[![Windows](https://img.shields.io/badge/Windows-0078D6?logo=windows&logoColor=white)](#三端)
[![Android](https://img.shields.io/badge/Android-3DDC84?logo=android&logoColor=white)](#三端)

[需求与定位](./docs/000-需求与产品定位.md) · [架构](./docs/001-架构设计.md) · [功能规格](./docs/002-功能规格.md) · [接口协议](./api-docs/) · [参与贡献](./CONTRIBUTING.md) · [MIT](./LICENSE)

</div>

<br />

> [!NOTE]
> 仓库当前已实现 Linux CLI（M0 / M1）、Windows 托盘（M2）与 Android App（M0.5 / M3，已在 Android 15 真机跑通）；M4 仅剩 D8 / D11 结论复核。想先搞清楚「要做什么」，从[需求与产品定位](./docs/000-需求与产品定位.md)读起；想直接看命令、字段和退出码，去[功能规格](./docs/002-功能规格.md)。

## KeMao Life 是什么

每天上课前翻教务网页、想知道宿舍还剩多少电、校园网掉线后又得开浏览器重新认证——三件事各有各的入口，在手机上都不好用。kemaolife 把它们收进三个原生端，底层共用一份 Go `core`：数据怎么取、怎么解析、周次怎么算、告警怎么判，只写一次。

- **课表查得慢？** 每次查询先联网刷新一次，刷新成功直接给最新数据；断网时回落旧快照并标 `stale:true` + `syncedAt`，飞行模式也能查整学期。Android 每次打开同样先刷新、失败沿用旧快照。
- **电费只看余额不够** 一条命令 / 一个 `op` 同时返回余额与每日用量，用来估算还能撑几天。电表数据约 1 天延迟，CLI 输出、告警通知与 Android 电费页都标注「数据截至」（Windows 托盘只显示当前值）。
- **校园网** Portal 探测 302 判出「已在线 / 未认证 / 不在校园网」，Windows 默认手动、自动可选，Android 全手动，Linux 不碰校园网也不存凭据。
- **三端不漂移** `internal/core` 只做领域逻辑，平台能力经端口注入；三端共享 DTO、错误码与告警判定，UI 只消费数据。单仓库、单 `go.mod`、多 `cmd`。

这是给自己用的工具，**不是学校官方产品**，与教务系统、水电系统、校园网 Portal 的运营方没有关联，也只查本人数据。

## 三端

| 端 | 长什么样 | 给谁用 | 能做什么 |
|---|---|---|---|
| **Linux** | 单二进制 `kemaolife` + 子命令 + Agent Skill | agent | 配置、课表四维度查询（默认当日）、电费余额与每日用量一次查 |
| **Windows** | `kemaolife-agent.exe` 原生托盘常驻（登录时自启一次，不保活） | 人 | 托盘看状态、电费轮询（默认 4 小时）、校园网判态轮询（默认 120 秒）、阈值提醒、托盘 `NIF_INFO` 通知、校园网默认手动 / 自动可选 |
| **Android** | 原生 App（Kotlin / Compose），底栏三 tab | 人 | 课表、电费与校园网手动动作、设置；每次打开并行发起查询与独立同步 |

> [!IMPORTANT]
> Windows 与 Android 都是原生 UI，三端一律不引入浏览器内核（WebView / CEF / Wails / PWA / Electron / Tauri）。Windows 不做课表主界面，Android 不加后台服务，Linux 不要 TUI / GUI 富文本输出。

## 它怎么工作

```
教务系统 ──CAS 登录──► 快照 ──本地计算──► 课表（每次查询先刷新，失败沿用旧快照）
                              └─ Android 每次打开独立同步一次（不阻塞展示）
水电后端 ──余额 + 每日用量──► 可用天数估算 ──►（Windows）阈值提醒（托盘 NIF_INFO 通知）
Portal   ──探测 302──► 判态 ──► 登录 / 下线 / 重登（Windows M2 已实现）
                              ├─ Windows 自动模式：作为电费轮询前置
                              └─ Windows 手动 / Android：按钮手动触发
```

几个不显眼但很关键的约定：教务 CAS 会话失效仍返回 200，必须判响应内容；电费 `body` 是字符串，缺参数时外层也可能「成功」，得解两层再判；教务服务器没有 IPv6 监听，HTTP 适配器强制 `tcp4`；Portal 禁跟随重定向，自签证书跳过校验只作用于 Portal HTTPS 客户端（探测地址是 HTTP 明文）。协议事实以 [`api-docs/`](./api-docs/) 为准，别凭记忆写接口。

## 现在到哪一步了

| 阶段 | 产出 | 状态 |
|---|---|---|
| 设计基线 | R1–R5、三端定位、架构与端口、共享契约、三个接口协议文档 | 已入库 |
| M0 · 契约 + core | DTO、错误码、`core` / `ports`、单测、`kemaolife` 跑通 | 已实现 |
| M1 · Linux CLI + skill | 单二进制 `kemaolife` + 子命令、`config`、快照缓存、SKILL.md、刷新降级、电费合并查询 | 已实现（aarch64 Linux 真机验证） |
| M2 · Windows 托盘 | 托盘状态与按模式切换的按钮组、4 小时电费轮询（600s 硬下限）、120 秒判态轮询（30s 下限）、阈值去重与静默时段、失败退避、托盘通知、DPAPI 凭据、原生设置窗口、开机自启 | 已实现（本机 Win11 验证） |
| M3 · Android App | `gomobile` 门面、三 tab、打开即查询 + 并行同步、电费页、校园网手动动作、Keystore 设置页 | 已实现（Android 15 真机验证） |
| M4 · 校园网自动登录复核 | Windows 自动模式已随 M2 实现，Android 手动适配器与平台网络配置已随 M3 落地；M4 仅按 D8 / D11 结论复核 Windows 链路 | 待 D8 / D11 定稿 |

完整里程碑与验收见 [000 文档 §7](./docs/000-需求与产品定位.md#7-里程碑)。

> [!TIP]
> **想提意见先看这里。** `docs/000` §6 的 D 项是待确认清单，「倾向」只是把契约写完整的描述性基线，不等于定稿；D 项由仓库所有者拍板，实现遇阻时取最保守默认值并标 `TODO(D<n>)`。

## 文档

| | |
|---|---|
| [需求与产品定位](./docs/000-需求与产品定位.md) | **先看这个**：R1–R5、功能矩阵、硬性约束、D 项清单、里程碑 |
| [架构设计](./docs/001-架构设计.md) | 分层、端口、用例层、Android JSON 门面、依赖约束 |
| [功能规格](./docs/002-功能规格.md) | 命名基线、CLI 与退出码、JSON 信封、门面 `op`、DTO、错误码、告警规则、验收清单 |
| [接口协议](./api-docs/) | 课表 CAS / 电费 `SWAEServlet` / 校园网 Portal，示例一律 Go |
| [贡献指南](./CONTRIBUTING.md) | Issue / PR 规范、提交信息格式、契约变更纪律、自检清单 |
| [安全与隐私](./SECURITY.md) · [行为准则](./CODE_OF_CONDUCT.md) | 敏感信息报告渠道、社区约定 |

需求与契约以 `docs/` 为准，协议事实只以 `api-docs/` 为准；`curl` 与 JSON 报文块仅作协议证据。三端详细功能与不做项见 [002 §5](./docs/002-功能规格.md#5-三端详细功能)。

## 参与贡献

欢迎 Issue 和 PR。M0/M1/M2/M3 已实现，当前可参与 Linux CLI、Windows 托盘、Android App 改进、文档契约审阅与 D 项讨论。完整规范在 [CONTRIBUTING.md](./CONTRIBUTING.md)，几条最容易踩的：

- **一个 PR 只做一件事**，从最新 `main` 拉分支，别夹带格式化或顺手优化。
- **标题写实际改动**，格式 `类型(范围): 说明`，例如 `feat(core): 快照版本升级到 2`；正文写清改了什么、为什么、怎么验证。
- **改契约先改文档**：CLI 命令与退出码、JSON 信封字段、DTO 字段名、错误码、门面 `op` 任一变动，先改 `docs/002-功能规格.md` 再动代码，三端字段名同步，新增字段必须可选。
- **Issue 用模板**：`[bug]` / `[feat]` / `[docs]` / `[protocol]` / `[question]`，按「背景 / 要求 / 验收」写，别贴日志原文或响应体。
- **提交前自检**：`gofmt -l .`、`go vet ./...`、`go test ./...`、分层与浏览器内核检查、Linux / Windows / Android 交叉编译、托盘 GUI 子系统、打包脚本 BOM 与 CRLF 都要过（权威清单见 `.github/workflows/ci.yml`）；测试只用 `httptest` + 本地夹具，不碰真实学校接口。

## 安装与构建（Linux）

```bash
go build -o kemaolife ./cmd/kemaolife
kemaolife config init                         # 交互录入学号与教务密码（0600）
kemaolife --json timetable                    # 课表（每次查询先刷新）
kemaolife --json electricity                  # 余额 + 每日用量
```

学校代码内置默认值为 `3126`（学校事实，非个人隐私），可用 `kemaolife config set customercode <学校代码>` 或构建期 `-ldflags "-X main.defaultCustomerCode=<学校代码>"` 覆盖。agent skill 见 [`skills/kemaolife/SKILL.md`](./skills/kemaolife/SKILL.md)，复制到 agent 的 skill 目录即可使用（D2 待定稿）。

正式发布在 [Releases](https://github.com/EchoPing07/kemaolife/releases) 提供 Linux `amd64` / `arm64` / `armv7` 静态二进制压缩包、Windows `amd64` / `arm64` 便携包、Android 按 ABI 拆分的四个已签名 APK、单独的 agent skill zip 与 `SHA256SUMS`；Linux 解压后用 `install -m 0755 kemaolife ~/.local/bin/kemaolife` 安装，Windows 解压后双击 `install.bat`，Android 下载对应架构的 APK 直接安装，校验和用 `sha256sum -c SHA256SUMS --ignore-missing` 核对。打 `vX.Y.Z` tag 会触发 [`.github/workflows/release.yml`](./.github/workflows/release.yml) 自动构建发布，tag 必须与 `internal/version.Version` 一致；Android `versionCode` 由 tag 派生（`major*10000 + minor*100 + patch`）。

## 安装（Windows）

下载 `kemaolife-agent_<版本>_windows_<架构>.zip`（`amd64` 或 `arm64`），**解压到任意目录后双击 `install.bat`**。脚本会结束旧实例、复制程序到 `%LOCALAPPDATA%\Programs\Kemaolife\`、建开始菜单快捷方式并启动一次，不需要管理员权限。也可直接双击 `kemaolife-agent.exe` 绿色运行，只是此时建议不要开启开机自启（自启项记录的是 exe 绝对路径，程序一挪位置就失效）。

卸载双击 `uninstall.bat`：结束进程、删自启项、删快捷方式与安装目录，**保留** `%LOCALAPPDATA%\Kemaolife\` 配置目录（含 DPAPI 加密的学号与凭据，重装可沿用）。

> [!NOTE]
> **本版没有代码签名**。Windows 会提示「Windows 已保护你的电脑」（点「更多信息 → 仍要运行」）；开启了 Smart App Control 的 Windows 11 设备会直接拦截且不提供放行按钮，只能到「Windows 安全中心」关闭 SAC（**关闭后无法重新开启**）。免费开源签名的跟进项见 [#4](https://github.com/EchoPing07/kemaolife/issues/4)。

> [!IMPORTANT]
> **`arm64` 版本尚未在 ARM 真机上验证**。构建可以通过，但托盘图标、右键菜单、设置窗口与凭据加解密都依赖手写的 Win32 结构体布局，未经实机确认；遇到问题请带 Windows 版本与现象提 Issue。不出 32 位 `386`（Windows 11 没有 32 位版）。

首次运行没有学号时会直接弹出设置窗口；之后只在托盘出现图标，左键单击开设置、右键出菜单。关掉设置窗口只是隐藏它，程序启动即查一次电费并判一次校园网状态，之后分别按默认 4 小时与 120 秒轮询；托盘菜单「退出」才是结束进程。「开机自启」默认勾选，但只在设置窗口里点「保存」后才写入注册表，首次安装不会自动生效。电费低于阈值时由托盘 `NIF_INFO` 弹出 Windows 通知，进程启动时设置 `Kemaolife.Agent` AUMID 用于归类，不实现 WinRT Toast / 操作中心归档。

## 构建（Windows）

Windows 端是托盘常驻程序，**正式构建必须按 GUI 子系统构建**：不带上 `-H=windowsgui` 会多挂一个控制台窗口（程序启动时会自己 `FreeConsole` 摘掉，但仍不符合 [002 §5.2](./docs/002-功能规格.md#52-windows-托盘常驻主动说) 的 GUI 子系统形态）：

```bash
GOOS=windows GOARCH=amd64 go build -trimpath `
  -ldflags "-s -w -H=windowsgui" `
  -o kemaolife-agent.exe ./cmd/kemaolife-agent
```

`arm64` 把 `GOARCH` 换成 `arm64` 即可。便携包的安装 / 卸载脚本在 [`packaging/windows/`](./packaging/windows/)，由 release 工作流打进 zip。

## 安装（Android）

从 Release 下载**一个**对应架构的 APK：多数近几年手机用 `arm64-v8a`，旧 32 位机用 `armeabi-v7a`，`x86` / `x86_64` 主要给模拟器用。系统需 Android 7.0（API 24）及以上。直接在手机上打开 APK 安装，首次侧载需要在系统提示里允许该来源的「安装未知应用」；也可 `adb install -r <APK>`。

四个包共用同一把发布私钥与同一 `applicationId`，**新版直接覆盖安装即可**，设置页 Keystore 里的凭据与应用私有目录里的课表快照都会保留；不要先卸载再装。发布签名证书的 SHA-256 指纹写在每版 Release 说明里，可用 `apksigner verify --print-certs <文件>` 核对；指纹不一致时不要安装、不要填任何凭据。本仓库只发侧载 APK，不上架应用商店，也没有自动更新。

## 交流

**QQ 交流群：1126474633** · [点此加群](https://qm.qq.com/q/rf4PfFM5RA)（也可以在 QQ 里搜群号加群）。

聊想法、问用法都行，但**群内讨论不代替 Issue 和 PR**：需求、D 项定稿、契约变更最后都要落到仓库的 Issue 与 `docs/`。请勿在群里发真实学号、密码、Cookie、课表快照或其他个人信息。

## 安全与隐私

- **只查本人数据**。电费接口无鉴权，不做批量遍历、代查他人、多账号，也不把电费 / 教务 / Portal 接口包装成对外服务（HTTP API、公开脚本、SaaS）。
- **个人隐私不进仓库**。真实学号、姓名、宿舍房间号、密码、Cookie、课表快照、会话值、`roomverify`、本机 MAC、`distoken`、指纹值一律不写进代码、注释、测试夹具、文档、Issue、PR 或提交信息；示例统一用 `<学号>`、`<密码>`、`<roomverify>`、`校区_楼栋_楼层_房间` 占位。学校代码（`3126`）、学校公开的 Portal / 探测 IP 与 BRAS 名称属学校与协议事实，允许入仓。
- **凭据只进平台加密存储**：Linux `0600`、Windows DPAPI、Android Keystore；校园网凭据独立字段，不复用教务密码；快照含 Cookie，等同凭据。日志只记错误类别、耗时、状态码。
- 学校代码是可覆盖的配置项：内置默认值 `3126` 写在 `cmd` 层，可用 `config set customercode` 或构建期注入覆盖，不作为 `internal/core` 通用常量发布。
- 电表数据约 1 天延迟，CLI 输出、告警通知与 Android 电费页必须标注「数据截至」，不能当实时值（Windows 托盘按契约只显示当前值）；Windows 电费轮询 ≥ 600 秒为硬下限，默认 4 小时，校园网判态轮询默认 120 秒、≥ 30 秒。

> [!WARNING]
> 发现敏感信息被误提交时，**不要公开引用原文**，按 [SECURITY.md](./SECURITY.md) 走私有渠道报告。学校系统自身的漏洞不属于本项目范围，请直接联系学校信息化部门。

## 许可证

[MIT](./LICENSE)。提交贡献即视为同意按 MIT 授权该贡献。

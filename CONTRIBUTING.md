# 贡献指南

谢谢参与。M0（契约 + core）、M0.5（Android 复用 spike）、M1（CLI + skill，linux / darwin）、M2（Windows 托盘常驻）与 M3（Android App）已实现；M4 仅剩 D8 / D11 结论复核。CLI 在 linux / darwin 提供预编译包（macOS 为 darwin amd64 / arm64，未经 Apple 签名 / 公证、未经过实机测试），也可源码自行构建（可选）；windows CLI 仍为后续计划。贡献以三端改进、文档契约审阅、D 项讨论与 Issue 为主，按本文规范提交代码。

参与前请先阅读 [README.md](README.md) 与 [docs/000-需求与产品定位.md](docs/000-需求与产品定位.md)。需求与契约以 `docs/` 为准，协议事实只以 `api-docs/` 为准；两者冲突时**不要自行纠偏**，开 Issue 说明具体条目。

## 1. 适用范围与隐私前置条件

本项目只服务**一个学校、一个账号**，且只查询账号本人数据。提交任何内容前，请确认：

- 不含真实学号、姓名、宿舍房间号、密码、Cookie、课表快照内容、会话值、本机 MAC、`distoken`、指纹值。
- 文档、注释、测试夹具、Issue、PR、提交信息中的个人隐私一律用占位符：`<学号>`、`<密码>`、`<roomverify>`、`<校区>_<楼栋>_<楼层>_<房间>`。学校代码（`3126`）与学校公开的 Portal / 探测 IP、BRAS 名称属学校与协议事实，允许入仓。
- 不把电费 / 教务 / Portal 接口包装成对外服务，不实现多账号、批量遍历或代查。
- 不复现、不转载学校系统的实际响应数据；测试用 `httptest` 与本地夹具。

误提交敏感信息时不要公开引用原文，按 [SECURITY.md](SECURITY.md) 私下报告。

## 2. Issue 规范

### 2.1 先搜索

开 Issue 前先搜已有 Issue（含已关闭的）与 `docs/000-需求与产品定位.md` §6 待确认清单，避免重复。D 项是待用户决策的问题，**倾向不等于定稿**；讨论 D 项请用「待确认项 / 使用疑问」模板并注明编号。

### 2.2 标题格式

| 类型 | 前缀 | 例子 |
|---|---|---|
| 缺陷 | `[bug]` | `[bug] Windows 轮询间隔低于 600 秒未被钳制` |
| 功能 / 变更 | `[feat]` | `[feat] 课表支持导出为表格` |
| 文档 | `[docs]` | `[docs] 架构文档端口清单缺一项` |
| 协议证据 | `[protocol]` | `[protocol] 电费缺 account 时外层 result_ 判据` |
| 待确认项 / 疑问 | `[question]` | `[question] D13 gomobile 工具链版本` |

标题写清「哪一端 / 哪个模块 + 一句话结论」，不用「求救」「不好使」「有个想法」。

### 2.3 模板与内容

请使用 GitHub 自动提供的模板（见 [.github/ISSUE_TEMPLATE/](.github/ISSUE_TEMPLATE/)）：

| 模板 | 适用 |
|---|---|
| 问题报告 | 可复现的缺陷、回归 |
| 功能建议 | 新能力、行为变更 |
| 待确认项 / 使用疑问 | D 项讨论、范围或工具链疑问（不自行定稿） |
| 文档与契约 | 文档错误、契约不一致、协议与实测冲突 |

内容按「背景 / 要求 / 验收」或模板字段填写，不贴对话记录、不贴日志原文、不贴响应体。复现步骤用本地夹具描述，不要求维护者使用真实账号。

### 2.4 标签

维护者按端与类型打标签：`linux` / `darwin` / `windows` / `android`（CLI 平台标签为 `linux` / `darwin`）、`core` / `app` / `platform`、`docs` / `api-docs` / `protocol`、`bug` / `enhancement` / `question`、`contract`、`D<n>`、`good first issue`。Issue 表单会自动带上 `bug` / `docs` / `enhancement` / `question` 四个默认标签，维护者需在仓库 **Issues → Labels** 中确认它们存在（新建仓库通常已自带）；其余标签由维护者按需创建。贡献者可建议标签，不必自行设置。

### 2.5 不在本项目范围内

- 新增第四个功能域、多学校 / 多账号抽象、代查他人数据。
- 电费充值、绑房间、解绑房间。
- 浏览器内核方案（WebView / CEF / Wails / PWA / Electron / Tauri）。
- Windows 课表主界面、Android 后台服务 / 自启动 / 桌面小组件、统计图表、TUI / GUI 富文本输出。
- 学校系统自身的漏洞与账号问题。

范围依据见 [docs/000-需求与产品定位.md §5](docs/000-需求与产品定位.md#5-硬性设计约束) 与 [docs/002-功能规格.md §5](docs/002-功能规格.md#5-三端详细功能)。

## 3. 分支与 Pull Request 规范

### 3.1 分支

Fork 后从默认分支（`main`）建短生命周期分支，命名 `<类型>/<简述>`，例如 `fix/electricity-poll-floor`、`docs/readme-contributing`、`feat/core-snapshot-v2`。不要把多个无关改动混进一个分支。

### 3.2 PR 标题与描述

标题沿用提交信息格式 `类型(范围): 说明`，例如 `docs(readme): 补齐贡献规范与 MIT 许可`。描述请填写 [PR 模板](.github/pull_request_template.md)，至少说明：

- 背景 / 目的，以及关联的 Issue（`Closes #123`）或 R / D 编号。
- 改动范围与不做的事。
- 是否触及对外契约（CLI 子命令与参数、退出码、JSON 信封字段、DTO 字段名、错误码、Android 门面 `op`）。
- 验证方式与自检结果。

一个 PR 只做一件事；纯格式化与「顺手优化」单独开，不与功能改动混在一起。

### 3.3 提交信息

采用 Conventional Commits 风格：

```
类型(范围): 说明
```

| 类型 | 用途 |
|---|---|
| `feat` | 新能力 |
| `fix` | 缺陷修复 |
| `docs` | 文档（含 README、CONTRIBUTING） |
| `refactor` | 不改变对外行为的重构 |
| `test` | 测试 |
| `build` | 构建与工具链 |
| `chore` | 杂项维护 |

范围用 `core` / `app` / `platform` / `cli` / `windows` / `android` / `api` / `docs` / `readme` 等。示例：`feat(core): 快照版本升级到 2`、`docs(api): 电费示例改为 Go`。

提交信息同样受 §1 隐私约束：不得写入真实凭据、账号或内网地址。

### 3.4 契约变更纪律

改动下列内容时，**先改 `docs/002-功能规格.md`，再改代码**，并在 PR 描述与提交信息中注明：

- CLI 子命令、参数、退出码（`0/1/2/3/4`）
- JSON 信封字段（`ok` / `data` / `error` / `ts` / `stale` / `syncedAt`）
- DTO 字段名（`DaySchedule`、`ElectricityStatus`、`ElectricityUsage`、`ElectricityReport`、`CampusStatus`、`Snapshot` 等）
- 错误码（`1001`–`5001`）
- Android 门面的 `op` 取值（含 `campusnet.*`）

规则：

1. 禁止只有一端改字段名；要么三端同步改，要么不改。
2. 新增字段必须可选，零值有明确语义，不让老快照 / 老客户端解析失败。
3. 快照结构变更必须升 `Version`，旧版本直接丢弃重拉，不做兼容迁移。
4. 退出码与 `error.code` 必须同域映射，不得出现「退出码 0 但 `ok:false`」。

D 项的定稿只能由用户决策；遇到 D 项阻塞实现时取最保守默认值，代码标 `// TODO(D<n>): 待定`，并在 PR 描述里单独列出。

### 3.5 协议实现规则（M0 起）

- 协议事实只来自 `api-docs/`，不凭记忆写接口。
- 教务 CAS 会话失效也返回 200，必须判响应内容；教务强制 IPv4（`tcp4`），在 `HTTPDoer` 适配器里禁用 v6。
- 电费 `body` 是字符串且缺参数时外层仍可能 `result_:"true"`，必须解 `body` 再判 `body.result`。
- Portal 必须禁跟随重定向，成败只看 body 的 `success`；自签证书跳过校验仅作用于 Portal HTTPS 客户端（探测地址是 HTTP 明文），禁止全局 `InsecureSkipVerify`。
- 测试禁止访问真实学校接口，全部用 `httptest` + 本地夹具；禁止依赖系统时间与真实文件系统路径。

## 4. 代码与分层要求（M0 起）

- 单 Go module；Go 代码 `gofmt` + `go vet` 干净，错误用 `%w` 包装，不用 `panic` 处理业务错误。
- 分层：`cmd/*` 与 `android/*` 只做触发器 / 呈现 / 平台能力 / UI，`internal/app` 做用例编排，`internal/agent` 做 Windows 端轮询 / 告警编排（只依赖端口，可跨平台单测），`internal/core` 是纯领域逻辑，`internal/platform` 只做适配器。`internal/core`、`internal/app` 与 `internal/agent` 中不得出现 `runtime.GOOS`、路径拼接、`os.Getenv`、`syscall`、直接 `time.Now()`、`fmt.Print*`。
- 新增平台差异先加端口（`internal/core/ports.go`），再写适配器；每个新端口实现要有内存 fake。
- 时间统一经 `Clock` 获取，构造 `time.Time` 明确 `time.Local`；列表输出显式排序（课表按起始节次 + 课程名，每日用量按日期升序）。
- 新增第三方依赖前先在 Issue 说明理由并取得确认；标准库能解决的用标准库。
- 文档编号固定，新增文档续 `003` 起；`docs/` 的「变更记录」表只追加、不改写，`docs/` 中不记录对话过程，示例值一律占位符，`api-docs/**` 示例代码必须是 Go。

## 5. 提交前自检（M0 起执行）

```bash
gofmt -l .
go vet ./...
go test ./...

# core 不得依赖平台 / UI 包
go list -deps ./internal/core/... | grep -E 'x/sys/windows|gioui|fyne|android' && echo FAIL

# 三端硬约束：无浏览器内核
grep -rniE 'webview|cef|wails|pwa|electron|tauri' --include='*.go' --include='*.kt' . && echo FAIL

# 分层：core / app / agent 里不该有平台判断
grep -rn 'runtime.GOOS' internal/core internal/app internal/agent && echo FAIL

# 隐私：把占位模板替换成自己已知的真实敏感值（每行一个）后逐个检查；
# 模板自身会出现在 AGENTS.md / CONTRIBUTING.md / ci.yml / release.yml，需排除以免自匹配误报。
# 学校代码 3126 与协议 IP 属学校事实，不在此检查范围
SENSITIVE_VALUES='<把已知的真实学号/房间号写在这里>'
printf '%s\n' "$SENSITIVE_VALUES" | while IFS= read -r value; do
  [ -z "$value" ] && continue
  if grep -rniF -e "$value" . \
      --exclude-dir=.git \
      --exclude=ci.yml --exclude=release.yml --exclude=AGENTS.md --exclude=CONTRIBUTING.md; then
    echo 'FAIL: personal data found'; exit 1
  fi
done

# CLI 源码自行构建 / 交叉编译（预编译包由 Release 提供：linux 三架构 + darwin amd64 / arm64）
GOOS=linux   GOARCH=amd64 go build ./internal/... ./cmd/kemaolife/...
GOOS=darwin  GOARCH=amd64 go build ./internal/... ./cmd/kemaolife/...
GOOS=darwin  GOARCH=arm64 go build ./internal/... ./cmd/kemaolife/...
GOOS=darwin  GOARCH=amd64 go vet ./cmd/kemaolife/... ./internal/platform/...
GOOS=darwin  GOARCH=arm64 go vet ./cmd/kemaolife/... ./internal/platform/...
GOOS=windows GOARCH=amd64 go build ./internal/...
GOOS=windows GOARCH=arm64 go build ./internal/...
# 托盘端必须是 GUI 子系统：console 子系统会挂一个控制台窗口（启动会 FreeConsole 摘掉，但不符合 docs/002 §5.2 形态）
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o kemaolife-agent.exe ./cmd/kemaolife-agent/
GOOS=android GOARCH=arm64 go build ./internal/... ./cmd/kemaolife-mobile/
```

CI 还校验 CLI 的 darwin 两架构（amd64 / arm64）build + vet、托盘 PE 的 GUI 子系统（Subsystem=2）与 `packaging/windows/*.ps1` / `*.bat` 的 UTF-8 BOM、CRLF；发版时核对 macOS 预编译包（darwin amd64 / arm64）打包与 `SHA256SUMS` 的 12 行；完整 Android AAR / APK（NDK + Gradle + gomobile）由 `.github/workflows/ci-android.yml` 在 Android 路径变更时构建并跑单元测试；权威清单见 `.github/workflows/ci.yml` 与 `.github/workflows/ci-android.yml`。

任一项失败先修完再提交，不要用 `//nolint`、注释掉检查或 `--no-verify` 绕过。设计阶段（无 Go 代码）只需执行隐私与占位符相关检查。

## 6. 评审与合并

- PR 需通过自检，且变更与 `docs/` 契约一致；触及契约时优先合并文档变更。
- 维护者可能要求拆分 PR、补充测试或补充变更记录；这属于契约一致性检查，不是额外要求。
- 合并方式默认 squash，合并后的提交信息沿用 §3.3 格式。
- 合并后请确认关联的 D 项状态、`docs/` 变更记录与三端字段名是否同步。

## 7. 行为准则

讨论聚焦问题本身，就事论事；不公开他人的真实数据，不要求他人提供账号或凭据用于排障，不把学校系统的实际响应内容贴进 Issue / PR。完整准则与报告渠道见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。

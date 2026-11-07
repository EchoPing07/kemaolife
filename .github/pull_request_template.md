## 背景 / 目的

<!-- 为什么改；关联 Issue 用 Closes #123，或注明 R 项 / D 项。不记录对话过程，不粘贴真实数据。 -->

## 改动范围

<!-- 涉及哪一端 / 哪一层 / 哪些文件；列出不做的事。 -->

- [ ] 仅文档 / 模板
- [ ] `internal/core`
- [ ] `internal/app`
- [ ] `internal/agent`
- [ ] `internal/platform`
- [ ] `cmd/*` 或 `android/*`
- [ ] `api-docs/**`

## 契约影响

**是否触及对外契约？** 是 / 否

触及项（勾选并说明三端如何同步）：

- [ ] CLI 子命令、参数、退出码（0/1/2/3/4）
- [ ] JSON 信封字段（`ok` / `data` / `error` / `ts` / `stale` / `syncedAt`）
- [ ] DTO 字段名（`DaySchedule`、`ElectricityStatus`、`ElectricityUsage`、`ElectricityReport`、`CampusStatus`、`Snapshot` 等）
- [ ] 错误码（1001–5001）
- [ ] Android 门面 `op` 取值（含 `campusnet.*`）
- [ ] 快照 `Version`（结构变更必须升版本，旧版本丢弃重拉）

若触及：确认已**先改 `docs/002-功能规格.md` 再改代码**，且新增字段可选、零值有明确语义、退出码与 `error.code` 同域映射。

## D 项

<!-- 若实现受待确认项影响：写 D 编号、所取的最保守默认值、代码中 `// TODO(D<n>): 待定` 的位置。D 项定稿由用户决策，本 PR 不自行定稿。 -->

无 / D<n>：

## 验证

自检结果（M0 起适用；设计阶段只需执行隐私与占位符检查）：

- [ ] `gofmt -l .` 无输出
- [ ] `go vet ./...`
- [ ] `go test ./...`（测试用 `httptest` + 本地夹具，未访问真实学校接口）
- [ ] `go list -deps ./internal/core/... | grep -E 'x/sys/windows|gioui|fyne|android'` 无匹配
- [ ] `grep -rniE 'webview|cef|wails|pwa|electron|tauri' --include='*.go' --include='*.kt' .` 无匹配
- [ ] `grep -rn 'runtime.GOOS' internal/core internal/app internal/agent` 无匹配
- [ ] 已知真实敏感值逐个检索无匹配（检查命令需排除模板自身所在的 `AGENTS.md` / `CONTRIBUTING.md` / `ci.yml`）
- [ ] `GOOS=linux GOARCH=amd64 go build ./internal/... ./cmd/kemaolife/...`
- [ ] `GOOS=darwin GOARCH=amd64 go build ./internal/... ./cmd/kemaolife/...`；`GOOS=darwin GOARCH=arm64 go build ./internal/... ./cmd/kemaolife/...`
- [ ] `GOOS=windows GOARCH=amd64 go build ./internal/...`；`GOOS=windows GOARCH=arm64 go build ./internal/...`
- [ ] `GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o kemaolife-agent.exe ./cmd/kemaolife-agent/`（托盘端必须是 GUI 子系统；console 子系统会挂控制台窗口，启动虽会 `FreeConsole` 摘掉，但不符合 docs/002 §5.2 形态）
- [ ] `GOOS=android GOARCH=arm64 go build ./internal/... ./cmd/kemaolife-mobile/`
- [ ] CI（`.github/workflows/ci.yml`）通过：另含 PE GUI 子系统校验、Windows 安装脚本 BOM / CRLF 校验；列表输出显式排序；时间统一经 `Clock` 获取

补充说明（可选）：

## 提交确认

- [ ] 提交信息为 `类型(范围): 说明`，且不含真实凭据 / 账号 / 内网地址
- [ ] 一个 PR 只做一件事，未混入「顺手优化」
- [ ] 未包含真实学号、姓名、宿舍房间号、密码、Cookie、课表快照内容、会话值、本机 MAC、`distoken` 或指纹值；个人隐私示例均为占位符（学校代码 `3126` 与协议 IP 属学校事实，可入仓）
- [ ] `docs/` 的「变更记录」表只追加了新行，未改写或删除已有行
- [ ] 新增第三方依赖或构建工具链已在 Issue 中说明理由并取得确认（无则勾选）
- [ ] 未实现非本人数据查询、批量遍历、代查或对外服务化
- [ ] 我已阅读 [CODE_OF_CONDUCT.md](https://github.com/EchoPing07/kemaolife/blob/main/CODE_OF_CONDUCT.md)，讨论聚焦问题本身

规范详见 [CONTRIBUTING.md](https://github.com/EchoPing07/kemaolife/blob/main/CONTRIBUTING.md)。

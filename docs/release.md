# 构建与发布

本文是版本号、产物清单、签名、CI 与 Release 正文的**唯一来源**。各端安装说明见 [cli.md](cli.md) / [windows.md](windows.md) / [android.md](android.md)，Android 本地构建的工具链与已知坑见 [android/README.md](../android/README.md)。

## 1. 版本号

- 仓库唯一来源：`internal/version.Version`（Go 常量），三端共用，发版只改这一行；Android `versionName` 与之同步。不由构建参数覆盖。
- `commit` 由编译期 `-ldflags "-X main.commit=<提交号>"` 注入。
- 正式发布在变更合并到 `main` 后打 `vX.Y.Z` tag，**tag 与源码常量一致才算发布**。
- Android `versionCode` 由 tag 派生：`major*10000 + minor*100 + patch`，跨版本严格递增，否则 Android 拒绝覆盖安装。

## 2. 发布流程

打 `vX.Y.Z` tag 触发 [`.github/workflows/release.yml`](../.github/workflows/release.yml)：

1. 校验 tag 与 `internal/version.Version` 一致，且提交属于 `main`。
2. 隐私自检（已知真实值逐个检索，排除模板自身）。
3. 按下表构建五类产物，校验签名 / 子系统 / 脚本编码。
4. 生成 `SHA256SUMS` 与 Release 正文（§6），发布到 [Releases](https://github.com/EchoPing07/kemaolife/releases)。

## 3. 产物清单

**OS × 架构支持**（「能构建」与「功能差异」是两件事，功能边界见各端文档）：

| 形态 | OS | 架构 | 交付方式 |
|---|---|---|---|
| CLI | linux | amd64 / arm64 / armv7 | 预编译 tar.gz + 源码构建 |
| CLI | darwin（macOS） | amd64 / arm64 | 预编译 tar.gz + 源码构建（试验性，见 §4） |
| CLI | windows | 后续计划 | 不在本期 |
| 桌面端（原生托盘） | windows | amd64 / arm64 | 便携 zip |
| 移动端（Compose App） | android | armeabi-v7a / arm64-v8a / x86 / x86_64 | 按 ABI 四个已签名侧载 APK |

**五类共 12 个包 + `SHA256SUMS`（12 行）**：

| 类别 | 包 |
|---|---|
| Linux CLI（3） | `kemaolife_<版本>_linux_{amd64,arm64,armv7}.tar.gz`（`CGO_ENABLED=0` 纯 Go 二进制） |
| macOS CLI（2） | `kemaolife_<版本>_darwin_amd64.tar.gz`（Intel Mac）、`kemaolife_<版本>_darwin_arm64.tar.gz`（Apple Silicon） |
| Windows 托盘（2） | `kemaolife-desktop_<版本>_windows_{amd64,arm64}.zip` |
| Android APK（4） | `kemaolife_<版本>_android_<ABI>.apk` |
| agent skill（1） | `kemaolife-skill_<版本>.zip` |

CLI 包内含 `kemaolife` / `LICENSE` / `INSTALL.txt`，Linux 与 macOS 同构；文件名用 `darwin`（仓库 `GOOS` 口径），展示名用 macOS。Windows zip 内含 `kemaolife-desktop.exe`（GUI 子系统）+ `install.bat` / `install.ps1` / `uninstall.bat` / `uninstall.ps1` + `LICENSE` / `INSTALL.txt`，脚本源码在 [`packaging/windows/`](../packaging/windows/)。

## 4. 签名与平台注意事项

**Android**：v2 + v3 签名（`minSdk 24` 不需 v1）。发布私钥跨 release 固定复用，四个 Secrets 由 `release.yml` 解码：

| Secret | 含义 |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | JKS 文件的 base64 |
| `ANDROID_KEYSTORE_PASSWORD` | keystore 口令 |
| `ANDROID_KEY_ALIAS` | key alias |
| `ANDROID_KEY_PASSWORD` | key 口令 |

密钥轮换或丢失会让已装用户必须卸载重装（并清掉 Keystore 凭据与私有目录快照）。发布证书 SHA-256 指纹写进每版 Release 说明供核对；四个包的签名与 ABI 隔离由 `apksigner verify` 与产物校验保证。本地未注入密钥时 release 变体回落 debug 签名，仅供自测，不能作为 Release 资产。

**macOS**：二进制由 Linux CI 交叉编译，**未经 Apple Developer ID 签名、未经公证，且未经过实机测试**，按试验性支持使用。完整的未确认清单与 Gatekeeper 处理见 [cli.md §6](cli.md#6-macos-注意事项试验性支持)。

**Windows**：**无代码签名**，SmartScreen / Smart App Control 会拦截；`arm64` 可构建但未在 ARM 真机验证。详见 [windows.md §2](windows.md#2-安装与卸载)。托盘端必须按 GUI 子系统构建（`-ldflags "-H=windowsgui"`），release 校验 PE 的 Subsystem=2；安装脚本校验 `.ps1` 的 UTF-8 BOM + CRLF 与 `.bat` 的 CRLF。

## 5. CI

| 工作流 | 触发 | 内容 |
|---|---|---|
| [`ci.yml`](../.github/workflows/ci.yml) | PR / main | `gofmt` / `go vet` / `go test`；分层与浏览器内核守线；隐私占位符检查；linux / darwin（amd64 + arm64）/ windows（amd64 + arm64）/ android 交叉编译与 darwin vet；托盘 PE GUI 子系统；安装脚本 BOM / CRLF |
| [`ci-android.yml`](../.github/workflows/ci-android.yml) | Android 路径变更 | 完整 Android 构建（JDK 17 / SDK 34 / NDK r27c / gomobile，见 [`.github/actions/setup-android-gomobile/`](../.github/actions/setup-android-gomobile/)）+ Kotlin 单测 |
| [`release.yml`](../.github/workflows/release.yml) | `vX.Y.Z` tag | §2 发布流程 |

`ci.yml` 是自检的权威清单；本地提交前自检命令见 [CONTRIBUTING.md §5](../CONTRIBUTING.md)。工具链版本（NDK / gomobile / JDK）以 `.github/actions/setup-android-gomobile/action.yml` 与 `android/README.md` 为准，两处保持一致。

## 6. Release 正文

正文只放「本版变更」与「下载」；安装教程、包内容清单、使用须知一律链接到各端文档，不在正文重复。渲染脚本是 [`render_release_notes.py`](../.github/scripts/render_release_notes.py)，结构固定：

1. **更新内容**：按端分组 `### Mobile / Desktop / CLI / Other`，空组省略。默认从上一 tag 到本 tag 的 commit 标题自动生成（去掉 `类型(范围):` 前缀，尾随 PR 号转链接），范围映射：`android` → Mobile，`desktop` / `windows` → Desktop，`cli` / `skill` → CLI，其余 → Other。
2. **下载**：CLI / Desktop / Mobile 三张「系统 × 架构」表，链接直达对应资产；agent skill 归 CLI 表。安全警示（macOS 未签名未实测、Windows 无代码签名、Android 覆盖安装勿卸载与签名指纹）压缩为表格下注脚，完整说明在各端文档。
3. 末尾 `SHA256SUMS` 链接与 **Full Changelog** compare 链接；不使用 `--generate-notes`。

需要用户口吻的变更说明或版本专属升级提示时，发版前在 `.github/release-notes/<tag>.md` 提交覆盖文件：内容整体替代自动生成的分组清单（`## 更新内容` 标题由脚本补），可含 `> [!IMPORTANT]` 升级提示；没有覆盖文件就用自动草稿。覆盖文件随 tag 入仓，受隐私自检约束。
# CLI 手册（linux / darwin）

`kemaolife` 是给 agent 用的单二进制命令行工具：**课表查询**与**电费查询**。CLI 被问、Windows 主动说、Android 随手看——三端共用一份 Go 核心，CLI 是其中「被问」的形态。

CLI **没有校园网能力**：不判态、不认证、不存校园网凭据。命令、退出码与字段的完整契约见 [contract.md](contract.md)。

## 1. 安装

### 预编译包（推荐）

从 [Releases](https://github.com/EchoPing07/kemaolife/releases) 下载对应平台的 `tar.gz`（产物清单见 [release.md](release.md)），解压后安装：

```bash
# Linux
tar -xzf kemaolife_<版本>_linux_amd64.tar.gz
install -m 0755 kemaolife ~/.local/bin/kemaolife

# macOS：Apple Silicon 用 arm64，Intel Mac 用 amd64
tar -xzf kemaolife_<版本>_darwin_arm64.tar.gz
xattr -d com.apple.quarantine kemaolife        # 解压后、首次运行前，见 §6
install -m 0755 kemaolife ~/.local/bin/kemaolife
```

包内含 `kemaolife` / `LICENSE` / `INSTALL.txt`。下载后先核对 `SHA256SUMS`（Linux 用 `sha256sum -c SHA256SUMS --ignore-missing`，macOS 用 `shasum -a 256 <压缩包>` 与对应行比对）。

### 源码构建

```bash
go build -o kemaolife ./cmd/kemaolife

# macOS 交叉编译
GOOS=darwin GOARCH=amd64 go build -o kemaolife ./cmd/kemaolife
GOOS=darwin GOARCH=arm64 go build -o kemaolife ./cmd/kemaolife
```

学校代码内置默认值 `3126`（学校事实，非个人隐私），可用 `kemaolife config set customercode <学校代码>` 或构建期 `-ldflags "-X main.defaultCustomerCode=<学校代码>"` 覆盖。

## 2. 配置

```bash
kemaolife config init                          # 交互录入学号与教务密码（0600）
kemaolife config show                          # 确认配置状态（学号与密码恒脱敏）
kemaolife config set customercode <学校代码>    # 需要时覆盖学校代码
```

- 凭据存 `~/.config/kemaolife/config.json`（目录 `0700` / 文件 `0600` / 原子写，`XDG_CONFIG_HOME` 优先），快照缓存存 `~/.cache/kemaolife/`（`XDG_CACHE_HOME` 优先）。macOS 与 Linux 同口径，不用 Keychain。
- `config init` 默认关闭密码回显；终端不支持关闭回显时，会先在 `stderr` 明确警告「输入可能被看到」再继续读取，不静默降级。
- 不要把凭据写进命令行参数（会进 shell 历史）。

## 3. 日常使用

```bash
kemaolife --json timetable                     # 今天什么课（默认当日）
kemaolife --json timetable week [N]            # 当周 / 第 N 周
kemaolife --json timetable date <YYYY-MM-DD>   # 指定日期
kemaolife --json timetable sync                # 只刷新快照，不查询
kemaolife --json electricity                   # 余额 + 每日用量
kemaolife --json electricity --month <YYYY-MM> # 指定月份用量
```

**每次课表查询都会先联网刷新一次**（会话有效时省掉登录），成功直接给最新数据；断网或会话失效时自动回落旧快照并标 `stale:true` + `syncedAt`，查询照常成功（退出码 `0`）。飞行模式下也能查整学期课表。`sync` 只在首次使用或快照被清时需要。

agent 固定带 `--json`、只解析 `stdout`；`stderr` 只有诊断与交互提示。全局参数（`--json` / `--config` / `--timeout` / `--verbose`）可放在子命令前后，详见 [contract.md §2](contract.md#2-cli-命令)。

## 4. 输出怎么读

字段定义见 [contract.md §7](contract.md#7-数据结构)，这里只列解读要点：

**课表**

- `entries: []` 表示当天没课，不是查询失败。
- `inSemester: false` 表示该日期不在本学期范围（假期 / 越界周次），这是**成功**（退出码 `0`），不是错误。
- **`stale: true` 表示本次刷新失败、返回的是旧快照**：必须告诉用户数据时间（`syncedAt`），不要说成「最新课表」。
- `periods` 是小节节次（1..12），`time` 已按大节换算好；`location` 是该课节所属时段的真实地点，不是整门课所有地点的汇总。
- `timetable week [N]` 的 `lastWeek` 是可选字段（学期最后一周），缺省视为 `0` = 未知。

**电费**

- 一次调用同时返回余额（`status`）与每日用量（`usage`），任一主请求失败整体报错，此时不要用上一次的数字冒充本次结果。
- **`collectedAt` 是数据采集时刻，`dataUntil` 是数据截止日**，都不是实时值：电表约有 1 天延迟，回答时必须写明「数据截至」。
- `days[]` 按日期升序；`dailyAvg` / `estimatedDays` 为 `null` 表示数据不足，不要编造可用天数。

## 5. 失败怎么办

先看退出码，再看 `error.code`（完整映射见 [contract.md §8](contract.md#8-错误码)）：

| 退出码 | 含义 | 应对 |
|---|---|---|
| `0` | 成功 | 读 `data` |
| `1` | 业务失败 | 读脱敏后的 `error.message`，向用户说明 |
| `2` | 参数错误 | 修正命令后重试，不要原样重跑 |
| `3` | 凭据缺失 / 会话失效 / 快照缺失 | 引导 `config init` 或 `timetable sync` |
| `4` | 网络或接口不可用 | 课表通常已自动回落旧快照（看 `stale`）；电费明确报错，不用缓存 |

常见错误码：`3001` 凭据缺失（引导配置）；`3002` 会话失效（刷新内部会重登一次，仍失败且有旧快照时按 `stale:true` 返回，不要把它当 `4002`）；`3003` 快照缺失 / 损坏（引导 `timetable sync`）；`4001` 网络不可达；`4002` 响应格式异常（上游可能改版，如实告知，不要自己修正协议）；`4003` 账号密码错误；`5001` 未绑定房间（需先在 H5 页面绑定，本工具不做绑定）。

## 6. macOS 注意事项（试验性支持）

**macOS CLI 未经过实机测试。** 仓库无 macOS 真机，darwin amd64 / arm64 仅交叉编译 + 静态审查 + 自动化单测通过，未在真实 Mac 上运行。以下五项均未实机确认：

1. 真实终端 `TIOCGETA` / `TIOCSETA` 回显控制与恢复；
2. Gatekeeper `com.apple.quarantine` 拦截与 `xattr -d` 处理；
3. 系统根证书链下的 HTTPS 请求；
4. `time.Local` 取到本地时区；
5. Apple Silicon 上 arm64 二进制首次执行不被系统拦截。

**预编译包未经 Apple Developer ID 签名、未经 Apple 公证（notarization）**：macOS 二进制由 Linux CI 交叉编译，无 Apple 付费账号证书、不做公证；arm64 二进制带 Go 链接器自动生成的 ad-hoc 代码签名（Apple Silicon 要求），这不等于 Apple 签名。浏览器下载解压后二进制可能带 `com.apple.quarantine`，首次运行被 Gatekeeper 拦截，需在解压后、首次运行前手动执行：

```bash
xattr -d com.apple.quarantine kemaolife        # 或 xattr -r -d com.apple.quarantine <目录>
```

仍不做 Developer ID 签名 / 公证 / Keychain / 第三方依赖。命令、输出与退出码和 Linux 一致。请按试验性支持使用，遇到问题带 macOS 版本与现象提 Issue。

## 7. Agent skill

[`skills/kemaolife/SKILL.md`](../skills/kemaolife/SKILL.md) 是配套 skill，复制到 agent 的 skill 目录即可使用，Release 也提供单独的 skill zip。skill 必须写明：

1. **触发场景**：今天有什么课 / 第 N 周课表 / 某天有没有课 / 电费还剩多少 / 版本与帮助。不含「能不能上网」——CLI 无校园网能力。
2. **命令速查**：`kemaolife` 前缀、子命令含义、`timetable` 默认当日、`electricity` 一条命令同时给余额与用量、周次与 `weekdayCn` 口径、`periods` 与 `time` 的含义。
3. **输出解读**：见 §4；`stale:true` 必须向用户说明数据时间。
4. **失败处理**：见 §5。
5. **隐私约束**：不复述密码、学号；不把姓名、宿舍号、`roomVerify` 拼进回复或日志；不代查他人；`config show` 中学号恒 `****`。完整规则见 [privacy.md](privacy.md)。
6. **固定用法**：所有命令带 `--json`，只解析 `stdout`。

## 8. 不做的事

GUI / TUI / 富文本输出、常驻、定时提醒、系统通知、区段查询、课表导出命令、校园网状态查询与凭据存储、Developer ID 签名 / 公证 / Keychain / 第三方依赖。windows CLI 为后续计划。
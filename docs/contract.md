# 对外契约参考

本文定义三端共享的**对外契约**：命名、CLI 命令、退出码、JSON 信封、门面 `op`、数据结构、错误码。实现必须与本文一致；协议细节见 [api-docs/](../api-docs/)，架构见 [architecture.md](architecture.md)。

契约变更纪律：改命令、退出码、信封字段、DTO 字段名、错误码、门面 `op` 时，**先改本文再改代码**，三端同步；新增字段必须可选（零值有明确语义）；快照结构变更必须升 `Version`，旧版本直接丢弃重拉，不做兼容迁移。

## 1. 命名基线

单仓库、单 `go.mod`、单二进制 + 子命令。Go module path 采用 GitHub 仓库地址作为权威命名，各端标识围绕 `kemaolife` 派生。

| 项 | 值 |
|---|---|
| Go module | `github.com/EchoPing07/kemaolife` |
| Go import 前缀 | `github.com/EchoPing07/kemaolife/...` |
| CLI 二进制 / 主命令 | `kemaolife`（单二进制 + 子命令） |
| CLI `cmd` 目录 | `cmd/kemaolife/` |
| CLI 配置 / 缓存（linux / darwin） | `~/.config/kemaolife/config.json`（`0600`）/ `~/.cache/kemaolife/`（遵循 `XDG_CACHE_HOME`） |
| CLI skill | `skills/kemaolife/SKILL.md` |
| Windows 二进制 / `cmd` 目录 | `kemaolife-desktop.exe` / `cmd/kemaolife-desktop/` |
| Windows 平台标识 | 自启值 `KemaolifeDesktop`、进程 AUMID `Kemaolife.Desktop`（用于归类托盘 `NIF_INFO` 通知）、数据目录 `%LOCALAPPDATA%\Kemaolife\` |
| Windows 启动参数 | `--background`：仅新实例初始驻留托盘、不显示主窗口；普通双击无参数时显示主窗口。注册表自启命令为 `"<kemaolife-desktop.exe 的绝对路径>" --background` |
| Android Go 门面 | `cmd/kemaolife-mobile/`，`package kemaolife`；`gomobile bind -target=android -androidapi 24 -javapkg com.github.echoping07.kemaolife.bridge -o android/app/libs/kemaolife.aar ./cmd/kemaolife-mobile` |
| Android app | applicationId / namespace / Kotlin 主包 `com.github.echoping07.kemaolife` |

约定：

1. Android 包名全小写：`EchoPing07` 在 applicationId 段小写化为 `echoping07`；`com.github.<user>.<repo>` 与 module path 的 `github.com/<user>/<repo>` 同源反向。
2. 生成 Java 类实测落在 `com.github.echoping07.kemaolife.bridge.kemaolife.Kemaolife`（`-javapkg` 只决定前缀，末段固定取 Go 包名）；Kotlin 调 `Kemaolife.init(...)` / `Kemaolife.call(...)`。改包名须同步 `android/.../data/Bridge.kt` 与 `android/README.md`。
3. 仓库与文档不出现任何凭据明文；学校代码属学校事实，可用内置默认值（当前 `3126`）；示例中的用户自填值写 `<学校代码>`。
4. CLI 支持 linux / darwin（源码构建 + 预编译产物），windows 为后续计划；桌面端仅 Windows、移动端仅 Android。内部标识（`cmd` 目录名、二进制名、AUMID、Android 包名、产物名）冻结，不因支持平台扩展而改名。Windows 端命名用 `desktop`：主窗口 + 托盘程序是给人的；`agent` 一词仅指 CLI 侧的 AI agent。
5. Windows 主窗口固定为今天 / 课表 / 电费 / 校园网 / 设置五页。关闭主窗口只隐藏到托盘；重复启动不带 `--background` 时唤起已有主窗口（已有实例仅驻留托盘、主窗口尚未创建时，通知其创建并显示）；重复启动带 `--background` 时直接退出，不改变已有实例的窗口状态；托盘左键打开主窗口并定位「今天」。窗口行为与完整规则见 [windows.md](windows.md)。

## 2. CLI 命令

```bash
kemaolife --version                             # 版本号（源码常量，不联网）
kemaolife --help                                # 帮助
kemaolife help <command>                        # 子命令帮助

kemaolife config init                           # 交互录入学号 + 教务密码（0600）
kemaolife config set customercode <学校代码>     # 覆盖学校代码内置默认值（默认 3126）
kemaolife config show                           # 展示配置状态；学号与密码恒脱敏

kemaolife timetable                             # 默认当日
kemaolife timetable sync                        # 登录并刷新快照（1 登录 + 2 请求）
kemaolife timetable today                       # 当日
kemaolife timetable week [N]                    # 省略 N = 当周；N = 指定周
kemaolife timetable date <YYYY-MM-DD>           # 指定日期

kemaolife electricity [--month YYYY-MM]         # 一次返回剩余电费 + 每日用量（2 主请求 + 1 采集时刻请求）
```

版本号取自 `internal/version.Version`（仓库唯一来源，三端共用），不由构建参数覆盖；`commit` 由编译期 `-ldflags "-X main.commit=<提交号>"` 注入。`--version` 不联网、不读配置；带 `--json` 时走 §5 信封，`data` 为 `{"version":"<版本号>","commit":"<提交号>"}`，否则输出一行纯文本。`--help` / `help` 输出纯文本帮助，不进信封。

**配置分域**

| 域 | 字段 | 来源 |
|---|---|---|
| 教务 | 学号、教务密码 | `config init` 交互录入；默认关闭回显（无法关闭时 `stderr` 明确警告并继续读取，输入可能被看到）、不进命令行参数、不写日志 |
| 电费 | 学号（默认继承教务，可在 Android 设置页关闭「继承教务信息」后单独配置）、学校代码 `customercode` | 学校代码有内置默认值，支持自定义覆盖；电费接口无密码（`api-docs/002` §1 无鉴权）；实际学号由 `Config.ElectricityStudent()` 决定：独立学号非空（去空白后）则用独立学号，否则回落教务学号 |
| 校园网 | 不存 | CLI 无校园网功能 |

Windows 配置与 CLI 分域独立，不共享文件：学号、教务密码、学校代码、校园网独立凭据及轮询 / 告警设置统一经用户级 DPAPI 加密后存于 `%LOCALAPPDATA%\Kemaolife\`，课表快照含 Cookie，同样加密存储。主窗口 / 托盘只消费本机状态和既有 DTO，不新增配置门面或对外字段。

学校代码默认值 **3126**（学校事实，非个人隐私），写在 `cmd` 层并支持 `-ldflags -X main.defaultCustomerCode=...` 覆盖；用户可 `config set customercode <学校代码>` 再覆盖。**禁止**写成 `internal/core` 包级常量或对外发布的通用常量。`config show` 用 `customerCodeSource: "default" | "custom" | "missing"` 标注来源；`missing` 仅在默认值与用户配置都为空时出现。

`config show` 输出（示例值为占位符）：

```json
{
  "studentId": "****",
  "eduPassword": "****",
  "customerCode": "****",
  "customerCodeSource": "default",
  "configPath": "<用户主目录>/.config/kemaolife/config.json"
}
```

**全局参数**

| 参数 | 语义 |
|---|---|
| `--json` | `stdout` 只输出 §5 信封；诊断信息全部走 `stderr` |
| `--config <path>` | 指定配置文件（默认 `~/.config/kemaolife/config.json`；有 `XDG_CONFIG_HOME` 时优先用之；macOS 同口径） |
| `--timeout <dur>` | 单次网络超时上限（默认 30s；也接受纯秒数） |
| `--verbose` | 诊断文本输出到 `stderr`（不含个人信息） |

全局参数可放在子命令之前或之后；支持 `--flag=value`、`-h` / `-v` 别名与 `--` 终止符。`--` 可放在第一个子命令之前或其参数之后：终止符之后的内容按子命令原样参数处理；`-v` / `--version` 在子命令前后均可生效，子命令之后同样可用 `--json` / `--verbose` / `--config` / `--timeout` / `-h` / `--help`。

**约定**

- 无 `--json` 时输出人类可读表格或单行状态文本，字段与 JSON 同源；允许额外的「数据时间 / 配置引导 / 权限」提示行，不输出 DTO 之外的业务数据。`--json` 时 `stdout` 恒只有信封（交互提示一律走 `stderr`）。
- agent 侧固定带 `--json`，只读 `stdout`。
- `kemaolife timetable` 无子命令等价于 `kemaolife timetable today`。
- 查询类子命令每次都会先尝试联网刷新快照，语义见 §4。`sync` 只做刷新、不查询。
- `kemaolife electricity` 一次调用内部发 2 个主请求（`getbindroom` + `getusedetail`），任一失败整体报错、不返回半份数据；两个主请求成功后还会发 1 个 `getstuindexpage` 取 `collecdate`（真实采集时刻），该请求失败或缺失时静默降级到 `opdtdate`（数据截止日），不影响整体成功。`--month` 省略时取当月。
- 周次从 `1` 起；`week` 越界时返回 `inSemester:false` 而非报错。CLI 的 `timetable week [N]` 只接受正整数（负数周次是门面能力，见 §6 规则 7）。
- 查询指向非学期内日期时，一律返回**成功信封**（`ok:true`、退出码 `0`），由 `data.inSemester:false` + `entries:[]` 标记；不产生 `error`、不触发 `1002`。`timetable week` 省略 N 且当前日期不在学期内时同样按成功返回（7 天全部 `inSemester:false`），不猜周次、不报错。
- 区段查询不作为命令暴露（`QueryRange` 仅供 `core` 内部当周 7 日计算）。
- CLI 不提供校园网状态查询，不存校园网凭据；不做 GUI、常驻、定时提醒、导出命令。

## 3. 退出码

| 码 | 含义 | agent 应对 |
|---|---|---|
| `0` | 成功 | — |
| `1` | 业务失败（接口返回业务错误码等；`inSemester:false` 不属此类） | 读 `error.message` 说明原因 |
| `2` | 参数错误 | 修正命令后再试，不要重试 |
| `3` | 凭据缺失 / 会话失效 | 引导 `kemaolife config init` 或 `kemaolife timetable sync` |
| `4` | 网络或接口不可用 | 课表刷新失败会自动回落旧快照（信封 `stale:true`）；仅当连旧快照都没有时才以 `4` 退出。电费失败按 §8 错误码映射（可为 `1` / `3` / `4`），不使用缓存 |

退出码与 `error.code` 必须同域映射（§8），不得出现「退出码 0 但 `ok:false`」。

## 4. 刷新与降级（CLI）

查询类子命令每次都先刷新快照（`timetable` / `today` / `week` / `date`），刷新失败则回落旧快照。

| 场景 | 行为 |
|---|---|
| 课表查询 · 刷新成功 | 用最新快照返回，`stale:false`，`syncedAt` 为本次同步时间 |
| 课表查询 · 刷新失败但有旧快照 | 沿用旧快照返回，**`stale:true` + `syncedAt`（旧快照的同步时间）**，退出码仍 `0`；不重复报 `3002` |
| 课表查询 · 刷新失败且无旧快照 | 退出码 `4`（`4001`，网络不可用）/ `3`（`3001`/`3002`/`3003`，凭据 / 会话 / 快照问题） |
| 课表查询 · 会话失效 | 刷新内部自动重登一次；仍失败则按上一行处理（有旧快照就 `stale:true` 返回） |
| 课表查询 · 非学期内日期 | 成功信封 + `data.inSemester:false`，退出码 `0`（与刷新成败无关） |
| `timetable sync` | 只刷新不查询；失败直接报错（不回落），退出码按错误码映射 |
| 电费查询 | 2 个主请求 + 1 个采集时刻请求；主请求失败按 §8 映射退出码，采集时刻请求失败静默降级；不使用缓存冒充新数据 |

要点：

1. **刷新失败不阻塞查询**：断网时课表照常可查，但必须让 agent 看到 `stale:true`，以便向用户说明「数据时间」。
2. **刷新失败不等于命令失败**：只要还有旧快照，退出码就是 `0`；`stale` 是唯一「旧数据」标记。
3. **无旧快照时才失败**：这是唯一需要用户先手动 `sync` 的场景（首次使用 / 快照被清）。缺教务凭据但有旧快照时仍按 `stale:true` 返回，退出码 `0`。
4. `sync` 与查询不合并：`sync` 用于「只想刷新、不要结果」，也是首次使用的引导步骤。
5. **会话失效的判定靠响应内容，且失效形状有两种**：① CAS 登录页 HTML；② `jw` 在会话过期后返回 HTTP `200` + 168~188 字节的 JS 跳转桩页（跳回 `aic` 的 `cas_server`），里面不含 `signupInputUserName` / 「统一身份认证」/ `__VIEWSTATE`。判据必须是「是否拿到真实教务内容」，不能只看状态码或登录页标记。
   - 漏判的后果：探活一直返回「有效」，刷新跳过重登，随后以 `4002`（锚点页解析失败）失败；而重登只在 `3002` 时触发，于是会话过期后**永久无法自愈**。
   - 因此「会话失效」必须统一报 `3002`，不得降级成 `4002`；`4002` 留给真正的结构变更（上游改版）。协议事实见 `api-docs/001-timetable` §1。
6. **登录前必须先登出 CAS SSO 会话**：快照里的 `CAS_SessionId` 仍有效时，`GET /cas_server/login` 不给登录表单而是直接 `302` 发 ticket（不校验账号密码）。必须先 `GET /cas_server/logout`（不需凭据，失败不阻断），否则既误报 `4002`，又会在换账号时用旧账号的 SSO 会话换票。协议事实见 `api-docs/001-timetable` §1 / §2.1。

Android 的动作划分与 CLI 不同，Windows 桌面也采用缓存与刷新分离：**查询只读缓存**（`CacheOnly`，0 请求，`stale` 恒 `false`），**联网刷新是独立动作**（Android 经 `timetable.sync`，Windows 直接调用 `SyncTimetable`）。Android 的离线标记是 UI 本地状态，只由 `timetable.sync` 的成败维护，不是信封 `stale` 字段；Windows 启动先读快照再后台同步，切周不联网，同步错误在页面单独显示。本节 `stale:true` 语义专指 CLI 的 `RefreshFirst` 链路。Android 触发策略见 [android.md](android.md)，Windows 触发策略见 [windows.md](windows.md)。

## 5. JSON 信封

CLI `--json` 与 Android 门面共用同一信封，字段名一致。

成功：

```json
{
  "ok": true,
  "data": {},
  "ts": "<响应时间>",
  "stale": false,
  "syncedAt": "<快照同步时间>"
}
```

失败：

```json
{
  "ok": false,
  "error": {"code": 3002, "message": "会话失效，需重新同步"},
  "ts": "<响应时间>"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `ok` | bool | 是否成功 |
| `data` | object | 业务数据，`ok:false` 时缺省 |
| `error` | object | `{code, message}`，`ok:false` 时存在 |
| `ts` | string | 响应生成时间（RFC3339，本地时区） |
| `stale` | bool | 课表查询恒给出。`RefreshFirst`（CLI）刷新失败沿用旧快照时为 `true`，否则 `false`；`CacheOnly`（Windows 今天 / 课表页与 Android 查询类 `op`）**恒 `false`**——纯读缓存没有「本次刷新失败」这回事，Windows 页面与 Android UI 的同步错误依据独立刷新结果单独维护。非课表命令可缺省 |
| `syncedAt` | string | 快照同步时间；课表查询恒给出（`RefreshFirst` 刷新成功时是本次时间、`stale:true` 或 `CacheOnly` 时是本地快照的同步时间）；CLI 与 Android 的 `timetable sync` 也给出 |

`error.code` 使用 §8 的 `1001`–`5001` 域；CLI 退出码 `0/1/2/3/4` 由该域映射得出，**JSON 信封内不出现退出码**。`ok:false` 时 `data` 缺省，`ok:true` 时 `error` 缺省。所有示例按 Go 强类型 DTO 编解码。

## 6. Android 门面

`gomobile bind` 只导出字符串函数，Kotlin 不接触领域对象：

```go
// cmd/kemaolife-mobile/bridge.go —— 唯一导出入口
package kemaolife

// Init 由 Kotlin 传入可写目录与解密后的凭据（不落盘、不进日志）。
func Init(configJSON string) string { /* ... */ }

// Call 请求 JSON 进，响应 JSON 出。
func Call(reqJSON string) string { /* ... */ }
```

### 6.1 op 基线

| op | 参数 | 对应 CLI | 说明 |
|---|---|---|---|
| `ping` | — | — | 自检，返回 `data{pong,version}` |
| `timetable.sync` | — | `kemaolife timetable sync` | 联网刷新快照 |
| `timetable.today` | — | `kemaolife timetable` | `CacheOnly` |
| `timetable.week` | `week`（可选，省略 = 当周） | `kemaolife timetable week [N]` | `CacheOnly` |
| `timetable.date` | `date`（`YYYY-MM-DD`） | `kemaolife timetable date` | `CacheOnly` |
| `timetable.probe` | — | — | 设置页「教务」详情页「测试」 |
| `timetable.export` | `optimizeLocation`（可选 bool，缺省 `false`） | — | 课表页「导出课表」（WakeUp CSV，见 §7.2） |
| `electricity.report` | `month`（`YYYY-MM`，可选） | `kemaolife electricity` | |
| `electricity.probe` | `month`（可选） | — | 设置页「电费」详情页「测试」 |
| `electricity.alert` | `status`、`mode`（可选）、`balanceThreshold`、`daysThreshold`、`state`（可选） | — | 电费低阈值提醒判定（纯计算，0 请求，复用 `core/alert.Evaluate`） |
| `campusnet.status` | — | — | |
| `campusnet.probe` | — | — | 设置页「校园网」详情页「测试」 |
| `campusnet.login` | — | — | |
| `campusnet.logout` | — | — | |
| `campusnet.relogin` | — | — | |
| `log.enable` | `enabled`（bool） | — | 设置页「日志」开关 |
| `log.list` | `limit`（可选，默认 100） | — | 设置页「日志」列表 |
| `log.detail` | `seq`（int） | — | 设置页「日志」单条详情 |
| `log.clear` | — | — | 设置页「日志」清空 |

### 6.2 Init 字段

`Init(configJSON)` 字段（`cmd/kemaolife-mobile/bridge.go` 的 `initConfig`；新增字段均可选，零值有明确语义）：

| 字段 | 语义 |
|---|---|
| `cacheDir` | 应用私有可写目录，用于存快照（含 Cookie） |
| `studentId` / `eduPassword` | 教务凭据，只存在于 Go 进程内存 |
| `customerCode` / `electricityStudentId` | 电费学校代码（空则取 `cmd` 层内置默认值）与可选独立学号（空则继承教务学号） |
| `campusUser` / `campusPassword` / `campusFingerprint` | 校园网独立凭据，不复用教务密码 |
| `timeZone` / `timeZoneOffsetSeconds` | 设备时区名与偏移兜底；`time.Local` 只在进程内校准一次 |
| `resetSnapshot` | 可选 bool，缺省 `false` |
| `requestLog` | 可选 bool，缺省 `false`；为 `true` 时打开请求记录 |

`cacheDir` 为空时 `Init` 回 `2001`；`customerCode` 为空时回落 `cmd` 层内置默认值，`electricityStudentId` 为空时回落 `studentId`；其余字段缺省为空串 / `0`，由对应动作自行判断凭据与能力是否可用。`Init` 成功返回 `data.version`；`ping` 与 `Init` 都只读版本常量、不联网。

### 6.3 规则

1. `op` 统一功能域命名（`timetable.*` / `electricity.*` / `campusnet.*` / `log.*`），不使用别名。
2. 不提供 `config.*`：Android 设置页由 Kotlin 直接写 Keystore 后重新 `Init(configJSON)`，凭据不经过门面回传。`resetSnapshot=true` 时在新会话发布、旧请求取消之后删除本地课表快照，用于凭据身份变更时避免旧账号数据先显示。
3. 校园网 `op` **全部手动触发**；不提供自动登录 `op`（自动模式仅 Windows）。探测失败属正常 `offline` 状态（§7.4），不是错误。
4. 响应一律使用 §5 信封；`error.code` 取 `1001`–`5001`，不输出 CLI 退出码。
5. 未实现 / 未知 `op` 返回参数错误 `2001`，不得静默返回空数据。
6. **课表查询类 `op` 只读本地快照**（`CacheOnly`，0 请求，`stale` 恒 `false`）：刷新是独立动作 `timetable.sync`。查询不要求 HTTP 端口，只依赖 `Clock` / `KVStore`；`timetable.export` 同理。`CacheOption` 只接受 `RefreshFirst` / `CacheOnly`，其他值返回 `2001`。无快照时：仅「快照文件不存在 / 无内容」映射 `3001`（文案为「先刷新一次」）以引导首次同步；快照损坏、版本不匹配、内容不完整或读取失败仍保留 `3003`（文案提示「请重新同步」），不压成配置问题。`timetable.week` 的 `week=0`（或省略）表示当周；正 / 负整数均表示显式周次，负数用于学期开始前的周，Java/Kotlin 侧切周不会跳回当周。
7. `*.probe` 三个 `op` 是「测试」按钮专用：动作本身的成败放在 `data.ok`，外层信封恒 `ok:true`，始终带 `elapsedMs` 与脱敏后的 `message`；失败时另给 `code`（取 §8 错误码）。`message` 只按错误码映射为固定中文文案，不带上游原文、用户输入或内部错误文本。回报只含可展示摘要，不含 `roomName` / `roomVerify` / Cookie；`data.detail` 是白名单字段，未知字段不展示。`electricity.probe` 成功时另给可选 `data.report`（`ElectricityReport`，其中 `roomName` 恒为空、`roomVerify` 恒脱敏），`campusnet.probe` 成功时另给可选 `data.campusStatus`，供 UI 用同一次请求的结果刷新卡片，不再二次联网；老客户端忽略这两个可选字段即可。
8. **会话代次**：每次成功 `Init` 递增会话代次并取消上一代次的请求上下文；在途旧 `Call` 可以继续跑完，但不得覆盖新一代次写出的快照（由会话绑定的 `context` 与代次守卫 KVStore 共同实现）。`time.Local` 只在进程内校准一次，重复 `Init` 不再写全局时区。
9. **请求记录**（设置页「日志」）：`log.enable` 打开 / 关闭记录，打开时清空旧条目；`log.list` 返回最近 `limit` 条摘要（新的在前，环形窗口默认 100 条）；`log.detail` 返回单条完整记录，序号已被丢弃时回 `2001`；`log.clear` 清空并回报清掉的条数。记录内容为该次 HTTP 往返的方法 / URL / 标头 / 请求正文 / 响应状态与标头、正文 / 耗时，标头与正文按原始字节保留，不做字段级遮罩。隐私约束（默认关闭、只存进程内存、不入仓、64 KiB 截断、关闭即清空）见 [privacy.md §7](privacy.md)。`Init` 的 `requestLog` 只用于「恢复上次打开的开关」，关闭一律走 `log.enable`（否则保存凭据触发的重新 `Init` 会静默关掉记录）。
10. **电费低阈值判定**（`electricity.alert`）：纯本地评估（0 请求），直接复用 `core/alert.Evaluate`，不设静默时段。请求须给 `status`（`ElectricityStatus` 必要展示字段：`balance` / `unit` / `collectedAt` / `estimatedDays`，不含 `roomName` / `roomVerify`），可选 `mode`（`"balance"`｜`"days"`，省略默认 `"balance"`；未知取值回 `2001`）、`balanceThreshold`（正数，默认 `10`）、`daysThreshold`（正整数，默认 `3`）、`state`（`{balanceAlerted, daysAlerted}`）。响应 `data` 为 `{notify, title, body, reason, state}`，其中 `reason` 取 `""` / `"balance"` / `"days"`，`body` 恒包含「数据截至 `<collectedAt>`」且不含宿舍标识。

## 7. 数据结构

### 7.1 课表

各命令 / `op` 的 `data` 结构（字段名稳定，新增字段必须可选）：

| 命令 / op | `data` 结构 |
|---|---|
| `timetable` / `today` / `date` | `DaySchedule`（本节） |
| `timetable week [N]` | `{ "week": <int>, "days": [<DaySchedule> × 7], "lastWeek": <int> }`（`lastWeek` 为可选字段；字段缺省视为 `0` = 未知） |
| `timetable sync` | `{ "syncedAt", "week1Monday", "lastWeek", "lessonCount" }`（不含 Cookie） |
| `timetable.export` | `{ "csv", "filename", "syncedAt", "lessonCount" }`（仅 Android 门面，见 §7.2） |
| `electricity` / `electricity.report` | `ElectricityReport`（§7.3） |
| `config init` | `{ "studentId": "****", "eduPassword": "****", "configPath" }` |
| `config set customercode` | `{ "customerCode": "****", "customerCodeSource" }` |
| `config show` | `{ "studentId", "eduPassword", "customerCode", "customerCodeSource", "configPath" }` |
| `--version --json` | `{ "version", "commit" }` |

Windows 端不暴露 CLI / JSON 信封，五页主窗口、托盘菜单、悬浮提示与设置页只消费本机状态与 DTO，不新增对外字段。Windows 的 `GetSchedule` / `GetWeekSchedule` 使用 `CacheOnly`，`SyncTimetable` 独立刷新；电费完整报告使用既有 `ElectricityReport`，不另设 Windows DTO。

`DaySchedule` 示例为字段示意；`<...>` 是占位符字符串，实际数值字段按字段表编码。

```json
{
  "date": "<YYYY-MM-DD>",
  "week": "<周次>",
  "weekday": "<1-7>",
  "weekdayCn": "<一/二/…/日>",
  "inSemester": true,
  "entries": [
    {
      "name": "<课程名>",
      "periods": "<起始小节>-<结束小节>",
      "time": "<上课时间>",
      "teacher": "<教师>",
      "location": "<地点>",
      "shortLocation": "<精简地点，可选>",
      "weeksText": "<周次文本>"
    }
  ]
}
```

| 字段 | 说明 |
|---|---|
| `date` | `YYYY-MM-DD` |
| `week` | 周次，从 1 起 |
| `weekday` | 周一 = 1 … 周日 = 7 |
| `weekdayCn` | 一 / 二 / … / 日 |
| `inSemester` | 是否在本学期范围 |
| `entries[]` | 当天课节；空数组 = 当天没课 |

`entries[].time` 覆盖 `periods` 的全部小节：取起始小节所在大节的开始时间与结束小节所在大节的结束时间，连上多个大节时自动跨越（如 `5-8` → `14:00~16:50`）；不连续节次取首尾小节的覆盖区间，节次超出作息表时为空串。`entries[]` 按 `periods` 起始节次升序；同一节次多门课按课程名排序，保证输出稳定可 diff。`entries[].location` 是该课节所属时段的真实地点，不是整门课所有地点的汇总；多个时段的同地点重复项不得跨时段去重。`entries[].shortLocation` 是可选字段，由 `core` 按课室命名规则由 `location` 纯函数精简得出（如去掉括号挂牌名与房号后的实训室后缀；无法精简时等于 `location`，原始地点为空时省略），供紧凑网格视图按需展示，不修改快照原文、不升快照 `Version`，老客户端忽略即可。`entries[].weeksText` 同样按所属时段生成。上游时间与地点片段无法可靠配对时，同步按 `4002` 报错，不返回地点不可信的快照。

快照结构：

| 字段 | 说明 |
|---|---|
| `version` | 快照格式版本，当前 `2`；版本不匹配即丢弃重拉 |
| `syncedAt` | 同步时间 |
| `week1Monday` | 第 1 周周一（日期锚点） |
| `week1Dates` | 第 1 周 7 个日期 |
| `slotTimes` | 6 个大节时间，如 `08:30~10:00` |
| `lastWeek` | 由课程推断的学期最后一周 |
| `lessons` | 展开后的课节（`weeks` 为数组） |
| `cookies` | 会话 Cookie，**敏感**，加密或 `0600` 存储 |

### 7.2 课表导出（WakeUp 课程表 CSV）

`timetable.export`（Android 门面，无 CLI 对应）把本地快照里的**整学期**课节导出为 WakeUp 课程表可导入的 CSV；只读缓存（`CacheOnly`，0 请求），不联网、不触发刷新。支持可选参数 `optimizeLocation`（bool，缺省 `false`）：为 `true` 时「地点」列写入 `core` 精简后的课室名，排序与去重随之按写入地点计算；为 `false`（或缺省）时保留原始地点。

格式以 WakeUp 官方文档为准（<https://wakeup.fun/doc/import_from_csv.html> 「五、课表正确填写示范」）：

| 项 | 规则 |
|---|---|
| 编码 / 行尾 | UTF-8 **带 BOM** + **CRLF**（与官方模板一致，否则 WPS 打开中文乱码） |
| 表头 | 固定为 `课程名称,星期,开始节数,结束节数,老师,地点,周数`，顺序不可变 |
| `星期` / `开始节数` / `结束节数` | 各是**一个具体数字**，不能是中文、不能带分隔符 |
| 老师 / 地点 | 可为空内容，但**必须写「无」**；留空会导致导入失败 |
| 周数 | 连续写 `1-16`；隔周写 `7-11单` / `12-16双`；间断用中文顿号分隔，如 `1-5、7-11单` |
| 行数 | 一门课多时段 / 不连续节次（`1,3`）各占一行，不压成一个区间；完全相同的行去重 |
| 输入过滤 | 跳过星期不在 1..7、课程名为空、周次为空的课节，周次只保留 1..60；隔周写法仅在同奇偶且连续 3 周以上时使用 |
| 排序 | 星期 → 开始节数 → 结束节数 → 课程名 → 地点；同一快照导出的文件逐字节一致 |
| 含逗号 / 引号的值 | 用双引号包裹并翻倍内部引号（CSV 标准转义），不破坏列结构 |
| 文件末尾 | 最后一条数据行也以 CRLF 结束；快照时间零值不可用时文件名回落 `kemaolife-timetable.csv` |

`data`：`{ "csv", "filename", "syncedAt", "lessonCount" }`。`csv` 是完整正文（含 BOM 与表头），`filename` 由 Go 侧按快照同步时间生成（如 `kemaolife-timetable-<YYYYMMDD>-<HHmm>.csv`），UI 不自己拼文件名；`syncedAt` 是快照同步时间，导出弹层必须与 CSV 一同展示该时间；`lessonCount` 是课节行数（不含表头）。CSV 经系统文件选择器（SAF）落盘，不申请存储权限。

无快照时回 `3001`（引导先刷新一次）；快照损坏 / 版本不匹配 / 内容不完整 / 读取失败，以及快照里没有可导出的课节，一律回 `3003`（提示重新同步），不返回只有表头的「成功」文件。

### 7.3 电费

`kemaolife electricity` 与 `electricity.report` 统一返回 **`ElectricityReport`**，内嵌两个 DTO。示例为字段示意；`<...>` 是占位符字符串，实际数值字段按字段表编码（数值或 `null`）。

```json
{
  "status": {
    "roomName": "<校区>_<楼栋>_<楼层>_<房间>",
    "roomVerify": "<已脱敏>",
    "balance": "<剩余量>",
    "unit": "度",
    "businessType": 0,
    "meterStatus": "<状态>",
    "collectedAt": "<采集时间>",
    "dailyAvg": "<近七日均量或 null>",
    "estimatedDays": "<估算天数或 null>"
  },
  "usage": {
    "yearMonth": "<YYYY-MM>",
    "monthUse": "<当月累计>",
    "dataUntil": "<YYYY-MM-DD>",
    "days": [
      {"date": "<YYYY-MM-DD>", "use": "<当日用量>"}
    ]
  }
}
```

`status`（`ElectricityStatus`）：

| 字段 | 说明 |
|---|---|
| `balance` | 剩余量（度） |
| `unit` | `度` / `吨` |
| `businessType` | `0` 电，`1` 水 |
| `collectedAt` | 数据采集时间；电表约 1 天延迟。展示口径见 [privacy.md §5](privacy.md) |
| `dailyAvg` | 取 `days[]` 最后 7 条的均值；无数据时为 `null`，不足 7 日按实际天数计算；均量 ≤ 0 时为 `0`（`estimatedDays` 为 `null`） |
| `estimatedDays` | `floor(balance / dailyAvg)`；`dailyAvg` 为 `null` 或 ≤ 0 时为 `null` |

`usage`（`ElectricityUsage`）：`yearMonth`、`monthUse`、`dataUntil`、`days[]`。`days[]` 按日期**升序**输出（`core` 显式排序，不依赖上游返回顺序）。

约定：`status` 与 `usage` 分别来自 `getbindroom` / `getusedetail` 两个主请求，**任一失败整体返回错误信封，不返回半份数据**。电费请求使用的学号由 `Config.ElectricityStudent()` 决定：独立学号去空白后非空就用它，否则回落教务学号（对应 Android 设置页「继承教务信息」开关，默认打开）；`collectedAt` 优先取 `getstuindexpage.collecdate`，失败或缺失时降级到 `getusedetail.opdtdate`。`roomName` 是本人宿舍标识，允许在本机 CLI 输出，但不进日志 / 错误信息 / agent 复述；`roomVerify` 恒脱敏。

### 7.4 校园网

```json
{
  "state": "online",
  "errorMsg": "<已知安全提示，可为空>",
  "checkedAt": "<检查时间>",
  "brasName": "<BRAS名称，可为空>",
  "brasAddress": "<BRAS地址，可为空>",
  "vlan": "<VLAN，可为空>",
  "deviceIp": "<本机内网IP，可为空>",
  "mac": "<本机MAC，可为空>"
}
```

`state`：`online`（已在线）｜`unauth`（未认证）｜`offline`（不在校园网 / 探测失败）。

- 该 DTO 供 **Windows 主窗口与托盘**及 **Android「更多」页**使用；**CLI 不暴露**。`checkedAt` 由 `app.GetCampusNetworkStatus` 用注入的 `Clock` 补齐；托盘 UI 本身不展示该时刻。`errorMsg` 来自上游 `Location`，必须经已知安全短语白名单过滤后才进入 DTO，未命中时置空；该字段是历史提示，不参与三态判定，Windows 主窗口 / 托盘菜单与 Android「更多」页校园网卡片均不渲染它，手动判态 / 动作失败原因另由各端错误提示呈现；自动判态失败归 `offline`，不展示失败原因。
- `brasName` / `brasAddress` / `vlan` 是可选的学校侧接入信息，`deviceIp` / `mac` 是可选的本机探测详情，旧客户端可忽略；展示与隐私边界见 [privacy.md §4](privacy.md)。
- 探测失败（超时 / 非 302 / 无 `Location`）返回 `ok:true` + `state:"offline"`，**不是错误**；`campusnet.login` / `logout` / `relogin` 在 `offline` 态调用返回 `4001`。Windows 主窗口与托盘的手动动作绕过门面直接调 `app` 用例，仍按同一语义处理。
- 「下线」为统一术语（`api-docs/003`），不写「登出」；「重登」= 先探测判态，若已在线则下线并复探（约 2~3s 生效等待），若原本未认证则跳过下线；随后登录并复探确认。`distoken` 由每次下线前的探测现取。

## 8. 错误码

`error.code` 与 CLI 退出码同域，便于 agent 直接判断。

| code | CLI 码 | 含义 |
|---|---|---|
| `0` | 0 | 成功（仅内部枚举；成功信封不出现 `error`，故 `code:0` 不出现在 JSON 中） |
| `1001` | 1 | 业务失败（上游 `body.result != "0"`；参数缺失 / 学号无效细分见下） |
| `1002` | 1 | 不在本学期范围（保留码位；常规越界查询按 `data.inSemester:false` 返回成功，不产生本码） |
| `2001` | 2 | 参数错误 |
| `3001` | 3 | 凭据缺失 |
| `3002` | 3 | 会话失效，需重新同步 |
| `3003` | 3 | 快照缺失、读取 / 解析失败、版本不匹配、内容不完整，或本地快照没有可导出的课节 |
| `4001` | 4 | 网络不可达 / 超时（含 `offline` 态调用校园网动作） |
| `4002` | 4 | 响应格式异常（含双层 JSON 解析失败） |
| `4003` | 4 | 上游拒绝（认证失败、账号或密码错误） |
| `5001` | 1 | 未绑定房间（电费） |

细分口径：`body.result = -20003` / `-1`（必填参数缺失或学号无效）按 `3001`（凭据 / 配置问题，退出码 3）；外层 `result_ != "true"` 按 `code_` 与脱敏后的文案映射为 `3001`（配置可自行修正）或 `4001`（上游未开通 / 网络不可用）。上游文案一律脱敏后才进入 `error.message`，不得原样透传。
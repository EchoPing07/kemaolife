# 架构设计

本文回答：**三端功能不同，如何只写一遍核心逻辑。** 对外契约（命令、信封、DTO、错误码、门面 `op`）见 [contract.md](contract.md)，构建与发布见 [release.md](release.md)。

## 1. 结论

**一份 `core`；差异用「端口 + 适配器」隔离；平台优化只发生在最外层。**

判断标准：`core` 不应知道自己跑在哪个平台。出现 `runtime.GOOS`，说明这段逻辑放错层了。

约 80% 代码只写一次（HTTP、解析、快照、周次计算、DTO、错误码、告警判定、用例编排、单测）；各端只写「触发器 + 呈现 + 平台能力」。

| 一份（`core` / `app`） | 各端各自实现 |
|---|---|
| CAS 登录、`Sync()`、快照编解码 | **谁触发**：进程调用 / ticker / 用户点击 |
| `Query(snapshot, date/week)` 纯计算 | **结果给谁**：stdout / 系统通知 / 屏幕 |
| 电费 `getbindroom` / `getusedetail` 请求与解析 | **存哪儿**：`0600` 文件 / DPAPI / Keystore |
| 校园网判态与登录/下线表单构造（纯函数） | **网络 I/O**：探测 302 与 POST 由 `internal/platform` 适配器执行 |
| 共享 DTO、统一错误码、快照版本 | **UI**：无 / 托盘 / Compose |
| 可用天数估算、告警去重判定 | **平台优化**：连接池、退避、缓存镜像等 |

「为平台单独优化」不是改核心，而是同一组用例的不同组合方式，外加一层平台适配器。

## 2. 分层

```
┌───────────────── 触发 / 呈现层（各端独有，全部原生） ─────────────────┐
│ CLI（linux / darwin）：`kemaolife timetable` 单次进程 / stdout / 退出码│
│ Windows：开机自启 + ticker + 托盘                                      │
│ Android：Compose（三 tab / 缓存优先 / 打开联网覆盖）                   │
└───────────────────────────────┬───────────────────────────────────────┘
                                │ 调用同一组用例
┌───────────────────────────────▼───────────────────────────────────────┐
│ internal/app · 用例编排                                    【100% 复用】│
│ SyncTimetable / GetSchedule / GetWeekSchedule / CheckElectricity       │
│ ExportTimetableCSV                                                     │
│ EnsureCampusNetwork / GetCampusNetworkStatus                           │
│ LoginCampusNetwork / LogoutCampusNetwork / ReloginCampusNetwork        │
│ （告警判定不在 app：core/alert.Evaluate 纯函数，由 agent.Engine 调）  │
├────────────────────────────────────────────────────────────────────────┤
│ internal/core · 纯领域逻辑                                 【100% 复用】│
│ timetable: client / snapshot / parse / query(纯函数) / export(纯函数)  │
│ electricity: client / protocol / flexjson / estimate                  │
│ campusnet: status 判定 / auth 表单构造（纯函数）                        │
│ alert: evaluate(纯函数)                                                │
│ model: DTO + 错误码 + 快照版本                                          │
└───────────────────────────────┬────────────────────────────────────────┘
                                │ 只依赖端口接口，不依赖实现
┌───────────────────────────────▼────────────────────────────────────────┐
│ core/ports: KVStore / HTTPDoer / NoRedirectDoer / Clock / Notifier    │
│             / NetworkReady / CampusNetwork                              │
└───────────────────────────────┬────────────────────────────────────────┘
┌───────────────────────────────▼────────────────────────────────────────┐
│ internal/platform · 适配器（build tags / 注入）                         │
│ CLI（linux / darwin）：0700 目录 / 0600 文件 / 原子写 + XDG 口径        │
│ windows：DPAPI + Portal 认证（托盘 / 通知在 cmd）                      │
│ android：快照复用私有目录文件存储，凭据加密留在 Kotlin                  │
└────────────────────────────────────────────────────────────────────────┘
```

依赖方向恒向下：`core` 不 import `app`，`app` 不 import `cmd`，`core` 绝不 import `platform` 实现。

`internal/agent` 是 Windows 端常驻编排（轮询 / 告警 / 托盘状态机），只依赖 `core` + ports + `app`，无平台调用，可跨平台单测。托盘与通知属呈现层，归 `cmd/kemaolife-agent/ui`，不进 `internal/platform`。

## 3. 目录结构

```
kemaolife/                         # GitHub: EchoPing07/kemaolife
├── go.mod                         # module github.com/EchoPing07/kemaolife（单 module）
├── internal/
│   ├── core/
│   │   ├── model/                 # DaySchedule / ElectricityStatus / ElectricityUsage / ElectricityReport / CampusStatus / ErrorCode
│   │   ├── timetable/
│   │   │   ├── client.go          # CAS 登录、fetchWeek1、fetchCourses（经 HTTPDoer）
│   │   │   ├── snapshot.go        # 快照结构、version、编解码
│   │   │   ├── parse.go           # expandWeeks / parseSktime / slotTimes
│   │   │   ├── query.go           # ★ 纯函数：Query / QueryRange（后者仅内部用于当周 7 日）
│   │   │   └── export.go          # ★ 纯函数：WakeUp 课程表 CSV（BOM + CRLF、显式排序）
│   │   ├── electricity/
│   │   │   ├── client.go          # SWAEServlet 表单 + 双层 JSON
│   │   │   ├── protocol.go        # 命令名与字段常量
│   │   │   ├── flexjson.go        # 外层 result_/body/code_ 的实测形状兼容
│   │   │   └── estimate.go        # 近 7 日均量 → 可用天数
│   │   ├── campusnet/
│   │   │   └── campusnet.go       # ★ 纯函数：302 Location 判态 + 登录/下线表单构造
│   │   ├── alert/evaluate.go      # ★ 纯函数：阈值判定 + 去重 + 充值后重置
│   │   ├── porttest/              # 端口内存 fake（KVStore / Clock / Notifier / HTTPDoer）
│   │   └── ports.go               # 只定义接口
│   ├── agent/                     # Windows 端编排（轮询 / 告警 / 托盘状态机，无平台调用）
│   ├── app/                       # 用例编排
│   │   ├── deps.go                # Deps：组装平台实现
│   │   ├── timetable.go           # SyncTimetable
│   │   ├── schedule.go            # GetSchedule / GetWeekSchedule（含 CacheOption）
│   │   ├── electricity.go         # CheckElectricity（2 主请求 + 采集时刻请求）
│   │   ├── export.go              # ExportTimetableCSV（只读快照、0 请求）
│   │   ├── config.go / timefmt.go # 配置读写与本地时区格式化
│   │   └── campusnet.go           # 校园网用例（GetCampusNetworkStatus / Login / Logout / Relogin / Ensure）
│   ├── version/                   # 仓库级发布版本号（三端共用唯一来源，见 release.md）
│   └── platform/                  # 各端适配器
│       ├── store_file.go          # KVStore：0700 目录 / 0600 文件 / 原子写（Android 复用私有目录）
│       ├── store_singlefile.go    # 整份配置文件（config.json）
│       ├── httpdoer.go            # 强制 tcp4 + DoNoRedirect
│       ├── paths.go / atomic.go   # XDG 目录与原子写（linux / darwin 同口径）
│       ├── clock.go               # RealClock
│       ├── terminal.go            # 默认关闭回显；失败时 stderr 警告后继续读
│       ├── termios_{linux,darwin,other}.go  # 按平台选 termios 实现
│       ├── notify_stub.go         # !windows && !android → no-op
│       ├── campusnet_stub.go      # !windows && !android → 返回 2001（CLI 不提供校园网能力）
│       ├── dpapi_windows.go / datadir_windows.go / autostart_windows.go
│       ├── portal.go / portal_{windows,android,other}.go  # Portal 探测 / 登录 / 下线 / 重登
│       └── requestlog.go          # Android 请求记录（包在既有 Transport 外层，见 privacy.md §7）
├── cmd/
│   ├── kemaolife/                 # CLI（linux / darwin）：单二进制 + 子命令
│   ├── kemaolife-agent/           # Windows：托盘 + 开机自启 + ticker（GUI 子系统无控制台）
│   │   ├── console_windows.go     # FreeConsole 兜底
│   │   └── ui/                    # 纯 Win32（syscall 直调 user32/gdi32/shell32），内嵌图标、菜单与通知
│   └── kemaolife-mobile/          # Android：gomobile bind 薄门面（§6）
├── android/                       # Kotlin + Compose，仅渲染与门面调用（构建见 android/README.md）
├── skills/kemaolife/SKILL.md      # agent skill
├── packaging/windows/             # Windows 便携包的安装 / 卸载脚本与 INSTALL.txt
├── .github/workflows/             # ci.yml / ci-android.yml / release.yml（见 release.md）
├── api-docs/                      # 协议事实（唯一来源）
└── docs/                          # 本文及其余文档
```

单 module 的理由：三端共用 `model` 与错误码，拆 module 后改一次 DTO 要同步三处，必然漂移；交叉编译只需 `GOOS=... go build ./cmd/...`。

## 4. 端口

```go
// internal/core/ports.go —— 只定义，不实现
package core

type KVStore interface {
    Get(key string) ([]byte, error)
    Put(key string, v []byte) error // 实现方负责权限 / 加密 / 原子写
    Delete(key string) error
}

type HTTPDoer interface {
    Do(*http.Request) (*http.Response, error)
}

// 可选能力：CAS 跨主机登录需逐跳吸收 Cookie。
type NoRedirectDoer interface {
    DoNoRedirect(*http.Request) (*http.Response, error)
}

type Clock interface {
    Now() time.Time
}

type Notifier interface {
    Notify(title, body string) error
}

// 校园网：自动模式只问「在不在校园网、要不要先认证」。
type NetworkReady interface {
    EnsureCampusNetwork(ctx context.Context) error
}

// 校园网：Windows 手动模式与 Android 手动动作共用的原始操作。
// 纯函数（ParseProbe / 表单构造）留在 core/campusnet，I/O 由平台适配器实现。
type CampusNetwork interface {
    Status(ctx context.Context) (CampusStatus, error)
    Login(ctx context.Context) error
    Logout(ctx context.Context) error
    Relogin(ctx context.Context) error // 先判态：在线则下线 → 复探；未认证跳过下线；随后登录 → 复探确认
}
```

`CampusNetwork` 与 `NetworkReady` 并列，不是别名：后者是「自动恢复」的组合用例入口。

| 端口 | 隔离的差异 | 额外收益 |
|---|---|---|
| `KVStore` | `0600` 文件 / DPAPI / Keystore | 快照可用内存实现做单测 |
| `NoRedirectDoer` | CAS 跨主机逐跳重定向 | 手动吸收路径参数里的 Cookie，避免自动跟随丢会话 |
| `HTTPDoer` | 代理、TLS、强制 IPv4、超时 | `httptest` 可模拟「会话失效返回 200」、双层 JSON、业务错误码 |
| `Clock` | 时区 | 周次、学期边界、`inSemester`、告警去重可注入任意日期做单测 |
| `Notifier` | Windows 通知 / no-op | 告警判定纯函数化；Windows 实现是 `cmd/kemaolife-agent/ui.Tray.Notify` |
| `NetworkReady` | Portal / 其他认证方式 | `core` 只放纯函数；探测 / 登录的 HTTP I/O 与自签证书策略锁在平台层 |
| `CampusNetwork` | 手动判态 / 登录 / 下线 / 重登 | Windows 与 Android 适配器实现同一组原语，`app` 用例不关心平台 |

**强制 IPv4**：教务 CAS 域名解析出了 AAAA 记录，但 aic 服务器没有 IPv6 监听。默认双栈拨号会先试 IPv6，连不上只能等超时（约 10 秒）再回退 IPv4。所以这里要的是**禁用 IPv6**，而不是「优选」：

```go
// internal/platform/httpdoer.go
dialer := &net.Dialer{Timeout: 5 * time.Second}
transport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        return dialer.DialContext(ctx, "tcp4", addr) // 强制 IPv4
    },
    TLSHandshakeTimeout: 10 * time.Second,
}
```

`Deps.HTTP` 是教务与电费共用的单一 `HTTPDoer`，`tcp4` 同时作用于两者；Portal 为独立平台适配器，不在此客户端内。

## 5. 用例层：触发器留在外面

```go
func SyncTimetable(ctx context.Context, d Deps, cfg model.Config) (SyncResult, error)  // 含可选会话探活
func GetSchedule(ctx context.Context, d Deps, cfg model.Config, date model.Date, mode CacheOption) (ScheduleResult, error)
func GetWeekSchedule(ctx context.Context, d Deps, cfg model.Config, week int, mode CacheOption) (WeekResult, error)
func ExportTimetableCSV(ctx context.Context, d Deps, cfg model.Config) (ExportResult, error) // 只读快照、0 请求
// CacheOption：RefreshFirst = 先刷新再计算（CLI）；CacheOnly = 只读本地快照、0 请求、stale 恒 false（Android）
func CheckElectricity(ctx context.Context, d Deps, cfg model.Config, yearMonth string) (model.ElectricityReport, error)
func alert.Evaluate(prev alert.State, cur model.ElectricityStatus, now time.Time, cfg alert.Config) (alert.Decision, alert.State) // ★ 纯函数
func EnsureCampusNetwork(ctx context.Context, d CampusDeps) error                    // 仅 Windows 自动模式
func GetCampusNetworkStatus(ctx context.Context, d CampusDeps) (model.CampusStatus, error)
func LoginCampusNetwork(ctx context.Context, d CampusDeps) error
func LogoutCampusNetwork(ctx context.Context, d CampusDeps) error
func ReloginCampusNetwork(ctx context.Context, d CampusDeps) error
```

差异只体现在「谁来调、多久调、结果去哪、活多久」：

| | CLI（linux / darwin） | Windows 常驻 | Android |
|---|---|---|---|
| `GetSchedule` / `GetWeekSchedule` | 每次进程调用一次，`RefreshFirst`：内部先刷新 | 不调 | 打开 / 切周，`CacheOnly`：只读本地快照 |
| `SyncTimetable` | 显式 `sync`（只刷新不查询） | 不调 | 打开 App（不受最小查询间隔约束）、切回课表 tab（按间隔）、手动刷新 / 设置页「测试」（不受节流） |
| `ExportTimetableCSV` | 不调 | 不调 | 课表页「导出课表」：只读快照，SAF 落盘 |
| `CheckElectricity` | 显式 `kemaolife electricity` | 启动立即查一次，之后 ticker 默认 4 小时；菜单可「立即查询电费」 | 打开 App + 进入更多按间隔 + 手动刷新 |
| `alert.Evaluate` | 不调 | 电费成功后由 `agent.Engine` 调用 | 不调 |
| `EnsureCampusNetwork` | 不调 | 仅自动模式，每轮电费轮询前置 | 不调 |
| `GetCampusNetworkStatus` 等 | 不调 | 启动立即判态一次，之后独立 ticker 默认 120s；手动按钮用 `CampusNetwork` | 「更多」页 / 手动动作前（默认仅 WiFi 下探测） |
| 结果去向 | stdout（`--json` 只出数据） | 托盘状态 + 悬浮提示 + 系统通知 | Compose 三 tab |
| 进程生命 | 秒级 | 天级（登录时自启一次，不保活） | 分钟级，随时被杀 |

CLI 不调用校园网用例。校园网用例用独立的 `app.CampusDeps`（`Clock` / `Net` / `Ready`），不挂在 `app.Deps` 下；`platform/campusnet_stub.go` 与 `notify_stub.go` 是备用实现（返回 `2001` / no-op），当前无装配点。

**课表查询 = 刷新 + 计算（CLI）**：`GetSchedule` 内部先调 `SyncTimetable`，成功后用新快照计算；刷新失败则回落旧快照并置 `stale:true + syncedAt`。刷新与查询是同一个用例内的两步。

**Android 走 `CacheOnly`**：查询只读本地快照、0 请求、`stale` 恒 `false`；联网刷新是独立动作（`timetable.sync`），成功后 UI 回读缓存覆盖界面。两者不合并的原因很直接：切周如果走「查询即刷新」，等于每次翻页都白打一次教务接口。各触发时机的完整规则见 [android.md](android.md)。

这些优化没有一个需要改 `core`：Android「不做后台」与 Windows「常驻」只是分别调用同一组用例；告警判定走 `core/alert.Evaluate`，只写一份、只测一份。

## 6. Android 复用：gomobile 字符串门面

`gomobile bind` 产出 `.aar`，Go 只导出字符串函数；UI 用 Kotlin / Compose 原生写。

- `core` / `app` 100% 复用，JNI 边界最窄：`gobind` 只允许导出字符串等受限类型，Go 侧只保留 `Init` / `Call` 两个字符串入口，DTO 不跨边界。
- 代价：需要 Android SDK / NDK；`x/mobile` 非官方正式支持。构建工具链与已知坑见 [android/README.md](../android/README.md)。
- 已排除的替代方案：Go 本地服务 + WebView（浏览器内核）、Kotlin 重写查询（周次算法漂移）、Gio 全 Go UI（违背三端差异定位）。

门面的 `op` 基线、`Init` 字段与响应规则是对外契约，见 [contract.md §6](contract.md#6-android-门面)。平台能力分工：

- Kotlin 启动时通过 `Init(configJSON)` 把「可写目录 + 解密后的凭据 + 设备时区」交给 Go；Go 用 `platform.NewFileStore(cacheDir)` 把快照留在应用私有目录，凭据只存在于进程内存。
- **加密与持久化留在 Kotlin，Go 只处理内存中的凭据。**设置页由 Kotlin 直接写 Keystore，保存后重新 `Init`，门面不提供 `config.*` op。
- 会话代次由 `generationStore` 守卫 `KVStore`：换账号后旧请求不能覆盖新会话快照；这不是新端口。
- 非敏感偏好、主题、课表展示镜像、启动时序、横向翻页、系统分享、QQ 交流群跳转、取件码跳转、检查更新都留在 Kotlin 呈现层，不重新实现业务计算，也不新增端口或门面 `op`。
- `CampusNetwork` 的 Android 适配器：探测地址是 HTTP 明文，Android 9+ 默认禁明文 → 用 `network_security_config.xml` 只对该探测地址放行，不设全局 `usesCleartextTraffic`；Portal 自签证书跳过校验由 Go 侧 Portal 专用 HTTPS 客户端完成，不全局 `InsecureSkipVerify`。
- 时区必须由 Kotlin 传给 Go（Android 的 zoneinfo 不在 Go 默认查找路径），见 [android/README.md §6](../android/README.md)。

## 7. Windows 原生层

Windows 是一个原生托盘程序，不是浏览器套壳：用标准库 `syscall` 直调 user32 / gdi32 / shell32 与 advapi32 / crypt32 实现托盘、菜单、设置窗口、通知、自启注册表项与 DPAPI，单二进制、免 .NET / WebView2 / 第三方依赖。通知由托盘 `NIF_INFO` 承载，进程启动时设置 `Kemaolife.Agent` AUMID 用于归类；不承诺 WinRT Toast 或操作中心归档。

校园网默认手动模式，自动模式才做轮询前置：

```
自动模式：启动 → EnsureCampusNetwork() → CheckElectricity() → alert.Evaluate() → Notifier
手动模式：启动 → CheckElectricity() ───────────────────────────────────→ alert.Evaluate() → Notifier
                          │（前置认证失败只更新托盘状态，不阻断本轮电费，也不自动登录）
```

行为规则（自启时机、轮询间隔、告警、设置项）见 [windows.md](windows.md)。

## 8. 依赖约束

| 目录 | 允许 | 禁止 |
|---|---|---|
| `internal/core/**` | 标准库、`encoding/json`、`http.Request`（经接口）、`time.Time`（经接口） | 路径、`syscall`、平台包、UI 库、全局可变状态、直接 `time.Now()`、`fmt.Print*` |
| `internal/app/**` | `core` + ports | 平台包、UI、直接文件 I/O |
| `internal/agent/**` | `core` + ports + `app` | 平台包、UI、`runtime.GOOS`、直接 `time.Now()`、`fmt.Print*` |
| `internal/platform/**` | 平台 API、build tags、cgo/purego | 业务逻辑、DTO 定义 |
| `cmd/**` | 参数解析、输出格式、触发循环、原生 UI、组装 `Deps` | 业务逻辑、协议解析、浏览器内核 |
| `android/**` | Compose UI、渲染、凭据加密、生命周期 | 课表/电费/认证计算、浏览器内核 |

新增平台差异时：先加端口（`internal/core/ports.go`），再写适配器；禁止在 `core` 里加 `if isWindows` 分支。每个新端口实现都要有内存 fake。

可执行的自检命令清单见 [CONTRIBUTING.md §5](../CONTRIBUTING.md)，CI 权威清单是 `.github/workflows/ci.yml`。

## 9. 复用收益

1. **周次计算只写一次只测一次**：`week1Monday` 锚点、跨周、`inSemester:false`、空课、越界周——三端不会出现「安卓算错一周」。
2. **接口变更只改一处**：网关加签名 / 改字段，改 `core` 三端同步受益。
3. **坑只踩一次**：「会话失效返回 200」、电费缺 `account` 外层仍成功、数据延迟等，全部有 `httptest` 回归。
4. **无版本漂移**：单 module + 交叉编译。
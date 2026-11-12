# Android 构建说明

> 状态：M3 已在 Android 15 真机跑通课表、电费与校园网手动动作的基础链路；单页周视图 / 横向翻页、WakeUp 课程表 CSV 导出、多级设置与外观页由 Go 单测、Kotlin 纯函数单测及构建流程覆盖，增量未在真机复验。
> `v0.3.0` 起 `release.yml` 自动产出四个已签名 APK；当前源码版本为 `v0.4.1`（见 `../internal/version/version.go`）。同版 Release 另新增 macOS 预编译 CLI 两个包（darwin amd64 / arm64，未经 Apple 签名 / 公证、未经过实机测试），本文件只说明 Android APK 的构建。
> 契约以 `docs/002` 为准，本文件只说「怎么把 APK 造出来」；呈现层行为详见 `docs/002 §5.3`。

## 1. 前置工具

| 组件 | 版本 | 说明 |
|---|---|---|
| JDK | **17** | AGP 8.7 要求 17；JDK 21+ 不保证可直接用 |
| Android SDK Platform | `android-34` | `compileSdk = 34` |
| Android Build-Tools | `34.0.0` | `apksigner` 也来自这里 |
| Android NDK | **r27c**（`27.2.12479018`） | `gomobile bind` 编译 `libgojni.so` 用 |
| Go | 1.24+ | 与 `../go.mod` 一致 |
| `gomobile` / `gobind` | 与 `go.mod` 里的 `golang.org/x/mobile` 同版本 | 见下 |

`ANDROID_HOME` / `ANDROID_NDK_HOME` 必须能定位到上面两项；`local.properties` 里写 `sdk.dir`。

`gomobile` 必须与 `go.mod` 里记录的 `golang.org/x/mobile` **完全同版本**，
否则 `gobind` 找不到 `golang.org/x/mobile/bind` 包：

```bash
go install golang.org/x/mobile/cmd/gomobile@"$(go list -m -f '{{.Version}}' golang.org/x/mobile)"
go install golang.org/x/mobile/cmd/gobind@"$(go list -m -f '{{.Version}}' golang.org/x/mobile)"
gomobile init
```

> `go.mod` 里出现 `golang.org/x/mobile` 是 gomobile 的硬要求（它要读模块里的
> `bind` 包）；这条依赖只服务构建工具链，不进 App 的运行时逻辑。

> 工具链版本（NDK / `gomobile` / JDK）仍属 `TODO(D13)` 待确认项；此处按描述性基线记录，
> 不代表 D13 已定稿。`.github/actions/setup-android-gomobile/action.yml` 与本节保持同一版本。

## 2. 生成 AAR（把 Go 门面编进 Android）

```bash
# 在仓库根目录
gomobile bind -target=android -androidapi 24 \
  -javapkg com.github.echoping07.kemaolife.bridge \
  -o android/app/libs/kemaolife.aar ./cmd/kemaolife-mobile
```

产出的 Java 类是 `com.github.echoping07.kemaolife.bridge.kemaolife.Kemaolife`。

> **注意多出来的一层 `kemaolife`**：`-javapkg` 只决定前缀，末段固定是该 Go 包的包名
> （`package kemaolife`）。M0.5 spike 实测如此，Kotlin 的 import 已按实测写。
>
> 改 `docs/002 §0` 里的包名时必须同步改这里与 `Bridge.kt` 的 import。

> `libs/*.aar` 是构建产物，**不入版本库**（见 `.gitignore`）；
> 新克隆仓库后先跑一次上面的命令。
>
> 改动 `cmd/kemaolife-mobile/bridge.go` 里的门面 `op` 后，必须先重新执行本节命令生成 AAR，
> 再跑 `:app:assembleDebug`；否则 Kotlin 侧仍链接旧的 AAR。

## 3. 本地构建 APK

```bash
cd android
./gradlew :app:assembleDebug        # 4 个 ABI 的 debug APK
./gradlew :app:assembleRelease      # 未注入发布密钥时回落 debug 签名，仅供自测
./gradlew :app:testDebugUnitTest    # 信封 / DTO / probe、查询间隔、镜像、启动时序、课程布局、翻页与分享正文纯函数单测
```

产物在 `app/build/outputs/apk/<变体>/app-<ABI>-<变体>.apk`。装到真机：

```bash
adb install -r app/build/outputs/apk/debug/app-arm64-v8a-debug.apk
adb shell am start -n com.github.echoping07.kemaolife/.MainActivity
```

`versionCode` 默认 `1`；发版由 `release.yml` 从 `vX.Y.Z` tag 计算并注入 `-PversionCode=<整数>`，规则 `major*10000 + minor*100 + patch`。跨版本必须严格递增，否则 Android 拒绝覆盖安装。

## 4. 已签名 Release 构建

正式分发用 `release.yml`：它安装本节同一套工具链、生成 AAR、从 GitHub Actions Secrets 解码发布 keystore，再执行 `:app:assembleRelease "-PversionCode=<整数>"`，最后用 `apksigner` 校验四个 APK 的 v2/v3 签名与证书指纹。

四个 Secrets（维护者在本机生成并保管，工作流代码不含任何口令或路径）：

| Secret | 含义 |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | JKS 文件的 base64 |
| `ANDROID_KEYSTORE_PASSWORD` | keystore 口令 |
| `ANDROID_KEY_ALIAS` | key alias |
| `ANDROID_KEY_PASSWORD` | key 口令 |

发布私钥**跨 release 固定复用**；一旦丢失或轮换，已安装用户必须卸载重装（Android 不允许换签名覆盖安装），而卸载会清掉 Keystore 凭据与应用私有目录快照。公钥 SHA-256 指纹不是隐私，会写进每版 Release 说明供核对。

本地要用同一把密钥构建时，只需设置 `KEMAOLIFE_KEYSTORE_FILE` 指向 JKS 路径，并导出上面后三个同名环境变量，再跑 `:app:assembleRelease`。本地不注入时 `release` 变体自动回落 debug 签名，**不能作为 Release 资产**。

## 5. 已知构建环境坑

| 现象 | 原因 | 处理 |
|---|---|---|
| `Your project path contains non-ASCII characters` | AGP 默认拒绝非 ASCII 工程路径 | `gradle.properties` 里已设 `android.overridePathCheck=true`；实测可构建 |
| `java.io.IOException: 文件名、目录名或卷标语法不正确` | `local.properties` 里 `sdk.dir` 用了转义反斜杠 | 写成正斜杠：`sdk.dir=D:/android-sdk` |
| `gomobile bind requires golang.org/x/mobile in the current module` | gomobile 要求模块内能解析 `bind` 包 | 见 §1 的 `go install` 与 `go.mod` 依赖 |
| `javac` 报 `编码 GBK 的不可映射字符` | gobind 把 Go 导出函数的注释原样写进 `.java`，javac 按平台默认编码读 | **导出符号的注释只用 ASCII**（`bridge.go` 的 `Init` / `Call` 已注明） |
| 启动即闪退 `ClassNotFoundException: ...MainActivity` | Kotlin 文件的 `package` 与所在目录不一致（`MainActivity.kt` 在根包目录却写了 `.ui` 包） | Kotlin 的 `package` 必须与目录一致：`MainActivity.kt` 在 `.../kemaolife/` 下就得是 `package com.github.echoping07.kemaolife` |
| 所有时间戳都带 `Z`、比实际差 8 小时 | Go 在 Android 上找不到 zoneinfo，`time.Local` 退化为 UTC | 见 §6 |
| 把工程放在中文路径下构建 | Windows + AGP 对非 ASCII 路径仍有零散问题 | 建一个 ASCII junction：`mklink /J C:\kmbuild D:\<中文路径>` |
| `apksigner verify` 在 PowerShell 里找不到 | 用的是 `apksigner.bat` 而非 Linux 版脚本 | CI 在 `ubuntu-latest` 上用 `$ANDROID_HOME/build-tools/34.0.0/apksigner`；Windows 本地用 `.bat` |

## 6. 时区必须由 Kotlin 传给 Go

**这是一个真机才暴露的坑**：Android 的 tzdata 在 `/system/usr/share/zoneinfo`，
不在 Go 的默认查找路径里。不处理时 `time.Local` 会退化成 **UTC**，
所有时间戳都差 8 小时（「数据截至 22:48」显示成「14:48Z」），
`DataUntil` 这类日期还可能整体倒退一天。

三处配合解决：

1. `cmd/kemaolife-mobile/bridge.go` 匿名导入 `time/tzdata`（内嵌约 450KB 时区库）；
2. Kotlin 在 `Init` 时传 `timeZone`（`TimeZone.getDefault().id`）与
   `timeZoneOffsetSeconds`（作为 `LoadLocation` 失败时的兜底）；
3. Go 侧 `applyTimeZone` 把 `time.Local` 设成设备时区。

> 为什么不改 `core`：`core` / `app` 一律通过 `Clock` 与 `time.Local` 取时间
> （AGENTS §6），它们不该知道自己在哪个平台。校准只发生在门面初始化这一处。

## 7. 应用图标

图标资源在 `res/mipmap-*/`：三份启动图标 PNG 都是同一张正方形原图的等比缩放，
不裁切、不扩边、不加光晕、不画遮罩；`ic_about_logo.png` 是同一张插画在关于页的独立资源。
形状与安全区由 launcher 决定。

| 资源 | 做法 | 谁在用 |
|---|---|---|
| `ic_launcher.xml`（`mipmap-anydpi-v26`） | 自适应图标：四象限背景 + 前景 | Android 8.0+ |
| `ic_launcher_foreground.png` | 整张原图缩放到 **108dp 画布** | 同上 |
| `ic_launcher.png` | 整张原图缩放到 **48dp** | API 24/25、不支持自适应图标时 |
| `ic_launcher_round.png` | 同 `ic_launcher.png`（不预裁圆） | 类原生 / 圆形屏 launcher |
| `ic_about_logo.png` | 关于页插画，连续曲率裁切展示 | 设置 → 关于 |

两条容易踩的规则：

1. **不要自己画遮罩**。画了圆形，落到圆角方形桌面（MIUI / 多数国产 ROM）上就会变成
   「圆图套方框」。形状一律交给系统：「圆角方形还是圆形」由 launcher 决定。
2. **改图标后必须卸载重装**。多数 launcher（含 MIUI）会按包名缓存图标，
   `adb install -r` 不会刷新缓存，看着像「没生效」。

> 自适应图标下 launcher 只显示画布中间 72/108 的可见区，这是平台行为；
> 因此原图外圈会被 launcher 遮掉一部分，PNG 本身不做任何裁切。

## 8. 明文与证书

`app/src/main/res/xml/network_security_config.xml` 只对探测地址放行明文：

- `5.5.5.5`（探测地址，必须明文 http，否则拿不到 302）
- `10.0.128.3`（Portal 域另配仅该域生效的信任锚点；自签证书跳过校验由 Go 侧 Portal 专用 HTTPS 客户端完成）

其余域名仍走系统默认策略；不设全局 `usesCleartextTraffic`，
也不全局关闭 TLS 校验（001 §6.2、AGENTS §5.4）。

Android 12+ 另有 `app/src/main/res/xml/data_extraction_rules.xml`：云备份与设备迁移
排除 `root` / `database` / `sharedpref` / `external` 全部私有数据；
配合 `AndroidManifest.xml` 的 `allowBackup=false`，凭据密文、Go 快照（含 Cookie）、
课表展示镜像与电费缓存都不参与备份 / 迁移（002 §5.3）。

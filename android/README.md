# Android 构建说明

> 状态：M3 骨架可安装可启动（`adb install` 后 App 能打开三 tab）。
> 契约以 `docs/002` 为准，本文件只说「怎么把 APK 造出来」。

## 1. 前置工具

| 组件 | 版本 | 说明 |
|---|---|---|
| JDK | **17** | AGP 8.7 要求 17；JDK 21+ 不保证可直接用 |
| Android SDK Platform | `android-34` | `compileSdk = 34` |
| Android Build-Tools | `34.0.0` | |
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

## 3. 构建 APK

```bash
cd android
./gradlew :app:assembleDebug        # 产物：app/build/outputs/apk/debug/app-debug.apk
./gradlew :app:assembleRelease      # 一期用 debug 签名，只为能直接装真机
```

装到真机：

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell am start -n com.github.echoping07.kemaolife/.MainActivity
```

## 4. 已知构建环境坑

| 现象 | 原因 | 处理 |
|---|---|---|
| `Your project path contains non-ASCII characters` | AGP 默认拒绝非 ASCII 工程路径 | `gradle.properties` 里已设 `android.overridePathCheck=true`；实测可构建 |
| `java.io.IOException: 文件名、目录名或卷标语法不正确` | `local.properties` 里 `sdk.dir` 用了转义反斜杠 | 写成正斜杠：`sdk.dir=D:/android-sdk` |
| `gomobile bind requires golang.org/x/mobile in the current module` | gomobile 要求模块内能解析 `bind` 包 | 见 §1 的 `go install` 与 `go.mod` 依赖 |
| `javac` 报 `编码 GBK 的不可映射字符` | gobind 把 Go 导出函数的注释原样写进 `.java`，javac 按平台默认编码读 | **导出符号的注释只用 ASCII**（`bridge.go` 的 `Init` / `Call` 已注明） |
| 启动即闪退 `ClassNotFoundException: ...MainActivity` | Kotlin 文件的 `package` 与所在目录不一致（`MainActivity.kt` 在根包目录却写了 `.ui` 包） | Kotlin 的 `package` 必须与目录一致：`MainActivity.kt` 在 `.../kemaolife/` 下就得是 `package com.github.echoping07.kemaolife` |
| 所有时间戳都带 `Z`、比实际差 8 小时 | Go 在 Android 上找不到 zoneinfo，`time.Local` 退化为 UTC | 见 §5 |
| 把工程放在中文路径下构建 | Windows + AGP 对非 ASCII 路径仍有零散问题 | 建一个 ASCII junction：`mklink /J C:\kmbuild D:\<中文路径>` |

## 5. 时区必须由 Kotlin 传给 Go

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

## 6. 应用图标

图标资源在 `res/mipmap-*/`，由同一张正方形原图派生，**不做裁切、不做放大**。
真正决定观感的是自适应图标的可见区规则，不是 PNG 本身。

| 资源 | 做法 | 谁在用 |
|---|---|---|
| `ic_launcher.xml`（`mipmap-anydpi-v26`） | 自适应图标：背景纯色 + 前景 | Android 8.0+ |
| `ic_launcher_foreground.png` | 原图缩到 **72/108** 的可见区居中，**不画任何遮罩** | 同上 |
| `ic_launcher.png` | 整张原图方形 | API 24/25、不支持自适应图标时 |
| `ic_launcher_round.png` | 整张原图 + 内切圆 | 仅类原生 / 圆形屏 launcher |

三条容易踩的规则：

1. **前景不要自己画遮罩**。画了圆形，落到圆角方形桌面（MIUI / 多数国产 ROM）上就会变成
   「圆图套方框」。形状一律交给系统：「圆角方形还是圆形」由 launcher 决定。
2. **前景只占中间 72/108**。自适应图标的安全区就是中间 72dp（四边各留 18dp），
   把整张图铺满 108dp 会让主体落到可见区之外，桌面上只剩一小团。
3. **改图标后必须卸载重装**。多数 launcher（含 MIUI）会按包名缓存图标，
   `adb install -r` 不会刷新缓存，看着像「没生效」。

> `ic_launcher_round.png` 只服务圆形屏 / 类原生桌面；方形桌面不会读它，
> 所以它里面画内切圆是安全的，不会影响 MIUI 上的显示。

## 7. 明文与证书

`app/src/main/res/xml/network_security_config.xml` 只对两个地址做例外：

- `5.5.5.5`（探测地址，必须明文 http，否则拿不到 302）
- `10.0.128.3`（Portal，自签证书）

其余域名仍走系统默认策略；不设全局 `usesCleartextTraffic`，
也不全局关闭 TLS 校验（001 §6.2、AGENTS §5.4）。

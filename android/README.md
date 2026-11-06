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
| 把工程放在中文路径下构建 | Windows + AGP 对非 ASCII 路径仍有零散问题 | 建一个 ASCII junction：`mklink /J C:\kmbuild D:\<中文路径>` |

## 5. 明文与证书

`app/src/main/res/xml/network_security_config.xml` 只对两个地址做例外：

- `5.5.5.5`（探测地址，必须明文 http，否则拿不到 302）
- `10.0.128.3`（Portal，自签证书）

其余域名仍走系统默认策略；不设全局 `usesCleartextTraffic`，
也不全局关闭 TLS 校验（001 §6.2、AGENTS §5.4）。

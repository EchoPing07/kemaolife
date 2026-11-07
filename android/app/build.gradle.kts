plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}

android {
    namespace = "com.github.echoping07.kemaolife"
    compileSdk = 34

    defaultConfig {
        applicationId = "com.github.echoping07.kemaolife"
        minSdk = 24
        targetSdk = 34
        // release.yml 从 vX.Y.Z tag 计算 versionCode 并用 -PversionCode 注入；
        // 本地构建不给该参数时退回 1，仅用于调试，不保证覆盖安装。
        versionCode = (project.findProperty("versionCode") as String?)?.toIntOrNull() ?: 1
        // 与 internal/version.Version 保持一致；发版只改 Go 侧唯一来源并同步这里。
        versionName = "0.4.1"
    }

    buildFeatures {
        compose = true
    }

    signingConfigs {
        // 分发必须跨 release 复用同一把私钥，否则 Android 拒绝覆盖安装，
        // 而卸载会清掉 Keystore 凭据与应用私有目录快照。
        // 四个值都来自 GitHub Actions Secrets，经环境变量注入；本地不写任何口令/路径。
        val keystorePath = System.getenv("KEMAOLIFE_KEYSTORE_FILE")
        val ksPassword = System.getenv("ANDROID_KEYSTORE_PASSWORD")
        val ksAlias = System.getenv("ANDROID_KEY_ALIAS")
        val ksKeyPassword = System.getenv("ANDROID_KEY_PASSWORD")
        if (keystorePath != null && ksPassword != null && ksAlias != null && ksKeyPassword != null) {
            create("release") {
                storeFile = file(keystorePath)
                storePassword = ksPassword
                keyAlias = ksAlias
                keyPassword = ksKeyPassword
                // minSdk 24 不需要 v1（JAR signing）；v2 + v3 覆盖全部受支持设备。
                enableV1Signing = false
                enableV2Signing = true
                enableV3Signing = true
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            // 本地未注入发布密钥时回落 debug 签名，仅供开发自测；
            // release.yml 会断言四个 Secrets 齐全，不会把 debug 签名当 Release 资产。
            signingConfig = signingConfigs.findByName("release")
                ?: signingConfigs.getByName("debug")
        }
    }

    splits {
        // 按 ABI 出 4 个 APK：gomobile 编出的 libgojni.so 是四份最大的体积来源，
        // 手机侧载只需下载自己架构的那一个包。
        abi {
            isEnable = true
            reset()
            include("armeabi-v7a", "arm64-v8a", "x86", "x86_64")
            isUniversalApk = false
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    testOptions {
        unitTests.isReturnDefaultValues = true
    }

    packaging {
        jniLibs {
            useLegacyPackaging = false
        }
    }
}

dependencies {
    // gomobile bind 产出的门面（cmd/kemaolife-mobile → android/app/libs/kemaolife.aar）。
    // 重新生成：gomobile bind -target=android -androidapi 24 -javapkg ...（见 android/README.md §2）。
    implementation(files("libs/kemaolife.aar"))

    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.activity:activity-compose:1.9.3")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.7")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.8.7")
    implementation("androidx.compose.material3:material3:1.3.1")
    implementation("androidx.compose.ui:ui:1.7.6")
    implementation("androidx.compose.ui:ui-tooling-preview:1.7.6")
    debugImplementation("androidx.compose.ui:ui-tooling:1.7.6")

    // 单元测试只碰「信封 → DTO 投影」这一段（不涉 Android framework）。
    // org.json 在 JVM 上需要单独的实现包（Android 里是系统自带）。
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.json:json:20240303")
}

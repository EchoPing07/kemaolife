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
        versionCode = 1
        versionName = "0.1.0"
    }

    buildFeatures {
        compose = true
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            // 用 debug 签名，保证 release 变体也能直接 adb install 到真机（一期不做分发）。
            signingConfig = signingConfigs.getByName("debug")
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
        // gomobile 的四个 ABI 都会打进包；不裁剪，避免在未知机型上缺 so。
        jniLibs {
            useLegacyPackaging = false
        }
    }
}

dependencies {
    // gomobile bind 产出的门面（cmd/kemaolife-mobile → android/app/libs/kemaolife.aar）。
    // 重新生成：make -C android aar（或见 android/README.md）。
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

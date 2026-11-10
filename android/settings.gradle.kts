pluginManagement {
    repositories {
        // 优先国内镜像：google() / mavenCentral() 在本机网络下经常握手超时。
        // 镜像缺包时由 gradlePluginPortal() 兜底。
        maven("https://maven.aliyun.com/repository/gradle-plugin")
        maven("https://maven.aliyun.com/repository/google")
        maven("https://maven.aliyun.com/repository/public")
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositories {
        maven("https://maven.aliyun.com/repository/google")
        maven("https://maven.aliyun.com/repository/public")
        google()
        mavenCentral()
        // gomobile 产出的 AAR 以本地文件形式引用，不走仓库。
    }
}

rootProject.name = "kemaolife"
include(":app")

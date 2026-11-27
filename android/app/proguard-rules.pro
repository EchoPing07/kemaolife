# gomobile bind 生成的 JNI 桥接层（kemaolife.aar 内置 proguard.txt 的显式兜底）：
# libgojni.so 通过 JNI FindClass / GetMethodID / GetStaticMethodID 反向调用这些类，
# 混淆或裁剪会导致运行期 UnsatisfiedLinkError / NoSuchMethodError。
-keep class go.** { *; }
-keep class com.github.echoping07.kemaolife.bridge.** { *; }

# AndroidViewModel 由 AndroidViewModelFactory 通过反射调用 (Application) 构造器。
-keepclassmembers class * extends androidx.lifecycle.AndroidViewModel {
    public <init>(android.app.Application);
}

# 预设主题枚举名直接持久化在 SharedPreferences（theme_color）中。
-keep enum com.github.echoping07.kemaolife.ui.theme.AppThemeColor { *; }

# 保留最小行号信息以便定位崩溃，不暴露源码文件名以外的本地路径。
-keepattributes LineNumberTable,SourceFile
-renamesourcefileattribute SourceFile

package com.github.echoping07.kemaolife.ui

/** 外观页展示文案；仅呈现层，不参与主题偏好的持久化键或业务逻辑。 */
internal object AppearanceText {
    const val ColorMode = "颜色模式"
    const val ModeSystem = "系统"
    const val ModeLight = "浅色"
    const val ModeDark = "深色"

    const val DisplayAndColor = "显示与色彩"
    const val DynamicColor = "莫奈取色"
    const val DynamicColorEnabledSummary = "正在使用系统壁纸生成的配色"
    const val DynamicColorDisabledSummary = "开启后优先使用系统壁纸配色"
    const val DynamicColorUnavailable = "当前设备不可用，需要 Android 12 或更高版本"

    const val Amoled = "AMOLED深色模式"
    const val AmoledActiveSummary = "深色背景使用纯黑，保留清晰的色彩层次"
    const val AmoledInactiveSummary = "可随时开启，纯黑背景仅在深色界面生效"

    const val ContinuousCorners = "连续曲率圆角"
    const val ContinuousCornersEnabledSummary = "卡片与按钮使用连续曲率圆角"
    const val ContinuousCornersDisabledSummary = "当前使用标准圆弧圆角，开启后全 App 改为连续曲率"

    const val FloatingBottomBar = "悬浮底栏"
    const val FloatingBottomBarEnabledSummary = "使用胶囊形悬浮底栏，页面内容透出到底边"
    const val FloatingBottomBarDisabledSummary = "使用贴底导航栏，页面内容与底栏上下分区"

    const val BottomBarLabels = "底栏文字标签"
    const val BottomBarLabelsEnabledSummary = "底栏同时显示图标与文字"
    const val BottomBarLabelsDisabledSummary = "底栏仅显示图标，形态更紧凑"

    const val Preview = "配色预览"
    const val SystemPalette = "系统动态配色"
    const val PreviewBackground = "界面背景"
    const val PreviewAction = "主要按钮"

    const val PresetThemes = "预设主题"
    const val PresetThemesSummary = "AOSP 基础色，点选后立即应用"
    const val PresetThemesDynamicSummary = "莫奈取色已开启；所选预设将在关闭取色后应用"
}

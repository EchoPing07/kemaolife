package com.github.echoping07.kemaolife.data

import java.util.Locale

const val THEME_MODE_SYSTEM = "SYSTEM"
const val THEME_MODE_LIGHT = "LIGHT"
const val THEME_MODE_DARK = "DARK"
const val DEFAULT_THEME_COLOR = "MONOCHROMATIC"
const val THEME_COLOR_CUSTOM = "CUSTOM"

/** 自定义主题色的默认种子色（青绿），与 8 个 AOSP 预设区分明显；始终为不透明 ARGB。 */
const val DEFAULT_CUSTOM_THEME_COLOR: Int = 0xFF26A69A.toInt()

/** 莫奈取色的唯一默认来源；偏好里已显式存过 false 时仍以本机值为准。 */
const val DEFAULT_DYNAMIC_COLOR = true

/**
 * 连续曲率圆角的唯一默认来源：关闭时用标准圆弧。
 *
 * 键不存在时按本默认值处理，不区分新装与升级 —— 老版本没有这个键，
 * 升级后也直接落到普通圆弧，不做偏好迁移（docs/android.md tab3）。
 */
const val DEFAULT_CONTINUOUS_CORNERS = false

/** 悬浮底栏的唯一默认来源：默认开启，可在「设置 → 外观」切回贴底导航栏。 */
const val DEFAULT_FLOATING_BOTTOM_BAR = true

/** 底栏文字标签的唯一默认来源：默认显示图标 + 文字，可在「设置 → 外观」切到纯图标。 */
const val DEFAULT_SHOW_BOTTOM_BAR_LABELS = true

/** 预测性返回手势的唯一默认来源：默认开启，可在设置「个性化」组关闭。 */
const val DEFAULT_PREDICTIVE_BACK = true

/** 本机外观偏好；字段与 AppState 的展示字段保持一一对应。 */
data class ThemeSettings(
    val themeColor: String = DEFAULT_THEME_COLOR,
    val themeMode: String = THEME_MODE_SYSTEM,
    val dynamicColor: Boolean = DEFAULT_DYNAMIC_COLOR,
    val amoled: Boolean = false,
    val continuousCorners: Boolean = DEFAULT_CONTINUOUS_CORNERS,
    val floatingBottomBar: Boolean = DEFAULT_FLOATING_BOTTOM_BAR,
    val showBottomBarLabels: Boolean = DEFAULT_SHOW_BOTTOM_BAR_LABELS,
    val predictiveBackEnabled: Boolean = DEFAULT_PREDICTIVE_BACK,
    val customThemeColor: Int = DEFAULT_CUSTOM_THEME_COLOR,
)

private val supportedThemeColors = setOf(
    // AOSP ThemePicker/ThemesStub 的 8 个基础色 + 用户自定义色；旧值通过 legacyThemeColorAliases 迁移。
    "MONOCHROMATIC",
    "RED",
    "ORANGE",
    "YELLOW",
    "GREEN",
    "BLUE",
    "PURPLE",
    "MAGENTA",
    THEME_COLOR_CUSTOM,
)

private val legacyThemeColorAliases = mapOf(
    "TEAL" to "GREEN",
    "ROSE" to "MAGENTA",
    "INDIGO" to "PURPLE",
    "MONOCHROME" to "MONOCHROMATIC",
)

/** 损坏或未来版本的值回退默认值，避免偏好数据阻塞界面启动。 */
fun normalizeThemeColor(value: String?): String {
    val normalized = value?.trim()?.uppercase() ?: return DEFAULT_THEME_COLOR
    if (normalized in supportedThemeColors) return normalized
    return legacyThemeColorAliases[normalized] ?: DEFAULT_THEME_COLOR
}

fun normalizeThemeMode(value: String?): String = when (value?.trim()?.uppercase()) {
    THEME_MODE_LIGHT -> THEME_MODE_LIGHT
    THEME_MODE_DARK -> THEME_MODE_DARK
    else -> THEME_MODE_SYSTEM
}

/** 颜色模式的唯一判态入口；SYSTEM 始终尊重当前系统深色配置。 */
fun isDarkTheme(themeMode: String, systemDark: Boolean): Boolean = when (normalizeThemeMode(themeMode)) {
    THEME_MODE_LIGHT -> false
    THEME_MODE_DARK -> true
    else -> systemDark
}

/** 自定义主题色强制为不透明 ARGB，避免透明通道污染色板派生。 */
fun normalizeCustomThemeColor(argb: Int): Int = (argb and 0x00FFFFFF) or (0xFF shl 24)

/** 将自定义主题色格式化为大写 `#RRGGBB` 文本，固定使用 `Locale.US`。 */
fun formatThemeColorHex(argb: Int): String =
    String.format(Locale.US, "#%06X", argb and 0x00FFFFFF)

/**
 * 解析用户输入或粘贴的十六进制颜色（支持 `#RRGGBB`、`RRGGBB`、`#RGB`、`RGB`）。
 * 非法长度、非十六进制字符或空白返回 null，不抛异常。
 */
fun parseThemeColorHex(input: String?): Int? {
    val raw = input?.trim()?.removePrefix("#") ?: return null
    val hex = when (raw.length) {
        3 -> buildString(6) {
            raw.forEach { ch ->
                if (!ch.isAsciiHexDigit()) return null
                append(ch).append(ch)
            }
        }
        6 -> {
            if (!raw.all { it.isAsciiHexDigit() }) return null
            raw
        }
        else -> return null
    }
    return normalizeCustomThemeColor(hex.toInt(16))
}

private fun Char.isAsciiHexDigit(): Boolean =
    this in '0'..'9' || this in 'a'..'f' || this in 'A'..'F'

package com.github.echoping07.kemaolife.data

const val THEME_MODE_SYSTEM = "SYSTEM"
const val THEME_MODE_LIGHT = "LIGHT"
const val THEME_MODE_DARK = "DARK"
const val DEFAULT_THEME_COLOR = "MONOCHROMATIC"

/** 莫奈取色的唯一默认来源；偏好里已显式存过 false 时仍以本机值为准。 */
const val DEFAULT_DYNAMIC_COLOR = true

/** 本机外观偏好；字段与 AppState 的展示字段保持一一对应。 */
data class ThemeSettings(
    val themeColor: String = DEFAULT_THEME_COLOR,
    val themeMode: String = THEME_MODE_SYSTEM,
    val dynamicColor: Boolean = DEFAULT_DYNAMIC_COLOR,
    val amoled: Boolean = false,
)

private val supportedThemeColors = setOf(
    // AOSP ThemePicker/ThemesStub 的 8 个基础色；旧值通过 legacyThemeColorAliases 迁移。
    "MONOCHROMATIC",
    "RED",
    "ORANGE",
    "YELLOW",
    "GREEN",
    "BLUE",
    "PURPLE",
    "MAGENTA",
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

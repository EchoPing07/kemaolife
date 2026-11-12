package com.github.echoping07.kemaolife.data

import android.content.Context

/** 仅保存非敏感的外观偏好，不与 Keystore 中的账号凭据混放。 */
class ThemePreferences(context: Context) {
    private val preferences = context.getSharedPreferences("appearance", Context.MODE_PRIVATE)

    /** 兼容旧调用：只返回主题色。新页面应通过 [loadSettings] 一次读取完整偏好。 */
    fun load(): String = loadSettings().themeColor

    fun loadSettings(): ThemeSettings = ThemeSettings(
        themeColor = normalizeThemeColor(preferences.getString(KEY_THEME, DEFAULT_THEME_COLOR)),
        themeMode = loadMode(),
        dynamicColor = loadDynamicColor(),
        amoled = loadAmoled(),
    )

    fun loadMode(): String = normalizeThemeMode(
        preferences.getString(KEY_THEME_MODE, THEME_MODE_SYSTEM),
    )

    // 键不存在（新安装 / 清数据）时取全局默认；已显式存过 false 时尊重用户选择。
    fun loadDynamicColor(): Boolean = preferences.getBoolean(KEY_DYNAMIC_COLOR, DEFAULT_DYNAMIC_COLOR)

    fun loadAmoled(): Boolean = preferences.getBoolean(KEY_AMOLED, false)

    /** 兼容旧的主题色保存接口。 */
    fun save(theme: String) {
        preferences.edit().putString(KEY_THEME, normalizeThemeColor(theme)).apply()
    }

    fun save(settings: ThemeSettings) {
        preferences.edit()
            .putString(KEY_THEME, normalizeThemeColor(settings.themeColor))
            .putString(KEY_THEME_MODE, normalizeThemeMode(settings.themeMode))
            .putBoolean(KEY_DYNAMIC_COLOR, settings.dynamicColor)
            .putBoolean(KEY_AMOLED, settings.amoled)
            .apply()
    }

    fun saveThemeMode(value: String) {
        preferences.edit().putString(KEY_THEME_MODE, normalizeThemeMode(value)).apply()
    }

    fun saveMode(value: String) = saveThemeMode(value)

    fun saveDynamicColor(enabled: Boolean) {
        preferences.edit().putBoolean(KEY_DYNAMIC_COLOR, enabled).apply()
    }

    fun saveAmoled(enabled: Boolean) {
        preferences.edit().putBoolean(KEY_AMOLED, enabled).apply()
    }

    private companion object {
        const val KEY_THEME = "theme_color"
        const val KEY_THEME_MODE = "theme_mode"
        const val KEY_DYNAMIC_COLOR = "dynamic_color"
        const val KEY_AMOLED = "amoled"
    }
}

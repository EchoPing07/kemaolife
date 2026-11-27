package com.github.echoping07.kemaolife.data

import android.content.Context

/** 仅保存非敏感的外观偏好，不与 Keystore 中的账号凭据混放。 */
class ThemePreferences(context: Context) {
    private val preferences = context.getSharedPreferences("appearance", Context.MODE_PRIVATE)

    /** 一次读取全部 9 项外观偏好；键缺失时回落全局默认，不做旧版本迁移。 */
    fun loadSettings(): ThemeSettings = ThemeSettings(
        themeColor = normalizeThemeColor(preferences.getString(KEY_THEME, DEFAULT_THEME_COLOR)),
        themeMode = normalizeThemeMode(preferences.getString(KEY_THEME_MODE, THEME_MODE_SYSTEM)),
        // 键不存在（新安装 / 清数据）时取全局默认；已显式存过 false 时尊重用户选择。
        dynamicColor = preferences.getBoolean(KEY_DYNAMIC_COLOR, DEFAULT_DYNAMIC_COLOR),
        amoled = preferences.getBoolean(KEY_AMOLED, false),
        continuousCorners = preferences.getBoolean(KEY_CONTINUOUS_CORNERS, DEFAULT_CONTINUOUS_CORNERS),
        floatingBottomBar = preferences.getBoolean(KEY_FLOATING_BOTTOM_BAR, DEFAULT_FLOATING_BOTTOM_BAR),
        showBottomBarLabels = preferences.getBoolean(
            KEY_SHOW_BOTTOM_BAR_LABELS,
            DEFAULT_SHOW_BOTTOM_BAR_LABELS,
        ),
        predictiveBackEnabled = preferences.getBoolean(KEY_PREDICTIVE_BACK, DEFAULT_PREDICTIVE_BACK),
        customThemeColor = normalizeCustomThemeColor(
            preferences.getInt(KEY_CUSTOM_THEME_COLOR, DEFAULT_CUSTOM_THEME_COLOR),
        ),
    )

    /** 一次批量持久化全部外观偏好（单次 `apply()`）。 */
    fun save(settings: ThemeSettings) {
        preferences.edit()
            .putString(KEY_THEME, normalizeThemeColor(settings.themeColor))
            .putInt(KEY_CUSTOM_THEME_COLOR, normalizeCustomThemeColor(settings.customThemeColor))
            .putString(KEY_THEME_MODE, normalizeThemeMode(settings.themeMode))
            .putBoolean(KEY_DYNAMIC_COLOR, settings.dynamicColor)
            .putBoolean(KEY_AMOLED, settings.amoled)
            .putBoolean(KEY_CONTINUOUS_CORNERS, settings.continuousCorners)
            .putBoolean(KEY_FLOATING_BOTTOM_BAR, settings.floatingBottomBar)
            .putBoolean(KEY_SHOW_BOTTOM_BAR_LABELS, settings.showBottomBarLabels)
            .putBoolean(KEY_PREDICTIVE_BACK, settings.predictiveBackEnabled)
            .apply()
    }

    private companion object {
        const val KEY_THEME = "theme_color"
        const val KEY_CUSTOM_THEME_COLOR = "custom_theme_color"
        const val KEY_THEME_MODE = "theme_mode"
        const val KEY_DYNAMIC_COLOR = "dynamic_color"
        const val KEY_AMOLED = "amoled"
        const val KEY_CONTINUOUS_CORNERS = "continuous_corners"
        const val KEY_FLOATING_BOTTOM_BAR = "floating_bottom_bar"
        const val KEY_SHOW_BOTTOM_BAR_LABELS = "show_bottom_bar_labels"
        const val KEY_PREDICTIVE_BACK = "predictive_back_enabled"
    }
}

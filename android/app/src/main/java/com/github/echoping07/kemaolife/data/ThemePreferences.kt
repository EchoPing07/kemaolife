package com.github.echoping07.kemaolife.data

import android.content.Context

/** 仅保存非敏感的外观偏好，不与 Keystore 中的账号凭据混放。 */
class ThemePreferences(context: Context) {
    private val preferences = context.getSharedPreferences("appearance", Context.MODE_PRIVATE)

    fun load(): String = preferences.getString(KEY_THEME, DEFAULT_THEME) ?: DEFAULT_THEME

    fun save(theme: String) {
        preferences.edit().putString(KEY_THEME, theme).apply()
    }

    private companion object {
        const val KEY_THEME = "theme_color"
        const val DEFAULT_THEME = "BLUE"
    }
}

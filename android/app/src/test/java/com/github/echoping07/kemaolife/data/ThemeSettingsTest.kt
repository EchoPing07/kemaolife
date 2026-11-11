package com.github.echoping07.kemaolife.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ThemeSettingsTest {

    @Test
    fun defaultsMatchLegacyAppearance() {
        assertEquals(
            ThemeSettings(themeColor = "MONOCHROMATIC", themeMode = "SYSTEM", dynamicColor = false, amoled = false),
            ThemeSettings(),
        )
    }

    @Test
    fun allEightPresetNamesSurviveValidation() {
        val presets = listOf(
            "MONOCHROMATIC", "RED", "ORANGE", "YELLOW",
            "GREEN", "BLUE", "PURPLE", "MAGENTA",
        )
        presets.forEach { assertEquals(it, normalizeThemeColor(it)) }
        assertEquals(8, presets.size)
    }

    @Test
    fun legacyPresetNamesMapToAospPresets() {
        assertEquals("GREEN", normalizeThemeColor("TEAL"))
        assertEquals("MAGENTA", normalizeThemeColor("ROSE"))
        assertEquals("PURPLE", normalizeThemeColor("INDIGO"))
        assertEquals("MONOCHROMATIC", normalizeThemeColor("MONOCHROME"))
    }

    @Test
    fun invalidThemeValuesFallBackWithoutThrowing() {
        listOf(null, "", "  ", "UNKNOWN").forEach {
            assertEquals(DEFAULT_THEME_COLOR, normalizeThemeColor(it))
        }
        assertEquals("MAGENTA", normalizeThemeColor(" rose "))
    }

    @Test
    fun invalidModeFallsBackToSystem() {
        listOf(null, "", "AUTO", "DARK_BLUE").forEach {
            assertEquals(THEME_MODE_SYSTEM, normalizeThemeMode(it))
        }
        assertEquals(THEME_MODE_LIGHT, normalizeThemeMode(" light "))
        assertEquals(THEME_MODE_DARK, normalizeThemeMode("dark"))
        assertEquals(THEME_MODE_SYSTEM, normalizeThemeMode("SYSTEM"))
    }

    @Test
    fun explicitModesIgnoreSystemAndSystemModeFollowsIt() {
        assertFalse(isDarkTheme(THEME_MODE_LIGHT, systemDark = true))
        assertFalse(isDarkTheme(THEME_MODE_LIGHT, systemDark = false))
        assertTrue(isDarkTheme(THEME_MODE_DARK, systemDark = false))
        assertTrue(isDarkTheme(THEME_MODE_DARK, systemDark = true))
        assertTrue(isDarkTheme(THEME_MODE_SYSTEM, systemDark = true))
        assertFalse(isDarkTheme(THEME_MODE_SYSTEM, systemDark = false))
    }
}

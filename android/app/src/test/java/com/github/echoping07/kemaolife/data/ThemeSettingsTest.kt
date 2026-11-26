package com.github.echoping07.kemaolife.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ThemeSettingsTest {

    @Test
    fun defaultAppearanceValues() {
        assertEquals(
            ThemeSettings(
                themeColor = "MONOCHROMATIC",
                themeMode = "SYSTEM",
                dynamicColor = true,
                amoled = false,
                continuousCorners = false,
                floatingBottomBar = true,
                showBottomBarLabels = true,
                predictiveBackEnabled = true,
                customThemeColor = DEFAULT_CUSTOM_THEME_COLOR,
            ),
            ThemeSettings(),
        )
    }

    @Test
    fun floatingBottomBarEnabledByDefaultAndCanBeDisabled() {
        assertTrue(DEFAULT_FLOATING_BOTTOM_BAR)
        assertTrue(ThemeSettings().floatingBottomBar)
        assertFalse(ThemeSettings(floatingBottomBar = false).floatingBottomBar)
    }

    @Test
    fun bottomBarLabelsEnabledByDefaultAndCanBeDisabled() {
        assertTrue(DEFAULT_SHOW_BOTTOM_BAR_LABELS)
        assertTrue(ThemeSettings().showBottomBarLabels)
        assertFalse(ThemeSettings(showBottomBarLabels = false).showBottomBarLabels)
    }

    @Test
    fun predictiveBackEnabledByDefaultAndCanBeDisabled() {
        assertTrue(DEFAULT_PREDICTIVE_BACK)
        assertTrue(ThemeSettings().predictiveBackEnabled)
        assertFalse(ThemeSettings(predictiveBackEnabled = false).predictiveBackEnabled)
    }

    @Test
    fun continuousCornersIsDisabledByDefault() {
        assertFalse(DEFAULT_CONTINUOUS_CORNERS)
        assertFalse(ThemeSettings().continuousCorners)
    }

    @Test
    fun continuousCornersCanBeTurnedOnExplicitly() {
        assertTrue(ThemeSettings(continuousCorners = true).continuousCorners)
    }

    @Test
    fun dynamicColorIsEnabledByDefault() {
        assertTrue(DEFAULT_DYNAMIC_COLOR)
        assertTrue(ThemeSettings().dynamicColor)
    }

    @Test
    fun dynamicColorCanBeTurnedOffExplicitly() {
        assertFalse(ThemeSettings(dynamicColor = false).dynamicColor)
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

    @Test
    fun customThemeColorSurvivesNormalization() {
        assertEquals(THEME_COLOR_CUSTOM, normalizeThemeColor("CUSTOM"))
        assertEquals(THEME_COLOR_CUSTOM, normalizeThemeColor(" custom "))
    }

    @Test
    fun normalizeCustomThemeColorAlwaysForcesOpaqueAlpha() {
        assertEquals(0xFF26A69A.toInt(), normalizeCustomThemeColor(0x0026A69A))
        assertEquals(0xFF123456.toInt(), normalizeCustomThemeColor(0x40123456))
        assertEquals(0xFF000000.toInt(), normalizeCustomThemeColor(0x00000000))
    }

    @Test
    fun formatAndParseThemeColorHexRoundTrip() {
        assertEquals("#26A69A", formatThemeColorHex(0xFF26A69A.toInt()))
        assertEquals("#00050A", formatThemeColorHex(0x1200050A))
        assertEquals(0xFF26A69A.toInt(), parseThemeColorHex("#26A69A"))
        assertEquals(0xFF26A69A.toInt(), parseThemeColorHex("  26a69a "))
        assertEquals(0xFFAABBCC.toInt(), parseThemeColorHex("#abc"))
        assertEquals(0xFFFF0088.toInt(), parseThemeColorHex("f08"))
    }

    @Test
    fun parseThemeColorHexRejectsInvalidInputs() {
        listOf(null, "", "   ", "#", "#12", "#1234", "#12345", "#1234567", "#FF26A69A", "#GGGGGG", "xyz")
            .forEach { assertNull("expected null for <$it>", parseThemeColorHex(it)) }
    }
}

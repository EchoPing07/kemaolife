package com.github.echoping07.kemaolife.ui.theme

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.abs

class CustomThemePaletteTest {

    @Test
    fun hsvAndHslConversionsRoundTripRepresentativeColors() {
        val samples = listOf(
            0xFFFF0000.toInt(),
            0xFF00FF00.toInt(),
            0xFF0000FF.toInt(),
            0xFF26A69A.toInt(),
            0xFFE11D48.toInt(),
            0xFFD97706.toInt(),
            0xFF808080.toInt(),
            0xFF000000.toInt(),
            0xFFFFFFFF.toInt(),
        )
        samples.forEach { argb ->
            val hsl = CustomThemePalette.rgbToHsl(argb)
            assertColorClose(argb, CustomThemePalette.hslToRgb(hsl.hue, hsl.saturation, hsl.lightness))

            val hsv = CustomThemePalette.rgbToHsv(argb)
            assertColorClose(argb, CustomThemePalette.hsvToRgb(hsv.hue, hsv.saturation, hsv.value))
        }
    }

    @Test
    fun generatedPalettesMeetReadableContrastAcrossExtremeSeeds() {
        val seeds = listOf(
            0xFFFFFF00.toInt(), // 纯黄（高明度难色）
            0xFF00FFFF.toInt(), // 纯青
            0xFF0000FF.toInt(), // 纯蓝（低明度）
            0xFFFF0000.toInt(), // 纯红
            0xFF26A69A.toInt(), // 默认青绿
            0xFF0A192F.toInt(), // 极深藏青
            0xFFFFE4EC.toInt(), // 极浅樱粉
            0xFF000000.toInt(), // 纯黑
            0xFFFFFFFF.toInt(), // 纯白
            0xFF808080.toInt(), // 中灰
        )
        seeds.forEach { seed ->
            val light = CustomThemePalette.generate(seed, dark = false)
            assertTrue(
                "light primary/onPrimary contrast for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(light.primary, light.onPrimary) >= 4.5,
            )
            assertTrue(
                "light primaryContainer/onPrimaryContainer contrast for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(light.primaryContainer, light.onPrimaryContainer) >= 7.0,
            )
            assertTrue(
                "light secondary/onSecondary contrast for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(light.secondary, light.onSecondary) >= 4.5,
            )
            assertTrue(
                "light tertiary/onTertiary contrast for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(light.tertiary, light.onTertiary) >= 4.5,
            )

            val dark = CustomThemePalette.generate(seed, dark = true)
            assertTrue(
                "dark primary/onPrimary contrast for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(dark.primary, dark.onPrimary) >= 4.5,
            )
            assertTrue(
                "dark primaryContainer/onPrimaryContainer contrast for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(dark.primaryContainer, dark.onPrimaryContainer) >= 4.5,
            )
            assertTrue(
                "dark primary visible on dark surface for ${seed.toHex()}",
                CustomThemePalette.contrastRatio(dark.primary, 0xFF0F1113.toInt()) >= 7.0,
            )
        }
    }

    @Test
    fun chromaticSeedPreservesHueAndAchromaticSeedStaysNeutral() {
        val tealLight = CustomThemePalette.generate(0xFF26A69A.toInt(), dark = false)
        val seedHue = CustomThemePalette.rgbToHsl(0xFF26A69A.toInt()).hue
        val primaryHue = CustomThemePalette.rgbToHsl(tealLight.primary).hue
        assertTrue("hue drift too large: $seedHue vs $primaryHue", hueDistance(seedHue, primaryHue) < 8f)

        val grayLight = CustomThemePalette.generate(0xFF777777.toInt(), dark = false)
        val grayPrimaryHsl = CustomThemePalette.rgbToHsl(grayLight.primary)
        assertTrue("gray seed should produce neutral primary", grayPrimaryHsl.saturation < 0.05f)
    }

    @Test
    fun lightPageSurfaceKeepsWhiteCardDistinct() {
        val white = 0xFFFFFFFF.toInt()
        listOf(0xFF26A69A.toInt(), 0xFFFFFF00.toInt(), 0xFF0000FF.toInt(), 0xFF000000.toInt()).forEach { seed ->
            val palette = CustomThemePalette.generate(seed, dark = false)
            val ratio = CustomThemePalette.contrastRatio(white, palette.surfacePage)
            assertTrue("white card should stand out from page surface (got $ratio)", ratio in 1.12..1.45)
        }
    }

    @Test
    fun transparentAlphaInSeedIsIgnored() {
        assertEquals(
            CustomThemePalette.generate(0xFF26A69A.toInt(), dark = false),
            CustomThemePalette.generate(0x0026A69A, dark = false),
        )
    }

    private fun assertColorClose(expected: Int, actual: Int) {
        assertEquals(0xFF, (actual ushr 24) and 0xFF)
        assertTrue(abs(((expected ushr 16) and 0xFF) - ((actual ushr 16) and 0xFF)) <= 1)
        assertTrue(abs(((expected ushr 8) and 0xFF) - ((actual ushr 8) and 0xFF)) <= 1)
        assertTrue(abs((expected and 0xFF) - (actual and 0xFF)) <= 1)
    }

    private fun hueDistance(a: Float, b: Float): Float {
        val diff = abs(a - b) % 360f
        return if (diff > 180f) 360f - diff else diff
    }

    private fun Int.toHex(): String = "#%06X".format(this and 0x00FFFFFF)
}

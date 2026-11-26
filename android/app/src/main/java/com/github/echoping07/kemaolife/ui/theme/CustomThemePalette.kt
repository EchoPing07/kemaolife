package com.github.echoping07.kemaolife.ui.theme

import com.github.echoping07.kemaolife.data.normalizeCustomThemeColor
import kotlin.math.abs
import kotlin.math.pow
import kotlin.math.roundToInt

internal data class HslColor(
    val hue: Float,
    val saturation: Float,
    val lightness: Float,
)

internal data class HsvColor(
    val hue: Float,
    val saturation: Float,
    val value: Float,
)

/**
 * 自定义主题色派生出的色板（纯 ARGB Int，不依赖 Android framework 以便 JVM 单测覆盖）。
 *
 * 与 8 个 AOSP 预设保持同一套页面明暗层级：浅色页面底色固定 HSL 明度 0.91，
 * 白卡（pure white）作为唯一亮面；各强调色角色按 CIE L* 相对亮度（Tone）二分求解，
 * 保证任意色相（含高明度黄 / 绿与极暗蓝 / 紫）在浅色与深色下都满足可读对比度。
 */
internal data class CustomPaletteColors(
    val primary: Int,
    val onPrimary: Int,
    val primaryContainer: Int,
    val onPrimaryContainer: Int,
    val secondary: Int,
    val onSecondary: Int,
    val secondaryContainer: Int,
    val onSecondaryContainer: Int,
    val tertiary: Int,
    val onTertiary: Int,
    val tertiaryContainer: Int,
    val onTertiaryContainer: Int,
    val surfacePage: Int,
    val surfaceHigh: Int,
    val surfaceHighest: Int,
)

internal object CustomThemePalette {
    fun rgbToHsl(argb: Int): HslColor {
        val r = ((argb ushr 16) and 0xFF) / 255f
        val g = ((argb ushr 8) and 0xFF) / 255f
        val b = (argb and 0xFF) / 255f
        val max = maxOf(r, g, b)
        val min = minOf(r, g, b)
        val delta = max - min
        val l = (max + min) * 0.5f
        if (delta == 0f) return HslColor(0f, 0f, l)

        val denom = 1f - abs(2f * l - 1f)
        val s = if (denom == 0f) 0f else (delta / denom).coerceIn(0f, 1f)
        val hue = computeHue(r, g, b, max, delta)
        return HslColor(hue, s, l)
    }

    fun hslToRgb(hue: Float, saturation: Float, lightness: Float): Int {
        val h = normalizeHue(hue)
        val s = saturation.coerceIn(0f, 1f)
        val l = lightness.coerceIn(0f, 1f)
        val c = (1f - abs(2f * l - 1f)) * s
        val x = c * (1f - abs((h / 60f) % 2f - 1f))
        val m = l - c * 0.5f
        return packSextantRgb(h, c, x, m)
    }

    fun rgbToHsv(argb: Int): HsvColor {
        val r = ((argb ushr 16) and 0xFF) / 255f
        val g = ((argb ushr 8) and 0xFF) / 255f
        val b = (argb and 0xFF) / 255f
        val max = maxOf(r, g, b)
        val min = minOf(r, g, b)
        val delta = max - min
        val s = if (max == 0f) 0f else (delta / max).coerceIn(0f, 1f)
        val hue = if (delta == 0f) 0f else computeHue(r, g, b, max, delta)
        return HsvColor(hue, s, max)
    }

    fun hsvToRgb(hue: Float, saturation: Float, value: Float): Int {
        val h = normalizeHue(hue)
        val s = saturation.coerceIn(0f, 1f)
        val v = value.coerceIn(0f, 1f)
        val c = v * s
        val x = c * (1f - abs((h / 60f) % 2f - 1f))
        val m = v - c
        return packSextantRgb(h, c, x, m)
    }

    /** WCAG 2.1 sRGB 相对亮度（0.0 = 纯黑，1.0 = 纯白）。 */
    fun relativeLuminance(argb: Int): Double {
        val r = srgbChannelToLinear(((argb ushr 16) and 0xFF) / 255.0)
        val g = srgbChannelToLinear(((argb ushr 8) and 0xFF) / 255.0)
        val b = srgbChannelToLinear((argb and 0xFF) / 255.0)
        return 0.2126 * r + 0.7152 * g + 0.0722 * b
    }

    /** WCAG 2.1 对比度比值（1.0 ~ 21.0）。 */
    fun contrastRatio(firstArgb: Int, secondArgb: Int): Double {
        val l1 = relativeLuminance(firstArgb)
        val l2 = relativeLuminance(secondArgb)
        val lighter = maxOf(l1, l2)
        val darker = minOf(l1, l2)
        return (lighter + 0.05) / (darker + 0.05)
    }

    /** Material 3 Tone（CIE L* 0..100）对应的目标相对亮度 Y。 */
    fun toneToLuminance(tone: Float): Double {
        val t = tone.coerceIn(0f, 100f).toDouble()
        return if (t <= 8.0) {
            t / 903.2962962962963
        } else {
            val f = (t + 16.0) / 116.0
            f * f * f
        }
    }

    /**
     * 固定色相与饱和度，通过二分搜索 HSL 明度使输出命中目标 Tone 的相对亮度。
     * 相对亮度对 HSL lightness 严格单调递增，16 次迭代即可稳定收敛到 8-bit 精度。
     */
    fun colorAtTone(hue: Float, saturation: Float, tone: Float): Int {
        if (tone <= 0f) return 0xFF000000.toInt()
        if (tone >= 100f) return 0xFFFFFFFF.toInt()
        val targetY = toneToLuminance(tone)
        var low = 0f
        var high = 1f
        repeat(16) {
            val mid = (low + high) * 0.5f
            val y = relativeLuminance(hslToRgb(hue, saturation, mid))
            if (y < targetY) low = mid else high = mid
        }
        return hslToRgb(hue, saturation, (low + high) * 0.5f)
    }

    /** 从任意用户种子色派生完整的 M3 强调色与浅色页面表面色阶。 */
    fun generate(seedArgb: Int, dark: Boolean): CustomPaletteColors {
        val normalized = normalizeCustomThemeColor(seedArgb)
        val hsl = rgbToHsl(normalized)
        // 近灰 / 极黑 / 极白缺乏稳定色相，回落单色中性策略。
        val isAchromatic = hsl.saturation < 0.04f || hsl.lightness < 0.03f || hsl.lightness > 0.97f

        val primaryHue = if (isAchromatic) 210f else hsl.hue
        val primarySat = if (isAchromatic) 0f else hsl.saturation.coerceIn(0.18f, 0.88f)
        val secondaryHue = primaryHue
        val secondarySat = if (isAchromatic) 0f else (hsl.saturation * 0.36f).coerceIn(0.10f, 0.34f)
        val tertiaryHue = if (isAchromatic) 210f else (hsl.hue + 60f) % 360f
        val tertiarySat = if (isAchromatic) 0f else (hsl.saturation * 0.56f).coerceIn(0.18f, 0.56f)

        val surfaceHue = primaryHue
        val surfaceSat = if (isAchromatic) 0.02f else (hsl.saturation * 0.16f).coerceIn(0.04f, 0.12f)
        val surfacePage = hslToRgb(surfaceHue, surfaceSat, 0.91f)
        val surfaceHigh = hslToRgb(surfaceHue, surfaceSat, 0.865f)
        val surfaceHighest = hslToRgb(surfaceHue, surfaceSat, 0.825f)

        return if (dark) {
            val primaryToneSat = (primarySat * 0.85f).coerceAtMost(0.78f)
            val tertiaryToneSat = (tertiarySat * 0.85f).coerceAtMost(0.70f)
            CustomPaletteColors(
                primary = colorAtTone(primaryHue, primaryToneSat, 80f),
                onPrimary = colorAtTone(primaryHue, primarySat, 20f),
                primaryContainer = colorAtTone(primaryHue, primarySat, 30f),
                onPrimaryContainer = colorAtTone(primaryHue, (primarySat * 0.75f).coerceAtMost(0.65f), 90f),
                secondary = colorAtTone(secondaryHue, secondarySat, 80f),
                onSecondary = colorAtTone(secondaryHue, secondarySat, 20f),
                secondaryContainer = colorAtTone(secondaryHue, secondarySat, 30f),
                onSecondaryContainer = colorAtTone(secondaryHue, secondarySat, 90f),
                tertiary = colorAtTone(tertiaryHue, tertiaryToneSat, 80f),
                onTertiary = colorAtTone(tertiaryHue, tertiarySat, 20f),
                tertiaryContainer = colorAtTone(tertiaryHue, tertiarySat, 30f),
                onTertiaryContainer = colorAtTone(tertiaryHue, (tertiarySat * 0.75f).coerceAtMost(0.60f), 90f),
                surfacePage = surfacePage,
                surfaceHigh = surfaceHigh,
                surfaceHighest = surfaceHighest,
            )
        } else {
            val containerSat = (primarySat * 0.78f).coerceAtMost(0.66f)
            val tertiaryContainerSat = (tertiarySat * 0.78f).coerceAtMost(0.60f)
            CustomPaletteColors(
                primary = colorAtTone(primaryHue, primarySat, 42f),
                onPrimary = 0xFFFFFFFF.toInt(),
                primaryContainer = colorAtTone(primaryHue, containerSat, 90f),
                onPrimaryContainer = colorAtTone(primaryHue, primarySat, 12f),
                secondary = colorAtTone(secondaryHue, secondarySat, 42f),
                onSecondary = 0xFFFFFFFF.toInt(),
                secondaryContainer = colorAtTone(secondaryHue, secondarySat, 90f),
                onSecondaryContainer = colorAtTone(secondaryHue, secondarySat, 12f),
                tertiary = colorAtTone(tertiaryHue, tertiarySat, 42f),
                onTertiary = 0xFFFFFFFF.toInt(),
                tertiaryContainer = colorAtTone(tertiaryHue, tertiaryContainerSat, 90f),
                onTertiaryContainer = colorAtTone(tertiaryHue, tertiarySat, 12f),
                surfacePage = surfacePage,
                surfaceHigh = surfaceHigh,
                surfaceHighest = surfaceHighest,
            )
        }
    }

    private fun computeHue(r: Float, g: Float, b: Float, max: Float, delta: Float): Float {
        val raw = when (max) {
            r -> ((g - b) / delta) % 6f
            g -> (b - r) / delta + 2f
            else -> (r - g) / delta + 4f
        } * 60f
        return normalizeHue(raw)
    }

    private fun normalizeHue(hue: Float): Float = ((hue % 360f) + 360f) % 360f

    private fun packSextantRgb(h: Float, c: Float, x: Float, m: Float): Int {
        val (r1, g1, b1) = when {
            h < 60f -> Triple(c, x, 0f)
            h < 120f -> Triple(x, c, 0f)
            h < 180f -> Triple(0f, c, x)
            h < 240f -> Triple(0f, x, c)
            h < 300f -> Triple(x, 0f, c)
            else -> Triple(c, 0f, x)
        }
        val r = ((r1 + m) * 255f).roundToInt().coerceIn(0, 255)
        val g = ((g1 + m) * 255f).roundToInt().coerceIn(0, 255)
        val b = ((b1 + m) * 255f).roundToInt().coerceIn(0, 255)
        return (0xFF shl 24) or (r shl 16) or (g shl 8) or b
    }

    private fun srgbChannelToLinear(channel: Double): Double =
        if (channel <= 0.04045) channel / 12.92 else ((channel + 0.055) / 1.055).pow(2.4)
}

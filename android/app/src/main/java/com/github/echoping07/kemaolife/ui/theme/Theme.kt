package com.github.echoping07.kemaolife.ui.theme

import android.os.Build
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.ui.animationsEnabled
import androidx.core.graphics.ColorUtils
import com.github.echoping07.kemaolife.data.DEFAULT_DYNAMIC_COLOR
import com.github.echoping07.kemaolife.data.THEME_MODE_DARK
import com.github.echoping07.kemaolife.data.THEME_MODE_LIGHT
import com.github.echoping07.kemaolife.data.isDarkTheme
import com.github.echoping07.kemaolife.data.normalizeThemeColor

/**
 * AOSP ThemePicker/ThemesStub 的 8 个基础色取自
 * `packages/apps/ThemePicker/themes/res/values/colors.xml` 的 `color_secondary_*`，
 * 名称取自同目录 `strings.xml`。颜色值本身只是生成配色方案的 seed，
 * 实际 primary / container / on-* 仍按 M3 色角色派生；预览不直接展示 seed。
 */
internal enum class AppThemeColor(
    val label: String,
    val color: Color,
    val secondarySeed: Color,
    val tertiarySeed: Color,
) {
    MONOCHROMATIC("单色", Color(0xFFFFFF00), Color(0xFF9A9A9A), Color(0xFFB0B0B0)),
    RED("红色", Color(0xFFFFB2B5), Color(0xFFD98F93), Color(0xFFB7779C)),
    ORANGE("橙色", Color(0xFFFFB868), Color(0xFFD99A56), Color(0xFFB78C77)),
    YELLOW("黄色", Color(0xFFE9C44A), Color(0xFFC7A73F), Color(0xFFA9AE66)),
    GREEN("绿色", Color(0xFFBAF293), Color(0xFF8FCE71), Color(0xFF70AE86)),
    BLUE("蓝色", Color(0xFF96CBFF), Color(0xFF71A9DB), Color(0xFF7C8FC4)),
    PURPLE("紫色", Color(0xFFCBBFFF), Color(0xFFA99CDC), Color(0xFF8F83C2)),
    MAGENTA("品红色", Color(0xFFF5ACFB), Color(0xFFD18CD7), Color(0xFFB777A8)),
}

/** 预览与实际主题共用生成逻辑，确保容器色和深色色阶一致。 */
internal fun AppThemeColor.previewColorScheme(dark: Boolean, amoled: Boolean = false): ColorScheme =
    presetColorScheme(this, dark, amoled)

/** 浅色表面色阶：单色预设无可用色相，沿用既有冷中性。 */
private fun AppThemeColor.lightSurface(lightness: Float): Color {
    if (this == AppThemeColor.MONOCHROMATIC) {
        // 用同一 HSL 公式的中性分支，明度与彩色预设一致。
        val hsl = floatArrayOf(210f, 0.02f, lightness)
        return Color(ColorUtils.HSLToColor(hsl))
    }
    val hsl = FloatArray(3)
    ColorUtils.colorToHSL(secondarySeed.toArgb(), hsl)
    hsl[1] = 0.12f
    hsl[2] = lightness
    return Color(ColorUtils.HSLToColor(hsl))
}

/**
 * 浅色页面底色：取当前预设 secondarySeed 的色相，固定低饱和与固定明度。
 *
 * 为什么固定明度而不是按 seed 直接混色：AOSP 8 个基础色的明度差很大（黄 vs 蓝），
 * 按比例混合会让绿色主题页面明显更亮、白卡对比变弱。统一 HSL 明度后 8 个预设的
 * 白卡 / 底色对比都在 1.2 左右，页面色相仍随主题变化，卡片始终是最亮的面。
 * 同色相的低明度派生色用于卡片内的指标块 / 弹层，避免暖色主题里嵌冷蓝灰。
 */
private data class LightSurfaces(val page: Color, val high: Color, val highest: Color)

private fun AppThemeColor.lightSurfaces(): LightSurfaces = LightSurfaces(
    page = lightSurface(0.91f),
    high = lightSurface(0.865f),
    highest = lightSurface(0.825f),
)

private fun presetColorScheme(theme: AppThemeColor, dark: Boolean, amoled: Boolean): ColorScheme {
    val argb = theme.color.toArgb()
    val secArgb = theme.secondarySeed.toArgb()
    val tertArgb = theme.tertiarySeed.toArgb()

    // AOSP 基础色是浅亮 seed；浅色主题先把它压到 M3 primary 的深色区间，
    // 再由 primaryContainer 派生为浅容器色，避免浅粉色 / 黄色直接作主色时对比不足。
    val lightPrimary = ColorUtils.blendARGB(argb, android.graphics.Color.BLACK, 0.45f)
    val lightContainer = ColorUtils.blendARGB(argb, android.graphics.Color.WHITE, 0.18f)
    val primary = if (dark) ColorUtils.blendARGB(argb, android.graphics.Color.WHITE, 0.15f)
    else lightPrimary
    val container = if (dark) ColorUtils.blendARGB(argb, android.graphics.Color.BLACK, 0.58f)
    else lightContainer
    val secondary = if (dark) ColorUtils.blendARGB(secArgb, android.graphics.Color.WHITE, 0.38f)
    else ColorUtils.blendARGB(secArgb, android.graphics.Color.BLACK, 0.06f)
    val secContainer = if (dark) ColorUtils.blendARGB(secArgb, android.graphics.Color.BLACK, 0.45f)
    else ColorUtils.blendARGB(secArgb, android.graphics.Color.WHITE, 0.84f)
    val tertiary = if (dark) ColorUtils.blendARGB(tertArgb, android.graphics.Color.WHITE, 0.38f)
    else ColorUtils.blendARGB(tertArgb, android.graphics.Color.BLACK, 0.06f)
    val tertContainer = if (dark) ColorUtils.blendARGB(tertArgb, android.graphics.Color.BLACK, 0.45f)
    else ColorUtils.blendARGB(tertArgb, android.graphics.Color.WHITE, 0.85f)

    // 浅色：页面底色、固定顶栏（surface）与底栏（surfaceContainer）共用同一色，
    // 卡片 / 分组项用最亮表面纯白，白卡成为唯一亮面；深色与 AMOLED 不参与。
    val surfaces = theme.lightSurfaces()
    val base = if (dark) {
        darkColorScheme(
            background = Color(0xFF0F1113),
            surface = Color(0xFF0F1113),
            surfaceContainerLowest = Color(0xFF090A0C),
            surfaceContainerLow = Color(0xFF181B1E),
            surfaceContainer = Color(0xFF202327),
            surfaceContainerHigh = Color(0xFF282C30),
            surfaceContainerHighest = Color(0xFF32363B),
            onSurface = Color(0xFFE2E4E6),
            onSurfaceVariant = Color(0xFFC2C7CB),
            outline = Color(0xFF8C9296),
            outlineVariant = Color(0xFF42484C),
        )
    } else {
        lightColorScheme(
            background = surfaces.page,
            surface = surfaces.page,
            surfaceContainerLowest = Color(0xFFFFFFFF),
            surfaceContainerLow = Color(0xFFFFFFFF),
            surfaceContainer = surfaces.page,
            surfaceContainerHigh = surfaces.high,
            surfaceContainerHighest = surfaces.highest,
            onSurface = Color(0xFF191C1E),
            onSurfaceVariant = Color(0xFF44474B),
            outline = Color(0xFF74777B),
            outlineVariant = Color(0xFFC4C7CC),
        )
    }
    val colored = base.copy(
        primary = Color(primary),
        onPrimary = if (dark) Color(0xFF101418) else Color.White,
        primaryContainer = Color(container),
        onPrimaryContainer = if (dark) Color(0xFFEAF2F8) else Color(primary).darken(0.38f),
        secondary = Color(secondary),
        onSecondary = if (dark) Color(0xFF101418) else Color.White,
        secondaryContainer = Color(secContainer),
        onSecondaryContainer = if (dark) Color(0xFFEAF2F8) else Color(secondary).darken(0.38f),
        tertiary = Color(tertiary),
        tertiaryContainer = Color(tertContainer),
        onTertiaryContainer = if (dark) Color(0xFFFFE9BD) else Color(tertiary).darken(0.38f),
    )
    return if (dark && amoled) colored.withAmoledSurfaces() else colored
}

/** 只覆盖深色背景与表面层级；动态主色、内容色和其它角色保持原样。 */
private fun ColorScheme.withAmoledSurfaces(): ColorScheme = copy(
    background = Color.Black,
    surface = Color.Black,
    surfaceDim = Color.Black,
    surfaceContainerLowest = Color.Black,
    surfaceContainerLow = Color(0xFF080808),
    surfaceContainer = Color(0xFF101010),
    surfaceContainerHigh = Color(0xFF181818),
    surfaceContainerHighest = Color(0xFF202020),
)

/**
 * 动态取色（Android 12+）的浅色分支与预设主题保持同一明暗关系。
 *
 * 系统浅色里 background / surface 是 tone 98、surfaceContainerLow 是 tone 96，
 * 直接用默认值会出现「页面比卡片更亮」。页面统一退到 surfaceContainer（tone 94），
 * 卡片升到 surfaceContainerLowest（纯白）；深色分支不动，避免纯黑上出现灰片。
 */
private fun ColorScheme.withLightPageSurfaces(): ColorScheme = copy(
    background = surfaceContainer,
    surface = surfaceContainer,
    surfaceContainerLow = surfaceContainerLowest,
)

private fun Color.darken(amount: Float) = Color(ColorUtils.blendARGB(toArgb(), android.graphics.Color.BLACK, amount))

@Composable
private fun animatedColor(target: Color, enabled: Boolean): Color =
    animateColorAsState(
        targetValue = target,
        animationSpec = if (enabled) tween(260) else tween(0),
        label = "themeColor",
    ).value

/** 整个色板一起过渡，包括动态取色、深浅色和 AMOLED 表面层级。 */
@Composable
private fun animatedColorScheme(target: ColorScheme): ColorScheme {
    val enabled = animationsEnabled()
    return target.copy(
        primary = animatedColor(target.primary, enabled),
        onPrimary = animatedColor(target.onPrimary, enabled),
        primaryContainer = animatedColor(target.primaryContainer, enabled),
        onPrimaryContainer = animatedColor(target.onPrimaryContainer, enabled),
        inversePrimary = animatedColor(target.inversePrimary, enabled),
        secondary = animatedColor(target.secondary, enabled),
        onSecondary = animatedColor(target.onSecondary, enabled),
        secondaryContainer = animatedColor(target.secondaryContainer, enabled),
        onSecondaryContainer = animatedColor(target.onSecondaryContainer, enabled),
        tertiary = animatedColor(target.tertiary, enabled),
        onTertiary = animatedColor(target.onTertiary, enabled),
        tertiaryContainer = animatedColor(target.tertiaryContainer, enabled),
        onTertiaryContainer = animatedColor(target.onTertiaryContainer, enabled),
        background = animatedColor(target.background, enabled),
        onBackground = animatedColor(target.onBackground, enabled),
        surface = animatedColor(target.surface, enabled),
        onSurface = animatedColor(target.onSurface, enabled),
        surfaceVariant = animatedColor(target.surfaceVariant, enabled),
        onSurfaceVariant = animatedColor(target.onSurfaceVariant, enabled),
        surfaceTint = animatedColor(target.surfaceTint, enabled),
        inverseSurface = animatedColor(target.inverseSurface, enabled),
        inverseOnSurface = animatedColor(target.inverseOnSurface, enabled),
        error = animatedColor(target.error, enabled),
        onError = animatedColor(target.onError, enabled),
        errorContainer = animatedColor(target.errorContainer, enabled),
        onErrorContainer = animatedColor(target.onErrorContainer, enabled),
        outline = animatedColor(target.outline, enabled),
        outlineVariant = animatedColor(target.outlineVariant, enabled),
        scrim = animatedColor(target.scrim, enabled),
        surfaceBright = animatedColor(target.surfaceBright, enabled),
        surfaceDim = animatedColor(target.surfaceDim, enabled),
        surfaceContainer = animatedColor(target.surfaceContainer, enabled),
        surfaceContainerHigh = animatedColor(target.surfaceContainerHigh, enabled),
        surfaceContainerHighest = animatedColor(target.surfaceContainerHighest, enabled),
        surfaceContainerLow = animatedColor(target.surfaceContainerLow, enabled),
        surfaceContainerLowest = animatedColor(target.surfaceContainerLowest, enabled),
    )
}

private val AppTypography = Typography().let { base ->
    Typography(
        displaySmall = base.displaySmall.copy(letterSpacing = 0.sp),
        headlineLarge = base.headlineLarge.copy(letterSpacing = 0.sp),
        headlineMedium = base.headlineMedium.copy(fontSize = 26.sp, lineHeight = 32.sp, letterSpacing = 0.sp),
        headlineSmall = base.headlineSmall.copy(fontSize = 22.sp, lineHeight = 28.sp, letterSpacing = 0.sp),
        titleLarge = base.titleLarge.copy(fontSize = 19.sp, lineHeight = 25.sp, letterSpacing = 0.sp),
        titleMedium = base.titleMedium.copy(letterSpacing = 0.sp),
        titleSmall = base.titleSmall.copy(letterSpacing = 0.sp),
    )
}

private val AppShapes = Shapes(
    extraSmall = smoothShape(8.dp),
    small = smoothShape(12.dp),
    medium = smoothShape(16.dp),
    large = smoothShape(24.dp),
    extraLarge = smoothShape(28.dp),
)

@Composable
fun KemaolifeTheme(
    themeColor: String = AppThemeColor.MONOCHROMATIC.name,
    /** 兼容旧调用；新的 themeMode 未显式传入时沿用这个有效深色值。 */
    darkTheme: Boolean = isSystemInDarkTheme(),
    themeMode: String = if (darkTheme) THEME_MODE_DARK else THEME_MODE_LIGHT,
    dynamicColor: Boolean = DEFAULT_DYNAMIC_COLOR,
    amoled: Boolean = false,
    content: @Composable () -> Unit,
) {
    val dark = isDarkTheme(themeMode, darkTheme)
    val theme = AppThemeColor.entries.firstOrNull { it.name == normalizeThemeColor(themeColor) }
        ?: AppThemeColor.MONOCHROMATIC
    val context = LocalContext.current
    val target = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val dynamic = if (dark) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
            when {
                dark && amoled -> dynamic.withAmoledSurfaces()
                !dark -> dynamic.withLightPageSurfaces()
                else -> dynamic
            }
        }
        else -> presetColorScheme(theme, dark, amoled)
    }
    MaterialTheme(
        colorScheme = animatedColorScheme(target),
        typography = AppTypography,
        shapes = AppShapes,
        content = content,
    )
}

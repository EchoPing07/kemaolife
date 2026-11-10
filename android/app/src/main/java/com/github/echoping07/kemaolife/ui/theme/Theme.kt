package com.github.echoping07.kemaolife.ui.theme

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.graphics.ColorUtils

/**
 * 公开的 Material Design 3 基础种子色彩体系。
 * 不照搬特定第三方应用的私有配色，采用公开规范的色阶与通用命名。
 */
internal enum class AppThemeColor(
    val label: String,
    val color: Color,
    val secondarySeed: Color,
    val tertiarySeed: Color,
) {
    BLUE("晴空蓝", Color(0xFF0061A4), Color(0xFF535F70), Color(0xFF6B5778)),
    TEAL("青碧海", Color(0xFF006A6A), Color(0xFF4A6363), Color(0xFF4B607C)),
    GREEN("松石绿", Color(0xFF2E6C3B), Color(0xFF526350), Color(0xFF386568)),
    ORANGE("琥珀金", Color(0xFF825500), Color(0xFF6C5C3F), Color(0xFF496547)),
    ROSE("拂晓粉", Color(0xFF984061), Color(0xFF73565F), Color(0xFF7C5635)),
    INDIGO("紫罗兰", Color(0xFF6750A4), Color(0xFF625B71), Color(0xFF7D5260)),
    MONOCHROME("板岩灰", Color(0xFF545F71), Color(0xFF5C5F62), Color(0xFF685B58)),
}

/** 为色盘轮盘提供的四象限预览配色 (主色、主容器、次容器、三级容器)。 */
internal data class PalettePreview(
    val primary: Color,
    val primaryContainer: Color,
    val secondaryContainer: Color,
    val tertiaryContainer: Color,
)

internal fun AppThemeColor.palettePreview(dark: Boolean): PalettePreview {
    val pArgb = color.toArgb()
    val sArgb = secondarySeed.toArgb()
    val tArgb = tertiarySeed.toArgb()

    val p = if (dark) ColorUtils.blendARGB(pArgb, android.graphics.Color.WHITE, 0.42f)
    else ColorUtils.blendARGB(pArgb, android.graphics.Color.BLACK, 0.08f)

    val pContainer = if (dark) ColorUtils.blendARGB(pArgb, android.graphics.Color.BLACK, 0.48f)
    else ColorUtils.blendARGB(pArgb, android.graphics.Color.WHITE, 0.82f)

    val sContainer = if (dark) ColorUtils.blendARGB(sArgb, android.graphics.Color.BLACK, 0.45f)
    else ColorUtils.blendARGB(sArgb, android.graphics.Color.WHITE, 0.84f)

    val tContainer = if (dark) ColorUtils.blendARGB(tArgb, android.graphics.Color.BLACK, 0.45f)
    else ColorUtils.blendARGB(tArgb, android.graphics.Color.WHITE, 0.85f)

    return PalettePreview(
        primary = Color(p),
        primaryContainer = Color(pContainer),
        secondaryContainer = Color(sContainer),
        tertiaryContainer = Color(tContainer),
    )
}

@Composable
private fun animatedColor(target: Color): Color =
    animateColorAsState(target, tween(260), label = "themeColor").value

@Composable
private fun appColorScheme(theme: AppThemeColor, dark: Boolean): ColorScheme {
    val argb = theme.color.toArgb()
    val secArgb = theme.secondarySeed.toArgb()
    val tertArgb = theme.tertiarySeed.toArgb()

    val primary = if (dark) ColorUtils.blendARGB(argb, android.graphics.Color.WHITE, 0.42f)
    else ColorUtils.blendARGB(argb, android.graphics.Color.BLACK, 0.08f)

    val container = if (dark) ColorUtils.blendARGB(argb, android.graphics.Color.BLACK, 0.48f)
    else ColorUtils.blendARGB(argb, android.graphics.Color.WHITE, 0.82f)

    val secondary = if (dark) ColorUtils.blendARGB(secArgb, android.graphics.Color.WHITE, 0.38f)
    else ColorUtils.blendARGB(secArgb, android.graphics.Color.BLACK, 0.06f)

    val secContainer = if (dark) ColorUtils.blendARGB(secArgb, android.graphics.Color.BLACK, 0.45f)
    else ColorUtils.blendARGB(secArgb, android.graphics.Color.WHITE, 0.84f)

    val tertiary = if (dark) ColorUtils.blendARGB(tertArgb, android.graphics.Color.WHITE, 0.38f)
    else ColorUtils.blendARGB(tertArgb, android.graphics.Color.BLACK, 0.06f)

    val tertContainer = if (dark) ColorUtils.blendARGB(tertArgb, android.graphics.Color.BLACK, 0.45f)
    else ColorUtils.blendARGB(tertArgb, android.graphics.Color.WHITE, 0.85f)

    // MD3 Expressive 规范中，浅色模式下页面背景使用柔和低饱和度的浅灰底色 (F3F4F6)，
    // 而卡片与组件提升到纯白 (FFFFFF，surfaceContainerLow)，从而让内容从底色上自然脱颖而出，
    // 消除“背景亮白刺眼、组件反而灰暗凹陷”的明度倒置问题。
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
            background = Color(0xFFF3F4F6),
            surface = Color(0xFFF3F4F6),
            surfaceContainerLowest = Color(0xFFFFFFFF),
            surfaceContainerLow = Color(0xFFFFFFFF),
            surfaceContainer = Color(0xFFE5E7EB),
            surfaceContainerHigh = Color(0xFFDDE1E6),
            surfaceContainerHighest = Color(0xFFD1D5DB),
            onSurface = Color(0xFF191C1E),
            onSurfaceVariant = Color(0xFF44474B),
            outline = Color(0xFF74777B),
            outlineVariant = Color(0xFFC4C7CC),
        )
    }
    return base.copy(
        primary = animatedColor(Color(primary)),
        onPrimary = animatedColor(if (dark) Color(0xFF101418) else Color.White),
        primaryContainer = animatedColor(Color(container)),
        onPrimaryContainer = animatedColor(if (dark) Color(0xFFEAF2F8) else Color(primary).darken(0.38f)),
        secondary = animatedColor(Color(secondary)),
        onSecondary = animatedColor(if (dark) Color(0xFF101418) else Color.White),
        secondaryContainer = animatedColor(Color(secContainer)),
        onSecondaryContainer = animatedColor(if (dark) Color(0xFFEAF2F8) else Color(secondary).darken(0.38f)),
        tertiary = animatedColor(Color(tertiary)),
        tertiaryContainer = animatedColor(Color(tertContainer)),
        onTertiaryContainer = animatedColor(if (dark) Color(0xFFFFE9BD) else Color(tertiary).darken(0.38f)),
    )
}

private fun Color.darken(amount: Float) = Color(ColorUtils.blendARGB(toArgb(), android.graphics.Color.BLACK, amount))

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

// 全面升级 MD3 Expressive 圆角规格，卡片升级至 28dp 超大连续圆角
private val AppShapes = Shapes(
    extraSmall = smoothShape(8.dp),
    small = smoothShape(12.dp),
    medium = smoothShape(16.dp),
    large = smoothShape(24.dp),
    extraLarge = smoothShape(28.dp),
)

@Composable
fun KemaolifeTheme(
    themeColor: String = AppThemeColor.BLUE.name,
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    val theme = AppThemeColor.entries.firstOrNull { it.name == themeColor } ?: AppThemeColor.BLUE
    MaterialTheme(
        colorScheme = appColorScheme(theme, darkTheme),
        typography = AppTypography,
        shapes = AppShapes,
        content = content,
    )
}

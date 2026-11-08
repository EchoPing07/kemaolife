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

internal enum class AppThemeColor(val label: String, val color: Color) {
    BLUE("晴蓝", Color(0xFF35658D)),
    INDIGO("靛青", Color(0xFF5D5F8D)),
    GREEN("森绿", Color(0xFF426B54)),
    TEAL("青碧", Color(0xFF326B70)),
    ORANGE("暖橙", Color(0xFF8A563A)),
    ROSE("胭粉", Color(0xFF8B5265)),
}

@Composable
private fun animatedColor(target: Color): Color =
    animateColorAsState(target, tween(260), label = "themeColor").value

@Composable
private fun appColorScheme(seed: Color, dark: Boolean): ColorScheme {
    val argb = seed.toArgb()
    val primary = if (dark) ColorUtils.blendARGB(argb, android.graphics.Color.WHITE, 0.42f)
    else ColorUtils.blendARGB(argb, android.graphics.Color.BLACK, 0.12f)
    val container = if (dark) ColorUtils.blendARGB(argb, android.graphics.Color.BLACK, 0.48f)
    else ColorUtils.blendARGB(argb, android.graphics.Color.WHITE, 0.78f)
    val base = if (dark) {
        darkColorScheme(
            background = Color(0xFF111315), surface = Color(0xFF111315),
            surfaceContainerLowest = Color(0xFF0B0D0E), surfaceContainerLow = Color(0xFF191C1E),
            surfaceContainer = Color(0xFF1E2123), surfaceContainerHigh = Color(0xFF282B2D),
            surfaceContainerHighest = Color(0xFF333638), onSurface = Color(0xFFE3E5E6),
            onSurfaceVariant = Color(0xFFC2C7CB), outline = Color(0xFF8C9296),
            outlineVariant = Color(0xFF42484C),
        )
    } else {
        lightColorScheme(
            background = Color(0xFFFAFAFA), surface = Color(0xFFFAFAFA),
            surfaceContainerLowest = Color(0xFFFFFFFF), surfaceContainerLow = Color(0xFFF1F3F4),
            surfaceContainer = Color(0xFFEBEDEF), surfaceContainerHigh = Color(0xFFE5E7E9),
            surfaceContainerHighest = Color(0xFFDFE2E4), onSurface = Color(0xFF1A1D1F),
            onSurfaceVariant = Color(0xFF454B4F), outline = Color(0xFF757C81),
            outlineVariant = Color(0xFFC5CACD),
        )
    }
    return base.copy(
        primary = animatedColor(Color(primary)),
        onPrimary = animatedColor(if (dark) Color(0xFF172026) else Color.White),
        primaryContainer = animatedColor(Color(container)),
        onPrimaryContainer = animatedColor(if (dark) Color(0xFFEAF2F8) else Color(primary).darken(0.38f)),
        secondary = animatedColor(Color(primary).desaturate(0.15f)),
        onSecondary = animatedColor(if (dark) Color(0xFF172026) else Color.White),
        secondaryContainer = animatedColor(Color(container)),
        onSecondaryContainer = animatedColor(if (dark) Color(0xFFEAF2F8) else Color(primary).darken(0.38f)),
        tertiary = animatedColor(if (dark) Color(0xFFF0B75A) else Color(0xFF815300)),
        tertiaryContainer = animatedColor(if (dark) Color(0xFF5D430B) else Color(0xFFFFE2A8)),
        onTertiaryContainer = animatedColor(if (dark) Color(0xFFFFE9BD) else Color(0xFF2A1B00)),
    )
}

private fun Color.darken(amount: Float) = Color(ColorUtils.blendARGB(toArgb(), android.graphics.Color.BLACK, amount))
private fun Color.desaturate(amount: Float): Color {
    val hsl = FloatArray(3)
    ColorUtils.colorToHSL(toArgb(), hsl)
    hsl[1] *= (1f - amount)
    return Color(ColorUtils.HSLToColor(hsl))
}

private val AppTypography = Typography().let { base ->
    Typography(
        displaySmall = base.displaySmall.copy(letterSpacing = 0.sp),
        headlineLarge = base.headlineLarge.copy(letterSpacing = 0.sp),
        headlineMedium = base.headlineMedium.copy(fontSize = 24.sp, lineHeight = 30.sp, letterSpacing = 0.sp),
        headlineSmall = base.headlineSmall.copy(fontSize = 21.sp, lineHeight = 27.sp, letterSpacing = 0.sp),
        titleLarge = base.titleLarge.copy(fontSize = 19.sp, lineHeight = 25.sp, letterSpacing = 0.sp),
        titleMedium = base.titleMedium.copy(letterSpacing = 0.sp),
        titleSmall = base.titleSmall.copy(letterSpacing = 0.sp),
    )
}

private val AppShapes = Shapes(
    extraSmall = smoothShape(6.dp),
    small = smoothShape(10.dp),
    medium = smoothShape(14.dp),
    large = smoothShape(20.dp),
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
        colorScheme = appColorScheme(theme.color, darkTheme),
        typography = AppTypography,
        shapes = AppShapes,
        content = content,
    )
}

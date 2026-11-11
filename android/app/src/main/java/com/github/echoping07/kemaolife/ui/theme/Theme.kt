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
import com.github.echoping07.kemaolife.data.THEME_MODE_DARK
import com.github.echoping07.kemaolife.data.THEME_MODE_LIGHT
import com.github.echoping07.kemaolife.data.isDarkTheme
import com.github.echoping07.kemaolife.data.normalizeThemeColor

/** 公开的 Material Design 3 基础种子色彩体系。 */
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

/** 预览与实际主题共用生成逻辑，确保容器色和深色色阶一致。 */
internal fun AppThemeColor.previewColorScheme(dark: Boolean, amoled: Boolean = false): ColorScheme =
    presetColorScheme(this, dark, amoled)

private fun presetColorScheme(theme: AppThemeColor, dark: Boolean, amoled: Boolean): ColorScheme {
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
    themeColor: String = AppThemeColor.BLUE.name,
    /** 兼容旧调用；新的 themeMode 未显式传入时沿用这个有效深色值。 */
    darkTheme: Boolean = isSystemInDarkTheme(),
    themeMode: String = if (darkTheme) THEME_MODE_DARK else THEME_MODE_LIGHT,
    dynamicColor: Boolean = false,
    amoled: Boolean = false,
    content: @Composable () -> Unit,
) {
    val dark = isDarkTheme(themeMode, darkTheme)
    val theme = AppThemeColor.entries.firstOrNull { it.name == normalizeThemeColor(themeColor) }
        ?: AppThemeColor.BLUE
    val context = LocalContext.current
    val target = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val dynamic = if (dark) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
            if (dark && amoled) dynamic.withAmoledSurfaces() else dynamic
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

package com.github.echoping07.kemaolife.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * 主题：完整的 Material 3 色板（brand seed 深蓝 #0F4C81）。
 *
 * 之前只覆写过 primary / error 几个槽位，其余槽位是 M3 默认的紫罗兰基线，
 * 于是「品牌蓝按钮 + 紫色胶囊 + 灰紫卡片」混在一起，看着像拼出来的。
 * 这里把 30 个槽位（含 surfaceContainer* 五级容器色）整套定死，
 * 组件在明暗两种模式下都只用同一套语义色。
 *
 * 不用动态取色：课表 / 电费是每天要看的固定页面，配色不宜随壁纸天天变。
 */
private val LightColors = lightColorScheme(
    primary = Color(0xFF0F4C81),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFCFE5FF),
    onPrimaryContainer = Color(0xFF001E33),

    secondary = Color(0xFF50606E),
    onSecondary = Color(0xFFFFFFFF),
    secondaryContainer = Color(0xFFD3E5F5),
    onSecondaryContainer = Color(0xFF0C1D29),

    tertiary = Color(0xFF7A5900),
    onTertiary = Color(0xFFFFFFFF),
    tertiaryContainer = Color(0xFFFFDEA6),
    onTertiaryContainer = Color(0xFF261A00),

    error = Color(0xFFBA1A1A),
    onError = Color(0xFFFFFFFF),
    errorContainer = Color(0xFFFFDAD6),
    onErrorContainer = Color(0xFF410002),

    background = Color(0xFFF8F9FB),
    onBackground = Color(0xFF191C1E),
    surface = Color(0xFFF8F9FB),
    onSurface = Color(0xFF191C1E),
    surfaceVariant = Color(0xFFDEE3EA),
    onSurfaceVariant = Color(0xFF42474E),

    surfaceContainerLowest = Color(0xFFFFFFFF),
    surfaceContainerLow = Color(0xFFF2F4F7),
    surfaceContainer = Color(0xFFECEEF2),
    surfaceContainerHigh = Color(0xFFE6E8EC),
    surfaceContainerHighest = Color(0xFFE0E3E7),

    outline = Color(0xFF72777F),
    outlineVariant = Color(0xFFC2C7CF),

    inverseSurface = Color(0xFF2D3133),
    inverseOnSurface = Color(0xFFEFF1F3),
    inversePrimary = Color(0xFF97CBFF),
    scrim = Color(0xFF000000),
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFF97CBFF),
    onPrimary = Color(0xFF003355),
    primaryContainer = Color(0xFF00496F),
    onPrimaryContainer = Color(0xFFCFE5FF),

    secondary = Color(0xFFB7C9D9),
    onSecondary = Color(0xFF22323F),
    secondaryContainer = Color(0xFF384956),
    onSecondaryContainer = Color(0xFFD3E5F5),

    tertiary = Color(0xFFF6C14B),
    onTertiary = Color(0xFF402D00),
    tertiaryContainer = Color(0xFF5C4200),
    onTertiaryContainer = Color(0xFFFFDEA6),

    error = Color(0xFFFFB4AB),
    onError = Color(0xFF690005),
    errorContainer = Color(0xFF93000A),
    onErrorContainer = Color(0xFFFFDAD6),

    background = Color(0xFF101417),
    onBackground = Color(0xFFE1E2E5),
    surface = Color(0xFF101417),
    onSurface = Color(0xFFE1E2E5),
    surfaceVariant = Color(0xFF42474E),
    onSurfaceVariant = Color(0xFFC2C7CF),

    surfaceContainerLowest = Color(0xFF0B0F11),
    surfaceContainerLow = Color(0xFF181C1F),
    surfaceContainer = Color(0xFF1C2023),
    surfaceContainerHigh = Color(0xFF262A2E),
    surfaceContainerHighest = Color(0xFF313538),

    outline = Color(0xFF8C9199),
    outlineVariant = Color(0xFF42474E),

    inverseSurface = Color(0xFFE1E2E5),
    inverseOnSurface = Color(0xFF2D3133),
    inversePrimary = Color(0xFF0F4C81),
    scrim = Color(0xFF000000),
)

/**
 * 字体：沿用 M3 基线，只收紧标题字距。
 *
 * 中文在小字号下 M3 默认的负字距会把标题挤得贴边，
 * 这里统一把 display / headline 的字距拉回 0。
 */
private val Typography = Typography().let { base ->
    Typography(
        displaySmall = base.displaySmall.copy(letterSpacing = 0.sp),
        headlineLarge = base.headlineLarge.copy(letterSpacing = 0.sp),
        headlineMedium = base.headlineMedium.copy(letterSpacing = 0.sp),
        headlineSmall = base.headlineSmall.copy(letterSpacing = 0.sp),
        titleLarge = base.titleLarge.copy(letterSpacing = 0.sp),
        titleMedium = base.titleMedium.copy(letterSpacing = 0.sp),
        titleSmall = base.titleSmall.copy(letterSpacing = 0.sp),
    )
}

/** M3 默认圆角偏保守，卡片 / 输入框统一 16dp 起步更贴合当前 MD3 观感。 */
private val Shapes = Shapes(
    extraSmall = androidx.compose.foundation.shape.RoundedCornerShape(6.dp),
    small = androidx.compose.foundation.shape.RoundedCornerShape(10.dp),
    medium = androidx.compose.foundation.shape.RoundedCornerShape(14.dp),
    large = androidx.compose.foundation.shape.RoundedCornerShape(20.dp),
    extraLarge = androidx.compose.foundation.shape.RoundedCornerShape(28.dp),
)

@Composable
fun KemaolifeTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        typography = Typography,
        shapes = Shapes,
        content = content,
    )
}

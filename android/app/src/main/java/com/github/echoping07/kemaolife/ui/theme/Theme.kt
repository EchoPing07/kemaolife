package com.github.echoping07.kemaolife.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext

/**
 * 主题：Material 3，深色跟随系统。
 *
 * 品牌色取深蓝（与图标底色一致）；Android 12+ 允许系统取色，
 * 但不用动态色把品牌色顶掉 —— 这个 App 需要一眼认出「课表」，配色不宜随机。
 */
private val LightColors = lightColorScheme(
    primary = Color(0xFF0F4C81),
    onPrimary = Color.White,
    primaryContainer = Color(0xFFD3E4F7),
    onPrimaryContainer = Color(0xFF001D33),
    secondary = Color(0xFF4F616E),
    tertiary = Color(0xFFB07A00),
    // 未认证 / 欠费这类需要注意的状态用琥珀色，与主色区分。
    error = Color(0xFFBA1A1A),
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFF9FCAF5),
    onPrimary = Color(0xFF003353),
    primaryContainer = Color(0xFF00496F),
    onPrimaryContainer = Color(0xFFD3E4F7),
    secondary = Color(0xFFB7C9D6),
    tertiary = Color(0xFFFFC53D),
    error = Color(0xFFFFB4AB),
)

@Composable
fun KemaolifeTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    val colors = when {
        // 动态取色仅在 Android 12+ 且用户显式选择时才用；默认保持品牌色，
        // 所以这里不用 dynamicXxxColorScheme（保留代码路径以便后续开关）。
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.S && false -> {
            val ctx = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(ctx) else dynamicLightColorScheme(ctx)
        }
        darkTheme -> DarkColors
        else -> LightColors
    }

    MaterialTheme(colorScheme = colors, content = content)
}

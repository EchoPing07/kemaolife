package com.github.echoping07.kemaolife.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.Immutable
import androidx.compose.ui.graphics.Color
import kotlin.math.abs

/**
 * 课表色块配色：一个课程固定一种颜色，三端不需要约定，本端按课程名散列。
 *
 * 为什么按课程名散列而不是按节次：同一个课程一天里可能出现在两个时段
 * （例如上午第 1-2 节、下午第 7-8 节），按课程上色才能一眼认出「这是同一门课」。
 *
 * 用 `String.hashCode()` 是安全的：Java 规定它的算法，跨设备跨版本结果一致，
 * 不会出现「同一个课程今天红明天绿」。
 */
@Immutable
internal data class CoursePalette(val container: Color, val onContainer: Color)

private val LightPalettes = listOf(
    CoursePalette(Color(0xFFFFD9DC), Color(0xFF6E1220)), // 玫瑰
    CoursePalette(Color(0xFFFFE0CC), Color(0xFF6B2E00)), // 蜜桃
    CoursePalette(Color(0xFFFFEFC1), Color(0xFF4F3B00)), // 琥珀
    CoursePalette(Color(0xFFDDF3D0), Color(0xFF1F4113)), // 新芽
    CoursePalette(Color(0xFFC9F1E3), Color(0xFF00382C)), // 薄荷
    CoursePalette(Color(0xFFCFE7FF), Color(0xFF0A3557)), // 天蓝
    CoursePalette(Color(0xFFE3DDFF), Color(0xFF2B1F63)), // 淡紫
    CoursePalette(Color(0xFFFBD9EE), Color(0xFF571B44)), // 樱粉
)

private val DarkPalettes = listOf(
    CoursePalette(Color(0xFF5C2A31), Color(0xFFFFD9DC)),
    CoursePalette(Color(0xFF5C3A22), Color(0xFFFFE0CC)),
    CoursePalette(Color(0xFF4C3C10), Color(0xFFFFEFC1)),
    CoursePalette(Color(0xFF2C4A1E), Color(0xFFDDF3D0)),
    CoursePalette(Color(0xFF0E4A3C), Color(0xFFC9F1E3)),
    CoursePalette(Color(0xFF1C3F5C), Color(0xFFCFE7FF)),
    CoursePalette(Color(0xFF3A2F6E), Color(0xFFE3DDFF)),
    CoursePalette(Color(0xFF542441), Color(0xFFFBD9EE)),
)

/** 按课程名取一个稳定配色。 */
@Composable
internal fun coursePalette(name: String): CoursePalette {
    val palettes = if (isSystemInDarkTheme()) DarkPalettes else LightPalettes
    return palettes[abs(name.hashCode()) % palettes.size]
}

package com.github.echoping07.kemaolife.ui

import androidx.compose.ui.graphics.PathFillType
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.addPathNodes
import androidx.compose.ui.unit.dp

/**
 * Tabler Icons，MIT License，2dp 圆头描边。
 * 保留官方 SVG 的路径数据，仅转为 Compose path，不自行重绘。
 * https://github.com/tabler/tabler-icons/tree/main/icons/outline
 */
internal object TabIcons {
    private fun icon(name: String, paths: List<String>): ImageVector =
        ImageVector.Builder(name, 24.dp, 24.dp, 24f, 24f).apply {
            paths.forEach { data ->
                // addPath 需要 List<PathNode>；官方 SVG 路径串先由 addPathNodes 解析（VectorKt）。
                addPath(
                    pathData = addPathNodes(data),
                    fill = null,
                    stroke = SolidColor(androidx.compose.ui.graphics.Color.Black),
                    strokeLineWidth = 2f,
                    strokeLineCap = StrokeCap.Round,
                    strokeLineJoin = StrokeJoin.Round,
                    pathFillType = PathFillType.NonZero,
                )
            }
        }.build()

    val Timetable = icon("CalendarWeek", listOf(
        "M4 7a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12",
        "M16 3v4", "M8 3v4", "M4 11h16",
        "M7 14h.013", "M10.01 14h.005", "M13.01 14h.005", "M16.015 14h.005",
        "M13.015 17h.005", "M7.01 17h.005", "M10.01 17h.005",
    ))

    val More = icon("LayoutDashboard", listOf(
        "M5 4h4a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1v-6a1 1 0 0 1 1 -1",
        "M5 16h4a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1v-2a1 1 0 0 1 1 -1",
        "M15 12h4a1 1 0 0 1 1 1v6a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1v-6a1 1 0 0 1 1 -1",
        "M15 4h4a1 1 0 0 1 1 1v2a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1v-2a1 1 0 0 1 1 -1",
    ))

    val Settings = icon("AdjustmentsHorizontal", listOf(
        "M12 6a2 2 0 1 0 4 0a2 2 0 1 0 -4 0", "M4 6h8", "M16 6h4",
        "M6 12a2 2 0 1 0 4 0a2 2 0 1 0 -4 0", "M4 12h2", "M10 12h10",
        "M15 18a2 2 0 1 0 4 0a2 2 0 1 0 -4 0", "M4 18h11", "M19 18h1",
    ))

    val Appearance = icon("Palette", listOf(
        "M12 21a9 9 0 0 1 0 -18c4.97 0 9 3.582 9 8c0 1.06 -.474 2.078 -1.318 2.828c-.844 .75 -1.989 1.172 -3.182 1.172h-2.5a2 2 0 0 0 -1 3.75a1.3 1.3 0 0 1 -1 2.25",
        "M7.5 10.5a1 1 0 1 0 2 0a1 1 0 1 0 -2 0",
        "M11.5 7.5a1 1 0 1 0 2 0a1 1 0 1 0 -2 0",
        "M15.5 10.5a1 1 0 1 0 2 0a1 1 0 1 0 -2 0",
    ))

    val Info = icon("InfoCircle", listOf(
        "M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0", "M12 9h.01", "M11 12h1v4h1",
    ))

    val Version = icon("Tag", listOf(
        "M6.5 7.5a1 1 0 1 0 2 0a1 1 0 1 0 -2 0",
        "M3 6v5.172a2 2 0 0 0 .586 1.414l7.71 7.71a2.41 2.41 0 0 0 3.408 0l5.592 -5.592a2.41 2.41 0 0 0 0 -3.408l-7.71 -7.71a2 2 0 0 0 -1.414 -.586h-5.172a3 3 0 0 0 -3 3",
    ))

    val Device = icon("DeviceMobile", listOf(
        "M6 5a2 2 0 0 1 2 -2h8a2 2 0 0 1 2 2v14a2 2 0 0 1 -2 2h-8a2 2 0 0 1 -2 -2v-14",
        "M11 4h2", "M12 17v.01",
    ))

    val Github = icon("BrandGithub", listOf(
        "M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5",
    ))

    val License = icon("License", listOf(
        "M15 21h-9a3 3 0 0 1 -3 -3v-1h10v2a2 2 0 0 0 4 0v-14a2 2 0 1 1 2 2h-2m2 -4h-11a3 3 0 0 0 -3 3v11",
        "M9 7h4", "M9 11h4",
    ))

    val Campus = icon("Wifi", listOf(
        "M12 18h.01", "M9.172 15.172a4 4 0 0 1 5.656 0",
        "M6.343 12.343a8 8 0 0 1 11.314 0",
        "M3.515 9.515c4.686 -4.687 12.284 -4.687 17 0",
    ))

    val Eye = icon("Eye", listOf(
        "M10 12a2 2 0 1 0 4 0a2 2 0 0 0 -4 0",
        "M21 12c-2.4 4 -5.4 6 -9 6c-3.6 0 -6.6 -2 -9 -6c2.4 -4 5.4 -6 9 -6c3.6 0 6.6 2 9 6",
    ))

    val EyeOff = icon("EyeOff", listOf(
        "M10.585 10.587a2 2 0 0 0 2.829 2.828",
        "M16.681 16.673a8.717 8.717 0 0 1 -4.681 1.327c-3.6 0 -6.6 -2 -9 -6c1.272 -2.12 2.712 -3.678 4.32 -4.674m2.86 -1.146a9.055 9.055 0 0 1 1.82 -.18c3.6 0 6.6 2 9 6c-.666 1.11 -1.379 2.067 -2.138 2.87",
        "M3 3l18 18",
    ))

    val User = icon("User", listOf(
        "M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0",
        "M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2",
    ))

    val Lock = icon("Lock", listOf(
        "M5 11a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z",
        "M8 9v-4a4 4 0 0 1 8 0v4",
    ))

    val School = icon("School", listOf(
        "M22 9l-10 -4l-10 4l10 4l10 -4v6",
        "M6 10.6v5.4a6 3 0 0 0 12 0v-5.4",
    ))

    val Fingerprint = icon("Fingerprint", listOf(
        "M18.9 7a8 8 0 0 0 -13.8 0",
        "M4 11a12 12 0 0 1 16 0",
        "M8 15a8 8 0 0 0 8 0",
        "M12 11v6",
    ))

    val Clear = icon("X", listOf(
        "M18 6l-12 12",
        "M6 6l12 12",
    ))
}

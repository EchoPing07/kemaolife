package com.github.echoping07.kemaolife.ui

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.DateRange
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material.icons.rounded.MoreVert
import androidx.compose.ui.graphics.vector.ImageVector

/**
 * 底栏图标集中定义。
 *
 * 只用 `material-icons-core` 里已有的图标：不引 `material-icons-extended`
 * （体积大，本 App 只要三个 tab）。
 *
 * 统一取 Rounded 变体：MD3 的底栏指示器是胶囊形，圆头线形图标比直角 Filled 更贴合。
 */
internal object TabIcons {
    val Timetable: ImageVector = Icons.Rounded.DateRange
    val More: ImageVector = Icons.Rounded.MoreVert
    val Settings: ImageVector = Icons.Rounded.Settings
}

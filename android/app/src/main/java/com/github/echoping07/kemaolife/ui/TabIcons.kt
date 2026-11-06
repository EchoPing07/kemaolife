package com.github.echoping07.kemaolife.ui

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.DateRange
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Settings
import androidx.compose.ui.graphics.vector.ImageVector

/**
 * 底栏图标集中定义。
 *
 * 用 `material-icons-core` 里已有的三个图标：只为核心图标加依赖，
 * 不引 `material-icons-extended`（体积大且本 App 只有三个 tab）。
 */
internal object TabIcons {
    val Timetable: ImageVector = Icons.Filled.DateRange
    val More: ImageVector = Icons.Filled.MoreVert
    val Settings: ImageVector = Icons.Filled.Settings
}

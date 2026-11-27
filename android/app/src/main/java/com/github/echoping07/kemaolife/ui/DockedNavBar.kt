package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarDefaults
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * 贴底导航栏尺寸常量。
 *
 * Material 3 的 NavigationBar 默认内容高 80dp（NavigationBarTokens.ContainerHeight）。
 * 带标签 item 的内容高 = 32dp 指示器 + 4dp 指示器内边距 + 4dp 到标签 + 16dp 标签行 = 56dp，
 * placeLabelAndIcon 再把上下留白补到至少 4dp，所以 64dp 是不裁标签的下限；再低会挤压图标与文字。
 * 纯图标模式仅 32dp 指示器垂直居中，56dp 留出上下各 12dp，形态更紧凑。
 */
internal object DockedNavBarMetrics {
    val ContentHeightWithLabel: Dp = 64.dp
    val ContentHeightIconOnly: Dp = 56.dp

    /** 贴底导航栏自身内容高度（不含系统手势区 inset），用于总高度计算与 Snackbar 抬高。 */
    fun contentHeight(showLabels: Boolean): Dp =
        if (showLabels) ContentHeightWithLabel else ContentHeightIconOnly
}

/**
 * 传统贴底导航栏（Material 3 [NavigationBar]）。
 *
 * 在「设置 → 外观 → 显示与色彩」关闭「悬浮底栏」时启用；
 * 与 [FloatingNavBar] 共用条目数据与「底栏文字标签」开关，并归属 [AppDestination.Home] 根场景。
 */
@Composable
internal fun DockedNavBar(
    items: List<FloatingNavItemData>,
    showLabels: Boolean,
    motionEnabled: Boolean,
    modifier: Modifier = Modifier,
) {
    if (items.isEmpty()) return

    val bottomInset =
        NavigationBarDefaults.windowInsets
            .asPaddingValues()
            .calculateBottomPadding()
    val contentHeight = DockedNavBarMetrics.contentHeight(showLabels)

    NavigationBar(
        modifier = modifier.height(contentHeight + bottomInset),
        containerColor = MaterialTheme.colorScheme.surfaceContainer,
        tonalElevation = 0.dp,
    ) {
        items.forEach { item ->
            NavigationBarItem(
                selected = item.selected,
                onClick = item.onClick,
                icon = {
                    AnimatedContent(
                        targetState = item.selected,
                        transitionSpec = {
                            if (motionEnabled) {
                                fadeIn(tween(180)) togetherWith fadeOut(tween(120))
                            } else {
                                EnterTransition.None togetherWith ExitTransition.None
                            }
                        },
                        label = "dockedNavIcon",
                    ) { isSelected ->
                        Icon(
                            imageVector = if (isSelected) item.selectedIcon else item.icon,
                            // 纯图标模式下向无障碍读屏提供完整 tab 名称。
                            contentDescription = if (showLabels) null else item.label,
                        )
                    }
                },
                label =
                    if (showLabels) {
                        { Text(item.label) }
                    } else {
                        null
                    },
                colors =
                    NavigationBarItemDefaults.colors(
                        selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
                        selectedTextColor = MaterialTheme.colorScheme.onSurface,
                        // 官方指示器形状与交互不变；主色容器比中性次容器更清楚地体现当前主题。
                        indicatorColor = MaterialTheme.colorScheme.primaryContainer,
                        unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    ),
            )
        }
    }
}

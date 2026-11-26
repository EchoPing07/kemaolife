package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/** 悬浮底栏条目描述：供主界面三 tab 复用。 */
internal data class FloatingNavItemData(
    val label: String,
    val icon: ImageVector,
    val selectedIcon: ImageVector,
    val selected: Boolean,
    val onClick: () -> Unit,
)

/**
 * 悬浮底栏尺寸与底部避让高度常量。
 *
 * 内容区不再被外层 Scaffold 硬切平，而是滚动到底栏后方并由各页尾部留白
 * 加上此避让高度，确保最后一项可以完整滚出底栏上方。
 */
internal object FloatingNavBarMetrics {
    val BottomMargin: Dp = 12.dp
    val ContainerPadding: Dp = 6.dp
    val ItemHeightWithLabel: Dp = 52.dp
    val ItemHeightIconOnly: Dp = 46.dp
    val BarHeightWithLabel: Dp = ItemHeightWithLabel + ContainerPadding * 2
    val BarHeightIconOnly: Dp = ItemHeightIconOnly + ContainerPadding * 2
    private val ScrollBreathingGap: Dp = 16.dp

    /** 悬浮底栏自身占用高度（不含系统手势区 inset），用于 Snackbar 抬高与页面留白。 */
    fun reservedHeight(showLabels: Boolean): Dp =
        (if (showLabels) BarHeightWithLabel else BarHeightIconOnly) + BottomMargin

    /** 列表 / 课表网格需要的尾部滚动留白（不含系统手势区 inset）。 */
    fun scrollClearance(showLabels: Boolean): Dp =
        reservedHeight(showLabels) + ScrollBreathingGap
}

/**
 * 胶囊形悬浮底栏（MD3 Expressive 风格）。
 *
 * - `showLabels` 由「设置 → 外观」的独立开关控制，开启显示图标 + 文字，关闭切为紧凑纯图标胶囊。
 * - `reveal` ∈ [0, 1] 与设置二级页及预测性返回联动：进入二级页时下滑淡出，
 *   完全隐藏后移出布局树避免拦截底部触控。
 */
@Composable
internal fun FloatingNavBar(
    items: List<FloatingNavItemData>,
    showLabels: Boolean,
    reveal: Float,
    motionEnabled: Boolean,
    modifier: Modifier = Modifier,
) {
    val clampedReveal = reveal.coerceIn(0f, 1f)
    if (clampedReveal <= 0.005f) return

    val density = LocalDensity.current
    val hideOffsetPx = with(density) { (FloatingNavBarMetrics.BarHeightWithLabel + 24.dp).toPx() }
    val scale = 0.92f + 0.08f * clampedReveal

    Box(
        modifier =
            modifier
                .navigationBarsPadding()
                .padding(horizontal = 20.dp, vertical = FloatingNavBarMetrics.BottomMargin)
                .graphicsLayer {
                    alpha = clampedReveal
                    translationY = (1f - clampedReveal) * hideOffsetPx
                    scaleX = scale
                    scaleY = scale
                },
        contentAlignment = Alignment.Center,
    ) {
        Surface(
            shape = CircleShape,
            color = MaterialTheme.colorScheme.surfaceContainerHigh,
            tonalElevation = 0.dp,
            shadowElevation = 6.dp,
            border = BorderStroke(0.5.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.35f)),
        ) {
            Row(
                modifier =
                    Modifier
                        .padding(FloatingNavBarMetrics.ContainerPadding)
                        .selectableGroup(),
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                items.forEach { item ->
                    FloatingNavBarItem(
                        item = item,
                        showLabel = showLabels,
                        motionEnabled = motionEnabled,
                    )
                }
            }
        }
    }
}

@Composable
private fun FloatingNavBarItem(
    item: FloatingNavItemData,
    showLabel: Boolean,
    motionEnabled: Boolean,
) {
    val height by animateDpAsState(
        targetValue = if (showLabel) FloatingNavBarMetrics.ItemHeightWithLabel else FloatingNavBarMetrics.ItemHeightIconOnly,
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.78f, stiffness = 700f) else tween(0),
        label = "floatingNavItemHeight",
    )
    val minWidth by animateDpAsState(
        targetValue = if (showLabel) 88.dp else 68.dp,
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.78f, stiffness = 700f) else tween(0),
        label = "floatingNavItemWidth",
    )
    val indicatorColor by animateColorAsState(
        targetValue = if (item.selected) MaterialTheme.colorScheme.primaryContainer else Color.Transparent,
        animationSpec = if (motionEnabled) tween(180) else tween(0),
        label = "floatingNavIndicator",
    )
    val iconTint by animateColorAsState(
        targetValue =
            if (item.selected) {
                MaterialTheme.colorScheme.onPrimaryContainer
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            },
        animationSpec = if (motionEnabled) tween(180) else tween(0),
        label = "floatingNavIconTint",
    )
    val labelTint by animateColorAsState(
        targetValue =
            if (item.selected) {
                MaterialTheme.colorScheme.onPrimaryContainer
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            },
        animationSpec = if (motionEnabled) tween(180) else tween(0),
        label = "floatingNavLabelTint",
    )

    Box(
        modifier =
            Modifier
                .widthIn(min = minWidth)
                .height(height)
                .clip(CircleShape)
                .background(indicatorColor)
                .semantics {
                    // 纯图标模式下仍向无障碍读屏提供完整 tab 名称。
                    if (!showLabel) contentDescription = item.label
                }
                .selectable(
                    selected = item.selected,
                    role = Role.Tab,
                    onClick = item.onClick,
                )
                .padding(horizontal = if (showLabel) 16.dp else 14.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            AnimatedContent(
                targetState = item.selected,
                transitionSpec = {
                    if (motionEnabled) {
                        fadeIn(tween(160)) togetherWith fadeOut(tween(110))
                    } else {
                        EnterTransition.None togetherWith ExitTransition.None
                    }
                },
                label = "floatingNavIcon",
            ) { isSelected ->
                Icon(
                    imageVector = if (isSelected) item.selectedIcon else item.icon,
                    contentDescription = null,
                    tint = iconTint,
                    modifier = Modifier.size(if (showLabel) 22.dp else 24.dp),
                )
            }
            AnimatedVisibility(
                visible = showLabel,
                enter =
                    if (motionEnabled) {
                        expandVertically(tween(180)) + fadeIn(tween(160))
                    } else {
                        EnterTransition.None
                    },
                exit =
                    if (motionEnabled) {
                        shrinkVertically(tween(160)) + fadeOut(tween(110))
                    } else {
                        ExitTransition.None
                    },
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = item.label,
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = if (item.selected) FontWeight.SemiBold else FontWeight.Medium,
                        color = labelTint,
                        maxLines = 1,
                        softWrap = false,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

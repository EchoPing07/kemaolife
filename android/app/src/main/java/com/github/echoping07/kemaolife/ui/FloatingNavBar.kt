package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandVertically
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.shrinkVertically
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlin.math.max
import kotlin.math.min

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
    val ItemGap: Dp = 4.dp
    val ItemHeightWithLabel: Dp = 52.dp
    val ItemHeightIconOnly: Dp = 46.dp
    val ItemWidthWithLabel: Dp = 88.dp
    val ItemWidthIconOnly: Dp = 68.dp
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

/** 底栏滑动胶囊的水平像素范围。 */
internal data class PillBounds(
    val leftPx: Float,
    val rightPx: Float,
) {
    val widthPx: Float
        get() = (rightPx - leftPx).coerceAtLeast(0f)
}

/**
 * 悬浮底栏滑动胶囊几何计算（纯函数，便于单测验证边界与快速折返）。
 *
 * 左右两边分别以槽位索引（`0 .. count - 1`）做独立弹簧插值：
 * 前导边更快起步、后随边略慢跟进，滑动途中胶囊自然拉长并在落位后弹性收敛；
 * 计算前取 `min` / `max` 保证快速反复切换时胶囊不会反转或塌缩。
 */
internal object FloatingNavIndicatorMath {
    fun bounds(
        leftSlot: Float,
        rightSlot: Float,
        itemCount: Int,
        totalWidthPx: Float,
        gapPx: Float,
        overshootLimitPx: Float = 0f,
        isRtl: Boolean = false,
    ): PillBounds {
        if (itemCount <= 0 || totalWidthPx <= 0f) return PillBounds(0f, 0f)
        val safeGap = gapPx.coerceAtLeast(0f)
        val safeOvershoot = overshootLimitPx.coerceAtLeast(0f)
        val slotWidth = ((totalWidthPx - safeGap * (itemCount - 1)) / itemCount).coerceAtLeast(0f)
        val step = slotWidth + safeGap

        val startSlot = min(leftSlot, rightSlot)
        val endSlot = max(leftSlot, rightSlot)
        val minBound = -safeOvershoot
        val maxBound = totalWidthPx + safeOvershoot

        val rawLeft = (startSlot * step).coerceIn(minBound, maxBound)
        val rawRight = (endSlot * step + slotWidth).coerceIn(rawLeft, maxBound)

        return if (isRtl) {
            PillBounds(
                leftPx = totalWidthPx - rawRight,
                rightPx = totalWidthPx - rawLeft,
            )
        } else {
            PillBounds(leftPx = rawLeft, rightPx = rawRight)
        }
    }
}

/**
 * 胶囊形悬浮底栏（MD3 Expressive 风格）。
 *
 * - 选中背景是共享滑动胶囊，由左右双弹簧在绘制阶段驱动，切 tab（含跨 tab）平滑滑过并带弹性拉伸。
 * - `showLabels` 由「设置 → 外观」的独立开关控制，开启显示图标 + 文字，关闭切为紧凑纯图标胶囊。
 * - 归属 [AppRoute.Home] 根场景：推入或预测性返回二级页时随根场景整体退场与回归。
 */
@Composable
internal fun FloatingNavBar(
    items: List<FloatingNavItemData>,
    showLabels: Boolean,
    motionEnabled: Boolean,
    modifier: Modifier = Modifier,
) {
    if (items.isEmpty()) return

    val density = LocalDensity.current

    val selectedIndex = items.indexOfFirst { it.selected }.coerceAtLeast(0)
    val leftSlot = remember { Animatable(selectedIndex.toFloat()) }
    val rightSlot = remember { Animatable(selectedIndex.toFloat()) }

    LaunchedEffect(selectedIndex, motionEnabled) {
        val target = selectedIndex.toFloat()
        if (!motionEnabled) {
            leftSlot.snapTo(target)
            rightSlot.snapTo(target)
        } else {
            val currentCenter = (leftSlot.value + rightSlot.value) / 2f
            val movingRight = target >= currentCenter
            // 前导边先发、后随边紧跟：横跨多 tab 时拉出流体胶囊感，落位时轻微过冲回弹。
            val leadSpec = spring<Float>(dampingRatio = 0.72f, stiffness = 560f)
            val trailSpec = spring<Float>(dampingRatio = 0.78f, stiffness = 310f)
            coroutineScope {
                launch {
                    leftSlot.animateTo(target, if (movingRight) trailSpec else leadSpec)
                }
                launch {
                    rightSlot.animateTo(target, if (movingRight) leadSpec else trailSpec)
                }
            }
        }
    }

    val itemHeight by animateDpAsState(
        targetValue = if (showLabels) FloatingNavBarMetrics.ItemHeightWithLabel else FloatingNavBarMetrics.ItemHeightIconOnly,
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.78f, stiffness = 700f) else tween(0),
        label = "floatingNavItemHeight",
    )
    val itemSlotWidth by animateDpAsState(
        targetValue = if (showLabels) FloatingNavBarMetrics.ItemWidthWithLabel else FloatingNavBarMetrics.ItemWidthIconOnly,
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.78f, stiffness = 700f) else tween(0),
        label = "floatingNavItemWidth",
    )
    val indicatorColor = MaterialTheme.colorScheme.primaryContainer
    val itemGapPx = with(density) { FloatingNavBarMetrics.ItemGap.toPx() }
    // 允许边缘 tab 落位时有约 2.5dp 弹性回弹空间（落在 6dp 容器内边距内，不触外边框）。
    val overshootLimitPx = with(density) { 2.5.dp.toPx() }

    Box(
        modifier =
            modifier
                .navigationBarsPadding()
                .padding(horizontal = 20.dp, vertical = FloatingNavBarMetrics.BottomMargin),
        contentAlignment = Alignment.Center,
    ) {
        Surface(
            shape = CircleShape,
            color = MaterialTheme.colorScheme.surfaceContainerHigh,
            tonalElevation = 0.dp,
            shadowElevation = 6.dp,
            border = BorderStroke(0.5.dp, MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.35f)),
        ) {
            Layout(
                content = {
                    items.forEach { item ->
                        FloatingNavBarItem(
                            item = item,
                            showLabel = showLabels,
                            motionEnabled = motionEnabled,
                        )
                    }
                },
                modifier =
                    Modifier
                        .padding(FloatingNavBarMetrics.ContainerPadding)
                        .selectableGroup()
                        .drawBehind {
                            val bounds =
                                FloatingNavIndicatorMath.bounds(
                                    leftSlot = leftSlot.value,
                                    rightSlot = rightSlot.value,
                                    itemCount = items.size,
                                    totalWidthPx = size.width,
                                    gapPx = itemGapPx,
                                    overshootLimitPx = overshootLimitPx,
                                    isRtl = layoutDirection == LayoutDirection.Rtl,
                                )
                            if (bounds.widthPx > 0f && size.height > 0f) {
                                val radius = size.height / 2f
                                drawRoundRect(
                                    color = indicatorColor,
                                    topLeft = Offset(bounds.leftPx, 0f),
                                    size = Size(bounds.widthPx, size.height),
                                    cornerRadius = CornerRadius(radius, radius),
                                )
                            }
                        },
            ) { measurables, constraints ->
                val count = measurables.size
                val gapPx = FloatingNavBarMetrics.ItemGap.roundToPx()
                val heightPx = itemHeight.roundToPx()
                val preferredSlotPx = itemSlotWidth.roundToPx()
                val maxPerSlot =
                    if (constraints.hasBoundedWidth && count > 0) {
                        ((constraints.maxWidth - gapPx * (count - 1)) / count).coerceAtLeast(0)
                    } else {
                        Int.MAX_VALUE
                    }
                val slotPx = preferredSlotPx.coerceAtMost(maxPerSlot)
                val itemConstraints = Constraints.fixed(slotPx, heightPx)
                val placeables = measurables.map { it.measure(itemConstraints) }
                val totalWidth =
                    (slotPx * count + gapPx * (count - 1).coerceAtLeast(0))
                        .coerceIn(constraints.minWidth, constraints.maxWidth)
                layout(totalWidth, heightPx) {
                    var x = 0
                    placeables.forEach { placeable ->
                        placeable.placeRelative(x, 0)
                        x += slotPx + gapPx
                    }
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
    val interactionSource = remember { MutableInteractionSource() }
    val pressed by interactionSource.collectIsPressedAsState()
    val pressScale by animateFloatAsState(
        targetValue = if (pressed) 0.92f else 1f,
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.60f, stiffness = 760f) else tween(0),
        label = "floatingNavPressScale",
    )
    val iconTint by animateColorAsState(
        targetValue =
            if (item.selected) {
                MaterialTheme.colorScheme.onPrimaryContainer
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            },
        animationSpec = if (motionEnabled) tween(200) else tween(0),
        label = "floatingNavIconTint",
    )
    val labelTint by animateColorAsState(
        targetValue =
            if (item.selected) {
                MaterialTheme.colorScheme.onPrimaryContainer
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            },
        animationSpec = if (motionEnabled) tween(200) else tween(0),
        label = "floatingNavLabelTint",
    )

    Box(
        modifier =
            Modifier
                .clip(CircleShape)
                .semantics {
                    // 纯图标模式下仍向无障碍读屏提供完整 tab 名称。
                    if (!showLabel) contentDescription = item.label
                }
                .selectable(
                    selected = item.selected,
                    interactionSource = interactionSource,
                    // 关闭点击水波纹 / 按压灰底遮罩，仅保留平滑滑动胶囊与图标动效。
                    indication = null,
                    role = Role.Tab,
                    onClick = item.onClick,
                )
                .padding(horizontal = if (showLabel) 12.dp else 14.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier =
                Modifier.graphicsLayer {
                    scaleX = pressScale
                    scaleY = pressScale
                },
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            AnimatedContent(
                targetState = item.selected,
                transitionSpec = {
                    if (motionEnabled) {
                        (
                            fadeIn(tween(150)) +
                                scaleIn(
                                    animationSpec = spring(dampingRatio = 0.54f, stiffness = 540f),
                                    initialScale = 0.78f,
                                )
                        ) togetherWith
                            (
                                fadeOut(tween(100)) +
                                    scaleOut(tween(100), targetScale = 0.84f)
                            )
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

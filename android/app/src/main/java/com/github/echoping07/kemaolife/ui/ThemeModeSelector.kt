package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.FiniteAnimationSpec
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandHorizontally
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkHorizontally
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.data.THEME_MODE_DARK
import com.github.echoping07.kemaolife.data.THEME_MODE_LIGHT
import com.github.echoping07.kemaolife.data.THEME_MODE_SYSTEM
import com.github.echoping07.kemaolife.data.normalizeThemeMode
import com.github.echoping07.kemaolife.ui.theme.appShape

/**
 * 连体钮的尺寸与动效参数，对齐 MD3 Expressive 的 connected button group 与
 * expressive 快节奏空间动效（`ExpressiveMotionTokens.SpringFastSpatial*`）。
 *
 * 为什么不直接用官方 `ToggleButton` / `ButtonGroupDefaults`：本仓库固定在
 * material3 1.3.1，该 API 从 1.4 起才有；升级依赖需先取得确认，故按官方参数手工复刻。
 */
private val DefaultSegmentHeight = 56.dp
private val SegmentGap = 2.dp

/**
 * 未选中段圆角 8dp = 官方 connected button group 的内角
 * （`ConnectedButtonGroupSmallTokens.InnerCornerCornerSize` → `ShapeTokens.CornerValueSmall`）。
 */
private val SegmentRestCorner = 8.dp
private const val DefaultSelectedWeight = 1.3f
private const val RestWeight = 1f

// 提升到文件级常量，避免每次重组重新分配弹簧 / 过渡规格对象。
private val SpatialFloatSpring: FiniteAnimationSpec<Float> = spring(dampingRatio = 0.6f, stiffness = 800f)
private val SpatialDpSpring: FiniteAnimationSpec<Dp> = spring(dampingRatio = 0.6f, stiffness = 800f)
private val SpatialColorSpring: FiniteAnimationSpec<Color> = spring(dampingRatio = 0.6f, stiffness = 800f)
private val SpatialSizeSpring: FiniteAnimationSpec<IntSize> = spring(dampingRatio = 0.6f, stiffness = 800f)
private val InstantFloatTween: FiniteAnimationSpec<Float> = tween(durationMillis = 0)
private val InstantDpTween: FiniteAnimationSpec<Dp> = tween(durationMillis = 0)
private val InstantColorTween: FiniteAnimationSpec<Color> = tween(durationMillis = 0)
private val InstantSizeTween: FiniteAnimationSpec<IntSize> = tween(durationMillis = 0)

private val IconEnterAnimated = expandHorizontally(SpatialSizeSpring) + fadeIn(SpatialFloatSpring)
private val IconExitAnimated = shrinkHorizontally(SpatialSizeSpring) + fadeOut(SpatialFloatSpring)
private val IconEnterInstant = expandHorizontally(InstantSizeTween) + fadeIn(InstantFloatTween)
private val IconExitInstant = shrinkHorizontally(InstantSizeTween) + fadeOut(InstantFloatTween)

/** 通用连体分段选项：支持可选前缀图标（选中时横向展开）。 */
internal data class SegmentedOption<T>(
    val value: T,
    val label: String,
    val icon: ImageVector? = null,
)

private val themeModeOptions = listOf(
    SegmentedOption(THEME_MODE_SYSTEM, AppearanceText.ModeSystem, TabIcons.Device),
    SegmentedOption(THEME_MODE_LIGHT, AppearanceText.ModeLight, TabIcons.Sun),
    SegmentedOption(THEME_MODE_DARK, AppearanceText.ModeDark, TabIcons.Moon),
)

/**
 * 通用 MD3 Expressive 连体分段选择器：段间 2dp 缝隙、选中段变宽、内角随选中态
 * 在 8dp 与半高胶囊间弹性形变、外侧两极保持完整胶囊端头，选中时图标横向展开。
 */
@Composable
internal fun <T> ConnectedSegmentedGroup(
    options: List<SegmentedOption<T>>,
    selected: T,
    onSelect: (T) -> Unit,
    modifier: Modifier = Modifier,
    height: Dp = DefaultSegmentHeight,
    selectedWeight: Float = DefaultSelectedWeight,
    itemHorizontalPadding: Dp = 4.dp,
) {
    val motionEnabled = animationsEnabled()
    Row(
        modifier = modifier.selectableGroup(),
        horizontalArrangement = Arrangement.spacedBy(SegmentGap),
    ) {
        options.forEachIndexed { index, option ->
            ConnectedSegment(
                option = option,
                selected = selected == option.value,
                leading = index == 0,
                trailing = index == options.lastIndex,
                height = height,
                selectedWeight = selectedWeight,
                horizontalPadding = itemHorizontalPadding,
                motionEnabled = motionEnabled,
                onClick = { onSelect(option.value) },
            )
        }
    }
}

/**
 * 颜色模式（系统 / 浅色 / 深色）的连体三选一。
 *
 * 为什么放在设置总览的「个性化」组而不是外观页：它是外观相关项里最常改的一项，
 * 占一行就能直接切换；外观页只保留配色类设置。
 */
@Composable
internal fun ThemeModeSelector(themeMode: String, onSelect: (String) -> Unit) {
    GroupSurface(position = GroupPosition.FIRST, modifier = Modifier.fillMaxWidth()) {
        Column {
            // 标题行与同组的「外观」「关于」等条目同构：25dp 图标 + 18dp 间距 + titleMedium，
            // 左侧 20dp 与条目正文同一条线，不另造一套排版。
            Row(
                modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 17.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = TabIcons.Contrast,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(25.dp),
                )
                Spacer(Modifier.width(18.dp))
                Text(
                    text = AppearanceText.ColorMode,
                    style = MaterialTheme.typography.titleMedium,
                )
            }
            ConnectedSegmentedGroup(
                options = themeModeOptions,
                selected = normalizeThemeMode(themeMode),
                onSelect = onSelect,
                modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 12.dp, bottom = 16.dp),
            )
        }
    }
}

@Composable
private fun <T> RowScope.ConnectedSegment(
    option: SegmentedOption<T>,
    selected: Boolean,
    leading: Boolean,
    trailing: Boolean,
    height: Dp,
    selectedWeight: Float,
    horizontalPadding: Dp,
    motionEnabled: Boolean,
    onClick: () -> Unit,
) {
    // 外角固定半高形成饱满胶囊端头，内角随选中态在 8dp 与半高之间弹性过渡。
    val outerCorner = height / 2
    val targetCorner = if (selected) outerCorner else SegmentRestCorner
    val weight by animateFloatAsState(
        targetValue = if (selected) selectedWeight else RestWeight,
        animationSpec = if (motionEnabled) SpatialFloatSpring else InstantFloatTween,
        label = "segmentWeight",
    )
    val corner by animateDpAsState(
        targetValue = targetCorner,
        animationSpec = if (motionEnabled) SpatialDpSpring else InstantDpTween,
        label = "segmentCorner",
    )
    val startCorner = if (leading) outerCorner else corner
    val endCorner = if (trailing) outerCorner else corner
    // 静止态复用 remember 缓存的 appShape；圆角过渡帧走 RoundedCornerShape，
    // 避免逐帧构造 SmoothCornerShape 与超椭圆 Path。
    val shape =
        if (corner == targetCorner) {
            appShape(
                topStart = startCorner,
                topEnd = endCorner,
                bottomEnd = endCorner,
                bottomStart = startCorner,
            )
        } else {
            RoundedCornerShape(
                topStart = startCorner,
                topEnd = endCorner,
                bottomEnd = endCorner,
                bottomStart = startCorner,
            )
        }
    val container by animateColorAsState(
        targetValue =
            if (selected) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.surfaceContainerHigh
            },
        animationSpec = if (motionEnabled) SpatialColorSpring else InstantColorTween,
        label = "segmentContainer",
    )
    val content by animateColorAsState(
        targetValue =
            if (selected) {
                MaterialTheme.colorScheme.onPrimary
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            },
        animationSpec = if (motionEnabled) SpatialColorSpring else InstantColorTween,
        label = "segmentContent",
    )
    Box(
        modifier =
            Modifier
                .weight(weight)
                .height(height)
                .clip(shape)
                .background(container)
                .selectable(selected = selected, role = Role.RadioButton, onClick = onClick)
                .padding(horizontal = horizontalPadding),
        contentAlignment = Alignment.Center,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            if (option.icon != null) {
                AnimatedVisibility(
                    visible = selected,
                    enter = if (motionEnabled) IconEnterAnimated else IconEnterInstant,
                    exit = if (motionEnabled) IconExitAnimated else IconExitInstant,
                ) {
                    Row {
                        Icon(
                            imageVector = option.icon,
                            contentDescription = null,
                            tint = content,
                            modifier = Modifier.size(18.dp),
                        )
                        Spacer(Modifier.width(6.dp))
                    }
                }
            }
            Text(
                text = option.label,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
                color = content,
                maxLines = 1,
                softWrap = false,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

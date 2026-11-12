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
import com.github.echoping07.kemaolife.ui.theme.smoothShape

/**
 * 连体钮的尺寸与动效参数，对齐 MD3 Expressive 的 connected button group 与
 * expressive 快节奏空间动效（`ExpressiveMotionTokens.SpringFastSpatial*`）。
 *
 * 为什么不直接用官方 `ToggleButton` / `ButtonGroupDefaults`：本仓库固定在
 * material3 1.3.1，该 API 从 1.4 起才有；升级依赖需先取得确认，故按官方参数手工复刻。
 */
private val ModeHeight = 56.dp
private val ModeGap = 2.dp

/**
 * 未选中段圆角 8dp = 官方 connected button group 的内角
 * （`ConnectedButtonGroupSmallTokens.InnerCornerCornerSize` → `ShapeTokens.CornerValueSmall`）。
 */
private val ModeRestCorner = 8.dp

/** 选中段圆角取半高，被形状自身钳制成完整胶囊，等价于官方的 `CornerFull`。 */
private val ModeSelectedCorner = ModeHeight / 2

/**
 * 首段 / 末段的外侧四角恒为最大半径：官方 connected group 的外角固定是 `CornerFull`，
 * 只有朝向相邻段的内角随选中态变化，这样整排两极始终是饱满的胶囊端头。
 */
private val ModeOuterCorner = ModeHeight / 2

private const val SelectedWeight = 1.3f
private const val RestWeight = 1f

/** 系统关闭动画时给 0 时长，避免出现「设置里关了动画但这里还在动」。 */
private fun <T> fastSpatialSpec(enabled: Boolean): FiniteAnimationSpec<T> = if (enabled) {
    spring<T>(dampingRatio = 0.6f, stiffness = 800f)
} else {
    tween<T>(durationMillis = 0)
}

/** 展开 / 收缩按尺寸插值，类型参数需显式给 IntSize。 */
private fun fastSpatialSizeSpec(enabled: Boolean): FiniteAnimationSpec<IntSize> =
    fastSpatialSpec(enabled)

private data class ThemeModeOption(
    val mode: String,
    val label: String,
    val icon: ImageVector,
)

private val themeModeOptions = listOf(
    ThemeModeOption(THEME_MODE_SYSTEM, AppearanceText.ModeSystem, TabIcons.Device),
    ThemeModeOption(THEME_MODE_LIGHT, AppearanceText.ModeLight, TabIcons.Sun),
    ThemeModeOption(THEME_MODE_DARK, AppearanceText.ModeDark, TabIcons.Moon),
)

/**
 * 颜色模式（系统 / 浅色 / 深色）的连体三选一。
 *
 * 为什么放在设置总览的「个性化」组而不是外观页：它是外观相关项里最常改的一项，
 * 占一行就能直接切换；外观页只保留配色类设置。
 *
 * 形态参考 MD3 Expressive 的 connected button group：段间 2dp 缝隙、选中段变宽
 * （1.3 倍权重、邻段被压窄）、圆角从 22dp 弹性放大到半高成胶囊、图标横向展开，
 * 未选中为中性容器色、选中为主色。
 */
@Composable
internal fun ThemeModeSelector(themeMode: String, onSelect: (String) -> Unit) {
    val current = normalizeThemeMode(themeMode)
    val motionEnabled = animationsEnabled()
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
            Row(
                modifier = Modifier
                    .padding(start = 20.dp, end = 16.dp, top = 12.dp, bottom = 16.dp)
                    .selectableGroup(),
                horizontalArrangement = Arrangement.spacedBy(ModeGap),
            ) {
                themeModeOptions.forEachIndexed { index, option ->
                    ThemeModeSegment(
                        option = option,
                        selected = current == option.mode,
                        leading = index == 0,
                        trailing = index == themeModeOptions.lastIndex,
                        motionEnabled = motionEnabled,
                        onClick = { onSelect(option.mode) },
                    )
                }
            }
        }
    }
}

@Composable
private fun RowScope.ThemeModeSegment(
    option: ThemeModeOption,
    selected: Boolean,
    leading: Boolean,
    trailing: Boolean,
    motionEnabled: Boolean,
    onClick: () -> Unit,
) {
    // 权重动画即「选中段变宽、邻段让位」：官方 connected group 的 1.3 倍选中权重。
    val weight: Float by animateFloatAsState(
        targetValue = if (selected) SelectedWeight else RestWeight,
        animationSpec = fastSpatialSpec(motionEnabled),
        label = "themeModeWeight",
    )
    val corner: Dp by animateDpAsState(
        targetValue = if (selected) ModeSelectedCorner else ModeRestCorner,
        animationSpec = fastSpatialSpec(motionEnabled),
        label = "themeModeCorner",
    )
    // 内角（朝向相邻段那一侧）随选中态在 8dp 与半高之间 morph；首段 / 末段的外侧四角恒为最大。
    val shape = smoothShape(
        topStart = if (leading) ModeOuterCorner else corner,
        bottomStart = if (leading) ModeOuterCorner else corner,
        topEnd = if (trailing) ModeOuterCorner else corner,
        bottomEnd = if (trailing) ModeOuterCorner else corner,
    )
    val container: Color by animateColorAsState(
        targetValue = if (selected) MaterialTheme.colorScheme.primary
        else MaterialTheme.colorScheme.surfaceContainerHigh,
        animationSpec = fastSpatialSpec(motionEnabled),
        label = "themeModeContainer",
    )
    val content: Color by animateColorAsState(
        targetValue = if (selected) MaterialTheme.colorScheme.onPrimary
        else MaterialTheme.colorScheme.onSurfaceVariant,
        animationSpec = fastSpatialSpec(motionEnabled),
        label = "themeModeContent",
    )
    Box(
        modifier = Modifier
            .weight(weight)
            .height(ModeHeight)
            .clip(shape)
            .background(container)
            .selectable(selected = selected, role = Role.RadioButton, onClick = onClick)
            .padding(horizontal = 4.dp),
        contentAlignment = Alignment.Center,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            // 图标只属于选中段：横向展开 / 收缩，是第二处空间动效。
            AnimatedVisibility(
                visible = selected,
                enter = expandHorizontally(fastSpatialSizeSpec(motionEnabled)) +
                    fadeIn(fastSpatialSpec(motionEnabled)),
                exit = shrinkHorizontally(fastSpatialSizeSpec(motionEnabled)) +
                    fadeOut(fastSpatialSpec(motionEnabled)),
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

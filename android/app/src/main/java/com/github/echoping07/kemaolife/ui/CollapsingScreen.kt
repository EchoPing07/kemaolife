package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import kotlin.math.roundToInt

internal val PageHorizontalPadding = 16.dp
private val ExpandedTitleSize = 34.sp
private val ExpandedTitleLineHeight = 42.sp
private val CollapsedTitleHeight = 52.dp
private val ExpandedTitleGap = 12.dp
private val ExpandedTitleHeight = 52.dp
private val PageBottomPadding = 24.dp

/** 两个标题使用独立排版；滚动只驱动位移与淡入淡出，不缩放返回按钮或标题字形。 */
internal fun collapsingTitleProgress(scrollPx: Int, collapseDistancePx: Int): Float {
    if (collapseDistancePx <= 0) return 1f
    return (scrollPx.toFloat() / collapseDistancePx).coerceIn(0f, 1f)
}

/**
 * 滚动内容的最小高度 = 视口高 + 收起距离。
 *
 * 收起进度只由滚动位移决定（见 [collapsingTitleProgress]），可短于一屏的页面
 * （关于、日志开关未开时、未填指纹的校园网）自然内容不满一屏 + 收起距离，
 * 可滚距离小于收起距离，大标题就停在半路、小标题永远淡不满。
 * 把最小高度补到「视口 + 收起距离」，短页也能滚到收起态；长页自然高度更大，
 * `heightIn` 不改变其高度，多出的空间只落在内容末尾（Column 顶部对齐）。
 *
 * 视口高未知（无界约束）或没有收起距离时返回 0，即不加最小高度约束。
 */
internal fun collapsingContentMinHeightPx(viewportPx: Int, collapseDistancePx: Int): Int {
    if (viewportPx <= 0 || collapseDistancePx <= 0) return 0
    return viewportPx + collapseDistancePx
}

/**
 * 状态栏 inset 由外层 Scaffold 统一提供。大标题随内容向上离场，
 * 收起后小标题直接在固定返回按钮右侧出现，不使用 LargeTopAppBar 的字形缩放。
 */
@Composable
internal fun CollapsingScreen(
    title: String,
    modifier: Modifier = Modifier,
    onBack: (() -> Unit)? = null,
    actions: @Composable RowScope.() -> Unit = {},
    scrollState: ScrollState? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val scroll = scrollState ?: rememberScrollState()
    val density = LocalDensity.current
    val collapseDistance = with(density) { (ExpandedTitleGap + ExpandedTitleHeight).toPx() }
    val progress = collapsingTitleProgress(scroll.value, collapseDistance.roundToInt())
    val titleOffset = with(density) { -ExpandedTitleHeight.toPx() * progress }
    val motionEnabled = animationsEnabled()

    BoxWithConstraints(modifier = modifier.fillMaxSize()) {
        // 视口高取本层约束：滚动区与固定顶栏共用同一个 Box，高度一致。
        val contentMinHeight = with(density) {
            collapsingContentMinHeightPx(
                viewportPx = if (constraints.hasBoundedHeight) constraints.maxHeight else 0,
                collapseDistancePx = collapseDistance.roundToInt(),
            ).toDp()
        }
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(scroll)
                .heightIn(min = contentMinHeight)
                .padding(horizontal = PageHorizontalPadding),
        ) {
            Spacer(Modifier.height(CollapsedTitleHeight + ExpandedTitleGap))
            Text(
                text = title,
                style = MaterialTheme.typography.headlineLarge.copy(
                    fontSize = ExpandedTitleSize,
                    lineHeight = ExpandedTitleLineHeight,
                ),
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(ExpandedTitleHeight)
                    .graphicsLayer {
                        translationY = if (motionEnabled) titleOffset else 0f
                        alpha = 1f - progress
                    },
            )
            Spacer(Modifier.height(16.dp))
            content()
            // 属于滚动内容的尾部留白：最后一张卡可以离开底栏，不形成固定空白带。
            // 内容不足一屏 + 收起距离时，[collapsingContentMinHeightPx] 会在其后补足空白，
            // 空白同样属于滚动内容，可以随内容一起滚走。
            Spacer(Modifier.height(PageBottomPadding))
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(CollapsedTitleHeight)
                .background(MaterialTheme.colorScheme.surface)
                .padding(horizontal = PageHorizontalPadding),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (onBack != null) {
                IconButton(onClick = onBack, modifier = Modifier.size(40.dp)) {
                    Box(
                        modifier = Modifier
                            .size(36.dp)
                            .background(MaterialTheme.colorScheme.surfaceContainerLowest, CircleShape),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Rounded.ArrowBack,
                            contentDescription = stringResource(R.string.action_back),
                            tint = MaterialTheme.colorScheme.onSurface,
                            modifier = Modifier.size(19.dp),
                        )
                    }
                }
                Spacer(Modifier.width(12.dp))
            }
            Text(
                text = title,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                modifier = Modifier
                    .weight(1f)
                    .alpha(if (motionEnabled) progress else if (scroll.value > 0) 1f else 0f),
            )
            Row(verticalAlignment = Alignment.CenterVertically, content = actions)
        }
    }
}

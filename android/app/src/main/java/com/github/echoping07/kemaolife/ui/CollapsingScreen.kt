package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
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

    Box(modifier = modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(scroll)
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

package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LargeTopAppBar
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberTopAppBarState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.lerp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R

/**
 * 各滚动页面共用的 MD3 大标题：标题从下沉的大字号随滚动收进顶栏。
 * 顶栏本身不消费竖向手势，内容通过 nestedScroll 驱动收起进度。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun CollapsingScreen(
    title: String,
    modifier: Modifier = Modifier,
    onBack: (() -> Unit)? = null,
    actions: @Composable RowScope.() -> Unit = {},
    scrollState: ScrollState? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val contentScrollState = scrollState ?: rememberScrollState()
    val scrollBehavior = TopAppBarDefaults.exitUntilCollapsedScrollBehavior(
        state = rememberTopAppBarState(),
        // 内容不足一屏时保留展开的大标题，不制造一段没有内容的滚动。
        canScroll = { contentScrollState.canScrollForward || contentScrollState.canScrollBackward },
    )
    val iconScale by animateFloatAsState(
        targetValue = 1f - scrollBehavior.state.collapsedFraction * 0.1f,
        animationSpec = spring(dampingRatio = 0.8f, stiffness = 500f),
        label = "navigationIcon",
    )
    Scaffold(
        modifier = modifier.nestedScroll(scrollBehavior.nestedScrollConnection),
        containerColor = MaterialTheme.colorScheme.surface,
        topBar = {
            LargeTopAppBar(
                title = {
                    val collapsed = scrollBehavior.state.collapsedFraction > 0.98f
                    Text(
                        text = title,
                        // 展开时稍大、略下沉；滚动收起后回到标准顶栏字阶。
                        style = if (collapsed) MaterialTheme.typography.titleLarge
                        else MaterialTheme.typography.headlineLarge.copy(
                            fontSize = 34.sp,
                            lineHeight = 42.sp,
                        ),
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                    )
                },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(
                            onClick = onBack,
                            modifier = Modifier.padding(start = 8.dp).size(40.dp),
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(lerp(34.dp, 30.dp, scrollBehavior.state.collapsedFraction))
                                    .graphicsLayer { scaleX = iconScale; scaleY = iconScale }
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
                    }
                },
                actions = actions,
                colors = TopAppBarDefaults.largeTopAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                    scrolledContainerColor = MaterialTheme.colorScheme.surface,
                    navigationIconContentColor = MaterialTheme.colorScheme.onSurface,
                    titleContentColor = MaterialTheme.colorScheme.onSurface,
                    actionIconContentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                ),
                scrollBehavior = scrollBehavior,
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(contentScrollState)
                .padding(horizontal = 16.dp),
            content = content,
        )
    }
}

package com.github.echoping07.kemaolife

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarDefaults
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.runtime.DisposableEffect
import com.github.echoping07.kemaolife.ui.AppOpenGate
import com.github.echoping07.kemaolife.ui.animationsEnabled
import com.github.echoping07.kemaolife.ui.AppViewModel
import com.github.echoping07.kemaolife.ui.MoreScreen
import com.github.echoping07.kemaolife.ui.ProbeDialog
import com.github.echoping07.kemaolife.ui.SettingsScreen
import com.github.echoping07.kemaolife.ui.TabIcons
import com.github.echoping07.kemaolife.ui.TimetableScreen
import com.github.echoping07.kemaolife.ui.theme.KemaolifeTheme

/** 单 Activity + 三 tab；设置页内部维护自己的返回栈。 */
class MainActivity : ComponentActivity() {
    private val viewModel: AppViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            val state by viewModel.state.collectAsStateWithLifecycle()
            KemaolifeTheme(
                themeColor = state.themeColor,
                themeMode = state.themeMode,
                dynamicColor = state.dynamicColor,
                amoled = state.amoled,
            ) {
                AppRoot(viewModel)
            }
        }
    }
}

private enum class Tab { TIMETABLE, MORE, SETTINGS }

// material3 1.3.1 的 NavigationBar 默认内容高 80dp（NavigationBarTokens.ContainerHeight）。
// 带标签 item 的内容高 = 32dp 指示器 + 4dp 指示器内边距 + 4dp 到标签 + 16dp 标签行 = 56dp，
// placeLabelAndIcon 再把上下留白补到至少 4dp，所以 64dp 是不裁标签的下限；再低会挤压图标与文字。
// 默认 windowInsets 仍需保留系统手势区，因此总高 = 64dp + 底部 inset。
private val NavBarContentHeight = 64.dp

@Composable
private fun AppRoot(viewModel: AppViewModel) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    var tab by rememberSaveable { mutableStateOf(Tab.TIMETABLE) }
    val snackbar = remember { SnackbarHostState() }
    val motionEnabled = animationsEnabled()

    // 回到前台也算「每次打开」：首帧缓存由 ViewModel 构造函数同步给出，
    // 这里只负责触发那一轮无条件联网。注册 observer 时 Activity 可能已处于 RESUMED，
    // Lifecycle 会补发一次没有 ON_STOP 前缀的 ON_RESUME —— 它不是真实打开，
    // 更不能让自动查询撞上 Bridge 的未初始化拦截。Gate 用 remember 跨重组存活。
    val lifecycleOwner = LocalLifecycleOwner.current
    val gate = remember { AppOpenGate() }
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_STOP -> gate.onStop()
                Lifecycle.Event.ON_RESUME -> if (gate.onResume()) {
                    viewModel.onAppOpened(force = true)
                }
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    LaunchedEffect(state.error) {
        state.error?.let {
            snackbar.showSnackbar(it)
            viewModel.consumeError()
        }
    }

    // 设置页三个功能域详情页的「测试」都进同一个弹窗，避免各写一套展示。
    state.probe?.let { probe ->
        ProbeDialog(
            state = probe,
            onClose = viewModel::closeProbe,
            onRetry = { viewModel.runProbe(probe.target) },
        )
    }

    Scaffold(
        // 状态栏与底部手势区分别只由这一层 Scaffold 消费一次。
        snackbarHost = { SnackbarHost(snackbar) },
        bottomBar = {
            // 固定高而非 heightIn(min)：后者不改变 maxHeight，内部 defaultMinSize(80dp) 仍会把
            // 内容撑回 80dp。固定高把 Row / item 的 80dp 下限 coerce 到 64dp，即带标签 item 的最小高。
            val bottomInset = NavigationBarDefaults.windowInsets
                .asPaddingValues()
                .calculateBottomPadding()
            NavigationBar(
                modifier = Modifier.height(NavBarContentHeight + bottomInset),
                containerColor = MaterialTheme.colorScheme.surfaceContainer,
                tonalElevation = 0.dp,
            ) {
                NavItem(
                    selected = tab == Tab.TIMETABLE,
                    onClick = {
                        tab = Tab.TIMETABLE
                        // 切回课表时按最小间隔补一次同步（打开 App 本身是特殊自动，不受限）。
                        viewModel.maybeAutoSync()
                    },
                    motionEnabled = motionEnabled,
                    label = stringResource(R.string.tab_timetable),
                    icon = TabIcons.Timetable,
                    selectedIcon = TabIcons.TimetableFilled,
                )
                NavItem(
                    selected = tab == Tab.MORE,
                    onClick = { tab = Tab.MORE },
                    motionEnabled = motionEnabled,
                    label = stringResource(R.string.tab_more),
                    icon = TabIcons.More,
                    selectedIcon = TabIcons.MoreFilled,
                )
                NavItem(
                    selected = tab == Tab.SETTINGS,
                    onClick = { tab = Tab.SETTINGS },
                    motionEnabled = motionEnabled,
                    label = stringResource(R.string.tab_settings),
                    icon = TabIcons.Settings,
                    selectedIcon = TabIcons.SettingsFilled,
                )
            }
        },
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            AnimatedContent(
                targetState = tab,
                transitionSpec = {
                    val direction = if (targetState.ordinal > initialState.ordinal) 1 else -1
                    if (motionEnabled) {
                        (fadeIn(tween(200, delayMillis = 30)) +
                            slideInHorizontally(tween(300, easing = FastOutSlowInEasing)) { direction * it / 18 } +
                            scaleIn(tween(300, easing = FastOutSlowInEasing), initialScale = 0.985f)) togetherWith
                            (fadeOut(tween(120)) + slideOutHorizontally(tween(220)) { -direction * it / 28 })
                    } else {
                        androidx.compose.animation.EnterTransition.None togetherWith
                            androidx.compose.animation.ExitTransition.None
                    }
                },
                label = "tabTransition",
            ) { currentTab ->
                when (currentTab) {
                    Tab.TIMETABLE -> TimetableScreen(state, viewModel)
                    Tab.MORE -> MoreScreen(state, viewModel, onGoSettings = { tab = Tab.SETTINGS })
                    Tab.SETTINGS -> SettingsScreen(state, viewModel)
                }
            }
        }
    }
}

@Composable
private fun RowScope.NavItem(
    selected: Boolean,
    onClick: () -> Unit,
    motionEnabled: Boolean,
    label: String,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    selectedIcon: androidx.compose.ui.graphics.vector.ImageVector,
) {
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = {
            AnimatedContent(
                targetState = selected,
                transitionSpec = {
                    if (motionEnabled) {
                        fadeIn(tween(180)) togetherWith fadeOut(tween(120))
                    } else {
                        androidx.compose.animation.EnterTransition.None togetherWith
                            androidx.compose.animation.ExitTransition.None
                    }
                },
                label = "navigationIcon",
            ) { isSelected ->
                Icon(if (isSelected) selectedIcon else icon, contentDescription = null)
            }
        },
        label = { Text(label) },
        colors = NavigationBarItemDefaults.colors(
            selectedIconColor = MaterialTheme.colorScheme.onPrimaryContainer,
            selectedTextColor = MaterialTheme.colorScheme.onSurface,
            // 官方指示器形状与交互不变；主色容器比中性次容器更清楚地体现当前主题。
            indicatorColor = MaterialTheme.colorScheme.primaryContainer,
            unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
            unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
        ),
    )
}

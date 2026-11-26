package com.github.echoping07.kemaolife

import android.os.Build
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
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.union
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.github.echoping07.kemaolife.ui.AppOpenGate
import com.github.echoping07.kemaolife.ui.AppViewModel
import com.github.echoping07.kemaolife.ui.FloatingNavBar
import com.github.echoping07.kemaolife.ui.FloatingNavBarMetrics
import com.github.echoping07.kemaolife.ui.FloatingNavItemData
import com.github.echoping07.kemaolife.ui.MoreScreen
import com.github.echoping07.kemaolife.ui.ProbeDialog
import com.github.echoping07.kemaolife.ui.SettingsScreen
import com.github.echoping07.kemaolife.ui.TabIcons
import com.github.echoping07.kemaolife.ui.TimetableScreen
import com.github.echoping07.kemaolife.ui.animationsEnabled
import com.github.echoping07.kemaolife.ui.theme.KemaolifeTheme

/** 单 Activity + 三 tab；设置页内部维护自己的二级导航与预测性返回。 */
class MainActivity : ComponentActivity() {
    private val viewModel: AppViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        // 小米 / 澎湃把只属于三键导航的白色对比度遮罩也画到手势提示线区域，
        // 悬浮底栏下必须关掉遮罩才能让页面内容干净透出到底边（小米开发者平台 pId=1629）。
        // enableEdgeToEdge() 内部会把该开关置为 true，因此必须放在它之后。
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            window.isNavigationBarContrastEnforced = false
        }
        setContent {
            val state by viewModel.state.collectAsStateWithLifecycle()
            KemaolifeTheme(
                themeColor = state.themeColor,
                themeMode = state.themeMode,
                dynamicColor = state.dynamicColor,
                amoled = state.amoled,
                continuousCorners = state.continuousCorners,
            ) {
                AppRoot(viewModel)
            }
        }
    }
}

private enum class Tab { TIMETABLE, MORE, SETTINGS }

@Composable
private fun AppRoot(viewModel: AppViewModel) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    var tab by rememberSaveable { mutableStateOf(Tab.TIMETABLE) }
    var settingsBottomBarReveal by remember { mutableFloatStateOf(1f) }
    val snackbar = remember { SnackbarHostState() }
    val motionEnabled = animationsEnabled()

    // 回到前台也算「每次打开」：首帧缓存由 ViewModel 构造函数同步给出，
    // 这里只负责触发那一轮无条件联网。注册 observer 时 Activity 可能已处于 RESUMED，
    // Lifecycle 会补发一次没有 ON_STOP 前缀的 ON_RESUME —— 它不是真实打开，
    // 更不能让自动查询撞上 Bridge 的未初始化拦截。Gate 用 remember 跨重组存活。
    val lifecycleOwner = LocalLifecycleOwner.current
    val gate = remember { AppOpenGate() }
    DisposableEffect(lifecycleOwner) {
        val observer =
            LifecycleEventObserver { _, event ->
                when (event) {
                    Lifecycle.Event.ON_STOP -> {
                        gate.onStop()
                    }

                    Lifecycle.Event.ON_RESUME -> {
                        if (gate.onResume()) {
                            viewModel.onAppOpened(force = true)
                        }
                    }

                    else -> {
                        Unit
                    }
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

    val navBottomInset = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val navScrollClearance = FloatingNavBarMetrics.scrollClearance(state.showBottomBarLabels) + navBottomInset
    val bottomBarReveal = if (tab == Tab.SETTINGS) settingsBottomBarReveal else 1f
    val snackbarBottomPadding =
        if (bottomBarReveal > 0.01f) {
            FloatingNavBarMetrics.reservedHeight(state.showBottomBarLabels) + 8.dp
        } else {
            16.dp
        }

    Scaffold(
        // 顶部状态栏由内容区单独消费，底边开放给内容透出悬浮底栏；
        // 各滚动容器通过 bottomContentPadding 留出底栏 / FAB 避让空间。
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        snackbarHost = {
            SnackbarHost(
                hostState = snackbar,
                modifier =
                    Modifier
                        .windowInsetsPadding(WindowInsets.navigationBars.union(WindowInsets.ime))
                        .padding(bottom = snackbarBottomPadding),
            )
        },
    ) { _ ->
        Box(modifier = Modifier.fillMaxSize()) {
            Box(
                modifier =
                    Modifier
                        .fillMaxSize()
                        .statusBarsPadding(),
            ) {
                AnimatedContent(
                    targetState = tab,
                    transitionSpec = {
                        val direction = if (targetState.ordinal > initialState.ordinal) 1 else -1
                        if (motionEnabled) {
                            // 平级 tab 要立刻响应：新旧页面同时进退，不等待淡出，也不叠加缩放。
                            (
                                fadeIn(tween(160)) +
                                    slideInHorizontally(tween(200, easing = FastOutSlowInEasing)) { direction * it / 18 }
                            ) togetherWith
                                (
                                    fadeOut(tween(160)) +
                                        slideOutHorizontally(tween(200, easing = FastOutSlowInEasing)) { -direction * it / 28 }
                                )
                        } else {
                            androidx.compose.animation.EnterTransition.None togetherWith
                                androidx.compose.animation.ExitTransition.None
                        }
                    },
                    label = "tabTransition",
                ) { currentTab ->
                    when (currentTab) {
                        Tab.TIMETABLE -> {
                            TimetableScreen(
                                state = state,
                                viewModel = viewModel,
                                bottomContentPadding = navScrollClearance,
                            )
                        }

                        Tab.MORE -> {
                            MoreScreen(
                                state = state,
                                viewModel = viewModel,
                                onGoSettings = { tab = Tab.SETTINGS },
                                bottomContentPadding = navScrollClearance,
                            )
                        }

                        Tab.SETTINGS -> {
                            SettingsScreen(
                                state = state,
                                viewModel = viewModel,
                                rootBottomPadding = navScrollClearance,
                                onBottomBarRevealChange = { settingsBottomBarReveal = it },
                            )
                        }
                    }
                }
            }

            FloatingNavBar(
                items =
                    listOf(
                        FloatingNavItemData(
                            label = stringResource(R.string.tab_timetable),
                            icon = TabIcons.Timetable,
                            selectedIcon = TabIcons.TimetableFilled,
                            selected = tab == Tab.TIMETABLE,
                            onClick = {
                                tab = Tab.TIMETABLE
                                // 切回课表时按最小间隔补一次同步（打开 App 本身是特殊自动，不受限）。
                                viewModel.maybeAutoSync()
                            },
                        ),
                        FloatingNavItemData(
                            label = stringResource(R.string.tab_more),
                            icon = TabIcons.More,
                            selectedIcon = TabIcons.MoreFilled,
                            selected = tab == Tab.MORE,
                            onClick = { tab = Tab.MORE },
                        ),
                        FloatingNavItemData(
                            label = stringResource(R.string.tab_settings),
                            icon = TabIcons.Settings,
                            selectedIcon = TabIcons.SettingsFilled,
                            selected = tab == Tab.SETTINGS,
                            onClick = { tab = Tab.SETTINGS },
                        ),
                    ),
                showLabels = state.showBottomBarLabels,
                reveal = bottomBarReveal,
                motionEnabled = motionEnabled,
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}

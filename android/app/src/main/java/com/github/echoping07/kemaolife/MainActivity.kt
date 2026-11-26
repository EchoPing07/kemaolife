package com.github.echoping07.kemaolife

import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.union
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation3.runtime.NavEntry
import androidx.navigation3.ui.NavDisplay
import com.github.echoping07.kemaolife.ui.AppBackStackOps
import com.github.echoping07.kemaolife.ui.AppNavTransitions
import com.github.echoping07.kemaolife.ui.AppOpenGate
import com.github.echoping07.kemaolife.ui.AppRoute
import com.github.echoping07.kemaolife.ui.AppState
import com.github.echoping07.kemaolife.ui.AppViewModel
import com.github.echoping07.kemaolife.ui.DockedNavBar
import com.github.echoping07.kemaolife.ui.DockedNavBarMetrics
import com.github.echoping07.kemaolife.ui.FloatingNavBar
import com.github.echoping07.kemaolife.ui.FloatingNavBarMetrics
import com.github.echoping07.kemaolife.ui.FloatingNavItemData
import com.github.echoping07.kemaolife.ui.MoreScreen
import com.github.echoping07.kemaolife.ui.PageBottomPadding
import com.github.echoping07.kemaolife.ui.ProbeDialog
import com.github.echoping07.kemaolife.ui.SettingsChildScreen
import com.github.echoping07.kemaolife.ui.SettingsOverviewScreen
import com.github.echoping07.kemaolife.ui.TabIcons
import com.github.echoping07.kemaolife.ui.TimetableScreen
import com.github.echoping07.kemaolife.ui.animationsEnabled
import com.github.echoping07.kemaolife.ui.rememberAppBackStack
import com.github.echoping07.kemaolife.ui.theme.KemaolifeTheme

/** 单 Activity + Jetpack Navigation 3 返回栈；顶层三 Tab 与二级页共享整屏边到边与预测性返回。 */
class MainActivity : ComponentActivity() {
    private val viewModel: AppViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        // 小米 / 澎湃把只属于三键导航的白色对比度遮罩也画到手势提示线区域，
        // 无论悬浮底栏还是贴底导航栏都必须关掉遮罩，才能让页面内容或底栏色干净透出到底边（小米开发者平台 pId=1629）。
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
    val backStack = rememberAppBackStack()
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
    val isOnHome = backStack.lastOrNull() == AppRoute.Home
    val snackbarBottomPadding =
        when {
            !isOnHome -> 16.dp
            state.floatingBottomBar -> FloatingNavBarMetrics.reservedHeight(state.showBottomBarLabels) + 8.dp
            else -> DockedNavBarMetrics.contentHeight(state.showBottomBarLabels) + 8.dp
        }

    Scaffold(
        // 状态栏与导航栏 inset 由各页面自身消费，确保 NavDisplay 整屏转场与预测性返回不出现顶部横切线。
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
        // 外层铺底色：视差缩放转场露出边缘时与主题底色一致，避免闪黑或闪白。
        Box(
            modifier =
                Modifier
                    .fillMaxSize()
                    .background(MaterialTheme.colorScheme.surface),
        ) {
            NavDisplay(
                backStack = backStack,
                onBack = { AppBackStackOps.pop(backStack) },
                transitionSpec = { AppNavTransitions.push(motionEnabled) },
                popTransitionSpec = { AppNavTransitions.pop(motionEnabled) },
                predictivePopTransitionSpec = { _ -> AppNavTransitions.pop(motionEnabled) },
                entryProvider = { route ->
                    NavEntry(key = route, contentKey = route.routeId) { key ->
                        when (key) {
                            AppRoute.Home -> {
                                HomeShell(
                                    tab = tab,
                                    state = state,
                                    viewModel = viewModel,
                                    navScrollClearance = navScrollClearance,
                                    motionEnabled = motionEnabled,
                                    onSelectTab = { next ->
                                        tab = next
                                        if (next == Tab.TIMETABLE) {
                                            // 切回课表时按最小间隔补一次同步（打开 App 本身是特殊自动，不受限）。
                                            viewModel.maybeAutoSync()
                                        }
                                    },
                                    onOpenSettingsChild = { child -> AppBackStackOps.push(backStack, child) },
                                )
                            }

                            is AppRoute.SettingsChild -> {
                                SettingsChildScreen(
                                    route = key,
                                    state = state,
                                    viewModel = viewModel,
                                    onBack = { AppBackStackOps.pop(backStack) },
                                )
                            }
                        }
                    }
                },
                modifier = Modifier.fillMaxSize(),
            )

            // 关闭「预测性返回手势」时由后注册的 BackHandler 接管系统返回：
            // 侧滑过程中不驱动 NavDisplay 跟手预览，松手确认后仍走常规 popTransitionSpec 退场。
            BackHandler(enabled = !state.predictiveBackEnabled && backStack.size > 1) {
                AppBackStackOps.pop(backStack)
            }
        }
    }
}

@Composable
private fun HomeShell(
    tab: Tab,
    state: AppState,
    viewModel: AppViewModel,
    navScrollClearance: Dp,
    motionEnabled: Boolean,
    onSelectTab: (Tab) -> Unit,
    onOpenSettingsChild: (AppRoute.SettingsChild) -> Unit,
) {
    val navItems =
        listOf(
            FloatingNavItemData(
                label = stringResource(R.string.tab_timetable),
                icon = TabIcons.Timetable,
                selectedIcon = TabIcons.TimetableFilled,
                selected = tab == Tab.TIMETABLE,
                onClick = { onSelectTab(Tab.TIMETABLE) },
            ),
            FloatingNavItemData(
                label = stringResource(R.string.tab_more),
                icon = TabIcons.More,
                selectedIcon = TabIcons.MoreFilled,
                selected = tab == Tab.MORE,
                onClick = { onSelectTab(Tab.MORE) },
            ),
            FloatingNavItemData(
                label = stringResource(R.string.tab_settings),
                icon = TabIcons.Settings,
                selectedIcon = TabIcons.SettingsFilled,
                selected = tab == Tab.SETTINGS,
                onClick = { onSelectTab(Tab.SETTINGS) },
            ),
        )
    // 悬浮模式下内容滚至底栏后方，需要留出底栏与手势区高度；贴底模式下底栏独立占位，内容区按常规留白即可。
    val timetableBottomPadding = if (state.floatingBottomBar) navScrollClearance else 16.dp
    val pageBottomPadding = if (state.floatingBottomBar) navScrollClearance else PageBottomPadding

    // 保持内容区在组合树中的位置恒定，切换底栏形态时不重建各 Tab 及其滚动状态。
    Box(
        modifier =
            Modifier
                .fillMaxSize()
                .background(MaterialTheme.colorScheme.surface),
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            Box(
                modifier =
                    Modifier
                        .weight(1f)
                        .fillMaxWidth(),
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
                            EnterTransition.None togetherWith ExitTransition.None
                        }
                    },
                    label = "tabTransition",
                ) { currentTab ->
                    when (currentTab) {
                        Tab.TIMETABLE -> {
                            TimetableScreen(
                                state = state,
                                viewModel = viewModel,
                                bottomContentPadding = timetableBottomPadding,
                            )
                        }

                        Tab.MORE -> {
                            MoreScreen(
                                state = state,
                                viewModel = viewModel,
                                onGoSettings = { onSelectTab(Tab.SETTINGS) },
                                bottomContentPadding = pageBottomPadding,
                            )
                        }

                        Tab.SETTINGS -> {
                            SettingsOverviewScreen(
                                state = state,
                                viewModel = viewModel,
                                rootBottomPadding = pageBottomPadding,
                                onNavigate = onOpenSettingsChild,
                            )
                        }
                    }
                }
            }

            if (!state.floatingBottomBar) {
                DockedNavBar(
                    items = navItems,
                    showLabels = state.showBottomBarLabels,
                    motionEnabled = motionEnabled,
                )
            }
        }

        if (state.floatingBottomBar) {
            FloatingNavBar(
                items = navItems,
                showLabels = state.showBottomBarLabels,
                motionEnabled = motionEnabled,
                modifier = Modifier.align(Alignment.BottomCenter),
            )
        }
    }
}

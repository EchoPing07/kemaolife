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
import androidx.compose.animation.scaleOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
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
            KemaolifeTheme(themeColor = state.themeColor) {
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
    val snackbar = remember { SnackbarHostState() }

    // 回到前台也算「每次打开」：先读缓存（state 里已有），再无条件尝试联网覆盖。
    // 冷启动由 ViewModel.init 触发，这里只处理 ON_RESUME，避免启动时重复两轮。
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) viewModel.onAppOpened(force = true)
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

    // 设置页 / 更多页的「测试」都进同一个弹窗，避免两处各写一套展示。
    state.probe?.let { probe ->
        ProbeDialog(
            state = probe,
            onClose = viewModel::closeProbe,
            onRetry = { viewModel.runProbe(probe.target) },
        )
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbar) },
        bottomBar = {
            NavigationBar(
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
                    label = stringResource(R.string.tab_timetable),
                    icon = { Icon(TabIcons.Timetable, contentDescription = null) },
                )
                NavItem(
                    selected = tab == Tab.MORE,
                    onClick = { tab = Tab.MORE },
                    label = stringResource(R.string.tab_more),
                    icon = { Icon(TabIcons.More, contentDescription = null) },
                )
                NavItem(
                    selected = tab == Tab.SETTINGS,
                    onClick = { tab = Tab.SETTINGS },
                    label = stringResource(R.string.tab_settings),
                    icon = { Icon(TabIcons.Settings, contentDescription = null) },
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
                    (fadeIn(tween(240, delayMillis = 40)) +
                        scaleIn(tween(280, easing = FastOutSlowInEasing), initialScale = 0.985f)) togetherWith
                        (fadeOut(tween(140)) + scaleOut(tween(180), targetScale = 0.99f))
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
    label: String,
    icon: @Composable () -> Unit,
) {
    NavigationBarItem(
        selected = selected,
        onClick = onClick,
        icon = icon,
        label = { Text(label) },
        colors = NavigationBarItemDefaults.colors(
            selectedIconColor = MaterialTheme.colorScheme.onSecondaryContainer,
            selectedTextColor = MaterialTheme.colorScheme.onSurface,
            indicatorColor = MaterialTheme.colorScheme.secondaryContainer,
            unselectedIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
            unselectedTextColor = MaterialTheme.colorScheme.onSurfaceVariant,
        ),
    )
}

package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.ContentTransform
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.runtime.toMutableStateList

/**
 * 全应用页面路由键（供 Jetpack Navigation 3 [androidx.navigation3.ui.NavDisplay] 使用）。
 *
 * 顶层三 Tab 与悬浮底栏共同作为 [Home] 根场景；所有子页面（当前为设置二级页，
 * 后续跨 Tab 详情页或更深层级均可直接追加）与 [Home] 同处一条返回栈，
 * 统一获得弹簧物理转场、状态持久化与系统预测性返回手势。
 */
internal sealed interface AppRoute {
    /** 稳定序列化标识：用于进程重建恢复返回栈及 Navigation 3 的 `contentKey`。 */
    val routeId: String

    data object Home : AppRoute {
        override val routeId: String = "home"
    }

    sealed interface SettingsChild : AppRoute {
        data object Education : SettingsChild {
            override val routeId: String = "settings/education"
        }

        data object Electricity : SettingsChild {
            override val routeId: String = "settings/electricity"
        }

        data object Campus : SettingsChild {
            override val routeId: String = "settings/campus"
        }

        data object Appearance : SettingsChild {
            override val routeId: String = "settings/appearance"
        }

        data object About : SettingsChild {
            override val routeId: String = "settings/about"
        }

        data object Logs : SettingsChild {
            override val routeId: String = "settings/logs"
        }
    }

    companion object {
        private val byId: Map<String, AppRoute> =
            listOf(
                Home,
                SettingsChild.Education,
                SettingsChild.Electricity,
                SettingsChild.Campus,
                SettingsChild.Appearance,
                SettingsChild.About,
                SettingsChild.Logs,
            ).associateBy(AppRoute::routeId)

        fun fromId(id: String): AppRoute? = byId[id]
    }
}

/**
 * 返回栈纯函数操作（不依赖 Android Framework，供 JVM 单测覆盖边界与防连点逻辑）。
 */
internal object AppBackStackOps {
    /** 压入目标页；连续重复点击同一入口时不叠加重复栈帧。 */
    fun push(
        stack: MutableList<AppRoute>,
        route: AppRoute,
    ): Boolean {
        if (stack.lastOrNull() == route) return false
        stack.add(route)
        return true
    }

    /** 弹出栈顶子页；始终保留根页面 [AppRoute.Home]，防止同一帧连按两次返回导致空栈崩溃。 */
    fun pop(stack: MutableList<AppRoute>): Boolean {
        if (stack.size <= 1) return false
        stack.removeAt(stack.lastIndex)
        return true
    }

    fun encode(stack: List<AppRoute>): List<String> = stack.map(AppRoute::routeId)

    /** 进程重建恢复返回栈：忽略未知旧标识，且保证栈底始终是 [AppRoute.Home]。 */
    fun decode(ids: List<String>): List<AppRoute> {
        val restored = ids.mapNotNull(AppRoute::fromId)
        if (restored.isEmpty()) return listOf(AppRoute.Home)
        return if (restored.first() == AppRoute.Home) {
            restored
        } else {
            listOf(AppRoute.Home) + restored.filterNot { it == AppRoute.Home }
        }
    }
}

/** 跨配置变更与进程重建持久化的 Navigation 3 返回栈。 */
@Composable
internal fun rememberAppBackStack(): SnapshotStateList<AppRoute> =
    rememberSaveable(
        saver =
            listSaver(
                save = { AppBackStackOps.encode(it) },
                restore = { AppBackStackOps.decode(it).toMutableStateList() },
            ),
    ) {
        mutableStateListOf(AppRoute.Home)
    }

/**
 * 统一的页面进退与预测性返回转场规格：
 * - 推入：新页从右侧全幅滑入（`+it → 0`），底层视差退后半屏（`0 → -it/2`）并缩小淡出；
 * - 弹出与预测性返回共享镜像路径：顶页向右侧全幅滑出，底层从半屏深处平滑放大淡入，
 *   配合 Navigation 3 内部的 `SeekableTransitionState` 与默认弹簧规格，拖拽与松手衔接自然。
 * - 系统关闭动画时立即切换，不执行位移与缩放。
 */
internal object AppNavTransitions {
    private const val UNDERLAY_SCALE = 0.7f

    fun push(motionEnabled: Boolean): ContentTransform =
        if (!motionEnabled) {
            EnterTransition.None togetherWith ExitTransition.None
        } else {
            slideInHorizontally(initialOffsetX = { it }) togetherWith
                (
                    slideOutHorizontally(targetOffsetX = { -it / 2 }) +
                        scaleOut(targetScale = UNDERLAY_SCALE) +
                        fadeOut()
                )
        }

    fun pop(motionEnabled: Boolean): ContentTransform =
        if (!motionEnabled) {
            EnterTransition.None togetherWith ExitTransition.None
        } else {
            (
                slideInHorizontally(initialOffsetX = { -it / 2 }) +
                    scaleIn(initialScale = UNDERLAY_SCALE) +
                    fadeIn()
            ) togetherWith slideOutHorizontally(targetOffsetX = { it })
        }
}

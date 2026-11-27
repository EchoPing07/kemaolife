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
import androidx.navigation3.runtime.NavKey

/** 设置二级页标识：每个枚举项对应稳定序列化 `routeId`。 */
enum class SettingsPage(val routeId: String) {
    EDUCATION("settings/education"),
    ELECTRICITY("settings/electricity"),
    CAMPUS("settings/campus"),
    APPEARANCE("settings/appearance"),
    ABOUT("settings/about"),
    LOGS("settings/logs"),
}

/**
 * 全应用页面路由键（供 Jetpack Navigation 3 [androidx.navigation3.ui.NavDisplay] 使用）。
 *
 * 顶层三 Tab 与悬浮底栏共同作为 [Home] 根场景；二级页（当前为 [Settings]）
 * 与 [Home] 同处一条返回栈，统一获得弹簧物理转场、状态持久化与系统预测性返回手势。
 */
sealed interface AppDestination : NavKey {
    /** 稳定序列化标识：用于进程重建恢复返回栈及 Navigation 3 的 `contentKey`。 */
    val routeId: String

    data object Home : AppDestination {
        override val routeId: String = "home"
    }

    data class Settings(val page: SettingsPage) : AppDestination {
        override val routeId: String
            get() = page.routeId
    }

    companion object {
        private val settingsById: Map<String, SettingsPage> =
            SettingsPage.entries.associateBy(SettingsPage::routeId)

        fun fromId(id: String): AppDestination? =
            if (id == Home.routeId) {
                Home
            } else {
                settingsById[id]?.let(AppDestination::Settings)
            }
    }
}

/**
 * 返回栈纯函数操作（不依赖 Android Framework，供 JVM 单测覆盖边界与防连点逻辑）。
 */
internal object AppBackStackOps {
    /** 压入目标页；连续重复点击同一入口时不叠加重复栈帧。 */
    fun push(
        stack: MutableList<AppDestination>,
        destination: AppDestination,
    ): Boolean {
        if (stack.lastOrNull() == destination) return false
        stack.add(destination)
        return true
    }

    /** 弹出栈顶子页；始终保留根页面 [AppDestination.Home]，防止同一帧连按两次返回导致空栈崩溃。 */
    fun pop(stack: MutableList<AppDestination>): Boolean {
        if (stack.size <= 1) return false
        stack.removeAt(stack.lastIndex)
        return true
    }

    fun encode(stack: List<AppDestination>): List<String> = stack.map(AppDestination::routeId)

    /** 进程重建恢复返回栈：忽略未知旧标识，且保证栈底始终是 [AppDestination.Home]。 */
    fun decode(ids: List<String>): List<AppDestination> {
        val restored = ids.mapNotNull(AppDestination::fromId)
        if (restored.isEmpty()) return listOf(AppDestination.Home)
        return if (restored.first() == AppDestination.Home) {
            restored
        } else {
            listOf(AppDestination.Home) + restored.filterNot { it == AppDestination.Home }
        }
    }
}

/** 跨配置变更与进程重建持久化的 Navigation 3 返回栈。 */
@Composable
internal fun rememberAppBackStack(): SnapshotStateList<AppDestination> =
    rememberSaveable(
        saver =
            listSaver(
                save = { AppBackStackOps.encode(it) },
                restore = { AppBackStackOps.decode(it).toMutableStateList() },
            ),
    ) {
        mutableStateListOf(AppDestination.Home)
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

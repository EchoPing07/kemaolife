package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.ContentTransform
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.Easing
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.runtime.Composable
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.saveable.listSaver
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.runtime.toMutableStateList
import androidx.compose.ui.unit.IntOffset
import androidx.navigation3.runtime.NavKey
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.exp
import kotlin.math.sin
import kotlin.math.sqrt

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
 * 欠阻尼弹簧的解析阶跃缓动（参数对齐 Miuix 默认页面转场 `response = 0.8f, damping = 0.95f`）：
 * 前段响应利落、后段绵长柔和收尾，在 `[0, 1]` 内严格单调无回弹。
 */
internal class NavSettleEasing(
    response: Float = 0.8f,
    damping: Float = 0.95f,
) : Easing {
    private val r: Double
    private val w: Double
    private val c2: Double

    init {
        val omega = 2.0 * PI / response
        val k = omega * omega
        val c = damping * 4.0 * PI / response
        w = sqrt(4.0 * k - c * c) / 2.0
        r = -c / 2.0
        c2 = r / w
    }

    override fun transform(fraction: Float): Float {
        if (fraction <= 0f) return 0f
        if (fraction >= 1f) return 1f
        val t = fraction.toDouble()
        return (exp(r * t) * (-cos(w * t) + c2 * sin(w * t)) + 1.0).toFloat()
    }
}

/**
 * 统一的页面进退与预测性返回转场规格：
 * - 推入：新页从右侧全幅滑入（`+it → 0`），底层无缩放、向左轻微视差偏移 25%（`0 → -it/4`）并轻度衰减透明度；
 * - 弹出：顶页向右侧全幅滑出，底层从左侧 25% 处平移归位，全程使用 `500ms` + [NavSettleEasing]；
 * - 预测性返回：Navigation 3 的 `SeekableTransitionState` 在松手后会继续沿同一条曲线播完剩余段，
 *   因此使用前段近 1:1 跟手、后段柔和减速的 [PredictivePopEasing]，避免 `LinearEasing` 松手后匀速硬停；
 * - 系统关闭动画时立即切换，不执行位移与淡入淡出。
 */
internal object AppNavTransitions {
    private const val DURATION_MILLIS = 500
    private const val PREDICTIVE_POP_DURATION_MILLIS = 450
    private const val UNDERLAY_OFFSET_DIVISOR = 4
    private const val UNDERLAY_ALPHA = 0.9f
    private val SettleEasing: Easing = NavSettleEasing()
    // 起点斜率 1.0（拖拽初期自然跟手），终点斜率 0.0（松手后柔和减速收尾，无硬停）。
    internal val PredictivePopEasing: Easing = CubicBezierEasing(0.3f, 0.3f, 0.35f, 1f)

    fun push(motionEnabled: Boolean): ContentTransform =
        if (!motionEnabled) {
            EnterTransition.None togetherWith ExitTransition.None
        } else {
            val slideSpec = tween<IntOffset>(durationMillis = DURATION_MILLIS, easing = SettleEasing)
            val fadeSpec = tween<Float>(durationMillis = DURATION_MILLIS, easing = SettleEasing)
            slideInHorizontally(animationSpec = slideSpec, initialOffsetX = { it }) togetherWith
                (
                    slideOutHorizontally(
                        animationSpec = slideSpec,
                        targetOffsetX = { -it / UNDERLAY_OFFSET_DIVISOR },
                    ) + fadeOut(animationSpec = fadeSpec, targetAlpha = UNDERLAY_ALPHA)
                )
        }

    fun pop(motionEnabled: Boolean): ContentTransform =
        popWithSpec(
            motionEnabled = motionEnabled,
            durationMillis = DURATION_MILLIS,
            easing = SettleEasing,
        )

    fun predictivePop(motionEnabled: Boolean): ContentTransform =
        popWithSpec(
            motionEnabled = motionEnabled,
            durationMillis = PREDICTIVE_POP_DURATION_MILLIS,
            easing = PredictivePopEasing,
        )

    private fun popWithSpec(
        motionEnabled: Boolean,
        durationMillis: Int,
        easing: Easing,
    ): ContentTransform =
        if (!motionEnabled) {
            EnterTransition.None togetherWith ExitTransition.None
        } else {
            val slideSpec = tween<IntOffset>(durationMillis = durationMillis, easing = easing)
            val fadeSpec = tween<Float>(durationMillis = durationMillis, easing = easing)
            (
                slideInHorizontally(
                    animationSpec = slideSpec,
                    initialOffsetX = { -it / UNDERLAY_OFFSET_DIVISOR },
                ) + fadeIn(animationSpec = fadeSpec, initialAlpha = UNDERLAY_ALPHA)
            ) togetherWith slideOutHorizontally(animationSpec = slideSpec, targetOffsetX = { it })
        }
}

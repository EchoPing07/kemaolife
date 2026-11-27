package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.spring
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.togetherWith
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.AppException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** 仅手动操作的结果：配置类异常与网络 / 业务失败不混为同一种图标。 */
internal enum class ActionFeedback { SUCCESS, FAILURE, UNAVAILABLE }

private const val FEEDBACK_DURATION_MS = 1600L

private sealed interface RefreshVisualState {
    data object Idle : RefreshVisualState
    data object Loading : RefreshVisualState
    data class Result(val feedback: ActionFeedback) : RefreshVisualState
}

/**
 * 每次点击产生新代次；结果只返回给本次点击，自动查询不得借用手动反馈。
 *
 * 自动清零定时器由控制器依托 ViewModel 作用域维护，不绑定具体页面组合生命周期，
 * 避免用户在 1.6s 内切 tab 导致结果图标残留不消。
 */
internal class ActionFeedbackController(
    private val scope: CoroutineScope,
    private val clearDelayMs: Long = FEEDBACK_DURATION_MS,
) {
    var generation: Long = 0L
        private set
    var result by mutableStateOf<ActionFeedback?>(null)
        private set
    private var clearJob: Job? = null

    fun begin(): Long {
        clearJob?.cancel()
        clearJob = null
        result = null
        return ++generation
    }

    fun complete(id: Long, feedback: ActionFeedback): Boolean {
        if (id != generation) return false
        result = feedback
        scheduleClear(id)
        return true
    }

    /** 无需加载态的即时反馈（如缺凭据或非 WiFi 跳过）。 */
    fun show(feedback: ActionFeedback) {
        val id = begin()
        complete(id, feedback)
    }

    fun clear() {
        clearJob?.cancel()
        clearJob = null
        generation++
        result = null
    }

    private fun scheduleClear(id: Long) {
        clearJob?.cancel()
        clearJob = scope.launch {
            delay(clearDelayMs)
            if (generation == id) {
                result = null
            }
        }
    }
}

internal fun actionFeedbackFor(throwable: Throwable): ActionFeedback =
    if (throwable is AppException && throwable.code in 3001..3003) {
        ActionFeedback.UNAVAILABLE
    } else {
        ActionFeedback.FAILURE
    }

@Composable
internal fun ActionResultIcon(
    result: ActionFeedback,
    modifier: Modifier = Modifier,
) {
    val (icon, tint, description) = when (result) {
        ActionFeedback.SUCCESS -> Triple(
            TabIcons.Check,
            if (MaterialTheme.colorScheme.surface.luminance() < 0.5f) Color(0xFF86D5A2) else Color(0xFF287445),
            stringResource(R.string.action_succeeded),
        )
        ActionFeedback.FAILURE -> Triple(TabIcons.Clear, MaterialTheme.colorScheme.error, stringResource(R.string.action_failed))
        ActionFeedback.UNAVAILABLE -> Triple(TabIcons.ExclamationMark, MaterialTheme.colorScheme.tertiary, stringResource(R.string.action_unavailable))
    }
    Icon(icon, contentDescription = description, tint = tint, modifier = modifier)
}

/** 通用 refresh 按钮：加载 / 结果 / 默认三态间使用轻微形变，不持续旋转图标。 */
@Composable
internal fun RefreshAction(
    loading: Boolean,
    result: ActionFeedback? = null,
    onClick: () -> Unit,
    contentDescription: String? = null,
    modifier: Modifier = Modifier,
) {
    val animated = animationsEnabled()
    val state = when {
        loading -> RefreshVisualState.Loading
        result != null -> RefreshVisualState.Result(result)
        else -> RefreshVisualState.Idle
    }
    IconButton(onClick = onClick, enabled = !loading, modifier = modifier) {
        AnimatedContent(
            targetState = state,
            transitionSpec = {
                if (animated) {
                    (fadeIn() + scaleIn(initialScale = 0.72f, animationSpec = spring(dampingRatio = 0.65f))) togetherWith
                        (fadeOut() + scaleOut(targetScale = 0.82f))
                } else {
                    EnterTransition.None togetherWith ExitTransition.None
                }
            },
            label = "refreshFeedback",
        ) { current ->
            when (current) {
                RefreshVisualState.Loading -> ExpressiveLoadingIndicator(size = 21.dp)
                RefreshVisualState.Idle -> Icon(
                    imageVector = TabIcons.Refresh,
                    contentDescription = contentDescription ?: stringResource(R.string.action_refresh),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                is RefreshVisualState.Result -> ActionResultIcon(current.feedback)
            }
        }
    }
}

@Composable
internal fun ActionIcon(
    icon: ImageVector,
    loading: Boolean,
    result: ActionFeedback?,
    tint: Color,
    contentDescription: String,
    modifier: Modifier = Modifier,
) {
    when {
        loading -> ExpressiveLoadingIndicator(size = 18.dp, color = tint, modifier = modifier)
        result != null -> ActionResultIcon(result, modifier = modifier)
        else -> Icon(icon, contentDescription = contentDescription, tint = tint, modifier = modifier)
    }
}

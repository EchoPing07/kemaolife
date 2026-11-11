package com.github.echoping07.kemaolife.ui

import android.provider.Settings
import androidx.compose.animation.AnimatedContent
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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import kotlinx.coroutines.delay

/** 仅手动操作的结果：配置类异常与网络 / 业务失败不混为同一种图标。 */
internal enum class ActionFeedback { SUCCESS, FAILURE, UNAVAILABLE }

private const val FEEDBACK_DURATION_MS = 1600L

/** 只为局部状态动效降级；页面的布局与手势保持完整。 */
@Composable
private fun motionEnabled(): Boolean {
    val context = LocalContext.current
    return Settings.Global.getFloat(
        context.contentResolver,
        Settings.Global.ANIMATOR_DURATION_SCALE,
        1f,
    ) != 0f
}

/** 每次点击产生新代次；结果只返回给本次点击，自动查询不得借用手动反馈。 */
internal class ActionFeedbackController {
    var generation by mutableLongStateOf(0L)
        private set
    var result by androidx.compose.runtime.mutableStateOf<ActionFeedback?>(null)
        private set
    private var resultGeneration = 0L

    fun begin(): Long {
        result = null
        return ++generation
    }

    fun complete(id: Long, feedback: ActionFeedback): Boolean {
        if (id != generation) return false
        result = feedback
        resultGeneration = id
        return true
    }

    suspend fun expire() {
        val id = resultGeneration
        delay(FEEDBACK_DURATION_MS)
        if (resultGeneration == id) result = null
    }

    fun clear() {
        generation++
        result = null
    }
}

@Composable
internal fun ActionFeedbackController.ExpireFeedback() {
    LaunchedEffect(result) {
        if (result != null) expire()
    }
}

internal fun actionFeedbackFor(throwable: Throwable): ActionFeedback =
    if (throwable is com.github.echoping07.kemaolife.data.AppException && throwable.code in 3001..3003) {
        ActionFeedback.UNAVAILABLE
    } else {
        ActionFeedback.FAILURE
    }

@Composable
internal fun ActionResultIcon(
    result: ActionFeedback,
    modifier: androidx.compose.ui.Modifier = androidx.compose.ui.Modifier,
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
    modifier: androidx.compose.ui.Modifier = androidx.compose.ui.Modifier,
) {
    val animated = motionEnabled()
    val state = when {
        loading -> "loading"
        result != null -> result.name
        else -> "idle"
    }
    IconButton(onClick = onClick, enabled = !loading, modifier = modifier) {
        AnimatedContent(
            targetState = state,
            transitionSpec = {
                if (animated) {
                    (fadeIn() + scaleIn(initialScale = 0.72f, animationSpec = spring(dampingRatio = 0.65f))) togetherWith
                        (fadeOut() + scaleOut(targetScale = 0.82f))
                } else {
                    androidx.compose.animation.EnterTransition.None togetherWith
                        androidx.compose.animation.ExitTransition.None
                }
            },
            label = "refreshFeedback",
        ) { current ->
            when {
                current == "loading" -> ExpressiveLoadingIndicator(size = 21.dp)
                current == "idle" -> Icon(
                    imageVector = TabIcons.Refresh,
                    contentDescription = contentDescription ?: stringResource(R.string.action_refresh),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = androidx.compose.ui.Modifier,
                )
                else -> ActionResultIcon(ActionFeedback.valueOf(current))
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
    modifier: androidx.compose.ui.Modifier = androidx.compose.ui.Modifier,
) {
    when {
        loading -> ExpressiveLoadingIndicator(size = 18.dp, color = tint, modifier = modifier)
        result != null -> ActionResultIcon(result, modifier = modifier)
        else -> Icon(icon, contentDescription = contentDescription, tint = tint, modifier = modifier)
    }
}

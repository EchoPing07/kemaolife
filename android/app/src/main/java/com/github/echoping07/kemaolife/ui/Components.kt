package com.github.echoping07.kemaolife.ui

import android.provider.Settings
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.keyframes
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.data.Entry
import com.github.echoping07.kemaolife.ui.theme.coursePalette
import com.github.echoping07.kemaolife.ui.theme.smoothShape
import kotlin.math.cos
import kotlin.math.min
import kotlin.math.sin

/**
 * 通用容器：MD3 Expressive filled card + 标题行。
 *
 * 卡片使用 [MaterialTheme.colorScheme.surfaceContainerLow]（浅色模式下为纯白），
 * 与带当前主题色相的浅灰页面底色形成明暗对比，让内容区域成为视觉焦点。
 * 圆角采用 extraLarge（28dp 平滑圆角），展现饱满包容的张力。
 */
@Composable
internal fun SectionCard(
    title: String,
    modifier: Modifier = Modifier,
    action: (@Composable () -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    Card(
        modifier = modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
        ),
        shape = MaterialTheme.shapes.extraLarge,
    ) {
        Column(modifier = Modifier.padding(top = 10.dp, bottom = 14.dp)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 22.dp, end = 12.dp, top = 4.dp, bottom = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                action?.invoke()
            }
            Column(modifier = Modifier.padding(horizontal = 22.dp, vertical = 6.dp), content = content)
        }
    }
}

/**
 * MD3 Expressive 标准的灵动波浪环形加载指示器 (Circular Wavy Progress Indicator)。
 *
 * 结合 Material 官方非对称缓动伸缩与流体环形正弦波浪，
 * 彻底告别传统死板细圈，展现富有弹性质感的现代科技感。
 */
/** 系统关闭动画时不运行无限装饰动画；状态与静态加载提示仍保留。 */
@Composable
internal fun animationsEnabled(): Boolean {
    val context = LocalContext.current
    // 每次组合读取系统开关，设置变更后无需重启；读取自身不创建持续动画。
    return Settings.Global.getFloat(
        context.contentResolver,
        Settings.Global.ANIMATOR_DURATION_SCALE,
        1f,
    ) != 0f
}

@Composable
internal fun ExpressiveLoadingIndicator(
    modifier: Modifier = Modifier,
    color: Color = MaterialTheme.colorScheme.primary,
    size: Dp = 22.dp,
    strokeWidth: Dp = if (size < 22.dp) 2.2.dp else 3.2.dp,
) {
    if (!animationsEnabled()) {
        Canvas(modifier = modifier.size(size)) {
            drawArc(
                color = color,
                startAngle = -90f,
                sweepAngle = 270f,
                useCenter = false,
                style = Stroke(strokeWidth.toPx(), cap = StrokeCap.Round),
            )
        }
        return
    }
    val transition = rememberInfiniteTransition(label = "expressiveWavy")

    // 基础自转相位 (0 ~ 360)
    val baseRotation by transition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 2200, easing = LinearEasing),
            repeatMode = RepeatMode.Restart,
        ),
        label = "baseRotation",
    )

    // 头部扩展相位 (0 ~ 360)
    val headAngle by transition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1332, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Restart,
        ),
        label = "headAngle",
    )

    // 尾部追赶相位 (0 ~ 360)
    val tailAngle by transition.animateFloat(
        initialValue = 0f,
        targetValue = 360f,
        animationSpec = infiniteRepeatable(
            animation = keyframes {
                durationMillis = 1332
                0f at 0 using FastOutSlowInEasing
                0f at 380 using FastOutSlowInEasing
                360f at 1332
            },
            repeatMode = RepeatMode.Restart,
        ),
        label = "tailAngle",
    )

    // 波浪流体向前行进相位
    val wavePhase by transition.animateFloat(
        initialValue = 0f,
        targetValue = (2 * Math.PI).toFloat(),
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 900, easing = LinearEasing),
            repeatMode = RepeatMode.Restart,
        ),
        label = "wavePhase",
    )

    Canvas(modifier = modifier.size(size)) {
        val w = this.size.width
        val h = this.size.height
        val cx = w / 2f
        val cy = h / 2f
        val strokePx = strokeWidth.toPx()
        val baseR = (min(w, h) - strokePx * 1.5f) / 2f
        val waveAmplitude = (baseR * 0.14f).coerceIn(1.2f, 3.2f)
        val waveCount = 5

        // 计算当前圆弧的扫描跨度
        val sweep = (headAngle - tailAngle).let { if (it < 0f) it + 360f else it }.coerceIn(18f, 290f)
        val startAngle = (baseRotation + tailAngle) % 360f

        val path = Path()
        val steps = (sweep / 4).toInt().coerceAtLeast(16)
        for (i in 0..steps) {
            val progress = i.toFloat() / steps
            val currentDeg = startAngle + sweep * progress
            val currentRad = Math.toRadians(currentDeg.toDouble()).toFloat()

            // 径向正弦波扰动
            val waveOffset = waveAmplitude * sin(waveCount * currentRad - wavePhase)
            val r = baseR + waveOffset
            val x = cx + r * cos(currentRad)
            val y = cy + r * sin(currentRad)

            if (i == 0) {
                path.moveTo(x, y)
            } else {
                path.lineTo(x, y)
            }
        }

        drawPath(
            path = path,
            color = color,
            style = Stroke(
                width = strokePx,
                cap = StrokeCap.Round,
                join = StrokeJoin.Round,
            ),
        )
    }
}

/** 今日课表条目（「更多」页用）。 */
@Composable
internal fun EntryRow(entry: Entry) {
    val palette = coursePalette(entry.name)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 9.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .width(4.dp)
                .height(34.dp)
                .clip(smoothShape(2.dp))
                .background(palette.container),
        )
        Spacer(Modifier.width(12.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = entry.name,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
            )
            val second = listOf(entry.location, entry.teacher)
                .filter { it.isNotBlank() }
                .joinToString(" · ")
            if (second.isNotBlank()) {
                Text(
                    text = second,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
                )
            }
        }
        Spacer(Modifier.width(8.dp))
        Column(horizontalAlignment = Alignment.End) {
            Text(
                text = entry.time.substringBefore('~').ifBlank { entry.periods },
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.SemiBold,
            )
            if (entry.periods.isNotBlank()) {
                Text(
                    text = "第 ${entry.periods} 节",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

/** 通用「加载中」占位：使用 M3 Expressive 形变加载动效。 */
@Composable
internal fun LoadingBlock(modifier: Modifier = Modifier) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 32.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        ExpressiveLoadingIndicator(size = 32.dp)
    }
}

/** 空态 / 错误提示块。 */
@Composable
internal fun EmptyHint(text: String, modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.extraLarge)
            .background(MaterialTheme.colorScheme.surfaceContainerLow)
            .padding(horizontal = 24.dp, vertical = 32.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/** 状态圆点：校园网三态、在线 / 未认证 etc. */
@Composable
internal fun StatusDot(color: Color, modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .size(10.dp)
            .clip(CircleShape)
            .background(color),
    )
}

/** 「数据截至 / 同步于」这类时间说明，统一弱化排版。 */
@Composable
internal fun MetaText(text: String, modifier: Modifier = Modifier, emphasize: Boolean = false) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelMedium,
        fontSize = 11.sp,
        color = if (emphasize) {
            MaterialTheme.colorScheme.tertiary
        } else {
            MaterialTheme.colorScheme.onSurfaceVariant
        },
        modifier = modifier,
    )
}

/** 校园网三态 → 本地化文案。 */
@Composable
internal fun campusStateLabel(state: CampusState): String = when (state) {
    CampusState.ONLINE -> stringResource(R.string.campusnet_online)
    CampusState.UNAUTH -> stringResource(R.string.campusnet_unauth)
    CampusState.OFFLINE -> stringResource(R.string.campusnet_offline)
}

package com.github.echoping07.kemaolife.ui

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
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.data.Entry
import com.github.echoping07.kemaolife.ui.theme.coursePalette
import com.github.echoping07.kemaolife.ui.theme.smoothShape

/**
 * 通用容器：MD3 filled card + 标题行。
 *
 * 卡片用 `surfaceContainerLowest`（浅色模式下就是纯白），页面底色是 `surface`：
 * 两者差一档，卡片边界清楚但不靠描边和阴影。
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
        shape = MaterialTheme.shapes.large,
    ) {
        Column(modifier = Modifier.padding(vertical = 8.dp)) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(start = 20.dp, end = 8.dp, top = 4.dp, bottom = 2.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.weight(1f),
                )
                action?.invoke()
            }
            Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp), content = content)
        }
    }
}

/** 主 tab 标题与设置详情页共用同一字阶，不让标题抢占内容空间。 */
@Composable
internal fun ScreenTitle(text: String, modifier: Modifier = Modifier) {
    Text(
        text = text,
        style = MaterialTheme.typography.headlineSmall,
        fontWeight = FontWeight.SemiBold,
        modifier = modifier.padding(start = 4.dp, top = 10.dp, bottom = 8.dp),
    )
}

/**
 * 今日课表条目（「更多」页用）。
 *
 * 左侧色条与课表网格里同一门课的色块同色（按课程名散列），
 * 这样两处能对上号。
 */
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

/** 小节之间的浅分隔线，避免每条都画完整卡片导致视觉噪音。 */
@Composable
internal fun ThinDivider(modifier: Modifier = Modifier) {
    HorizontalDivider(
        modifier = modifier,
        thickness = 0.5.dp,
        color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.6f),
    )
}

/** 通用「加载中」占位。 */
@Composable
internal fun LoadingBlock(modifier: Modifier = Modifier) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(vertical = 28.dp),
        horizontalArrangement = Arrangement.Center,
    ) {
        CircularProgressIndicator(modifier = Modifier.size(22.dp), strokeWidth = 2.dp)
    }
}

/** 空态 / 错误提示块。 */
@Composable
internal fun EmptyHint(text: String, modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.large)
            .background(MaterialTheme.colorScheme.surfaceContainerLowest)
            .padding(horizontal = 20.dp, vertical = 28.dp),
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
            .size(9.dp)
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

/** 校园网三态 → 本地化文案（Windows 托盘与 Android 用同一套术语，见 docs/002 §3.3）。 */
@Composable
internal fun campusStateLabel(state: CampusState): String = when (state) {
    CampusState.ONLINE -> stringResource(R.string.campusnet_online)
    CampusState.UNAUTH -> stringResource(R.string.campusnet_unauth)
    CampusState.OFFLINE -> stringResource(R.string.campusnet_offline)
}

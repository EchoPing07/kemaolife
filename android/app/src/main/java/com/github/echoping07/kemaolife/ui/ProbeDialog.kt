package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.ui.theme.appShape

/**
 * 「测试」弹窗：打开即跑一次，展示耗时、结果与实际返回值。
 *
 * 为什么要有它：课表 / 电费 / 校园网三条链路的失败原因差别很大（凭据错、网络不通、
 * 上游改字段、没绑房间……），只在界面弹一句「操作失败」没法自查。
 * 弹窗把 `*.probe` 的结构化回报摊开，用户截个图就能判断问题在哪。
 *
 * 展示内容全部来自 Go 侧已脱敏的字段（P2）：不出现 Cookie、`roomverify`、宿舍号。
 */
@Composable
internal fun ProbeDialog(state: ProbeState, onClose: () -> Unit, onRetry: () -> Unit) {
    // 弹窗只负责展示：测试由触发按钮那一侧发起（runProbe），
    // 在这里再 LaunchedEffect 触发一次会变成两轮真实请求。
    val target = state.target

    val result = state.result
    AlertDialog(
        onDismissRequest = onClose,
        confirmButton = {
            TextButton(onClick = onClose) { Text(stringResource(R.string.action_ok)) }
        },
        dismissButton = {
            TextButton(onClick = onRetry, enabled = !state.running) {
                Text(stringResource(R.string.action_test_again))
            }
        },
        shape = appShape(PageCorners.Card),
        title = {
            Column {
                Text(
                    text = stringResource(R.string.probe_title, probeName(target)),
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.SemiBold,
                )
                Text(
                    text = stringResource(R.string.probe_subtitle),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        text = {
            Column(modifier = Modifier.verticalScroll(rememberScrollState())) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    when {
                        state.running -> {
                            ExpressiveLoadingIndicator(size = 18.dp)
                            Spacer(Modifier.width(10.dp))
                            Text(
                                text = stringResource(R.string.probe_running),
                                style = MaterialTheme.typography.bodyMedium,
                            )
                        }

                        result == null -> {
                            Text(
                                text = stringResource(R.string.probe_idle),
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }

                        result.ok -> {
                            Icon(
                                imageVector = Icons.Rounded.CheckCircle,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.size(18.dp),
                            )
                            Spacer(Modifier.width(8.dp))
                            Text(
                                text = stringResource(R.string.probe_passed, result.elapsedMs),
                                style = MaterialTheme.typography.bodyMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.primary,
                            )
                        }

                        else -> {
                            Icon(
                                imageVector = TabIcons.AlertCircle,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.error,
                                modifier = Modifier.size(18.dp),
                            )
                            Spacer(Modifier.width(8.dp))
                            Text(
                                text = stringResource(R.string.probe_failed, result.elapsedMs),
                                style = MaterialTheme.typography.bodyMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.error,
                            )
                        }
                    }
                }

                if (result != null && !state.running) {
                    if (!result.ok) {
                        Spacer(Modifier.height(10.dp))
                        Text(
                            text = result.message,
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        Spacer(Modifier.height(4.dp))
                        MetaText(stringResource(R.string.probe_error_code, result.code))
                    }
                    AnimatedVisibility(visible = result.detail.isNotEmpty()) {
                        Column {
                            Spacer(Modifier.height(14.dp))
                            Card(
                                colors = CardDefaults.cardColors(
                                    containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                                ),
                                shape = appShape(PageCorners.Nested),
                            ) {
                                Column(modifier = Modifier.padding(14.dp)) {
                                    orderedDetails(result.detail).forEach { (key, value) ->
                                        Row(
                                            modifier = Modifier
                                                .fillMaxWidth()
                                                .padding(vertical = 4.dp),
                                            horizontalArrangement = Arrangement.SpaceBetween,
                                            verticalAlignment = Alignment.Top,
                                        ) {
                                            Text(
                                                text = probeFieldLabel(key),
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                            )
                                            Spacer(Modifier.width(12.dp))
                                            Text(
                                                text = probeFieldValue(key, value),
                                                style = MaterialTheme.typography.bodySmall,
                                                fontWeight = FontWeight.Medium,
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
    )
}

@Composable
private fun probeName(target: ProbeTarget): String = stringResource(
    when (target) {
        ProbeTarget.TIMETABLE -> R.string.section_edu
        ProbeTarget.ELECTRICITY -> R.string.section_power
        ProbeTarget.CAMPUS -> R.string.section_portal
    },
)

/**
 * 字段展示顺序：已知字段按「先结论、后细节」排，未知字段排最后并按名字排序。
 *
 * `detail` 由 `ProbeResult.from` 按白名单构造，这里再按固定顺序呈现，
 * 避免上游字段顺序变化让同一份结果每次打开弹窗的排版都不一样。
 */
private fun orderedDetails(detail: Map<String, String>): List<Pair<String, String>> {
    val order = listOf(
        "state", "checkedAt",
        "balance", "unit", "collectedAt", "estimatedDays", "monthUse", "dataUntil", "dayCount",
        "syncedAt", "week1Monday", "lastWeek", "lessonCount",
    )
    val known = order.mapNotNull { key -> detail[key]?.let { key to it } }
    val rest = detail.filterKeys { it !in order }.toSortedMap().map { it.key to it.value }
    return known + rest
}

/** 值的中文化 / 简短化：wire 值与完整时间戳都不该原样摊给用户看。 */
private fun probeFieldValue(key: String, value: String): String {
    if (value.isBlank()) return "—"
    return when (key) {
        "state" -> when (CampusState.from(value)) {
            CampusState.ONLINE -> "已在线"
            CampusState.UNAUTH -> "未认证"
            CampusState.OFFLINE -> "不在校园网"
        }
        // RFC3339 → MM/dd HH:mm，与页面上的「同步于 / 数据截至」同一格式。
        "syncedAt", "collectedAt", "checkedAt" -> shortStamp(value)
        // 纯日期 → yyyy/M/d，与课表日期同一格式。
        "week1Monday", "dataUntil" -> if (value.length == 10) prettyDate(value) else value
        else -> value
    }
}

/** probe 回报字段 → 中文标签；未知字段原样显示，方便上游加字段时不用改客户端。 */
@Composable
private fun probeFieldLabel(key: String): String = when (key) {
    "syncedAt" -> stringResource(R.string.probe_field_synced_at)
    "week1Monday" -> stringResource(R.string.probe_field_week1)
    "lastWeek" -> stringResource(R.string.probe_field_last_week)
    "lessonCount" -> stringResource(R.string.probe_field_lesson_count)
    "balance" -> stringResource(R.string.probe_field_balance)
    "unit" -> stringResource(R.string.probe_field_unit)
    "collectedAt" -> stringResource(R.string.data_collected_at_plain)
    "estimatedDays" -> stringResource(R.string.metric_estimated_days)
    "monthUse" -> stringResource(R.string.metric_month_usage)
    "dataUntil" -> stringResource(R.string.probe_field_data_until)
    "dayCount" -> stringResource(R.string.probe_field_day_count)
    "state" -> stringResource(R.string.probe_field_state)
    "checkedAt" -> stringResource(R.string.probe_field_checked_at)
    else -> key
}

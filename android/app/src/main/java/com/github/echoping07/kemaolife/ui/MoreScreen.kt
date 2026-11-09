package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.Crossfade
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import com.github.echoping07.kemaolife.ui.theme.smoothShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.ElectricityReport
import java.util.Locale

/**
 * tab2 · 更多：今日概览 + 电费 + 校园网（docs/002 §5.3）。
 *
 * 三块都是「只读 + 手动刷新」：Android 不做后台、不做定时提醒，
 * 所以每次进入本页才拉数据，离开即停。
 *
 * 排版按「一眼能读」组织：余额是整个页面的视觉重心（headline 大数字），
 * 其余信息降级为 label；同类信息不再各包一张描边卡片，而是统一 filled card +
 * 内部横向分隔（M3 的 card 本身已经有容器色，再套描边会显得廉价）。
 */
@Composable
internal fun MoreScreen(
    state: AppState,
    viewModel: AppViewModel,
    onGoSettings: () -> Unit,
) {
    // 进入本页就并行拉一次电费与判态（Android 不做后台，离开即停）。
    // 放在这里而不是只绑在底栏 onClick 上：tab 可能因重建直接恢复到本页。
    LaunchedEffect(Unit) {
        if (state.electricity == null) viewModel.refreshElectricity()
        if (state.campus == null) viewModel.refreshCampusStatus()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        ScreenTitle(stringResource(R.string.tab_more))

        if (state.needsSetup) {
            SetupCard(onGoSettings)
            Spacer(Modifier.height(12.dp))
        }

        TodayOverview(state)

        Spacer(Modifier.height(12.dp))
        ElectricityCard(state, viewModel)

        Spacer(Modifier.height(12.dp))
        CampusCard(state, viewModel)

        Spacer(Modifier.height(28.dp))
    }
}

@Composable
private fun SetupCard(onGoSettings: () -> Unit) {
    SectionCard(title = stringResource(R.string.setup_missing_title)) {
        Text(
            text = stringResource(R.string.setup_missing_body),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(12.dp))
        FilledTonalButton(onClick = onGoSettings) {
            Text(stringResource(R.string.setup_missing_action))
        }
    }
}

@Composable
private fun TodayOverview(state: AppState) {
    val day: DaySchedule? = state.todayDay
    val title = stringResource(R.string.section_today)
    SectionCard(title = title) {
        when {
            day == null -> PlaceholderRow(
                text = state.timetableError ?: stringResource(R.string.loading),
            )

            !day.inSemester -> PlaceholderRow(stringResource(R.string.timetable_not_in_semester))

            day.entries.isEmpty() -> PlaceholderRow(stringResource(R.string.timetable_no_class))

            else -> day.entries.forEachIndexed { index, entry ->
                if (index > 0) ThinDivider()
                EntryRow(entry)
            }
        }
    }
}

@Composable
private fun PlaceholderRow(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(vertical = 6.dp),
    )
}

@Composable
private fun ElectricityCard(state: AppState, viewModel: AppViewModel) {
    val report = state.electricity
    val unit = report?.status?.unit?.ifBlank { "度" } ?: "度"

    SectionCard(
        title = stringResource(R.string.section_electricity),
        action = {
            RefreshAction(
                loading = state.electricityLoading,
                onClick = { viewModel.refreshElectricity() },
            )
        },
    ) {
        when {
            state.electricityLoading && report == null -> LoadingBlock()

            report == null -> PlaceholderRow(stringResource(R.string.electricity_not_queried))

            else -> ElectricityBody(report, unit)
        }
    }
}

@Composable
private fun ElectricityBody(report: ElectricityReport, unit: String) {
    val status = report.status

    // 余额是这一页唯一的大数字：先给数字，再给单位，单位刻意小一号。
    Row(verticalAlignment = Alignment.Bottom) {
        AnimatedContent(
            targetState = formatNumber(status.balance),
            transitionSpec = { fadeIn(tween(260)) togetherWith fadeOut(tween(140)) },
            label = "balanceChange",
        ) { balance ->
            Text(
                text = balance,
                style = MaterialTheme.typography.displaySmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.primary,
            )
        }
        Spacer(Modifier.width(6.dp))
        Text(
            text = unit,
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(bottom = 6.dp),
        )
    }

    // 「数据截至」必须出现：电表约有 1 天延迟，不能当实时值（R2 要求 4）。
    if (status.collectedAt.isNotBlank()) {
        Spacer(Modifier.height(2.dp))
        MetaText(
            text = stringResource(R.string.data_collected_at, shortStamp(status.collectedAt)),
            emphasize = true,
        )
    }

    Spacer(Modifier.height(14.dp))
    MetricGrid(
        left = stringResource(R.string.metric_estimated_days) to
            (status.estimatedDays?.let { stringResource(R.string.days_value, it) }
                ?: stringResource(R.string.value_unknown)),
        right = stringResource(R.string.metric_month_usage) to
            stringResource(R.string.usage_value, formatNumber(report.usage.monthUse), unit),
    )

    if (status.dailyAvg != null) {
        Spacer(Modifier.height(6.dp))
        Text(
            text = stringResource(R.string.daily_avg, formatNumber(status.dailyAvg), unit),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }

    if (report.usage.dataUntil.isNotBlank()) {
        Spacer(Modifier.height(2.dp))
        MetaText(stringResource(R.string.usage_until, report.usage.dataUntil))
    }

    if (report.usage.days.isNotEmpty()) {
        Spacer(Modifier.height(14.dp))
        UsageSparkline(report.usage.days.map { it.use })
    }
}

/** 两列指标：label 在上、value 在下，避免长句换行把布局顶乱。 */
@Composable
private fun MetricGrid(left: Pair<String, String>, right: Pair<String, String>) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.medium)
            .background(MaterialTheme.colorScheme.surfaceContainerHigh)
            .padding(horizontal = 14.dp, vertical = 12.dp),
    ) {
        MetricCell(left.first, left.second, Modifier.weight(1f))
        Box(
            modifier = Modifier
                .width(1.dp)
                .height(32.dp)
                .background(MaterialTheme.colorScheme.outlineVariant),
        )
        MetricCell(right.first, right.second, Modifier.weight(1f), paddingStart = 14.dp)
    }
}

@Composable
private fun MetricCell(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    paddingStart: androidx.compose.ui.unit.Dp = 0.dp,
) {
    Column(modifier = modifier.padding(start = paddingStart)) {
        Text(
            text = value,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
        )
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/**
 * 近几日的用量柱：只是把 Go 侧已排序好的 `days[]` 画出来，不做任何统计（AGENTS A3）。
 *
 * 用等宽柱 + 最高值归一，柱高一眼能看出「今天用得特别多」。
 */
@Composable
private fun UsageSparkline(values: List<Double>) {
    val max = values.maxOrNull()?.takeIf { it > 0.0 } ?: return
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(48.dp),
        horizontalArrangement = Arrangement.spacedBy(3.dp),
        verticalAlignment = Alignment.Bottom,
    ) {
        values.forEach { v ->
            val ratio by animateFloatAsState(
                targetValue = (v / max).toFloat().coerceIn(0.06f, 1f),
                animationSpec = tween(420),
                label = "usageBar",
            )
            Box(
                modifier = Modifier
                    .weight(1f)
                    .height((48 * ratio).dp)
                    .clip(smoothShape(3.dp))
                    .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.35f)),
            )
        }
    }
}

@Composable
private fun CampusCard(state: AppState, viewModel: AppViewModel) {
    val campus = state.campus
    val busy = state.campusLoading || state.campusActing != null
    var showDetails by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf(false) }

    SectionCard(
        title = stringResource(R.string.section_campusnet),
        action = {
            RefreshAction(
                loading = state.campusLoading,
                contentDescription = stringResource(R.string.action_probe),
                onClick = { viewModel.refreshCampusStatus() },
            )
        },
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            val tint = when (campus?.state) {
                CampusState.ONLINE -> MaterialTheme.colorScheme.primary
                CampusState.UNAUTH -> MaterialTheme.colorScheme.tertiary
                else -> MaterialTheme.colorScheme.outline
            }
            StatusDot(color = tint)
            Spacer(Modifier.width(8.dp))
            Crossfade(targetState = campus?.state, animationSpec = tween(240), label = "campusState") {
                Text(
                    text = campus?.let { campusStateLabel(it.state) }
                        ?: stringResource(R.string.loading),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = tint,
                )
            }
            if (busy) {
                Spacer(Modifier.width(10.dp))
                ExpressiveLoadingIndicator(size = 16.dp)
            }
        }

        if (campus?.checkedAt?.isNotBlank() == true) {
            Spacer(Modifier.height(2.dp))
            MetaText(stringResource(R.string.campus_probed_at, shortStamp(campus.checkedAt)))
        }

        // offline 是正常状态而非认证失败：蜂窝数据下必然 offline，要给出说明而不是报错。
        if (campus?.state == CampusState.OFFLINE) {
            Spacer(Modifier.height(6.dp))
            Text(
                text = stringResource(R.string.campusnet_offline_hint),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        if (campus?.errorMsg?.isNotBlank() == true) {
            Spacer(Modifier.height(6.dp))
            Text(
                text = campus.errorMsg.lineSequence().first().substringBefore('；').substringBefore(';').trim(),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        val details = listOf(
            stringResource(R.string.campus_bras_name) to campus?.brasName.orEmpty(),
            stringResource(R.string.campus_bras_address) to campus?.brasAddress.orEmpty(),
            stringResource(R.string.campus_vlan) to campus?.vlan.orEmpty(),
            stringResource(R.string.campus_device_ip) to campus?.deviceIp.orEmpty(),
            stringResource(R.string.campus_mac) to campus?.mac.orEmpty(),
        ).filter { it.second.isNotBlank() }
        if (details.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            Text(
                text = stringResource(R.string.campus_access_details),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.clickable { showDetails = !showDetails }.padding(vertical = 4.dp),
            )
            AnimatedVisibility(visible = showDetails) {
                Column(modifier = Modifier.fillMaxWidth().padding(top = 6.dp)) {
                    details.forEach { (label, value) ->
                        Row(modifier = Modifier.fillMaxWidth().padding(vertical = 5.dp)) {
                            Text(
                                text = label,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.width(96.dp),
                            )
                            Text(
                                text = value,
                                style = MaterialTheme.typography.bodySmall,
                                modifier = Modifier.weight(1f),
                            )
                        }
                    }
                }
            }
        }

        Spacer(Modifier.height(14.dp))
        // 按钮按当前状态给推荐动作：未认证时只有「登录」是实心，已在线时「下线」实心。
        // 与状态无关的动作降级为描边，避免三个同等重的按钮让人猜该点哪个。
        val enabled = state.campusActionsEnabled && state.campusActing == null
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            CampusButton(
                label = stringResource(R.string.action_login),
                filled = campus?.state == CampusState.UNAUTH,
                enabled = enabled && campus?.state == CampusState.UNAUTH,
                onClick = { viewModel.campusAction(CampusAction.LOGIN) },
                modifier = Modifier.weight(1f),
            )
            CampusButton(
                label = stringResource(R.string.action_relogin),
                filled = false,
                enabled = enabled,
                onClick = { viewModel.campusAction(CampusAction.RELOGIN) },
                modifier = Modifier.weight(1f),
            )
            CampusButton(
                label = stringResource(R.string.action_logout),
                filled = campus?.state == CampusState.ONLINE,
                enabled = enabled && campus?.state == CampusState.ONLINE,
                onClick = { viewModel.campusAction(CampusAction.LOGOUT) },
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun CampusButton(
    label: String,
    filled: Boolean,
    enabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    if (filled) {
        Button(onClick = onClick, enabled = enabled, shape = CircleShape, modifier = modifier) {
            Text(label, maxLines = 1)
        }
    } else {
        OutlinedButton(onClick = onClick, enabled = enabled, shape = CircleShape, modifier = modifier) {
            Text(label, maxLines = 1)
        }
    }
}

/** 卡片右上角的刷新 / 探测按钮：加载中显示小转圈，避免重复点击。 */
@Composable
internal fun RefreshAction(
    loading: Boolean,
    onClick: () -> Unit,
    contentDescription: String? = null,
) {
    Box(modifier = Modifier.size(44.dp), contentAlignment = Alignment.Center) {
        if (loading) {
            ExpressiveLoadingIndicator(size = 20.dp)
        } else {
            TextButton(onClick = onClick) {
                Icon(
                    imageVector = Icons.Rounded.Refresh,
                    contentDescription = contentDescription ?: stringResource(R.string.action_refresh),
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(20.dp),
                )
            }
        }
    }
}

/** 数值格式化：最多两位小数，去掉多余的 0，保证 35.99 / 36 都能正常显示。 */
internal fun formatNumber(value: Double): String {
    return if (value == value.toLong().toDouble()) {
        value.toLong().toString()
    } else {
        String.format(Locale.US, "%.2f", value)
    }
}

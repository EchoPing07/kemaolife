package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedCard
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
import com.github.echoping07.kemaolife.data.DaySchedule
import java.util.Locale

/**
 * tab2 · 更多：当天课表概览 + 电费 + 校园网（docs/002 §5.3）。
 *
 * 三块都是「只读 + 手动刷新」：Android 不做后台、不做定时提醒，
 * 所以每次进入本页才拉数据，离开即停。
 */
@Composable
internal fun MoreScreen(
    state: AppState,
    viewModel: AppViewModel,
    onGoSettings: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        Spacer(Modifier.height(12.dp))

        if (state.needsSetup) {
            OutlinedCard(modifier = Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp)) {
                    Text(
                        text = "还没有配置账号",
                        style = MaterialTheme.typography.titleSmall,
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(
                        text = "填写学号与密码后才能查询课表与电费。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Spacer(Modifier.height(10.dp))
                    FilledTonalButton(onClick = onGoSettings) {
                        Text(stringResource(R.string.tab_settings))
                    }
                }
            }
            Spacer(Modifier.height(16.dp))
        }

        // ---- 今日概览 ----
        SectionTitle(stringResource(R.string.section_today))
        TodayOverview(state)

        Spacer(Modifier.height(20.dp))

        // ---- 电费 ----
        SectionTitle(stringResource(R.string.section_electricity))
        ElectricityBlock(state, viewModel)

        Spacer(Modifier.height(20.dp))

        // ---- 校园网 ----
        SectionTitle(stringResource(R.string.section_campusnet))
        CampusBlock(state, viewModel)

        Spacer(Modifier.height(28.dp))
    }
}

@Composable
private fun TodayOverview(state: AppState) {
    val day: DaySchedule? = state.todayDay
    when {
        day == null -> EmptyHint(stringResource(R.string.bridge_not_ready))
        !day.inSemester -> EmptyHint(stringResource(R.string.timetable_not_in_semester))
        day.entries.isEmpty() -> EmptyHint(stringResource(R.string.timetable_no_class))
        else -> OutlinedCard(modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp)) {
                day.entries.forEachIndexed { index, entry ->
                    if (index > 0) ThinDivider()
                    EntryRow(entry)
                }
            }
        }
    }
}

@Composable
private fun ElectricityBlock(state: AppState, viewModel: AppViewModel) {
    val report = state.electricity
    OutlinedCard(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "剩余电量",
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                )
                TextButton(onClick = { viewModel.refreshElectricity() }) {
                    Text(stringResource(R.string.action_refresh))
                }
            }

            if (state.electricityLoading && report == null) {
                LoadingBlock()
                return@Column
            }
            if (report == null) {
                Text(
                    text = "尚未查询",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                return@Column
            }

            val status = report.status
            Text(
                text = stringResource(
                    R.string.balance_unit,
                    formatNumber(status.balance),
                    status.unit.ifBlank { "度" },
                ),
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(Modifier.height(4.dp))
            // 「数据截至」必须出现：电表约有 1 天延迟，不能当实时值（R2 要求 4）。
            if (status.collectedAt.isNotBlank()) {
                Text(
                    text = stringResource(R.string.data_collected_at, status.collectedAt),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.tertiary,
                )
            }
            Spacer(Modifier.height(10.dp))
            Text(
                text = status.estimatedDays
                    ?.let { stringResource(R.string.estimated_days, it) }
                    ?: stringResource(R.string.estimated_days_unknown),
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                text = stringResource(
                    R.string.month_usage,
                    formatNumber(report.usage.monthUse),
                    status.unit.ifBlank { "度" },
                ),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (report.usage.dataUntil.isNotBlank()) {
                Text(
                    text = "用量统计至 ${report.usage.dataUntil}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun CampusBlock(state: AppState, viewModel: AppViewModel) {
    val campus = state.campus
    OutlinedCard(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(16.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    if (state.campusLoading || state.campusActing != null) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(16.dp),
                            strokeWidth = 2.dp,
                        )
                        Spacer(Modifier.size(8.dp))
                    }
                    Text(
                        text = campus?.let { campusStateLabel(it.state) }
                            ?: stringResource(R.string.loading),
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                        color = when (campus?.state) {
                            CampusState.ONLINE -> MaterialTheme.colorScheme.primary
                            CampusState.UNAUTH -> MaterialTheme.colorScheme.tertiary
                            else -> MaterialTheme.colorScheme.onSurfaceVariant
                        },
                    )
                }
                TextButton(onClick = { viewModel.refreshCampusStatus() }) {
                    Text(stringResource(R.string.action_probe))
                }
            }

            if (campus?.checkedAt?.isNotBlank() == true) {
                Text(
                    text = "探测于 ${campus.checkedAt}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
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

            Spacer(Modifier.height(12.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(
                    onClick = { viewModel.campusAction(CampusAction.LOGIN) },
                    enabled = state.campusActionsEnabled && state.campusActing == null,
                ) { Text(stringResource(R.string.action_login)) }

                OutlinedButton(
                    onClick = { viewModel.campusAction(CampusAction.LOGOUT) },
                    enabled = state.campusActionsEnabled && state.campusActing == null,
                ) { Text(stringResource(R.string.action_logout)) }

                OutlinedButton(
                    onClick = { viewModel.campusAction(CampusAction.RELOGIN) },
                    enabled = state.campusActionsEnabled && state.campusActing == null,
                ) { Text(stringResource(R.string.action_relogin)) }
            }
        }
    }
}

/** 数值格式化：最多两位小数，去掉多余的 0，保证 35.99 / 36 都能正常显示。 */
private fun formatNumber(value: Double): String {
    return if (value == value.toLong().toDouble()) {
        value.toLong().toString()
    } else {
        String.format(Locale.US, "%.2f", value)
    }
}

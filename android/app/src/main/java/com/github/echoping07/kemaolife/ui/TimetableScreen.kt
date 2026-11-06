package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.WeekSchedule
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

/**
 * tab1 · 课表（docs/002 §5.3）。
 *
 * 三段：今日 / 本周 / 选日期。所有视图都消费同一份 Go 侧计算结果，
 * 刷新失败时显示「离线数据（同步于 …）」，而不是把旧数据当新数据（R1 验收）。
 */
@Composable
internal fun TimetableScreen(state: AppState, viewModel: AppViewModel) {
    var section by remember { mutableIntStateOf(0) }

    Column(modifier = Modifier.fillMaxSize()) {
        TabRow(selectedTabIndex = section) {
            listOf(
                stringResource(R.string.timetable_today),
                stringResource(R.string.timetable_week),
                stringResource(R.string.timetable_pick_date),
            ).forEachIndexed { index, label ->
                Tab(
                    selected = section == index,
                    onClick = { section = index },
                    text = { Text(label) },
                )
            }
        }

        when (section) {
            0 -> TodaySection(state, viewModel)
            1 -> WeekSection(state, viewModel)
            else -> DateSection(state, viewModel)
        }
    }
}

@Composable
private fun TodaySection(state: AppState, viewModel: AppViewModel) {
    val day = state.todayDay
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        Spacer(Modifier.height(12.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column {
                Text(
                    text = day?.let { "第 ${it.week} 周 · 周${it.weekdayCn}" } ?: "",
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(
                    text = day?.date ?: "",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            FilledTonalButton(onClick = { viewModel.refreshTimetable() }) {
                Text(stringResource(R.string.action_refresh))
            }
        }

        Spacer(Modifier.height(8.dp))
        // 数据时间：stale 时必须说明「这是旧数据」（docs/002 §2 stale 语义）。
        if (state.today?.stale == true) {
            Text(
                text = stringResource(R.string.data_stale, state.today.syncedAt),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.tertiary,
            )
        } else if (!state.today?.syncedAt.isNullOrBlank()) {
            Text(
                text = stringResource(R.string.data_synced_at, state.today.syncedAt),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        Spacer(Modifier.height(8.dp))
        when {
            state.timetableLoading && day == null -> LoadingBlock()
            // 失败有具体原因就显示原因；只有拿到过数据但当天没课才显示「当天没课」。
            state.timetableError != null -> EmptyHint(state.timetableError!!)
            day == null -> EmptyHint(stringResource(R.string.bridge_not_ready))
            !day.inSemester -> EmptyHint(stringResource(R.string.timetable_not_in_semester))
            day.entries.isEmpty() -> EmptyHint(stringResource(R.string.timetable_no_class))
            else -> day.entries.forEachIndexed { index, entry ->
                if (index > 0) ThinDivider()
                EntryRow(entry)
            }
        }
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
private fun WeekSection(state: AppState, viewModel: AppViewModel) {
    // 首次进入本周视图时按「当周」拉一次（week = 0 交给 Go 判定当周）。
    var loaded by remember { mutableStateOf(false) }
    if (!loaded) {
        loaded = true
        viewModel.loadWeek(0)
    }

    val week = state.week
    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 10.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            OutlinedButton(
                onClick = { week?.let { viewModel.loadWeek(it.week - 1) } },
                enabled = week != null && week.week > 1,
            ) { Text(stringResource(R.string.timetable_prev_week)) }

            Text(
                text = week?.let { "第 ${it.week} 周" } ?: stringResource(R.string.loading),
                style = MaterialTheme.typography.titleMedium,
            )

            OutlinedButton(onClick = { week?.let { viewModel.loadWeek(it.week + 1) } }) {
                Text(stringResource(R.string.timetable_next_week))
            }
        }

        if (state.weekStale) {
            Text(
                text = stringResource(R.string.data_stale, state.weekSyncedAt),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.tertiary,
                modifier = Modifier.padding(horizontal = 16.dp),
            )
        }

        if (week == null || state.weekLoading) {
            LoadingBlock()
            return@Column
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(
                start = 16.dp, end = 16.dp, top = 8.dp, bottom = 24.dp,
            ),
        ) {
            items(week.days) { day -> WeekDayBlock(day) }
        }
    }
}

@Composable
private fun WeekDayBlock(day: DaySchedule) {
    Column(modifier = Modifier.fillMaxWidth().padding(bottom = 6.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "周${day.weekdayCn}",
                style = MaterialTheme.typography.titleSmall,
                modifier = Modifier.padding(end = 8.dp),
            )
            Text(
                text = day.date,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        when {
            !day.inSemester -> Text(
                text = stringResource(R.string.timetable_not_in_semester),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(vertical = 6.dp),
            )

            day.entries.isEmpty() -> Text(
                text = stringResource(R.string.timetable_no_class),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(vertical = 6.dp),
            )

            else -> day.entries.forEach { EntryRow(it) }
        }
        ThinDivider()
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DateSection(state: AppState, viewModel: AppViewModel) {
    var showPicker by remember { mutableStateOf(false) }
    val picked = state.todayDay

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        Spacer(Modifier.height(12.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Button(onClick = { showPicker = true }) {
                Text(stringResource(R.string.timetable_pick_date))
            }
            Text(
                text = picked?.date ?: "",
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        Spacer(Modifier.height(12.dp))
        when {
            state.timetableLoading -> LoadingBlock()
            state.timetableError != null -> EmptyHint(state.timetableError!!)
            picked == null -> EmptyHint(stringResource(R.string.bridge_not_ready))
            !picked.inSemester -> EmptyHint(stringResource(R.string.timetable_not_in_semester))
            picked.entries.isEmpty() -> EmptyHint(stringResource(R.string.timetable_no_class))
            else -> picked.entries.forEachIndexed { index, entry ->
                if (index > 0) ThinDivider()
                EntryRow(entry)
            }
        }
        Spacer(Modifier.height(24.dp))
    }

    if (showPicker) {
        val pickerState = rememberDatePickerState()
        DatePickerDialog(
            onDismissRequest = { showPicker = false },
            confirmButton = {
                TextButton(onClick = {
                    // DatePicker 给的是 UTC 毫秒：必须按 UTC 取日历日再转 LocalDate，
                    // 否则在东八区会把「今天选的日期」变成前一天。
                    pickerState.selectedDateMillis?.let { millis ->
                        val date = Instant.ofEpochMilli(millis)
                            .atZone(ZoneId.of("UTC"))
                            .toLocalDate()
                        viewModel.loadDate(date)
                    }
                    showPicker = false
                }) { Text(stringResource(R.string.action_ok)) }
            },
            dismissButton = {
                TextButton(onClick = { showPicker = false }) {
                    Text(stringResource(android.R.string.cancel))
                }
            },
        ) {
            DatePicker(state = pickerState)
        }
    }
}

@Composable
internal fun EmptyHint(text: String) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        color = MaterialTheme.colorScheme.surfaceVariant,
        shape = MaterialTheme.shapes.medium,
    ) {
        Text(
            text = text,
            modifier = Modifier.padding(20.dp),
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

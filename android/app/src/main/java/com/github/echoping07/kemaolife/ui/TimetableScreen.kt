package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.rounded.DateRange
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.Entry
import com.github.echoping07.kemaolife.ui.theme.coursePalette
import java.time.Instant
import java.time.ZoneId

/** 网格行高：12 小节，纵向可滚动。 */
private val RowHeight = 54.dp

/** 左侧节次栏宽度。 */
private val GutterWidth = 20.dp

private val WEEKDAY_CN = listOf("一", "二", "三", "四", "五", "六", "日")

/**
 * tab1 · 课表（docs/002 §5.3）。
 *
 * 一屏承担三件事：
 *  1. 头部两行：选中日 + 周次 + 数据时间 + 刷新 / 选日期 / 切周；
 *  2. 当周 7 天网格（12 小节 × 7 列），一眼看完全周；
 *  3. 网格下方是选中日的完整条目（网格列太窄，长课程名 / 地点放不下）。
 *
 * 切周只发一次 `timetable.week`（week=0 表示当周，由 Go 侧解析）；
 * 选日期先 `timetable.date` 拿到该日所属周次，再拉整周 —— 周次一律由 Go 侧算（AGENTS A3）。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun TimetableScreen(state: AppState, viewModel: AppViewModel) {
    var showPicker by remember { mutableStateOf(false) }
    val week = state.week
    val selected = state.selectedDay

    Column(modifier = Modifier.fillMaxSize()) {
        Header(
            selected = selected,
            loading = state.weekLoading,
            stale = state.weekStale,
            syncedAt = state.weekSyncedAt,
            isCurrentWeek = week?.days?.any { it.date == state.todayDate } == true,
            onPrevWeek = { week?.let { viewModel.loadWeek(it.week - 1) } },
            onNextWeek = { week?.let { viewModel.loadWeek(it.week + 1) } },
            onBackToCurrent = { viewModel.loadWeek(0) },
            onRefresh = { viewModel.refreshWeek() },
            onPickDate = { showPicker = true },
        )

        WeekStrip(
            days = week?.days.orEmpty(),
            selectedDate = state.selectedDate,
            todayDate = state.todayDate,
            onSelect = { date -> viewModel.selectDate(date) },
        )

        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(bottom = 24.dp),
        ) {
            when {
                week == null && state.weekLoading -> {
                    LoadingBlock()
                    EmptyHint(
                        text = state.timetableError ?: stringResource(R.string.loading),
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                }

                week == null -> EmptyHint(
                    text = state.timetableError ?: stringResource(R.string.bridge_not_ready),
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                )

                week.days.none { it.inSemester } -> EmptyHint(
                    text = stringResource(R.string.timetable_not_in_semester),
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                )

                else -> {
                    WeekGrid(days = week.days, selectedDate = state.selectedDate)
                    if (selected != null) SelectedDayCard(selected)
                }
            }
        }
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
                        viewModel.pickDate(date)
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

/**
 * 头部两行，尽量给网格让位：
 *   第一行：选中日大日期 + 刷新 / 选日期；
 *   第二行：周次 · 星期 + 数据时间 + 回到本周（非本周时）+ 切周箭头。
 *
 * 刻意不放 TopAppBar：这个页面每天要看的区域是网格本身，标题栏纯属占地方。
 */
@Composable
private fun Header(
    selected: DaySchedule?,
    loading: Boolean,
    stale: Boolean,
    syncedAt: String,
    isCurrentWeek: Boolean,
    onPrevWeek: () -> Unit,
    onNextWeek: () -> Unit,
    onBackToCurrent: () -> Unit,
    onRefresh: () -> Unit,
    onPickDate: () -> Unit,
) {
    Column(modifier = Modifier.padding(start = 20.dp, end = 8.dp, top = 4.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = selected?.date?.let(::prettyDate) ?: stringResource(R.string.tab_timetable),
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            Box(modifier = Modifier.size(40.dp), contentAlignment = Alignment.Center) {
                if (loading) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    IconButton(onClick = onRefresh, modifier = Modifier.size(40.dp)) {
                        Icon(
                            imageVector = Icons.Rounded.Refresh,
                            contentDescription = stringResource(R.string.action_refresh),
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(22.dp),
                        )
                    }
                }
            }
            IconButton(onClick = onPickDate, modifier = Modifier.size(40.dp)) {
                Icon(
                    imageVector = Icons.Rounded.DateRange,
                    contentDescription = stringResource(R.string.timetable_pick_date),
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(22.dp),
                )
            }
        }

        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(end = 4.dp),
        ) {
            val weekText = buildList {
                if (selected != null && selected.week > 0) add(stringResource(R.string.week_only, selected.week))
                if (selected != null && selected.weekdayCn.isNotBlank()) {
                    add(stringResource(R.string.weekday_cn, selected.weekdayCn))
                }
            }.joinToString("  ·  ")
            Text(
                text = weekText,
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Spacer(Modifier.width(10.dp))
            // 数据时间必须能追溯到具体时刻（docs/002 §2 stale 语义）。
            Box(modifier = Modifier.weight(1f)) {
                when {
                    stale -> MetaText(
                        text = stringResource(R.string.data_stale_short, shortStamp(syncedAt)),
                        emphasize = true,
                    )
                    syncedAt.isNotBlank() -> MetaText(
                        text = stringResource(R.string.data_synced_at, shortStamp(syncedAt)),
                    )
                }
            }
            if (!isCurrentWeek) {
                TextButton(
                    onClick = onBackToCurrent,
                    contentPadding = PaddingValues(horizontal = 8.dp),
                ) {
                    Text(
                        text = stringResource(R.string.week_current),
                        style = MaterialTheme.typography.labelMedium,
                        maxLines = 1,
                    )
                }
            }
            IconButton(onClick = onPrevWeek, modifier = Modifier.size(30.dp)) {
                Icon(
                    Icons.AutoMirrored.Rounded.KeyboardArrowLeft,
                    contentDescription = stringResource(R.string.timetable_prev_week),
                    modifier = Modifier.size(22.dp),
                )
            }
            IconButton(onClick = onNextWeek, modifier = Modifier.size(30.dp)) {
                Icon(
                    Icons.AutoMirrored.Rounded.KeyboardArrowRight,
                    contentDescription = stringResource(R.string.timetable_next_week),
                    modifier = Modifier.size(22.dp),
                )
            }
        }
    }
}

/** 当周 7 天选择条：一行「星期 + 日号」，点哪天就在下面看哪天的详情。 */
@Composable
private fun WeekStrip(
    days: List<DaySchedule>,
    selectedDate: String,
    todayDate: String,
    onSelect: (String) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 8.dp, end = 12.dp, top = 2.dp),
    ) {
        Spacer(Modifier.width(GutterWidth))
        for (i in 0 until 7) {
            val day = days.getOrNull(i)
            val isSelected = day != null && day.date == selectedDate
            val isToday = day != null && day.date == todayDate
            Column(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(12.dp))
                    .clickable(enabled = day != null) { day?.let { onSelect(it.date) } }
                    .padding(vertical = 2.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    text = day?.weekdayCn ?: WEEKDAY_CN[i],
                    style = MaterialTheme.typography.labelMedium,
                    color = if (isSelected) {
                        MaterialTheme.colorScheme.primary
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    },
                )
                Spacer(Modifier.height(3.dp))
                Box(
                    modifier = Modifier
                        .size(28.dp)
                        .clip(CircleShape)
                        .background(
                            when {
                                isSelected -> MaterialTheme.colorScheme.primary
                                isToday -> MaterialTheme.colorScheme.primaryContainer
                                else -> Color.Transparent
                            },
                        ),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = day?.date?.substringAfterLast('-')?.trimStart('0') ?: "·",
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = if (isSelected || isToday) FontWeight.Bold else FontWeight.Normal,
                        color = when {
                            isSelected -> MaterialTheme.colorScheme.onPrimary
                            isToday -> MaterialTheme.colorScheme.onPrimaryContainer
                            else -> MaterialTheme.colorScheme.onSurface
                        },
                    )
                }
            }
        }
    }
}

/**
 * 12 小节 × 7 天的课表网格。
 *
 * 行号是教务的小节号（1..12），块高由 `periods` 展开后的跨度决定；
 * 同一天撞课时按 lane 横向平分列宽（见 `CourseLayout.layoutCourseBlocks`）。
 */
@Composable
private fun WeekGrid(days: List<DaySchedule>, selectedDate: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(end = 10.dp, top = 4.dp)
            .height(RowHeight * TIMETABLE_ROWS),
    ) {
        // 左侧小节号：只有数字，不画额外背景，避免和课程块抢注意力。
        Column(modifier = Modifier.width(GutterWidth)) {
            for (row in 1..TIMETABLE_ROWS) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(RowHeight),
                    contentAlignment = Alignment.TopCenter,
                ) {
                    Text(
                        text = row.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        fontSize = 10.sp,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        for (col in 0 until 7) {
            val day = days.getOrNull(col)
            val isSelected = day != null && day.date == selectedDate
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .background(
                        if (isSelected) {
                            MaterialTheme.colorScheme.surfaceContainerLow
                        } else {
                            Color.Transparent
                        },
                    ),
            ) {
                RowGuideLines()
                val blocks = remember(day?.date, day?.entries) {
                    layoutCourseBlocks(day?.entries.orEmpty())
                }
                DayColumn(blocks = blocks, rowHeight = RowHeight)
                // 选中列只在最上面画一条短线做标记：整列染色会把课表压暗。
                if (isSelected) {
                    Box(
                        modifier = Modifier
                            .padding(horizontal = 3.dp)
                            .fillMaxWidth()
                            .height(3.dp)
                            .clip(RoundedCornerShape(2.dp))
                            .background(MaterialTheme.colorScheme.primary),
                    )
                }
            }
        }
    }
}

/** 每小节一条浅横线，让 12 行可数。 */
@Composable
private fun RowGuideLines() {
    Column(Modifier.fillMaxSize()) {
        for (row in 1..TIMETABLE_ROWS) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(RowHeight)
                    .border(
                        width = 0.5.dp,
                        color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.30f),
                        shape = RoundedCornerShape(0.dp),
                    ),
            )
        }
    }
}

/** 一天的课程块：按 lane 平分宽度、按小节号定 y。 */
@Composable
private fun DayColumn(blocks: List<CourseBlock>, rowHeight: Dp) {
    Layout(
        modifier = Modifier.fillMaxSize(),
        content = {
            for (block in blocks) {
                val palette = coursePalette(block.entry.name)
                Box(
                    modifier = Modifier
                        .padding(1.5.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .background(palette.container)
                        .padding(horizontal = 3.dp, vertical = 3.dp),
                ) {
                    CourseCell(block.entry, palette.onContainer)
                }
            }
        },
    ) { measurables, constraints ->
        val rowPx = rowHeight.roundToPx()
        val placeables = measurables.mapIndexed { index, measurable ->
            val block = blocks[index]
            val laneWidth = constraints.maxWidth / block.lanes.coerceAtLeast(1)
            measurable.measure(
                Constraints.fixed(width = laneWidth, height = rowPx * block.spanRows),
            )
        }
        layout(constraints.maxWidth, rowPx * TIMETABLE_ROWS) {
            placeables.forEachIndexed { index, placeable ->
                val block = blocks[index]
                val laneWidth = constraints.maxWidth / block.lanes.coerceAtLeast(1)
                placeable.place(x = block.lane * laneWidth, y = rowPx * block.startRow)
            }
        }
    }
}

@Composable
private fun CourseCell(entry: Entry, textColor: Color) {
    Column {
        val start = entry.time.substringBefore('~')
        if (start.isNotBlank()) {
            Text(
                text = start,
                style = MaterialTheme.typography.labelSmall,
                fontSize = 9.sp,
                lineHeight = 11.sp,
                fontWeight = FontWeight.SemiBold,
                color = textColor,
                maxLines = 1,
            )
        }
        // 列宽只够 4~5 个字：行数封顶 + 省略号，而不是让文字被裁掉半个字。
        Text(
            text = entry.name,
            style = MaterialTheme.typography.labelMedium,
            fontSize = 10.sp,
            lineHeight = 12.sp,
            fontWeight = FontWeight.SemiBold,
            color = textColor,
            maxLines = 4,
            overflow = TextOverflow.Ellipsis,
        )
        if (entry.location.isNotBlank()) {
            Text(
                text = entry.location,
                style = MaterialTheme.typography.labelSmall,
                fontSize = 9.sp,
                lineHeight = 11.sp,
                color = textColor.copy(alpha = 0.85f),
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
        if (entry.teacher.isNotBlank()) {
            Text(
                text = entry.teacher,
                style = MaterialTheme.typography.labelSmall,
                fontSize = 9.sp,
                lineHeight = 11.sp,
                color = textColor.copy(alpha = 0.85f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

/** 选中日的完整课表：网格里放不下的长课程名 / 地点在这里完整显示。 */
@Composable
private fun SelectedDayCard(day: DaySchedule) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "${prettyDate(day.date)} 周${day.weekdayCn}",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(Modifier.width(8.dp))
            if (day.week > 0) {
                Surface(color = MaterialTheme.colorScheme.secondaryContainer, shape = CircleShape) {
                    Text(
                        text = stringResource(R.string.week_only, day.week),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
                    )
                }
            }
        }
        Spacer(Modifier.height(8.dp))
        SectionCard(title = stringResource(R.string.section_selected_day)) {
            when {
                !day.inSemester -> Text(
                    text = stringResource(R.string.timetable_not_in_semester),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

                day.entries.isEmpty() -> Text(
                    text = stringResource(R.string.timetable_no_class),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

                else -> day.entries.forEachIndexed { index, entry ->
                    if (index > 0) ThinDivider()
                    EntryRow(entry)
                }
            }
        }
    }
}

/** `2026-11-07` → `2026/11/7`：去掉前导零，与课表习惯一致。 */
internal fun prettyDate(iso: String): String {
    val parts = iso.split('-')
    if (parts.size != 3) return iso
    val month = parts[1].trimStart('0').ifBlank { "0" }
    val day = parts[2].trimStart('0').ifBlank { "0" }
    return "${parts[0]}/$month/$day"
}

/** RFC3339 → `11/07 21:42`；只用于「同步于」这类次要信息，省掉年份与秒。 */
internal fun shortStamp(rfc3339: String): String {
    if (rfc3339.length < 16) return rfc3339
    return rfc3339.substring(5, 10).replace('-', '/') + " " + rfc3339.substring(11, 16)
}

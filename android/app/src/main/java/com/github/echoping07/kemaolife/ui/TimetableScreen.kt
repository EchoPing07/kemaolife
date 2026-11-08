package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.rounded.DateRange
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.Surface
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.Layout
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.Entry
import com.github.echoping07.kemaolife.ui.theme.coursePalette
import com.github.echoping07.kemaolife.ui.theme.smoothShape
import java.time.Instant
import java.time.ZoneId

private val RowHeight = 52.dp
private val GutterWidth = 26.dp
private val GridEndPadding = 8.dp
private val WEEKDAY_CN = listOf("一", "二", "三", "四", "五", "六", "日")
private const val MOTION_MEDIUM = 280

private data class CourseTarget(val day: DaySchedule, val entry: Entry)

/**
 * 课表以整周为唯一主视图。头部只占两行，课程详情进入底部弹层，
 * 避免在网格下再重复一份当天列表，把首屏空间留给课程本身。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun TimetableScreen(state: AppState, viewModel: AppViewModel) {
    var showPicker by remember { mutableStateOf(false) }
    var detail by remember { mutableStateOf<CourseTarget?>(null) }
    val week = state.week
    val selected = state.selectedDay

    Column(modifier = Modifier.fillMaxSize()) {
        TimetableHeader(
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
            onSelect = viewModel::selectDate,
        )

        AnimatedContent(
            targetState = week,
            transitionSpec = {
                val forward = (targetState?.week ?: 0) >= (initialState?.week ?: 0)
                slideInHorizontally(
                    animationSpec = tween(MOTION_MEDIUM, easing = FastOutSlowInEasing),
                    initialOffsetX = { if (forward) it / 5 else -it / 5 },
                ) + fadeIn(tween(180)) togetherWith
                    slideOutHorizontally(
                        animationSpec = tween(MOTION_MEDIUM, easing = FastOutSlowInEasing),
                        targetOffsetX = { if (forward) -it / 5 else it / 5 },
                    ) + fadeOut(tween(140))
            },
            label = "weekTransition",
            modifier = Modifier.weight(1f),
        ) { displayedWeek ->
            when {
                displayedWeek == null && state.weekLoading -> LoadingBlock()
                displayedWeek == null -> EmptyHint(
                    text = state.timetableError ?: stringResource(R.string.bridge_not_ready),
                    modifier = Modifier.padding(20.dp),
                )
                displayedWeek.days.none { it.inSemester } -> EmptyHint(
                    text = stringResource(R.string.timetable_not_in_semester),
                    modifier = Modifier.padding(20.dp),
                )
                else -> WeekGrid(
                    days = displayedWeek.days,
                    selectedDate = state.selectedDate,
                    onCourseClick = { day, entry -> detail = CourseTarget(day, entry) },
                )
            }
        }
    }

    if (showPicker) {
        val initial = runCatching { java.time.LocalDate.parse(state.selectedDate) }
            .getOrElse { java.time.LocalDate.now() }
        val pickerState = rememberDatePickerState(
            initialSelectedDateMillis = initial.atStartOfDay(ZoneId.of("UTC")).toInstant().toEpochMilli(),
        )
        Dialog(
            onDismissRequest = { showPicker = false },
            properties = DialogProperties(usePlatformDefaultWidth = false),
        ) {
            Surface(
                shape = MaterialTheme.shapes.extraLarge,
                color = MaterialTheme.colorScheme.surfaceContainerHigh,
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp),
            ) {
                Column {
                    DatePicker(
                        state = pickerState,
                        title = null,
                        headline = null,
                        showModeToggle = false,
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 4.dp),
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.End,
                    ) {
                        TextButton(onClick = { showPicker = false }) { Text(stringResource(android.R.string.cancel)) }
                        TextButton(onClick = {
                            pickerState.selectedDateMillis?.let { millis ->
                                val date = Instant.ofEpochMilli(millis).atZone(ZoneId.of("UTC")).toLocalDate()
                                viewModel.pickDate(date)
                            }
                            showPicker = false
                        }) { Text(stringResource(R.string.action_confirm)) }
                    }
                }
            }
        }
    }

    detail?.let { target ->
        AlertDialog(
            onDismissRequest = { detail = null },
            confirmButton = {
                TextButton(onClick = { detail = null }) { Text(stringResource(R.string.action_ok)) }
            },
            shape = MaterialTheme.shapes.extraLarge,
            text = { CourseDetail(entry = target.entry, day = target.day) },
        )
    }
}

/** 日期、周次和操作集中在同一视觉单元，不再让同步时间另占一整行。 */
@Composable
private fun TimetableHeader(
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
    Column(modifier = Modifier.padding(start = 16.dp, end = 6.dp, top = 2.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            AnimatedContent(
                targetState = selected?.date.orEmpty(),
                transitionSpec = { fadeIn(tween(180)) togetherWith fadeOut(tween(120)) },
                label = "dateTransition",
                modifier = Modifier.weight(1f),
            ) { date ->
                Text(
                    text = if (date.isBlank()) stringResource(R.string.tab_timetable) else prettyDate(date),
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                )
            }
            WeekArrow(onPrevWeek, stringResource(R.string.timetable_prev_week), true)
            AnimatedVisibility(visible = !isCurrentWeek) {
                TextButton(
                    onClick = onBackToCurrent,
                    contentPadding = PaddingValues(horizontal = 8.dp),
                ) { Text(stringResource(R.string.week_current), maxLines = 1) }
            }
            WeekArrow(onNextWeek, stringResource(R.string.timetable_next_week), false)
            HeaderIcon(onPickDate, Icons.Rounded.DateRange, stringResource(R.string.timetable_pick_date))
            val rotation by animateFloatAsState(
                targetValue = if (loading) 360f else 0f,
                animationSpec = tween(500, easing = FastOutSlowInEasing),
                label = "refreshRotation",
            )
            IconButton(onClick = onRefresh, enabled = !loading, modifier = Modifier.size(40.dp)) {
                if (loading) {
                    CircularProgressIndicator(modifier = Modifier.size(19.dp), strokeWidth = 2.dp)
                } else {
                    Icon(
                        imageVector = Icons.Rounded.Refresh,
                        contentDescription = stringResource(R.string.action_refresh),
                        modifier = Modifier.size(21.dp).rotate(rotation),
                    )
                }
            }
        }
        Row(
            modifier = Modifier.padding(end = 12.dp, bottom = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (selected != null) {
                Text(
                    text = buildString {
                        if (selected.week > 0) append(stringResource(R.string.week_only, selected.week))
                        if (selected.weekdayCn.isNotBlank()) {
                            if (isNotEmpty()) append("  ·  ")
                            append(stringResource(R.string.weekday_cn, selected.weekdayCn))
                        }
                    },
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.primary,
                    fontWeight = FontWeight.Medium,
                )
            }
            Spacer(Modifier.weight(1f))
            when {
                stale -> MetaText(
                    text = stringResource(R.string.data_stale_short, shortStamp(syncedAt)),
                    emphasize = true,
                )
                syncedAt.isNotBlank() -> MetaText(stringResource(R.string.data_synced_at, shortStamp(syncedAt)))
            }
        }
    }
}

@Composable
private fun WeekArrow(onClick: () -> Unit, description: String, previous: Boolean) {
    IconButton(onClick = onClick, modifier = Modifier.size(36.dp)) {
        Icon(
            imageVector = if (previous) Icons.AutoMirrored.Rounded.KeyboardArrowLeft
            else Icons.AutoMirrored.Rounded.KeyboardArrowRight,
            contentDescription = description,
            modifier = Modifier.size(23.dp),
        )
    }
}

@Composable
private fun HeaderIcon(onClick: () -> Unit, icon: androidx.compose.ui.graphics.vector.ImageVector, description: String) {
    IconButton(onClick = onClick, modifier = Modifier.size(40.dp)) {
        Icon(icon, contentDescription = description, modifier = Modifier.size(21.dp))
    }
}

/** 星期条与网格共用完全相同的左右约束，保证七列逐像素对齐。 */
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
            .padding(end = GridEndPadding, bottom = 4.dp),
    ) {
        Column(
            modifier = Modifier.width(GutterWidth),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            val month = days.firstOrNull()?.date?.split('-')?.getOrNull(1)?.toIntOrNull()
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = month?.toString().orEmpty(),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = "月",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        for (index in 0 until 7) {
            val day = days.getOrNull(index)
            val isSelected = day != null && day.date == selectedDate
            val isToday = day != null && day.date == todayDate
            val contentColor by animateColorAsState(
                targetValue = if (isToday) MaterialTheme.colorScheme.onSurface
                else MaterialTheme.colorScheme.onSurfaceVariant,
                animationSpec = tween(220),
                label = "dayContentColor",
            )
            Column(
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 3.dp, vertical = 3.dp)
                    .clip(MaterialTheme.shapes.small)
                    .background(if (isSelected) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.6f) else Color.Transparent)
                    .clickable(enabled = day != null, role = Role.Button) { day?.let { onSelect(it.date) } }
                    .padding(vertical = 5.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    text = day?.weekdayCn ?: WEEKDAY_CN[index],
                    style = MaterialTheme.typography.labelMedium,
                    color = contentColor,
                    fontWeight = if (isToday) FontWeight.Bold else FontWeight.Normal,
                    maxLines = 1,
                )
                Spacer(Modifier.height(3.dp))
                Text(
                    text = day?.date?.substringAfterLast('-')?.toIntOrNull()?.toString().orEmpty(),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = if (isToday) FontWeight.Bold else FontWeight.Medium,
                    color = if (isSelected) MaterialTheme.colorScheme.onPrimaryContainer else contentColor,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun WeekGrid(
    days: List<DaySchedule>,
    selectedDate: String,
    onCourseClick: (DaySchedule, Entry) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(end = GridEndPadding, bottom = 16.dp)
            .height(RowHeight * TIMETABLE_ROWS),
    ) {
        Column(modifier = Modifier.width(GutterWidth)) {
            for (row in 1..TIMETABLE_ROWS) {
                Box(
                    modifier = Modifier.fillMaxWidth().height(RowHeight),
                    contentAlignment = Alignment.TopCenter,
                ) {
                    Text(
                        text = row.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 5.dp),
                    )
                }
            }
        }
        for (column in 0 until 7) {
            val day = days.getOrNull(column)
            val selected = day != null && day.date == selectedDate
            val columnColor by animateColorAsState(
                targetValue = if (selected) MaterialTheme.colorScheme.primary.copy(alpha = 0.025f)
                else Color.Transparent,
                label = "selectedColumn",
            )
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .background(columnColor),
            ) {
                Column(Modifier.fillMaxSize()) {
                    repeat(TIMETABLE_ROWS) {
                        Box(modifier = Modifier.fillMaxWidth().height(RowHeight)) {
                            HorizontalDivider(
                                modifier = Modifier.align(Alignment.BottomCenter),
                                thickness = 0.5.dp,
                                color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.25f),
                            )
                        }
                    }
                }
                val blocks = remember(day?.date, day?.entries) { layoutCourseBlocks(day?.entries.orEmpty()) }
                DayColumn(day = day, blocks = blocks, rowHeight = RowHeight, onCourseClick = onCourseClick)
            }
        }
    }
}

/** 最后一 lane 吸收整数除法余数，课程块不会在列右边留下细缝。 */
@Composable
private fun DayColumn(
    day: DaySchedule?,
    blocks: List<CourseBlock>,
    rowHeight: Dp,
    onCourseClick: (DaySchedule, Entry) -> Unit,
) {
    Layout(
        modifier = Modifier.fillMaxSize(),
        content = {
            blocks.forEach { block ->
                val palette = coursePalette(block.entry.name)
                Box(
                    modifier = Modifier
                        .padding(horizontal = 1.5.dp, vertical = 2.dp)
                        .clip(smoothShape(10.dp))
                        .background(palette.container)
                        .clickable(role = Role.Button) { day?.let { onCourseClick(it, block.entry) } }
                        .padding(horizontal = 4.dp, vertical = 4.dp),
                ) { CourseCell(block.entry, palette.onContainer) }
            }
        },
    ) { measurables, constraints ->
        val rowPx = rowHeight.roundToPx()
        fun laneStart(lane: Int, lanes: Int) = constraints.maxWidth * lane / lanes
        fun laneEnd(lane: Int, lanes: Int) = constraints.maxWidth * (lane + 1) / lanes
        val placeables = measurables.mapIndexed { index, measurable ->
            val block = blocks[index]
            measurable.measure(
                Constraints.fixed(
                    width = laneEnd(block.lane, block.lanes) - laneStart(block.lane, block.lanes),
                    height = rowPx * block.spanRows,
                ),
            )
        }
        layout(constraints.maxWidth, rowPx * TIMETABLE_ROWS) {
            placeables.forEachIndexed { index, placeable ->
                val block = blocks[index]
                placeable.place(
                    x = laneStart(block.lane, block.lanes),
                    y = rowPx * block.startRow,
                )
            }
        }
    }
}

@Composable
private fun CourseCell(entry: Entry, textColor: Color) {
    Column(verticalArrangement = Arrangement.spacedBy(1.dp)) {
        Text(
            text = entry.time.substringBefore('~'),
            style = MaterialTheme.typography.labelSmall,
            fontSize = 9.sp,
            lineHeight = 11.sp,
            fontWeight = FontWeight.Medium,
            color = textColor.copy(alpha = 0.82f),
            maxLines = 1,
        )
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
                color = textColor.copy(alpha = 0.82f),
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

@Composable
private fun CourseDetail(entry: Entry, day: DaySchedule?) {
    val palette = coursePalette(entry.name)
    Column(modifier = Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(12.dp)
                    .clip(CircleShape)
                    .background(palette.container),
            )
            Spacer(Modifier.width(10.dp))
            Text(
                text = day?.let { "${prettyDate(it.date)}  ·  周${it.weekdayCn}" }.orEmpty(),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Spacer(Modifier.height(10.dp))
        Text(
            text = entry.name,
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.SemiBold,
        )
        Spacer(Modifier.height(16.dp))
        CourseDetailRow(stringResource(R.string.detail_time), entry.time, entry.periods)
        if (entry.location.isNotBlank()) {
            CourseDetailRow(stringResource(R.string.detail_location), entry.location)
        }
        if (entry.teacher.isNotBlank()) {
            CourseDetailRow(stringResource(R.string.detail_teacher), entry.teacher)
        }
        if (entry.weeksText.isNotBlank()) {
            CourseDetailRow(stringResource(R.string.detail_weeks), entry.weeksText)
        }
    }
}

@Composable
private fun CourseDetailRow(label: String, value: String, supporting: String = "") {
    Row(modifier = Modifier.fillMaxWidth().padding(vertical = 7.dp)) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(64.dp),
        )
        Column {
            Text(text = value, style = MaterialTheme.typography.bodyMedium)
            if (supporting.isNotBlank()) {
                Text(
                    text = stringResource(R.string.detail_periods, supporting),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
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

internal fun shortStamp(rfc3339: String): String {
    if (rfc3339.length < 16) return rfc3339
    return rfc3339.substring(5, 10).replace('-', '/') + " " + rfc3339.substring(11, 16)
}

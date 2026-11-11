package com.github.echoping07.kemaolife.ui

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.ScrollState
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
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
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.Entry
import com.github.echoping07.kemaolife.ui.theme.coursePalette
import com.github.echoping07.kemaolife.ui.theme.smoothShape
import kotlinx.coroutines.launch

private val RowHeight = 52.dp
private val WeekStripHeight = 64.dp
private val GutterWidth = 26.dp
private val GridEndPadding = 8.dp
private val WEEKDAY_CN = listOf("一", "二", "三", "四", "五", "六", "日")

private data class CourseTarget(val day: DaySchedule, val entry: Entry)

/**
 * 课表以整周为唯一主视图。头部只占两行，课程详情进入底部弹层，
 * 避免在网格下再重复一份当天列表，把首屏空间留给课程本身。
 *
 * 网格区是**横向翻页容器**，不是「松手才切」的抽手手势：[HorizontalPager]
 * 拖动过程中相邻周已经在同一屏幕上跟手位移，用户能看到下一周的课程随手指滑进来
 * （docs/002 §5.3 tab1）。邻居页内容由 ViewModel 预取进内存，否则拖到一半只有留白。
 */
@Composable
internal fun TimetableScreen(state: AppState, viewModel: AppViewModel) {
    var showWeekPicker by remember { mutableStateOf(false) }
    var detail by remember { mutableStateOf<CourseTarget?>(null) }
    val week = state.week
    viewModel.timetableFeedback.ExpireFeedback()

    // 翻页序列：页码只由周次序列换算，UI 不自己算周（AGENTS A3）。
    // 还没有课表时不给 pager 任何页，直接展示空态。
    val pages = remember(state.lastWeek, week?.week) {
        WeekPager.pages(state.lastWeek, week?.week ?: 0)
    }
    val startPage = remember(pages, week?.week) {
        WeekPager.indexOf(pages, week?.week ?: 0)
    }
    val pagerState = rememberPagerState(initialPage = startPage) { pages.size }
    val scope = rememberCoroutineScope()
    // 竖向滚动位置跨周共享：12 行网格每页高度相同，翻页时保持同一滚动位置，
    // 用户「先滚到第 5 大节，再左右看各周的 5-6 节」不会每次都被拉回顶部。
    val gridScroll = rememberScrollState()

    // 周次弹窗与手势共用 pager，二者不会维护两套浏览状态。
    fun goToPage(target: Int) {
        if (target in pages.indices) {
            scope.launch { pagerState.animateScrollToPage(target) }
        }
    }

    fun goToWeek(target: Int) {
        val index = WeekPager.indexOf(pages, target)
        if (index == 0 && target !in pages) return
        goToPage(index)
    }

    // 页码落定 → 读该页对应周（0 请求），并补一格邻页。
    //
    // 用 settledPage 而不是 currentPage：后者在拖动过程中就提前跳变，
    // 滑到一半重新读缓存会把邻居页刷成占位，跟手就断了。
    // 只在「落定周 != 当前显示的周」时才读，避免与下面的对齐逻辑互相触发成环。
    LaunchedEffect(pagerState.settledPage, pages) {
        val targetWeek = pages.getOrNull(pagerState.settledPage) ?: return@LaunchedEffect
        if (targetWeek != week?.week) viewModel.loadCachedWeek(targetWeek)
        viewModel.prefetchNeighborWeeks()
    }

    // 周次由弹窗 / 回到当周 / 同步回读 / 首帧校正改变时对齐 pager。
    // 用 scrollToPage 而非 animate：这只是「跟随数据」，不该和用户的拖动抢动画。
    LaunchedEffect(week?.week, pages) {
        val current = week?.week ?: return@LaunchedEffect
        val target = WeekPager.indexOf(pages, current)
        if (target in pages.indices && pagerState.settledPage != target && !pagerState.isScrollInProgress) {
            pagerState.scrollToPage(target)
        }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        TimetableHeader(
            todayDate = state.todayDate,
            week = week?.week,
            currentWeek = state.currentWeek,
            isCurrentWeek = week?.days?.any { it.date == state.todayDate } == true,
            // 周几只在「浏览的就是今天所在周」时有意义，取该周里今天那条的 weekdayCn；
            // 取不到（缓存还没覆盖今天）就留空，不自己按日期推算星期（AGENTS A3）。
            todayWeekday = week?.days?.firstOrNull { it.date == state.todayDate }?.weekdayCn.orEmpty(),
            loading = state.weekLoading || state.timetableSyncing,
            refreshResult = viewModel.timetableFeedback.result,
            stale = state.weekStale,
            syncedAt = state.weekSyncedAt,
            onOpenWeekPicker = { showWeekPicker = true },
            onRefresh = { viewModel.refreshWeek() },
            exporting = state.exportLoading,
            onExport = { viewModel.exportTimetable() },
        )

        // 星期日期行在翻页容器**内部**：它与下面的网格属于同一周，
        // 拖动时日期必须跟着课表一起走，否则日期停在旧周而课表滑到新周，两者错位。
        // 网格区 = 横向翻页容器：拖动时两侧课表同屏跟手位移，松手吸附（docs/002 §5.3 tab1）。
        // 没有课表时不给 pager 任何页，直接走空态 / 加载态，避免「0 页 0 号页」边界问题。
        val showPager = week != null && pages.isNotEmpty()
        Box(modifier = Modifier.weight(1f)) {
            if (showPager) {
                HorizontalPager(
                    state = pagerState,
                    // 多组合两侧各一页：拖动开始的瞬间邻居页已经布局好，
                    // 否则要等它进入可见区域才开始测量，跟手会踉一下。
                    beyondViewportPageCount = 1,
                    modifier = Modifier.fillMaxSize(),
                ) { page ->
                    val pageWeek = pages.getOrNull(page)
                    // 命中内存缓存就直接画（预取结果），否则回退到当前 state 的周
                    // —— 回退只在「刚落到还没预取到的页」短暂出现，随后 LaunchedEffect 补读。
                    val days = pageWeek?.let { w ->
                        state.weekNeighbors[w]?.days ?: state.week?.takeIf { it.week == w }?.days
                    }
                    // 只有「当前页」才显示加载圈与「还没有课表」文案：
                    // 邻居页只是还没预取到，给它转圈或空态提示都会误导（拖动时一闪而过）。
                    PageContent(
                        days = days,
                        state = when {
                            days != null -> PageState.CONTENT
                            pageWeek == week?.week -> PageState.CURRENT
                            else -> PageState.PENDING
                        },
                        loading = state.weekLoading,
                        error = state.timetableError,
                        todayDate = state.todayDate,
                        scrollState = gridScroll,
                        onSync = { viewModel.syncTimetable() },
                        onCourseClick = { day, entry -> detail = CourseTarget(day, entry) },
                    )
                }
            } else {
                PageContent(
                    days = week?.days,
                    state = if (week != null) PageState.CONTENT else PageState.CURRENT,
                    loading = state.weekLoading,
                    error = state.timetableError,
                    todayDate = state.todayDate,
                    scrollState = gridScroll,
                    onSync = { viewModel.syncTimetable() },
                    onCourseClick = { day, entry -> detail = CourseTarget(day, entry) },
                )
            }
        }
    }

    if (showWeekPicker) {
        WeekPickerDialog(
            weeks = pages,
            selectedWeek = week?.week,
            currentWeek = state.currentWeek,
            onSelect = { target ->
                showWeekPicker = false
                goToWeek(target)
            },
            onBackToCurrent = {
                showWeekPicker = false
                // 当前周由 Go 的 week=0 判定，UI 不自己换算周次。
                viewModel.loadCachedWeek(0)
            },
            onDismiss = { showWeekPicker = false },
        )
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

    // 导出弹层：确认文件名与数据时间后再落到用户选的位置（SAF，不需要存储权限）。
    state.exportSheet?.let { sheet ->
        TimetableExportDialog(state = sheet, onDismiss = { viewModel.closeExport() })
    }
}

/**
 * 一页在翻页容器里的呈现状态。
 *
 * 为什么不复用 `weekLoading` 一个布尔：pager 会同时组合相邻页，邻居页「还没取到数据」
 * 和当前页「正在加载」是两件事，混用会让邻居页闪出加载圈或「本机还没有课表」。
 */
private enum class PageState { CONTENT, CURRENT, PENDING }

/**
 * 一页的内容：整周网格 / 不在学期 / 缓存缺失 / 加载中。
 */
@Composable
private fun PageContent(
    days: List<DaySchedule>?,
    state: PageState,
    loading: Boolean,
    error: String?,
    todayDate: String,
    scrollState: ScrollState,
    onSync: () -> Unit,
    onCourseClick: (DaySchedule, Entry) -> Unit,
) {
    when {
        days != null && days.none { it.inSemester } -> {
            WeekStrip(days = days, todayDate = todayDate)
            EmptyHint(
                text = stringResource(R.string.timetable_not_in_semester),
                modifier = Modifier.padding(20.dp),
            )
        }
        days != null -> Column(Modifier.fillMaxSize()) {
            // 星期条属于「这一周」，必须与网格同在一页：翻页时日期和课表一起跟手移动。
            WeekStrip(days = days, todayDate = todayDate)
            WeekGrid(
                days = days,
                scrollState = scrollState,
                onCourseClick = onCourseClick,
            )
        }
        state == PageState.CURRENT && loading -> LoadingBlock()
        state == PageState.CURRENT -> MissingCacheHint(
            // 真错误走 snackbar / 设置页；空态只给中性引导 + 可操作入口，
            // 不把内部初始化状态摊到课表位。
            text = error ?: stringResource(R.string.timetable_cache_missing),
            onSync = onSync,
        )
        // 邻居页还没预取到：留白即可，落页时 LaunchedEffect 会补读并补上内容。
        else -> Box(Modifier.fillMaxSize())
    }
}

/**
 * 头部第二行：浏览周 + 今天周几（本周）/ 当前为第 N 周（非本周）。
 *
 * 本周时右侧只有周几：这时「浏览周」与「当前周」是同一个数，再显示「当前为第 N 周」
 * 就是同一句话重复两遍；换成今天周几才提供了新信息（docs/002 §5.3 tab1）。
 */
@Composable
private fun TimetableHeader(
    todayDate: String,
    week: Int?,
    currentWeek: Int?,
    isCurrentWeek: Boolean,
    todayWeekday: String,
    loading: Boolean,
    refreshResult: ActionFeedback?,
    stale: Boolean,
    syncedAt: String,
    onOpenWeekPicker: () -> Unit,
    onRefresh: () -> Unit,
    exporting: Boolean,
    onExport: () -> Unit,
) {
    Column(modifier = Modifier.padding(start = 16.dp, end = 6.dp)) {
        Row(verticalAlignment = Alignment.Top) {
            Column(
                modifier = Modifier
                    .weight(1f)
                    .clip(MaterialTheme.shapes.medium)
                    .clickable(role = Role.Button, onClick = onOpenWeekPicker),
            ) {
                Text(
                    text = if (todayDate.isBlank()) stringResource(R.string.tab_timetable) else prettyDate(todayDate),
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                )
                if (week != null) {
                    Text(
                        text = buildString {
                            if (week > 0) append(stringResource(R.string.week_only, week))
                            if (isCurrentWeek) {
                                // 本周：周次右侧并排今天周几（第 5 周 周一），
                                // 不再重复「当前为第 N 周」——浏览周就是当前周。
                                if (todayWeekday.isNotBlank()) {
                                    if (isNotEmpty()) append("  ")
                                    append(stringResource(R.string.weekday_with_prefix, todayWeekday))
                                }
                            } else {
                                if (isNotEmpty()) append("  ")
                                append(
                                    currentWeek?.let { stringResource(R.string.week_actual_current, it) }
                                        ?: stringResource(R.string.week_not_current),
                                )
                            }
                        },
                        style = MaterialTheme.typography.labelLarge,
                        // 周次与周几都用正文色：主题色在这行偏弱，容易只看见数字。
                        color = MaterialTheme.colorScheme.onSurface,
                        fontWeight = FontWeight.Medium,
                        maxLines = 1,
                        modifier = Modifier.padding(top = 3.dp, bottom = 4.dp),
                    )
                }
            }

            // 右侧一列：图标按钮在上（导出在刷新左边），数据时间紧跟按钮下方右对齐。
            // 按钮不参与左侧两行的高度居中：它要贴顶，时间才会落在按钮下面，
            // 而不是被挤到日期那一行的右侧。
            Column(horizontalAlignment = Alignment.End) {
                Row {
                    HeaderActionButton(
                        enabled = !exporting,
                        loading = exporting,
                        icon = TabIcons.Export,
                        contentDescription = stringResource(R.string.action_export),
                        onClick = onExport,
                    )
                    RefreshAction(
                        loading = loading,
                        result = refreshResult,
                        contentDescription = stringResource(R.string.action_refresh),
                        onClick = onRefresh,
                        modifier = Modifier.size(40.dp).offset(y = HeaderIconLift),
                    )
                }
                when {
                    stale -> MetaText(
                        text = stringResource(R.string.data_stale_short, shortStamp(syncedAt)),
                        emphasize = true,
                    )
                    syncedAt.isNotBlank() -> MetaText(
                        stringResource(R.string.data_synced_at, shortStamp(syncedAt)),
                    )
                }
            }
        }
    }
}

/** 头部图标按钮：导出 / 刷新同构，只差图标、状态与回调。 */
@Composable
private fun HeaderActionButton(
    enabled: Boolean,
    loading: Boolean,
    icon: ImageVector,
    contentDescription: String,
    onClick: () -> Unit,
) {
    IconButton(onClick = onClick, enabled = enabled, modifier = Modifier.size(40.dp)) {
        // 40dp 的点击框内图标是垂直居中的；整列已贴顶，再抬 5dp 才与日期那一行对齐，
        // 否则图标会明显低于日期字。
        if (loading) {
            ExpressiveLoadingIndicator(size = 20.dp, modifier = Modifier.offset(y = HeaderIconLift))
        } else {
            Icon(
                imageVector = icon,
                contentDescription = contentDescription,
                modifier = Modifier
                    .size(21.dp)
                    .offset(y = HeaderIconLift),
            )
        }
    }
}

/** 头部图标在 40dp 点击框内的上抬量，让图标与日期那一行视觉对齐。 */
private val HeaderIconLift = (-5).dp

/** 周次与手势翻页共用同一个页码序列，弹窗不再引入另一套日期选择状态。 */
@Composable
private fun WeekPickerDialog(
    weeks: List<Int>,
    selectedWeek: Int?,
    currentWeek: Int?,
    onSelect: (Int) -> Unit,
    onBackToCurrent: () -> Unit,
    onDismiss: () -> Unit,
) {
    Dialog(onDismissRequest = onDismiss) {
        val listState = rememberLazyListState(
            initialFirstVisibleItemIndex = weeks.indexOf(selectedWeek).coerceAtLeast(0),
        )
        Surface(
            shape = MaterialTheme.shapes.extraLarge,
            color = MaterialTheme.colorScheme.surfaceContainerHigh,
        ) {
            Column(modifier = Modifier.padding(vertical = 20.dp)) {
                Text(
                    text = stringResource(R.string.timetable_pick_week),
                    style = MaterialTheme.typography.titleLarge,
                    modifier = Modifier.padding(horizontal = 24.dp),
                )
                LazyColumn(
                    state = listState,
                    modifier = Modifier.heightIn(max = 420.dp).padding(top = 10.dp),
                ) {
                    items(weeks, key = { it }) { week ->
                        val selected = week == selectedWeek
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable(role = Role.Button) { onSelect(week) }
                                .padding(horizontal = 24.dp, vertical = 14.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                text = stringResource(R.string.week_only, week),
                                style = MaterialTheme.typography.bodyLarge,
                                color = if (selected) MaterialTheme.colorScheme.primary
                                else MaterialTheme.colorScheme.onSurface,
                                fontWeight = if (selected) FontWeight.SemiBold else FontWeight.Normal,
                                modifier = Modifier.weight(1f),
                            )
                            if (week == currentWeek) {
                                Text(
                                    text = stringResource(R.string.week_current_label),
                                    style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.primary,
                                )
                            }
                        }
                    }
                }
                Row(
                    modifier = Modifier.fillMaxWidth().padding(start = 12.dp, end = 12.dp, top = 8.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    TextButton(onClick = onBackToCurrent) { Text(stringResource(R.string.week_current)) }
                    TextButton(onClick = onDismiss) { Text(stringResource(android.R.string.cancel)) }
                }
            }
        }
    }
}

/** 星期条与网格共用完全相同的左右约束，保证七列逐像素对齐。 */
@Composable
private fun WeekStrip(
    days: List<DaySchedule>,
    todayDate: String,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(WeekStripHeight)
            .padding(end = GridEndPadding, bottom = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(
            modifier = Modifier.width(GutterWidth),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            val month = days.firstOrNull()?.date?.split('-')?.getOrNull(1)?.toIntOrNull()
            // 数字与「月」用完全相同的字号 / 字重 / 颜色：两者不同字号时，
            // 「月」会又小又淡地悬在数字下面，看起来像两个不相干的字。
            val monthStyle = MaterialTheme.typography.titleSmall
            Text(
                text = month?.toString().orEmpty(),
                style = monthStyle,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = "月",
                style = monthStyle,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onSurface,
            )
        }
        for (index in 0 until 7) {
            val day = days.getOrNull(index)
            val isToday = day != null && day.date == todayDate
            // 今天只靠字重强调，不改颜色、不加背景（docs/002 §5.3 tab1）。
            // 取 ExtraBold(800)。
            // 字号保持全列一致，否则今天那列变高、上下两个字会与邻列错位。
            val todayWeight = FontWeight.ExtraBold
            val contentColor by animateColorAsState(
                targetValue = if (isToday) MaterialTheme.colorScheme.onSurface
                else MaterialTheme.colorScheme.onSurfaceVariant,
                animationSpec = tween(220),
                label = "dayContentColor",
            )
            Column(
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 3.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    text = day?.weekdayCn ?: WEEKDAY_CN[index],
                    style = MaterialTheme.typography.labelMedium,
                    color = contentColor,
                    fontWeight = if (isToday) todayWeight else FontWeight.Normal,
                    maxLines = 1,
                )
                Spacer(Modifier.height(3.dp))
                Text(
                    text = day?.date?.substringAfterLast('-')?.toIntOrNull()?.toString().orEmpty(),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = if (isToday) todayWeight else FontWeight.Medium,
                    color = contentColor,
                    maxLines = 1,
                )
            }
        }
    }
}

@Composable
private fun WeekGrid(
    days: List<DaySchedule>,
    scrollState: ScrollState,
    onCourseClick: (DaySchedule, Entry) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(scrollState)
            .padding(end = GridEndPadding, bottom = 16.dp)
            .height(RowHeight * TIMETABLE_ROWS),
    ) {
        Column(modifier = Modifier.width(GutterWidth)) {
            for (row in 1..TIMETABLE_ROWS) {
                Box(
                    modifier = Modifier.fillMaxWidth().height(RowHeight),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = row.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
        for (column in 0 until 7) {
            val day = days.getOrNull(column)
            Box(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight(),
            ) {
                // 网格不画行间横线：12 行背景横线在课程块之间反复穿插，视觉噪音大于引导，
                // 行列定位改由左侧节次数字与星期条承担（docs/002 §5.3 tab1）。
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

/** 本地还没有课表缓存时的空态：给出同步出口。 */
@Composable
private fun MissingCacheHint(text: String, onSync: () -> Unit) {
    Column(
        modifier = Modifier.padding(20.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        EmptyHint(text = text)
        Spacer(Modifier.height(12.dp))
        androidx.compose.material3.Button(onClick = onSync, shape = CircleShape) {
            Text(stringResource(R.string.action_sync))
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

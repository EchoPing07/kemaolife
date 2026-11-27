package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBars
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.union
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.LogEntry
import com.github.echoping07.kemaolife.data.LogSummary
import com.github.echoping07.kemaolife.ui.theme.appShape
import kotlinx.coroutines.delay
import kotlin.math.roundToInt

/** 宽屏（平板 / 横屏）下把内容收在可读宽度内；手机竖屏保持满宽。 */
private val LogsContentMaxWidth = 560.dp

/** 与 [CollapsingScreen] 对齐的折叠顶栏尺寸，保证日志页与其他设置二级页视觉一致。 */
private val LogsExpandedTitleSize = 34.sp
private val LogsExpandedTitleLineHeight = 42.sp
private val LogsCollapsedTitleHeight = 52.dp
private val LogsExpandedTitleGap = 12.dp
private val LogsExpandedTitleHeight = 52.dp

/** 详情正文分块上限：防止单条 64 KiB+ HTML / 压缩 JSON 构建巨型单 Text 布局。 */
private const val LogChunkMaxLines = 30
private const val LogChunkMaxChars = 2_000

/**
 * 把大段原始标头 / 正文按行与字符上限切成小块，供 `LazyColumn` 按需布局。
 *
 * 正常多行文本优先在换行处切分；无换行的压缩 JSON / HTML 按字符上限切分并避开 UTF-16 代理对。
 */
internal fun splitLogTextChunks(
    text: String,
    maxLines: Int = LogChunkMaxLines,
    maxChars: Int = LogChunkMaxChars,
): List<String> {
    if (text.isBlank()) return emptyList()
    val lineLimit = maxLines.coerceAtLeast(1)
    val charLimit = maxChars.coerceAtLeast(1)
    val chunks = ArrayList<String>((text.length / charLimit) + 1)
    var pos = 0
    val len = text.length
    while (pos < len) {
        val chunkStart = pos
        var linesInChunk = 0
        var chunkEnd = pos
        while (pos < len && linesInChunk < lineLimit) {
            val nl = text.indexOf('\n', pos)
            val lineEnd = if (nl == -1) len else nl
            val nextPos = if (nl == -1) len else nl + 1
            val candidateLen = lineEnd - chunkStart

            if (candidateLen <= charLimit) {
                chunkEnd = lineEnd
                pos = nextPos
                linesInChunk++
                if (nextPos == len && nl != -1) {
                    // 原文以换行结尾时不再额外产出末尾空块。
                    break
                }
            } else if (linesInChunk > 0) {
                // 当前块已有内容，把超出的整行留给下一块。
                break
            } else {
                // 单行本身超过字符上限（如压缩 JSON）：按上限硬切，并避免拆断 UTF-16 代理对。
                var splitEnd = (chunkStart + charLimit).coerceAtMost(lineEnd)
                if (splitEnd < lineEnd &&
                    text[splitEnd - 1].isHighSurrogate() &&
                    text[splitEnd].isLowSurrogate()
                ) {
                    if (splitEnd - 1 > chunkStart) {
                        splitEnd--
                    } else {
                        splitEnd++
                    }
                }
                chunkEnd = splitEnd
                pos = splitEnd
                break
            }
        }
        chunks.add(text.substring(chunkStart, chunkEnd))
    }
    return chunks
}

/** 仅在 `graphicsLayer` 绘制阶段读取，避免滚动驱动折叠标题时触发组合。 */
private fun lazyCollapseScrollPx(listState: LazyListState, collapseDistancePx: Int): Int =
    if (listState.firstVisibleItemIndex == 0) {
        listState.firstVisibleItemScrollOffset
    } else {
        collapseDistancePx
    }

/**
 * 设置 → 日志（docs/android.md tab3）。
 *
 * 两件事：一根「记录请求」开关，和开关打开后才有的请求列表。
 * 记录完全在 Go 侧完成（`internal/platform.RequestLog`），这里只呈现 ——
 * 标头与正文按原始字节展示，界面上只提示「含账号信息、勿外发」。
 *
 * 本页自带折叠顶栏与 `LazyColumn`（不嵌套外层 `CollapsingScreen` 的 `verticalScroll`），
 * 使最多 100 条日志按需组合，滚动时不产生多余重组。
 */
@Composable
internal fun LogsScreen(
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
) {
    val lifecycleOwner = LocalLifecycleOwner.current
    val detailOpen = state.logDetail != null

    // 进入本页读一次（0 请求，只读 Go 进程内存）。
    LaunchedEffect(Unit) { viewModel.loadRequestLog() }
    // 离页时释放日志视图快照，避免后台常驻大块列表状态。
    DisposableEffect(Unit) {
        onDispose { viewModel.clearRequestLogView() }
    }
    // 记录开着且未展开详情弹层时轻量轮询，并在退到后台时随生命周期挂起：
    // 用户往往边刷新课表 / 电费边看日志，不自动刷新就得来回切页才能看到新条目。`log.list` 只读内存、0 网络请求。
    LaunchedEffect(lifecycleOwner, state.requestLogEnabled, detailOpen) {
        if (!state.requestLogEnabled || detailOpen) return@LaunchedEffect
        lifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
            while (true) {
                delay(2_000)
                viewModel.loadRequestLog()
            }
        }
    }

    val listState = rememberLazyListState()
    val density = LocalDensity.current
    val collapseDistancePx = with(density) { (LogsExpandedTitleGap + LogsExpandedTitleHeight).toPx().roundToInt() }
    val expandedTitleHeightPx = with(density) { LogsExpandedTitleHeight.toPx() }
    val motionEnabled = animationsEnabled()
    val title = stringResource(R.string.settings_logs)
    val statusBarTop = WindowInsets.statusBars.asPaddingValues().calculateTopPadding()
    val bottomInset = WindowInsets.navigationBars.union(WindowInsets.ime).asPaddingValues().calculateBottomPadding()
    val bodyMedium = MaterialTheme.typography.bodyMedium
    val urlStyle = remember(bodyMedium) { bodyMedium.copy(fontFamily = FontFamily.Monospace) }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.surface),
    ) {
        LazyColumn(
            state = listState,
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(
                start = PageHorizontalPadding,
                end = PageHorizontalPadding,
                top = statusBarTop + LogsCollapsedTitleHeight,
                bottom = bottomInset + PageBottomPadding,
            ),
        ) {
            item(key = "header", contentType = "header") {
                Column(modifier = Modifier.fillMaxWidth()) {
                    Spacer(Modifier.height(LogsExpandedTitleGap))
                    Text(
                        text = title,
                        style = MaterialTheme.typography.headlineLarge.copy(
                            fontSize = LogsExpandedTitleSize,
                            lineHeight = LogsExpandedTitleLineHeight,
                        ),
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(LogsExpandedTitleHeight)
                            .graphicsLayer {
                                val scrollPx = lazyCollapseScrollPx(listState, collapseDistancePx)
                                val progress = collapsingTitleProgress(scrollPx, collapseDistancePx)
                                translationY = if (motionEnabled) -expandedTitleHeightPx * progress else 0f
                                alpha = 1f - progress
                            },
                    )
                    Spacer(Modifier.height(16.dp))
                }
            }

            item(key = "switch", contentType = "switch") {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = if (state.requestLogEnabled) 4.dp else 0.dp),
                    contentAlignment = Alignment.TopCenter,
                ) {
                    // 宽屏居中收窄：整行拉满会让 URL 与正文横跨 1000dp，反而更难读。
                    GroupSurface(
                        position = GroupPosition.ONLY,
                        modifier = Modifier
                            .widthIn(max = LogsContentMaxWidth)
                            .fillMaxWidth(),
                    ) {
                        RequestLogSwitchRow(
                            checked = state.requestLogEnabled,
                            onCheckedChange = viewModel::setRequestLog,
                        )
                    }
                }
            }

            if (state.requestLogEnabled) {
                val log = state.log
                when {
                    // 还没读过列表：先别下「暂无记录」的结论。
                    log == null -> {
                        item(key = "loading", contentType = "loading") {
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 8.dp),
                                contentAlignment = Alignment.TopCenter,
                            ) {
                                LoadingBlock(
                                    modifier = Modifier
                                        .widthIn(max = LogsContentMaxWidth)
                                        .fillMaxWidth(),
                                )
                            }
                        }
                    }

                    log.entries.isEmpty() -> {
                        item(key = "empty", contentType = "empty") {
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 8.dp),
                                contentAlignment = Alignment.TopCenter,
                            ) {
                                EmptyHint(
                                    text = stringResource(R.string.logs_empty),
                                    shape = appShape(PageCorners.Card),
                                    modifier = Modifier
                                        .widthIn(max = LogsContentMaxWidth)
                                        .fillMaxWidth(),
                                )
                            }
                        }
                    }

                    else -> {
                        items(
                            items = log.entries,
                            key = { it.seq },
                            contentType = { "log_entry" },
                        ) { entry ->
                            // 每条记录独立成卡：它们是并列的一次次请求，不是一组设置项。
                            Box(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 8.dp),
                                contentAlignment = Alignment.TopCenter,
                            ) {
                                LogSummaryCard(
                                    entry = entry,
                                    urlStyle = urlStyle,
                                    onClick = { viewModel.openRequestLog(entry.seq) },
                                    modifier = Modifier
                                        .widthIn(max = LogsContentMaxWidth)
                                        .fillMaxWidth(),
                                )
                            }
                        }
                    }
                }
            }
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(MaterialTheme.colorScheme.surface)
                .statusBarsPadding()
                .height(LogsCollapsedTitleHeight)
                .padding(horizontal = PageHorizontalPadding),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack, modifier = Modifier.size(40.dp)) {
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .background(MaterialTheme.colorScheme.surfaceContainerLowest, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = TabIcons.ArrowLeft,
                        contentDescription = stringResource(R.string.action_back),
                        tint = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.size(20.dp),
                    )
                }
            }
            Spacer(Modifier.width(12.dp))
            Text(
                text = title,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                modifier = Modifier
                    .weight(1f)
                    .graphicsLayer {
                        val scrollPx = lazyCollapseScrollPx(listState, collapseDistancePx)
                        val progress = collapsingTitleProgress(scrollPx, collapseDistancePx)
                        alpha = if (motionEnabled) progress else if (scrollPx > 0) 1f else 0f
                    },
            )
            // 日志页的清空按钮：只清内容，不影响「记录请求」开关。
            LogClearAction(
                enabled = state.log?.count?.let { it > 0 } == true,
                onClick = viewModel::clearRequestLog,
            )
        }
    }

    state.logDetail?.let { entry ->
        LogDetailSheet(entry = entry, onDismiss = viewModel::closeRequestLog)
    }
}

/** 开关卡：标题 + 说明 + Switch，整行单一 toggleable 语义（与设置页其它开关行同构）。 */
@Composable
private fun RequestLogSwitchRow(checked: Boolean, onCheckedChange: (Boolean) -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .toggleable(
                value = checked,
                role = Role.Switch,
                onValueChange = onCheckedChange,
            )
            .padding(start = 20.dp, end = 16.dp, top = 16.dp, bottom = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = stringResource(R.string.logs_record_requests),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = stringResource(R.string.logs_record_requests_hint),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 3.dp),
            )
        }
        Spacer(Modifier.width(12.dp))
        Switch(checked = checked, onCheckedChange = null)
    }
}

/** 列表项：方法 + 时间一行，URL 一行，状态码 + 耗时一行。 */
@Composable
private fun LogSummaryCard(
    entry: LogSummary,
    urlStyle: TextStyle,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    GroupSurface(position = GroupPosition.ONLY, onClick = onClick, modifier = modifier) {
        Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 14.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = entry.method.ifBlank { "—" },
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.weight(1f))
                Text(
                    text = entry.clock,
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Text(
                // URL 用等宽字体：斜杠与端口号在比例字体里很难对齐比对。
                // softWrap 默认开启，超长 URL 会按字符折行而不是溢出屏幕。
                text = entry.url.ifBlank { "—" },
                style = urlStyle,
                modifier = Modifier.padding(top = 6.dp),
            )
            Row(
                modifier = Modifier.padding(top = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = if (entry.error.isNotBlank()) {
                        stringResource(R.string.logs_status_failed)
                    } else {
                        stringResource(R.string.logs_status, entry.status)
                    },
                    style = MaterialTheme.typography.labelLarge,
                    color = if (isFailure(entry)) {
                        MaterialTheme.colorScheme.error
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    },
                    fontWeight = FontWeight.Medium,
                )
                Spacer(Modifier.width(14.dp))
                Text(
                    text = stringResource(R.string.logs_duration, entry.durationMs),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

/** 网络层失败（根本没有状态码）或 4xx / 5xx 都按失败着色。 */
private fun isFailure(entry: LogSummary): Boolean =
    entry.error.isNotBlank() || entry.status >= 400 || entry.status == 0

/**
 * 详情弹层：时间 / URL / 方法 / 状态码 / 耗时 + 分块虚拟化的标头与正文。
 *
 * 为什么是弹层而不是弹窗：一条请求的标头 + 正文常有一屏以上，`AlertDialog`
 * 会把内容压成一小块；弹层可以整屏滚动，也贴合「点开某条记录」的心智。
 *
 * 高度按屏幕比例给上限（而不是固定 dp）：小屏不会顶出屏幕，平板 / 横屏也不会
 * 只占一小条。标头与正文经 [splitLogTextChunks] 预切分后以 `LazyColumn` 按需渲染，
 * 避免 64 KiB+ HTML / JSON 触发巨型单 Text 布局。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun LogDetailSheet(entry: LogEntry, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val screenHeight = LocalConfiguration.current.screenHeightDp.dp
    val detailListState = rememberLazyListState()

    val requestHeaderChunks = remember(entry.requestHeaders) { splitLogTextChunks(entry.requestHeaders) }
    val requestBodyChunks = remember(entry.requestBody) { splitLogTextChunks(entry.requestBody) }
    val responseHeaderChunks = remember(entry.responseHeaders) { splitLogTextChunks(entry.responseHeaders) }
    val responseBodyChunks = remember(entry.responseBody) { splitLogTextChunks(entry.responseBody) }

    val requestHeadersTitle = stringResource(R.string.logs_request_headers)
    val requestBodyTitle = stringResource(R.string.logs_request_body)
    val responseHeadersTitle = stringResource(R.string.logs_response_headers)
    val responseBodyTitle = stringResource(R.string.logs_response_body)

    val onlyBlockShape = appShape(PageCorners.Field)
    val topBlockShape = appShape(PageCorners.Field, 0.dp)
    val bottomBlockShape = appShape(0.dp, PageCorners.Field)
    val blockBackgroundColor = MaterialTheme.colorScheme.surfaceContainerHighest
    val baseBodySmall = MaterialTheme.typography.bodySmall
    val monoBlockStyle = remember(baseBodySmall) {
        baseBodySmall.copy(
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            lineHeight = 17.sp,
        )
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        shape = appShape(PageCorners.Card),
        containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Box(
            modifier = Modifier.fillMaxWidth(),
            contentAlignment = Alignment.TopCenter,
        ) {
            LazyColumn(
                state = detailListState,
                modifier = Modifier
                    .widthIn(max = LogsContentMaxWidth)
                    .fillMaxWidth()
                    // 上限而不是固定高：短记录不留大片空白，长记录可以滚。
                    .heightIn(max = screenHeight * 0.78f),
                contentPadding = PaddingValues(start = 20.dp, end = 20.dp, bottom = 28.dp),
            ) {
                item(key = "meta", contentType = "meta") {
                    Column(modifier = Modifier.fillMaxWidth()) {
                        Text(
                            text = stringResource(R.string.logs_detail_title),
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.Bold,
                        )
                        Spacer(Modifier.height(14.dp))

                        DetailField(stringResource(R.string.logs_field_time), entry.stamp)
                        DetailField(stringResource(R.string.logs_field_url), entry.url, monospace = true)
                        DetailField(stringResource(R.string.logs_field_method), entry.method)
                        DetailField(
                            stringResource(R.string.logs_field_status),
                            if (entry.error.isNotBlank()) {
                                stringResource(R.string.logs_status_failed)
                            } else {
                                entry.status.toString()
                            },
                        )
                        DetailField(
                            stringResource(R.string.logs_field_duration),
                            stringResource(R.string.logs_duration, entry.durationMs),
                        )
                        if (entry.error.isNotBlank()) {
                            DetailField(stringResource(R.string.logs_field_error), entry.error)
                        }
                    }
                }

                logTextBlock(
                    keyPrefix = "req_headers",
                    title = requestHeadersTitle,
                    chunks = requestHeaderChunks,
                    onlyShape = onlyBlockShape,
                    topShape = topBlockShape,
                    bottomShape = bottomBlockShape,
                    backgroundColor = blockBackgroundColor,
                    textStyle = monoBlockStyle,
                )
                logTextBlock(
                    keyPrefix = "req_body",
                    title = requestBodyTitle,
                    chunks = requestBodyChunks,
                    onlyShape = onlyBlockShape,
                    topShape = topBlockShape,
                    bottomShape = bottomBlockShape,
                    backgroundColor = blockBackgroundColor,
                    textStyle = monoBlockStyle,
                )
                logTextBlock(
                    keyPrefix = "resp_headers",
                    title = responseHeadersTitle,
                    chunks = responseHeaderChunks,
                    onlyShape = onlyBlockShape,
                    topShape = topBlockShape,
                    bottomShape = bottomBlockShape,
                    backgroundColor = blockBackgroundColor,
                    textStyle = monoBlockStyle,
                )
                logTextBlock(
                    keyPrefix = "resp_body",
                    title = responseBodyTitle,
                    chunks = responseBodyChunks,
                    onlyShape = onlyBlockShape,
                    topShape = topBlockShape,
                    bottomShape = bottomBlockShape,
                    backgroundColor = blockBackgroundColor,
                    textStyle = monoBlockStyle,
                )

                item(key = "mask_hint", contentType = "hint") {
                    Text(
                        text = stringResource(R.string.logs_mask_hint),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 6.dp),
                    )
                }
            }
        }
    }
}

/**
 * 标头 / 正文虚拟化分块：标题 + 首尾圆角、中间无缝拼接的等宽文本框；空内容整块省略。
 *
 * 正文用极浅的容器色而不是描边：弹层里已经有足够多层级，再加边框会让整页变成表格。
 */
private fun LazyListScope.logTextBlock(
    keyPrefix: String,
    title: String,
    chunks: List<String>,
    onlyShape: Shape,
    topShape: Shape,
    bottomShape: Shape,
    backgroundColor: Color,
    textStyle: TextStyle,
) {
    if (chunks.isEmpty()) return
    item(key = "${keyPrefix}_title", contentType = "block_title") {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 6.dp, bottom = 8.dp),
        )
    }
    itemsIndexed(
        items = chunks,
        key = { index, _ -> "${keyPrefix}_$index" },
        contentType = { _, _ -> "block_chunk" },
    ) { index, chunk ->
        val isFirst = index == 0
        val isLast = index == chunks.lastIndex
        val shape = when {
            isFirst && isLast -> onlyShape
            isFirst -> topShape
            isLast -> bottomShape
            else -> RectangleShape
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = if (isLast) 4.dp else 0.dp)
                .clip(shape)
                .background(backgroundColor)
                .padding(
                    start = 12.dp,
                    end = 12.dp,
                    top = if (isFirst) 12.dp else 0.dp,
                    bottom = if (isLast) 12.dp else 0.dp,
                ),
        ) {
            SelectionContainer {
                Text(
                    text = chunk,
                    style = textStyle,
                )
            }
        }
    }
}

/** 详情字段：小号灰色标签 + 正文值（长值换行，不截断）。 */
@Composable
private fun DetailField(label: String, value: String, monospace: Boolean = false) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(bottom = 12.dp),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = value.ifBlank { "—" },
            style = if (monospace) {
                MaterialTheme.typography.bodyMedium.copy(fontFamily = FontFamily.Monospace)
            } else {
                MaterialTheme.typography.bodyMedium
            },
            modifier = Modifier.padding(top = 2.dp),
        )
    }
}

/** 日志页右上角的「清空」按钮；样式与三个详情页的「测试」按钮一致。 */
@Composable
internal fun LogClearAction(enabled: Boolean, onClick: () -> Unit) {
    IconButton(onClick = onClick, enabled = enabled, modifier = Modifier.size(40.dp)) {
        Icon(
            imageVector = TabIcons.Trash,
            contentDescription = stringResource(R.string.logs_clear),
            tint = if (enabled) {
                MaterialTheme.colorScheme.onSurfaceVariant
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.38f)
            },
            modifier = Modifier.size(23.dp),
        )
    }
}

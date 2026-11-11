package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.LogEntry
import com.github.echoping07.kemaolife.data.LogSummary
import com.github.echoping07.kemaolife.ui.theme.smoothShape

/** 宽屏（平板 / 横屏）下把内容收在可读宽度内；手机竖屏保持满宽。 */
private val LogsContentMaxWidth = 560.dp
private val LogsWideBreakpoint = 600.dp

/**
 * 设置 → 日志（docs/002 §5.3 tab3）。
 *
 * 两件事：一根「记录请求」开关，和开关打开后才有的请求列表。
 * 记录完全在 Go 侧完成（`internal/platform.RequestLog`），这里只呈现 ——
 * 标头与正文按原始字节展示，界面上只提示「含账号信息、勿外发」。
 *
 * 视觉沿用设置页的分组卡片：开关是一张卡，每条日志也是同款卡片、独立圆角，
 * 点开走底部弹层看完整请求，避免在窄屏上把长正文挤进列表。
 */
@Composable
internal fun LogsScreen(state: AppState, viewModel: AppViewModel) {
    // 进入本页读一次（0 请求，只读 Go 进程内存）。
    LaunchedEffect(Unit) { viewModel.loadRequestLog() }
    // 记录开着时轻量轮询：用户往往边刷新课表 / 电费边看日志，
    // 不自动刷新就得来回切页才能看到新条目。`log.list` 只读内存、0 网络请求。
    LaunchedEffect(state.requestLogEnabled) {
        if (!state.requestLogEnabled) return@LaunchedEffect
        while (true) {
            kotlinx.coroutines.delay(2_000)
            viewModel.loadRequestLog()
        }
    }

    BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
        // 宽屏居中收窄：整行拉满会让 URL 与正文横跨 1000dp，反而更难读。
        val wide = maxWidth >= LogsWideBreakpoint
        Column(
            modifier = Modifier
                .align(Alignment.TopCenter)
                .then(if (wide) Modifier.widthIn(max = LogsContentMaxWidth) else Modifier)
                .fillMaxWidth(),
        ) {
            GroupSurface(position = GroupPosition.ONLY, modifier = Modifier.fillMaxWidth()) {
                RequestLogSwitchRow(
                    checked = state.requestLogEnabled,
                    onCheckedChange = viewModel::setRequestLog,
                )
            }

            if (state.requestLogEnabled) {
                Spacer(Modifier.height(12.dp))
                when {
                    // 还没读过列表：先别下「暂无记录」的结论。
                    state.log == null -> LoadingBlock()
                    state.log.entries.isEmpty() -> EmptyHint(text = stringResource(R.string.logs_empty))
                    else -> state.log.entries.forEachIndexed { index, entry ->
                        // 每条记录独立成卡：它们是并列的一次次请求，不是一组设置项。
                        if (index > 0) Spacer(Modifier.height(8.dp))
                        LogSummaryCard(entry = entry, onClick = { viewModel.openRequestLog(entry.seq) })
                    }
                }
            }
        }
    }

    state.logDetail?.let { entry ->
        LogDetailSheet(entry = entry, onDismiss = viewModel::closeRequestLog)
    }
}

/** 开关卡：标题 + 说明 + Switch，整行可点（与设置页其它开关行同构）。 */
@Composable
private fun RequestLogSwitchRow(checked: Boolean, onCheckedChange: (Boolean) -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onCheckedChange(!checked) }
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
        Switch(checked = checked, onCheckedChange = onCheckedChange)
    }
}

/** 列表项：方法 + 时间一行，URL 一行，状态码 + 耗时一行。 */
@Composable
private fun LogSummaryCard(entry: LogSummary, onClick: () -> Unit) {
    GroupSurface(position = GroupPosition.ONLY, onClick = onClick, modifier = Modifier.fillMaxWidth()) {
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
                style = MaterialTheme.typography.bodyMedium.copy(fontFamily = FontFamily.Monospace),
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
 * 详情弹层：时间 / URL / 方法 / 状态码 / 耗时 + 标头与正文。
 *
 * 为什么是弹层而不是弹窗：一条请求的标头 + 正文常有一屏以上，`AlertDialog`
 * 会把内容压成一小块；弹层可以整屏滚动，也贴合「点开某条记录」的心智。
 *
 * 高度按屏幕比例给上限（而不是固定 dp）：小屏不会顶出屏幕，平板 / 横屏也不会
 * 只占一小条。正文保留原样换行与等宽字体，便于比对两次请求的差异。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun LogDetailSheet(entry: LogEntry, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val screenHeight = LocalConfiguration.current.screenHeightDp.dp
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        shape = smoothShape(28.dp),
        containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        BoxWithConstraints(modifier = Modifier.fillMaxWidth()) {
            val wide = maxWidth >= LogsWideBreakpoint
            Column(
                modifier = Modifier
                    .align(Alignment.TopCenter)
                    .then(if (wide) Modifier.widthIn(max = LogsContentMaxWidth) else Modifier.fillMaxWidth())
                    // 上限而不是固定高：短记录不留大片空白，长记录可以滚。
                    .heightIn(max = screenHeight * 0.78f)
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 20.dp)
                    .padding(bottom = 28.dp),
            ) {
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

                LogBlock(stringResource(R.string.logs_request_headers), entry.requestHeaders)
                LogBlock(stringResource(R.string.logs_request_body), entry.requestBody)
                LogBlock(stringResource(R.string.logs_response_headers), entry.responseHeaders)
                LogBlock(stringResource(R.string.logs_response_body), entry.responseBody)

                Spacer(Modifier.height(6.dp))
                Text(
                    text = stringResource(R.string.logs_mask_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
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

/**
 * 标头 / 正文块：标题 + 等宽文本框；空内容整块省略。
 *
 * 正文用极浅的容器色而不是描边：弹层里已经有足够多层级，
 * 再加边框会让整页变成表格。
 */
@Composable
private fun LogBlock(title: String, body: String) {
    if (body.isBlank()) return
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 6.dp, bottom = 4.dp),
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(bottom = 8.dp),
        )
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .clip(smoothShape(14.dp))
                .background(MaterialTheme.colorScheme.surfaceContainerHighest)
                .padding(12.dp),
        ) {
            Text(
                text = body,
                style = MaterialTheme.typography.bodySmall.copy(
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    lineHeight = 17.sp,
                ),
            )
        }
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

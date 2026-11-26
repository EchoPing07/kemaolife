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
import com.github.echoping07.kemaolife.ui.theme.appShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.KeyboardArrowDown
import androidx.compose.material3.Button
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.ElectricityReport
import java.util.Locale

/**
 * tab2 · 更多：今日概览 + 电费 + 校园网 + 取件码跳转（docs/android.md）。
 *
 * 前三块都是「只读 + 手动刷新」：Android 不做后台、不做定时提醒，
 * 所以每次进入本页才拉数据，离开即停。取件码卡片是纯深链跳转，不发请求。
 *
 * 排版按「一眼能读」组织：余额是整个页面的视觉重心（headline 大数字），
 * 其余信息降级为 label；同类信息不再各包一张描边卡片，而是统一 filled card +
 * 同类信息用留白与分离容器组织，不靠描边和横线堆层次。
 */
@Composable
internal fun MoreScreen(
    state: AppState,
    viewModel: AppViewModel,
    onGoSettings: () -> Unit,
    bottomContentPadding: Dp = PageBottomPadding,
) {
    // 只把稳定依赖放进 effect：ViewModel 自己读最新 state，不捕获整个 AppState。
    // 进入本页的自动查询按最小间隔节流；手动刷新按钮不受约束。
    LaunchedEffect(Unit) { viewModel.onEnterMore() }
    viewModel.timetableFeedback.ExpireFeedback()
    viewModel.electricityFeedback.ExpireFeedback()
    viewModel.campusFeedback.ExpireFeedback()
    viewModel.campusActionFeedback.ExpireFeedback()

    CollapsingScreen(
        title = stringResource(R.string.tab_more),
        bottomContentPadding = bottomContentPadding,
    ) {
        if (state.credentials.studentId.isBlank() || state.credentials.electricityAccount.isBlank()) {
            SetupCard(
                missingEducation = state.credentials.studentId.isBlank(),
                missingElectricity = state.credentials.electricityAccount.isBlank(),
                onGoSettings = onGoSettings,
            )
            Spacer(Modifier.height(12.dp))
        }

        TodayOverview(state, viewModel)

        Spacer(Modifier.height(16.dp))
        ElectricityCard(state, viewModel)

        Spacer(Modifier.height(16.dp))
        CampusCard(state, viewModel)

        Spacer(Modifier.height(16.dp))
        PickupCodeCard()
    }
}

@Composable
private fun SetupCard(
    missingEducation: Boolean,
    missingElectricity: Boolean,
    onGoSettings: () -> Unit,
) {
    SectionCard(title = stringResource(R.string.setup_missing_title)) {
        Text(
            // 教务与电费已拆开：只缺一个时不要笼统说「都要填」。
            text = when {
                missingEducation && missingElectricity -> stringResource(R.string.setup_missing_body)
                missingEducation -> stringResource(R.string.setup_missing_education)
                else -> stringResource(R.string.setup_missing_electricity)
            },
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
private fun TodayOverview(state: AppState, viewModel: AppViewModel) {
    val day: DaySchedule? = state.todayDay
    val title = stringResource(R.string.section_today)
    SectionCard(
        title = title,
        action = {
            // 与电费 / 校园网保持一致：手动同步课表，不受最小查询间隔约束（docs/android.md）。
            RefreshAction(
                loading = state.weekLoading || state.timetableSyncing,
                result = viewModel.timetableFeedback.result,
                onClick = { viewModel.refreshWeek() },
                modifier = Modifier.size(40.dp),
            )
        },
    ) {
        when {
            day == null -> PlaceholderRow(
                text = state.timetableError ?: stringResource(R.string.loading),
            )

            !day.inSemester -> PlaceholderRow(stringResource(R.string.timetable_not_in_semester))

            day.entries.isEmpty() -> PlaceholderRow(stringResource(R.string.timetable_no_class))

            else -> day.entries.forEach { entry ->
                EntryRow(
                    entry,
                    optimizeLocation = state.optimizeLocationDisplay,
                    prefixLocationAt = state.prefixLocationAt,
                    hideOverflow = state.hideTimetableOverflow,
                )
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
                result = viewModel.electricityFeedback.result,
                onClick = { viewModel.refreshElectricity() },
                modifier = Modifier.size(40.dp),
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

    // 「数据截至」必须出现：电表约有 1 天延迟，不能当实时值（docs/privacy.md §5）。
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
    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        MetricCell(
            left.first,
            left.second,
            Modifier
                .weight(1f)
                .clip(appShape(PageCorners.Nested))
                .background(MaterialTheme.colorScheme.surfaceContainerHigh)
                .padding(horizontal = 14.dp, vertical = 12.dp),
        )
        MetricCell(
            right.first,
            right.second,
            Modifier
                .weight(1f)
                .clip(appShape(PageCorners.Nested))
                .background(MaterialTheme.colorScheme.surfaceContainerHigh)
                .padding(horizontal = 14.dp, vertical = 12.dp),
        )
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
                    .clip(appShape(3.dp))
                    .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.35f)),
            )
        }
    }
}

@Composable
private fun CampusCard(state: AppState, viewModel: AppViewModel) {
    val campus = state.campus
    var showDetails by androidx.compose.runtime.remember { androidx.compose.runtime.mutableStateOf(false) }

    SectionCard(
        title = stringResource(R.string.section_campusnet),
        action = {
            RefreshAction(
                loading = state.campusLoading,
                result = viewModel.campusFeedback.result,
                contentDescription = stringResource(R.string.action_probe),
                onClick = { viewModel.refreshCampusStatus() },
                modifier = Modifier.size(40.dp),
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
            Crossfade(targetState = campus?.state, animationSpec = tween(240), label = "campusState") { state ->
                Text(
                    text = state?.let { campusStateLabel(it) }
                        ?: stringResource(R.string.loading),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    color = tint,
                )
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
                text = stringResource(
                    if (state.campusSkipped) R.string.campusnet_wifi_only_hint
                    else R.string.campusnet_offline_hint,
                ),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        // errorMsg 是上游 Location 的历史提示，不参与判态；已在线时夹带
        // 「正在进行外网拨号」会造成误读，卡片按三态收敛。手动判态 / 动作
        // 失败原因另走 snackbar，自动判态失败归 offline，不在此处渲染。
        val details = listOf(
            stringResource(R.string.campus_bras_name) to campus?.brasName.orEmpty(),
            stringResource(R.string.campus_bras_address) to campus?.brasAddress.orEmpty(),
            stringResource(R.string.campus_vlan) to campus?.vlan.orEmpty(),
            stringResource(R.string.campus_device_ip) to campus?.deviceIp.orEmpty(),
            stringResource(R.string.campus_mac) to campus?.mac.orEmpty(),
        ).filter { it.second.isNotBlank() }
        if (details.isNotEmpty()) {
            Spacer(Modifier.height(12.dp))
            // 水波纹要圆角又不能削字、不能缩进：12dp 小圆角配 10dp 垂直内边距，
            // 圆弧在文字顶部高度处内收不足 0.2dp（两种曲率模式都安全），
            // 水平保持 0，与上方状态行左对齐；整卡反馈不适用（卡内还有别的内容）。
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clip(appShape(PageCorners.GroupInner))
                    .clickable { showDetails = !showDetails }
                    .padding(vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = stringResource(R.string.campus_access_details),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.weight(1f),
                )
                val rotation by animateFloatAsState(if (showDetails) 180f else 0f, label = "detailsChevron")
                Icon(
                    imageVector = Icons.Rounded.KeyboardArrowDown,
                    contentDescription = if (showDetails) stringResource(R.string.action_collapse) else stringResource(R.string.action_expand),
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.rotate(rotation),
                )
            }
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
        // 当前推荐动作用实心按钮；其余降级为 tonal。操作只在对应按钮上显示进度。
        val enabled = state.campusActionsEnabled && state.campusActing == null
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            CampusButton(
                label = stringResource(R.string.action_login),
                icon = TabIcons.Login,
                emphasized = campus?.state == CampusState.UNAUTH,
                enabled = enabled && campus?.state == CampusState.UNAUTH,
                loading = state.campusActing == CampusAction.LOGIN,
                onClick = { viewModel.campusAction(CampusAction.LOGIN) },
                modifier = Modifier.weight(1f),
            )
            CampusButton(
                label = stringResource(R.string.action_relogin),
                icon = TabIcons.Refresh,
                emphasized = false,
                enabled = enabled,
                loading = state.campusActing == CampusAction.RELOGIN,
                onClick = { viewModel.campusAction(CampusAction.RELOGIN) },
                modifier = Modifier.weight(1f),
            )
            CampusButton(
                label = stringResource(R.string.action_logout),
                icon = TabIcons.Logout,
                emphasized = campus?.state == CampusState.ONLINE,
                enabled = enabled && campus?.state == CampusState.ONLINE,
                loading = state.campusActing == CampusAction.LOGOUT,
                onClick = { viewModel.campusAction(CampusAction.LOGOUT) },
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun CampusButton(
    label: String,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    emphasized: Boolean,
    enabled: Boolean,
    loading: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val content: @Composable androidx.compose.foundation.layout.RowScope.() -> Unit = {
        ActionIcon(
            icon = icon,
            loading = loading,
            result = null,
            tint = if (emphasized) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSecondaryContainer,
            contentDescription = label,
            modifier = Modifier.size(17.dp),
        )
        Spacer(Modifier.width(5.dp))
        Text(label, maxLines = 1)
    }
    if (emphasized) {
        Button(
            onClick = onClick,
            enabled = enabled,
            shape = appShape(PageCorners.Field),
            modifier = modifier.height(44.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 8.dp),
            content = content,
        )
    } else {
        FilledTonalButton(
            onClick = onClick,
            enabled = enabled,
            shape = appShape(PageCorners.Field),
            modifier = modifier.height(44.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 8.dp),
            content = content,
        )
    }
}

/** 取件码卡片：两个深链跳转按钮，不拉数据、无刷新动作（docs/android.md tab2）。 */
@Composable
private fun PickupCodeCard() {
    SectionCard(title = stringResource(R.string.section_pickup)) {
        Text(
            text = stringResource(R.string.pickup_hint),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            PickupButton(stringResource(R.string.pickup_taobao), PickupDestination.TAOBAO, TabIcons.BrandTaobao, Modifier.weight(1f))
            PickupButton(stringResource(R.string.pickup_pinduoduo), PickupDestination.PINDUODUO, TabIcons.Link, Modifier.weight(1f))
        }
    }
}

@Composable
private fun PickupButton(
    label: String,
    destination: PickupDestination,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    FilledTonalButton(
        onClick = { openPickupCode(context, destination, label) },
        shape = appShape(PageCorners.Field),
        modifier = modifier.height(44.dp),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 8.dp),
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            modifier = Modifier.size(17.dp),
        )
        Spacer(Modifier.width(5.dp))
        Text(label, maxLines = 1)
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

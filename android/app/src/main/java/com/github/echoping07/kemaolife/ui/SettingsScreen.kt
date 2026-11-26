package com.github.echoping07.kemaolife.ui

import android.Manifest
import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.FiniteAnimationSpec
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandHorizontally
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.shrinkHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.core.app.ActivityCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.AppPreferences
import com.github.echoping07.kemaolife.data.Credentials
import com.github.echoping07.kemaolife.data.ElectricityNotifier
import com.github.echoping07.kemaolife.ui.theme.appShape
import kotlinx.coroutines.launch

/** 带右下角悬浮保存钮的二级页尾部留白（56dp FAB + 20dp 外距 + 20dp 呼吸空隙，不含导航栏 inset）。 */
private val FabBottomClearance = 96.dp

/** 设置总览（位于 [AppRoute.Home] 的设置 Tab 内）；二级页路由与预测性返回由外层 NavDisplay 统一管理。 */
@Composable
internal fun SettingsOverviewScreen(
    state: AppState,
    viewModel: AppViewModel,
    rootBottomPadding: Dp = PageBottomPadding,
    onNavigate: (AppRoute.SettingsChild) -> Unit,
) {
    CollapsingScreen(
        title = stringResource(R.string.tab_settings),
        bottomContentPadding = rootBottomPadding,
        modifier = Modifier.fillMaxSize(),
    ) {
        SettingsOverview(
            state = state,
            onThemeMode = viewModel::setThemeMode,
            onPredictiveBackEnabled = viewModel::setPredictiveBackEnabled,
            onEducation = { onNavigate(AppRoute.SettingsChild.Education) },
            onElectricity = { onNavigate(AppRoute.SettingsChild.Electricity) },
            onCampus = { onNavigate(AppRoute.SettingsChild.Campus) },
            onAppearance = { onNavigate(AppRoute.SettingsChild.Appearance) },
            onAbout = { onNavigate(AppRoute.SettingsChild.About) },
            onLogs = { onNavigate(AppRoute.SettingsChild.Logs) },
        )
    }
}

/** 设置二级页分发入口：由 NavDisplay 的 `entryProvider` 按路由渲染。 */
@Composable
internal fun SettingsChildScreen(
    route: AppRoute.SettingsChild,
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
) {
    val navBottom = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val simpleBottomPadding = navBottom + PageBottomPadding
    val fabBottomPadding = navBottom + FabBottomClearance

    when (route) {
        AppRoute.SettingsChild.Education -> {
            EducationSettings(
                state = state,
                viewModel = viewModel,
                onBack = onBack,
                bottomContentPadding = fabBottomPadding,
            )
        }

        AppRoute.SettingsChild.Electricity -> {
            ElectricitySettings(
                state = state,
                viewModel = viewModel,
                onBack = onBack,
                bottomContentPadding = fabBottomPadding,
            )
        }

        AppRoute.SettingsChild.Campus -> {
            CampusSettings(
                state = state,
                viewModel = viewModel,
                onBack = onBack,
                bottomContentPadding = fabBottomPadding,
            )
        }

        AppRoute.SettingsChild.Appearance -> {
            CollapsingScreen(
                title = stringResource(R.string.settings_appearance),
                onBack = onBack,
                bottomContentPadding = simpleBottomPadding,
                modifier = Modifier.fillMaxSize(),
            ) {
                AppearanceScreen(state, viewModel)
            }
        }

        AppRoute.SettingsChild.About -> {
            CollapsingScreen(
                title = stringResource(R.string.settings_about),
                onBack = onBack,
                bottomContentPadding = simpleBottomPadding,
                modifier = Modifier.fillMaxSize(),
            ) {
                AboutScreen(state, viewModel)
            }
        }

        AppRoute.SettingsChild.Logs -> {
            CollapsingScreen(
                title = stringResource(R.string.settings_logs),
                onBack = onBack,
                actions = {
                    // 日志页的清空按钮：只清内容，不影响「记录请求」开关。
                    LogClearAction(
                        enabled = state.log?.count?.let { it > 0 } == true,
                        onClick = viewModel::clearRequestLog,
                    )
                },
                bottomContentPadding = simpleBottomPadding,
                modifier = Modifier.fillMaxSize(),
            ) {
                LogsScreen(state, viewModel)
            }
        }
    }
}

@Composable
private fun ProbeTestAction(onClick: () -> Unit) {
    IconButton(onClick = onClick, modifier = Modifier.size(40.dp)) {
        Icon(
            imageVector = TabIcons.PlugConnected,
            contentDescription = stringResource(R.string.action_test),
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.size(23.dp),
        )
    }
}

@Composable
private fun SettingsOverview(
    state: AppState,
    onThemeMode: (String) -> Unit,
    onPredictiveBackEnabled: (Boolean) -> Unit,
    onEducation: () -> Unit,
    onElectricity: () -> Unit,
    onCampus: () -> Unit,
    onAppearance: () -> Unit,
    onAbout: () -> Unit,
    onLogs: () -> Unit,
) {
    Column(modifier = Modifier.fillMaxWidth()) {
        SettingsGroupLabel(stringResource(R.string.settings_accounts))
        SettingsDestination(
            icon = TabIcons.Timetable,
            title = stringResource(R.string.section_edu),
            supporting =
                if (state.credentials.studentId.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else {
                    stringResource(R.string.settings_edu_configured)
                },
            onClick = onEducation,
            position = GroupPosition.FIRST,
        )
        GroupGap()
        SettingsDestination(
            icon = TabIcons.Bolt,
            title = stringResource(R.string.section_power),
            supporting =
                if (state.credentials.electricityAccount.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else if (state.credentials.electricityInheritEducation) {
                    stringResource(R.string.settings_electricity_inherited)
                } else {
                    stringResource(R.string.settings_electricity_own_account)
                },
            onClick = onElectricity,
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsDestination(
            icon = TabIcons.Campus,
            title = stringResource(R.string.section_portal),
            supporting =
                if (state.credentials.campusUser.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else {
                    stringResource(R.string.settings_campus_configured)
                },
            onClick = onCampus,
            position = GroupPosition.LAST,
        )

        SettingsGroupLabel(stringResource(R.string.settings_personalization))
        // 颜色模式放在外观入口上方：它是本组最常改的一项，可直接在总览切换，
        // 外观页只保留配色类设置（莫奈取色 / AMOLED / 连续曲率圆角 / 预设主题）。
        ThemeModeSelector(themeMode = state.themeMode, onSelect = onThemeMode)
        GroupGap()
        SettingsDestination(
            icon = TabIcons.Appearance,
            title = stringResource(R.string.settings_appearance),
            supporting = stringResource(R.string.settings_theme_hint),
            onClick = onAppearance,
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsSwitchDestination(
            icon = TabIcons.HandMove,
            title = stringResource(R.string.settings_predictive_back),
            supporting = stringResource(R.string.settings_predictive_back_hint),
            checked = state.predictiveBackEnabled,
            onCheckedChange = onPredictiveBackEnabled,
            position = GroupPosition.LAST,
        )

        // 关于、日志、交流与分享归为「关于与支持」：它们不影响账号、也不影响外观，
        // 单独一组避免混在「个性化」里让外观项被挤成半组。
        SettingsGroupLabel(stringResource(R.string.settings_about_support))
        SettingsDestination(
            icon = TabIcons.Info,
            title = stringResource(R.string.settings_about),
            supporting = stringResource(R.string.settings_about_hint),
            onClick = onAbout,
            position = GroupPosition.FIRST,
        )
        GroupGap()
        SettingsDestination(
            icon = TabIcons.Bug,
            title = stringResource(R.string.settings_logs),
            supporting =
                if (state.requestLogEnabled) {
                    stringResource(R.string.settings_logs_hint)
                } else {
                    stringResource(R.string.settings_logs_idle)
                },
            onClick = onLogs,
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        ContactDestination()
        GroupGap()
        ShareDestination()

        if (!state.bridgeReady && state.bridgeError != null) {
            Text(
                text = state.bridgeError,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.padding(16.dp),
            )
        }
    }
}

/** 交流只跳 QQ 交流群，不带任何本机数据；加群逻辑见 [joinQQGroup]。 */
@Composable
private fun ContactDestination() {
    val context = LocalContext.current
    SettingsDestination(
        icon = TabIcons.QQ,
        title = stringResource(R.string.settings_contact),
        supporting = stringResource(R.string.settings_contact_hint, QQ_GROUP_NUMBER),
        onClick = { joinQQGroup(context) },
        position = GroupPosition.MIDDLE,
    )
}

/** 分享只发固定的公开项目信息，不读取任何查询结果 / 日志 / 凭据。 */
@Composable
private fun ShareDestination() {
    val context = LocalContext.current
    val appName = stringResource(R.string.app_name)
    val chooserTitle = stringResource(R.string.share_chooser_title)
    SettingsDestination(
        icon = TabIcons.Share,
        title = stringResource(R.string.settings_share),
        supporting = stringResource(R.string.settings_share_hint),
        onClick = { shareProject(context, appName, chooserTitle) },
        position = GroupPosition.LAST,
    )
}

@Composable
private fun SettingsGroupLabel(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelLarge,
        fontWeight = FontWeight.Bold,
        color = MaterialTheme.colorScheme.primary,
        // 页面已在 16dp 内容边距内，分组标题不再额外缩进，与大标题共用左边线。
        modifier = Modifier.padding(top = 22.dp, bottom = 8.dp),
    )
}

/** MD3 Expressive list 项：无圆形图标底，以 3dp 缝隙区分条目。 */
@Composable
internal fun SettingsDestination(
    icon: ImageVector,
    title: String,
    supporting: String,
    onClick: () -> Unit,
    position: GroupPosition = GroupPosition.ONLY,
) {
    GroupSurface(position = position, onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 17.dp, bottom = 17.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(25.dp),
            )
            Spacer(Modifier.width(18.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(text = title, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 3.dp),
                )
            }
            Icon(
                imageVector = TabIcons.ChevronRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(21.dp),
            )
        }
    }
}

/** 设置总览内的开关条目：与 [SettingsDestination] 共用左侧图标与文字基线。 */
@Composable
private fun SettingsSwitchDestination(
    icon: ImageVector,
    title: String,
    supporting: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    position: GroupPosition = GroupPosition.ONLY,
) {
    GroupSurface(position = position, modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier =
                Modifier
                    .fillMaxWidth()
                    .toggleable(
                        value = checked,
                        role = Role.Switch,
                        onValueChange = onCheckedChange,
                    ).padding(start = 20.dp, end = 16.dp, top = 15.dp, bottom = 15.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(25.dp),
            )
            Spacer(Modifier.width(18.dp))
            Column(modifier = Modifier.weight(1f).padding(end = 12.dp)) {
                Text(text = title, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 3.dp),
                )
            }
            Switch(checked = checked, onCheckedChange = null)
        }
    }
}

@Composable
private fun EducationSettings(
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
    bottomContentPadding: Dp,
) {
    var studentId by remember(state.credentials.studentId) { mutableStateOf(state.credentials.studentId) }
    var password by remember { mutableStateOf("") }
    var interval by remember(state.timetableInterval) { mutableStateOf(state.timetableInterval.toString()) }
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) password = ""
    }
    CollapsingScreen(
        title = stringResource(R.string.section_edu),
        onBack = onBack,
        actions = { ProbeTestAction { viewModel.runProbe(ProbeTarget.TIMETABLE) } },
        bottomContentPadding = bottomContentPadding,
        floatingActionButton = {
            SaveFab(
                saved = state.settingsSaved,
                onSavedShown = viewModel::consumeSettingsSaved,
                onSave = {
                    // 增量提交：基于 ViewModel 里的最新凭据合并，不拿组合时的快照覆盖其它域。
                    val student = studentId.trim()
                    val pwd = password
                    viewModel.updateCredentials {
                        it.copy(studentId = student, eduPassword = pwd)
                    }
                    // 间隔与本页凭据一起保存；保存按钮负责钳制并回填实际值。
                    viewModel.setTimetableInterval(
                        interval.toIntOrNull() ?: AppPreferences.DEFAULT_TIMETABLE_INTERVAL,
                    )
                    interval =
                        viewModel.state.value.timetableInterval
                            .toString()
                },
            )
        },
        modifier = Modifier.fillMaxSize(),
    ) {
        SettingsGroupLabel(stringResource(R.string.settings_group_configuration))
        SettingsFormCard {
            AppTextField(
                value = studentId,
                onValueChange = { studentId = it },
                label = stringResource(R.string.field_student_id),
                leadingIcon = TabIcons.User,
                keyboardType = KeyboardType.Number,
            )
            Spacer(Modifier.height(14.dp))
            AppTextField(
                value = password,
                onValueChange = { password = it },
                label = stringResource(R.string.field_edu_password),
                leadingIcon = TabIcons.Lock,
                isPassword = true,
                supporting = stringResource(R.string.password_kept_hint),
                imeAction = ImeAction.Done,
            )
        }

        SettingsGroupLabel(stringResource(R.string.settings_group_advanced))
        SettingsFormCard(position = GroupPosition.FIRST) {
            IntervalField(
                value = interval,
                onValueChange = { interval = it },
                label = stringResource(R.string.field_min_interval),
                supporting =
                    stringResource(
                        R.string.interval_hint_education,
                        AppPreferences.MIN_TIMETABLE_INTERVAL,
                    ),
                minimum = AppPreferences.MIN_TIMETABLE_INTERVAL,
            )
        }
        GroupGap()
        GroupSurface(position = GroupPosition.MIDDLE, modifier = Modifier.fillMaxWidth()) {
            // 开关即改即生效，不必等「保存」：它不影响凭据，只控制课表紧凑位的展示口径。
            SettingsSwitchRow(
                title = stringResource(R.string.field_optimize_location_display),
                supporting = stringResource(R.string.field_optimize_location_display_hint),
                checked = state.optimizeLocationDisplay,
                onCheckedChange = viewModel::setOptimizeLocationDisplay,
            )
        }
        GroupGap()
        GroupSurface(position = GroupPosition.MIDDLE, modifier = Modifier.fillMaxWidth()) {
            SettingsSwitchRow(
                title = stringResource(R.string.field_prefix_location_at),
                supporting = stringResource(R.string.field_prefix_location_at_hint),
                checked = state.prefixLocationAt,
                onCheckedChange = viewModel::setPrefixLocationAt,
            )
        }
        GroupGap()
        GroupSurface(position = GroupPosition.MIDDLE, modifier = Modifier.fillMaxWidth()) {
            SettingsSwitchRow(
                title = stringResource(R.string.field_hide_timetable_overflow),
                supporting = stringResource(R.string.field_hide_timetable_overflow_hint),
                checked = state.hideTimetableOverflow,
                onCheckedChange = viewModel::setHideTimetableOverflow,
            )
        }
        GroupGap()
        GroupSurface(position = GroupPosition.LAST, modifier = Modifier.fillMaxWidth()) {
            SettingsSwitchRow(
                title = stringResource(R.string.field_optimize_location_export),
                supporting = stringResource(R.string.field_optimize_location_export_hint),
                checked = state.optimizeLocationExport,
                onCheckedChange = viewModel::setOptimizeLocationExport,
            )
        }
    }
}

/**
 * 电费设置：与教务拆开（docs/android.md tab3）。
 *
 * 为什么单独一页：电费接口无独立密码，但用户可以只查电费、不配教务；
 * 学校代码也属于电费而不是教务。混在一页会让人以为「不填教务密码就查不了电费」。
 */
@Composable
private fun ElectricitySettings(
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
    bottomContentPadding: Dp,
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    var inherit by remember(state.credentials.electricityInheritEducation) {
        mutableStateOf(state.credentials.electricityInheritEducation)
    }
    var ownAccount by remember(state.credentials.electricityStudentId) {
        mutableStateOf(state.credentials.electricityStudentId)
    }
    var customerCode by remember(state.credentials.customerCode) {
        mutableStateOf(state.credentials.customerCode)
    }
    var interval by remember(state.electricityInterval) {
        mutableStateOf(state.electricityInterval.toString())
    }
    var balanceThreshold by remember(state.electricityBalanceThreshold) {
        mutableStateOf(AppPreferences.formatBalanceThreshold(state.electricityBalanceThreshold))
    }
    var daysThreshold by remember(state.electricityDaysThreshold) {
        mutableStateOf(state.electricityDaysThreshold.toString())
    }
    var hasNotificationPermission by remember {
        mutableStateOf(ElectricityNotifier.hasPermission(context))
    }

    // 从系统通知设置页返回时立即同步最新权限态。
    DisposableEffect(lifecycleOwner, context) {
        val observer =
            LifecycleEventObserver { _, event ->
                if (event == Lifecycle.Event.ON_RESUME) {
                    hasNotificationPermission = ElectricityNotifier.hasPermission(context)
                }
            }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    val notificationPermissionLauncher =
        rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            hasNotificationPermission = ElectricityNotifier.hasPermission(context)
            if (granted && hasNotificationPermission) {
                viewModel.onNotificationPermissionGranted()
            }
        }

    fun requestOrOpenNotificationSettings() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            val activity = context.findActivity()
            val canAskDirectly =
                viewModel.shouldRequestNotificationPermissionOnLaunch() ||
                    (
                        activity != null &&
                            ActivityCompat.shouldShowRequestPermissionRationale(
                                activity,
                                Manifest.permission.POST_NOTIFICATIONS,
                            )
                        )
            if (canAskDirectly) {
                viewModel.markNotificationPermissionAsked()
                notificationPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
                return
            }
        }
        ElectricityNotifier.openNotificationSettings(context)
    }

    CollapsingScreen(
        title = stringResource(R.string.section_power),
        onBack = onBack,
        actions = { ProbeTestAction { viewModel.runProbe(ProbeTarget.ELECTRICITY) } },
        bottomContentPadding = bottomContentPadding,
        floatingActionButton = {
            SaveFab(
                saved = state.settingsSaved,
                onSavedShown = viewModel::consumeSettingsSaved,
                onSave = {
                    val useInherit = inherit
                    val own = if (useInherit) "" else ownAccount.trim()
                    val code = customerCode.trim().ifBlank { Credentials.DEFAULT_CUSTOMER_CODE }
                    viewModel.updateCredentials {
                        it.copy(
                            electricityInheritEducation = useInherit,
                            electricityStudentId = own,
                            customerCode = code,
                        )
                    }
                    viewModel.setElectricityInterval(
                        interval.toIntOrNull() ?: AppPreferences.DEFAULT_ELECTRICITY_INTERVAL,
                    )
                    interval =
                        viewModel.state.value.electricityInterval
                            .toString()
                    viewModel.setElectricityAlertThresholds(
                        balanceThreshold =
                            balanceThreshold.toDoubleOrNull()
                                ?: AppPreferences.DEFAULT_BALANCE_THRESHOLD,
                        daysThreshold =
                            daysThreshold.toIntOrNull()
                                ?: AppPreferences.DEFAULT_DAYS_THRESHOLD,
                    )
                    balanceThreshold =
                        AppPreferences.formatBalanceThreshold(
                            viewModel.state.value.electricityBalanceThreshold,
                        )
                    daysThreshold =
                        viewModel.state.value.electricityDaysThreshold
                            .toString()
                },
            )
        },
        modifier = Modifier.fillMaxSize(),
    ) {
        SettingsGroupLabel(stringResource(R.string.settings_group_configuration))
        GroupSurface(position = GroupPosition.FIRST, modifier = Modifier.fillMaxWidth()) {
            SettingsSwitchRow(
                title = stringResource(R.string.field_inherit_edu),
                supporting = stringResource(R.string.field_inherit_edu_hint),
                checked = inherit,
                onCheckedChange = { checked ->
                    // 关闭继承时把教务学号带过来当起点：大多数人只是「电费想用同一个号」。
                    if (!checked && ownAccount.isBlank()) ownAccount = state.credentials.studentId
                    inherit = checked
                },
            )
        }
        GroupGap()
        SettingsFormCard(position = GroupPosition.LAST) {
            AppTextField(
                // 已保存的凭据里继承时 electricityStudentId 恒为空，所以这里显示教务学号；
                // 未保存的编辑态用本地 ownAccount，不掺 state 里的旧值。
                value = if (inherit) state.credentials.studentId else ownAccount,
                onValueChange = { ownAccount = it },
                label = stringResource(R.string.field_electricity_student_id),
                leadingIcon = TabIcons.User,
                keyboardType = KeyboardType.Number,
                // 继承时不接受编辑：实际用的就是教务学号，允许改会造成
                // 「显示的学号」与「真正请求的学号」不一致。
                enabled = !inherit,
                supporting = if (inherit) stringResource(R.string.field_inherit_edu_value) else null,
            )
        }

        SettingsGroupLabel(stringResource(R.string.settings_group_notification))
        GroupSurface(
            position = if (state.electricityAlertEnabled) GroupPosition.FIRST else GroupPosition.ONLY,
            modifier = Modifier.fillMaxWidth(),
        ) {
            SettingsSwitchRow(
                title = stringResource(R.string.field_electricity_alert),
                supporting = stringResource(R.string.field_electricity_alert_hint),
                checked = state.electricityAlertEnabled,
                onCheckedChange = { enabled ->
                    viewModel.setElectricityAlertEnabled(enabled)
                    if (enabled && !ElectricityNotifier.hasPermission(context)) {
                        requestOrOpenNotificationSettings()
                    }
                },
            )
        }
        if (state.electricityAlertEnabled) {
            if (!hasNotificationPermission) {
                GroupGap()
                GroupSurface(
                    position = GroupPosition.MIDDLE,
                    onClick = ::requestOrOpenNotificationSettings,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Row(
                        modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 15.dp, bottom = 15.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Icon(
                            imageVector = TabIcons.AlertCircle,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.size(22.dp),
                        )
                        Spacer(Modifier.width(14.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = stringResource(R.string.alert_permission_missing_title),
                                style = MaterialTheme.typography.titleMedium,
                                color = MaterialTheme.colorScheme.error,
                            )
                            Text(
                                text = stringResource(R.string.alert_permission_missing_hint),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(top = 3.dp),
                            )
                        }
                        Icon(
                            imageVector = TabIcons.ChevronRight,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.size(21.dp),
                        )
                    }
                }
            }
            GroupGap()
            ElectricityAlertModeSelector(
                mode = state.electricityAlertMode,
                onSelect = viewModel::setElectricityAlertMode,
                position = GroupPosition.MIDDLE,
            )
            GroupGap()
            SettingsFormCard(position = GroupPosition.LAST) {
                if (state.electricityAlertMode == AppPreferences.ALERT_MODE_DAYS) {
                    val parsedDays = daysThreshold.toIntOrNull()
                    val daysOutOfRange =
                        parsedDays != null &&
                            (
                                parsedDays < AppPreferences.MIN_DAYS_THRESHOLD ||
                                    parsedDays > AppPreferences.MAX_DAYS_THRESHOLD
                                )
                    AppTextField(
                        value = daysThreshold,
                        onValueChange = { text ->
                            daysThreshold = text.filter { it.isDigit() }.take(3)
                        },
                        label = stringResource(R.string.field_alert_days_threshold),
                        leadingIcon = TabIcons.Clock,
                        keyboardType = KeyboardType.Number,
                        imeAction = ImeAction.Done,
                        supporting =
                            if (daysOutOfRange) {
                                stringResource(
                                    R.string.alert_days_threshold_clamped,
                                    AppPreferences.MIN_DAYS_THRESHOLD,
                                    AppPreferences.MAX_DAYS_THRESHOLD,
                                )
                            } else {
                                stringResource(
                                    R.string.alert_days_threshold_hint,
                                    AppPreferences.DEFAULT_DAYS_THRESHOLD,
                                )
                            },
                    )
                } else {
                    val parsedBalance = balanceThreshold.toDoubleOrNull()
                    val balanceOutOfRange =
                        parsedBalance != null &&
                            (
                                parsedBalance < AppPreferences.MIN_BALANCE_THRESHOLD ||
                                    parsedBalance > AppPreferences.MAX_BALANCE_THRESHOLD
                                )
                    AppTextField(
                        value = balanceThreshold,
                        onValueChange = { text ->
                            balanceThreshold = AppPreferences.sanitizeDecimalInput(text)
                        },
                        label = stringResource(R.string.field_alert_balance_threshold),
                        leadingIcon = TabIcons.Bolt,
                        keyboardType = KeyboardType.Decimal,
                        imeAction = ImeAction.Done,
                        supporting =
                            if (balanceOutOfRange) {
                                stringResource(
                                    R.string.alert_balance_threshold_clamped,
                                    AppPreferences.formatBalanceThreshold(AppPreferences.MIN_BALANCE_THRESHOLD),
                                    AppPreferences.formatBalanceThreshold(AppPreferences.MAX_BALANCE_THRESHOLD),
                                )
                            } else {
                                stringResource(
                                    R.string.alert_balance_threshold_hint,
                                    AppPreferences.formatBalanceThreshold(AppPreferences.DEFAULT_BALANCE_THRESHOLD),
                                )
                            },
                    )
                }
            }
        }

        SettingsGroupLabel(stringResource(R.string.settings_group_advanced))
        SettingsFormCard(position = GroupPosition.FIRST) {
            AppTextField(
                value = customerCode,
                onValueChange = { customerCode = it },
                label = stringResource(R.string.field_customer_code),
                leadingIcon = TabIcons.School,
                keyboardType = KeyboardType.Number,
            )
        }
        GroupGap()
        SettingsFormCard(position = GroupPosition.LAST) {
            IntervalField(
                value = interval,
                onValueChange = { interval = it },
                label = stringResource(R.string.field_min_interval),
                supporting =
                    stringResource(
                        R.string.interval_hint_electricity,
                        AppPreferences.MIN_ELECTRICITY_INTERVAL,
                    ),
                minimum = AppPreferences.MIN_ELECTRICITY_INTERVAL,
            )
        }
    }
}

private data class AlertModeOption(
    val mode: String,
    val labelRes: Int,
    val icon: ImageVector,
)

private val alertModeOptions = listOf(
    AlertModeOption(AppPreferences.ALERT_MODE_BALANCE, R.string.alert_mode_balance, TabIcons.Bolt),
    AlertModeOption(AppPreferences.ALERT_MODE_DAYS, R.string.alert_mode_days, TabIcons.Clock),
)

/**
 * 通知方式（电费 / 预计天数）连体二选一，复用颜色模式选择器的 MD3 Expressive 胶囊动效。
 */
@Composable
private fun ElectricityAlertModeSelector(
    mode: String,
    onSelect: (String) -> Unit,
    position: GroupPosition,
) {
    val current = AppPreferences.normalizeAlertMode(mode)
    val motionEnabled = animationsEnabled()
    GroupSurface(position = position, modifier = Modifier.fillMaxWidth()) {
        Column {
            Column(modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 16.dp)) {
                Text(
                    text = stringResource(R.string.field_alert_mode),
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(
                    text = stringResource(R.string.field_alert_mode_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 3.dp),
                )
            }
            Row(
                modifier =
                    Modifier
                        .padding(start = 20.dp, end = 16.dp, top = 12.dp, bottom = 16.dp)
                        .selectableGroup(),
                horizontalArrangement = Arrangement.spacedBy(2.dp),
            ) {
                alertModeOptions.forEachIndexed { index, option ->
                    AlertModeSegment(
                        option = option,
                        selected = current == option.mode,
                        leading = index == 0,
                        trailing = index == alertModeOptions.lastIndex,
                        motionEnabled = motionEnabled,
                        onClick = { onSelect(option.mode) },
                    )
                }
            }
        }
    }
}

private fun <T> alertModeSpec(enabled: Boolean): FiniteAnimationSpec<T> =
    if (enabled) spring(dampingRatio = 0.6f, stiffness = 800f) else tween(0)

private fun alertModeSizeSpec(enabled: Boolean): FiniteAnimationSpec<IntSize> =
    alertModeSpec(enabled)

@Composable
private fun RowScope.AlertModeSegment(
    option: AlertModeOption,
    selected: Boolean,
    leading: Boolean,
    trailing: Boolean,
    motionEnabled: Boolean,
    onClick: () -> Unit,
) {
    val segmentHeight = 48.dp
    val outerCorner = segmentHeight / 2
    val weight by animateFloatAsState(
        targetValue = if (selected) 1.25f else 1f,
        animationSpec = alertModeSpec(motionEnabled),
        label = "alertModeWeight",
    )
    val innerCorner by animateDpAsState(
        targetValue = if (selected) outerCorner else 8.dp,
        animationSpec = alertModeSpec(motionEnabled),
        label = "alertModeCorner",
    )
    val shape =
        appShape(
            topStart = if (leading) outerCorner else innerCorner,
            bottomStart = if (leading) outerCorner else innerCorner,
            topEnd = if (trailing) outerCorner else innerCorner,
            bottomEnd = if (trailing) outerCorner else innerCorner,
        )
    val container: Color by animateColorAsState(
        targetValue =
            if (selected) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.surfaceContainerHigh
            },
        animationSpec = alertModeSpec(motionEnabled),
        label = "alertModeContainer",
    )
    val content: Color by animateColorAsState(
        targetValue =
            if (selected) {
                MaterialTheme.colorScheme.onPrimary
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant
            },
        animationSpec = alertModeSpec(motionEnabled),
        label = "alertModeContent",
    )
    Box(
        modifier =
            Modifier
                .weight(weight)
                .height(segmentHeight)
                .clip(shape)
                .background(container)
                .selectable(selected = selected, role = Role.RadioButton, onClick = onClick)
                .padding(horizontal = 6.dp),
        contentAlignment = Alignment.Center,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            AnimatedVisibility(
                visible = selected,
                enter = expandHorizontally(alertModeSizeSpec(motionEnabled)) + fadeIn(alertModeSpec(motionEnabled)),
                exit = shrinkHorizontally(alertModeSizeSpec(motionEnabled)) + fadeOut(alertModeSpec(motionEnabled)),
            ) {
                Row {
                    Icon(
                        imageVector = option.icon,
                        contentDescription = null,
                        tint = content,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.width(6.dp))
                }
            }
            Text(
                text = stringResource(option.labelRes),
                style = MaterialTheme.typography.titleSmall,
                fontWeight = if (selected) FontWeight.Bold else FontWeight.Medium,
                color = content,
                maxLines = 1,
                softWrap = false,
                overflow = TextOverflow.Ellipsis,
            )
        }
    }
}

private tailrec fun Context.findActivity(): Activity? =
    when (this) {
        is Activity -> this
        is ContextWrapper -> baseContext.findActivity()
        else -> null
    }

@Composable
private fun CampusSettings(
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
    bottomContentPadding: Dp,
) {
    var user by remember(state.credentials.campusUser) { mutableStateOf(state.credentials.campusUser) }
    var password by remember { mutableStateOf("") }
    var fingerprint by remember(state.credentials.campusFingerprint) {
        mutableStateOf(state.credentials.campusFingerprint)
    }
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) password = ""
    }
    CollapsingScreen(
        title = stringResource(R.string.section_portal),
        onBack = onBack,
        actions = { ProbeTestAction { viewModel.runProbe(ProbeTarget.CAMPUS) } },
        bottomContentPadding = bottomContentPadding,
        floatingActionButton = {
            SaveFab(
                saved = state.settingsSaved,
                onSavedShown = viewModel::consumeSettingsSaved,
                onSave = {
                    val account = user.trim()
                    val pwd = password
                    val fp = fingerprint.trim()
                    viewModel.updateCredentials {
                        it.copy(campusUser = account, campusPassword = pwd, campusFingerprint = fp)
                    }
                },
            )
        },
        modifier = Modifier.fillMaxSize(),
    ) {
        SettingsGroupLabel(stringResource(R.string.settings_group_configuration))
        SettingsFormCard {
            AppTextField(
                value = user,
                onValueChange = { user = it },
                label = stringResource(R.string.field_campus_user),
                leadingIcon = TabIcons.User,
            )
            Spacer(Modifier.height(14.dp))
            AppTextField(
                value = password,
                onValueChange = { password = it },
                label = stringResource(R.string.field_campus_password),
                leadingIcon = TabIcons.Lock,
                isPassword = true,
                supporting = stringResource(R.string.password_kept_hint),
            )
        }

        SettingsGroupLabel(stringResource(R.string.settings_group_advanced))
        SettingsFormCard(position = GroupPosition.FIRST) {
            AppTextField(
                value = fingerprint,
                onValueChange = { fingerprint = it },
                label = stringResource(R.string.field_campus_fingerprint),
                leadingIcon = TabIcons.Fingerprint,
                supporting = stringResource(R.string.field_optional),
                imeAction = ImeAction.Done,
            )
        }
        GroupGap()
        GroupSurface(position = GroupPosition.LAST, modifier = Modifier.fillMaxWidth()) {
            // 开关即改即生效，不必等「保存」：它不影响凭据，只影响何时去探测。
            SettingsSwitchRow(
                title = stringResource(R.string.field_campus_wifi_only),
                supporting = stringResource(R.string.field_campus_wifi_only_hint),
                checked = state.campusWifiOnly,
                onCheckedChange = { viewModel.setCampusWifiOnly(it) },
            )
        }
    }
}

/** 教务 / 电费 / 校园网右下角纯图标悬浮保存按钮；具备入场弹跳、按压形变、保存后圆角与图标弹性切换动效。 */
@Composable
private fun SaveFab(
    saved: Boolean,
    onSavedShown: () -> Unit,
    onSave: () -> Unit,
) {
    val motionEnabled = animationsEnabled()
    val interactionSource = remember { MutableInteractionSource() }
    val pressed by interactionSource.collectIsPressedAsState()
    var tapCount by remember { mutableIntStateOf(0) }
    val latestSaved by rememberUpdatedState(saved)
    val latestOnSavedShown by rememberUpdatedState(onSavedShown)

    // 离页时若仍处在「已保存」短暂态，先消费掉避免切入另一个二级页时误闪完成图标。
    DisposableEffect(Unit) {
        onDispose {
            if (latestSaved) latestOnSavedShown()
        }
    }

    // 入场弹性显现：随二级页滑入轻快弹出，关闭动画时直接满尺寸呈现。
    val enterScale = remember { Animatable(if (motionEnabled) 0.72f else 1f) }
    val enterAlpha = remember { Animatable(if (motionEnabled) 0f else 1f) }
    LaunchedEffect(motionEnabled) {
        if (motionEnabled) {
            kotlinx.coroutines.delay(60)
            kotlinx.coroutines.coroutineScope {
                launch {
                    enterScale.animateTo(1f, spring(dampingRatio = 0.58f, stiffness = 460f))
                }
                launch {
                    enterAlpha.animateTo(1f, tween(180))
                }
            }
        } else {
            enterScale.snapTo(1f)
            enterAlpha.snapTo(1f)
        }
    }

    // 保存成功后的确认弹跳脉冲；重复点击保存时同样给出即时回弹并重置展示计时。
    val pulseScale = remember { Animatable(1f) }
    LaunchedEffect(saved, tapCount) {
        if (saved) {
            if (motionEnabled) {
                pulseScale.animateTo(1.10f, tween(95, easing = FastOutSlowInEasing))
                pulseScale.animateTo(1f, spring(dampingRatio = 0.48f, stiffness = 460f))
            } else {
                pulseScale.snapTo(1f)
            }
            kotlinx.coroutines.delay(1800)
            latestOnSavedShown()
        }
    }

    val pressScale by animateFloatAsState(
        targetValue = if (pressed) 0.90f else 1f,
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.58f, stiffness = 780f) else tween(0),
        label = "saveFabPressScale",
    )
    // 圆角随状态形变：待保存 16dp 圆角方、按压收紧到 12dp、保存成功后弹性变身为 28dp 完整圆形。
    val cornerRadius by animateDpAsState(
        targetValue =
            when {
                saved && pressed -> 22.dp
                saved -> 28.dp
                pressed -> 12.dp
                else -> 16.dp
            },
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.58f, stiffness = 560f) else tween(0),
        label = "saveFabCorner",
    )
    val containerColor by animateColorAsState(
        targetValue =
            if (saved) {
                MaterialTheme.colorScheme.tertiaryContainer
            } else {
                MaterialTheme.colorScheme.primary
            },
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.80f, stiffness = 420f) else tween(0),
        label = "saveFabContainer",
    )
    val contentColor by animateColorAsState(
        targetValue =
            if (saved) {
                MaterialTheme.colorScheme.onTertiaryContainer
            } else {
                MaterialTheme.colorScheme.onPrimary
            },
        animationSpec = if (motionEnabled) spring(dampingRatio = 0.80f, stiffness = 420f) else tween(0),
        label = "saveFabContent",
    )
    val totalScale = enterScale.value * pressScale * pulseScale.value

    FloatingActionButton(
        onClick = {
            tapCount++
            onSave()
        },
        interactionSource = interactionSource,
        shape = appShape(cornerRadius),
        containerColor = containerColor,
        contentColor = contentColor,
        modifier =
            Modifier.graphicsLayer {
                scaleX = totalScale
                scaleY = totalScale
                alpha = enterAlpha.value
            },
    ) {
        AnimatedContent(
            targetState = saved,
            transitionSpec = {
                if (motionEnabled) {
                    (
                        fadeIn(tween(180)) +
                            scaleIn(
                                animationSpec = spring(dampingRatio = 0.54f, stiffness = 520f),
                                initialScale = 0.55f,
                            )
                    ) togetherWith
                        (
                            fadeOut(tween(110)) +
                                scaleOut(tween(110), targetScale = 0.65f)
                        )
                } else {
                    EnterTransition.None togetherWith ExitTransition.None
                }
            },
            label = "saveFabIcon",
        ) { isSaved ->
            Icon(
                imageVector = if (isSaved) TabIcons.Check else TabIcons.Save,
                contentDescription = stringResource(if (isSaved) R.string.save_done else R.string.action_save),
                modifier = Modifier.size(24.dp),
            )
        }
    }
}

/** 表单卡片只负责承载一组相关控件；组名留在卡片外，保持清晰的视觉层级。 */
@Composable
private fun SettingsFormCard(
    position: GroupPosition = GroupPosition.ONLY,
    content: @Composable ColumnScope.() -> Unit,
) {
    GroupSurface(position = position, modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 22.dp),
            content = content,
        )
    }
}

/**
 * 最小值约束的间隔输入框。
 *
 * 键入过程中不做钳制（否则输入「50」会在敲 5 时被改成下限），
 * 与本页凭据一起点击「保存」时钳制并回填实际生效值。
 */
@Composable
private fun IntervalField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    supporting: String,
    minimum: Int,
) {
    val parsed = value.toIntOrNull()
    val invalid = parsed != null && parsed < minimum
    AppTextField(
        value = value,
        onValueChange = { text -> onValueChange(text.filter { it.isDigit() }.take(4)) },
        label = label,
        leadingIcon = TabIcons.Clock,
        keyboardType = KeyboardType.Number,
        imeAction = ImeAction.Done,
        supporting =
            if (invalid) {
                stringResource(R.string.interval_clamped, minimum, AppPreferences.INTERVAL_UNIT)
            } else {
                supporting
            },
    )
}

/**
 * 设置页开关行：标题 + 说明 + Switch，整行是一个开关语义（与外观页一致）。
 *
 * 调用方直接铺进 GroupSurface，不套 SettingsFormCard 的二次内边距：
 * toggleable 在 padding 之前，水波纹铺满整张圆角卡片而不是卡内一块矩形；
 * 内边距留在行内，文字离圆角足够远，不会被削字。
 */
@Composable
private fun SettingsSwitchRow(
    title: String,
    supporting: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier =
            Modifier
                .fillMaxWidth()
                .toggleable(
                    value = checked,
                    role = Role.Switch,
                    onValueChange = onCheckedChange,
                ).padding(start = 20.dp, end = 16.dp, top = 15.dp, bottom = 15.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f).padding(end = 12.dp)) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
            )
            Text(
                text = supporting,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 3.dp),
            )
        }
        Switch(checked = checked, onCheckedChange = null)
    }
}

/** 现代 MD3 Expressive 风格输入框：平滑大圆角 + 前缀图标 + 密码明暗切换眼睛。 */
@Composable
private fun AppTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    leadingIcon: ImageVector? = null,
    supporting: String? = null,
    isPassword: Boolean = false,
    keyboardType: KeyboardType = KeyboardType.Text,
    imeAction: ImeAction = ImeAction.Next,
    enabled: Boolean = true,
) {
    var passwordVisible by rememberSaveable { mutableStateOf(false) }

    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        enabled = enabled,
        label = { Text(label) },
        leadingIcon =
            leadingIcon?.let {
                {
                    Icon(
                        imageVector = it,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(20.dp),
                    )
                }
            },
        trailingIcon = {
            if (isPassword) {
                val icon = if (passwordVisible) TabIcons.EyeOff else TabIcons.Eye
                IconButton(onClick = { passwordVisible = !passwordVisible }) {
                    Icon(
                        imageVector = icon,
                        contentDescription = if (passwordVisible) "隐藏密码" else "显示密码",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(20.dp),
                    )
                }
            } else if (value.isNotEmpty()) {
                IconButton(onClick = { onValueChange("") }) {
                    Icon(
                        imageVector = TabIcons.Clear,
                        contentDescription = "清空",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(18.dp),
                    )
                }
            }
        },
        supportingText = supporting?.let { { Text(it) } },
        singleLine = true,
        visualTransformation = if (isPassword && !passwordVisible) PasswordVisualTransformation() else VisualTransformation.None,
        keyboardOptions =
            KeyboardOptions(
                keyboardType = if (isPassword) KeyboardType.Password else keyboardType,
                imeAction = imeAction,
            ),
        colors =
            androidx.compose.material3.OutlinedTextFieldDefaults.colors(
                // 输入框在白色分组卡内；用卡片同色而不是页面底色，避免在白卡里嵌一块灰。
                focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerLow,
                unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerLow,
                focusedBorderColor = MaterialTheme.colorScheme.primary,
                unfocusedBorderColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f),
            ),
        shape = appShape(PageCorners.Field),
        modifier = Modifier.fillMaxWidth(),
    )
}

package com.github.echoping07.kemaolife.ui

import android.Manifest
import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.EnterTransition
import androidx.compose.animation.ExitTransition
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
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
import androidx.compose.foundation.selection.toggleable
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
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
import androidx.compose.ui.unit.Dp
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

/** 带右下角悬浮保存钮的二级页尾部留白（56dp FAB + 20dp 外距 + 20dp 呼吸空隙，不含导航栏 inset）。 */
private val FabBottomClearance = 96.dp

private val PasswordMask = PasswordVisualTransformation()
private val SaveFabEnterSpec = spring<Float>(dampingRatio = 0.62f, stiffness = 460f)
private val SaveFabPulseUpSpec = tween<Float>(durationMillis = 95, easing = FastOutSlowInEasing)
private val SaveFabPulseDownSpec = spring<Float>(dampingRatio = 0.5f, stiffness = 460f)
private val SaveFabIconEnter =
    fadeIn(tween(160)) + scaleIn(spring(dampingRatio = 0.56f, stiffness = 520f), initialScale = 0.6f)
private val SaveFabIconExit = fadeOut(tween(100)) + scaleOut(tween(100), targetScale = 0.65f)
private val SaveFabNoTransition = EnterTransition.None togetherWith ExitTransition.None

/** 设置总览（位于 [AppDestination.Home] 的设置 Tab 内）；二级页路由与预测性返回由外层 NavDisplay 统一管理。 */
@Composable
internal fun SettingsOverviewScreen(
    state: AppState,
    viewModel: AppViewModel,
    rootBottomPadding: Dp = PageBottomPadding,
    onOpen: (SettingsPage) -> Unit,
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
            onOpen = onOpen,
        )
    }
}

/** 设置二级页分发入口：由 NavDisplay 的 `entryProvider` 按页面渲染。 */
@Composable
internal fun SettingsChildScreen(
    page: SettingsPage,
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
) {
    // 日志页自带可折叠顶栏与 LazyColumn 虚拟化列表，不可再外包 verticalScroll 的 CollapsingScreen。
    if (page == SettingsPage.LOGS) {
        LogsScreen(state = state, viewModel = viewModel, onBack = onBack)
        return
    }

    val navBottom = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val simpleBottomPadding = navBottom + PageBottomPadding
    val fabBottomPadding = navBottom + FabBottomClearance

    when (page) {
        SettingsPage.EDUCATION -> EducationSettings(state, viewModel, onBack, fabBottomPadding)
        SettingsPage.ELECTRICITY -> ElectricitySettings(state, viewModel, onBack, fabBottomPadding)
        SettingsPage.CAMPUS -> CampusSettings(state, viewModel, onBack, fabBottomPadding)
        SettingsPage.APPEARANCE -> {
            CollapsingScreen(
                title = stringResource(R.string.settings_appearance),
                onBack = onBack,
                bottomContentPadding = simpleBottomPadding,
                modifier = Modifier.fillMaxSize(),
            ) {
                AppearanceScreen(state, viewModel)
            }
        }
        SettingsPage.ABOUT -> {
            CollapsingScreen(
                title = stringResource(R.string.settings_about),
                onBack = onBack,
                bottomContentPadding = simpleBottomPadding,
                modifier = Modifier.fillMaxSize(),
            ) {
                AboutScreen(state, viewModel)
            }
        }
        SettingsPage.LOGS -> Unit
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

/** 教务 / 电费 / 校园网共用详情页骨架：顶栏「测试」动作 + 右下角悬浮保存钮 + 保存后回调。 */
@Composable
private fun DomainSettingsScreen(
    title: String,
    probeTarget: ProbeTarget,
    state: AppState,
    viewModel: AppViewModel,
    onBack: () -> Unit,
    bottomContentPadding: Dp,
    onSave: () -> Unit,
    onSaved: (() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val latestOnSaved by rememberUpdatedState(onSaved)
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) latestOnSaved?.invoke()
    }
    CollapsingScreen(
        title = title,
        onBack = onBack,
        actions = { ProbeTestAction { viewModel.runProbe(probeTarget) } },
        bottomContentPadding = bottomContentPadding,
        floatingActionButton = {
            SaveFab(
                saved = state.settingsSaved,
                onSavedShown = viewModel::consumeSettingsSaved,
                onSave = onSave,
            )
        },
        modifier = Modifier.fillMaxSize(),
        content = content,
    )
}

@Composable
private fun SettingsOverview(
    state: AppState,
    onThemeMode: (String) -> Unit,
    onPredictiveBackEnabled: (Boolean) -> Unit,
    onOpen: (SettingsPage) -> Unit,
) {
    val context = LocalContext.current
    val appName = stringResource(R.string.app_name)
    val chooserTitle = stringResource(R.string.share_chooser_title)

    Column(modifier = Modifier.fillMaxWidth()) {
        SettingsGroupLabel(stringResource(R.string.settings_accounts))
        SettingsRow(
            icon = TabIcons.Timetable,
            title = stringResource(R.string.section_edu),
            supporting =
                if (state.credentials.studentId.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else {
                    stringResource(R.string.settings_edu_configured)
                },
            onClick = { onOpen(SettingsPage.EDUCATION) },
            position = GroupPosition.FIRST,
        )
        GroupGap()
        SettingsRow(
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
            onClick = { onOpen(SettingsPage.ELECTRICITY) },
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsRow(
            icon = TabIcons.Campus,
            title = stringResource(R.string.section_portal),
            supporting =
                if (state.credentials.campusUser.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else {
                    stringResource(R.string.settings_campus_configured)
                },
            onClick = { onOpen(SettingsPage.CAMPUS) },
            position = GroupPosition.LAST,
        )

        SettingsGroupLabel(stringResource(R.string.settings_personalization))
        // 颜色模式放在外观入口上方：它是本组最常改的一项，可直接在总览切换，
        // 外观页只保留配色类设置（莫奈取色 / AMOLED / 连续曲率圆角 / 预设主题）。
        ThemeModeSelector(themeMode = state.themeMode, onSelect = onThemeMode)
        GroupGap()
        SettingsRow(
            icon = TabIcons.Appearance,
            title = stringResource(R.string.settings_appearance),
            supporting = stringResource(R.string.settings_theme_hint),
            onClick = { onOpen(SettingsPage.APPEARANCE) },
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsSwitchRow(
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
        SettingsRow(
            icon = TabIcons.Info,
            title = stringResource(R.string.settings_about),
            supporting = stringResource(R.string.settings_about_hint),
            onClick = { onOpen(SettingsPage.ABOUT) },
            position = GroupPosition.FIRST,
        )
        GroupGap()
        SettingsRow(
            icon = TabIcons.Bug,
            title = stringResource(R.string.settings_logs),
            supporting =
                if (state.requestLogEnabled) {
                    stringResource(R.string.settings_logs_hint)
                } else {
                    stringResource(R.string.settings_logs_idle)
                },
            onClick = { onOpen(SettingsPage.LOGS) },
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        // 交流只跳 QQ 交流群，不带任何本机数据；加群逻辑见 joinQQGroup。
        SettingsRow(
            icon = TabIcons.QQ,
            title = stringResource(R.string.settings_contact),
            supporting = stringResource(R.string.settings_contact_hint, QQ_GROUP_NUMBER),
            onClick = { joinQQGroup(context) },
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        // 分享只发固定的公开项目信息，不读取任何查询结果 / 日志 / 凭据。
        SettingsRow(
            icon = TabIcons.Share,
            title = stringResource(R.string.settings_share),
            supporting = stringResource(R.string.settings_share_hint),
            onClick = { shareProject(context, appName, chooserTitle) },
            position = GroupPosition.LAST,
        )

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

/** 通用设置列表行：支持可选左侧图标、强调着色、右侧自定义区或默认箭头。 */
@Composable
internal fun SettingsRow(
    title: String,
    supporting: String,
    onClick: (() -> Unit)? = null,
    position: GroupPosition = GroupPosition.ONLY,
    icon: ImageVector? = null,
    tint: Color = Color.Unspecified,
    showChevron: Boolean = onClick != null,
    trailing: (@Composable () -> Unit)? = null,
) {
    val iconTint = if (tint != Color.Unspecified) tint else MaterialTheme.colorScheme.onSurfaceVariant
    GroupSurface(position = position, onClick = onClick, modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 16.dp, bottom = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (icon != null) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = iconTint,
                    modifier = Modifier.size(25.dp),
                )
                Spacer(Modifier.width(18.dp))
            }
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    color = tint,
                )
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 3.dp),
                )
            }
            when {
                trailing != null -> trailing()
                showChevron -> Icon(
                    imageVector = TabIcons.ChevronRight,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(21.dp),
                )
            }
        }
    }
}

/**
 * 统一开关行：支持可选左侧图标与禁用态，整行作为单个开关语义。
 *
 * toggleable 位于 padding 之前，水波纹铺满整张圆角卡片；内边距留在行内避免削字。
 */
@Composable
private fun SettingsSwitchRow(
    title: String,
    supporting: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    position: GroupPosition = GroupPosition.ONLY,
    icon: ImageVector? = null,
    enabled: Boolean = true,
) {
    GroupSurface(position = position, modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier =
                Modifier
                    .fillMaxWidth()
                    .toggleable(
                        value = checked,
                        enabled = enabled,
                        role = Role.Switch,
                        onValueChange = onCheckedChange,
                    ).padding(start = 20.dp, end = 16.dp, top = 15.dp, bottom = 15.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (icon != null) {
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(25.dp),
                )
                Spacer(Modifier.width(18.dp))
            }
            Column(modifier = Modifier.weight(1f).padding(end = 12.dp)) {
                Text(text = title, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = supporting,
                    style = MaterialTheme.typography.bodySmall,
                    color =
                        if (enabled) {
                            MaterialTheme.colorScheme.onSurfaceVariant
                        } else {
                            MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
                        },
                    modifier = Modifier.padding(top = 3.dp),
                )
            }
            Switch(checked = checked, onCheckedChange = null, enabled = enabled)
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

    DomainSettingsScreen(
        title = stringResource(R.string.section_edu),
        probeTarget = ProbeTarget.TIMETABLE,
        state = state,
        viewModel = viewModel,
        onBack = onBack,
        bottomContentPadding = bottomContentPadding,
        onSaved = { password = "" },
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
            interval = viewModel.state.value.timetableInterval.toString()
        },
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
        // 开关即改即生效，不必等「保存」：它不影响凭据，只控制课表紧凑位的展示口径。
        SettingsSwitchRow(
            title = stringResource(R.string.field_optimize_location_display),
            supporting = stringResource(R.string.field_optimize_location_display_hint),
            checked = state.optimizeLocationDisplay,
            onCheckedChange = viewModel::setOptimizeLocationDisplay,
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsSwitchRow(
            title = stringResource(R.string.field_prefix_location_at),
            supporting = stringResource(R.string.field_prefix_location_at_hint),
            checked = state.prefixLocationAt,
            onCheckedChange = viewModel::setPrefixLocationAt,
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsSwitchRow(
            title = stringResource(R.string.field_hide_timetable_overflow),
            supporting = stringResource(R.string.field_hide_timetable_overflow_hint),
            checked = state.hideTimetableOverflow,
            onCheckedChange = viewModel::setHideTimetableOverflow,
            position = GroupPosition.MIDDLE,
        )
        GroupGap()
        SettingsSwitchRow(
            title = stringResource(R.string.field_optimize_location_export),
            supporting = stringResource(R.string.field_optimize_location_export_hint),
            checked = state.optimizeLocationExport,
            onCheckedChange = viewModel::setOptimizeLocationExport,
            position = GroupPosition.LAST,
        )
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

    DomainSettingsScreen(
        title = stringResource(R.string.section_power),
        probeTarget = ProbeTarget.ELECTRICITY,
        state = state,
        viewModel = viewModel,
        onBack = onBack,
        bottomContentPadding = bottomContentPadding,
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
            interval = viewModel.state.value.electricityInterval.toString()
            viewModel.setElectricityAlertThresholds(
                balanceThreshold =
                    balanceThreshold.toDoubleOrNull()
                        ?: AppPreferences.DEFAULT_BALANCE_THRESHOLD,
                daysThreshold =
                    daysThreshold.toIntOrNull()
                        ?: AppPreferences.DEFAULT_DAYS_THRESHOLD,
            )
            balanceThreshold =
                AppPreferences.formatBalanceThreshold(viewModel.state.value.electricityBalanceThreshold)
            daysThreshold = viewModel.state.value.electricityDaysThreshold.toString()
        },
    ) {
        SettingsGroupLabel(stringResource(R.string.settings_group_configuration))
        SettingsSwitchRow(
            title = stringResource(R.string.field_inherit_edu),
            supporting = stringResource(R.string.field_inherit_edu_hint),
            checked = inherit,
            onCheckedChange = { checked ->
                // 关闭继承时把教务学号带过来当起点：大多数人只是「电费想用同一个号」。
                if (!checked && ownAccount.isBlank()) ownAccount = state.credentials.studentId
                inherit = checked
            },
            position = GroupPosition.FIRST,
        )
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
            position = if (state.electricityAlertEnabled) GroupPosition.FIRST else GroupPosition.ONLY,
        )
        if (state.electricityAlertEnabled) {
            if (!hasNotificationPermission) {
                GroupGap()
                SettingsRow(
                    icon = TabIcons.AlertCircle,
                    title = stringResource(R.string.alert_permission_missing_title),
                    supporting = stringResource(R.string.alert_permission_missing_hint),
                    tint = MaterialTheme.colorScheme.error,
                    onClick = ::requestOrOpenNotificationSettings,
                    position = GroupPosition.MIDDLE,
                )
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

/** 通知方式（电费 / 预计天数）连体二选一，复用 [ConnectedSegmentedGroup]。 */
@Composable
private fun ElectricityAlertModeSelector(
    mode: String,
    onSelect: (String) -> Unit,
    position: GroupPosition,
) {
    val balanceLabel = stringResource(R.string.alert_mode_balance)
    val daysLabel = stringResource(R.string.alert_mode_days)
    val options = remember(balanceLabel, daysLabel) {
        listOf(
            SegmentedOption(AppPreferences.ALERT_MODE_BALANCE, balanceLabel, TabIcons.Bolt),
            SegmentedOption(AppPreferences.ALERT_MODE_DAYS, daysLabel, TabIcons.Clock),
        )
    }
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
            ConnectedSegmentedGroup(
                options = options,
                selected = AppPreferences.normalizeAlertMode(mode),
                onSelect = onSelect,
                height = 48.dp,
                selectedWeight = 1.25f,
                itemHorizontalPadding = 6.dp,
                modifier = Modifier.padding(start = 20.dp, end = 16.dp, top = 12.dp, bottom = 16.dp),
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

    DomainSettingsScreen(
        title = stringResource(R.string.section_portal),
        probeTarget = ProbeTarget.CAMPUS,
        state = state,
        viewModel = viewModel,
        onBack = onBack,
        bottomContentPadding = bottomContentPadding,
        onSaved = { password = "" },
        onSave = {
            val account = user.trim()
            val pwd = password
            val fp = fingerprint.trim()
            viewModel.updateCredentials {
                it.copy(campusUser = account, campusPassword = pwd, campusFingerprint = fp)
            }
        },
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
        // 开关即改即生效，不必等「保存」：它不影响凭据，只影响何时去探测。
        SettingsSwitchRow(
            title = stringResource(R.string.field_campus_wifi_only),
            supporting = stringResource(R.string.field_campus_wifi_only_hint),
            checked = state.campusWifiOnly,
            onCheckedChange = viewModel::setCampusWifiOnly,
            position = GroupPosition.LAST,
        )
    }
}

/** 教务 / 电费 / 校园网右下角纯图标悬浮保存按钮；缩放与透明度全部下沉至 graphicsLayer，动画帧零重组。 */
@Composable
private fun SaveFab(
    saved: Boolean,
    onSavedShown: () -> Unit,
    onSave: () -> Unit,
) {
    val motionEnabled = animationsEnabled()
    var tapCount by remember { mutableIntStateOf(0) }
    val latestSaved by rememberUpdatedState(saved)
    val latestOnSavedShown by rememberUpdatedState(onSavedShown)
    val enter = remember { Animatable(if (motionEnabled) 0f else 1f) }
    val pulse = remember { Animatable(1f) }

    // 离页时若仍处在「已保存」短暂态，先消费掉避免切入另一个二级页时误闪完成图标。
    DisposableEffect(Unit) {
        onDispose { if (latestSaved) latestOnSavedShown() }
    }
    LaunchedEffect(motionEnabled) {
        if (motionEnabled) enter.animateTo(1f, SaveFabEnterSpec) else enter.snapTo(1f)
    }
    LaunchedEffect(saved, tapCount) {
        if (!saved) return@LaunchedEffect
        if (motionEnabled) {
            pulse.animateTo(1.08f, SaveFabPulseUpSpec)
            pulse.animateTo(1f, SaveFabPulseDownSpec)
        } else {
            pulse.snapTo(1f)
        }
        kotlinx.coroutines.delay(1800)
        latestOnSavedShown()
    }

    FloatingActionButton(
        onClick = {
            tapCount++
            onSave()
        },
        shape = if (saved) CircleShape else appShape(16.dp),
        containerColor =
            if (saved) MaterialTheme.colorScheme.tertiaryContainer else MaterialTheme.colorScheme.primary,
        contentColor =
            if (saved) MaterialTheme.colorScheme.onTertiaryContainer else MaterialTheme.colorScheme.onPrimary,
        modifier =
            Modifier.graphicsLayer {
                val e = enter.value
                val s = (0.76f + 0.24f * e) * pulse.value
                scaleX = s
                scaleY = s
                alpha = e.coerceIn(0f, 1f)
            },
    ) {
        AnimatedContent(
            targetState = saved,
            transitionSpec = {
                if (motionEnabled) SaveFabIconEnter togetherWith SaveFabIconExit else SaveFabNoTransition
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
        visualTransformation = if (isPassword && !passwordVisible) PasswordMask else VisualTransformation.None,
        keyboardOptions =
            KeyboardOptions(
                keyboardType = if (isPassword) KeyboardType.Password else keyboardType,
                imeAction = imeAction,
            ),
        colors =
            OutlinedTextFieldDefaults.colors(
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

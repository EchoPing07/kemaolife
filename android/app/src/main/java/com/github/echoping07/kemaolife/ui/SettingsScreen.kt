package com.github.echoping07.kemaolife.ui

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import com.github.echoping07.kemaolife.ui.theme.appShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.DateRange
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.unit.sp
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.AppPreferences
import com.github.echoping07.kemaolife.data.Credentials

private enum class SettingsPage { ROOT, EDUCATION, ELECTRICITY, CAMPUS, APPEARANCE, ABOUT, LOGS }

/** 设置总览只承担导航；凭据表单放进各自页面，避免三个功能域一次全部铺开。 */
@Composable
internal fun SettingsScreen(state: AppState, viewModel: AppViewModel) {
    var pageName by rememberSaveable { mutableStateOf(SettingsPage.ROOT.name) }
    val page = SettingsPage.valueOf(pageName)
    val motionEnabled = animationsEnabled()
    BackHandler(enabled = page != SettingsPage.ROOT) { pageName = SettingsPage.ROOT.name }

    Box(modifier = Modifier.fillMaxSize()) {

        val title = stringResource(
            when (page) {
                SettingsPage.ROOT -> R.string.tab_settings
                SettingsPage.EDUCATION -> R.string.section_edu
                SettingsPage.ELECTRICITY -> R.string.section_power
                SettingsPage.CAMPUS -> R.string.section_portal
                SettingsPage.APPEARANCE -> R.string.settings_appearance
                SettingsPage.ABOUT -> R.string.settings_about
                SettingsPage.LOGS -> R.string.settings_logs
            },
        )
        val testTarget = when (page) {
            SettingsPage.EDUCATION -> ProbeTarget.TIMETABLE
            SettingsPage.ELECTRICITY -> ProbeTarget.ELECTRICITY
            SettingsPage.CAMPUS -> ProbeTarget.CAMPUS
            else -> null
        }
        // 进入 / 返回都使用新的顶栏与滚动状态，不继承上一页的折叠高度。
        androidx.compose.runtime.key(page) {
            CollapsingScreen(
                title = title,
                onBack = if (page == SettingsPage.ROOT) null else ({ pageName = SettingsPage.ROOT.name }),
                actions = {
                    if (testTarget != null) {
                        IconButton(onClick = { viewModel.runProbe(testTarget) }, modifier = Modifier.size(40.dp)) {
                            Icon(
                                imageVector = TabIcons.PlugConnected,
                                contentDescription = stringResource(R.string.action_test),
                                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.size(23.dp),
                            )
                        }
                    }
                    // 日志页的清空按钮：只清内容，不影响「记录请求」开关。
                    if (page == SettingsPage.LOGS) {
                        LogClearAction(
                            enabled = state.log?.count?.let { it > 0 } == true,
                            onClick = viewModel::clearRequestLog,
                        )
                    }
                },
                modifier = Modifier.fillMaxSize(),
            ) {
                AnimatedContent(
                    targetState = page,
                    transitionSpec = {
                        if (!motionEnabled) {
                            androidx.compose.animation.EnterTransition.None togetherWith
                                androidx.compose.animation.ExitTransition.None
                        } else {
                            val direction = if (targetState != SettingsPage.ROOT) 1 else -1
                            slideInHorizontally(
                                animationSpec = spring(dampingRatio = 0.86f, stiffness = Spring.StiffnessMediumLow),
                            ) { direction * it / 4 } + fadeIn(tween(180)) togetherWith
                                slideOutHorizontally(
                                    animationSpec = spring(dampingRatio = 1f, stiffness = Spring.StiffnessMedium),
                                ) { -direction * it / 6 } + fadeOut(tween(120))
                        }
                    },
                    label = "settingsNavigation",
                    modifier = Modifier.fillMaxSize(),
                ) { currentPage ->
                    when (currentPage) {
                        SettingsPage.ROOT -> SettingsOverview(
                            state = state,
                            onThemeMode = viewModel::setThemeMode,
                            onEducation = { pageName = SettingsPage.EDUCATION.name },
                            onElectricity = { pageName = SettingsPage.ELECTRICITY.name },
                            onCampus = { pageName = SettingsPage.CAMPUS.name },
                            onAppearance = { pageName = SettingsPage.APPEARANCE.name },
                            onAbout = { pageName = SettingsPage.ABOUT.name },
                            onLogs = { pageName = SettingsPage.LOGS.name },
                        )
                        SettingsPage.EDUCATION -> EducationSettings(state, viewModel)
                        SettingsPage.ELECTRICITY -> ElectricitySettings(state, viewModel)
                        SettingsPage.CAMPUS -> CampusSettings(state, viewModel)
                        SettingsPage.APPEARANCE -> AppearanceScreen(state, viewModel)
                        SettingsPage.ABOUT -> AboutScreen()
                        SettingsPage.LOGS -> LogsScreen(state, viewModel)
                    }
                }
            }
        }
    }
}

@Composable
private fun SettingsOverview(
    state: AppState,
    onThemeMode: (String) -> Unit,
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
            icon = Icons.Rounded.DateRange,
            title = stringResource(R.string.section_edu),
            supporting = if (state.credentials.studentId.isBlank()) {
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
            supporting = if (state.credentials.electricityAccount.isBlank()) {
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
            supporting = if (state.credentials.campusUser.isBlank()) {
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
            supporting = if (state.requestLogEnabled) {
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
        color = MaterialTheme.colorScheme.onSurfaceVariant,
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
                imageVector = Icons.AutoMirrored.Rounded.KeyboardArrowRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(21.dp),
            )
        }
    }
}

@Composable
private fun EducationSettings(state: AppState, viewModel: AppViewModel) {
    var studentId by remember(state.credentials.studentId) { mutableStateOf(state.credentials.studentId) }
    var password by remember { mutableStateOf("") }
    var interval by remember(state.timetableInterval) { mutableStateOf(state.timetableInterval.toString()) }
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) password = ""
    }
    CredentialForm(
        title = stringResource(R.string.section_edu),
        note = stringResource(R.string.edu_note),
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
            interval = viewModel.state.value.timetableInterval.toString()
        },
    ) {
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
        Spacer(Modifier.height(18.dp))
        IntervalField(
            value = interval,
            onValueChange = { interval = it },
            label = stringResource(R.string.field_min_interval),
            supporting = stringResource(
                R.string.interval_hint_education,
                AppPreferences.MIN_TIMETABLE_INTERVAL,
            ),
            minimum = AppPreferences.MIN_TIMETABLE_INTERVAL,
        )
    }
}

/**
 * 电费设置：与教务拆开（docs/002 §5.3 tab3）。
 *
 * 为什么单独一页：电费接口无独立密码，但用户可以只查电费、不配教务；
 * 学校代码也属于电费而不是教务。混在一页会让人以为「不填教务密码就查不了电费」。
 */
@Composable
private fun ElectricitySettings(state: AppState, viewModel: AppViewModel) {
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

    CredentialForm(
        title = stringResource(R.string.section_power),
        note = stringResource(R.string.power_no_password_hint),
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
            interval = viewModel.state.value.electricityInterval.toString()
        },
    ) {
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
        Spacer(Modifier.height(6.dp))
        AppTextField(
            // 已保存的凭据里继承时 electricityStudentId 恒为空，所以这里显示教务学号；
            // 未保存的编辑态用本地 ownAccount，不掺 state 里的旧值。
            value = if (inherit) state.credentials.studentId else ownAccount,
            onValueChange = { ownAccount = it },
            label = stringResource(R.string.field_electricity_student_id),
            leadingIcon = TabIcons.User,
            keyboardType = KeyboardType.Number,
            // 继承时不接受编辑：实际用的就是教务学号，允许改会造成
            //「显示的学号」与「真正请求的学号」不一致。
            enabled = !inherit,
            supporting = if (inherit) stringResource(R.string.field_inherit_edu_value) else null,
        )
        Spacer(Modifier.height(14.dp))
        AppTextField(
            value = customerCode,
            onValueChange = { customerCode = it },
            label = stringResource(R.string.field_customer_code),
            leadingIcon = TabIcons.School,
            keyboardType = KeyboardType.Number,
        )
        Spacer(Modifier.height(18.dp))
        IntervalField(
            value = interval,
            onValueChange = { interval = it },
            label = stringResource(R.string.field_min_interval),
            supporting = stringResource(
                R.string.interval_hint_electricity,
                AppPreferences.MIN_ELECTRICITY_INTERVAL,
            ),
            minimum = AppPreferences.MIN_ELECTRICITY_INTERVAL,
        )
    }
}

@Composable
private fun CampusSettings(state: AppState, viewModel: AppViewModel) {
    var user by remember(state.credentials.campusUser) { mutableStateOf(state.credentials.campusUser) }
    var password by remember { mutableStateOf("") }
    var fingerprint by remember(state.credentials.campusFingerprint) {
        mutableStateOf(state.credentials.campusFingerprint)
    }
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) password = ""
    }
    CredentialForm(
        title = stringResource(R.string.section_portal),
        note = stringResource(R.string.campus_independent_hint),
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
    ) {
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
        Spacer(Modifier.height(14.dp))
        AppTextField(
            value = fingerprint,
            onValueChange = { fingerprint = it },
            label = stringResource(R.string.field_campus_fingerprint),
            leadingIcon = TabIcons.Fingerprint,
            supporting = stringResource(R.string.field_optional),
            imeAction = ImeAction.Done,
        )
        Spacer(Modifier.height(18.dp))
        // 开关即改即生效，不必等「保存」：它不影响凭据，只影响何时去探测。
        SettingsSwitchRow(
            title = stringResource(R.string.field_campus_wifi_only),
            supporting = stringResource(R.string.field_campus_wifi_only_hint),
            checked = state.campusWifiOnly,
            onCheckedChange = { viewModel.setCampusWifiOnly(it) },
        )
    }
}

@Composable
private fun CredentialForm(
    title: String,
    note: String,
    saved: Boolean,
    onSavedShown: () -> Unit,
    onSave: () -> Unit,
    content: @Composable ColumnScope.() -> Unit,
) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = note,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            lineHeight = 18.sp,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
        )

        Spacer(Modifier.height(12.dp))

        GroupSurface(position = GroupPosition.ONLY, modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 22.dp)) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(bottom = 16.dp),
                )
                content()
            }
        }

        LaunchedEffect(saved) {
            if (saved) {
                kotlinx.coroutines.delay(1800)
                onSavedShown()
            }
        }
        if (saved) {
            Row(
                modifier = Modifier.padding(top = 16.dp, start = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = Icons.Rounded.CheckCircle,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.width(6.dp))
                Text(
                    text = stringResource(R.string.save_done),
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.primary,
                    fontWeight = FontWeight.SemiBold,
                )
            }
        }

        androidx.compose.material3.Button(
            onClick = onSave,
            shape = CircleShape,
            modifier = Modifier
                .fillMaxWidth()
                .padding(vertical = 24.dp)
                .height(52.dp),
        ) {
            Text(
                text = stringResource(R.string.action_save),
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
            )
        }
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
        supporting = if (invalid) {
            stringResource(R.string.interval_clamped, minimum, AppPreferences.INTERVAL_UNIT)
        } else {
            supporting
        },
    )
}

/** 设置页开关行：标题 + 说明 + Switch，整行可点。 */
@Composable
private fun SettingsSwitchRow(
    title: String,
    supporting: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(appShape(PageCorners.Field))
            .clickable { onCheckedChange(!checked) }
            .padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = supporting,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
        Spacer(Modifier.width(12.dp))
        Switch(checked = checked, onCheckedChange = onCheckedChange)
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
        leadingIcon = leadingIcon?.let {
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
        keyboardOptions = KeyboardOptions(
            keyboardType = if (isPassword) KeyboardType.Password else keyboardType,
            imeAction = imeAction,
        ),
        colors = androidx.compose.material3.OutlinedTextFieldDefaults.colors(
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

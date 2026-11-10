package com.github.echoping07.kemaolife.ui

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.FastOutSlowInEasing
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import com.github.echoping07.kemaolife.ui.theme.smoothShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.automirrored.rounded.KeyboardArrowRight
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.DateRange
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.TextButton
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

private enum class SettingsPage { ROOT, EDUCATION, ELECTRICITY, CAMPUS, APPEARANCE, ABOUT }

/** 设置总览只承担导航；凭据表单放进各自页面，避免三个功能域一次全部铺开。 */
@Composable
internal fun SettingsScreen(state: AppState, viewModel: AppViewModel) {
    var pageName by rememberSaveable { mutableStateOf(SettingsPage.ROOT.name) }
    val page = SettingsPage.valueOf(pageName)
    BackHandler(enabled = page != SettingsPage.ROOT) { pageName = SettingsPage.ROOT.name }

    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.padding(start = 12.dp, end = 16.dp, top = 8.dp, bottom = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (page != SettingsPage.ROOT) {
                IconButton(
                    onClick = { pageName = SettingsPage.ROOT.name },
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.surfaceContainerHigh),
                ) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Rounded.ArrowBack,
                        contentDescription = stringResource(R.string.action_back),
                        tint = MaterialTheme.colorScheme.onSurface,
                        modifier = Modifier.size(20.dp),
                    )
                }
                Spacer(Modifier.width(8.dp))
            } else {
                Spacer(Modifier.width(4.dp))
            }
            Text(
                text = stringResource(
                    when (page) {
                        SettingsPage.ROOT -> R.string.tab_settings
                        SettingsPage.EDUCATION -> R.string.section_edu
                        SettingsPage.ELECTRICITY -> R.string.section_power
                        SettingsPage.CAMPUS -> R.string.section_portal
                        SettingsPage.APPEARANCE -> R.string.settings_appearance
                        SettingsPage.ABOUT -> R.string.settings_about
                    },
                ),
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold,
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 6.dp, vertical = 8.dp),
            )
            // 教务 / 电费 / 校园网页右上角都给一个「测试」，点击即开弹窗跑对应链路。
            val testTarget = when (page) {
                SettingsPage.EDUCATION -> ProbeTarget.TIMETABLE
                SettingsPage.ELECTRICITY -> ProbeTarget.ELECTRICITY
                SettingsPage.CAMPUS -> ProbeTarget.CAMPUS
                else -> null
            }
            if (testTarget != null) {
                IconButton(onClick = { viewModel.runProbe(testTarget) }) {
                    Icon(
                        imageVector = TabIcons.Flask,
                        contentDescription = stringResource(R.string.action_test),
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(22.dp),
                    )
                }
            }
        }

        AnimatedContent(
            targetState = page,
            transitionSpec = {
                if (targetState != SettingsPage.ROOT) {
                    slideInHorizontally(tween(280, easing = FastOutSlowInEasing)) { it / 3 } +
                        fadeIn(tween(220)) togetherWith
                        slideOutHorizontally(tween(280, easing = FastOutSlowInEasing)) { -it / 5 } +
                        fadeOut(tween(160))
                } else {
                    slideInHorizontally(tween(280, easing = FastOutSlowInEasing)) { -it / 3 } +
                        fadeIn(tween(220)) togetherWith
                        slideOutHorizontally(tween(280, easing = FastOutSlowInEasing)) { it / 5 } +
                        fadeOut(tween(160))
                }
            },
            label = "settingsNavigation",
            modifier = Modifier.weight(1f),
        ) { currentPage ->
            when (currentPage) {
                SettingsPage.ROOT -> SettingsOverview(
                    state = state,
                    onEducation = { pageName = SettingsPage.EDUCATION.name },
                    onElectricity = { pageName = SettingsPage.ELECTRICITY.name },
                    onCampus = { pageName = SettingsPage.CAMPUS.name },
                    onAppearance = { pageName = SettingsPage.APPEARANCE.name },
                    onAbout = { pageName = SettingsPage.ABOUT.name },
                )
                SettingsPage.EDUCATION -> EducationSettings(state, viewModel)
                SettingsPage.ELECTRICITY -> ElectricitySettings(state, viewModel)
                SettingsPage.CAMPUS -> CampusSettings(state, viewModel)
                SettingsPage.APPEARANCE -> AppearanceScreen(state, viewModel)
                SettingsPage.ABOUT -> AboutScreen()
            }
        }
    }
}

@Composable
private fun SettingsOverview(
    state: AppState,
    onEducation: () -> Unit,
    onElectricity: () -> Unit,
    onCampus: () -> Unit,
    onAppearance: () -> Unit,
    onAbout: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        Surface(
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            shape = MaterialTheme.shapes.extraLarge,
            modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
        ) {
            Row(
                modifier = Modifier.padding(18.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(42.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = TabIcons.Lock,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(20.dp),
                    )
                }
                Spacer(Modifier.width(14.dp))
                Column {
                    Text(
                        text = "硬件级加密存储",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = stringResource(R.string.credentials_keystore_hint),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 2.dp),
                    )
                }
            }
        }

        SettingsGroupLabel(stringResource(R.string.settings_accounts))
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
            shape = MaterialTheme.shapes.extraLarge,
        ) {
            SettingsDestination(
                icon = Icons.Rounded.DateRange,
                title = stringResource(R.string.section_edu),
                supporting = if (state.credentials.studentId.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else {
                    stringResource(R.string.settings_edu_configured)
                },
                onClick = onEducation,
            )
            ThinDivider(modifier = Modifier.padding(horizontal = 18.dp))
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
            )
            ThinDivider(modifier = Modifier.padding(horizontal = 18.dp))
            SettingsDestination(
                icon = TabIcons.Campus,
                title = stringResource(R.string.section_portal),
                supporting = if (state.credentials.campusUser.isBlank()) {
                    stringResource(R.string.settings_not_configured)
                } else {
                    stringResource(R.string.settings_campus_configured)
                },
                onClick = onCampus,
            )
        }

        SettingsGroupLabel(stringResource(R.string.settings_personalization))
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
            shape = MaterialTheme.shapes.extraLarge,
        ) {
            SettingsDestination(
                icon = TabIcons.Appearance,
                title = stringResource(R.string.settings_appearance),
                supporting = stringResource(R.string.settings_theme_color),
                onClick = onAppearance,
            )
            ThinDivider(modifier = Modifier.padding(horizontal = 18.dp))
            SettingsDestination(
                icon = TabIcons.Info,
                title = stringResource(R.string.settings_about),
                supporting = stringResource(R.string.settings_about_hint),
                onClick = onAbout,
            )
        }

        if (!state.bridgeReady && state.bridgeError != null) {
            Text(
                text = state.bridgeError,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.padding(16.dp),
            )
        }
        Spacer(Modifier.height(28.dp))
    }
}

@Composable
private fun SettingsGroupLabel(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelLarge,
        fontWeight = FontWeight.Bold,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(start = 12.dp, top = 22.dp, bottom = 8.dp),
    )
}

/** MD3 Expressive list 项：整洁卡片内部条目。 */
@Composable
private fun SettingsDestination(
    icon: ImageVector,
    title: String,
    supporting: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 18.dp, vertical = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        BoxIcon(icon)
        Spacer(Modifier.width(16.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = supporting,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
        Icon(
            imageVector = Icons.AutoMirrored.Rounded.KeyboardArrowRight,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.size(20.dp),
        )
    }
}

@Composable
private fun BoxIcon(icon: ImageVector) {
    Box(
        modifier = Modifier
            .size(40.dp)
            .clip(CircleShape)
            .background(MaterialTheme.colorScheme.surfaceContainerHigh),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(20.dp),
        )
    }
}

@Composable
private fun EducationSettings(state: AppState, viewModel: AppViewModel) {
    var studentId by remember(state.credentials.studentId) { mutableStateOf(state.credentials.studentId) }
    var password by remember { mutableStateOf("") }
    var interval by remember(state.timetableInterval) { mutableStateOf(state.timetableInterval.toString()) }
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) {
            password = ""
            viewModel.consumeSettingsSaved()
        }
    }
    CredentialForm(
        title = stringResource(R.string.section_edu),
        note = stringResource(R.string.edu_note),
        saved = state.settingsSaved,
        onSave = {
            // 增量提交：基于 ViewModel 里的最新凭据合并，不拿组合时的快照覆盖其它域。
            val student = studentId.trim()
            val pwd = password
            viewModel.updateCredentials {
                it.copy(studentId = student, eduPassword = pwd)
            }
            // 间隔先落到偏好，保证「保存」与「应用间隔」两条路径结果一致（含钳制）。
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
            onCommit = { minutes ->
                viewModel.setTimetableInterval(minutes)
                interval = viewModel.state.value.timetableInterval.toString()
            },
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
            onCommit = { minutes ->
                viewModel.setElectricityInterval(minutes)
                interval = viewModel.state.value.electricityInterval.toString()
            },
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
        if (state.settingsSaved) {
            password = ""
            viewModel.consumeSettingsSaved()
        }
    }
    CredentialForm(
        title = stringResource(R.string.section_portal),
        note = stringResource(R.string.campus_independent_hint),
        saved = state.settingsSaved,
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
    onSave: () -> Unit,
    content: @Composable ColumnScope.() -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        // 安全提示横幅卡片
        Surface(
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            shape = MaterialTheme.shapes.extraLarge,
            modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
        ) {
            Row(
                modifier = Modifier.padding(18.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(38.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primaryContainer),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        imageVector = TabIcons.Lock,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(18.dp),
                    )
                }
                Spacer(Modifier.width(14.dp))
                Text(
                    text = note,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    lineHeight = 18.sp,
                    modifier = Modifier.weight(1f),
                )
            }
        }

        Spacer(Modifier.height(16.dp))

        // 表单卡片容器
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = androidx.compose.material3.CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
            ),
            shape = MaterialTheme.shapes.extraLarge,
        ) {
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
 * 失焦 / 完成时提交，由 ViewModel 按上下限钳制并回填实际生效值。
 */
@Composable
private fun IntervalField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    supporting: String,
    minimum: Int,
    onCommit: (Int) -> Unit,
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
    // 键盘「完成」不会触发焦点变化，所以再加一个显式的提交按钮，
    // 保证改完立刻生效、并且能看到被钳制后的实际值。
    Row(
        modifier = Modifier.fillMaxWidth().padding(top = 6.dp),
        horizontalArrangement = androidx.compose.foundation.layout.Arrangement.End,
    ) {
        TextButton(onClick = { onCommit(parsed ?: minimum) }) {
            Text(stringResource(R.string.action_apply_interval))
        }
    }
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
            .clip(smoothShape(16.dp))
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
            focusedContainerColor = MaterialTheme.colorScheme.surface,
            unfocusedContainerColor = MaterialTheme.colorScheme.surface,
            focusedBorderColor = MaterialTheme.colorScheme.primary,
            unfocusedBorderColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.5f),
        ),
        shape = smoothShape(16.dp),
        modifier = Modifier.fillMaxWidth(),
    )
}

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
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material.icons.rounded.Settings
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
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
import com.github.echoping07.kemaolife.data.Credentials

private enum class SettingsPage { ROOT, EDUCATION, CAMPUS, APPEARANCE, ABOUT }

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
                        SettingsPage.CAMPUS -> R.string.section_portal
                        SettingsPage.APPEARANCE -> R.string.settings_appearance
                        SettingsPage.ABOUT -> R.string.settings_about
                    },
                ),
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.padding(horizontal = 6.dp, vertical = 8.dp),
            )
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
                    onCampus = { pageName = SettingsPage.CAMPUS.name },
                    onAppearance = { pageName = SettingsPage.APPEARANCE.name },
                    onAbout = { pageName = SettingsPage.ABOUT.name },
                )
                SettingsPage.EDUCATION -> EducationSettings(state, viewModel)
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

        SettingsGroupLabel(stringResource(R.string.settings_electricity_title))
        Surface(
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            shape = MaterialTheme.shapes.extraLarge,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 18.dp)) {
                Text(
                    text = stringResource(R.string.power_no_password_hint),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = stringResource(R.string.settings_electricity_source),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
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
    var customerCode by remember(state.credentials.customerCode) {
        mutableStateOf(state.credentials.customerCode)
    }
    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) {
            password = ""
            viewModel.consumeSettingsSaved()
        }
    }
    CredentialForm(
        title = "教务与电费凭据",
        note = stringResource(R.string.edu_note),
        saved = state.settingsSaved,
        onSave = {
            viewModel.saveCredentials(
                state.credentials.copy(
                    studentId = studentId.trim(),
                    eduPassword = password,
                    customerCode = customerCode.trim().ifBlank { Credentials.DEFAULT_CUSTOMER_CODE },
                ),
            )
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
        )
        Spacer(Modifier.height(18.dp))
        Text(
            text = stringResource(R.string.section_power),
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary,
        )
        Text(
            text = stringResource(R.string.power_no_password_hint),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 4.dp, bottom = 10.dp),
        )
        AppTextField(
            value = customerCode,
            onValueChange = { customerCode = it },
            label = stringResource(R.string.field_customer_code),
            leadingIcon = TabIcons.School,
            keyboardType = KeyboardType.Number,
            imeAction = ImeAction.Done,
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
        title = "校园网认证凭据",
        note = stringResource(R.string.campus_independent_hint),
        saved = state.settingsSaved,
        onSave = {
            viewModel.saveCredentials(
                state.credentials.copy(
                    campusUser = user.trim(),
                    campusPassword = password,
                    campusFingerprint = fingerprint.trim(),
                ),
            )
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
) {
    var passwordVisible by rememberSaveable { mutableStateOf(false) }

    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
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

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
            modifier = Modifier.padding(start = 8.dp, end = 16.dp, top = 2.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (page != SettingsPage.ROOT) {
                IconButton(onClick = { pageName = SettingsPage.ROOT.name }) {
                    Icon(
                        imageVector = Icons.AutoMirrored.Rounded.ArrowBack,
                        contentDescription = stringResource(R.string.action_back),
                    )
                }
            } else {
                Spacer(Modifier.width(8.dp))
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
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(horizontal = 8.dp, vertical = 12.dp),
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
            color = MaterialTheme.colorScheme.primaryContainer,
            shape = MaterialTheme.shapes.large,
        ) {
            Row(
                modifier = Modifier.padding(20.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = Icons.Rounded.Lock,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onPrimaryContainer,
                )
                Spacer(Modifier.width(14.dp))
                Text(
                    text = stringResource(R.string.credentials_keystore_hint),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onPrimaryContainer,
                )
            }
        }

        SettingsGroupLabel(stringResource(R.string.settings_accounts))
        SettingsGroup {
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
            Spacer(Modifier.height(2.dp))
            SettingsDestination(
                icon = TabIcons.Campus,
                shapePosition = SettingsShapePosition.LAST,
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
        SettingsGroup {
            SettingsDestination(
                icon = TabIcons.Appearance,
                title = stringResource(R.string.settings_appearance),
                supporting = stringResource(R.string.settings_theme_color),
                onClick = onAppearance,
            )
            Spacer(Modifier.height(2.dp))
            SettingsDestination(
                icon = TabIcons.Info,
                shapePosition = SettingsShapePosition.LAST,
                title = stringResource(R.string.settings_about),
                supporting = stringResource(R.string.settings_about_hint),
                onClick = onAbout,
            )
        }

        SettingsGroupLabel(stringResource(R.string.settings_electricity_title))
        Surface(
            color = MaterialTheme.colorScheme.surfaceContainerLow,
            shape = MaterialTheme.shapes.large,
        ) {
            Column(modifier = Modifier.padding(horizontal = 20.dp, vertical = 16.dp)) {
                Text(
                    text = stringResource(R.string.power_no_password_hint),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
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
    }
}

@Composable
private fun SettingsGroupLabel(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.padding(start = 12.dp, top = 24.dp, bottom = 8.dp),
    )
}

@Composable
private fun SettingsGroup(content: @Composable ColumnScope.() -> Unit) {
    Column(content = content)
}

private enum class SettingsShapePosition { SINGLE, FIRST, MIDDLE, LAST }

/** MD3 Expressive list：同一组使用 4dp 小圆角，首尾用 28dp 大圆角，项间留 2dp。 */
@Composable
private fun SettingsDestination(
    icon: ImageVector,
    title: String,
    supporting: String,
    onClick: () -> Unit,
    shapePosition: SettingsShapePosition = SettingsShapePosition.FIRST,
) {
    val shape = when (shapePosition) {
        SettingsShapePosition.SINGLE -> MaterialTheme.shapes.large
        SettingsShapePosition.FIRST -> smoothShape(28.dp, 4.dp)
        SettingsShapePosition.MIDDLE -> smoothShape(4.dp)
        SettingsShapePosition.LAST -> smoothShape(4.dp, 28.dp)
    }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surfaceContainerLow, shape)
            .clip(shape)
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        BoxIcon(icon)
        Spacer(Modifier.width(16.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(text = title, style = MaterialTheme.typography.titleMedium)
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
        )
    }
}

@Composable
private fun BoxIcon(icon: ImageVector) {
    Icon(
        imageVector = icon,
        contentDescription = null,
        tint = MaterialTheme.colorScheme.primary,
        modifier = Modifier
            .size(36.dp)
            .clip(CircleShape)
            .background(MaterialTheme.colorScheme.secondaryContainer)
            .padding(8.dp),
    )
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
            keyboardType = KeyboardType.Number,
        )
        AppTextField(
            value = password,
            onValueChange = { password = it },
            label = stringResource(R.string.field_edu_password),
            isPassword = true,
            supporting = stringResource(R.string.password_kept_hint),
        )
        SettingsGroupLabel(stringResource(R.string.section_power))
        FieldNote(stringResource(R.string.power_no_password_hint))
        AppTextField(
            value = customerCode,
            onValueChange = { customerCode = it },
            label = stringResource(R.string.field_customer_code),
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
        )
        AppTextField(
            value = password,
            onValueChange = { password = it },
            label = stringResource(R.string.field_campus_password),
            isPassword = true,
            supporting = stringResource(R.string.password_kept_hint),
        )
        AppTextField(
            value = fingerprint,
            onValueChange = { fingerprint = it },
            label = stringResource(R.string.field_campus_fingerprint),
            supporting = stringResource(R.string.field_optional),
            imeAction = ImeAction.Done,
        )
    }
}

@Composable
private fun CredentialForm(
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
        FieldNote(note)
        Spacer(Modifier.padding(top = 16.dp))
        content()
        if (saved) {
            Row(
                modifier = Modifier.padding(top = 16.dp),
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
                )
            }
        }
        androidx.compose.material3.Button(
            onClick = onSave,
            modifier = Modifier.fillMaxWidth().padding(vertical = 24.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 14.dp),
        ) { Text(stringResource(R.string.action_save)) }
    }
}

@Composable
private fun FieldNote(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(horizontal = 4.dp),
    )
}

/** 密码只保存在当前 composition，不进 Bundle；留空时仍沿用旧密码。 */
@Composable
private fun AppTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    supporting: String? = null,
    isPassword: Boolean = false,
    keyboardType: KeyboardType = KeyboardType.Text,
    imeAction: ImeAction = ImeAction.Next,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label) },
        supportingText = supporting?.let { { Text(it) } },
        singleLine = true,
        visualTransformation = if (isPassword) PasswordVisualTransformation() else VisualTransformation.None,
        keyboardOptions = KeyboardOptions(
            keyboardType = if (isPassword) KeyboardType.Password else keyboardType,
            imeAction = imeAction,
        ),
        shape = MaterialTheme.shapes.medium,
        modifier = Modifier.fillMaxWidth().padding(top = 14.dp),
    )
}

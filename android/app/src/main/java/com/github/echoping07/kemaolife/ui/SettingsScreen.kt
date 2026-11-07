package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.data.Credentials

/**
 * tab3 · 设置：分功能域录入账号密码（docs/002 §5.3）。
 *
 * 两条硬约束：
 *  1. 密码不回显：输入框恒用 PasswordVisualTransformation；
 *  2. 密码留空表示「不修改」：保存时由 CredentialStore 把旧值补回去。
 *
 * 凭据的加密与持久化全在 Kotlin（Keystore），Go 只拿解密后的内存值。
 *
 * 排版：每个功能域一张 filled card，说明文字放在卡片头部而不是塞进
 * `supportingText` —— 输入框下方每多一行小字，表单就显得越碎。
 */
@Composable
internal fun SettingsScreen(state: AppState, viewModel: AppViewModel) {
    // 用当前凭据做输入框初值；密码框始终留空，避免把已存密码回填到 UI。
    var studentId by remember(state.credentials.studentId) {
        mutableStateOf(state.credentials.studentId)
    }
    var eduPassword by remember { mutableStateOf("") }
    var customerCode by remember(state.credentials.customerCode) {
        mutableStateOf(state.credentials.customerCode)
    }
    var campusUser by remember(state.credentials.campusUser) {
        mutableStateOf(state.credentials.campusUser)
    }
    var campusPassword by remember { mutableStateOf("") }
    var campusFingerprint by remember(state.credentials.campusFingerprint) {
        mutableStateOf(state.credentials.campusFingerprint)
    }

    LaunchedEffect(state.settingsSaved) {
        if (state.settingsSaved) {
            // 保存成功后清空密码框：明文不该在输入框里长期驻留。
            eduPassword = ""
            campusPassword = ""
            viewModel.consumeSettingsSaved()
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
    ) {
        ScreenTitle(stringResource(R.string.tab_settings))

        // Keystore 说明放在最上面一行，作为整页的前提，而不是塞在某张卡片里。
        Text(
            text = stringResource(R.string.credentials_keystore_hint),
            style = MaterialTheme.typography.bodySmall,
            fontSize = 12.sp,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(start = 4.dp, bottom = 12.dp),
        )

        if (state.settingsSaved) {
            SavedBadge()
            Spacer(Modifier.height(12.dp))
        }

        // ---- 教务 ----
        SectionCard(title = stringResource(R.string.section_edu)) {
            FieldNote(stringResource(R.string.edu_note))
            Spacer(Modifier.height(10.dp))
            AppTextField(
                value = studentId,
                onValueChange = { studentId = it },
                label = stringResource(R.string.field_student_id),
                keyboardType = KeyboardType.Number,
            )
            Spacer(Modifier.height(10.dp))
            AppTextField(
                value = eduPassword,
                onValueChange = { eduPassword = it },
                label = stringResource(R.string.field_edu_password),
                isPassword = true,
                supporting = stringResource(R.string.password_kept_hint),
                imeAction = ImeAction.Next,
            )
        }

        Spacer(Modifier.height(12.dp))

        // ---- 电费 ----
        SectionCard(title = stringResource(R.string.section_power)) {
            FieldNote(stringResource(R.string.power_no_password_hint))
            Spacer(Modifier.height(10.dp))
            AppTextField(
                value = customerCode,
                onValueChange = { customerCode = it },
                label = stringResource(R.string.field_customer_code),
                keyboardType = KeyboardType.Number,
            )
        }

        Spacer(Modifier.height(12.dp))

        // ---- 校园网 ----
        SectionCard(title = stringResource(R.string.section_portal)) {
            FieldNote(stringResource(R.string.campus_independent_hint))
            Spacer(Modifier.height(10.dp))
            AppTextField(
                value = campusUser,
                onValueChange = { campusUser = it },
                label = stringResource(R.string.field_campus_user),
            )
            Spacer(Modifier.height(10.dp))
            AppTextField(
                value = campusPassword,
                onValueChange = { campusPassword = it },
                label = stringResource(R.string.field_campus_password),
                isPassword = true,
                supporting = stringResource(R.string.password_kept_hint),
            )
            Spacer(Modifier.height(10.dp))
            AppTextField(
                value = campusFingerprint,
                onValueChange = { campusFingerprint = it },
                label = stringResource(R.string.field_campus_fingerprint),
                supporting = stringResource(R.string.field_optional),
                imeAction = ImeAction.Done,
            )
        }

        if (!state.bridgeReady && state.bridgeError != null) {
            Spacer(Modifier.height(12.dp))
            Text(
                text = state.bridgeError!!,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        Spacer(Modifier.height(20.dp))
        Button(
            onClick = {
                viewModel.saveCredentials(
                    Credentials(
                        studentId = studentId.trim(),
                        eduPassword = eduPassword,
                        customerCode = customerCode.trim()
                            .ifBlank { Credentials.DEFAULT_CUSTOMER_CODE },
                        campusUser = campusUser.trim(),
                        campusPassword = campusPassword,
                        campusFingerprint = campusFingerprint.trim(),
                    ),
                )
            },
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
        ) {
            Text(
                text = stringResource(R.string.action_save),
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
        }

        Spacer(Modifier.height(32.dp))
    }
}

@Composable
private fun SavedBadge() {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Icon(
            imageVector = Icons.Rounded.CheckCircle,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(18.dp),
        )
        Text(
            text = stringResource(R.string.save_done),
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.primary,
        )
        Spacer(Modifier.weight(1f))
    }
}

/** 卡片头部的一句话说明：解释「这块为什么只填这些」。 */
@Composable
private fun FieldNote(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodySmall,
        fontSize = 12.sp,
        lineHeight = 17.sp,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
}

/** 统一外观的输入框：filled 变体 + 固定高度，避免各字段看着像不同组件。 */
@Composable
private fun AppTextField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    supporting: String? = null,
    isPassword: Boolean = false,
    keyboardType: KeyboardType = KeyboardType.Text,
    imeAction: ImeAction = ImeAction.Next,
) {
    // Filled 变体：M3 的 filled text field 用容器色而不是描边，
    // 一屏十来个输入框时不会变成一堆黑框（描边版在本页显得很重）。
    TextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(label) },
        supportingText = supporting?.let { { Text(it, fontSize = 11.sp) } },
        singleLine = true,
        visualTransformation = if (isPassword) {
            PasswordVisualTransformation()
        } else {
            VisualTransformation.None
        },
        keyboardOptions = KeyboardOptions(
            keyboardType = if (isPassword) KeyboardType.Password else keyboardType,
            imeAction = imeAction,
        ),
        shape = MaterialTheme.shapes.medium,
        colors = TextFieldDefaults.colors(
            focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
            unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
            disabledContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
            focusedIndicatorColor = MaterialTheme.colorScheme.primary,
            unfocusedIndicatorColor = Color.Transparent,
            disabledIndicatorColor = Color.Transparent,
        ),
        modifier = modifier.fillMaxWidth(),
    )
}

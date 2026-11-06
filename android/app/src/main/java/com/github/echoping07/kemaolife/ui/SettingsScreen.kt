package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
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
        Spacer(Modifier.height(12.dp))

        Text(
            text = stringResource(R.string.credentials_keystore_hint),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        Spacer(Modifier.height(16.dp))
        SectionTitle(stringResource(R.string.section_edu))
        OutlinedTextField(
            value = studentId,
            onValueChange = { studentId = it },
            label = { Text(stringResource(R.string.field_student_id)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Number,
                imeAction = ImeAction.Next,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = eduPassword,
            onValueChange = { eduPassword = it },
            label = { Text(stringResource(R.string.field_edu_password)) },
            supportingText = { Text(stringResource(R.string.password_kept_hint)) },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Password,
                imeAction = ImeAction.Next,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(20.dp))
        SectionTitle(stringResource(R.string.section_power))
        OutlinedTextField(
            value = customerCode,
            onValueChange = { customerCode = it },
            label = { Text(stringResource(R.string.field_customer_code)) },
            supportingText = { Text(stringResource(R.string.power_no_password_hint)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Number,
                imeAction = ImeAction.Next,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(20.dp))
        SectionTitle(stringResource(R.string.section_portal))
        OutlinedTextField(
            value = campusUser,
            onValueChange = { campusUser = it },
            label = { Text(stringResource(R.string.field_campus_user)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = campusPassword,
            onValueChange = { campusPassword = it },
            label = { Text(stringResource(R.string.field_campus_password)) },
            supportingText = { Text(stringResource(R.string.campus_independent_hint)) },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Password,
                imeAction = ImeAction.Next,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = campusFingerprint,
            onValueChange = { campusFingerprint = it },
            label = { Text(stringResource(R.string.field_campus_fingerprint)) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(24.dp))
        Row(modifier = Modifier.fillMaxWidth()) {
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
                modifier = Modifier.fillMaxWidth(),
            ) { Text(stringResource(R.string.action_save)) }
        }

        if (!state.bridgeReady && state.bridgeError != null) {
            Spacer(Modifier.height(12.dp))
            Text(
                text = state.bridgeError,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }

        Spacer(Modifier.height(32.dp))
    }
}

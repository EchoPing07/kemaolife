package com.github.echoping07.kemaolife.ui

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import com.github.echoping07.kemaolife.ui.theme.appShape

/**
 * 关于页「检查更新」的半屏底部弹层。
 *
 * 高度固定为屏幕的一半且禁止部分展开态：M3 默认的部分展开态会截断内容，
 * 用户还得自己上拖才看得到底部按钮。结果区占满剩余空间可滚，按钮恒定贴底。
 *
 * 只展示当前版本 / 远端版本 / 一句结论，不拉 Release notes、不列资产、
 * 不下载安装包；检查失败也没有重试按钮 —— 关掉弹层重新点一次即可重查。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun UpdateCheckSheet(
    state: UpdateCheckState,
    onDismiss: () -> Unit,
    onOpenReleases: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val screenHeight = LocalConfiguration.current.screenHeightDp.dp
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        shape = appShape(PageCorners.Card),
        containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .height(screenHeight * 0.5f)
                .padding(horizontal = 20.dp),
        ) {
            Text(
                text = stringResource(R.string.update_sheet_title),
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(Modifier.height(16.dp))

            Column(
                modifier = Modifier
                    .weight(1f)
                    .verticalScroll(rememberScrollState()),
            ) {
                if (state.status == UpdateCheckStatus.RUNNING) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        ExpressiveLoadingIndicator(size = 24.dp)
                        Spacer(Modifier.width(14.dp))
                        Text(
                            text = stringResource(R.string.update_checking),
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                } else {
                    VersionField(
                        label = stringResource(R.string.update_current_version),
                        value = state.currentVersion,
                    )
                    VersionField(
                        label = stringResource(R.string.update_remote_version),
                        value = state.remoteVersion ?: stringResource(R.string.update_value_unknown),
                    )
                    Spacer(Modifier.height(16.dp))
                    val (conclusion, color) = when (state.status) {
                        UpdateCheckStatus.UPDATE_AVAILABLE -> stringResource(R.string.update_available) to
                            MaterialTheme.colorScheme.primary
                        UpdateCheckStatus.UP_TO_DATE -> stringResource(R.string.update_up_to_date) to
                            MaterialTheme.colorScheme.onSurface
                        UpdateCheckStatus.FAILED -> stringResource(R.string.update_failed) to
                            MaterialTheme.colorScheme.error
                        else -> "" to MaterialTheme.colorScheme.onSurface
                    }
                    Text(
                        text = conclusion,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.SemiBold,
                        color = color,
                    )
                    Spacer(Modifier.height(8.dp))
                    Text(
                        text = stringResource(R.string.update_sheet_note),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }

            // 底部唯一动作：无论检查结果如何都可点，它只是外链，不依赖检查结果。
            Button(
                onClick = onOpenReleases,
                shape = CircleShape,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
            ) {
                Icon(
                    imageVector = TabIcons.Export,
                    contentDescription = null,
                    modifier = Modifier.size(20.dp),
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = stringResource(R.string.update_go_releases),
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Bold,
                )
            }
            Spacer(Modifier.height(24.dp))
        }
    }
}

@Composable
private fun VersionField(label: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 7.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.weight(1f),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
        )
    }
}

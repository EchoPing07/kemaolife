package com.github.echoping07.kemaolife.ui

import android.content.Context
import android.net.Uri
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.github.echoping07.kemaolife.R
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * 导出弹层：确认文件名 / 数据时间后再落盘。
 *
 * 为什么不是「点一下就静默写到某个目录」：Android 10+ 没有通用可写公共目录
 * （分区存储），猜路径必然在某些机型上失败；用系统文件选择器（SAF）让用户
 * 自己选位置，既不需要任何存储权限，也不会把文件写进用户找不到的地方。
 *
 * 「分享到微信 / QQ」由用户在文件保存后用系统能力完成，
 * 应用内不再接一套分享面板（WakeUp 的导入链路本身就是从文件管理器 /
 * 聊天应用打开 CSV）。
 */
@Composable
internal fun TimetableExportDialog(state: ExportState, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val export = state.export
    var saving by remember { mutableStateOf(false) }

    val picker = rememberLauncherForActivityResult(
        ActivityResultContracts.CreateDocument(CSV_MIME_TYPE),
    ) { uri ->
        if (uri == null) {
            // 用户取消选择：留在弹层里，让他能再点一次保存。
            saving = false
            return@rememberLauncherForActivityResult
        }
        scope.launch {
            // 正文在 IO 线程编码一次，写盘也在 IO 线程。
            val bytes = withContext(Dispatchers.IO) { export.csv.toByteArray(Charsets.UTF_8) }
            val ok = withContext(Dispatchers.IO) { writeCsv(context, uri, bytes) }
            saving = false
            // 保存结果用 Toast 而不是 snackbar：弹层马上关闭，snackbar 会跟着消失。
            Toast.makeText(
                context,
                context.getString(if (ok) R.string.export_saved else R.string.export_failed),
                Toast.LENGTH_SHORT,
            ).show()
            if (ok) onDismiss()
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.export_title)) },
        text = {
            Column(modifier = Modifier.fillMaxWidth()) {
                ExportRow(stringResource(R.string.export_filename), export.filename)
                ExportRow(
                    stringResource(R.string.export_lessons),
                    stringResource(R.string.export_lessons_value, export.lessonCount),
                )
                // 数据时间必须展示：导出的不是实时数据，用户得知道这是哪次同步的课表。
                if (export.syncedAt.isNotBlank()) {
                    ExportRow(stringResource(R.string.export_synced_at), shortStamp(export.syncedAt))
                }
                Spacer(Modifier.height(10.dp))
                Text(
                    text = stringResource(R.string.export_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = {
            TextButton(
                enabled = !saving,
                onClick = {
                    saving = true
                    // 文件名由 Go 侧给出（带快照同步时间），UI 不自己拼。
                    picker.launch(export.filename)
                },
            ) {
                Text(stringResource(if (saving) R.string.export_saving else R.string.action_save))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(android.R.string.cancel)) }
        },
        shape = MaterialTheme.shapes.extraLarge,
    )
}

@Composable
private fun ExportRow(label: String, value: String) {
    Column(modifier = Modifier.fillMaxWidth().height(46.dp)) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(text = value, style = MaterialTheme.typography.bodyMedium, maxLines = 1)
    }
}

/**
 * 写入 CSV 正文。
 *
 * 优先用 `"wt"` 模式：部分 provider（尤其从模板复制来的文件）会保留旧内容尾部，
 * 截断模式才能保证导出的文件正好是这份课表。但并非所有 provider 都实现 `wt`
 * （会抛 IllegalArgumentException），所以退回普通的 `"w"` —— 宁可少截断，
 * 也不能让用户换个保存位置就失败。BOM 已由 Go 侧写进正文。
 */
private fun writeCsv(context: Context, uri: Uri, bytes: ByteArray): Boolean {
    for (mode in listOf("wt", "w")) {
        try {
            val out = context.contentResolver.openOutputStream(uri, mode) ?: continue
            out.use {
                it.write(bytes)
                it.flush()
            }
            return true
        } catch (e: IllegalArgumentException) {
            // 该 provider 不支持这个模式：换下一个。
            continue
        } catch (e: Exception) {
            // 其它失败（空间不足 / 权限 / provider 异常）不再重试，直接回报失败。
            // 不读取 e.message 之外的内容：路径与 provider 细节不进 UI（P2）。
            return false
        }
    }
    return false
}

private const val CSV_MIME_TYPE = "text/csv"

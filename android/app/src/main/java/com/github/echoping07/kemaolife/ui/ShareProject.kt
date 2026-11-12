package com.github.echoping07.kemaolife.ui

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.widget.Toast
import com.github.echoping07.kemaolife.R

private const val PROJECT_REPOSITORY_URL = "https://github.com/EchoPing07/kemaolife"

/**
 * 分享正文只放公开的项目信息：名称取 app_name，地址是仓库首页。
 * 不拼入任何查询结果 / 凭据 / 日志内容，避免系统分享面板把本机私有数据带出去（P1 / P2）。
 */
internal fun shareProjectText(appName: String): String =
    "KeMao Life · $appName\n项目地址：$PROJECT_REPOSITORY_URL"

/**
 * 拉起系统分享面板，而不是应用内自建一套目标列表：
 * `createChooser` 每次都让用户挑目标，不设默认应用，也不需要查询其它应用（Android 11+ 包可见性）。
 * Activity 上下文以外调用时补 `FLAG_ACTIVITY_NEW_TASK`：Compose 的 `LocalContext` 通常是 Activity，
 * 但保留这一层可避免被非 Activity 上下文复用时抛 `AndroidRuntimeException`。
 */
internal fun shareProject(context: Context, appName: String, chooserTitle: String) {
    val sendIntent = Intent(Intent.ACTION_SEND).apply {
        type = "text/plain"
        putExtra(Intent.EXTRA_TEXT, shareProjectText(appName))
    }
    val chooser = Intent.createChooser(sendIntent, chooserTitle)
    if (context !is Activity) chooser.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
    try {
        context.startActivity(chooser)
    } catch (e: ActivityNotFoundException) {
        // 极少数设备上没有能接收纯文本的应用：提示一次，不让点击直接崩溃。
        Toast.makeText(context, R.string.share_no_target, Toast.LENGTH_SHORT).show()
    }
}

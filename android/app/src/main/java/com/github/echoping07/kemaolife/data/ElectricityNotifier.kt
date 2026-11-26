package com.github.echoping07.kemaolife.data

import android.Manifest
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.github.echoping07.kemaolife.MainActivity
import com.github.echoping07.kemaolife.R

/**
 * Android 系统通知投递封装：只负责渠道创建、权限查询、通知展示与跳转系统通知设置。
 *
 * 阈值判定、去重与正文格式化全由 Go `core/alert.Evaluate` 经 `electricity.alert` 完成（AGENTS A3），
 * 这里不碰任何业务数值；通知正文已由 Go 侧保证带「数据截至」且不含宿舍标识（P2）。
 */
internal object ElectricityNotifier {
    const val CHANNEL_ID = "electricity_alert"
    const val NOTIFICATION_ID = 1001
    const val EXTRA_OPEN_TAB = "kemaolife_open_tab"
    const val TAB_MORE = "more"

    /** 当前是否具备向系统投递通知的权限（兼容 Android 13+ 运行时权限与系统总开关）。 */
    fun hasPermission(context: Context): Boolean {
        val granted =
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                ContextCompat.checkSelfPermission(
                    context,
                    Manifest.permission.POST_NOTIFICATIONS,
                ) == PackageManager.PERMISSION_GRANTED
            } else {
                true
            }
        return granted && NotificationManagerCompat.from(context).areNotificationsEnabled()
    }

    /** 确保通知渠道存在（Android 8.0+ 必需）；幂等，可重复调用。 */
    fun ensureChannel(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        val channel =
            NotificationChannel(
                CHANNEL_ID,
                context.getString(R.string.notification_channel_electricity_name),
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply {
                description = context.getString(R.string.notification_channel_electricity_desc)
            }
        manager.createNotificationChannel(channel)
    }

    /**
     * 投递低电费系统通知；返回是否成功送达通知管理器。
     *
     * 未获通知权限时直接返回 false，调用方据此保留未置位的去重状态，
     * 待用户后续开启权限后仍能收到提醒。
     */
    @SuppressLint("MissingPermission")
    fun post(context: Context, title: String, body: String): Boolean {
        if (!hasPermission(context)) return false
        ensureChannel(context)

        val tapIntent =
            Intent(context, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP
                putExtra(EXTRA_OPEN_TAB, TAB_MORE)
            }
        val pendingIntent =
            PendingIntent.getActivity(
                context,
                NOTIFICATION_ID,
                tapIntent,
                PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
            )

        val summaryLine = body.lineSequence().firstOrNull().orEmpty()
        val notification =
            NotificationCompat.Builder(context, CHANNEL_ID)
                .setSmallIcon(R.drawable.ic_stat_electricity)
                .setContentTitle(title)
                .setContentText(summaryLine)
                .setStyle(NotificationCompat.BigTextStyle().bigText(body))
                .setPriority(NotificationCompat.PRIORITY_DEFAULT)
                .setAutoCancel(true)
                .setContentIntent(pendingIntent)
                .build()

        return runCatching {
            NotificationManagerCompat.from(context).notify(NOTIFICATION_ID, notification)
            true
        }.getOrDefault(false)
    }

    /** 打开本应用的系统通知设置页（用户永久拒绝权限或系统级关通知时引导）。 */
    fun openNotificationSettings(context: Context) {
        val intent =
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).apply {
                    putExtra(Settings.EXTRA_APP_PACKAGE, context.packageName)
                }
            } else {
                Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS).apply {
                    data = Uri.fromParts("package", context.packageName, null)
                }
            }.apply {
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
        runCatching { context.startActivity(intent) }
    }
}

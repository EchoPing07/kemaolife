package com.github.echoping07.kemaolife.ui

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import com.github.echoping07.kemaolife.R

/** 交流群信息只放公开的群号与加群短链，与本仓库 README「交流」一节一致。 */
internal const val QQ_GROUP_NUMBER = "1126474633"

internal const val QQ_GROUP_SHARE_URL = "https://qm.qq.com/q/rf4PfFM5RA"

/**
 * 拉起手机 QQ 打开本项目的交流群。
 *
 * 顺序是「QQ 群卡片 → 加群短链」：先用 QQ 自己的 `mqqapi://card/show_pslcard` 直接打开群卡片
 * （群号是公开信息，不涉及 authKey，应用内不做任何鉴权）；QQ 未安装或版本不认这个 scheme 时，
 * 再退回 `qm.qq.com` 的加群短链，系统会交给浏览器或 QQ 应用内打开同一个群。
 * 两条链接都不携带本机数据（不拼学号 / 快照 / 日志）。
 */
internal fun joinQQGroup(context: Context) {
    val intents = listOf(
        Intent(Intent.ACTION_VIEW, Uri.parse(qqGroupCardUri(QQ_GROUP_NUMBER))),
        Intent(Intent.ACTION_VIEW, Uri.parse(QQ_GROUP_SHARE_URL)),
    )
    for (intent in intents) {
        // Compose 的 LocalContext 通常是 Activity；非 Activity 上下文不补 NEW_TASK 会抛 AndroidRuntimeException。
        if (context !is Activity) intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            context.startActivity(intent)
            return
        } catch (e: Exception) {
            // 该 scheme 没有接收者（没装 QQ / 版本过旧）或系统拒绝，继续试下一条。
        }
    }
    Toast.makeText(context, R.string.settings_contact_failed, Toast.LENGTH_SHORT).show()
}

/**
 * 拼 QQ 群卡片 scheme（纯函数，便于单测）。
 *
 * 官方一键加群组件要求从 `qun.qq.com/join.html` 取的 `key`，本仓库只有公开分享短链、没有该 key；
 * 走群卡片 scheme 只需公开群号，用户点「加入群聊」时由 QQ 自己做申请，应用侧不接触任何凭据。
 */
internal fun qqGroupCardUri(groupNumber: String): String =
    "mqqapi://card/show_pslcard?src_type=internal&version=1&uin=$groupNumber&card_type=group&source=qrcode"

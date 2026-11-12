package com.github.echoping07.kemaolife.ui

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import com.github.echoping07.kemaolife.R

/** 交流群信息只放公开的群号、加群 key 与加群短链，与本仓库 README「交流」一节一致。 */
internal const val QQ_GROUP_NUMBER = "1126474633"

/**
 * 官方「一键加群」key，由群管理在 `qun.qq.com/join.html` 按群生成。
 * 它与具体群绑定、等于公开邀请凭证，任何安装本 App 的人都能用它申请加群 ——
 * 这正是它的设计用途，不属于 P1 的个人隐私（不是学号 / 密码 / Cookie / 房间号）。
 * 换群时必须改这里并重新发版。
 */
internal const val QQ_GROUP_JOIN_KEY = "MstzqKDJ4qvHs__hGIRFVZohU6MX923z"

internal const val QQ_GROUP_SHARE_URL = "https://qm.qq.com/q/rf4PfFM5RA"

/**
 * 拉起手机 QQ 的加群申请页。
 *
 * 两条路径，前者是官方「一键加群」组件给的跳转，后者是 README「交流」一节的公开短链：
 * ① `mqqopensdkapi://bizAgent/qm/qr` 把 `qm.qq.com/cgi-bin/qm/qr?...&k=<key>` 包给 QQ，
 *    QQ 自己取群资料并直接弹出「申请加入」页，用户一步到位；
 * ② QQ 未安装或版本不认该 scheme 时，退回 `qm.qq.com` 短链，由浏览器或 QQ 应用内打开同一个群。
 *
 * 两者都是隐式 Intent，只能判断「有没有接收者」；QQ 收到但内部失败时这里看不到，
 * 所以失败提示只在两条都没有接收者时给出。
 */
internal fun joinQQGroup(context: Context) {
    val intents = listOf(
        Intent(Intent.ACTION_VIEW, Uri.parse(qqGroupJoinUri(QQ_GROUP_JOIN_KEY))),
        Intent(Intent.ACTION_VIEW, Uri.parse(QQ_GROUP_SHARE_URL)),
    )
    for (intent in intents) {
        // 官方 snippet 默认不加 NEW_TASK：加群页按返回可回到本 App。
        // 只有 Compose 之外的上下文（Application / Service）必须补，否则抛 AndroidRuntimeException。
        if (context !is Activity) intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            context.startActivity(intent)
            return
        } catch (e: Exception) {
            // 没有接收者（没装 QQ / 版本过旧）或系统拒绝，继续试下一条。
        }
    }
    Toast.makeText(context, R.string.settings_contact_failed, Toast.LENGTH_SHORT).show()
}

/**
 * 拼官方一键加群 scheme（纯函数，便于单测；只用 JDK 编码器，不依赖 Android 的 `Uri`）。
 *
 * 内层 URL 按官方生成的 Android 代码原样保留（`http` 明文 + 这些固定参数），
 * 只额外做一次 percent-encode —— 改成 https 或删参数都可能让 QQ 侧匹配失败。
 */
internal fun qqGroupJoinUri(key: String): String {
    val inner = "http://qm.qq.com/cgi-bin/qm/qr?from=app&p=android&jump_from=webapi&k=$key"
    // 内层只有 ASCII 且无空格，URLEncoder 的 `+` 代表空格行为不会触发。
    return "mqqopensdkapi://bizAgent/qm/qr?url=" + java.net.URLEncoder.encode(inner, "UTF-8")
}

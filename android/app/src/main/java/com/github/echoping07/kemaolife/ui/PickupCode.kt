package com.github.echoping07.kemaolife.ui

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.widget.Toast
import com.github.echoping07.kemaolife.R

/**
 * 取件码深链（api-docs/004）：纯呈现层跳转，不发请求、不读凭据、不新增门面 op。
 * 链接整理自 pickup-code-launcher 的公开实现，全部是公开 scheme / 官方 H5 地址，
 * 不拼任何本机数据（学号 / 快照 / 日志）。
 */
internal enum class PickupDestination(
    val packageName: String,
    val appUris: List<String>,
    val webUri: String,
) {
    TAOBAO(
        packageName = "com.taobao.taobao",
        appUris = listOf(
            "taobao://m.taobao.com/tbopen/index.html?h5Url=https%3A%2F%2Fpages-fast.m.taobao.com%2Fwow%2Fz%2Funiapp%2F1011717%2Flast-mile-fe%2Fend-collect-platform%2Fidentity-code%3Fx-ssr%3Dtrue",
            "tbopen://m.taobao.com/tbopen/index.html?h5Url=https%3A%2F%2Fpages-fast.m.taobao.com%2Fwow%2Fz%2Funiapp%2F1011717%2Flast-mile-fe%2Fend-collect-platform%2Fidentity-code%3Fx-ssr%3Dtrue",
        ),
        webUri = "https://pages-fast.m.taobao.com/wow/z/uniapp/1011717/last-mile-fe/end-collect-platform/identity-code?x-ssr=true",
    ),
    PINDUODUO(
        packageName = "com.xunmeng.pinduoduo",
        // 候选从参数最全到裸路由逐级降级，兼容不同版本拼多多（api-docs/004 §2③）。
        appUris = listOf(
            "pinduoduo://com.xunmeng.pinduoduo/mdkd/package?tab=ID_CODE&entry_source=11&refer_page_name=login&refer_page_sn=10169",
            "pinduoduo://com.xunmeng.pinduoduo/mdkd/package?tab=ID_CODE&entry_source=11",
            "pinduoduo://com.xunmeng.pinduoduo/mdkd/package",
        ),
        webUri = "https://m.pinduoduo.net/mdkd/package?entry_source=18&extra_params=from_wx_jump%3D1&p_channel=0",
    ),
}

/** 一次跳转尝试：uri + 可选钉住的目标包名（null 表示交系统选择接收者）。 */
internal data class PickupAttempt(val uri: String, val packageName: String?)

/**
 * 按 api-docs/004 §3 的顺序生成回退链（纯函数，便于单测）：
 * 私有 scheme 候选钉包名 → 官方 H5 钉包名（部分版本 App 自己注册了 App Links）→ H5 交浏览器。
 */
internal fun pickupAttempts(destination: PickupDestination): List<PickupAttempt> =
    destination.appUris.map { PickupAttempt(it, destination.packageName) } +
        PickupAttempt(destination.webUri, destination.packageName) +
        PickupAttempt(destination.webUri, null)

/**
 * 与「交流」入口同一模式：直接 startActivity + 捕获异常判断有没有接收者，
 * 不用 resolveActivity 预判（少一次 Android 11+ 包可见性查询）。
 * 目标 App 收到后内部失败这里看不到，失败提示只在整条链都没有接收者时给出。
 */
internal fun openPickupCode(context: Context, destination: PickupDestination, title: String) {
    for (attempt in pickupAttempts(destination)) {
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(attempt.uri))
        if (attempt.packageName != null) intent.setPackage(attempt.packageName)
        intent.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        if (context !is Activity) intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            context.startActivity(intent)
            return
        } catch (e: Exception) {
            // 无接收者（未安装 / 版本不认该 scheme）或系统拒绝，继续试下一条。
        }
    }
    Toast.makeText(context, context.getString(R.string.pickup_open_failed, title), Toast.LENGTH_SHORT).show()
}
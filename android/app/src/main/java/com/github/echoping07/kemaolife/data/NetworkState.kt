package com.github.echoping07.kemaolife.data

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities

/**
 * 当前连接是不是（校园）WiFi。
 *
 * 校园网认证只在校园网内才有意义：蜂窝数据下探测必然是 `offline`，
 * 白跑一次请求还会让用户以为「认证坏了」。设置页的「仅 WiFi 下探测」
 * 就是让这一步提前短路（默认打开）。
 *
 * 只读系统能力，不涉及权限申请：`ACCESS_NETWORK_STATE` 在 Manifest 里。
 */
class NetworkState(private val context: Context) {

    fun isOnWifi(): Boolean {
        // 拿不到系统服务 / 拿不到网络能力一律 fail-closed：开关的语义是「非 WiFi 不发探测请求」，
        // 不确定的网络绝不能默认放行（否则蜂窝 / VPN / 以太网也会真发 Portal 探测）。
        val manager = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
            ?: return false
        val capabilities = manager.getNetworkCapabilities(manager.activeNetwork) ?: return false
        // VPN 一律排除：探测必须直连校园网，走代理会误判 offline。
        if (capabilities.hasTransport(NetworkCapabilities.TRANSPORT_VPN)) return false
        // 刻意不要求 NET_CAPABILITY_VALIDATED：校园网未认证时正是 captive portal，
        // 系统不会标「已验证」，此时恰恰需要探测判出 unauth 让用户去登录。
        return capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)
    }
}

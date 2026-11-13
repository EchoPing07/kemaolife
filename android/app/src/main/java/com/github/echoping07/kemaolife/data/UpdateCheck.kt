package com.github.echoping07.kemaolife.data

import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

/**
 * 关于页「检查更新」的版本号来源与比较（docs/002 §5.3 tab3）。
 *
 * 为什么不走 Go 门面：这是第 4 类外部目标，与三个功能域无关，也不读本机快照 /
 * 凭据；新增门面 `op` 会把一个纯呈现层的公开信息查询塞进 core 的契约面。
 * 这里只用 JDK 标准库，不新增第三方依赖（AGENTS A5）。
 *
 * 为什么用 VERSION_NAME 而不是 VERSION_CODE：本地构建未注入 `-PversionCode`
 * 时默认是 1，用它比较会让所有调试构建永远显示「有新版本」。
 */
object UpdateCheck {
    /** 官方端点返回最新的非 draft、非 prerelease release；只需要它的 `tag_name`。 */
    private const val LATEST_RELEASE_URL =
        "https://api.github.com/repos/EchoPing07/kemaolife/releases/latest"
    private const val CONNECT_TIMEOUT_MS = 8_000
    private const val READ_TIMEOUT_MS = 8_000

    // 发布 tag 由 release.yml 限定为 vX.Y.Z；`v` 前缀可选，三段都必须是十进制数。
    // 解析后按数值重排展示，不把上游字符串原样带进 UI（P2）。
    private val VERSION_PATTERN = Regex("""^v?(\d+)\.(\d+)\.(\d+)$""")

    data class Version(val major: Int, val minor: Int, val patch: Int) : Comparable<Version> {
        // 三段数值比较：字符串字典序会把 0.10.0 判成比 0.9.0 旧。
        override fun compareTo(other: Version): Int = when {
            major != other.major -> major.compareTo(other.major)
            minor != other.minor -> minor.compareTo(other.minor)
            else -> patch.compareTo(other.patch)
        }

        override fun toString(): String = "$major.$minor.$patch"
    }

    /** 非法格式（含溢出、附加后缀）一律返回 null，由调用方归入检查失败。 */
    fun parseVersion(raw: String): Version? {
        val match = VERSION_PATTERN.matchEntire(raw.trim()) ?: return null
        val major = match.groupValues[1].toIntOrNull() ?: return null
        val minor = match.groupValues[2].toIntOrNull() ?: return null
        val patch = match.groupValues[3].toIntOrNull() ?: return null
        return Version(major, minor, patch)
    }

    /**
     * 取远端 `tag_name`。上游错误正文 / 响应头都不回传给 UI：只抛不含明细的异常，
     * 调用方一律用本地固定文案展示（P2）。请求不带凭据、Cookie、学号或设备标识。
     */
    fun fetchLatestTag(): String {
        val connection = (URL(LATEST_RELEASE_URL).openConnection() as HttpURLConnection).apply {
            connectTimeout = CONNECT_TIMEOUT_MS
            readTimeout = READ_TIMEOUT_MS
            requestMethod = "GET"
            // HttpURLConnection 默认跟随重定向，此处不改写全局设置。
            setRequestProperty("Accept", "application/vnd.github+json")
            setRequestProperty("User-Agent", "kemaolife-android")
        }
        try {
            if (connection.responseCode != HttpURLConnection.HTTP_OK) throw IllegalStateException()
            val body = connection.inputStream.bufferedReader(Charsets.UTF_8).use { it.readText() }
            return JSONObject(body).getString("tag_name")
        } finally {
            connection.disconnect()
        }
    }
}

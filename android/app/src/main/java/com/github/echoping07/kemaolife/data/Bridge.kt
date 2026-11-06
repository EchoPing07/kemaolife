package com.github.echoping07.kemaolife.data

import android.content.Context
import com.github.echoping07.kemaolife.bridge.kemaolife.Kemaolife
import org.json.JSONObject

/**
 * 门面调用的唯一入口（docs/002 §2.1）：Kotlin 只构造请求 JSON、解析响应信封。
 *
 * 为什么集中在 DataSource：`op` 是契约（改动要先改 docs/002），
 * 散落在 UI 里迟早会拼错字符串或漏判 `ok`。
 *
 * 线程纪律：所有 `Call` 都会做网络 I/O，**必须在 IO 线程调用**（见 Repository）。
 * 门面内部是同步阻塞的 C 调用，放在主线程会 ANR。
 */
class Bridge(private val context: Context) {

    /** 门面是否初始化成功过；false 时所有调用直接返回失败信封，不再触碰 JNI。 */
    @Volatile
    private var initialized = false

    @Volatile
    private var initError: String? = null

    /**
     * 把「可写目录 + 解密后的凭据」交给 Go（001 §6.2）。
     *
     * Kotlin 负责 Keystore 加解密与持久化，Go 只处理内存里的凭据；
     * 设置页保存后重新调用本方法即可热更新，门面不提供 `config.*` op。
     */
    fun init(credentials: Credentials): Envelope {
        // 时区必须由 Kotlin 传入：Go 在 Android 上找不到系统的 zoneinfo 目录，
        // time.Local 会退化成 UTC，导致所有时间戳差 8 小时（见 android/README.md）。
        val zone = java.util.TimeZone.getDefault()
        val config = JSONObject()
            .put("cacheDir", cacheDir().absolutePath)
            .put("studentId", credentials.studentId)
            .put("eduPassword", credentials.eduPassword)
            .put("customerCode", credentials.customerCode)
            .put("campusUser", credentials.campusUser)
            .put("campusPassword", credentials.campusPassword)
            .put("campusFingerprint", credentials.campusFingerprint)
            .put("timeZone", zone.id)
            .put("timeZoneOffsetSeconds", zone.getOffset(System.currentTimeMillis()) / 1000)

        val env = Envelope.parse(Kemaolife.init(config.toString()))
        initialized = env.ok
        initError = env.error?.message
        return env
    }

    fun isReady(): Boolean = initialized

    fun readyError(): String? = initError

    /** 发一次 `op`；任何异常都被收敛成失败信封，绝不抛到 UI。 */
    fun call(op: String, vararg params: Pair<String, Any>): Envelope {
        if (!initialized) {
            return Envelope(
                ok = false,
                data = null,
                error = ApiError(2001, initError ?: "桥接尚未初始化"),
                ts = "",
                stale = false,
                syncedAt = "",
            )
        }
        return try {
            val req = JSONObject().put("op", op)
            for ((k, v) in params) req.put(k, v)
            Envelope.parse(Kemaolife.call(req.toString()))
        } catch (e: Throwable) {
            // 这里不读取 e.message 之外的任何内容，避免把内部路径 / 堆栈带进 UI 或日志。
            Envelope(
                ok = false,
                data = null,
                error = ApiError(4001, "调用失败，请稍后重试"),
                ts = "",
                stale = false,
                syncedAt = "",
            )
        }
    }

    /**
     * 快照存放目录：应用私有目录，不申请外部存储权限。
     *
     * 快照含 Cookie，等同凭据（P6）——所以只能落在私有目录里，
     * 且随后由 Keystore 加密的凭据分离存放，两者都不进日志。
     */
    private fun cacheDir() = java.io.File(context.filesDir, "snapshot").apply { mkdirs() }
}

package com.github.echoping07.kemaolife.data

import org.json.JSONObject

/**
 * 门面响应信封（docs/002 §2）。
 *
 * 与 CLI `--json` 同构：`ok` / `data` / `error{code,message}` / `ts` / `stale` / `syncedAt`。
 * Kotlin 侧只做反序列化与展示，不重新计算任何业务数据（AGENTS A3）。
 */
data class Envelope(
    val ok: Boolean,
    val data: JSONObject?,
    val error: ApiError?,
    val ts: String,
    val stale: Boolean,
    val syncedAt: String,
) {
    companion object {
        /**
         * 解析门面返回的 JSON。
         *
         * 解析失败时返回一个 4002 的失败信封而不是抛异常：
         * 门面是 JNI 边界，任何异常都会跨语言边界传播，很难处理。
         */
        fun parse(raw: String): Envelope {
            return try {
                val obj = JSONObject(raw)
                val errObj = obj.optJSONObject("error")
                Envelope(
                    ok = obj.optBoolean("ok", false),
                    data = obj.optJSONObject("data"),
                    error = errObj?.let {
                        ApiError(it.optInt("code", 4002), it.optString("message"))
                    },
                    ts = obj.optString("ts"),
                    stale = obj.optBoolean("stale", false),
                    syncedAt = obj.optString("syncedAt"),
                )
            } catch (e: Exception) {
                Envelope(
                    ok = false,
                    data = null,
                    error = ApiError(4002, "响应格式异常"),
                    ts = "",
                    stale = false,
                    syncedAt = "",
                )
            }
        }
    }
}

/** 错误对象；`code` 取 docs/002 §4 的 1001–5001 域。 */
data class ApiError(val code: Int, val message: String)

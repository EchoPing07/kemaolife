package com.github.echoping07.kemaolife.data

import org.json.JSONArray
import org.json.JSONObject

/**
 * 冷启动首帧的电费结果缓存（docs/android.md「每次打开」）。
 *
 * 与 [TimetableCache] 同口径：Kotlin 只按字段白名单持久化 UI 必需展示字段，
 * 不含 `roomName` / `roomVerify` / 学号 / 凭据 / Cookie（docs/android.md §7、AGENTS P1/P2）。
 * 镜像落在应用私有 `filesDir/electricity_cache.json`，`allowBackup=false` 且
 * `data_extraction_rules.xml` 已排除全部私有数据（`domain="root"`）。
 * JSON 结构与历史版本完全兼容，升级后可直接读取。
 */
object ElectricityCache {

    fun encode(report: ElectricityReport): String {
        val status = JSONObject()
            .put("balance", report.status.balance)
            .put("unit", report.status.unit)
            .put("collectedAt", report.status.collectedAt)
            .put("dailyAvg", report.status.dailyAvg ?: JSONObject.NULL)
            .put("estimatedDays", report.status.estimatedDays ?: JSONObject.NULL)
        val days = JSONArray().also { arr ->
            report.usage.days.forEach { day ->
                arr.put(JSONObject().put("date", day.date).put("use", day.use))
            }
        }
        val usage = JSONObject()
            .put("yearMonth", report.usage.yearMonth)
            .put("monthUse", report.usage.monthUse)
            .put("dataUntil", report.usage.dataUntil)
            .put("days", days)
        return JSONObject()
            .put("status", status)
            .put("usage", usage)
            .toString()
    }

    /** 空串、损坏内容或缺 `status`/`usage` 一律视为「没有缓存」，不阻断启动。 */
    fun decode(raw: String): ElectricityReport? {
        if (raw.isBlank()) return null
        return try {
            val obj = JSONObject(raw)
            val statusObj = obj.optJSONObject("status") ?: return null
            val usageObj = obj.optJSONObject("usage") ?: return null
            ElectricityReport(
                // 强制剥离 roomName，避免外部/历史脏数据把宿舍标识带入 UI（P2）。
                status = ElectricityStatus.from(statusObj).copy(roomName = ""),
                usage = ElectricityUsage.from(usageObj),
            )
        } catch (e: Exception) {
            null
        }
    }
}

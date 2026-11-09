package com.github.echoping07.kemaolife.data

import org.json.JSONArray
import org.json.JSONObject

/**
 * 冷启动首帧的课表镜像（docs/002 §5.3「每次打开」）。
 *
 * 课表正本快照含 Cookie，只能由 Go 会话在门面 `Init` 之后读取；若把「打开即出」
 * 的缓存也押在 `Init` 上，课表位会先空一帧，甚至把未初始化的失败信封当成空态
 * 文案显示出来。这里与电费缓存同口径：Kotlin 只持久化 DTO 的展示字段，
 * ViewModel 构造时同步读出，**不等待 JNI**；`Init` 完成后仍由 `CacheOnly` 读回
 * 校正跨周 / 跨学期，并回写本镜像。
 *
 * 字段白名单是刻意的：不写 Cookie / 凭据 / 学号 / 上游快照结构，也不做周次计算
 * （AGENTS A3）。镜像落在应用私有 `app_prefs`，`allowBackup=false` 且
 * `data_extraction_rules.xml` 已排除全部 sharedpref。
 */
data class CachedWeek(val schedule: WeekSchedule, val syncedAt: String)

object TimetableCache {

    fun encode(schedule: WeekSchedule, syncedAt: String): String = JSONObject()
        .put("week", schedule.week)
        .put("syncedAt", syncedAt)
        .put("days", JSONArray().also { days ->
            schedule.days.forEach { day -> days.put(day.toJson()) }
        })
        .toString()

    /** 空串、损坏内容或缺 `days` 一律视为「没有缓存」，由调用方走中性空态。 */
    fun decode(raw: String): CachedWeek? {
        if (raw.isBlank()) return null
        return try {
            val obj = JSONObject(raw)
            val days = obj.optJSONArray("days") ?: return null
            if (days.length() == 0) return null
            val parsed = ArrayList<DaySchedule>(days.length())
            for (i in 0 until days.length()) {
                parsed.add(DaySchedule.from(days.optJSONObject(i) ?: return null))
            }
            CachedWeek(WeekSchedule(obj.optInt("week"), parsed), obj.optString("syncedAt"))
        } catch (e: Exception) {
            null
        }
    }

    /** 只投影 DTO 字段；缺字段由 `DaySchedule.from` 的零值语义兜住。 */
    private fun DaySchedule.toJson(): JSONObject = JSONObject()
        .put("date", date)
        .put("week", week)
        .put("weekday", weekday)
        .put("weekdayCn", weekdayCn)
        .put("inSemester", inSemester)
        .put("entries", JSONArray().also { entries ->
            this.entries.forEach { entry -> entries.put(entry.toJson()) }
        })

    private fun Entry.toJson(): JSONObject = JSONObject()
        .put("name", name)
        .put("periods", periods)
        .put("time", time)
        .put("teacher", teacher)
        .put("location", location)
        .put("weeksText", weeksText)
}

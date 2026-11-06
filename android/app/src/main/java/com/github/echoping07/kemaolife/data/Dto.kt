package com.github.echoping07.kemaolife.data

import org.json.JSONArray
import org.json.JSONObject

/**
 * 三端共享 DTO 的 Kotlin 只读镜像（docs/002 §3）。
 *
 * 这里**只做字段投影**，不做任何计算：周次、均量、可用天数全部来自 Go 侧 `core`，
 * 保证三端不会各算一遍算出不同结果（AGENTS A3 / A6）。
 */
data class DaySchedule(
    val date: String,
    val week: Int,
    val weekday: Int,
    val weekdayCn: String,
    val inSemester: Boolean,
    val entries: List<Entry>,
) {
    companion object {
        fun from(obj: JSONObject): DaySchedule = DaySchedule(
            date = obj.optString("date"),
            week = obj.optInt("week"),
            weekday = obj.optInt("weekday"),
            weekdayCn = obj.optString("weekdayCn"),
            inSemester = obj.optBoolean("inSemester"),
            entries = obj.optJSONArray("entries").mapObjects { Entry.from(it) },
        )
    }
}

data class Entry(
    val name: String,
    val periods: String,
    val time: String,
    val teacher: String,
    val location: String,
    val weeksText: String,
) {
    companion object {
        fun from(obj: JSONObject): Entry = Entry(
            name = obj.optString("name"),
            periods = obj.optString("periods"),
            time = obj.optString("time"),
            teacher = obj.optString("teacher"),
            location = obj.optString("location"),
            weeksText = obj.optString("weeksText"),
        )
    }
}

/** `timetable.week` 的 data：周次 + 7 天。 */
data class WeekSchedule(val week: Int, val days: List<DaySchedule>) {
    companion object {
        fun from(obj: JSONObject): WeekSchedule = WeekSchedule(
            week = obj.optInt("week"),
            days = obj.optJSONArray("days").mapObjects { DaySchedule.from(it) },
        )
    }
}

/** `timetable.sync` 的 data：同步结果摘要（不含 Cookie）。 */
data class SyncResult(
    val syncedAt: String,
    val week1Monday: String,
    val lastWeek: Int,
    val lessonCount: Int,
) {
    companion object {
        fun from(obj: JSONObject): SyncResult = SyncResult(
            syncedAt = obj.optString("syncedAt"),
            week1Monday = obj.optString("week1Monday"),
            lastWeek = obj.optInt("lastWeek"),
            lessonCount = obj.optInt("lessonCount"),
        )
    }
}

data class ElectricityReport(val status: ElectricityStatus, val usage: ElectricityUsage) {
    companion object {
        fun from(obj: JSONObject): ElectricityReport = ElectricityReport(
            status = ElectricityStatus.from(obj.optJSONObject("status") ?: JSONObject()),
            usage = ElectricityUsage.from(obj.optJSONObject("usage") ?: JSONObject()),
        )
    }
}

data class ElectricityStatus(
    val roomName: String,
    val balance: Double,
    val unit: String,
    val collectedAt: String,
    val dailyAvg: Double?,
    val estimatedDays: Int?,
) {
    companion object {
        fun from(obj: JSONObject): ElectricityStatus = ElectricityStatus(
            roomName = obj.optString("roomName"),
            balance = obj.optDouble("balance", 0.0),
            unit = obj.optString("unit"),
            collectedAt = obj.optString("collectedAt"),
            // null 有明确语义：「无数据」/「无法估算」（docs/002 §3.2），不能用 0 代替。
            dailyAvg = if (obj.isNull("dailyAvg")) null else obj.optDouble("dailyAvg"),
            estimatedDays = if (obj.isNull("estimatedDays")) null else obj.optInt("estimatedDays"),
        )
    }
}

data class ElectricityUsage(
    val yearMonth: String,
    val monthUse: Double,
    val dataUntil: String,
    val days: List<DayUsage>,
) {
    companion object {
        fun from(obj: JSONObject): ElectricityUsage = ElectricityUsage(
            yearMonth = obj.optString("yearMonth"),
            monthUse = obj.optDouble("monthUse", 0.0),
            dataUntil = obj.optString("dataUntil"),
            days = obj.optJSONArray("days").mapObjects {
                DayUsage(it.optString("date"), it.optDouble("use", 0.0))
            },
        )
    }
}

data class DayUsage(val date: String, val use: Double)

/** 校园网三态。`offline` 是正常状态，不是错误（docs/002 §3.3）。 */
enum class CampusState(val wire: String) {
    ONLINE("online"),
    UNAUTH("unauth"),
    OFFLINE("offline");

    companion object {
        fun from(wire: String): CampusState =
            entries.firstOrNull { it.wire == wire } ?: OFFLINE
    }
}

data class CampusStatus(
    val state: CampusState,
    val errorMsg: String,
    val checkedAt: String,
) {
    companion object {
        fun from(obj: JSONObject): CampusStatus = CampusStatus(
            state = CampusState.from(obj.optString("state")),
            errorMsg = obj.optString("errorMsg"),
            checkedAt = obj.optString("checkedAt"),
        )
    }
}

/** 把 JSON 数组里的对象逐个投影成 T；缺字段时跳过而不是抛异常。 */
private inline fun <T> JSONArray?.mapObjects(block: (JSONObject) -> T): List<T> {
    if (this == null) return emptyList()
    val out = ArrayList<T>(length())
    for (i in 0 until length()) {
        optJSONObject(i)?.let { out.add(block(it)) }
    }
    return out
}

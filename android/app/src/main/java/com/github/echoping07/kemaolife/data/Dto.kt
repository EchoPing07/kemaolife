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
    val brasName: String = "",
    val brasAddress: String = "",
    val vlan: String = "",
    val deviceIp: String = "",
    val mac: String = "",
) {
    companion object {
        fun from(obj: JSONObject): CampusStatus = CampusStatus(
            state = CampusState.from(obj.optString("state")),
            errorMsg = obj.optString("errorMsg"),
            checkedAt = obj.optString("checkedAt"),
            brasName = obj.optString("brasName"),
            brasAddress = obj.optString("brasAddress"),
            vlan = obj.optString("vlan"),
            deviceIp = obj.optString("deviceIp"),
            mac = obj.optString("mac"),
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

/**
 * 设置页「测试」按钮的统一回报（`*.probe` 三个 op 同构）。
 *
 * `ok` 是测试动作本身的成败，外面那层信封恒为成功 —— 失败时也要能看到
 * 耗时与脱敏后的原因，否则用户只知道「点了一下没反应」。
 */
data class ProbeResult(
    val ok: Boolean,
    val elapsedMs: Long,
    val code: Int,
    val message: String,
    val detail: Map<String, String>,
    /** `electricity.probe` 同一请求带回的完整报告，供 UI 直接刷新卡片（不再二次联网）。 */
    val report: ElectricityReport? = null,
    /** `campusnet.probe` 同一请求带回的完整判态，供 UI 直接刷新卡片。 */
    val campusStatus: CampusStatus? = null,
) {
    companion object {
        /** 可展示字段白名单：上游新增字段一律丢弃，不把未知内容原样摊给用户（P2）。 */
        private val DETAIL_KEYS = listOf(
            "state", "checkedAt",
            "balance", "unit", "collectedAt", "estimatedDays", "monthUse", "dataUntil", "dayCount",
            "syncedAt", "week1Monday", "lastWeek", "lessonCount",
        )

        fun from(obj: JSONObject): ProbeResult {
            val detailObj = obj.optJSONObject("detail")
            val detail = LinkedHashMap<String, String>()
            DETAIL_KEYS.forEach { key ->
                if (detailObj?.has(key) != true) return@forEach
                // null / 空值保留为空串，交给 UI 显示占位符「—」；
                // 既不跳过整个字段，也不把字面量 "null" 印到弹窗上。
                detail[key] = if (detailObj.isNull(key)) "" else detailObj.opt(key)?.toString().orEmpty()
            }
            return ProbeResult(
                ok = obj.optBoolean("ok", false),
                elapsedMs = obj.optLong("elapsedMs", 0L),
                code = obj.optInt("code", 0),
                message = obj.optString("message"),
                detail = detail,
                report = obj.optJSONObject("report")?.let { ElectricityReport.from(it) },
                campusStatus = obj.optJSONObject("campusStatus")?.let { CampusStatus.from(it) },
            )
        }
    }
}

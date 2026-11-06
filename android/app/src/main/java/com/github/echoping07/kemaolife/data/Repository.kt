package com.github.echoping07.kemaolife.data

import java.time.LocalDate

/**
 * 用例调用面：UI 只通过这里访问数据，不理解 `op` 字符串，也不做业务计算。
 *
 * 所有方法都是阻塞调用，**必须在 IO 调度器上执行**（ViewModel 负责切换）。
 * 返回的 [Result] 要么带 DTO，要么带可展示的错误文案。
 */
class Repository(private val bridge: Bridge) {

    fun isReady() = bridge.isReady()
    fun readyError() = bridge.readyError()

    fun init(credentials: Credentials): Envelope = bridge.init(credentials)

    // ---- 课表 -------------------------------------------------------------

    /** `timetable.sync`：只刷新不查询（下拉刷新用）。 */
    fun sync(): Result<SyncResult> =
        bridge.call("timetable.sync").toResult { SyncResult.from(it) }

    /**
     * `timetable.today`：内部先刷新再计算。
     *
     * 刷新失败但有旧快照时 Go 侧返回 `ok:true` + `stale:true`（docs/002 §1.3），
     * 所以这里不需要额外的「离线兜底」逻辑，只需要把 stale 透传给 UI。
     */
    fun today(): Result<ScheduleView> =
        bridge.call("timetable.today").toSchedule()

    fun date(date: LocalDate): Result<ScheduleView> =
        bridge.call("timetable.date", "date" to date.toString()).toSchedule()

    /** `timetable.week`：`week <= 0` 表示当周。 */
    fun week(week: Int): Result<WeekView> {
        val env = bridge.call("timetable.week", "week" to week)
        val data = env.data
        if (!env.ok || data == null) {
            return Result.failure(AppException(env.error))
        }
        return Result.success(
            WeekView(
                schedule = WeekSchedule.from(data),
                stale = env.stale,
                syncedAt = env.syncedAt,
            ),
        )
    }

    // ---- 电费 -------------------------------------------------------------

    /** `electricity.report`：一次拿余额 + 每日用量，任一失败整体报错。 */
    fun electricity(): Result<ElectricityReport> =
        bridge.call("electricity.report").toResult { ElectricityReport.from(it) }

    // ---- 校园网 -----------------------------------------------------------

    fun campusStatus(): Result<CampusStatus> =
        bridge.call("campusnet.status").toResult { CampusStatus.from(it) }

    fun campusLogin(): Result<CampusStatus> =
        bridge.call("campusnet.login").toResult { CampusStatus.from(it) }

    fun campusLogout(): Result<CampusStatus> =
        bridge.call("campusnet.logout").toResult { CampusStatus.from(it) }

    fun campusRelogin(): Result<CampusStatus> =
        bridge.call("campusnet.relogin").toResult { CampusStatus.from(it) }

    // ---- 信封 → Result ----------------------------------------------------

    private fun Envelope.toSchedule(): Result<ScheduleView> {
        val data = data
        if (!ok || data == null) return Result.failure(AppException(error))
        return Result.success(
            ScheduleView(
                day = DaySchedule.from(data),
                stale = stale,
                syncedAt = syncedAt,
            ),
        )
    }

    private fun <T> Envelope.toResult(map: (org.json.JSONObject) -> T): Result<T> {
        val data = data
        if (!ok || data == null) return Result.failure(AppException(error))
        return try {
            Result.success(map(data))
        } catch (e: Exception) {
            Result.failure(AppException(ApiError(4002, "响应格式异常")))
        }
    }
}

/**
 * 课表响应：DTO + 数据时间。
 *
 * `stale` / `syncedAt` 是查询语义的一部分（docs/002 §2），必须和 DTO 一起传给 UI，
 * 不能由展示层猜。
 */
data class ScheduleView(val day: DaySchedule, val stale: Boolean, val syncedAt: String)

data class WeekView(val schedule: WeekSchedule, val stale: Boolean, val syncedAt: String)

/**
 * 用例失败。message 已由 Go 侧脱敏（P2），UI 直接展示即可。
 *
 * 携带 code 是为了区分「该引导去设置页」还是「该提示网络」，
 * 不是为了让 UI 重新判断业务逻辑。
 */
class AppException(val error: ApiError?) : Exception(error?.message ?: "未知错误") {
    val code: Int get() = error?.code ?: 4001
    val displayMessage: String get() = error?.message ?: "未知错误"

    /** 3001 = 凭据缺失，需要引导用户去设置页（docs/002 §4）。 */
    val needsCredentials: Boolean get() = code == 3001
}

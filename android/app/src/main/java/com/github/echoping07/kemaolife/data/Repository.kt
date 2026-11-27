package com.github.echoping07.kemaolife.data

import org.json.JSONObject

/**
 * 用例调用面：UI 只通过这里访问数据，不理解 `op` 字符串，也不做业务计算。
 *
 * 所有方法都是阻塞调用，**必须在 IO 调度器上执行**（ViewModel 负责切换）。
 * 返回的 [Result] 要么带 DTO，要么带可展示的错误文案。
 *
 * 调用方必须先区分「读缓存」与「联网」两条链路（docs/android.md）：
 *  - [week] 纯读本地快照，0 请求，用于打开 App 与切周；
 *  - [sync] 联网刷新一次，成功后 UI 再调 [week] 回来。
 * 课表查询**不再**隐式带刷新，切周不会把整份课表重拉一遍。
 */
class Repository(private val bridge: Bridge) {

    fun isReady() = bridge.isReady()
    fun readyError() = bridge.readyError()

    fun init(credentials: Credentials, resetSnapshot: Boolean = false, requestLog: Boolean = false): Envelope =
        bridge.init(credentials, resetSnapshot, requestLog)

    // ---- 课表 -------------------------------------------------------------

    /** `timetable.sync`：联网刷新一次（打开 App 的后台动作、手动刷新用）。 */
    fun sync(): Result<SyncResult> =
        bridge.call("timetable.sync").toResult { SyncResult.from(it) }

    /** `timetable.week`：纯读缓存，`week == 0` 表示当周，负数也是显式周次。 */
    fun week(week: Int = 0): Result<WeekView> =
        bridge.call("timetable.week", "week" to week).toResult { env, data ->
            WeekView(
                schedule = WeekSchedule.from(data),
                syncedAt = env.syncedAt,
                // 可选字段：缺省 0 = 未知，翻页容器不得据此拦截翻页（docs/contract.md §7.1）。
                lastWeek = data.optInt("lastWeek"),
            )
        }

    /** `timetable.probe`：设置页「测试」，同步一次并回报耗时与快照摘要。 */
    fun testTimetable(): Result<ProbeResult> =
        bridge.call("timetable.probe").toResult { ProbeResult.from(it) }

    /**
     * `timetable.export`：导出整学期课表（WakeUp 课程表 CSV）。
     *
     * 只读本地快照（0 请求），与 [week] 同一条链路：导出的是本机已有的课表，
     * 不联网、不触发刷新；数据时间由返回的 `syncedAt` 给出。
     */
    fun exportTimetable(optimizeLocation: Boolean = false): Result<TimetableExport> =
        bridge.call("timetable.export", "optimizeLocation" to optimizeLocation)
            .toResult { TimetableExport.from(it) }

    // ---- 电费 -------------------------------------------------------------

    /** `electricity.report`：一次拿余额 + 每日用量，任一失败整体报错。 */
    fun electricity(): Result<ElectricityReport> =
        bridge.call("electricity.report").toResult { ElectricityReport.from(it) }

    /** `electricity.probe`：设置页「测试」，查询一次并回报耗时与摘要。 */
    fun testElectricity(): Result<ProbeResult> =
        bridge.call("electricity.probe").toResult { ProbeResult.from(it) }

    /**
     * `electricity.alert`：调用 Go `core/alert.Evaluate` 判定是否低于阈值（0 请求）。
     *
     * 只传必要展示字段，不传 `roomName` / `roomVerify`（P2）；告警判定与文案生成全在 Go 侧（AGENTS A3）。
     */
    fun evaluateElectricityAlert(
        status: ElectricityStatus,
        mode: String,
        balanceThreshold: Double,
        daysThreshold: Int,
        prevState: ElectricityAlertState,
    ): Result<ElectricityAlertDecision> {
        val statusJson = JSONObject()
            .put("balance", status.balance)
            .put("unit", status.unit)
            .put("collectedAt", status.collectedAt)
            .put("estimatedDays", status.estimatedDays ?: JSONObject.NULL)
        return bridge.call(
            "electricity.alert",
            "status" to statusJson,
            "mode" to mode,
            "balanceThreshold" to balanceThreshold,
            "daysThreshold" to daysThreshold,
            "state" to prevState.toJson(),
        ).toResult { ElectricityAlertDecision.from(it) }
    }

    // ---- 校园网 -----------------------------------------------------------

    fun campusStatus(): Result<CampusStatus> =
        bridge.call("campusnet.status").toResult { CampusStatus.from(it) }

    fun campusLogin(): Result<CampusStatus> =
        bridge.call("campusnet.login").toResult { CampusStatus.from(it) }

    fun campusLogout(): Result<CampusStatus> =
        bridge.call("campusnet.logout").toResult { CampusStatus.from(it) }

    fun campusRelogin(): Result<CampusStatus> =
        bridge.call("campusnet.relogin").toResult { CampusStatus.from(it) }

    /** `campusnet.probe`：设置页「测试」，只探测一次判态。 */
    fun testCampus(): Result<ProbeResult> =
        bridge.call("campusnet.probe").toResult { ProbeResult.from(it) }

    // ---- 请求记录（设置页「日志」） ---------------------------------------

    /** `log.enable`：开关请求记录；关闭会同时清空已记录内容（docs/contract.md §6 规则 10）。 */
    fun setRequestLog(enabled: Boolean): Result<LogState> =
        bridge.call("log.enable", "enabled" to enabled).toResult { LogState.from(it) }

    /** `log.list`：最近若干条摘要（新的在前），不含正文。 */
    fun requestLog(): Result<LogState> =
        bridge.call("log.list").toResult { LogState.from(it) }

    /** `log.detail`：单条完整记录（原始标头与正文）。 */
    fun requestLogDetail(seq: Long): Result<LogEntry> =
        bridge.call("log.detail", "seq" to seq).toResult { LogEntry.from(it) }

    /** `log.clear`：清空记录。 */
    fun clearRequestLog(): Result<LogState> =
        bridge.call("log.clear").toResult { LogState.from(it) }

    // ---- 信封 → Result ----------------------------------------------------

    private fun <T> Envelope.toResult(map: (Envelope, JSONObject) -> T): Result<T> {
        val data = data
        if (!ok || data == null) return Result.failure(AppException(error))
        return try {
            Result.success(map(this, data))
        } catch (e: Exception) {
            Result.failure(AppException(ApiError(4002, "响应格式异常")))
        }
    }

    private fun <T> Envelope.toResult(map: (JSONObject) -> T): Result<T> =
        toResult { _, data -> map(data) }
}

/**
 * 课表响应：DTO + 数据时间。
 *
 * `syncedAt` 是查询语义的一部分（docs/contract.md §5），必须和 DTO 一起传给 UI；
 * 离线标记只由 `timetable.sync` 的事务维护，纯读缓存不覆写它（docs/android.md）。
 */
data class WeekView(
    val schedule: WeekSchedule,
    val syncedAt: String,
    /** 学期最后一周（`timetable.week` 的可选 `lastWeek`）；0 = 未知。 */
    val lastWeek: Int = 0,
)

/**
 * 用例失败。message 已由 Go 侧脱敏（P2），UI 直接展示即可。
 *
 * 携带 code 是为了区分「该引导去设置页」还是「该提示网络」，
 * 不是为了让 UI 重新判断业务逻辑。
 */
class AppException(val error: ApiError?) : Exception(error?.message ?: "未知错误") {
    val code: Int get() = error?.code ?: 4001
    val displayMessage: String get() = error?.message ?: "未知错误"
}

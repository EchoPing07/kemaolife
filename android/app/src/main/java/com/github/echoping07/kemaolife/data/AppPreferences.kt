package com.github.echoping07.kemaolife.data

import android.content.Context
import android.content.SharedPreferences
import android.os.SystemClock

/**
 * 非敏感的行为偏好：查询间隔、探测开关、上次自动查询时刻、电费结果缓存。
 *
 * 为什么不放 Keystore：这些值不是凭据（P5 只管学号 / 密码 / Cookie），
 * 塞进加密块只会让「读一个数字」也要过一遍 KeyStore；也不放
 * [Credentials]（那是 Init 要传给 Go 的东西，间隔只影响 UI 的触发时机）。
 *
 * 隐私口径：电费缓存只写 UI 必需字段，不含 `roomName` / `roomVerify` / 学号 /
 * Cookie（P2）；`roomName` 是本人宿舍标识，既不需要展示也不需要落盘。
 * 课表镜像只写 `WeekSchedule` 的展示字段，同样不含 Cookie / 凭据。
 */
class AppPreferences(context: Context) {

    private val prefs = context.getSharedPreferences("app_prefs", Context.MODE_PRIVATE)

    /**
     * 时间戳 / 缓存需要跨进程存活：`.apply()` 异步落盘，进程被杀时可能丢。
     * 调用方都在 IO 调度器上，用同步 `commit()` 并放在这里统一处理。
     */
    private fun SharedPreferences.editCommit(block: SharedPreferences.Editor.() -> Unit) {
        val editor = edit()
        editor.block()
        editor.commit()
    }

    /** 课表自动同步的最小间隔（分钟）。手动刷新不受它约束。 */
    var timetableIntervalMinutes: Int
        get() = clamped(prefs.getInt(KEY_TIMETABLE_INTERVAL, DEFAULT_INTERVAL), MIN_INTERVAL)
        set(value) = prefs.editCommit {
            putInt(KEY_TIMETABLE_INTERVAL, clamped(value, MIN_INTERVAL))
        }

    /** 电费自动查询的最小间隔（分钟）。与课表同下限、同默认值（docs/002 §5.3）。 */
    var electricityIntervalMinutes: Int
        get() = clamped(prefs.getInt(KEY_ELECTRICITY_INTERVAL, DEFAULT_INTERVAL), MIN_INTERVAL)
        set(value) = prefs.editCommit {
            putInt(KEY_ELECTRICITY_INTERVAL, clamped(value, MIN_INTERVAL))
        }

    /** 校园网「仅在 WiFi 下探测」开关（默认打开）。 */
    var campusWifiOnly: Boolean
        get() = prefs.getBoolean(KEY_CAMPUS_WIFI_ONLY, true)
        set(value) = prefs.editCommit { putBoolean(KEY_CAMPUS_WIFI_ONLY, value) }

    /**
     * 请求记录开关（默认关闭，docs/002 §2.1 规则 10）。
     *
     * 为什么要落盘：Go 侧记录器只存进程内存，进程被系统杀死后内容就没了；
     * 开关状态必须跨进程存活，否则每次冷启动都要用户重新打开。
     * 记录内容本身**不**落盘（含 Cookie 等凭据，等同凭据，P6）。
     */
    var requestLogEnabled: Boolean
        get() = prefs.getBoolean(KEY_REQUEST_LOG, false)
        set(value) = prefs.editCommit { putBoolean(KEY_REQUEST_LOG, value) }

    /**
     * 上次课表同步**尝试**时刻（`SystemClock.elapsedRealtime()`，单调时钟）。
     * 0 表示从未同步。记「尝试」而不是「成功」：失败后不记账会切一次 tab 再打一次接口。
     */
    var lastTimetableSyncAt: Long
        get() = prefs.getLong(KEY_LAST_TIMETABLE_SYNC, 0L)
        set(value) = prefs.editCommit { putLong(KEY_LAST_TIMETABLE_SYNC, value) }

    /** 上次电费查询时刻（单调时钟，口径同 [lastTimetableSyncAt]）。 */
    var lastElectricityAt: Long
        get() = prefs.getLong(KEY_LAST_ELECTRICITY, 0L)
        set(value) = prefs.editCommit { putLong(KEY_LAST_ELECTRICITY, value) }

    /**
     * 上一次电费结果（UI 必需字段的 JSON）。
     *
     * 为什么要落盘：电费页冷启动时也要「先出缓存再联网」（docs/002 §5.3）。
     * 由 Kotlin 白名单重建，不含 `roomName` / `roomVerify` / 学号 / Cookie。
     */
    var electricityCacheJson: String
        get() = prefs.getString(KEY_ELECTRICITY_CACHE, "").orEmpty()
        set(value) = prefs.editCommit { putString(KEY_ELECTRICITY_CACHE, value) }

    /**
     * 上次课表周视图的展示镜像（[TimetableCache] 编码）。
     *
     * 冷启动首帧要在门面 Init 之前就渲染课表：正本快照含 Cookie 只能由 Go 读，
     * 而 Init 是 JNI 调用，排在 IO 协程里。这里只存 DTO 展示字段（docs/002 §5.3）。
     */
    var timetableCacheJson: String
        get() = prefs.getString(KEY_TIMETABLE_CACHE, "").orEmpty()
        set(value) = prefs.editCommit { putString(KEY_TIMETABLE_CACHE, value) }

    /** 换账号时与电费缓存一起清掉，避免旧账号课表先显示（docs/002 §2.1 规则 2）。 */
    fun clearTimetableCache() {
        prefs.editCommit { putString(KEY_TIMETABLE_CACHE, "") }
    }

    /** 清除电费缓存：换账号 / 凭据失效时用，旧账号数据不能先显示。 */
    fun clearElectricityCache() {
        prefs.editCommit {
            putString(KEY_ELECTRICITY_CACHE, "")
            putLong(KEY_LAST_ELECTRICITY, 0L)
        }
    }

    /** 单调时钟上的「现在」；用于所有自动查询节流。 */
    fun nowElapsed(): Long = SystemClock.elapsedRealtime()

    /** 间隔是否已过；`last == 0`（从没查过）视为已过。 */
    fun isDue(last: Long, minutes: Int, now: Long = nowElapsed()): Boolean =
        isDueAt(last, minutes, now)

    companion object {
        /** 课表与电费的最小间隔：默认一致，均为 5 分钟（docs/002 §5.3）。 */
        const val DEFAULT_INTERVAL = 5
        const val DEFAULT_TIMETABLE_INTERVAL = DEFAULT_INTERVAL
        const val DEFAULT_ELECTRICITY_INTERVAL = DEFAULT_INTERVAL

        /** 两个域共用的硬下限：1 分钟；低于它钳制并提示。 */
        const val MIN_INTERVAL = 1
        const val MIN_TIMETABLE_INTERVAL = MIN_INTERVAL
        const val MIN_ELECTRICITY_INTERVAL = MIN_INTERVAL

        /** 上限：一天一次已经等于「不自动查」。 */
        const val MAX_INTERVAL = 1440

        const val INTERVAL_UNIT = "分钟"

        /**
         * 纯钳制函数（不依赖 Android，单测直接调用生产实现）。
         *
         * `minimum` 也钳到合法区间：外部传入 MIN_INTERVAL > MAX_INTERVAL 时
         * `coerceIn` 会抛 IllegalArgumentException，不能让设置页因此崩掉。
         */
        fun clamped(minutes: Int, minimum: Int = MIN_INTERVAL): Int {
            val low = minimum.coerceIn(MIN_INTERVAL, MAX_INTERVAL)
            // coerceIn 对负数 / 超大值也安全；Low > MAX_INTERVAL 时已钳住。
            return minutes.coerceIn(low, MAX_INTERVAL)
        }

        /**
         * 间隔是否已过的纯函数版本（单测用；不读系统时钟）。
         *
         * `last <= 0` 表示从没查过，视为「已到期」。时钟异常（`last > now`）
         * 时重置为已到期：单调时钟重启后归零，wall-clock 回拨也不该永久抑制自动查询。
         * 用饱和减法避免 `now - last` 在极端注入值下溢出。
         */
        fun isDueAt(last: Long, minutes: Int, now: Long): Boolean {
            if (last <= 0L) return true
            if (now <= last) return true
            val intervalMs = clamped(minutes).toLong() * 60_000L
            return now - last >= intervalMs
        }

        private const val KEY_TIMETABLE_INTERVAL = "timetable_interval_minutes"
        private const val KEY_ELECTRICITY_INTERVAL = "electricity_interval_minutes"
        private const val KEY_CAMPUS_WIFI_ONLY = "campus_wifi_only"
        private const val KEY_REQUEST_LOG = "request_log_enabled"
        private const val KEY_LAST_TIMETABLE_SYNC = "last_timetable_sync_at"
        private const val KEY_LAST_ELECTRICITY = "last_electricity_at"
        private const val KEY_ELECTRICITY_CACHE = "electricity_cache_json"
        private const val KEY_TIMETABLE_CACHE = "timetable_cache_json"
    }
}

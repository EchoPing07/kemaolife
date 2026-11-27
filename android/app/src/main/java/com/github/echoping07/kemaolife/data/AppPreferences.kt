package com.github.echoping07.kemaolife.data

import android.content.Context
import android.content.SharedPreferences
import android.os.SystemClock
import java.io.File

/**
 * 非敏感的行为偏好：查询间隔、探测开关、上次自动查询时刻、课表/电费缓存。
 *
 * 为什么不放 Keystore：这些值不是凭据（P5 只管学号 / 密码 / Cookie），
 * 塞进加密块只会让「读一个数字」也要过一遍 KeyStore；也不放
 * [Credentials]（那是 Init 要传给 Go 的东西，间隔只影响 UI 的触发时机）。
 *
 * 为什么课表与电费 JSON 缓存放在 `filesDir` 独立文件而非 `app_prefs`：
 * SharedPreferences 每次修改任何标量偏好都会重序列化整份 XML，大段 JSON 放进去会拖慢所有设置读写。
 * `filesDir` 同样受 `allowBackup=false` 与 `data_extraction_rules.xml`（`domain="root"`）保护。
 *
 * 隐私口径：电费缓存只写 UI 必需字段，不含 `roomName` / `roomVerify` / 学号 /
 * Cookie（P2）；`roomName` 是本人宿舍标识，既不需要展示也不需要落盘。
 * 课表镜像只写 `WeekSchedule` 的展示字段，同样不含 Cookie / 凭据。
 */
class AppPreferences(context: Context) {

    private val prefs = context.getSharedPreferences("app_prefs", Context.MODE_PRIVATE)
    private val timetableCacheFile = File(context.filesDir, TIMETABLE_CACHE_FILE)
    private val electricityCacheFile = File(context.filesDir, ELECTRICITY_CACHE_FILE)
    private val cacheFileLock = Any()

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

    /** 电费自动查询的最小间隔（分钟）。与课表同下限、同默认值（docs/android.md）。 */
    var electricityIntervalMinutes: Int
        get() = clamped(prefs.getInt(KEY_ELECTRICITY_INTERVAL, DEFAULT_INTERVAL), MIN_INTERVAL)
        set(value) = prefs.editCommit {
            putInt(KEY_ELECTRICITY_INTERVAL, clamped(value, MIN_INTERVAL))
        }

    /** 电费低阈值通知开关（默认打开，docs/android.md）。 */
    var electricityAlertEnabled: Boolean
        get() = prefs.getBoolean(KEY_ELECTRICITY_ALERT_ENABLED, DEFAULT_ELECTRICITY_ALERT_ENABLED)
        set(value) = prefs.editCommit { putBoolean(KEY_ELECTRICITY_ALERT_ENABLED, value) }

    /** 电费通知触发方式：`balance`（默认）或 `days`（预计天数）。 */
    var electricityAlertMode: String
        get() = normalizeAlertMode(prefs.getString(KEY_ELECTRICITY_ALERT_MODE, ALERT_MODE_BALANCE))
        set(value) = prefs.editCommit {
            putString(KEY_ELECTRICITY_ALERT_MODE, normalizeAlertMode(value))
        }

    /** 电费余额提醒阈值（度，默认 10 度）；以字符串落盘避免 Float 精度漂移。 */
    var electricityBalanceThreshold: Double
        get() {
            val raw = prefs.getString(KEY_ELECTRICITY_BALANCE_THRESHOLD, null)?.toDoubleOrNull()
            return if (raw == null) DEFAULT_BALANCE_THRESHOLD else clampedBalanceThreshold(raw)
        }
        set(value) = prefs.editCommit {
            putString(
                KEY_ELECTRICITY_BALANCE_THRESHOLD,
                formatBalanceThreshold(clampedBalanceThreshold(value)),
            )
        }

    /** 电费预计可用天数提醒阈值（天，默认 3 天）。 */
    var electricityDaysThreshold: Int
        get() = clampedDaysThreshold(prefs.getInt(KEY_ELECTRICITY_DAYS_THRESHOLD, DEFAULT_DAYS_THRESHOLD))
        set(value) = prefs.editCommit {
            putInt(KEY_ELECTRICITY_DAYS_THRESHOLD, clampedDaysThreshold(value))
        }

    /**
     * 原子保存电费余额与预计天数阈值；任一值变化时在同一次 `commit()` 中重置告警去重状态，
     * 避免设置页保存时连续触发多次 fsync。
     */
    fun setElectricityAlertThresholds(balanceThreshold: Double, daysThreshold: Int) {
        val nextBalance = clampedBalanceThreshold(balanceThreshold)
        val nextDays = clampedDaysThreshold(daysThreshold)
        val changed =
            electricityBalanceThreshold != nextBalance ||
                electricityDaysThreshold != nextDays
        prefs.editCommit {
            putString(KEY_ELECTRICITY_BALANCE_THRESHOLD, formatBalanceThreshold(nextBalance))
            putInt(KEY_ELECTRICITY_DAYS_THRESHOLD, nextDays)
            if (changed) {
                putBoolean(KEY_ALERT_BALANCE_ALERTED, false)
                putBoolean(KEY_ALERT_DAYS_ALERTED, false)
            }
        }
    }

    /** 电费提醒去重状态（对应 Go `core/alert.State`），跨冷启动保留，充值回升后由 Go 重置。 */
    var electricityAlertState: ElectricityAlertState
        get() = ElectricityAlertState(
            balanceAlerted = prefs.getBoolean(KEY_ALERT_BALANCE_ALERTED, false),
            daysAlerted = prefs.getBoolean(KEY_ALERT_DAYS_ALERTED, false),
        )
        set(value) = prefs.editCommit {
            putBoolean(KEY_ALERT_BALANCE_ALERTED, value.balanceAlerted)
            putBoolean(KEY_ALERT_DAYS_ALERTED, value.daysAlerted)
        }

    /** 是否已自动发起过一次系统通知权限申请（避免用户拒绝后每次启动反复打扰）。 */
    var notificationPermissionAsked: Boolean
        get() = prefs.getBoolean(KEY_NOTIFICATION_PERMISSION_ASKED, false)
        set(value) = prefs.editCommit { putBoolean(KEY_NOTIFICATION_PERMISSION_ASKED, value) }

    /** 教务高级：「课室显示优化」开关（默认开启，docs/android.md）。 */
    var optimizeLocationDisplay: Boolean
        get() = prefs.getBoolean(KEY_OPTIMIZE_LOCATION_DISPLAY, DEFAULT_OPTIMIZE_LOCATION_DISPLAY)
        set(value) = prefs.editCommit { putBoolean(KEY_OPTIMIZE_LOCATION_DISPLAY, value) }

    /** 教务高级：「课室前显示@」开关（默认开启，docs/android.md）。 */
    var prefixLocationAt: Boolean
        get() = prefs.getBoolean(KEY_PREFIX_LOCATION_AT, DEFAULT_PREFIX_LOCATION_AT)
        set(value) = prefs.editCommit { putBoolean(KEY_PREFIX_LOCATION_AT, value) }

    /** 教务高级：「过长隐藏」开关（默认关闭，docs/android.md）。 */
    var hideTimetableOverflow: Boolean
        get() = prefs.getBoolean(KEY_HIDE_TIMETABLE_OVERFLOW, DEFAULT_HIDE_TIMETABLE_OVERFLOW)
        set(value) = prefs.editCommit { putBoolean(KEY_HIDE_TIMETABLE_OVERFLOW, value) }

    /** 教务高级：「导出优化课室」开关（默认关闭，docs/android.md）。 */
    var optimizeLocationExport: Boolean
        get() = prefs.getBoolean(KEY_OPTIMIZE_LOCATION_EXPORT, DEFAULT_OPTIMIZE_LOCATION_EXPORT)
        set(value) = prefs.editCommit { putBoolean(KEY_OPTIMIZE_LOCATION_EXPORT, value) }

    /** 校园网「仅在 WiFi 下探测」开关（默认打开）。 */
    var campusWifiOnly: Boolean
        get() = prefs.getBoolean(KEY_CAMPUS_WIFI_ONLY, true)
        set(value) = prefs.editCommit { putBoolean(KEY_CAMPUS_WIFI_ONLY, value) }

    /**
     * 请求记录开关（默认关闭，docs/contract.md §6 规则 10）。
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
     * 上一次电费结果（UI 必需字段的 JSON，由 [ElectricityCache] 编解码）。
     *
     * 为什么要落盘：电费页冷启动时也要「先出缓存再联网」（docs/android.md）。
     * 存进 `filesDir/electricity_cache.json`；升级后首次读取若文件尚不存在，
     * 自动从旧版 `app_prefs` 迁移并移除旧键。
     */
    var electricityCacheJson: String
        get() = synchronized(cacheFileLock) {
            readCacheFile(
                file = electricityCacheFile,
                readLegacy = { legacyPrefValue(KEY_ELECTRICITY_CACHE) },
                removeLegacy = { removeLegacyPref(KEY_ELECTRICITY_CACHE) },
            )
        }
        set(value) = synchronized(cacheFileLock) {
            writeCacheFile(electricityCacheFile, value)
            removeLegacyPref(KEY_ELECTRICITY_CACHE)
        }

    /**
     * 上次课表周视图的展示镜像（[TimetableCache] 编码）。
     *
     * 冷启动首帧要在门面 Init 之前就渲染课表：正本快照含 Cookie 只能由 Go 读，
     * 而 Init 是 JNI 调用，排在 IO 协程里。这里只存 DTO 展示字段（docs/android.md），
     * 落在 `filesDir/timetable_cache.json`，首次读取自动迁移旧版 `app_prefs` 键。
     */
    var timetableCacheJson: String
        get() = synchronized(cacheFileLock) {
            readCacheFile(
                file = timetableCacheFile,
                readLegacy = { legacyPrefValue(KEY_TIMETABLE_CACHE) },
                removeLegacy = { removeLegacyPref(KEY_TIMETABLE_CACHE) },
            )
        }
        set(value) = synchronized(cacheFileLock) {
            writeCacheFile(timetableCacheFile, value)
            removeLegacyPref(KEY_TIMETABLE_CACHE)
        }

    /** 换账号时与电费缓存一起清掉，避免旧账号课表先显示（docs/contract.md §6 规则 2）。 */
    fun clearTimetableCache() {
        synchronized(cacheFileLock) {
            deleteCacheFile(timetableCacheFile)
            if (prefs.contains(KEY_TIMETABLE_CACHE)) {
                prefs.editCommit { remove(KEY_TIMETABLE_CACHE) }
            }
        }
    }

    /** 清除电费缓存：换账号 / 凭据失效时用，旧账号数据不能先显示。 */
    fun clearElectricityCache() {
        synchronized(cacheFileLock) {
            deleteCacheFile(electricityCacheFile)
            prefs.editCommit {
                if (prefs.contains(KEY_ELECTRICITY_CACHE)) remove(KEY_ELECTRICITY_CACHE)
                putLong(KEY_LAST_ELECTRICITY, 0L)
            }
        }
    }

    private fun legacyPrefValue(key: String): String? =
        if (prefs.contains(key)) prefs.getString(key, "").orEmpty() else null

    private fun removeLegacyPref(key: String) {
        if (prefs.contains(key)) {
            prefs.edit().remove(key).apply()
        }
    }

    /** 重置电费低阈值通知的去重标记（换账号 / 调整阈值 / 切换通知方式时调用）。 */
    fun clearElectricityAlertState() {
        prefs.editCommit {
            putBoolean(KEY_ALERT_BALANCE_ALERTED, false)
            putBoolean(KEY_ALERT_DAYS_ALERTED, false)
        }
    }

    /** 单调时钟上的「现在」；用于所有自动查询节流。 */
    fun nowElapsed(): Long = SystemClock.elapsedRealtime()

    /** 间隔是否已过；`last == 0`（从没查过）视为已过。 */
    fun isDue(last: Long, minutes: Int, now: Long = nowElapsed()): Boolean =
        isDueAt(last, minutes, now)

    companion object {
        /** 课表与电费的最小间隔：默认一致，均为 5 分钟（docs/android.md）。 */
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

        const val DEFAULT_OPTIMIZE_LOCATION_DISPLAY = true
        const val DEFAULT_PREFIX_LOCATION_AT = true
        const val DEFAULT_HIDE_TIMETABLE_OVERFLOW = false
        const val DEFAULT_OPTIMIZE_LOCATION_EXPORT = false

        /** 电费低阈值通知默认值（docs/android.md）：默认打开、按电费 10 度提醒，预计天数默认 3 天。 */
        const val DEFAULT_ELECTRICITY_ALERT_ENABLED = true
        const val ALERT_MODE_BALANCE = "balance"
        const val ALERT_MODE_DAYS = "days"
        const val DEFAULT_BALANCE_THRESHOLD = 10.0
        const val MIN_BALANCE_THRESHOLD = 0.1
        const val MAX_BALANCE_THRESHOLD = 9999.0
        const val DEFAULT_DAYS_THRESHOLD = 3
        const val MIN_DAYS_THRESHOLD = 1
        const val MAX_DAYS_THRESHOLD = 365

        fun normalizeAlertMode(raw: String?): String =
            if (raw?.trim()?.lowercase(java.util.Locale.US) == ALERT_MODE_DAYS) {
                ALERT_MODE_DAYS
            } else {
                ALERT_MODE_BALANCE
            }

        /** 把电费余额阈值钳制进合法正数区间并保留最多两位小数。 */
        fun clampedBalanceThreshold(value: Double): Double {
            if (!value.isFinite()) return DEFAULT_BALANCE_THRESHOLD
            val clamped = value.coerceIn(MIN_BALANCE_THRESHOLD, MAX_BALANCE_THRESHOLD)
            return kotlin.math.round(clamped * 100.0) / 100.0
        }

        /** 把预计可用天数阈值钳制进 [1, 365]。 */
        fun clampedDaysThreshold(days: Int): Int =
            days.coerceIn(MIN_DAYS_THRESHOLD, MAX_DAYS_THRESHOLD)

        /** 简洁格式化余额阈值：10.0 → "10"，10.5 → "10.5"，10.25 → "10.25"。 */
        fun formatBalanceThreshold(value: Double): String {
            val rounded = kotlin.math.round(value * 100.0) / 100.0
            return if (rounded == rounded.toLong().toDouble()) {
                rounded.toLong().toString()
            } else {
                String.format(java.util.Locale.US, "%.2f", rounded)
                    .trimEnd('0')
                    .trimEnd('.')
            }
        }

        /** 过滤余额阈值输入：仅允许数字与单个小数点，限制整数位与小数位长度。 */
        fun sanitizeDecimalInput(
            text: String,
            maxIntegerDigits: Int = 4,
            maxFractionDigits: Int = 2,
        ): String {
            val out = StringBuilder()
            var seenDot = false
            var intDigits = 0
            var fracDigits = 0
            for (ch in text) {
                when {
                    ch in '0'..'9' -> {
                        if (!seenDot && intDigits < maxIntegerDigits) {
                            out.append(ch)
                            intDigits++
                        } else if (seenDot && fracDigits < maxFractionDigits) {
                            out.append(ch)
                            fracDigits++
                        }
                    }

                    ch == '.' && !seenDot -> {
                        if (out.isEmpty()) {
                            out.append('0')
                            intDigits = 1
                        }
                        out.append('.')
                        seenDot = true
                    }
                }
            }
            return out.toString()
        }

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

        /**
         * 同步读取文件缓存；文件尚不存在且旧版 `app_prefs` 有残留时，透明迁移到文件并删除旧键。
         * 纯 JVM 实现，便于单测直接覆盖原子写与迁移路径（不依赖 Robolectric）。
         */
        internal fun readCacheFile(
            file: File,
            readLegacy: () -> String?,
            removeLegacy: () -> Unit,
        ): String {
            if (file.isFile) {
                return runCatching { file.readText(Charsets.UTF_8) }.getOrDefault("")
            }
            val legacy = readLegacy() ?: return ""
            if (legacy.isEmpty()) {
                removeLegacy()
                return ""
            }
            if (writeCacheFile(file, legacy)) {
                removeLegacy()
            }
            return legacy
        }

        /** 原子写 UTF-8 缓存文件（同目录临时文件 + rename）；空串时删除文件。 */
        internal fun writeCacheFile(file: File, content: String): Boolean {
            if (content.isEmpty()) {
                deleteCacheFile(file)
                return true
            }
            val parent = file.parentFile
            val tmp = File(parent, "${file.name}.tmp")
            return try {
                if (parent != null && !parent.exists()) parent.mkdirs()
                tmp.writeText(content, Charsets.UTF_8)
                if (!tmp.renameTo(file)) {
                    // POSIX (Android/Linux) renameTo 原子覆盖已有目标；Windows JVM 单测目标已存在时返回 false。
                    if (!file.delete() || !tmp.renameTo(file)) {
                        tmp.delete()
                        return false
                    }
                }
                true
            } catch (_: Exception) {
                runCatching { tmp.delete() }
                false
            }
        }

        /** 删除缓存文件及可能残留的临时文件。 */
        internal fun deleteCacheFile(file: File) {
            runCatching { if (file.exists()) file.delete() }
            val tmp = File(file.parentFile, "${file.name}.tmp")
            runCatching { if (tmp.exists()) tmp.delete() }
        }

        private const val TIMETABLE_CACHE_FILE = "timetable_cache.json"
        private const val ELECTRICITY_CACHE_FILE = "electricity_cache.json"

        private const val KEY_TIMETABLE_INTERVAL = "timetable_interval_minutes"
        private const val KEY_ELECTRICITY_INTERVAL = "electricity_interval_minutes"
        private const val KEY_ELECTRICITY_ALERT_ENABLED = "electricity_alert_enabled"
        private const val KEY_ELECTRICITY_ALERT_MODE = "electricity_alert_mode"
        private const val KEY_ELECTRICITY_BALANCE_THRESHOLD = "electricity_balance_threshold"
        private const val KEY_ELECTRICITY_DAYS_THRESHOLD = "electricity_days_threshold"
        private const val KEY_ALERT_BALANCE_ALERTED = "electricity_alert_balance_alerted"
        private const val KEY_ALERT_DAYS_ALERTED = "electricity_alert_days_alerted"
        private const val KEY_NOTIFICATION_PERMISSION_ASKED = "notification_permission_asked"
        private const val KEY_OPTIMIZE_LOCATION_DISPLAY = "optimize_location_display"
        private const val KEY_PREFIX_LOCATION_AT = "prefix_location_at"
        private const val KEY_HIDE_TIMETABLE_OVERFLOW = "hide_timetable_overflow"
        private const val KEY_OPTIMIZE_LOCATION_EXPORT = "optimize_location_export"
        private const val KEY_CAMPUS_WIFI_ONLY = "campus_wifi_only"
        private const val KEY_REQUEST_LOG = "request_log_enabled"
        private const val KEY_LAST_TIMETABLE_SYNC = "last_timetable_sync_at"
        private const val KEY_LAST_ELECTRICITY = "last_electricity_at"
        private const val KEY_ELECTRICITY_CACHE = "electricity_cache_json"
        private const val KEY_TIMETABLE_CACHE = "timetable_cache_json"
    }
}

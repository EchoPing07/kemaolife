package com.github.echoping07.kemaolife.ui

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.github.echoping07.kemaolife.data.AppException
import com.github.echoping07.kemaolife.data.AppPreferences
import com.github.echoping07.kemaolife.data.Bridge
import com.github.echoping07.kemaolife.data.CachedWeek
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.data.CampusStatus
import com.github.echoping07.kemaolife.data.CredentialStore
import com.github.echoping07.kemaolife.data.Credentials
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.DEFAULT_CONTINUOUS_CORNERS
import com.github.echoping07.kemaolife.data.DEFAULT_DYNAMIC_COLOR
import com.github.echoping07.kemaolife.data.ElectricityReport
import com.github.echoping07.kemaolife.data.LogEntry
import com.github.echoping07.kemaolife.data.LogState
import com.github.echoping07.kemaolife.data.NetworkState
import com.github.echoping07.kemaolife.data.ProbeResult
import com.github.echoping07.kemaolife.data.Repository
import com.github.echoping07.kemaolife.data.ThemePreferences
import com.github.echoping07.kemaolife.data.TimetableCache
import com.github.echoping07.kemaolife.data.TimetableExport
import com.github.echoping07.kemaolife.data.WeekSchedule
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/**
 * 单例式的应用状态容器：三个 tab 共用一个 ViewModel（`MainActivity` 持有）。
 *
 * 课表状态只有一个来源：`week`（整周 7 天）。离线标记 `weekStale` 只由
 * `timetable.sync` 的成功 / 失败事务提交，任何纯读缓存的 `repo.week()` 都不得覆写它：
 * `CacheOnly` 的 `stale` 恒为 false，那是「本次没有刷新」，不是「上次刷新成功了」。
 *
 * 打开 App 的顺序（docs/002 §5.3）：**构造函数里同步**读课表 / 电费镜像缓存，
 * 首帧立刻渲染，再无条件联网覆盖；其余自动触发（切回课表 tab、进入更多页、
 * 回到前台之外）按最小间隔节流，手动不受限。
 *
 * 课表镜像是 DTO 展示字段的 Kotlin 副本：正本快照含 Cookie，只能由 Go 在
 * `Init` 后读，而 `Init` 排在 IO 协程里；没有镜像，课表位会先空一帧，
 * 甚至把「桥接尚未初始化」的失败信封当成空态文案。Gate 与 `pendingOpen`
 * 保证冷启动只发起一轮无条件联网，且不会在 `Init` 之前误触发。
 */
class AppViewModel(app: Application) : AndroidViewModel(app) {

    private val store = CredentialStore(app)
    private val bridge = Bridge(app)
    private val repo = Repository(bridge)
    private val themePreferences = ThemePreferences(app)
    private val preferences = AppPreferences(app)
    private val networkState = NetworkState(app)

    // API 24 无 java.time，且未启用 desugaring；用平台格式化取本地日期，不自行算时区（A3）。
    private val today: String = SimpleDateFormat("yyyy-MM-dd", Locale.US).format(Date())

    // 每个链路只保留一个在途 Job：用 Job.isActive 判重入，比普通 Boolean 更可靠
    // ——协程被取消时 Job 自然完成，不会留下永远为 true 的标志。
    private var syncJob: Job? = null
    private var electricityJob: Job? = null
    private var campusJob: Job? = null
    private var campusActionJob: Job? = null
    private var probeJob: Job? = null
    private var exportJob: Job? = null
    private var requestLogJob: Job? = null
    private var requestLogDetailJob: Job? = null

    // 图标结果态只跟随显式手动点击；自动查询共用 Job，但不能反复闪成功提示。
    internal val timetableFeedback = ActionFeedbackController()
    internal val electricityFeedback = ActionFeedbackController()
    internal val campusFeedback = ActionFeedbackController()
    internal val campusActionFeedback = ActionFeedbackController()

    // 课表读取代次：切周 / 同步回读都可能并发，只允许最新一次写 state。
    @Volatile
    private var weekSeq = 0L

    /**
     * 翻页容器的邻周缓存（周次 → 该周整周数据）；对外通过 `AppState.weekNeighbors` 暴露。
     *
     * 为什么必须有它：跟手翻页时 pager 会同时组合相邻页，若邻居页现读缓存
     * 就来不及渲染 —— 拖到一半只能看到留白，「跟手」就失去意义。
     * 只存内存不落盘：正本快照已含全部课节，落盘只会多一份可过期的副本。
     */
    @Volatile
    private var weekCache: Map<Int, WeekSchedule> = emptyMap()

    // 预取只保留一个在途 Job：滑得快时不需要为每个中间周都排队读一次。
    private var prefetchJob: Job? = null

    // 测试弹窗代次：关闭再开另一个功能域时，旧结果不得写回新弹窗。
    @Volatile
    private var probeSeq = 0L

    private val saveMutex = Mutex()

    /** Init 未就绪时到来的「打开」只挂起一次，Init 完成后补跑，不丢也不重。 */
    @Volatile
    private var pendingOpen = false

    // 首帧缓存同步读：SharedPreferences 的读取很轻，且必须在 ViewModel 构造时完成，
    // 不能像以前那样塞进 viewModelScope.launch（那会等到协程调度 + JNI Init 之后）。
    private val cachedWeek: CachedWeek? = TimetableCache.decode(preferences.timetableCacheJson).also {
        // 旧版本镜像可能含整门课聚合地点：立即清掉，首帧不再显示不可信教室。
        if (it == null && preferences.timetableCacheJson.isNotEmpty()) {
            preferences.clearTimetableCache()
        }
    }

    private val _state = MutableStateFlow(
        AppState(
            todayDate = today,
            currentWeek = cachedWeek?.schedule?.week,
            week = cachedWeek?.schedule,
            // 更多页「今日概览」也从镜像里取：否则首帧课表出了、概览还空着。
            todayDay = cachedWeek?.schedule?.days?.firstOrNull { it.date == today },
            weekSyncedAt = cachedWeek?.syncedAt.orEmpty(),
            electricity = cachedElectricityReport(),
            timetableInterval = preferences.timetableIntervalMinutes,
            electricityInterval = preferences.electricityIntervalMinutes,
            campusWifiOnly = preferences.campusWifiOnly,
            requestLogEnabled = preferences.requestLogEnabled,
        ),
    )
    val state: StateFlow<AppState> = _state.asStateFlow()

    init {
        val credentials = store.load()
        _state.update {
            it.copy(
                credentials = credentials,
                themeColor = themePreferences.load(),
                themeMode = themePreferences.loadMode(),
                dynamicColor = themePreferences.loadDynamicColor(),
                amoled = themePreferences.loadAmoled(),
                continuousCorners = themePreferences.loadContinuousCorners(),
            )
        }
        // 老版本用 wall-clock 存时间戳，升到单调时钟后那些值全部失效，重查一次即可。
        migrateClockBase()
        viewModelScope.launch {
            // 冷启动：门面 Init 之后先用 CacheOnly 校正跨周 / 跨学期的镜像，
            // 再无条件联网。镜像已在首帧显示，这里的回读只负责「更正」。
            withContext(Dispatchers.IO) { repo.init(credentials, requestLog = preferences.requestLogEnabled) }
            _state.update { it.copy(bridgeReady = repo.isReady(), bridgeError = repo.readyError()) }

            fetchWeek(0)
            // 「每次打开」是特殊的自动：不受最小间隔约束（docs/002 §5.3）。
            // 与挂起的 ON_RESUME 合并，保证冷启动恰好一轮。
            pendingOpen = true
            replayPendingOpen()
        }
    }

    /** wall-clock 时间戳迁移：比单调时钟还大的旧值一律清零，首次自动查询照常触发。 */
    private fun migrateClockBase() {
        val now = preferences.nowElapsed()
        if (preferences.lastTimetableSyncAt > now) preferences.lastTimetableSyncAt = 0L
        if (preferences.lastElectricityAt > now) preferences.lastElectricityAt = 0L
    }

    /** 打开 App（冷启动或从后台回到前台）：课表与电费都无条件尝试获取一次。 */
    fun onAppOpened(force: Boolean = true) {
        // Init 是 JNI 调用，可能晚于 ON_RESUME。挂起而不是丢弃：Init 完成后补跑，
        // 既不撞 Bridge 的未初始化拦截，也不会永久少掉这一轮无条件联网。
        if (StartupTiming.shouldDefer(repo.isReady())) {
            pendingOpen = true
            return
        }
        dispatchOpen(force)
    }

    /** Init 完成后消费挂起的「打开」；冷启动与后台返回共用这一条路径。 */
    private fun replayPendingOpen() {
        if (StartupTiming.consumePending(repo.isReady(), pendingOpen)) {
            pendingOpen = false
            dispatchOpen(force = true)
        }
    }

    private fun dispatchOpen(force: Boolean) {
        // 每个链路同时只留一个在途请求：回到前台时冷启动同步可能还在跑，
        // 不需要再叠一轮完整 CAS 登录 / 电费查询。
        if (syncJob?.isActive != true) maybeAutoSync(force)
        if (electricityJob?.isActive != true) maybeAutoElectricity(force)
        val s = _state.value
        if (s.campus == null) refreshCampusStatus(manual = false)
    }

    /** 进入「更多」页：电费按间隔节流；校园网状态未知时判态一次。ViewModel 自己读最新 state。 */
    fun onEnterMore() {
        maybeAutoElectricity()
        val s = _state.value
        if (s.campus == null || s.campusSkipped) refreshCampusStatus(manual = false)
    }

    // ---- 课表 -------------------------------------------------------------

    /**
     * 读一次整周缓存并提交到 state（0 请求）。
     *
     * `fromSync=true` 表示这是「同步成功后的回读」：只有它才能把 weekStale 清成 false，
     * 且只有回读成功才提交数据时间；回读失败保留旧数据与旧数据时间并置离线标记，
     * 避免「同步写盘成功但界面还是旧课表」被当成新数据（docs/002 §5.3）。
     */
    private suspend fun fetchWeek(
        week: Int = 0,
        fromSync: Boolean = false,
    ): Boolean {
        val seq = ++weekSeq
        _state.update { it.copy(weekLoading = true) }
        // 只把「当周」写进首帧镜像：切周读到的是别的周次，
        // 存进去会让下次冷启动先闪一帧过期周次。当周由 Go 的 week=0 判定，
        // Kotlin 不自己算周（AGENTS A3）。
        val result = withContext(Dispatchers.IO) {
            repo.week(week).onSuccess { view ->
                if (week == 0) cacheWeek(view.schedule, view.syncedAt)
            }
        }
        return result.fold(
            onSuccess = { view ->
                applyWeek(
                    schedule = view.schedule,
                    syncedAt = view.syncedAt,
                    fromSync = fromSync,
                    seq = seq,
                    viewLastWeek = view.lastWeek,
                    currentWeek = if (week == 0) view.schedule.week else null,
                )
                // 落页后补一格邻页：保证下一次拖动时邻居页已经有内容（docs/002 §5.3 tab1）。
                prefetchNeighborWeeks()
                seq == weekSeq
            },
            onFailure = { ex ->
                _state.update { s ->
                    if (seq != weekSeq) return@update s
                    // 门面未就绪是启动时序问题（自动查询已挂起补跑），不是课表数据错误；
                    // 不能让它变成课表位 / 更多页概览的文案。
                    val internalNotReady = !repo.isReady() && (ex as? AppException)?.code == 2001
                    s.copy(
                        weekLoading = false,
                        timetableError = if (internalNotReady) null
                        else (ex as? AppException)?.displayMessage ?: "课表读取失败",
                        // 同步成功但回读失败：旧课表不能冒充本次新数据。
                        weekStale = if (fromSync && s.week != null) true else s.weekStale,
                    )
                }
                false
            },
        )
    }

    /** 打开 App、切周、保存设置后的纯缓存读取；不覆写离线标记。 */
    fun loadCachedWeek(week: Int = 0) {
        viewModelScope.launch { fetchWeek(week) }
    }

    /**
     * 预取当前周两侧的整周数据（docs/002 §5.3 tab1 翻页）。
     *
     * 失败一律静默：预取只是为了让邻居页拖动时就有内容，
     * 失败了落页时还会再走一次 `CacheOnly` 读取，不得因此弹错误或清空当前页。
     */
    fun prefetchNeighborWeeks() {
        if (prefetchJob?.isActive == true) return
        val s = _state.value
        val current = s.week?.week ?: return
        val targets = WeekPager.neighbors(current, s.lastWeek).filter { it !in weekCache }
        if (targets.isEmpty()) return
        prefetchJob = viewModelScope.launch {
            val fetched = withContext(Dispatchers.IO) {
                coroutineScope {
                    targets.map { week ->
                        async {
                            // 预取同样只读缓存（0 请求），不触发任何联网动作。
                            repo.week(week).getOrNull()?.schedule?.let { week to it }
                        }
                    }.awaitAll()
                }
            }.filterNotNull().toMap()
            if (fetched.isEmpty()) return@launch
            weekCache = weekCache + fetched
            val keep = WeekPager.retained(weekCache.keys, _state.value.week?.week ?: current)
            weekCache = weekCache.filterKeys { it in keep }
            // 同步进 state：pager 的邻居页靠它渲染，不写入 state 就永远看不到预取结果。
            _state.update { it.copy(weekNeighbors = weekCache) }
        }
    }

    /** 联网同步一次课表；成功后回读缓存覆盖界面。不做间隔判断。 */
    internal fun syncTimetable(feedbackId: Long? = null) {
        if (syncJob?.isActive == true) return
        syncJob = viewModelScope.launch {
            try {
                _state.update { it.copy(timetableSyncing = true) }
                val result = withContext(Dispatchers.IO) { repo.sync() }
                val attemptedAt = preferences.nowElapsed()
                preferences.lastTimetableSyncAt = attemptedAt

                result.fold(
                    onSuccess = {
                        // 用户在同步期间切了周也不跳过刷新：否则刚同步成功却不更新界面；
                        // 回读当前用户正在看的那一周，并用 generation 防止旧结果覆盖新选择。
                        val desired = _state.value.week?.week ?: 0
                        val refreshed = fetchWeek(desired, fromSync = true)
                        feedbackId?.let {
                            timetableFeedback.complete(
                                it,
                                if (refreshed) ActionFeedback.SUCCESS else ActionFeedback.FAILURE,
                            )
                        }
                        _state.update { it.copy(timetableSyncing = false) }
                    },
                    onFailure = { ex ->
                        feedbackId?.let { timetableFeedback.complete(it, actionFeedbackFor(ex)) }
                        _state.update { s ->
                            s.copy(
                                timetableSyncing = false,
                                // 只标离线，不弹错误、不清缓存、不改数据时间。
                                weekStale = s.week != null,
                            )
                        }
                        if (_state.value.week == null) pushError(ex)
                    },
                )
            } finally {
                _state.update { it.copy(timetableSyncing = false, weekLoading = false) }
            }
        }
    }

    /** 按间隔决定要不要自动同步（切回课表 tab 等）。`force=true` 用于每次打开 App。 */
    fun maybeAutoSync(force: Boolean = false) {
        if (syncJob?.isActive == true) return
        if (!force && !preferences.isDue(
                preferences.lastTimetableSyncAt,
                _state.value.timetableInterval,
            )
        ) {
            return
        }
        val c = _state.value.credentials
        if (c.studentId.isBlank() || c.eduPassword.isBlank()) return
        syncTimetable()
    }

    /** 头部刷新按钮：手动动作，不受最小间隔约束。 */
    fun refreshWeek() {
        if (syncJob?.isActive == true) return
        val c = _state.value.credentials
        if (c.studentId.isBlank() || c.eduPassword.isBlank()) {
            timetableFeedback.begin()
            timetableFeedback.complete(timetableFeedback.generation, ActionFeedback.UNAVAILABLE)
            return
        }
        syncTimetable(timetableFeedback.begin())
    }

    // ---- 导出 -------------------------------------------------------------

    /**
     * 导出整学期课表为 WakeUp 课程表 CSV（只读缓存，0 请求）。
     *
     * 为什么不在导出前顺手同步一次：导出是「把本机已有的课表带走」，
     * 用户要的是现在这份数据；顺带联网会让一次手动点按变成一次可能失败的请求，
     * 还要重新处理离线标记。数据时间随 [AppState.exportSheet] 一并展示，
     * 用户能自己看到这份 CSV 对应哪次同步（docs/002 §5.3）。
     */
    fun exportTimetable() {
        if (exportJob?.isActive == true) return
        exportJob = viewModelScope.launch {
            _state.update { it.copy(exportLoading = true) }
            try {
                val result = withContext(Dispatchers.IO) { repo.exportTimetable() }
                result.fold(
                    onSuccess = { export ->
                        _state.update {
                            it.copy(exportLoading = false, exportSheet = ExportState(export = export))
                        }
                    },
                    onFailure = { ex ->
                        _state.update { it.copy(exportLoading = false) }
                        pushError(ex)
                    },
                )
            } finally {
                _state.update { it.copy(exportLoading = false) }
            }
        }
    }

    /** 用户保存完或关掉弹层：清掉手里的 CSV 正文，不长期驻留内存。 */
    fun closeExport() = _state.update { it.copy(exportSheet = null) }

    // ---- 电费 -------------------------------------------------------------

    /** 电费查询并渲染；不做间隔判断（手动刷新与「测试」都走这里）。 */
    private fun queryElectricity(feedbackId: Long? = null) {
        if (electricityJob?.isActive == true) return
        electricityJob = viewModelScope.launch {
            try {
                _state.update { it.copy(electricityLoading = true) }
                val result = withContext(Dispatchers.IO) { repo.electricity() }
                val attemptedAt = preferences.nowElapsed()
                preferences.lastElectricityAt = attemptedAt
                result.fold(
                    onSuccess = { report ->
                        feedbackId?.let { electricityFeedback.complete(it, ActionFeedback.SUCCESS) }
                        withContext(Dispatchers.IO) { cacheElectricityReport(report) }
                        _state.update {
                            it.copy(electricity = report, electricityLoading = false)
                        }
                    },
                    onFailure = { ex ->
                        feedbackId?.let { electricityFeedback.complete(it, actionFeedbackFor(ex)) }
                        // 失败不清缓存：旧电费照常可看。
                        // 自动查询每次打开都会跑，失败文案不进全局 snackbar；
                        // 诊断走设置页电费「测试」弹窗与日志页（docs/002 §5.3）。
                        _state.update { it.copy(electricityLoading = false) }
                    },
                )
            } finally {
                _state.update { it.copy(electricityLoading = false) }
            }
        }
    }

    /** 按间隔决定要不要自动查一次电费；`force=true` 用于每次打开 App。 */
    fun maybeAutoElectricity(force: Boolean = false) {
        if (electricityJob?.isActive == true) return
        if (!force && !preferences.isDue(
                preferences.lastElectricityAt,
                _state.value.electricityInterval,
            )
        ) {
            return
        }
        if (_state.value.credentials.electricityAccount.isBlank()) return
        queryElectricity()
    }

    fun refreshElectricity() {
        if (electricityJob?.isActive == true) return
        if (_state.value.credentials.electricityAccount.isBlank()) {
            electricityFeedback.begin()
            electricityFeedback.complete(electricityFeedback.generation, ActionFeedback.UNAVAILABLE)
            return
        }
        queryElectricity(electricityFeedback.begin())
    }

    // ---- 校园网 -----------------------------------------------------------

    /** 「仅在 WiFi 下探测」打开时，非 WiFi 一律不发任何 `campusnet.*` 请求。 */
    private fun campusAllowed(): Boolean =
        !_state.value.campusWifiOnly || networkState.isOnWifi()

    private fun applyCampusOfflineSkipped() {
        campusJob?.cancel()
        campusActionJob?.cancel()
        _state.update { s ->
            s.copy(
                campus = CampusStatus(state = CampusState.OFFLINE, errorMsg = "", checkedAt = ""),
                campusSkipped = true,
                campusLoading = false,
                campusActing = null,
            )
        }
    }

    private fun applyCampusFailure(ex: Throwable) {
        val message = (ex as? AppException)?.displayMessage ?: "校园网判态失败"
        _state.update { s ->
            s.copy(
                // 探测失败归 offline（docs/002 §3.3），不留在「加载中…」。
                campus = CampusStatus(state = CampusState.OFFLINE, errorMsg = message, checkedAt = ""),
                campusSkipped = false,
                campusLoading = false,
            )
        }
    }

    fun refreshCampusStatus(manual: Boolean = true) {
        if (!campusAllowed()) {
            applyCampusOfflineSkipped()
            if (manual) {
                campusFeedback.begin()
                campusFeedback.complete(campusFeedback.generation, ActionFeedback.UNAVAILABLE)
            }
            return
        }
        val feedbackId = if (manual) campusFeedback.begin() else null
        campusJob?.cancel()
        campusJob = viewModelScope.launch {
            try {
                _state.update { it.copy(campusLoading = true, campusSkipped = false) }
                val result = withContext(Dispatchers.IO) { repo.campusStatus() }
                result.fold(
                    onSuccess = { status ->
                        feedbackId?.let { campusFeedback.complete(it, ActionFeedback.SUCCESS) }
                        _state.update { it.copy(campus = status, campusLoading = false) }
                    },
                    onFailure = { ex ->
                        feedbackId?.let { campusFeedback.complete(it, actionFeedbackFor(ex)) }
                        applyCampusFailure(ex)
                        if (manual) pushError(ex)
                    },
                )
            } finally {
                _state.update { it.copy(campusLoading = false) }
            }
        }
    }

    /** 手动登录 / 下线 / 重登；不再失败后复探（会绕过 WiFi 门控且多一次请求）。 */
    fun campusAction(action: CampusAction) {
        if (campusActionJob?.isActive == true) return
        val c = _state.value.credentials
        if (c.campusUser.isBlank() || c.campusPassword.isBlank() || !campusAllowed()) {
            if (!campusAllowed()) applyCampusOfflineSkipped()
            campusActionFeedback.begin()
            campusActionFeedback.complete(campusActionFeedback.generation, ActionFeedback.UNAVAILABLE)
            return
        }
        val feedbackId = campusActionFeedback.begin()
        campusActionJob?.cancel()
        campusActionJob = viewModelScope.launch {
            try {
                _state.update { it.copy(campusActing = action) }
                val result = withContext(Dispatchers.IO) {
                    when (action) {
                        CampusAction.LOGIN -> repo.campusLogin()
                        CampusAction.LOGOUT -> repo.campusLogout()
                        CampusAction.RELOGIN -> repo.campusRelogin()
                    }
                }
                result.fold(
                    onSuccess = { status ->
                        campusActionFeedback.complete(feedbackId, ActionFeedback.SUCCESS)
                        _state.update { it.copy(campus = status, campusActing = null) }
                    },
                    onFailure = { ex ->
                        campusActionFeedback.complete(feedbackId, actionFeedbackFor(ex))
                        pushError(ex)
                        applyCampusFailure(ex)
                        _state.update { it.copy(campusActing = null) }
                    },
                )
            } finally {
                _state.update { it.copy(campusActing = null, campusLoading = false) }
            }
        }
    }

    fun setCampusWifiOnly(enabled: Boolean) {        preferences.campusWifiOnly = enabled
        _state.update { it.copy(campusWifiOnly = enabled) }
        if (enabled && !networkState.isOnWifi()) {
            // 打开开关时立刻把残留的 online/unauth 置为 offline，动作按钮同时禁用。
            applyCampusOfflineSkipped()
        } else {
            refreshCampusStatus(manual = false)
        }
    }

    // ---- 请求记录（设置页「日志」） ---------------------------------------

    /** 首次进入日志页：读一次最近列表（0 请求，只读 Go 进程内存）。 */
    fun loadRequestLog() {
        if (requestLogJob?.isActive == true) return
        requestLogJob = viewModelScope.launch {
            val result = withContext(Dispatchers.IO) { repo.requestLog() }
            result.fold(
                onSuccess = { log ->
                    // 轮询每 2s 一次：内容没变就不动 state，避免无意义的重组。
                    if (_state.value.log != log) _state.update { it.copy(log = log) }
                },
                onFailure = { ex ->
                    // 门面未就绪是启动时序问题，不弹错；下次进入页面会再读一次。
                    if (!repo.isReady()) return@fold
                    pushError(ex)
                },
            )
        }
    }

    /**
     * 开关请求记录。
     *
     * 为什么要落盘后重新 `Init`：Go 侧记录器挂在会话的 HTTP 适配器上，
     * **真正的开关动作走 `log.enable`**（不需要重新 Init）；持久化只为了让冷启动后的
     * `Init` 带上 `requestLog`，把用户上次的选择恢复回来。
     */
    fun setRequestLog(enabled: Boolean) {
        // 先改本地状态：开关是即时动作，不能等网络 / JNI 回来再动。
        preferences.requestLogEnabled = enabled
        _state.update { it.copy(requestLogEnabled = enabled, log = null) }
        requestLogJob?.cancel()
        requestLogJob = viewModelScope.launch {
            val result = withContext(Dispatchers.IO) { repo.setRequestLog(enabled) }
            result.fold(
                onSuccess = { log -> _state.update { it.copy(log = log) } },
                onFailure = { ex ->
                    // 回滚本地状态：没真的开起来就不能显示「已开启」。
                    preferences.requestLogEnabled = !enabled
                    _state.update { it.copy(requestLogEnabled = !enabled) }
                    if (!repo.isReady()) return@fold
                    pushError(ex)
                },
            )
        }
    }

    /** 日志页轮询用：读一次列表（0 请求，只读 Go 进程内存）。 */
    fun refreshRequestLog() = loadRequestLog()

    /** 点开某条日志：取完整记录（含原始标头与正文）。 */
    fun openRequestLog(seq: Long) {
        if (requestLogDetailJob?.isActive == true) return
        _state.update { it.copy(logDetail = null, logDetailLoading = true) }
        requestLogDetailJob = viewModelScope.launch {
            val result = withContext(Dispatchers.IO) { repo.requestLogDetail(seq) }
            result.fold(
                onSuccess = { entry ->
                    _state.update { it.copy(logDetail = entry, logDetailLoading = false) }
                },
                onFailure = { ex ->
                    // 条目已被环形窗口丢弃时会回 2001：把列表刷一遍，弹窗不弹错。
                    _state.update { it.copy(logDetailLoading = false) }
                    pushError(ex)
                    loadRequestLog()
                },
            )
        }
    }

    fun closeRequestLog() {
        requestLogDetailJob?.cancel()
        _state.update { it.copy(logDetail = null, logDetailLoading = false) }
    }
    /** 清空记录：只清内容，不改开关。 */
    fun clearRequestLog() {
        if (requestLogDetailJob?.isActive == true) return
        requestLogDetailJob = viewModelScope.launch {
            val result = withContext(Dispatchers.IO) { repo.clearRequestLog() }
            result.fold(
                onSuccess = { log ->
                    _state.update { it.copy(log = log, logDetail = null) }
                },
                onFailure = { ex -> pushError(ex) },
            )
        }
    }

    // ---- 测试（弹窗） -----------------------------------------------------

    private fun probeActive(id: Long, target: ProbeTarget, s: AppState): Boolean {
        val p = s.probe ?: return false
        return p.id == id && p.target == target
    }

    /** 跑一次功能域自检；用同一次请求的结果刷新 UI，不再二次联网。 */
    fun runProbe(target: ProbeTarget) {
        probeJob?.cancel()
        val id = ++probeSeq
        _state.update { it.copy(probe = ProbeState(id = id, target = target, running = true)) }
        probeJob = viewModelScope.launch {
            try {
                if (target == ProbeTarget.CAMPUS && !campusAllowed()) {
                    val skipped = ProbeResult(
                        ok = false,
                        elapsedMs = 0L,
                        code = 0,
                        message = getApplication<Application>()
                            .getString(com.github.echoping07.kemaolife.R.string.probe_skipped_wifi),
                        detail = mapOf("state" to "offline"),
                    )
                    _state.update { s ->
                        if (!probeActive(id, target, s)) return@update s
                        s.copy(probe = ProbeState(id, target, running = false, result = skipped))
                    }
                    applyCampusOfflineSkipped()
                    return@launch
                }

                val result = withContext(Dispatchers.IO) {
                    when (target) {
                        ProbeTarget.TIMETABLE -> repo.testTimetable()
                        ProbeTarget.ELECTRICITY -> repo.testElectricity()
                        ProbeTarget.CAMPUS -> repo.testCampus()
                    }
                }
                result.fold(
                    onSuccess = { probe -> commitProbe(id, target, probe) },
                    onFailure = { ex ->
                        val failure = ProbeResult(
                            ok = false,
                            elapsedMs = 0L,
                            code = (ex as? AppException)?.code ?: 0,
                            message = (ex as? AppException)?.displayMessage ?: "测试失败",
                            detail = emptyMap(),
                        )
                        _state.update { s ->
                            if (!probeActive(id, target, s)) return@update s
                            s.copy(probe = ProbeState(id, target, running = false, result = failure))
                        }
                    },
                )
            } finally {
                _state.update { s ->
                    val p = s.probe
                    if (p != null && p.id == id && p.running) {
                        s.copy(probe = p.copy(running = false))
                    } else {
                        s
                    }
                }
            }
        }
    }

    private suspend fun commitProbe(id: Long, target: ProbeTarget, probe: ProbeResult) {
        if (!probeActive(id, target, _state.value)) return

        if (probe.ok) {
            when (target) {
                ProbeTarget.TIMETABLE -> preferences.lastTimetableSyncAt = preferences.nowElapsed()
                ProbeTarget.ELECTRICITY -> preferences.lastElectricityAt = preferences.nowElapsed()
                ProbeTarget.CAMPUS -> Unit
            }
        }
        _state.update { s ->
            if (!probeActive(id, target, s)) return@update s
            s.copy(probe = ProbeState(id, target, running = false, result = probe))
        }
        if (!probe.ok || !probeActive(id, target, _state.value)) return

        when (target) {
            ProbeTarget.TIMETABLE -> {
                // 测试已同步并写盘：回读缓存（0 请求）把界面更新成同一份数据。
                val desired = _state.value.week?.week ?: 0
                fetchWeek(desired, fromSync = true)
            }
            ProbeTarget.ELECTRICITY -> probe.report?.let { report ->
                withContext(Dispatchers.IO) { cacheElectricityReport(report) }
                _state.update { s ->
                    if (!probeActive(id, target, s)) return@update s
                    s.copy(electricity = report)
                }
            }
            ProbeTarget.CAMPUS -> probe.campusStatus?.let { status ->
                _state.update { s ->
                    if (!probeActive(id, target, s)) return@update s
                    s.copy(campus = status, campusSkipped = false)
                }
            }
        }
    }

    fun closeProbe() {
        probeJob?.cancel()
        probeJob = null
        _state.update { it.copy(probe = null) }
    }

    // ---- 设置 -------------------------------------------------------------

    /**
     * 增量保存凭据：设置页只提交本域字段，由 ViewModel 基于最新 state 合并，
     * 避免三个表单各自捕获旧 `state.credentials` 互相覆盖。Mutex 串行化 Init。
     */
    fun updateCredentials(change: (Credentials) -> Credentials) {
        viewModelScope.launch {
            saveMutex.withLock {
                val previous = _state.value.credentials
                val next = change(previous)
                val merged = withContext(Dispatchers.IO) { store.save(next, previous) }

                // 账号身份变了：清掉旧账号的电费缓存，并让 Go 删掉旧快照（含 Cookie）。
                val identityChanged = merged.studentId != previous.studentId ||
                    merged.eduPassword != previous.eduPassword ||
                    merged.electricityAccount != previous.electricityAccount ||
                    merged.customerCode != previous.customerCode
                if (identityChanged) {
                    preferences.clearElectricityCache()
                    preferences.clearTimetableCache()
                    _state.update {
                        it.copy(
                            electricity = null,
                            week = null,
                            currentWeek = null,
                            lastWeek = 0,
                            todayDay = null,
                            weekStale = false,
                            weekSyncedAt = "",
                            timetableError = null,
                        )
                    }
                }

                val env = withContext(Dispatchers.IO) {
                    repo.init(
                        merged,
                        resetSnapshot = identityChanged,
                        requestLog = preferences.requestLogEnabled,
                    )
                }
                _state.update {
                    it.copy(
                        credentials = merged,
                        bridgeReady = repo.isReady(),
                        bridgeError = repo.readyError(),
                        settingsSaved = env.ok,
                    )
                }
                if (!env.ok) {
                    pushError(AppException(env.error))
                    return@withLock
                }

                // 冷启动 Init 失败时挂起的那一轮打开，在设置页保存成功后补跑。
                replayPendingOpen()
                fetchWeek(_state.value.week?.week ?: 0)
                // 首次配置 / 换账号后立刻取一次，不必再让用户手动点刷新。
                if (identityChanged || _state.value.week == null) maybeAutoSync(force = true)
                if (identityChanged || _state.value.electricity == null) {
                    maybeAutoElectricity(force = true)
                }
            }
        }
    }

    fun setTimetableInterval(minutes: Int) {
        preferences.timetableIntervalMinutes = minutes
        _state.update { it.copy(timetableInterval = preferences.timetableIntervalMinutes) }
    }

    fun setElectricityInterval(minutes: Int) {
        preferences.electricityIntervalMinutes = minutes
        _state.update { it.copy(electricityInterval = preferences.electricityIntervalMinutes) }
    }

    fun setThemeColor(value: String) {
        themePreferences.save(value)
        _state.update { it.copy(themeColor = value) }
    }

    fun setThemeMode(value: String) {
        themePreferences.saveMode(value)
        _state.update { it.copy(themeMode = themePreferences.loadMode()) }
    }

    fun setDynamicColor(value: Boolean) {
        themePreferences.saveDynamicColor(value)
        _state.update { it.copy(dynamicColor = value) }
    }

    fun setAmoled(value: Boolean) {
        themePreferences.saveAmoled(value)
        _state.update { it.copy(amoled = value) }
    }

    fun setContinuousCorners(value: Boolean) {
        themePreferences.saveContinuousCorners(value)
        _state.update { it.copy(continuousCorners = value) }
    }

    fun consumeSettingsSaved() = _state.update { it.copy(settingsSaved = false) }

    fun consumeError() = _state.update { it.copy(error = null) }

    /** 只写 DTO 展示字段；快照里的 Cookie / 凭据一概不进 SharedPreferences。 */
    private fun cacheWeek(schedule: WeekSchedule, syncedAt: String) {
        preferences.timetableCacheJson = TimetableCache.encode(schedule, syncedAt)
    }

    /** 读上一次的电费结果；没有或解析失败都返回 null（不阻断启动）。 */
    private fun cachedElectricityReport(): ElectricityReport? {
        val raw = preferences.electricityCacheJson
        if (raw.isBlank()) return null
        return runCatching { ElectricityReport.from(org.json.JSONObject(raw)) }.getOrNull()
    }

    /** 只写 UI 必需字段：`roomName` 是个人隐私，既不展示也不落盘（P2）。 */
    private fun cacheElectricityReport(report: ElectricityReport) {
        val status = org.json.JSONObject()
            .put("balance", report.status.balance)
            .put("unit", report.status.unit)
            .put("collectedAt", report.status.collectedAt)
            .put("dailyAvg", report.status.dailyAvg ?: org.json.JSONObject.NULL)
            .put("estimatedDays", report.status.estimatedDays ?: org.json.JSONObject.NULL)
        val days = org.json.JSONArray()
        report.usage.days.forEach {
            days.put(org.json.JSONObject().put("date", it.date).put("use", it.use))
        }
        val usage = org.json.JSONObject()
            .put("yearMonth", report.usage.yearMonth)
            .put("monthUse", report.usage.monthUse)
            .put("dataUntil", report.usage.dataUntil)
            .put("days", days)
        preferences.electricityCacheJson = org.json.JSONObject()
            .put("status", status)
            .put("usage", usage)
            .toString()
    }

    private fun applyWeek(
        schedule: WeekSchedule,
        syncedAt: String,
        fromSync: Boolean,
        seq: Long,
        viewLastWeek: Int = 0,
        currentWeek: Int? = null,
    ) {
        _state.update { s ->
            if (seq != weekSeq) return@update s
            // 翻页容器的邻周缓存：当前周一定要在，否则滑回来会闪一下空白。
            weekCache = weekCache + (schedule.week to schedule)
            val keepWeeks = WeekPager.retained(weekCache.keys, schedule.week)
            weekCache = weekCache.filterKeys { it in keepWeeks }
            s.copy(
                week = schedule,
                currentWeek = currentWeek ?: s.currentWeek,
                weekNeighbors = weekCache,
                // 可选 lastWeek：0 = 未知，翻页容器此时不拦截翻页（docs/002 §3.1）。
                lastWeek = if (viewLastWeek > 0) viewLastWeek else s.lastWeek,
                todayDay = schedule.days.firstOrNull { it.date == s.todayDate } ?: s.todayDay,
                // 纯读缓存不碰离线标记；只有「同步 + 回读」这个事务成功才清成 false。
                weekStale = if (fromSync) false else s.weekStale,
                weekSyncedAt = syncedAt,
                weekLoading = false,
                timetableError = null,
            )
        }
    }

    private fun pushError(t: Throwable) {
        // 门面未就绪是启动时序问题，不是用户能处理的操作失败；自动查询已改为
        // 挂起补跑，这一层只是兜底，避免再把「桥接尚未初始化」弹成 snackbar。
        if (t is AppException && t.code == 2001 && !repo.isReady()) return
        val message = (t as? AppException)?.displayMessage ?: "操作失败，请稍后重试"
        _state.update { s ->
            s.copy(error = message, weekLoading = false)
        }
    }
}

enum class CampusAction { LOGIN, LOGOUT, RELOGIN }

/** 设置页「测试」覆盖的功能域。 */
enum class ProbeTarget { TIMETABLE, ELECTRICITY, CAMPUS }

/** 导出弹层的瞬时状态：CSV 正文 + 文件名 + 数据时间；关掉即丢弃。 */
data class ExportState(val export: TimetableExport)

/** 一次测试的瞬时状态（弹窗打开期间有效）；id 用于淘汰过期结果。 */
data class ProbeState(
    val id: Long = 0L,
    val target: ProbeTarget,
    val running: Boolean,
    val result: ProbeResult? = null,
)

data class AppState(
    val bridgeReady: Boolean = false,
    val bridgeError: String? = null,

    val credentials: Credentials = Credentials(),

    val week: WeekSchedule? = null,
    /** 今天所属周；浏览其他周时仍保留，与 week.week 分离。 */
    val currentWeek: Int? = null,
    /** 学期最后一周（`timetable.week` 的可选 `lastWeek`，0 = 未知）：翻页容器的边界。 */
    val lastWeek: Int = 0,
    /** 翻页容器可用的整周数据（周次 → 周数据），含当前周与预取到的邻周。 */
    val weekNeighbors: Map<Int, WeekSchedule> = emptyMap(),
    val weekStale: Boolean = false,
    val weekSyncedAt: String = "",
    val weekLoading: Boolean = false,
    val timetableSyncing: Boolean = false,
    val timetableError: String? = null,

    val todayDate: String = "",
    val todayDay: DaySchedule? = null,

    val electricity: ElectricityReport? = null,
    val electricityLoading: Boolean = false,

    val campus: CampusStatus? = null,
    val campusLoading: Boolean = false,
    val campusActing: CampusAction? = null,
    val campusSkipped: Boolean = false,

    /** 两项自动查询的最小间隔（分钟），来自本地偏好；默认均为 5 分钟、下限 1 分钟。 */
    val timetableInterval: Int = AppPreferences.DEFAULT_TIMETABLE_INTERVAL,
    val electricityInterval: Int = AppPreferences.DEFAULT_ELECTRICITY_INTERVAL,
    val campusWifiOnly: Boolean = true,

    val themeColor: String = "BLUE",
    val themeMode: String = "SYSTEM",
    val dynamicColor: Boolean = DEFAULT_DYNAMIC_COLOR,
    val amoled: Boolean = false,
    val continuousCorners: Boolean = DEFAULT_CONTINUOUS_CORNERS,
    val settingsSaved: Boolean = false,
    val error: String? = null,

    /** 请求记录开关（默认关闭，docs/002 §2.1 规则 10）。 */
    val requestLogEnabled: Boolean = false,
    /** 日志页列表；`null` 表示还没读过（首次进页时加载）。 */
    val log: LogState? = null,
    /** 点开的日志详情；非 null 时展示底部弹层。 */
    val logDetail: LogEntry? = null,
    val logDetailLoading: Boolean = false,

    val probe: ProbeState? = null,

    /** 导出弹层：非 null 时展示文件名 / 课节数 / 数据时间与「保存」。 */
    val exportSheet: ExportState? = null,
    val exportLoading: Boolean = false,
) {
    /** 校园网动作用：offline / 已跳过探测时不允许登录 / 下线 / 重登。 */
    val campusActionsEnabled: Boolean
        get() = campus != null && campus.state != CampusState.OFFLINE && !campusSkipped
}

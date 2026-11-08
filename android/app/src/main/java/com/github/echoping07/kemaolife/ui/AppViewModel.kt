package com.github.echoping07.kemaolife.ui

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.github.echoping07.kemaolife.data.AppException
import com.github.echoping07.kemaolife.data.Bridge
import com.github.echoping07.kemaolife.data.CampusState
import com.github.echoping07.kemaolife.data.CampusStatus
import com.github.echoping07.kemaolife.data.CredentialStore
import com.github.echoping07.kemaolife.data.Credentials
import com.github.echoping07.kemaolife.data.DaySchedule
import com.github.echoping07.kemaolife.data.ElectricityReport
import com.github.echoping07.kemaolife.data.Entry
import com.github.echoping07.kemaolife.data.Repository
import com.github.echoping07.kemaolife.data.ThemePreferences
import com.github.echoping07.kemaolife.data.WeekSchedule
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.time.LocalDate

/**
 * 单例式的应用状态容器：三个 tab 共用一个 ViewModel（`MainActivity` 持有）。
 *
 * 为什么不用多个 ViewModel：三个 tab 共享同一份凭据、同一个门面会话；
 * 拆开就要在它们之间同步状态，收益为负。
 *
 * 课表状态只有一个来源：`week`（整周 7 天）。选中日、今日都在它里面取，
 * 不再单独维护「今日 / 本周 / 选日期」三份数据 —— 那是 UI 难对齐的根源。
 */
class AppViewModel(app: Application) : AndroidViewModel(app) {

    private val store = CredentialStore(app)
    private val bridge = Bridge(app)
    private val repo = Repository(bridge)
    private val themePreferences = ThemePreferences(app)

    private val today: String = LocalDate.now().toString()

    private val _state = MutableStateFlow(AppState(selectedDate = today, todayDate = today))
    val state: StateFlow<AppState> = _state.asStateFlow()

    init {
        // 启动路径：先解密凭据 → Init 门面 → 再并行联网。
        // 这个顺序是 R1 验收项「每次打开先出本地课表」（docs/002 §5.3）：
        // 周查询内部先刷新，刷新失败则自动回落旧快照并标 stale。
        val credentials = store.load()
        _state.update { it.copy(credentials = credentials, themeColor = themePreferences.load()) }
        viewModelScope.launch {
            withContext(Dispatchers.IO) { repo.init(credentials) }
            _state.update { it.copy(bridgeReady = repo.isReady(), bridgeError = repo.readyError()) }
            loadInitial()
        }
    }

    /**
     * 首次加载：① 周课表（内部先刷新）+ ② 一次独立 `timetable.sync` 兜底，两者并行。
     *
     * ② 成功但 ① 失败时只更新 `syncedAt`，不覆写 `stale`（docs/002 §5.3「每次打开」）。
     */
    private fun loadInitial() {
        viewModelScope.launch {
            _state.update { it.copy(weekLoading = true) }

            val weekDeferred = async(Dispatchers.IO) { repo.week(0) }
            val syncDeferred = async(Dispatchers.IO) { repo.sync() }
            val week = weekDeferred.await()
            val synced = syncDeferred.await()

            week.onSuccess { view -> applyWeek(view.schedule, view.stale, view.syncedAt) }
            week.onFailure { ex ->
                _state.update { s ->
                    s.copy(
                        weekLoading = false,
                        needsSetup = s.needsSetup || (ex as? AppException)?.needsCredentials == true,
                        timetableError = (ex as? AppException)?.displayMessage ?: "课表加载失败",
                    )
                }
            }

            synced.onSuccess { result ->
                if (_state.value.week != null) {
                    // stale 由 Go 查询信封给出；这里只补数据时间。
                    _state.update { s -> s.copy(weekSyncedAt = result.syncedAt, timetableError = null) }
                } else {
                    // ① 失败但 ② 刷新成功：快照已经有了，再查一次就能出数据。
                    withContext(Dispatchers.IO) { repo.week(0) }.onSuccess {
                        applyWeek(it.schedule, it.stale, it.syncedAt)
                    }
                }
            }
        }
    }

    /** 切周：`week = 0` 表示当周，由 Go 侧解析（假期返回 7 天 inSemester:false）。 */
    fun loadWeek(week: Int) {
        viewModelScope.launch {
            _state.update { it.copy(weekLoading = true) }
            val result = withContext(Dispatchers.IO) { repo.week(week) }
            result.onSuccess { applyWeek(it.schedule, it.stale, it.syncedAt) }
            result.onFailure { ex ->
                pushError(ex)
                _state.update { s -> s.copy(weekLoading = false) }
            }
        }
    }

    /** 头部刷新按钮：重拉当前这一周（内部先刷新快照）。 */
    fun refreshWeek() {
        val current = _state.value.week?.week ?: 0
        loadWeek(current)
    }

    /**
     * 点周条上的某一天：该天已经在 `week.days` 里，纯本地选中，不发请求。
     */
    fun selectDate(date: String) {
        _state.update { it.copy(selectedDate = date) }
    }

    /**
     * 选日期：先问 Go 侧这一天属于第几周（`timetable.date`），再拉整周。
     *
     * 周次计算属于 `core`（AGENTS A3），Kotlin 不自己拿 `week1Monday` 做除法；
     * 因此这里会多一次查询 —— 选日期是低频动作，正确性比省一个请求重要。
     */
    fun pickDate(date: LocalDate) {
        viewModelScope.launch {
            _state.update { it.copy(weekLoading = true) }
            val day = withContext(Dispatchers.IO) { repo.date(date) }
            day.onFailure { ex ->
                pushError(ex)
                _state.update { s -> s.copy(weekLoading = false) }
            }
            val resolved = day.getOrNull() ?: return@launch
            val weekNo = resolved.day.week

            val week = withContext(Dispatchers.IO) { repo.week(weekNo) }
            week.onSuccess { view ->
                applyWeek(view.schedule, view.stale, view.syncedAt, selectDate = date.toString())
            }
            week.onFailure { ex ->
                pushError(ex)
                _state.update { s -> s.copy(weekLoading = false) }
            }
        }
    }

    /** 电费刷新（tab2 手动动作）。 */
    fun refreshElectricity() {
        viewModelScope.launch {
            _state.update { it.copy(electricityLoading = true) }
            val result = withContext(Dispatchers.IO) { repo.electricity() }
            result.onSuccess { report ->
                _state.update { it.copy(electricity = report, electricityLoading = false) }
            }
            result.onFailure {
                pushError(it)
                _state.update { s -> s.copy(electricityLoading = false) }
            }
        }
    }

    /** 校园网判态。探测失败归 offline 且 `ok:true`，不算错误（docs/002 §3.3）。 */
    fun refreshCampusStatus() {
        viewModelScope.launch {
            _state.update { it.copy(campusLoading = true) }
            val result = withContext(Dispatchers.IO) { repo.campusStatus() }
            result.onSuccess { status ->
                _state.update { it.copy(campus = status, campusLoading = false) }
            }
            result.onFailure {
                pushError(it)
                _state.update { s -> s.copy(campusLoading = false) }
            }
        }
    }

    /** 手动登录 / 下线 / 重登；成功后用返回的最新判态刷新 UI。 */
    fun campusAction(action: CampusAction) {
        viewModelScope.launch {
            _state.update { it.copy(campusActing = action) }
            val result = withContext(Dispatchers.IO) {
                when (action) {
                    CampusAction.LOGIN -> repo.campusLogin()
                    CampusAction.LOGOUT -> repo.campusLogout()
                    CampusAction.RELOGIN -> repo.campusRelogin()
                }
            }
            result.onSuccess { status ->
                _state.update { it.copy(campus = status, campusActing = null) }
            }
            result.onFailure {
                pushError(it)
                // 失败也复探一次：动作可能「受理失败」但也可能已经改变了状态。
                val status = withContext(Dispatchers.IO) { repo.campusStatus() }
                _state.update { s -> s.copy(campus = status.getOrNull() ?: s.campus, campusActing = null) }
            }
        }
    }

    /** 保存设置：写 Keystore 后重新 Init 门面（设置页保存后立即生效）。 */
    fun saveCredentials(next: Credentials) {
        viewModelScope.launch {
            val merged = withContext(Dispatchers.IO) { store.save(next, _state.value.credentials) }
            val env = withContext(Dispatchers.IO) { repo.init(merged) }
            _state.update {
                it.copy(
                    credentials = merged,
                    bridgeReady = repo.isReady(),
                    bridgeError = repo.readyError(),
                    needsSetup = !env.ok,
                    settingsSaved = env.ok,
                )
            }
            if (env.ok) loadInitial() else pushError(AppException(env.error))
        }
    }

    fun setThemeColor(value: String) {
        themePreferences.save(value)
        _state.update { it.copy(themeColor = value) }
    }

    fun consumeSettingsSaved() = _state.update { it.copy(settingsSaved = false) }

    fun consumeError() = _state.update { it.copy(error = null) }

    private fun applyWeek(
        schedule: WeekSchedule,
        stale: Boolean,
        syncedAt: String,
        selectDate: String? = null,
    ) {
        _state.update { s ->
            // 保持选中日落在这一周里：切周后旧选中日可能已经不在本周了。
            val keep = selectDate ?: s.selectedDate
            val inWeek = schedule.days.any { it.date == keep }
            val fallback = schedule.days.firstOrNull { it.date == s.todayDate }?.date
                ?: schedule.days.firstOrNull()?.date
                ?: s.selectedDate
            s.copy(
                week = schedule,
                // 今日概览（「更多」页）不能被切周影响：只有当本次拉到的这一周
                // 恰好含今天时才更新缓存，否则保留上一次的今日数据。
                todayDay = schedule.days.firstOrNull { it.date == s.todayDate } ?: s.todayDay,
                weekStale = stale,
                weekSyncedAt = syncedAt,
                weekLoading = false,
                selectedDate = if (inWeek) keep else fallback,
                // 查到数据就说明凭据可用，顺手清掉「需要配置」与上一次的错误。
                needsSetup = false,
                timetableError = null,
            )
        }
    }

    private fun pushError(t: Throwable) {
        val message = (t as? AppException)?.displayMessage ?: "操作失败，请稍后重试"
        val needsSetup = (t as? AppException)?.needsCredentials == true
        _state.update { s ->
            s.copy(
                error = message,
                needsSetup = needsSetup || s.needsSetup,
                weekLoading = false,
            )
        }
    }
}

enum class CampusAction { LOGIN, LOGOUT, RELOGIN }

/**
 * 单一 UI 状态。
 *
 * 每个字段都能为 null：null 表示「还没查过」，与「查到了但没数据」是不同的语义，
 * 用空字符串或 0 代替会让 UI 无法区分。
 */
data class AppState(
    val bridgeReady: Boolean = false,
    val bridgeError: String? = null,
    val needsSetup: Boolean = false,

    val credentials: Credentials = Credentials(),

    /** 当周（或用户切到的那一周）的 7 天课表 —— 课表 UI 的唯一数据源。 */
    val week: WeekSchedule? = null,
    val weekStale: Boolean = false,
    val weekSyncedAt: String = "",
    val weekLoading: Boolean = false,
    /** 课表「连旧快照都没有」时的原因；有数据时为 null。 */
    val timetableError: String? = null,

    /** 选中日与今天，都是 `YYYY-MM-DD`；选中日必须落在 [week] 里（由 ViewModel 保证）。 */
    val selectedDate: String = "",
    val todayDate: String = "",
    /**
     * 今天那一天的课表（「更多」页今日概览用）。
     *
     * 单独缓存而不是从 [week] 里取：用户可能把课表切到别的周，
     * 那时「今日概览」仍应显示今天，而不是变成空白。
     */
    val todayDay: DaySchedule? = null,

    val electricity: ElectricityReport? = null,
    val electricityLoading: Boolean = false,

    val campus: CampusStatus? = null,
    val campusLoading: Boolean = false,
    val campusActing: CampusAction? = null,

    val themeColor: String = "BLUE",
    val settingsSaved: Boolean = false,
    val error: String? = null,
) {
    /** 校园网动作用：offline 时不允许登录 / 下线 / 重登（docs/002 §3.3）。 */
    val campusActionsEnabled: Boolean
        get() = campus != null && campus.state != CampusState.OFFLINE

    /** 周里与 [selectedDate] 对应的那天；切周后若选中日不在本周则为 null。 */
    val selectedDay: DaySchedule?
        get() = week?.days?.firstOrNull { it.date == selectedDate }

    val todayEntries: List<Entry> get() = todayDay?.entries ?: emptyList()
}

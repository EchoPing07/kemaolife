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
import com.github.echoping07.kemaolife.data.Repository
import com.github.echoping07.kemaolife.data.ScheduleView
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
 */
class AppViewModel(app: Application) : AndroidViewModel(app) {

    private val store = CredentialStore(app)
    private val bridge = Bridge(app)
    private val repo = Repository(bridge)

    private val _state = MutableStateFlow(AppState())
    val state: StateFlow<AppState> = _state.asStateFlow()

    init {
        // 启动路径：先解密凭据 → Init 门面 → 立刻用本地快照渲染 → 再并行联网。
        // 这个顺序是 R1 验收项「每次打开先出本地课表」（docs/002 §5.3）。
        val credentials = store.load()
        _state.update { it.copy(credentials = credentials) }
        viewModelScope.launch {
            withContext(Dispatchers.IO) { repo.init(credentials) }
            _state.update { it.copy(bridgeReady = repo.isReady(), bridgeError = repo.readyError()) }
            loadTodayAndSync()
        }
    }

    /** 今日课表 + 并行同步（两个独立动作，见 docs/002 §5.3「每次打开」）。 */
    private fun loadTodayAndSync() {
        viewModelScope.launch {
            _state.update { it.copy(timetableLoading = true) }

            // ① / ② 并行发起：查询内部先刷新，独立同步用于兜底，二者互不阻塞。
            val todayDeferred = async(Dispatchers.IO) { repo.today() }
            val syncDeferred = async(Dispatchers.IO) { repo.sync() }
            val today = todayDeferred.await()
            val synced = syncDeferred.await()

            // 只有「刷新失败 + 连旧快照都没有」才会走到 onFailure，此时必须告知用户，
            // 不能静默留白（否则界面会永远停在空白，用户不知道是没课还是没配置）。
            today.onSuccess { applyToday(it) }
            if (today.isFailure) {
                val ex = today.exceptionOrNull()
                _state.update { s ->
                    s.copy(
                        timetableLoading = false,
                        needsSetup = s.needsSetup || (ex as? AppException)?.needsCredentials == true,
                        timetableError = (ex as? AppException)?.displayMessage ?: "课表加载失败",
                    )
                }
            }

            synced.onSuccess { result ->
                if (_state.value.today != null) {
                    // stale 由 Go 查询信封给出；UI 只更新数据时间，不覆写 stale。
                    _state.update { s ->
                        s.copy(
                            today = s.today?.copy(syncedAt = result.syncedAt),
                            timetableError = null,
                        )
                    }
                } else {
                    // ① 失败但 ② 刷新成功：快照已经有了，再查一次就能出数据。
                    withContext(Dispatchers.IO) { repo.today() }.onSuccess { applyToday(it) }
                }
            }
        }
    }

    /** 手动刷新课表（tab1 的刷新按钮）。 */
    fun refreshTimetable() {
        viewModelScope.launch {
            _state.update { it.copy(refreshing = true) }
            val result = withContext(Dispatchers.IO) { repo.today() }
            result.onSuccess { applyToday(it) }
            result.onFailure { pushError(it) }
            _state.update { it.copy(refreshing = false) }
        }
    }

    /** 选日期：`date` 查询，同样先在 Go 侧刷新再计算。 */
    fun loadDate(date: LocalDate) {
        viewModelScope.launch {
            _state.update { it.copy(timetableLoading = true) }
            val result = withContext(Dispatchers.IO) { repo.date(date) }
            result.onSuccess { applyToday(it) }
            result.onFailure { pushError(it) }
            _state.update { it.copy(timetableLoading = false) }
        }
    }

    /** 切周：`week = 0` 表示当周，由 Go 侧解析（假期时返回 7 天 inSemester:false）。 */
    fun loadWeek(week: Int) {
        viewModelScope.launch {
            _state.update { it.copy(weekLoading = true) }
            val result = withContext(Dispatchers.IO) { repo.week(week) }
            result.onSuccess { view ->
                _state.update {
                    it.copy(
                        week = view.schedule,
                        weekStale = view.stale,
                        weekSyncedAt = view.syncedAt,
                        weekLoading = false,
                    )
                }
            }
            result.onFailure {
                pushError(it)
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
            if (env.ok) loadTodayAndSync() else pushError(AppException(env.error))
        }
    }

    fun consumeSettingsSaved() = _state.update { it.copy(settingsSaved = false) }

    fun consumeError() = _state.update { it.copy(error = null) }

    private fun applyToday(view: ScheduleView) {
        _state.update {
            it.copy(
                today = view,
                timetableLoading = false,
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
                timetableLoading = false,
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

    val today: ScheduleView? = null,
    val timetableLoading: Boolean = false,
    val refreshing: Boolean = false,
    /** 课表「连旧快照都没有」时的原因；有数据时为 null。 */
    val timetableError: String? = null,

    val week: WeekSchedule? = null,
    val weekStale: Boolean = false,
    val weekSyncedAt: String = "",
    val weekLoading: Boolean = false,

    val electricity: ElectricityReport? = null,
    val electricityLoading: Boolean = false,

    val campus: CampusStatus? = null,
    val campusLoading: Boolean = false,
    val campusActing: CampusAction? = null,

    val settingsSaved: Boolean = false,
    val error: String? = null,
) {
    /** 校园网动作用：offline 时不允许登录 / 下线 / 重登（docs/002 §3.3）。 */
    val campusActionsEnabled: Boolean
        get() = campus != null && campus.state != CampusState.OFFLINE

    val todayEntries: List<com.github.echoping07.kemaolife.data.Entry>
        get() = today?.day?.entries ?: emptyList()

    val todayDay: DaySchedule? get() = today?.day
}

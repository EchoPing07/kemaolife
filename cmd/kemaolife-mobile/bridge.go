// Package kemaolife 是 Android 侧的唯一门面：只导出两个字符串入口
// Init / Call（docs/002 §2.1、001 §6.1）。
//
// 为什么只有字符串：gobind 对导出类型限制很严（结构体切片、跨 JNI 接口都不行），
// 所以领域对象一律不跨边界，Go 内部保持强类型，只在边界付一次 JSON 编解码。
//
// 构建（TODO(D13)：工具链待定稿）：
//
//	gomobile bind -target=android -androidapi 24 \
//	    -javapkg com.github.echoping07.kemaolife.bridge \
//	    -o android/app/libs/kemaolife.aar ./cmd/kemaolife-mobile
package kemaolife

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	// Android 的 tzdata 在 /system/usr/share/zoneinfo，不在 Go 的默认查找路径里，
	// 不内嵌这份数据的话 time.Local 会退化成 UTC，所有时间戳都会差 8 小时
	// （「数据截至 22:48」显示成「14:48Z」）。内嵌约 450KB，换来时区确定可用。
	_ "time/tzdata"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
	"github.com/EchoPing07/kemaolife/internal/platform"
	buildversion "github.com/EchoPing07/kemaolife/internal/version"
)

// defaultCustomerCode 是学校代码内置默认值（学校事实，非个人隐私）。
//
// 按 AGENTS §5.6 放在 cmd 层而不是 internal/core，用户可在设置页覆盖。
const defaultCustomerCode = "3126"

// defaultTimeout 是单次网络请求上限；课表同步含 CAS 登录 + 2 个请求，留足余量。
const defaultTimeout = 30 * time.Second

// initConfig 是 Kotlin 通过 Init 传入的全部信息（凭据已由 Keystore 解密）。
//
// 凭据只存在于内存，Go 侧不落盘、不进日志（P5 / P2）。
type initConfig struct {
	// CacheDir 是应用私有目录下的可写目录，用于存快照（含 Cookie，等同凭据）。
	CacheDir string `json:"cacheDir"`

	// 教务：学号 + 密码（课表）。
	StudentID   string `json:"studentId"`
	EduPassword string `json:"eduPassword"`

	// 电费：学校代码 + 可选独立学号，无密码（api-docs/002 §1）。
	CustomerCode string `json:"customerCode"`
	// ElectricityStudentID 为空表示继承教务学号（设置页默认打开该开关）。
	ElectricityStudentID string `json:"electricityStudentId"`

	// 校园网：独立凭据，不复用教务密码（D9 已定稿）。
	CampusUser        string `json:"campusUser"`
	CampusPassword    string `json:"campusPassword"`
	CampusFingerprint string `json:"campusFingerprint"`

	// TimeZone 是设备时区（Kotlin 的 TimeZone.getDefault().id，如 Asia/Shanghai）。
	// TimeZoneOffsetSeconds 是同一时刻的偏移秒数，作为 LoadLocation 失败时的兜底。
	//
	// 为什么必须由 Kotlin 传：Go 在 Android 上找不到 zoneinfo 目录，
	// time.Local 会退化成 UTC；而 core / app 的时间口径全部基于 time.Local（AGENTS §6）。
	TimeZone              string `json:"timeZone"`
	TimeZoneOffsetSeconds int    `json:"timeZoneOffsetSeconds"`

	// ResetSnapshot 为 true 时先删除本地课表快照再替换会话：凭据身份变更时必须
	// 防止旧账号的课表先显示出来（docs/002 §2.1 规则 2）。缺省 false，老客户端不受影响。
	ResetSnapshot bool `json:"resetSnapshot"`
}

// session 是一次 Init 组装好的依赖；配置随 Init 更新，避免每次调用重复解析。
//
// ctx / cancel 绑定会话代次：设置页保存凭据后重新 Init 会取消上一代次的在途请求，
// 避免旧账号的响应覆盖新会话写出的快照（docs/002 §2.1 规则 9）。
type session struct {
	deps   app.Deps
	campus app.CampusDeps
	cfg    model.Config
	clock  platform.RealClock

	ctx    context.Context
	cancel context.CancelFunc
	gen    uint64
}

var (
	mu         sync.Mutex
	cur        *session
	generation uint64
)

// timeZoneOnce 保证 time.Local 只写一次。
//
// time.Local 是进程级全局变量，core 里的 model.NewDate / ParseInLocation 都以它为准；
// 保存设置时会再 Init 一次，若每次都写，就会与 IO 线程上正在格式化的 Call 形成数据竞争
// （go test -race 能抓到，但 CI 未开）。时区在进程启动后按设备值固定一次是安全的折衷：
// 设备时区变更由系统重建进程，App 内不动态跟随。
var timeZoneOnce sync.Once

// Init is called by Kotlin once at startup and again after the settings page
// saves (docs/002 §2.1). It returns the §2 envelope as JSON.
//
// gobind copies exported doc comments into generated Java sources, and javac
// reads them with the platform default charset (GBK on the build machine).
// Keep every comment on an exported symbol ASCII-only, otherwise the AAR build
// fails on non-ASCII bytes. Chinese rationale stays on the unexported helpers.
//
// A failed Init keeps the previous session alive: one bad call must not brick
// a running app. TODO(D12): envelope unification still pending user decision.
func Init(configJSON string) string {
	var cfg initConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return failure(model.WrapError(model.CodeInvalidArgument, "初始化参数无法解析", err))
	}
	cacheDir := strings.TrimSpace(cfg.CacheDir)
	if cacheDir == "" {
		return failure(model.NewError(model.CodeInvalidArgument, "初始化缺少可写目录"))
	}

	store, err := platform.NewFileStore(cacheDir)
	if err != nil {
		return failure(model.WrapError(model.CodeInvalidArgument, "无法使用缓存目录", err))
	}

	// 时区校准放在目录校验之后：失败的 Init 不允许动全局 time.Local。
	applyTimeZone(cfg.TimeZone, cfg.TimeZoneOffsetSeconds)

	clock := platform.RealClock{}
	doer := platform.NewHTTPDoer(defaultTimeout)

	customerCode := strings.TrimSpace(cfg.CustomerCode)
	if customerCode == "" {
		customerCode = defaultCustomerCode
	}

	// 校园网凭据每轮现读：设置页保存后重新 Init 即生效，不需要重启 App。
	cred := cfg
	portal := platform.NewPortalNetwork(func() platform.PortalCredential {
		return platform.PortalCredential{
			UserID:      cred.CampusUser,
			Password:    cred.CampusPassword,
			Fingerprint: cred.CampusFingerprint,
		}
	}, "")

	s := &session{
		deps: app.Deps{
			Clock:               clock,
			HTTP:                doer,
			DefaultCustomerCode: customerCode,
		},
		campus: app.CampusDeps{Clock: clock, Net: portal, Ready: portal},
		cfg: model.Config{
			StudentID:            strings.TrimSpace(cfg.StudentID),
			EduPassword:          cfg.EduPassword,
			CustomerCode:         customerCode,
			ElectricityStudentID: strings.TrimSpace(cfg.ElectricityStudentID),
		},
		clock: clock,
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())

	var prevCancel context.CancelFunc
	mu.Lock()
	generation++
	s.gen = generation
	// Store 必须在发布 cur 之前装好：另一个线程拿到新 session 时不能看到 nil Store。
	// 会话代次守卫让被取消的旧同步即使已拿到响应，也不能再写快照。
	s.deps.Store = generationStore{inner: store, gen: s.gen}
	if cur != nil {
		prevCancel = cur.cancel
	}
	cur = s
	mu.Unlock()

	// 新一代次发布后再取消旧请求；旧 Call 会拿到 context.Canceled 并收敛成失败信封。
	if prevCancel != nil {
		prevCancel()
	}

	// 旧请求已被取消 / 写入已被拦住，这时删旧快照不会留下旧账号数据。
	if cfg.ResetSnapshot {
		if err := s.deps.Store.Delete(timetable.SnapshotStoreKey); err != nil {
			return failure(model.WrapError(model.CodeNetworkDown, "清理旧快照失败", err))
		}
	}

	return success(clock.Now(), map[string]any{
		"version": buildversion.Version,
	})
}

// generationStore 让旧会话的写盘失效，而不影响纯读缓存。
//
// 为什么需要它：context 取消只能拦住还在等网络的请求；已经拿到响应的旧同步
// 仍会走到 SaveSnapshot。凭据变更后再 Init 时，旧会话写进来的 Cookie / 课表
// 会污染新账号的本地状态（docs/002 §2.1 规则 9）。
type generationStore struct {
	inner core.KVStore
	gen   uint64
}

func (g generationStore) Get(key string) ([]byte, error) { return g.inner.Get(key) }

func (g generationStore) Put(key string, v []byte) error {
	if !isCurrentGeneration(g.gen) {
		return model.NewError(model.CodeNetworkDown, "会话已更新，写入已取消")
	}
	return g.inner.Put(key, v)
}

func (g generationStore) Delete(key string) error {
	if !isCurrentGeneration(g.gen) {
		return model.NewError(model.CodeNetworkDown, "会话已更新，删除已取消")
	}
	return g.inner.Delete(key)
}

func isCurrentGeneration(gen uint64) bool {
	mu.Lock()
	defer mu.Unlock()
	return cur != nil && cur.gen == gen
}

// callRequest 是门面请求（docs/002 §2.1）。op 之外的字段按 op 取用。
type callRequest struct {
	Op    string `json:"op"`
	Week  int    `json:"week"`
	Date  string `json:"date"`
	Month string `json:"month"`
}

// Call takes a request JSON and returns a response JSON. Unknown op yields
// error code 2001 instead of an empty success (docs/002 §2.1 rule 5).
//
// ASCII-only comment on purpose: see Init.
func Call(reqJSON string) string {
	var req callRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		return failure(model.WrapError(model.CodeInvalidArgument, "请求无法解析", err))
	}

	s := currentSession()
	if s == nil {
		return failure(model.NewError(model.CodeInvalidArgument, "门面尚未初始化"))
	}
	op := strings.TrimSpace(req.Op)

	switch op {
	case "ping":
		return success(s.clock.Now(), map[string]any{
			"pong":    true,
			"version": buildversion.Version,
		})
	case "timetable.sync":
		return s.timetableSync()
	case "timetable.today":
		return s.timetableDate(model.NewDate(s.clock.Now()))
	case "timetable.date":
		date, err := model.ParseDate(strings.TrimSpace(req.Date))
		if err != nil {
			return failure(model.WrapError(model.CodeInvalidArgument, "日期格式应为 YYYY-MM-DD", err))
		}
		return s.timetableDate(date)
	case "timetable.week":
		return s.timetableWeek(req.Week)
	case "timetable.probe":
		// 设置页测试按钮：同步一次并回报耗时与课时数（docs/002 §5.3）。
		return s.timetableProbe()
	case "electricity.report":
		return s.electricityReport(req.Month)
	case "electricity.probe":
		return s.electricityProbe(req.Month)
	case "campusnet.status":
		return s.campusStatus()
	case "campusnet.probe":
		return s.campusProbe()
	case "campusnet.login":
		return s.campusAction(app.LoginCampusNetwork)
	case "campusnet.logout":
		return s.campusAction(app.LogoutCampusNetwork)
	case "campusnet.relogin":
		return s.campusAction(app.ReloginCampusNetwork)
	}
	return failure(model.NewError(model.CodeInvalidArgument, "未知 op"))
}

func currentSession() *session {
	mu.Lock()
	defer mu.Unlock()
	return cur
}

// timetableSync 对应 CLI 的 `timetable sync`：只刷新、不查询。
//
// data.syncedAt 由用例给出；信封层 syncedAt 也必须给出，保持与 CLI 同构（docs/002 §2）。
func (s *session) timetableSync() string {
	result, err := app.SyncTimetable(s.ctx, s.deps, s.cfg)
	if err != nil {
		return failure(err)
	}
	env := model.Success(s.clock.Now(), result)
	env.SyncedAt = result.SyncedAt
	return encode(env)
}

// timetableDate 是 `timetable date` 的门面口径：**只读缓存**。
//
// 为什么与 CLI 不同：CLI 的查询动作自带刷新（docs/002 §1.3），而 Android 的
// 刷新是 UI 另起的独立动作 `timetable.sync`（docs/002 §5.3）。查询直接读缓存，
// 切周、选日期、每次打开都不再联网，联网失败也照常出旧课表。
//
// 纯读缓存没有「刷新失败」这回事，所以 `stale` 恒为 false；
// 数据时间仍由信封的 `syncedAt` 给出，UI 据此显示「同步于」。
func (s *session) timetableDate(date model.Date) string {
	result, err := app.GetSchedule(s.ctx, s.deps, s.cfg, date, app.CacheOnly)
	if err != nil {
		return failure(err)
	}
	return envelopeWithStale(s.clock.Now(), result.Day, result.Stale, result.SyncedAt)
}

// timetableWeek 查询指定周；week == 0 表示当周，负数也是显式周次（学期前的周）。
func (s *session) timetableWeek(week int) string {
	result, err := app.GetWeekSchedule(s.ctx, s.deps, s.cfg, week, app.CacheOnly)
	if err != nil {
		return failure(err)
	}
	data := map[string]any{"week": result.Week, "days": result.Days}
	// lastWeek 是可选字段（0 = 未知）：Android 翻页容器用它确定翻页边界，
	// 老客户端忽略即可（docs/002 §3.1）。
	if result.LastWeek > 0 {
		data["lastWeek"] = result.LastWeek
	}
	return envelopeWithStale(s.clock.Now(), data, result.Stale, result.SyncedAt)
}

// electricityReport 对应 `kemaolife electricity`（ElectricityReport）。
func (s *session) electricityReport(month string) string {
	report, err := app.CheckElectricity(s.ctx, s.deps, s.cfg, month)
	if err != nil {
		return failure(err)
	}
	return success(s.clock.Now(), report)
}

// campusStatus 判态；offline 是正常状态，不是错误（docs/002 §3.3）。
func (s *session) campusStatus() string {
	status, err := app.GetCampusNetworkStatus(s.ctx, s.campus)
	if err != nil {
		return failure(err)
	}
	return success(s.clock.Now(), status)
}

// campusAction 执行手动登录 / 下线 / 重登，成功后回带最新判态。
//
// 回带判态的理由：动作本身可能「受理成功但未生效」，UI 需要立刻看到真实状态。
func (s *session) campusAction(action func(ctx context.Context, d app.CampusDeps) error) string {
	if err := action(s.ctx, s.campus); err != nil {
		return failure(err)
	}
	status, err := app.GetCampusNetworkStatus(s.ctx, s.campus)
	if err != nil {
		return failure(err)
	}
	return success(s.clock.Now(), status)
}

// probeResult 是「测试」按钮的统一回报（docs/002 §5.3 tab1 / tab2）。
//
// 测试动作一定会有个结果，所以这里刻意让信封 `ok:true`、成败放在 `data.ok`：
// 失败时也要能看到耗时与脱敏后的原因，才能在设置页判断是网络慢还是凭据错。
//
// Report / CampusStatus 是可选字段：让 UI 用**同一次请求**的结果刷新卡片，
// 不再二次联网；老客户端忽略即可（docs/002 §2.1 规则 8）。
type probeResult struct {
	OK           bool                     `json:"ok"`
	ElapsedMs    int64                    `json:"elapsedMs"`
	Code         model.ErrorCode          `json:"code,omitempty"`
	Message      string                   `json:"message"`
	Detail       map[string]any           `json:"detail,omitempty"`
	Report       *model.ElectricityReport `json:"report,omitempty"`
	CampusStatus *model.CampusStatus      `json:"campusStatus,omitempty"`
}

// probeOutcome 是一次测试动作的可展示产出。
type probeOutcome struct {
	Detail       map[string]any
	Report       *model.ElectricityReport
	CampusStatus *model.CampusStatus
}

// timed 跑一次测试动作，记录耗时并把错误收敛成 probeResult。
//
// 信封时间走注入的 Clock：与其它 op 同一口径，不在这里再读一次系统时钟。
// 失败 message 一律按错误码映射固定中文文案，绝不透传上游原文 / 用户输入（P2）。
func (s *session) timed(run func() (probeOutcome, error)) string {
	start := s.clock.Now()
	outcome, err := run()
	out := probeResult{OK: err == nil, ElapsedMs: s.clock.Now().Sub(start).Milliseconds()}
	if err != nil {
		out.Code = model.CodeOf(err)
		out.Message = model.CodeMessage(out.Code)
	} else {
		out.Detail = outcome.Detail
		out.Report = outcome.Report
		out.CampusStatus = outcome.CampusStatus
	}
	return success(s.clock.Now(), out)
}

// timetableProbe 同步一次教务并回报耗时与快照摘要。
func (s *session) timetableProbe() string {
	return s.timed(func() (probeOutcome, error) {
		result, err := app.SyncTimetable(s.ctx, s.deps, s.cfg)
		if err != nil {
			return probeOutcome{}, err
		}
		return probeOutcome{Detail: map[string]any{
			"syncedAt":    result.SyncedAt,
			"week1Monday": result.Week1Monday,
			"lastWeek":    result.LastWeek,
			"lessonCount": result.LessonCount,
		}}, nil
	})
}

// electricityProbe 查询一次电费并回报耗时与摘要（不含 roomName / roomVerify，P2）。
func (s *session) electricityProbe(month string) string {
	return s.timed(func() (probeOutcome, error) {
		report, err := app.CheckElectricity(s.ctx, s.deps, s.cfg, month)
		if err != nil {
			return probeOutcome{}, err
		}
		// 缓存用的副本不带宿舍标识：roomName 是个人隐私，UI 也不需要它。
		report.Status.RoomName = ""
		report.Status.RoomVerify = "已脱敏"
		detail := map[string]any{
			"balance":     report.Status.Balance,
			"unit":        report.Status.Unit,
			"collectedAt": report.Status.CollectedAt,
			"monthUse":    report.Usage.MonthUse,
			"dataUntil":   report.Usage.DataUntil,
			"dayCount":    len(report.Usage.Days),
		}
		if report.Status.EstimatedDays != nil {
			detail["estimatedDays"] = *report.Status.EstimatedDays
		} else {
			detail["estimatedDays"] = "" // 让 UI 显示「—」，不要序列化成字面量 null
		}
		return probeOutcome{Detail: detail, Report: &report}, nil
	})
}

// campusProbe 探测一次校园网判态并回报耗时与三态。
func (s *session) campusProbe() string {
	return s.timed(func() (probeOutcome, error) {
		status, err := app.GetCampusNetworkStatus(s.ctx, s.campus)
		if err != nil {
			return probeOutcome{}, err
		}
		return probeOutcome{
			Detail: map[string]any{
				"state":     status.State,
				"checkedAt": status.CheckedAt,
			},
			CampusStatus: &status,
		}, nil
	})
}

// envelopeWithStale 组装带 stale / syncedAt 的课表信封（docs/002 §2）。
func envelopeWithStale(now time.Time, data any, stale bool, syncedAt time.Time) string {
	env := model.Success(now, data)
	env.Stale = model.BoolPtr(stale)
	env.SyncedAt = formatRFC3339(syncedAt)
	return encode(env)
}

func success(now time.Time, data any) string { return encode(model.Success(now, data)) }

func failure(err error) string {
	// 时间用固定语义即可：失败信封的 ts 只用于展示，不必读时钟端口。
	return encode(model.Failure(time.Now(), sanitize(err)))
}

// sanitize 保证进信封的文案不带上游原文（P2：上游文案脱敏后才可输出）。
func sanitize(err error) error {
	return model.NewError(model.CodeOf(err), model.MessageOf(err))
}

func encode(env model.Envelope) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		// 编码失败是内部错误：返回一个手工构造的最小失败信封，绝不返回空串。
		return `{"ok":false,"error":{"code":2001,"message":"响应编码失败"}}`
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatRFC3339 与 app 层同口径：本地时区，零值输出空串。
func formatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format(time.RFC3339)
}

// applyTimeZone 把进程的本地时区设定为设备时区，且只在进程内生效一次。
//
// 优先级：按名字加载成功 > 按偏移构造固定时区。前者能自动处理夏令时，
// 后者退化为固定偏移（中国大陆无夏令时，实际等价）。
//
// 只做一次是为了避免与在途 Call 并发写 time.Local（见 timeZoneOnce 注释）；
// 失败 Init 不会走到这里（目录校验在前）。
func applyTimeZone(name string, offsetSeconds int) {
	timeZoneOnce.Do(func() { applyTimeZoneOnce(name, offsetSeconds) })
}

func applyTimeZoneOnce(name string, offsetSeconds int) {
	name = strings.TrimSpace(name)
	if name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			time.Local = loc
			return
		}
	}
	if offsetSeconds != 0 {
		zone := name
		if zone == "" {
			zone = "Local"
		}
		time.Local = time.FixedZone(zone, offsetSeconds)
	}
}

// resetTimeZoneForTest 只给本包单测重置 once，生产代码不会调用。
// 单测之间串行运行，重置后立刻由被测路径取值，不构成生产侧的数据竞争。
func resetTimeZoneForTest() { timeZoneOnce = sync.Once{} }

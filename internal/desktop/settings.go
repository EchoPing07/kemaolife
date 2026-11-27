package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/alert"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// ConfigStoreKey 是 Windows 侧配置在 KVStore 中的键。
const ConfigStoreKey = "desktop/config"

// ErrSettingsClamped 表示设置已保存并生效，但有间隔被下限钳制。
// 它不是失败：UI 要显示提示，不能按「保存失败」把窗口留在错误态。
var ErrSettingsClamped = errors.New("设置间隔已钳制")

// Settings 是 Windows 端的全部可配置项（docs/windows.md）。
//
// 与 CLI 版的 model.Config 分开：Windows 额外需要校园网独立凭据
// 与轮询 / 告警参数；教务密码是可选的新增字段，老配置可继续读取。
// 落盘前由 DPAPI 加密（P5）。
type Settings struct {
	// 电费：学号 + 学校代码（电费接口无密码，api-docs/contract.md §1）。
	StudentID    string `json:"studentId"`
	CustomerCode string `json:"customerCode,omitempty"`

	// 教务：可选；留空时仍能使用电费与校园网功能。
	EduPassword string `json:"eduPassword,omitempty"`

	// 校园网：独立凭据，不复用教务 / 电费账号。
	CampusUser        string `json:"campusUser"`
	CampusPassword    string `json:"campusPassword"`
	CampusFingerprint string `json:"campusFingerprint"`

	// 模式：默认手动（docs/windows.md）。
	AutoMode bool `json:"autoMode"`

	// CampusWiredOnly 限制自动认证只在有线出口发起（docs/windows.md §5.1）。
	// 宿舍路由器已认证时，从 Wi-Fi 出口登录个人账号会顶掉路由器账号。
	// 默认 false 不限制；老配置无此字段，升级后行为不变。
	CampusWiredOnly bool `json:"campusWiredOnly,omitempty"`

	// AutoStart 与轮询 / 告警参数。
	AutoStart        bool    `json:"autoStart"`
	PollSeconds      int     `json:"pollSeconds"`
	CampusProbeSecs  int     `json:"campusProbeSeconds,omitempty"`
	BalanceThreshold float64 `json:"balanceThreshold"`
	DaysThreshold    int     `json:"daysThreshold"`
	QuietStartHour   int     `json:"quietStartHour"`
	QuietEndHour     int     `json:"quietEndHour"`

	// AutoUpdateCheck 开启后每次启动后台检查一次新版本（docs/windows.md §3.5）。
	// 默认 false；老配置无此字段，升级后行为不变。
	AutoUpdateCheck bool `json:"autoUpdateCheck,omitempty"`

	// ThemeMode 控制主窗口外观主题：system（跟随系统，默认）/ light / dark。
	// 可选字段，老配置缺省按 system 处理（docs/windows.md §3.5）。
	ThemeMode string `json:"themeMode,omitempty"`

	// 课表显示选项（docs/windows.md §3.2 / §3.5）。
	// 不加 omitempty：前两项默认 true，关闭时必须显式写入 false 才能覆盖
	// Load() 的默认值；老配置缺字段时反序列化自动保持默认值。
	OptimizeLocationDisplay bool `json:"optimizeLocationDisplay"`
	PrefixLocationAt        bool `json:"prefixLocationAt"`
	HideTimetableOverflow   bool `json:"hideTimetableOverflow"`
}

// 主窗口外观主题取值（docs/windows.md §3.5）。
const (
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

// NormalizeThemeMode 将外部输入的主题模式归一化为 system / light / dark。
func NormalizeThemeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ThemeLight:
		return ThemeLight
	case ThemeDark:
		return ThemeDark
	default:
		return ThemeSystem
	}
}

// 校园网状态探测间隔的默认值与下限（docs/windows.md）。
//
// 下限取 30s 而不是 0：每次探测都是一次真实 HTTP 请求，间隔过小等于
// 持续打 Portal，既没必要也容易被判异常。与电费轮询的 600s 硬下限无关 ——
// 那一条约束的是电费接口（AGENTS §5.5）。
const (
	DefaultCampusProbeSeconds = 120
	CampusProbeFloorSeconds   = 30
)

// DefaultSettings 返回首次启动的默认值（轮询 4 小时、阈值 10 度 / 3 天、静默 23–07）。
func DefaultSettings() Settings {
	return Settings{
		AutoMode:                false, // 默认手动
		AutoStart:               true,  // 登录时自启一次
		AutoUpdateCheck:         false, // 默认不自动检查更新
		ThemeMode:               ThemeSystem,
		OptimizeLocationDisplay: true,  // 默认开启课室显示优化
		PrefixLocationAt:        true,  // 默认课室前显示 @
		HideTimetableOverflow:   false, // 默认完整换行，不截断
		PollSeconds:             int(alert.DefaultPollInterval / time.Second),
		CampusProbeSecs:         DefaultCampusProbeSeconds,
		BalanceThreshold:        alert.DefaultBalanceThreshold,
		DaysThreshold:           alert.DefaultDaysThreshold,
		QuietStartHour:          alert.DefaultQuietStartHour,
		QuietEndHour:            alert.DefaultQuietEndHour,
	}
}

// normalize 补齐 / 钳制非法值，返回是否发生了钳制（供 UI 提示）。
//
// 轮询间隔低于 600s 一律钳制（AGENTS §5.5：不可放行）。
func (s *Settings) normalize() (clamped bool) {
	def := DefaultSettings()
	s.ThemeMode = NormalizeThemeMode(s.ThemeMode)
	if s.PollSeconds <= 0 {
		s.PollSeconds = def.PollSeconds
	}
	if d, c := alert.NormalizePollInterval(time.Duration(s.PollSeconds) * time.Second); c {
		s.PollSeconds = int(d / time.Second)
		clamped = true
	}
	if d, c := NormalizeCampusProbeInterval(secondsDuration(s.CampusProbeSecs)); c {
		s.CampusProbeSecs = int(d / time.Second)
		clamped = true
	}
	if s.BalanceThreshold <= 0 {
		s.BalanceThreshold = def.BalanceThreshold
	}
	if s.DaysThreshold <= 0 {
		s.DaysThreshold = def.DaysThreshold
	}
	// 静默时段允许 0–23；越界按默认。
	if s.QuietStartHour < 0 || s.QuietStartHour > 23 {
		s.QuietStartHour = def.QuietStartHour
	}
	if s.QuietEndHour < 0 || s.QuietEndHour > 23 {
		s.QuietEndHour = def.QuietEndHour
	}
	return clamped
}

// PollInterval 返回已钳制的轮询间隔。
func (s Settings) PollInterval() time.Duration {
	d, _ := alert.NormalizePollInterval(time.Duration(s.PollSeconds) * time.Second)
	return d
}

// CampusProbeInterval 返回已钳制的校园网探测间隔。
func (s Settings) CampusProbeInterval() time.Duration {
	d, _ := NormalizeCampusProbeInterval(secondsDuration(s.CampusProbeSecs))
	return d
}

// NormalizeCampusProbeInterval 把探测间隔钳制进 [30s, ∞)，零值取默认 120s。
//
// 零值取默认而不是报错：老配置里没有这个字段（omitempty），
// 升级后必须能继续用默认 120s 跑起来（契约纪律：新增字段必须可选，见 docs/contract.md 开头）。
func NormalizeCampusProbeInterval(d time.Duration) (time.Duration, bool) {
	floor := CampusProbeFloorSeconds * time.Second
	if d <= 0 {
		return DefaultCampusProbeSeconds * time.Second, false
	}
	if d < floor {
		return floor, true
	}
	return d, false
}

// AlertConfig 把设置里的告警参数转成 core/alert 的配置。
func (s Settings) AlertConfig() alert.Config {
	return alert.Config{
		BalanceThreshold: s.BalanceThreshold,
		DaysThreshold:    s.DaysThreshold,
		QuietStartHour:   s.QuietStartHour,
		QuietEndHour:     s.QuietEndHour,
	}
}

// TimetableConfig 转成 app 课表用例需要的 model.Config。
func (s Settings) TimetableConfig() model.Config {
	return model.Config{
		StudentID:    s.StudentID,
		EduPassword:  s.EduPassword,
		CustomerCode: s.CustomerCode,
	}
}

// ElectricityConfig 转成 app.CheckElectricity 需要的 model.Config。
func (s Settings) ElectricityConfig(defaultCustomerCode string) model.Config {
	code := s.CustomerCode
	if code == "" {
		code = defaultCustomerCode
	}
	return model.Config{StudentID: s.StudentID, CustomerCode: code}
}

// SettingsStore 负责配置的读写与加密（加密由传入的 Store 装饰器完成）。
type SettingsStore struct {
	store core.KVStore
}

// NewSettingsStore 包装一个已加密的 KVStore。
func NewSettingsStore(store core.KVStore) *SettingsStore {
	return &SettingsStore{store: store}
}

// Load 读配置；缺失时返回默认值而不是错误（首次启动是正常路径）。
func (s *SettingsStore) Load() (Settings, error) {
	cfg := DefaultSettings()
	raw, err := s.store.Get(ConfigStoreKey)
	if err != nil || len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		// 配置损坏不阻断启动：回到默认值，让用户重新填一次。
		return DefaultSettings(), fmt.Errorf("配置无法解析，已回到默认值: %w", err)
	}
	cfg.normalize()
	return cfg, nil
}

// SettingsWereClamped 判断用户填的间隔是否被硬下限改写。
//
// ParseSettingsResult 已把 next 钳成合法值，SettingsStore.Save 再 normalize
// 时已看不到越界输入，所以钳制提示必须拿「表单原文 vs 生效值」比较；
// 否则低于下限会被静默改写（AGENTS §5.5 要求「钳制并提示」）。
func SettingsWereClamped(res SettingsResult, next Settings) bool {
	poll, err := strconv.Atoi(strings.TrimSpace(res.PollSeconds))
	if err == nil && poll > 0 && poll < int(next.PollInterval()/time.Second) {
		return true
	}
	probe, err := strconv.Atoi(strings.TrimSpace(res.CampusProbeSecs))
	return err == nil && probe > 0 && probe < int(next.CampusProbeInterval()/time.Second)
}

// ClampedSettingsHint 是成功保存后在设置窗口底部展示的钳制说明。
func ClampedSettingsHint(s Settings) string {
	return fmt.Sprintf("低于下限，已按电费 %d 秒 / 探测 %d 秒执行",
		int(s.PollInterval()/time.Second),
		int(s.CampusProbeInterval()/time.Second))
}

// Save 规范化后写盘；返回是否因下限被钳制。
func (s *SettingsStore) Save(cfg Settings) (bool, error) {
	clamped := cfg.normalize()
	raw, err := json.Marshal(cfg)
	if err != nil {
		return clamped, fmt.Errorf("配置编码失败: %w", err)
	}
	if err := s.store.Put(ConfigStoreKey, raw); err != nil {
		return clamped, fmt.Errorf("配置写入失败: %w", err)
	}
	return clamped, nil
}

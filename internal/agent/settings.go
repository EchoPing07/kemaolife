package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/alert"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// ConfigStoreKey 是 Windows 侧配置在 KVStore 中的键。
const ConfigStoreKey = "agent/config"

// Settings 是 Windows 端的全部可配置项（docs/002 §5.2）。
//
// 与 Linux 的 model.Config 分开：Windows 不需要教务凭据（不做课表），
// 但需要校园网凭据（独立字段，D9 已定稿）与轮询 / 告警参数。
// 落盘前由 DPAPI 加密（P5）。
type Settings struct {
	// 电费：学号 + 学校代码（电费接口无密码，api-docs/002 §1）。
	StudentID    string `json:"studentId"`
	CustomerCode string `json:"customerCode,omitempty"`

	// 校园网：独立凭据，不复用教务 / 电费账号。
	CampusUser        string `json:"campusUser"`
	CampusPassword    string `json:"campusPassword"`
	CampusFingerprint string `json:"campusFingerprint"`

	// 模式：默认手动（docs/002 §5.2）。
	AutoMode bool `json:"autoMode"`

	// AutoStart 与轮询 / 告警参数。
	AutoStart        bool    `json:"autoStart"`
	PollSeconds      int     `json:"pollSeconds"`
	BalanceThreshold float64 `json:"balanceThreshold"`
	DaysThreshold    int     `json:"daysThreshold"`
	QuietStartHour   int     `json:"quietStartHour"`
	QuietEndHour     int     `json:"quietEndHour"`
}

// DefaultSettings 返回首次启动的默认值（轮询 4 小时、阈值 10 度 / 3 天、静默 23–07）。
func DefaultSettings() Settings {
	return Settings{
		AutoMode:         false, // 默认手动
		AutoStart:        true,  // 登录时自启一次
		PollSeconds:      int(alert.DefaultPollInterval / time.Second),
		BalanceThreshold: alert.DefaultBalanceThreshold,
		DaysThreshold:    alert.DefaultDaysThreshold,
		QuietStartHour:   alert.DefaultQuietStartHour,
		QuietEndHour:     alert.DefaultQuietEndHour,
	}
}

// normalize 补齐 / 钳制非法值，返回是否发生了钳制（供 UI 提示）。
//
// 轮询间隔低于 600s 一律钳制（AGENTS §5.5：不可放行）。
func (s *Settings) normalize() (clamped bool) {
	def := DefaultSettings()
	if s.PollSeconds <= 0 {
		s.PollSeconds = def.PollSeconds
	}
	if d, c := alert.NormalizePollInterval(time.Duration(s.PollSeconds) * time.Second); c {
		s.PollSeconds = int(d / time.Second)
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

// AlertConfig 把设置里的告警参数转成 core/alert 的配置。
func (s Settings) AlertConfig() alert.Config {
	return alert.Config{
		BalanceThreshold: s.BalanceThreshold,
		DaysThreshold:    s.DaysThreshold,
		QuietStartHour:   s.QuietStartHour,
		QuietEndHour:     s.QuietEndHour,
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

package app

import (
	"encoding/json"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// ConfigStoreKey 是配置在 ConfigStore 中的键。
const ConfigStoreKey = "config"

// LoadConfig 读配置；缺失时返回零值配置与 nil（首次使用不是错误）。
func LoadConfig(d Deps) (model.Config, error) {
	var cfg model.Config
	raw, err := d.ConfigStore.Get(ConfigStoreKey)
	if err != nil || len(raw) == 0 {
		// 首次使用：仍然给出内置默认学校代码，否则 config show 会显示 missing。
		cfg.CustomerCode = d.DefaultCustomerCode
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, model.WrapError(model.CodeInvalidArgument, "配置文件无法解析，请重新执行 config init", err)
	}
	if cfg.CustomerCode == "" {
		cfg.CustomerCode = d.DefaultCustomerCode
	}
	return cfg, nil
}

// SaveConfig 写配置。实现方（platform）负责 0600 与原子写（P5）。
func SaveConfig(d Deps, cfg model.Config) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return model.WrapError(model.CodeInvalidArgument, "配置编码失败", err)
	}
	if err := d.ConfigStore.Put(ConfigStoreKey, raw); err != nil {
		return model.WrapError(model.CodeNetworkDown, "配置写入失败", err)
	}
	return nil
}

// CustomerCodeSource 返回学校代码来源：custom（显式覆盖）/ default（内置默认）/ missing。
func CustomerCodeSource(d Deps, cfg model.Config) string {
	switch {
	case strings.TrimSpace(cfg.CustomerCode) == "":
		return "missing"
	case d.DefaultCustomerCode != "" && cfg.CustomerCode == d.DefaultCustomerCode:
		return model.CustomerCodeSourceDefault
	default:
		return model.CustomerCodeSourceCustom
	}
}

// MaskSecret 把敏感值脱敏为 ****；空值也显示 ****（docs/002 §1.1：恒脱敏）。
func MaskSecret(string) string { return "****" }

// ensureDeps 是各用例的共同前置检查，避免 nil 端口在运行期 panic。
func ensureDeps(d Deps, needHTTP, needStore bool) error {
	if d.Clock == nil {
		return model.NewError(model.CodeInvalidArgument, "缺少时钟端口")
	}
	if needHTTP && d.HTTP == nil {
		return model.NewError(model.CodeInvalidArgument, "缺少 HTTP 端口")
	}
	if needStore && d.Store == nil {
		return model.NewError(model.CodeInvalidArgument, "缺少存储端口")
	}
	return nil
}

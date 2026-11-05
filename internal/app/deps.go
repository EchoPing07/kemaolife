// Package app 是用例编排层：只调 core 与端口，不含平台判断（AGENTS §3）。
package app

import (
	"github.com/EchoPing07/kemaolife/internal/core"
)

// Deps 是各用例需要的端口与配置。
//
// 全部由 cmd 组装；app 不认识文件系统、不认识 GOOS。
type Deps struct {
	Clock core.Clock
	// HTTP 是教务与电费共用的一次性请求器（可共享连接池）。
	HTTP core.HTTPDoer
	// Store 存快照（含 Cookie，等同凭据）。
	Store core.KVStore
	// ConfigStore 存配置（0600）。
	ConfigStore core.KVStore

	// DefaultCustomerCode 由构建期注入或发行包配置提供，仓库内不出现真实值（AGENTS §5.6）。
	DefaultCustomerCode string

	// CASBaseURL / JSBBaseURL / ElectricityEndpoint 仅供测试覆盖，正常留空。
	// 教务实测是两个主机：CAS 在 aic，jsxsd 在 jw（见 internal/core/timetable/protocol.go）。
	CASBaseURL          string
	JSBBaseURL          string
	ElectricityEndpoint string
}

package model

// Config 是 Linux 端本地配置（docs/002 §1.1「配置分域」）。
//
// 敏感字段（学号 / 教务密码）落盘必须是 0600 文件（P5）；展示时恒脱敏。
// 校园网凭据不在此结构里：Linux 不做校园网（R3）。
type Config struct {
	StudentID   string `json:"studentId"`
	EduPassword string `json:"eduPassword"`
	// CustomerCode 是电费的学校代码，因校而异（api-docs/002 §10.2）。
	// 允许随发行包内置默认值，但仓库内不得出现真实值（AGENTS §5.6）。
	CustomerCode string `json:"customerCode,omitempty"`
}

// CustomerCodeSourceDefault / Custom 用于 config show 标注学校代码来源。
const (
	CustomerCodeSourceDefault = "default"
	CustomerCodeSourceCustom  = "custom"
)

// CredentialComplete 判断查询电费所需的身份信息是否齐全（电费无密码）。
func (c Config) CredentialComplete() bool {
	return c.StudentID != "" && c.CustomerCode != ""
}

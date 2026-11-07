package model

// Config 是 CLI 版本地配置（docs/002 §1.1「配置分域」）。
//
// 敏感字段（学号 / 教务密码）落盘必须是 0600 文件（P5）；展示时恒脱敏。
// 校园网凭据不在此结构里：CLI 版不做校园网（R3）。
type Config struct {
	StudentID   string `json:"studentId"`
	EduPassword string `json:"eduPassword"`
	// CustomerCode 是电费的学校代码，因校而异（api-docs/002 §10.2）。
	// 学校代码属学校事实，内置默认值写在 cmd 层（3126），不写成 core 包级常量。
	CustomerCode string `json:"customerCode,omitempty"`
}

// CustomerCodeSourceDefault / Custom 用于 config show 标注学校代码来源。
const (
	CustomerCodeSourceDefault = "default"
	CustomerCodeSourceCustom  = "custom"
)

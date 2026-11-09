package model

import "strings"

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
	// ElectricityStudentID 是电费单独使用的学号；为空时回落 StudentID。
	//
	// 为什么允许与教务学号不同：两端设置页把「电费继承教务信息」做成开关，
	// 关掉后用户可以只填电费学号而不填教务密码（docs/002 §5.3 tab3）。
	ElectricityStudentID string `json:"electricityStudentId,omitempty"`
}

// ElectricityStudent 返回电费应使用的学号：独立学号（去空白后非空）优先，
// 否则回落教务学号。空白独立学号必须视为「没配」，否则上游会拿到一串空格
// 并在解析失败时不回落有效学号（docs/002 §1.1 配置分域）。
func (c Config) ElectricityStudent() string {
	if id := strings.TrimSpace(c.ElectricityStudentID); id != "" {
		return id
	}
	return strings.TrimSpace(c.StudentID)
}

// CustomerCodeSourceDefault / Custom 用于 config show 标注学校代码来源。
const (
	CustomerCodeSourceDefault = "default"
	CustomerCodeSourceCustom  = "custom"
)

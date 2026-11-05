package model

// 全部 DTO 字段名以 docs/002 §3 为准；三端共享，改动需先改文档。

// Entry 是某一天的一节课。
type Entry struct {
	Name      string `json:"name"`
	Periods   string `json:"periods"` // 如 "1-2"
	Time      string `json:"time"`    // 如 "08:30~10:00"
	Teacher   string `json:"teacher"`
	Location  string `json:"location"`
	WeeksText string `json:"weeksText"` // 如 "1-16周"
}

// DaySchedule 是某一天的课表。
type DaySchedule struct {
	Date       Date    `json:"date"`
	Week       int     `json:"week"`
	Weekday    int     `json:"weekday"`   // 周一 = 1 … 周日 = 7
	WeekdayCn  string  `json:"weekdayCn"` // 一 / 二 / … / 日
	InSemester bool    `json:"inSemester"`
	Entries    []Entry `json:"entries"`
}

// ElectricityStatus 是余额概况，来自 getbindroom。
type ElectricityStatus struct {
	RoomName      string   `json:"roomName"`
	RoomVerify    string   `json:"roomVerify"` // 恒脱敏后输出
	Balance       float64  `json:"balance"`
	Unit          string   `json:"unit"` // 度 / 吨
	BusinessType  int      `json:"businessType"`
	MeterStatus   string   `json:"meterStatus"`
	CollectedAt   string   `json:"collectedAt"`   // RFC3339，数据采集时间
	DailyAvg      *float64 `json:"dailyAvg"`      // 近 7 日均量，数据不足为 null
	EstimatedDays *int     `json:"estimatedDays"` // floor(balance/dailyAvg)，无意义为 null
}

// DayUsage 是某一天的用电量。
type DayUsage struct {
	Date string  `json:"date"`
	Use  float64 `json:"use"`
}

// ElectricityUsage 是某月的每日用量，来自 getusedetail。
type ElectricityUsage struct {
	YearMonth string     `json:"yearMonth"`
	MonthUse  float64    `json:"monthUse"`
	DataUntil string     `json:"dataUntil"` // 数据截止日 YYYY-MM-DD
	Days      []DayUsage `json:"days"`      // 按日期升序
}

// ElectricityReport 是一次电费查询的完整结果（两个请求，任一失败整体报错）。
type ElectricityReport struct {
	Status ElectricityStatus `json:"status"`
	Usage  ElectricityUsage  `json:"usage"`
}

// CampusState 是校园网三态。
type CampusState string

const (
	CampusOnline  CampusState = "online"  // 已在线
	CampusUnauth  CampusState = "unauth"  // 未认证
	CampusOffline CampusState = "offline" // 不在校园网 / 探测失败
)

// CampusStatus 是校园网状态 DTO（Windows / Android 使用，Linux 不暴露）。
type CampusStatus struct {
	State     CampusState `json:"state"`
	ErrorMsg  string      `json:"errorMsg"`
	CheckedAt string      `json:"checkedAt"`
}

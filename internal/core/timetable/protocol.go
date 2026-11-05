package timetable

// 本文件集中放教务协议的事实（实测于 2026-11，见 api-docs/001-timetable）。
//
// 实测结论与文档早期描述的差异（以本文件为准）：
//   - 认证走统一 CAS（aic 主机），但 service 必须指向 jw 主机的 jsxsd；
//   - jsxsd 会话 Cookie 是 bzb_jsxsd（需带 ;path=/ 前缀的形式由 http 层处理）；
//   - 课程数据不是「全学期 JSON」，而是 xskb_list.do POST 返回 `{"code":0,"data":[...]}`；
//   - 周次/节次合并在单字段 `sktime`（如 `周三第1、2节{2,4,7-10}全周`）；
//   - 日期锚点来自 loadkb?zc=1 的表格文本（形如 `08-31`），不是 `YYYY-MM-DD`。
//
// 改动这些常量属于协议修正：必须以实测为准并同步 api-docs/001-timetable。

const (
	// CASBase 是统一身份认证站点（实测：aic 主机）。
	CASBase = "https://aic.gdkm.edu.cn"

	// JSBBase 是教务 jsxsd 站点（实测：与 CAS 不同主机）。
	JSBBase = "https://jw.gdkm.edu.cn"

	// CASLoginPath 是 CAS 登录端点，service 参数指向 JSBTimetableService。
	CASLoginPath = "/cas_server/login"

	// JSBTimetableService 是 CAS 登录成功后回跳的教务入口。
	//
	// 必须是 http（实测为明文 http），且主机是 jw 而不是 aic：
	// 用 https 或 aic 主机会被 CAS 跳到 /cas_server/error。
	JSBTimetableService = "http://jw.gdkm.edu.cn/jsxsd/"

	// Week1AnchorPath 取第 1 周 7 个日期的锚点（表格文本，形如 08-31）。
	Week1AnchorPath = "/jsxsd/framework/mainV_index_loadkb.htmlx?zc=1"
	// HomePath 是登录后的教务主页，用于探测会话是否有效。
	HomePath = "/jsxsd/framework/xsMainV.htmlx"

	// CourseListPath 取课表 JSON（POST）。
	CourseListPath = "/jsxsd/xskb/xskb_list.do?viweType=1&needData=1"

	// CAS 表单字段名（ASP.NET WebForms，实测）。
	CASFieldUser            = "signupInputUserName"
	CASFieldPass            = "signupInputPassword"
	CASFieldSubmit          = "btn_login"
	CASFieldViewState       = "__VIEWSTATE"
	CASFieldViewStateGen    = "__VIEWSTATEGENERATOR"
	CASFieldEventValidation = "__EVENTVALIDATION"

	// SessionCookie / ServerCookie 是 jsxsd 的两个会话 Cookie（实测，缺一不可）。
	SessionCookie = "bzb_jsxsd"
	ServerCookie  = "SERVERID_jsxsd"

	// CourseListFormFields 是课表 POST 的固定表单字段（实测）。
	formFieldViewType = "viweType"
	formFieldNeedData = "needData"
	formFieldBaseURL  = "baseUrl"
	formFieldSfykb    = "sfykb"
	formFieldXSFL     = "xsflMapListJsonStr"
	formFieldPageNum  = "pageNum"
	formFieldPageSize = "pageSize"
)

// Course JSON 字段名（实测）。
const (
	fieldCourseName    = "kc_mc"    // 课程名
	fieldCourseTime    = "sktime"   // 上课安排：`周三第1、2节{2,4,7-10}全周`
	fieldCourseTeacher = "jg0101mc" // 教师
	fieldCourseRoom    = "skddmc"   // 地点（多个用 ; 分隔）
	fieldCourseCampus  = "skxqmc"   // 校区
	fieldCourseCode    = "kch"      // 课程号
	fieldCourseClass   = "kt_mc"    // 教学班
)

// CourseListKeys 是可能承载课程行的顶层数组键。
var CourseListKeys = []string{"data", "rows"}

// SlotTimes 是 6 个大节时间（实测于 loadkb 表格，供时间展示）。
//
// 教务不下发「大节时间表」字段，只在表格里渲染；这里按实测值固定，
// 若学校调整作息需要更新（api-docs/001 §7 待验证）。
var SlotTimes = []string{
	"08:30~10:00",
	"10:20~11:50",
	"14:00~15:20",
	"15:30~16:50",
	"18:30~19:50",
	"20:00~21:20",
}

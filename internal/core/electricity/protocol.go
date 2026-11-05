package electricity

// 协议常量；事实来源 api-docs/002-electricity。
//
// 改动这些值属于协议修正，须以实测为准并同步 api-docs/002。

const (
	// Endpoint 是 SWAEServlet 端点（api-docs/002 §1）。
	Endpoint = "https://xqh5.17wanxiao.com/smartWaterAndElectricityService/SWAEServlet"

	// CmdGetBindRoom 取已绑定房间与余额概况（api-docs/002 §5、§6.1）。
	CmdGetBindRoom = "getbindroom"

	// CmdGetUseDetail 取指定月份每日用量（api-docs/002 §5、§6.2）。
	CmdGetUseDetail = "getusedetail"

	// CmdGetStuIndexPage 取首页聚合数据；本项目只从中读数据采集时刻。
	CmdGetStuIndexPage = "getstuindexpage"

	// BusinessTypeElectric 是电；水为 1（api-docs/002 §7）。
	BusinessTypeElectric = "0"

	// 外层字段名（api-docs/002 §4）。
	outerFieldCode    = "code_"
	outerFieldResult  = "result_"
	outerFieldBody    = "body"
	outerFieldMessage = "message_"

	// 内层 body 字段（api-docs/002 §6.1、§6.2）。
	bodyFieldResult     = "result"
	bodyFieldMessage    = "message"
	bodyFieldRoomName   = "roomfullname"
	bodyFieldRoomVer    = "roomverify"
	bodyFieldExistFlag  = "existflag"
	bodyFieldDetail     = "detaillist"
	bodyFieldYearMonth  = "yearmonth"
	bodyFieldMonthUse   = "monthuse"
	bodyFieldOpdtDate   = "opdtdate" // 数据截止日
	bodyFieldDayUses    = "dayuselist"
	bodyFieldModList    = "modlist"    // getstuindexpage 的数据块数组
	bodyFieldCollecDate = "collecdate" // 数据采集时刻，如 "2026-11-05 22:24:52"

	detailFieldBusinessType = "businesstype"
	detailFieldOdd          = "odd" // 剩余量
	detailFieldStatus       = "status"

	dayFieldDate   = "date"
	dayFieldDayUse = "dayuse"
	dayFieldUse    = "use"
)

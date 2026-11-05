package main

import (
	"fmt"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// 帮助文本是纯文本，不进 JSON 信封（docs/002 §1.1）。

const rootHelp = `kemaolife —— 课表与电费查询（Linux CLI，供 agent 使用）

用法:
  kemaolife <命令> [参数] [全局参数]

命令:
  config init                       交互录入学号与教务密码（配置权限 0600）
  config set customercode <代码>    覆盖内置的学校代码
  config show                       显示配置状态（学号与密码恒脱敏）
  timetable                         查询当日课表（等价于 timetable today）
  timetable sync                    登录教务并刷新快照
  timetable today                   查询当日课表
  timetable week [N]                查询第 N 周课表，省略 N 为当周
  timetable date <YYYY-MM-DD>       查询指定日期课表
  electricity [--month YYYY-MM]     一次返回剩余电费与每日用量
  help <命令>                       查看子命令帮助

全局参数:
  --json             stdout 只输出 JSON 信封，诊断信息走 stderr
  --config <path>    指定配置文件（默认 ~/.config/kemaolife/config.json）
  --timeout <dur>    单次网络超时上限（默认 30s，示例 30s / 1m）
  --verbose          诊断信息输出到 stderr（不含个人信息）
  --version          输出版本号（不联网、不读配置）
  --help             输出本帮助

退出码:
  0 成功  1 业务失败  2 参数错误  3 凭据缺失或会话失效  4 网络或接口不可用

说明:
  课表查询只读本地快照，0 网络请求、可离线；仅 timetable sync 会联网。
  电费数据约有 1 天延迟，输出中的「数据截至」是采集时间，不是实时值。
`

func printHelp(w interface{ Write([]byte) (int, error) }) {
	fmt.Fprint(w, rootHelp)
}

func cmdHelp(args []string) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}
	text, ok := commandHelp[args[0]]
	if !ok {
		return model.NewError(model.CodeInvalidArgument,
			"未知子命令 "+quote(args[0])+"，执行 kemaolife --help 查看用法")
	}
	writeStdout(text)
	return nil
}

var commandHelp = map[string]string{
	"config": `config —— 配置管理

  kemaolife config init
      交互录入学号与教务密码，写入 <配置文件>（权限 0600）。
      密码不回显、不进命令行参数、不写日志。

  kemaolife config set customercode <学校代码>
      覆盖学校代码内置默认值。学校代码是电费接口的必填项。

  kemaolife config show
      显示配置状态；学号与密码恒显示为 ****。

  支持的参数: --json --config <path>
`,
	"timetable": `timetable —— 课表查询

  kemaolife timetable                 当日课表
  kemaolife timetable sync            登录教务并刷新快照（1 登录 + 2 请求）
  kemaolife timetable today           当日课表
  kemaolife timetable week [N]        第 N 周课表；省略 N 为当周
  kemaolife timetable date YYYY-MM-DD 指定日期课表

  查询只读本地快照，0 网络请求、可离线；快照缺失时返回退出码 3，
  不会自动联网。非学期内日期返回 inSemester:false 与空 entries（退出码 0）。

  支持的参数: --json --config <path> --timeout <dur>
`,
	"electricity": `electricity —— 电费查询

  kemaolife electricity [--month YYYY-MM]

  一次调用返回剩余电费与每日用量（内部 2 个请求）。任一请求失败整体报错，
  不返回半份数据。--month 省略时取当月。

  输出中的 collectedAt 是数据采集时间（电表约 1 天延迟），必须标注「数据截至」。

  支持的参数: --json --config <path> --timeout <dur>
`,
}

// versionInfo 是 --version --json 的 data。
type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func cmdVersion(opts options) error {
	if opts.json {
		writeJSON(stdout, model.Success(time.Now(), versionInfo{Version: version, Commit: commit}))
		return nil
	}
	writeStdout(fmt.Sprintf("kemaolife %s (%s)\n", version, commit))
	return nil
}

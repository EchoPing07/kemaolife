package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/platform"
)

// stdin 在测试里被替换，避免直接依赖终端。
var stdin io.Reader = os.Stdin

// cmdConfig 实现 config init / set customercode / show（docs/002 §1.1）。
func cmdConfig(ctx context.Context, opts options, rt *session, args []string) error {
	_ = ctx // 配置操作不联网，保留签名与其它子命令一致
	if len(args) == 0 {
		return model.NewError(model.CodeInvalidArgument,
			"config 需要子命令：init、set、show")
	}
	switch args[0] {
	case "init":
		return configInit(opts, rt)
	case "set":
		return configSet(opts, rt, args[1:])
	case "show":
		return configShow(opts, rt)
	}
	return model.NewError(model.CodeInvalidArgument,
		"未知 config 子命令 "+quote(args[0])+"，可用：init、set、show")
}

// configInitSchema 是 --json 模式下 config init 的 data（学号与密码恒脱敏）。
type configInitSchema struct {
	StudentID   string `json:"studentId"`
	EduPassword string `json:"eduPassword"`
	ConfigPath  string `json:"configPath"`
}

// configInit 录入学号与教务密码。
//
// 密码不经命令行参数（会进 shell history / ps），只从 stdin 读；
// stdin 是终端时关闭回显。两者都不写日志。
func configInit(opts options, rt *session) error {
	// 交互提示只在人类使用时打印；管道喂入时不污染 stdout（agent 固定带 --json）。
	// 判据要同时满足「stdin 是终端」与「stdin 没被替换」，否则测试 / 自动化会看到提示语。
	interactive := stdinIsTerminal()

	// 注意：两次读取必须共用同一个 bufio.Reader。每次新建 reader 会把
	// 下一次的输入提前缓冲进丢掉的那个 reader 里，导致「密码读不到」。
	reader := bufio.NewReader(stdin)

	if interactive {
		writePrompt("学号: ")
	}
	studentID, err := readLine(reader)
	if err != nil {
		return model.WrapError(model.CodeInvalidArgument, "读取学号失败", err)
	}
	studentID = strings.TrimSpace(studentID)
	if studentID == "" {
		return model.NewError(model.CodeInvalidArgument, "学号不能为空")
	}

	if interactive {
		writePrompt("教务密码（输入不回显）: ")
	}
	password, err := platform.ReadSecret(reader, interactive)
	if interactive {
		writePrompt("\n")
	}
	if err != nil {
		return model.WrapError(model.CodeInvalidArgument, "读取密码失败", err)
	}
	if password == "" {
		return model.NewError(model.CodeInvalidArgument, "教务密码不能为空")
	}

	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	cfg.StudentID = studentID
	cfg.EduPassword = password
	if err := app.SaveConfig(rt.deps, cfg); err != nil {
		return err
	}

	data := configInitSchema{
		StudentID:   app.MaskSecret(studentID),
		EduPassword: app.MaskSecret(password),
		ConfigPath:  rt.configPath,
	}
	return envelope(opts, model.Envelope{
		OK:   true,
		Data: data,
		TS:   formatNow(rt),
	}, func() {
		writeStdout("配置已保存（权限 0600）: " + rt.configPath + "\n")
		if cfg.CustomerCode == "" && rt.deps.DefaultCustomerCode == "" {
			writeStdout("提示：尚未设置学校代码，电费查询前请执行 kemaolife config set customercode <学校代码>\n")
		}
	})
}

// configSet 支持 `set customercode <学校代码>`。
func configSet(opts options, rt *session, args []string) error {
	if len(args) == 0 {
		return model.NewError(model.CodeInvalidArgument,
			"config set 需要字段名，目前支持：customercode")
	}
	field := args[0]
	if field != "customercode" {
		return model.NewError(model.CodeInvalidArgument,
			"不支持修改字段 "+quote(field)+"，目前支持：customercode")
	}
	if len(args) < 2 {
		return model.NewError(model.CodeInvalidArgument,
			"缺少取值：kemaolife config set customercode <学校代码>")
	}
	value := strings.TrimSpace(args[1])
	if value == "" {
		return model.NewError(model.CodeInvalidArgument, "学校代码不能为空")
	}

	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	cfg.CustomerCode = value
	if err := app.SaveConfig(rt.deps, cfg); err != nil {
		return err
	}

	// 学校代码属敏感配置：对外只报来源，不回显取值（P1 / docs/002 §1.1）。
	data := map[string]string{
		"customerCode":       app.MaskSecret(value),
		"customerCodeSource": app.CustomerCodeSource(rt.deps, cfg),
	}
	return envelope(opts, model.Envelope{OK: true, Data: data, TS: formatNow(rt)}, func() {
		writeStdout("学校代码已更新（来源 " + data["customerCodeSource"] + "）\n")
	})
}

// configShowSchema 是 config show 的 data（docs/002 §1.1）。
type configShowSchema struct {
	StudentID          string `json:"studentId"`
	EduPassword        string `json:"eduPassword"`
	CustomerCode       string `json:"customerCode"`
	CustomerCodeSource string `json:"customerCodeSource"`
	ConfigPath         string `json:"configPath"`
}

// configShow 展示配置状态；学号、密码、学校代码恒脱敏（恒 ****）。
func configShow(opts options, rt *session) error {
	cfg, err := app.LoadConfig(rt.deps)
	if err != nil {
		return err
	}
	source := app.CustomerCodeSource(rt.deps, cfg)

	data := configShowSchema{
		StudentID:          app.MaskSecret(cfg.StudentID),
		EduPassword:        app.MaskSecret(cfg.EduPassword),
		CustomerCode:       app.MaskSecret(cfg.CustomerCode),
		CustomerCodeSource: source,
		ConfigPath:         rt.configPath,
	}
	return envelope(opts, model.Envelope{OK: true, Data: data, TS: formatNow(rt)}, func() {
		rows := [][]string{
			{"学号", data.StudentID},
			{"教务密码", data.EduPassword},
			{"学校代码", data.CustomerCode + "（来源 " + source + "）"},
			{"配置文件", data.ConfigPath},
		}
		writeStdout(renderTable([]string{"项", "值"}, rows))
		if cfg.StudentID == "" || cfg.EduPassword == "" {
			writeStdout("\n尚未配置教务账号，请执行 kemaolife config init\n")
		}
	})
}

// stdinIsTerminal 判断当前是否在和真人交互。
func stdinIsTerminal() bool {
	if stdin != io.Reader(os.Stdin) {
		return false
	}
	return platform.IsTerminal()
}

// readLine 读一行，容忍裸 \r\n。
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	if err == io.EOF && line == "" {
		return "", fmt.Errorf("输入已结束")
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// writePrompt 把交互提示写到 stderr：--json 时 stdout 必须只有信封（docs/002 §1.1）。
func writePrompt(s string) { fmt.Fprint(stderr, s) }

// formatNow 统一信封 ts；命令内部时间一律经 deps.Clock（AGENTS §6）。
func formatNow(rt *session) string {
	return rt.deps.Clock.Now().In(time.Local).Format(time.RFC3339)
}

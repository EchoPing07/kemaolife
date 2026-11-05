package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// defaultTimeout 是 --timeout 未指定时的单次网络超时上限（docs/002 §1.1）。
const defaultTimeout = 30 * time.Second

// defaultCustomerCode 是本项目的学校代码内置默认值（学校事实，非个人隐私）。
// 可用 -ldflags "-X main.defaultCustomerCode=<学校代码>" 覆盖；
// 用户还能用 config set customercode 再覆盖。
var defaultCustomerCode = "3126"

// options 是全局参数解析结果（docs/002 §1.1「全局参数」）。
type options struct {
	json       bool
	configPath string
	timeout    time.Duration
	verbose    bool
	version    bool
	help       bool
}

// parseGlobalFlags 只解析出现在命令之前的全局参数；子命令自己的参数由各 cmd* 处理。
//
// 这样 `kemaolife timetable week 3 --json` 与 `kemaolife --json timetable week 3`
// 都能工作，agent 不需要记住参数位置。
func parseGlobalFlags(args []string) (options, []string, error) {
	opts := options{timeout: defaultTimeout}
	rest := make([]string, 0, len(args))

	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") {
			// 第一个非参数即子命令；其后全部交给子命令（含它自己的 --json）。
			rest = append(rest, args[i:]...)
			break
		}

		name, value, hasValue := splitFlag(arg)
		switch name {
		case "-h", "--help":
			opts.help = true
		case "-v", "--version":
			opts.version = true
		case "--json":
			opts.json = true
		case "--verbose":
			opts.verbose = true
		case "--config":
			v, err := flagValue(name, value, hasValue, args, &i)
			if err != nil {
				return opts, rest, err
			}
			opts.configPath = v
		case "--timeout":
			v, err := flagValue(name, value, hasValue, args, &i)
			if err != nil {
				return opts, rest, err
			}
			d, err := parseTimeout(v)
			if err != nil {
				return opts, rest, err
			}
			opts.timeout = d
		default:
			return opts, rest, model.NewError(model.CodeInvalidArgument,
				"未知参数 "+quote(name)+"，执行 kemaolife --help 查看用法")
		}
		i++
	}

	// 子命令之后仍可能带全局参数（agent 习惯写在末尾）；把它们摘出来。
	tail, err := extractTrailingGlobals(&opts, rest)
	if err != nil {
		return opts, nil, err
	}
	return opts, tail, nil
}

// extractTrailingGlobals 把子命令参数里的全局开关摘走，其余保持原序。
// 带值参数（--config / --timeout）也支持放在末尾，agent 不必记住参数位置。
func extractTrailingGlobals(opts *options, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, hasValue := splitFlag(arg)
		switch name {
		case "--json":
			opts.json = true
		case "--verbose":
			opts.verbose = true
		case "-h", "--help":
			opts.help = true
		case "--config", "--timeout":
			v, err := flagValue(name, value, hasValue, args, &i)
			if err != nil {
				return nil, err
			}
			if name == "--config" {
				opts.configPath = v
			} else {
				d, err := parseTimeout(v)
				if err != nil {
					return nil, err
				}
				opts.timeout = d
			}
		default:
			out = append(out, arg)
		}
	}
	return out, nil
}

func splitFlag(arg string) (name, value string, hasValue bool) {
	if idx := strings.IndexByte(arg, '='); idx > 0 {
		return arg[:idx], arg[idx+1:], true
	}
	return arg, "", false
}

func flagValue(name, value string, hasValue bool, args []string, i *int) (string, error) {
	if hasValue {
		return value, nil
	}
	if *i+1 >= len(args) {
		return "", model.NewError(model.CodeInvalidArgument, "参数 "+name+" 缺少取值")
	}
	*i++
	return args[*i], nil
}

// parseTimeout 解析 `30s` / `1m` 这类时长；也接受纯秒数。
func parseTimeout(raw string) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, model.NewError(model.CodeInvalidArgument, "--timeout 取值不能为空")
	}
	if secs, err := strconv.Atoi(s); err == nil {
		if secs <= 0 {
			return 0, model.NewError(model.CodeInvalidArgument, "--timeout 必须大于 0")
		}
		return time.Duration(secs) * time.Second, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, model.NewError(model.CodeInvalidArgument,
			fmt.Sprintf("--timeout 取值无法解析: %q（示例：30s、1m）", raw))
	}
	return d, nil
}

func quote(s string) string { return strconv.Quote(s) }

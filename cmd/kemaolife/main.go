package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/platform"
)

// 版本信息由编译期注入（docs/002 §1.1）：
//
//	go build -ldflags "-X main.version=<版本号> -X main.commit=<提交号>"
//
// --version 不联网、不读配置。
var (
	version = "0.0.0-dev"
	commit  = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run 只做「解析参数 → 组装 Deps → 调用例 → 输出 / 退出码」，不含业务逻辑（AGENTS §3）。
func run(args []string) int {
	opts, rest, err := parseGlobalFlags(args)
	if err != nil {
		return failWith(opts, nil, err)
	}

	if opts.version {
		return finish(opts, nil, cmdVersion(opts))
	}
	if opts.help || (len(rest) == 0) {
		printHelp(stdout)
		return 0
	}
	if rest[0] == "help" {
		return finish(opts, nil, cmdHelp(rest[1:]))
	}

	// 命令分派前先校验子命令，避免为未知命令创建目录、读配置。
	if !knownCommand(rest[0]) {
		return failWith(opts, nil, model.NewError(model.CodeInvalidArgument,
			"未知命令 "+quote(rest[0])+"，执行 kemaolife --help 查看用法"))
	}

	rt, err := newRuntime(opts)
	if err != nil {
		return failWith(opts, nil, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if opts.verbose {
		writeStderr("诊断：已启动，超时上限 " + opts.timeout.String())
	}

	timeout := opts.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return finish(opts, rt, dispatch(ctx, opts, rt, rest))
}

func knownCommand(name string) bool {
	switch name {
	case "config", "timetable", "electricity":
		return true
	}
	return false
}

// dispatch 把子命令路由到实现；返回值统一交给 finish 处理输出与退出码。
func dispatch(ctx context.Context, opts options, rt *session, args []string) error {
	switch args[0] {
	case "config":
		return cmdConfig(ctx, opts, rt, args[1:])
	case "timetable":
		return cmdTimetable(ctx, opts, rt, args[1:])
	case "electricity":
		return cmdElectricity(ctx, opts, rt, args[1:])
	}
	return model.NewError(model.CodeInvalidArgument, "未知命令")
}

// finish 是唯一成功出口：输出已由子命令写好，这里只决定退出码。
func finish(opts options, rt *session, err error) int {
	if err != nil {
		return failWith(opts, rt, err)
	}
	return 0
}

// failWith 统一失败出口：--json 走 §2 信封，否则走 stderr 纯文本。
func failWith(opts options, rt *session, err error) int {
	now := time.Now()
	if rt != nil && rt.deps.Clock != nil {
		now = rt.deps.Clock.Now()
	}
	if opts.json {
		writeJSON(stdout, model.Failure(now, err))
	} else {
		writeStderr("错误: " + model.MessageOf(err))
	}
	return model.ExitCode(err)
}

// session 是一次调用组装好的依赖与目标路径。
type session struct {
	deps       app.Deps
	configPath string
	cacheDir   string
}

// newRuntime 是唯一知道「Linux 用哪个适配器」的地方。
func newRuntime(opts options) (*session, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, model.WrapError(model.CodeInvalidArgument, "无法确定用户主目录", err)
	}

	cacheDir, err := platform.DefaultCacheDir(os.Getenv, home)
	if err != nil {
		return nil, model.WrapError(model.CodeInvalidArgument, "无法确定缓存目录", err)
	}
	store, err := platform.NewFileStore(cacheDir)
	if err != nil {
		return nil, model.WrapError(model.CodeInvalidArgument, "无法使用缓存目录", err)
	}

	configPath := opts.configPath
	if configPath == "" {
		dir, err := platform.DefaultConfigDir(os.Getenv, home)
		if err != nil {
			return nil, model.WrapError(model.CodeInvalidArgument, "无法确定配置目录", err)
		}
		configPath = dir + string(os.PathSeparator) + "config.json"
	}
	configStore, err := platform.NewSingleFileStore(configPath, app.ConfigStoreKey)
	if err != nil {
		return nil, model.WrapError(model.CodeInvalidArgument, "无法使用配置目录", err)
	}

	doer := platform.NewHTTPDoer(opts.timeout)
	return &session{
		deps: app.Deps{
			Clock:               platform.RealClock{},
			HTTP:                doer,
			Store:               store,
			ConfigStore:         configStore,
			DefaultCustomerCode: defaultCustomerCode,
		},
		configPath: configPath,
		cacheDir:   cacheDir,
	}, nil
}

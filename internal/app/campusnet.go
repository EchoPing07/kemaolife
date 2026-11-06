package app

import (
	"context"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// CampusDeps 是校园网用例需要的端口。
//
// 单独一组而不是塞进 Deps：Linux 端没有校园网能力（000 §3.1），
// 让 Deps 常带一个必填的校园网端口，只会逼 Linux 端塞一个只会报错的桩。
type CampusDeps struct {
	Clock core.Clock
	Net   core.CampusNetwork
	Ready core.NetworkReady
}

// GetCampusNetworkStatus 判态（docs/002 §3.3）。
//
// 探测失败在网络适配器里已被归为 offline，所以这里不把 offline 当错误：
// 「不在校园网」是正常状态，只有真正的内部故障才返回错误。
func GetCampusNetworkStatus(ctx context.Context, d CampusDeps) (model.CampusStatus, error) {
	if err := ensureCampusDeps(d); err != nil {
		return model.CampusStatus{}, err
	}
	status, err := d.Net.Status(ctx)
	if err != nil {
		return model.CampusStatus{}, err
	}
	if status.CheckedAt == "" {
		status.CheckedAt = formatRFC3339(d.Clock.Now())
	}
	return status, nil
}

// LoginCampusNetwork 手动登录（Windows 手动模式 / Android 手动动作）。
func LoginCampusNetwork(ctx context.Context, d CampusDeps) error {
	if err := ensureCampusDeps(d); err != nil {
		return err
	}
	return d.Net.Login(ctx)
}

// LogoutCampusNetwork 手动下线。
func LogoutCampusNetwork(ctx context.Context, d CampusDeps) error {
	if err := ensureCampusDeps(d); err != nil {
		return err
	}
	return d.Net.Logout(ctx)
}

// ReloginCampusNetwork 手动重登：先判态，在线才下线并复探；随后登录 → 复探确认。
func ReloginCampusNetwork(ctx context.Context, d CampusDeps) error {
	if err := ensureCampusDeps(d); err != nil {
		return err
	}
	return d.Net.Relogin(ctx)
}

// EnsureCampusNetwork 是自动模式的轮询前置：确保网络可用。
//
// 只在 Windows 自动模式调用（001 §5）；手动模式调它会违反「不自动登录」。
func EnsureCampusNetwork(ctx context.Context, d CampusDeps) error {
	if err := ensureCampusDeps(d); err != nil {
		return err
	}
	if d.Ready == nil {
		return model.NewError(model.CodeInvalidArgument, "缺少网络就绪端口")
	}
	return d.Ready.EnsureCampusNetwork(ctx)
}

func ensureCampusDeps(d CampusDeps) error {
	if d.Clock == nil {
		return model.NewError(model.CodeInvalidArgument, "缺少时钟端口")
	}
	if d.Net == nil {
		return model.NewError(model.CodeInvalidArgument, "缺少校园网端口")
	}
	return nil
}

//go:build !windows && !android

package platform

import (
	"context"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// StubCampusNetwork 是 Linux 的校园网空实现（000 §3.1：Linux 不提供校园网功能）。
//
// 它让 app 的 Deps 组装在三端一致（001 §5），被调用即返回参数错误 ——
// 不静默成功，也不假装「不在校园网」。
type StubCampusNetwork struct{}

func (StubCampusNetwork) Status(context.Context) (model.CampusStatus, error) {
	return model.CampusStatus{}, model.NewError(model.CodeInvalidArgument,
		"Linux 端不提供校园网功能")
}

func (StubCampusNetwork) Login(context.Context) error   { return errNoCampusNet() }
func (StubCampusNetwork) Logout(context.Context) error  { return errNoCampusNet() }
func (StubCampusNetwork) Relogin(context.Context) error { return errNoCampusNet() }

// EnsureCampusNetwork 满足 core.NetworkReady。
func (StubCampusNetwork) EnsureCampusNetwork(context.Context) error { return errNoCampusNet() }

func errNoCampusNet() error {
	return model.NewError(model.CodeInvalidArgument, "Linux 端不提供校园网功能")
}

var (
	_ core.CampusNetwork = StubCampusNetwork{}
	_ core.NetworkReady  = StubCampusNetwork{}
)

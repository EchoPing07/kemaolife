//go:build !windows && !android

package platform

import (
	"context"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// StubCampusNetwork 是 CLI 版（非 Windows/Android）的校园网空实现
// （000 §3.1：CLI 版不提供校园网功能）。
// Android 不复用这个桩：它走 portal.go + portal_android.go 的真实适配器。
//
// 它是备用适配器，当前无装配点；被调用即返回参数错误 ——
// 不静默成功，也不假装「不在校园网」。
type StubCampusNetwork struct{}

func (StubCampusNetwork) Status(context.Context) (model.CampusStatus, error) {
	return model.CampusStatus{}, model.NewError(model.CodeInvalidArgument,
		"CLI 版不提供校园网功能")
}

func (StubCampusNetwork) Login(context.Context) error   { return errNoCampusNet() }
func (StubCampusNetwork) Logout(context.Context) error  { return errNoCampusNet() }
func (StubCampusNetwork) Relogin(context.Context) error { return errNoCampusNet() }

// EnsureCampusNetwork 满足 core.NetworkReady。
func (StubCampusNetwork) EnsureCampusNetwork(context.Context) error { return errNoCampusNet() }

func errNoCampusNet() error {
	return model.NewError(model.CodeInvalidArgument, "CLI 版不提供校园网功能")
}

var (
	_ core.CampusNetwork = StubCampusNetwork{}
	_ core.NetworkReady  = StubCampusNetwork{}
)

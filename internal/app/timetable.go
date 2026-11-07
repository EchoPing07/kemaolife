package app

import (
	"context"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// SyncResult 是一次同步的产出摘要；给调用方展示，不含 Cookie。
type SyncResult struct {
	SyncedAt    string `json:"syncedAt"`
	Week1Monday string `json:"week1Monday"`
	LastWeek    int    `json:"lastWeek"`
	LessonCount int    `json:"lessonCount"`
}

// SyncTimetable 登录并刷新快照（1 登录 + 2 请求）。
//
// 会话失效时内部自动重登一次（docs/002 §1.3「同步时会话失效」）；
// 仍失败返回 3002。这是 CLI 版唯一会联网的课表路径。
func SyncTimetable(ctx context.Context, d Deps, cfg model.Config) (SyncResult, error) {
	// 先判凭据，再检查端口：缺凭据是用户能自己修的问题，不该报成内部配置错误。
	if cfg.StudentID == "" || cfg.EduPassword == "" {
		return SyncResult{}, model.NewError(model.CodeCredentialMissing,
			"未配置教务账号，请先填写学号与教务密码")
	}
	if err := ensureDeps(d, true, true); err != nil {
		return SyncResult{}, err
	}

	client := timetable.NewClient(d.HTTP, d.Clock, d.CASBaseURL, d.JSBBaseURL)

	// 复用快照里的 Cookie 时先探活：仍然有效就能省掉一次完整登录。
	hasSession := false
	if old, err := timetable.LoadSnapshot(d.Store); err == nil && len(old.Cookies) > 0 {
		client.SetCookies(old.Cookies)
		if alive, err := client.IsAlive(ctx); err == nil && alive {
			hasSession = true
		}
	}

	snap, err := syncWithRelogin(ctx, client, cfg, hasSession)
	if err != nil {
		return SyncResult{}, err
	}

	if err := timetable.SaveSnapshot(d.Store, snap); err != nil {
		return SyncResult{}, err
	}
	return SyncResult{
		SyncedAt:    formatRFC3339(snap.SyncedAt),
		Week1Monday: snap.Week1Monday.String(),
		LastWeek:    snap.LastWeek,
		LessonCount: len(snap.Lessons),
	}, nil
}

// syncWithRelogin 执行同步，必要时登录。
//
// 规则（docs/002 §1.3）：
//   - 已有可用会话（hasSession）→ 直接同步；失败则重登一次再同步。
//   - 没有可用会话（首次 / 快照缺失 / 会话已失效）→ 先登录再同步，
//     其后的同步失败若仍是「会话失效」，重登一次后重试。
//
// 这里只允许一次重登：避免密码错误时无限重试。
func syncWithRelogin(ctx context.Context, client *timetable.Client, cfg model.Config, hasSession bool) (*timetable.Snapshot, error) {
	if !hasSession {
		if err := client.Login(ctx, cfg.StudentID, cfg.EduPassword); err != nil {
			return nil, err
		}
	}

	snap, err := client.Sync(ctx)
	if err == nil {
		return snap, nil
	}
	if model.CodeOf(err) != model.CodeSessionExpired {
		return nil, err
	}

	// 会话在同步途中失效：重登一次。
	if loginErr := client.Login(ctx, cfg.StudentID, cfg.EduPassword); loginErr != nil {
		return nil, loginErr
	}
	if snap, err = client.Sync(ctx); err == nil {
		return snap, nil
	}
	if model.CodeOf(err) != model.CodeSessionExpired {
		return nil, err
	}
	return nil, model.NewError(model.CodeSessionExpired,
		"教务会话失效且重新登录未成功，请确认账号密码后重试")
}

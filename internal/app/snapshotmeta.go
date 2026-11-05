package app

import (
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// SnapshotMeta 是快照的可公开摘要：只有时间与形状，不含 Cookies。
//
// 展示层（CLI 的 syncedAt、「上次同步于」）只需要这些，
// 让它们拿不到 Cookie 是刻意的（P6：快照等同凭据）。
type SnapshotMeta struct {
	SyncedAt    time.Time
	Week1Monday model.Date
	LastWeek    int
	LessonCount int
}

// SnapshotMetaOf 读快照摘要；快照缺失 / 版本不匹配返回 3003，不联网重拉。
func SnapshotMetaOf(d Deps) (*SnapshotMeta, error) {
	if err := ensureDeps(d, false, true); err != nil {
		return nil, err
	}
	snap, err := timetable.LoadSnapshot(d.Store)
	if err != nil {
		return nil, err
	}
	return &SnapshotMeta{
		SyncedAt:    snap.SyncedAt,
		Week1Monday: snap.Week1Monday,
		LastWeek:    snap.LastWeek,
		LessonCount: len(snap.Lessons),
	}, nil
}

package timetable

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// SnapshotVersion 是快照格式版本；不匹配即丢弃重拉，不做兼容迁移（AGENTS §4.3）。
const SnapshotVersion = 2

// ErrSnapshotAbsent 是「从来没同步过」的哨兵，供 app 层把首次使用映射为 3001。
//
// 只有「文件不存在 / 无内容」才算真的没同步过；JSON 损坏、版本不匹配、内容
// 不完整、IO 错误都不能压成配置问题，否则用户会被引导去改凭据而不是重新同步。
var ErrSnapshotAbsent = errors.New("snapshot absent")

// SnapshotStoreKey 是快照在 KVStore 中的键。
const SnapshotStoreKey = "timetable/snapshot"

// Snapshot 是教务同步产出的课表数据包。含 Cookies，等同凭据（P6）。
type Snapshot struct {
	Version     int               `json:"version"`
	SyncedAt    time.Time         `json:"syncedAt"`
	Week1Monday model.Date        `json:"week1Monday"`
	Week1Dates  []model.Date      `json:"week1Dates"`
	SlotTimes   []string          `json:"slotTimes"` // 6 个大节时间
	LastWeek    int               `json:"lastWeek"`
	Lessons     []Lesson          `json:"lessons"`
	Cookies     map[string]string `json:"cookies"`
}

// Lesson 是一节课的周次展开结果（不含日期）。
type Lesson struct {
	Name      string `json:"name"`
	Periods   string `json:"periods"`
	Weekday   int    `json:"weekday"` // 周一 = 1 … 周日 = 7
	Weeks     []int  `json:"weeks"`
	Teacher   string `json:"teacher"`
	Location  string `json:"location"`
	WeeksText string `json:"weeksText"`
}

// LoadSnapshot 从 KVStore 读快照并校验版本。
//
// 缺失或版本不匹配一律返回 3003（CodeSnapshotMissing）：由 app 层先刷新，
// 刷新也失败且无可用快照时才把该错误交给调用方（docs/002 §1.3）。
func LoadSnapshot(store core.KVStore) (*Snapshot, error) {
	raw, err := store.Get(SnapshotStoreKey)
	if err != nil || len(raw) == 0 {
		// 只有「存储为空 / 键不存在」才可判定为首次使用；其它读取错误仍按 3003 交给上层。
		if err == nil || errors.Is(err, core.ErrKeyNotFound) {
			return nil, model.WrapError(model.CodeSnapshotMissing,
				"本地无课表快照，请先执行 kemaolife timetable sync", ErrSnapshotAbsent)
		}
		return nil, model.WrapError(model.CodeSnapshotMissing,
			"课表快照读取失败，请重新同步", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, model.WrapError(model.CodeSnapshotMissing,
			"课表快照无法解析，请重新同步", err)
	}
	if snap.Version != SnapshotVersion {
		return nil, model.NewError(model.CodeSnapshotMissing,
			fmt.Sprintf("课表快照版本为 %d，当前需要 %d，请重新同步", snap.Version, SnapshotVersion))
	}
	if snap.Week1Monday.IsZero() || len(snap.SlotTimes) == 0 {
		return nil, model.NewError(model.CodeSnapshotMissing, "课表快照内容不完整，请重新同步")
	}
	return &snap, nil
}

// SaveSnapshot 把快照写入 KVStore。实现方负责 0600 / 加密与原子写。
func SaveSnapshot(store core.KVStore, snap *Snapshot) error {
	if snap == nil {
		return model.NewError(model.CodeInvalidArgument, "快照为空")
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return model.WrapError(model.CodeMalformedResponse, "课表快照编码失败", err)
	}
	if err := store.Put(SnapshotStoreKey, raw); err != nil {
		return model.WrapError(model.CodeNetworkDown, "课表快照写入失败", err)
	}
	return nil
}

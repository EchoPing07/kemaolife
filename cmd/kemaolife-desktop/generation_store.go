package main

import (
	"sync"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// snapshotGeneration 只让当前账号代次改写课表快照。
// context 取消拦不住已经拿到响应的旧同步，所以写盘前还需一次原子代次检查。
type snapshotGeneration struct {
	mu  sync.RWMutex
	gen uint64
}

func (g *snapshotGeneration) next() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gen++
	return g.gen
}

func (g *snapshotGeneration) current() uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.gen
}

type generationSnapshotStore struct {
	inner core.KVStore
	gens  *snapshotGeneration
	gen   uint64
}

func (s generationSnapshotStore) Get(key string) ([]byte, error) { return s.inner.Get(key) }

func (s generationSnapshotStore) Put(key string, value []byte) error {
	if key == timetable.SnapshotStoreKey && s.gens.current() != s.gen {
		return model.NewError(model.CodeNetworkDown, "账号已更新，旧课表写入已取消")
	}
	return s.inner.Put(key, value)
}

func (s generationSnapshotStore) Delete(key string) error {
	if key == timetable.SnapshotStoreKey && s.gens.current() != s.gen {
		return model.NewError(model.CodeNetworkDown, "账号已更新，旧课表删除已取消")
	}
	return s.inner.Delete(key)
}

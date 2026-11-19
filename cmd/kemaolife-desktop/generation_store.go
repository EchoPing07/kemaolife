//go:build windows

package main

import (
	"errors"
	"sync"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// errGenerationSuperseded 是 cmd 层内部哨兵：旧代次写盘被取消。
// 它不进入任何对外信封或用户文案（调用方在代次不符时直接丢弃该错误），
// 因此不复用领域错误码，避免把「账号切换」误报成网络故障。
var errGenerationSuperseded = errors.New("账号已更新，旧课表写入已取消")

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

// withCurrent 在同一把锁内完成代次检查与写动作：
// 检查与写盘之间不允许 next() 插入，否则旧同步仍可能覆盖新账号快照。
func (g *snapshotGeneration) withCurrent(gen uint64, fn func() error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.gen != gen {
		return errGenerationSuperseded
	}
	return fn()
}

type generationSnapshotStore struct {
	inner core.KVStore
	gens  *snapshotGeneration
	gen   uint64
}

func (s generationSnapshotStore) Get(key string) ([]byte, error) { return s.inner.Get(key) }

func (s generationSnapshotStore) Put(key string, value []byte) error {
	if key == timetable.SnapshotStoreKey {
		return s.gens.withCurrent(s.gen, func() error { return s.inner.Put(key, value) })
	}
	return s.inner.Put(key, value)
}

func (s generationSnapshotStore) Delete(key string) error {
	if key == timetable.SnapshotStoreKey {
		return s.gens.withCurrent(s.gen, func() error { return s.inner.Delete(key) })
	}
	return s.inner.Delete(key)
}

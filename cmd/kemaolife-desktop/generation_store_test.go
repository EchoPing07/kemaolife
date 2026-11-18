//go:build windows

package main

import (
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
	"github.com/EchoPing07/kemaolife/internal/core/timetable"
)

// 换账号后旧同步即使已拿到响应，也不能把旧课表写回本地快照。
func TestGenerationSnapshotStoreBlocksStaleWrite(t *testing.T) {
	inner := porttest.NewMemStore()
	gens := &snapshotGeneration{}
	stale := generationSnapshotStore{inner: inner, gens: gens, gen: gens.next()}
	current := generationSnapshotStore{inner: inner, gens: gens, gen: gens.next()}

	if err := current.Put(timetable.SnapshotStoreKey, []byte("new")); err != nil {
		t.Fatalf("当前代次写入失败: %v", err)
	}
	if err := stale.Put(timetable.SnapshotStoreKey, []byte("old")); err == nil {
		t.Fatal("旧代次写入必须被拒绝")
	} else if model.CodeOf(err) != model.CodeNetworkDown {
		t.Errorf("拒绝原因应为网络类错误码, got %v", model.CodeOf(err))
	}

	got, err := inner.Get(timetable.SnapshotStoreKey)
	if err != nil {
		t.Fatalf("读取快照: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("快照被旧代次覆盖: %q", got)
	}
}

// 代次守卫只拦课表快照，配置等其他键照常可写。
func TestGenerationSnapshotStoreOnlyGuardsSnapshot(t *testing.T) {
	inner := porttest.NewMemStore()
	gens := &snapshotGeneration{}
	stale := generationSnapshotStore{inner: inner, gens: gens, gen: gens.next()}
	gens.next()

	if err := stale.Put("desktop/config", []byte("{}")); err != nil {
		t.Fatalf("非快照键不应被代次拦截: %v", err)
	}
}

// 旧代次的删除同样失效，避免换账号清快照时误删新账号刚写入的数据。
func TestGenerationSnapshotStoreBlocksStaleDelete(t *testing.T) {
	inner := porttest.NewMemStore()
	gens := &snapshotGeneration{}
	stale := generationSnapshotStore{inner: inner, gens: gens, gen: gens.next()}
	current := generationSnapshotStore{inner: inner, gens: gens, gen: gens.next()}

	if err := current.Put(timetable.SnapshotStoreKey, []byte("new")); err != nil {
		t.Fatalf("当前代次写入失败: %v", err)
	}
	if err := stale.Delete(timetable.SnapshotStoreKey); err == nil {
		t.Fatal("旧代次删除必须被拒绝")
	}
	if _, err := inner.Get(timetable.SnapshotStoreKey); err != nil {
		t.Fatalf("新账号快照不应被旧代次删除: %v", err)
	}
}

// 读路径必须始终放行：切周只读缓存不能被代次守卫挡住。
func TestGenerationSnapshotStoreAllowsRead(t *testing.T) {
	inner := porttest.NewMemStore()
	if err := inner.Put(timetable.SnapshotStoreKey, []byte("cached")); err != nil {
		t.Fatal(err)
	}
	gens := &snapshotGeneration{}
	stale := generationSnapshotStore{inner: inner, gens: gens, gen: gens.next()}
	gens.next()

	got, err := stale.Get(timetable.SnapshotStoreKey)
	if err != nil {
		t.Fatalf("读缓存必须放行: %v", err)
	}
	if string(got) != "cached" {
		t.Errorf("读到 %q", got)
	}
}

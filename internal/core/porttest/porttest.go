// Package porttest 提供端口的内存实现，供单测与 CLI 组装使用。
//
// 只依赖标准库，不触网、不读系统时间、不写真实文件系统路径（AGENTS §7）。
package porttest

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemStore 是 KVStore 的 map 实现。
type MemStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func NewMemStore() *MemStore {
	return &MemStore{data: make(map[string][]byte)}
}

var ErrKeyNotFound = errors.New("key not found")

func (s *MemStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, nil
}

func (s *MemStore) Put(key string, v []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string][]byte)
	}
	out := make([]byte, len(v))
	copy(out, v)
	s.data[key] = out
	return nil
}

func (s *MemStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// Keys 返回已排序的键，便于断言。
func (s *MemStore) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FakeClock 是可手动拨动的时钟。
type FakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{t: t} }

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// Recorder 记录通知，用于断言「只提醒一次」之类的行为。
type Recorder struct {
	mu    sync.Mutex
	Items []Notification
}

type Notification struct {
	Title string
	Body  string
}

func (r *Recorder) Notify(title, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Items = append(r.Items, Notification{Title: title, Body: body})
	return nil
}

func (r *Recorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Items)
}

func (r *Recorder) Last() (Notification, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.Items) == 0 {
		return Notification{}, false
	}
	return r.Items[len(r.Items)-1], true
}

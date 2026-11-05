package platform

import "time"

// RealClock 是 core.Clock 的真实实现。只有最外层可以用它；
// core / app 内一律通过注入的 Clock 取时间（AGENTS §6）。
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

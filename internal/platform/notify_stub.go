//go:build !windows && !android

package platform

// NoopNotifier 是 Linux 的空通知实现：Linux 不做定时提醒、不发系统通知
// （000 §4 功能矩阵：系统通知 ❌）。
type NoopNotifier struct{}

func (NoopNotifier) Notify(string, string) error { return nil }

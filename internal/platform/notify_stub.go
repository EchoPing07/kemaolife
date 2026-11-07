//go:build !windows && !android

package platform

// NoopNotifier 是非 Windows/Android 的 CLI 平台的空通知实现：
// CLI（linux / darwin）不做定时提醒、不发系统通知
// （000 §4 功能矩阵：系统通知 ❌）。
type NoopNotifier struct{}

func (NoopNotifier) Notify(string, string) error { return nil }

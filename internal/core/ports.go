// Package core 定义平台端口（AGENTS §3.1：「先加端口，再写适配器」）。
//
// 这里只有接口，没有任何实现；实现一律落在 internal/platform，用 build tags 隔离。
package core

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
)

// ErrKeyNotFound 是 KVStore 的「键不存在」标准哨兵。
//
// 为什么需要统一：app 层要区分「从没同步过」（可引导首次同步）与「快照读取失败」
// （必须提示重新同步）。各适配器用 %w 包装本哨兵，core 只判 errors.Is。
var ErrKeyNotFound = errors.New("key not found")

// KVStore 隔离「存哪儿 / 怎么加密」的差异（0600 文件 / DPAPI / Keystore）。
// 实现方负责权限、加密与原子写；失败时不得留下半份数据；键不存在时返回包装了
// ErrKeyNotFound 的错误。
type KVStore interface {
	Get(key string) ([]byte, error)
	Put(key string, v []byte) error
	Delete(key string) error
}

// HTTPDoer 隔离代理、TLS、超时与「强制 IPv4」等传输层细节。
//
// 教务 aic 服务器解析出 AAAA 记录但没有 IPv6 监听，必须由适配器禁用 v6
// （见 api-docs/001-timetable §2、§6），core 不感知。
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// NoRedirectDoer 是 HTTPDoer 的可选能力：发一次请求且**不跟随重定向**。
//
// 为什么需要它：CAS 登录链路跨两个主机（aic → jw），会话 Cookie 在中途的
// 302 响应里下发。若由 http.Client 自动跟随，中间跳的 Set-Cookie 会被丢弃，
// core 就拼不出会话，只能误判为「密码错误」。
//
// 这里不强制所有实现都提供：core 会先做类型断言，取不到时退化为
// 「按 Do 返回值读 Location 自己跟」，只是多一跳开销。
type NoRedirectDoer interface {
	DoNoRedirect(*http.Request) (*http.Response, error)
}

// Clock 让周次、学期边界、告警去重都能在单测里指定时间；core 内禁止直接 time.Now()。
type Clock interface {
	Now() time.Time
}

// Notifier 隔离系统通知（Windows）与 no-op（其余平台）。
type Notifier interface {
	Notify(title, body string) error
}

// NetworkReady 是自动模式的恢复入口：确保网络可用（Windows 自动轮询前置）。
type NetworkReady interface {
	EnsureCampusNetwork(ctx context.Context) error
}

// CampusNetwork 是手动判态 / 登录 / 下线 / 重登的原始操作，
// Windows 手动模式与 Android 手动动作共用同一组原语（AGENTS §3.1）。
//
// 它不等于 NetworkReady：后者是「自动恢复」的组合用例入口。
type CampusNetwork interface {
	Status(ctx context.Context) (model.CampusStatus, error)
	Login(ctx context.Context) error
	Logout(ctx context.Context) error
	Relogin(ctx context.Context) error
}

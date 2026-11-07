//go:build !windows && !android

package platform

// defaultPortalUserAgent 只用于「能编译」：非 Windows/Android 的 CLI 平台
// 不实现校园网认证（000 §3.1），真正被调用的是 campusnet_stub.go 里返回 2001 的桩。
//
// 保留一个默认值是为了让 portal.go 在所有平台都能通过 go vet ./...。
const defaultPortalUserAgent = "Mozilla/5.0"

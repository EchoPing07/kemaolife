//go:build windows

package platform

// defaultPortalUserAgent 是 Windows 端 POST Portal 时使用的 UA。
//
// 实测缺 UA 会被拦（api-docs/003 §2 ②），所以沿用与桌面浏览器一致的形态。
const defaultPortalUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

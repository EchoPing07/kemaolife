//go:build android

package platform

// defaultPortalUserAgent 是 Android 端 POST Portal 时使用的 UA。
//
// 实测缺 UA 会被拦（api-docs/003 §2 ②）。这里写移动端形态而不是桌面形态：
// Portal 侧对移动 UA 的判定更接近用户实际使用场景。
// 版本号固定不带机型 / 系统版本信息，避免把设备指纹写进仓库（P1）。
const defaultPortalUserAgent = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"

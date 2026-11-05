//go:build windows

package platform

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/EchoPing07/kemaolife/internal/core"
)

// Windows 上凭据不能靠文件权限位保护（P5 / R5.3）：os.Chmod 只清只读位，
// 同机任何进程都能读。真正的边界是 DPAPI —— 密钥由**当前用户**登录凭据派生，
// 换个账户（或换台机器）解不开。
//
// 用 syscall 直调 crypt32.dll，不引入 golang.org/x/sys（AGENTS A5）。

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

const (
	// CRYPTPROTECT_UI_FORBIDDEN：后台进程不得弹 UI，失败就返回错误。
	cryptProtectUIFOrbidden = 0x1

	// protectedMagic 让「已加密」可自证：将来若混入明文文件，
	// 解密路径能明确报错而不是把密文当 JSON 解析。
	protectedMagic = "KLM\x01"
)

// dataBlob 对应 Win32 的 DATA_BLOB。
type dataBlob struct {
	size uint32
	data *byte
}

// Protect 用当前用户凭据加密。
func Protect(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, nil
	}
	in := dataBlob{size: uint32(len(plain)), data: &plain[0]}
	var out dataBlob

	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, // szDataDescr
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		cryptProtectUIFOrbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(plain)
	if r == 0 {
		return nil, fmt.Errorf("DPAPI 加密失败: %w", err)
	}
	defer freeBlob(&out)

	buf := make([]byte, 0, len(protectedMagic)+int(out.size))
	buf = append(buf, protectedMagic...)
	buf = append(buf, unsafe.Slice(out.data, out.size)...)
	return buf, nil
}

// Unprotect 用当前用户凭据解密。文件不是 DPAPI 密文时返回错误，
// 不静默当明文使用（否则「加密失败」会伪装成「配置正常」）。
func Unprotect(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, nil
	}
	if len(protected) < len(protectedMagic) || string(protected[:len(protectedMagic)]) != protectedMagic {
		return nil, fmt.Errorf("数据不是本机 DPAPI 密文，拒绝使用")
	}
	body := protected[len(protectedMagic):]
	if len(body) == 0 {
		return nil, fmt.Errorf("DPAPI 密文为空")
	}

	in := dataBlob{size: uint32(len(body)), data: &body[0]}
	var out dataBlob

	r, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, // description out
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		cryptProtectUIFOrbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(body)
	if r == 0 {
		return nil, fmt.Errorf("DPAPI 解密失败（可能换了 Windows 账户）: %w", err)
	}
	defer freeBlob(&out)

	return unsafe.Slice(out.data, out.size), nil
}

// freeBlob 释放 CryptProtectData / CryptUnprotectData 分配的内存（必须 LocalFree）。
func freeBlob(b *dataBlob) {
	if b.data == nil {
		return
	}
	procLocalFree.Call(uintptr(unsafe.Pointer(b.data)))
	b.data = nil
	b.size = 0
}

// EncryptedStore 给任意 KVStore 加一层 DPAPI。
//
// 做成装饰器而不是两个新实现：快照（多键）与配置（单文件）都能复用，
// 加密策略只有一处，测试也只测一处。
type EncryptedStore struct {
	inner core.KVStore
}

// NewEncryptedStore 包装 inner；inner 为 nil 时返回 nil，由调用方报配置错误。
func NewEncryptedStore(inner core.KVStore) *EncryptedStore {
	if inner == nil {
		return nil
	}
	return &EncryptedStore{inner: inner}
}

func (s *EncryptedStore) Get(key string) ([]byte, error) {
	raw, err := s.inner.Get(key)
	if err != nil {
		return nil, err
	}
	return Unprotect(raw)
}

func (s *EncryptedStore) Put(key string, v []byte) error {
	protected, err := Protect(v)
	if err != nil {
		return err
	}
	return s.inner.Put(key, protected)
}

func (s *EncryptedStore) Delete(key string) error { return s.inner.Delete(key) }

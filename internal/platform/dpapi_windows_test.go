//go:build windows

package platform

import (
	"testing"

	"github.com/EchoPing07/kemaolife/internal/core/porttest"
)

// DPAPI 往返：同一用户能解开，换密文前缀会拒绝。
//
// 这是 P5 的核心保证：凭据落盘后只有当前 Windows 用户能读回来。
func TestProtectUnprotectRoundTrip(t *testing.T) {
	plain := []byte(`{"campusPassword":"<密码>","studentId":"<学号>"}`)
	protected, err := Protect(plain)
	if err != nil {
		t.Fatalf("Protect: %v", err)
	}
	if string(protected) == string(plain) {
		t.Fatal("密文不能与明文相同")
	}

	back, err := Unprotect(protected)
	if err != nil {
		t.Fatalf("Unprotect: %v", err)
	}
	if string(back) != string(plain) {
		t.Errorf("解出 = %q, want %q", back, plain)
	}
}

// 空输入保持空（首次启动没有配置文件时不该报错）。
func TestProtectEmpty(t *testing.T) {
	if out, err := Protect(nil); err != nil || out != nil {
		t.Errorf("Protect(nil) = %v, %v", out, err)
	}
	if out, err := Unprotect(nil); err != nil || out != nil {
		t.Errorf("Unprotect(nil) = %v, %v", out, err)
	}
}

// 明文 / 陌生字节必须被拒绝，不能静默当明文用（否则「没加密」会伪装成正常）。
func TestUnprotectRejectsPlaintext(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`{"studentId":"<学号>"}`),
		[]byte("random-garbage"),
		[]byte("KLM\x01"),
	} {
		if _, err := Unprotect(raw); err == nil {
			t.Errorf("应拒绝非本机密文: %q", raw)
		}
	}
}

// 装饰器对上层透明：Put 后取回明文，且内层存的是密文。
func TestEncryptedStore(t *testing.T) {
	inner := porttest.NewMemStore()
	store := NewEncryptedStore(inner)
	if store == nil {
		t.Fatal("NewEncryptedStore 不应返回 nil")
	}

	secret := []byte(`{"campusPassword":"<密码>"}`)
	if err := store.Put("agent/config", secret); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// 内层必须已是密文。
	raw, err := inner.Get("agent/config")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == string(secret) {
		t.Error("内层存储不应是明文")
	}

	back, err := store.Get("agent/config")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(back) != string(secret) {
		t.Errorf("读回 = %q", back)
	}

	if err := store.Delete("agent/config"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := inner.Get("agent/config"); err == nil {
		t.Error("删除应透传到内层")
	}
}

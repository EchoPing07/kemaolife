// Package platform 提供端口适配器。本文件是非 Windows/Android 的 CLI 平台（linux / darwin）的 KVStore：
// 目录权限 0700、文件权限 0600、原子写（P5：凭据只能进平台加密存储）；
// macOS 沿用同一套 POSIX 路径与权限口径，不引 Keychain（AGENTS A5）。
package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/core"
)

// FileStore 是基于文件的 KVStore：每个键一个文件，放在 root 目录下。
type FileStore struct {
	root string
}

// NewFileStore 创建（或复用）目录 store。root 为空时返回错误，由 cmd 决定默认路径。
func NewFileStore(root string) (*FileStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("存储目录为空")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	// MkdirAll 不会改已存在目录的权限；显式收紧，避免历史上是 0755 的目录继续放凭据。
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("收紧存储目录权限失败: %w", err)
	}
	return &FileStore{root: root}, nil
}

func (s *FileStore) Get(key string) ([]byte, error) {
	path, err := s.pathOf(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", core.ErrKeyNotFound, key)
		}
		return nil, err
	}
	return data, nil
}

// Put 原子写：先写同目录临时文件（0600），再 rename 覆盖。
//
// 直接截断写会在进程被杀时留下半份快照 —— 快照损坏会让后续查询全部报 3003。
func (s *FileStore) Put(key string, v []byte) error {
	path, err := s.pathOf(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		return fmt.Errorf("生成临时文件名失败: %w", err)
	}
	tmp := path + ".tmp-" + hex.EncodeToString(suffix)

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	if _, err := f.Write(v); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("同步临时文件失败: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("替换文件失败: %w", err)
	}
	return nil
}

func (s *FileStore) Delete(key string) error {
	path, err := s.pathOf(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// pathOf 把键映射为文件路径，同时防止键逃出 root。
//
// 含 `..` 的键直接拒绝，而不是「clean 成安全路径」：
// 静默改名会让调用方以为写到了 A，实际写在 B。
func (s *FileStore) pathOf(key string) (string, error) {
	normalized := strings.ReplaceAll(key, "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("非法存储键: %q", key)
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("非法存储键: %q", key)
		}
	}

	path := filepath.Join(s.root, filepath.FromSlash(normalized))
	rootAbs, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(pathAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("存储键逃出根目录: %q", key)
	}
	return path, nil
}

// Stat 返回文件权限与大小，供 `config show` 之类的自检使用。
func (s *FileStore) Stat(key string) (os.FileMode, int64, error) {
	path, err := s.pathOf(key)
	if err != nil {
		return 0, 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	return info.Mode().Perm(), info.Size(), nil
}

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// SingleFileStore 是「整份内容存在一个文件里」的 KVStore，用于配置。
//
// 为什么不用 FileStore：docs/002 §0 把配置路径固定为
// `~/.config/kemaolife/config.json`，而不是按 key 分文件。
// 只接受一个键（config），其余键一律报错，避免误用。
type SingleFileStore struct {
	path string
	key  string
}

// NewSingleFileStore 绑定一个文件路径与它唯一接受的键。
func NewSingleFileStore(path, key string) (*SingleFileStore, error) {
	if path == "" || key == "" {
		return nil, fmt.Errorf("配置路径或键为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}
	return &SingleFileStore{path: path, key: key}, nil
}

func (s *SingleFileStore) Get(key string) ([]byte, error) {
	if key != s.key {
		return nil, fmt.Errorf("配置存储只接受键 %q", s.key)
	}
	return os.ReadFile(s.path)
}

func (s *SingleFileStore) Put(key string, v []byte) error {
	if key != s.key {
		return fmt.Errorf("配置存储只接受键 %q", s.key)
	}
	return WriteFileAtomic(s.path, v, 0o600)
}

func (s *SingleFileStore) Delete(key string) error {
	if key != s.key {
		return fmt.Errorf("配置存储只接受键 %q", s.key)
	}
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Path 返回配置文件的绝对路径，供 `config show` 展示。
func (s *SingleFileStore) Path() string { return s.path }

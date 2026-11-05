package electricity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// flexBool 兼容上游两种写法：`"result_":"true"` 与 `"result_":true`。
//
// api-docs/002 §4 记录的是字符串形式，但实测上游在拒绝分支会给布尔值
// （`{"code_":8099,"result_":false,...}`）。用 bool 直接解码会让整个响应
// 解析失败并被误报成「响应格式异常」，把配置问题伪装成协议问题。
type flexBool bool

func (b *flexBool) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*b = false
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "true_":
			*b = true
		default:
			*b = false
		}
		return nil
	}
	var v bool
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return err
	}
	*b = flexBool(v)
	return nil
}

func (b flexBool) MarshalJSON() ([]byte, error) {
	if bool(b) {
		return []byte(`"true"`), nil
	}
	return []byte(`"false"`), nil
}

// flexString 兼容 `"body":"{...}"` 与 `"body":{...}`（对象直接内联）。
//
// 实测上游在拒绝分支可能没有 body，或直接给对象；两种情况都不应该
// 让整个响应解析失败。
type flexString string

func (s *flexString) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*s = ""
		return nil
	}
	if trimmed[0] == '"' {
		var str string
		if err := json.Unmarshal(trimmed, &str); err != nil {
			return err
		}
		*s = flexString(str)
		return nil
	}
	// 对象 / 数组：原样保存，后续按 JSON 解析。
	*s = flexString(trimmed)
	return nil
}

// flexInt 兼容 `"code_":0` 与 `"code_":"0"`。
type flexInt int

func (n *flexInt) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*n = 0
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		var v int
		if _, err := fmtSscan(strings.TrimSpace(s), &v); err != nil {
			*n = 0
			return nil
		}
		*n = flexInt(v)
		return nil
	}
	var v int
	if err := json.Unmarshal(trimmed, &v); err != nil {
		*n = 0
		return nil
	}
	*n = flexInt(v)
	return nil
}

// fmtSscan 是 fmt.Sscan 的极薄包装，避免在本文件再引入 fmt 时与其他文件冲突。
func fmtSscan(s string, v *int) (int, error) {
	return fmt.Sscan(s, v)
}

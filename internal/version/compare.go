package version

import (
	"cmp"
	"strconv"
	"strings"
)

// Number 是三段十进制发布版本号（release.yml 限定 tag 为 vX.Y.Z）。
type Number struct {
	Major, Minor, Patch int
}

// Parse 解析 `vX.Y.Z` 或 `X.Y.Z`；`v` 前缀可选，三段都必须是十进制数。
// 非法格式（含溢出、附加后缀）一律返回 false，由调用方归入检查失败。
func Parse(raw string) (Number, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Number{}, false
	}
	var nums [3]int
	for i, p := range parts {
		if p == "" {
			return Number{}, false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return Number{}, false
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return Number{}, false
		}
		nums[i] = n
	}
	return Number{Major: nums[0], Minor: nums[1], Patch: nums[2]}, true
}

// Compare 按数值逐段比较：字符串字典序会把 0.10.0 判成比 0.9.0 旧。
func Compare(a, b Number) int {
	if c := cmp.Compare(a.Major, b.Major); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Minor, b.Minor); c != 0 {
		return c
	}
	return cmp.Compare(a.Patch, b.Patch)
}

func (v Number) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}

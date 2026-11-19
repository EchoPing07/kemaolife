package version_test

import (
	"testing"

	"github.com/EchoPing07/kemaolife/internal/version"
)

func TestParseAcceptsPlainAndVPrefix(t *testing.T) {
	for _, raw := range []string{"0.5.2", "v0.5.2", " v0.5.2 "} {
		v, ok := version.Parse(raw)
		if !ok || v != (version.Number{Major: 0, Minor: 5, Patch: 2}) {
			t.Errorf("Parse(%q) = %v, %v，期望 0.5.2", raw, v, ok)
		}
	}
	v, _ := version.Parse("v0.5.2")
	if v.String() != "0.5.2" {
		t.Errorf("String() = %q，期望重排为无前缀的 0.5.2", v.String())
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	for _, raw := range []string{
		"", "0.5", "0.5.2.1", "0.5.x", "v", "0.5.2-beta", "0.5.2+build",
		"-1.0.0", "0.-1.0", "0.0.-2", "99999999999999999999.0.0", "0..2", "abc",
	} {
		if _, ok := version.Parse(raw); ok {
			t.Errorf("Parse(%q) 不应接受", raw)
		}
	}
}

func TestCompareIsNumericNotLexicographic(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.10.0", "0.9.0", 1},
		{"0.9.0", "0.10.0", -1},
		{"1.0.0", "0.99.99", 1},
		{"0.5.3", "0.5.2", 1},
		{"0.5.2", "0.5.2", 0},
		{"v0.5.2", "0.5.2", 0},
		{"0.5.2", "0.5.3", -1},
	}
	for _, tc := range cases {
		a, okA := version.Parse(tc.a)
		b, okB := version.Parse(tc.b)
		if !okA || !okB {
			t.Fatalf("夹具版本号非法: %q / %q", tc.a, tc.b)
		}
		if got := version.Compare(a, b); got != tc.want {
			t.Errorf("Compare(%q, %q) = %d，期望 %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// 本机版本不低于远端时不得提示更新（同版本与降级发布都算「无更新」）。
func TestCompareNoUpdateWhenNotNewer(t *testing.T) {
	local, _ := version.Parse("0.5.5")
	for _, tag := range []string{"v0.5.5", "0.5.4", "0.4.9"} {
		remote, _ := version.Parse(tag)
		if version.Compare(remote, local) > 0 {
			t.Errorf("%s 不应判为比 0.5.5 新", tag)
		}
	}
}

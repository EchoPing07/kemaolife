//go:build windows

package ui

import "testing"

// 设置页是单列布局：控件必须完整落在内容区横向范围内。
func TestSettingsLayoutFitsContentWidth(t *testing.T) {
	cases := []struct {
		name        string
		left, right int32
	}{
		{"default", 216, 1092},
		{"minimum", 216, 932},
		{"narrow", 216, 700},
	}
	identity := func(dp int32) int32 { return dp }
	for _, tc := range cases {
		g := settingsGeometryForWidth(tc.left, tc.right, identity)
		if g.leftX < tc.left || g.leftX+g.colW > tc.right {
			t.Errorf("%s: 列越界 left=%d colW=%d right=%d", tc.name, g.leftX, g.colW, tc.right)
		}
		if g.labelW >= g.colW {
			t.Errorf("%s: 标签宽 %d 必须小于列宽 %d", tc.name, g.labelW, g.colW)
		}
	}
}

// 纵向顺序必须单调递增，否则控件会互相叠压。
func TestSettingsLayoutVerticalOrder(t *testing.T) {
	identity := func(dp int32) int32 { return dp }
	g := settingsGeometryForWidth(216, 1092, identity)

	last := int32(0)
	for _, y := range []int32{
		g.eduTop, g.eduRows[0], g.eduRows[1], g.eduRows[2],
		g.campusTop, g.campusRows[0], g.campusRows[1], g.campusRows[2],
		g.modeTop, g.modeRows[0], g.modeRows[1], g.modeRows[2],
		g.runTop, g.runRows[0], g.runRows[3], g.quietY,
		g.saveY, g.hintY,
	} {
		if y <= last {
			t.Fatalf("纵向坐标非递增: %d 出现在 %d 之后", y, last)
		}
		last = y
	}
	if g.contentH <= g.hintY {
		t.Errorf("内容高度应大于提示行: contentH=%d hintY=%d", g.contentH, g.hintY)
	}
}

// 滚动范围：矮窗口必须可滚、高窗口不滚、退化视口不产生负值。
func TestSettingsScrollMetrics(t *testing.T) {
	identity := func(dp int32) int32 { return dp }
	g := settingsGeometryForWidth(216, 1092, identity)
	contentBottom := g.quietY + 30

	// 矮窗口（底部按钮固定在 saveY=500）：表单底部被视口裁掉，必须能滚。
	viewport, maxScroll := settingsScrollMetrics(96, 492, contentBottom)
	if viewport != 396 {
		t.Errorf("视口高应为 396, got %d", viewport)
	}
	if maxScroll <= 0 {
		t.Errorf("矮窗口 maxScroll 应 > 0, got %d", maxScroll)
	}
	if maxScroll != contentBottom-492 {
		t.Errorf("maxScroll 应恰好补齐被裁掉的 %d, got %d", contentBottom-492, maxScroll)
	}

	// 高窗口：表单完整可见，不需要滚动。
	viewport, maxScroll = settingsScrollMetrics(96, contentBottom+96, contentBottom)
	if maxScroll != 0 {
		t.Errorf("高窗口 maxScroll 应为 0, got %d", maxScroll)
	}
	if viewport != contentBottom {
		t.Errorf("视口高应为 %d, got %d", contentBottom, viewport)
	}

	// 退化视口（bottom < top）：钳到 1，不得返回负数或除零。
	viewport, _ = settingsScrollMetrics(96, 90, contentBottom)
	if viewport != 1 {
		t.Errorf("退化视口应钳到 1, got %d", viewport)
	}
}

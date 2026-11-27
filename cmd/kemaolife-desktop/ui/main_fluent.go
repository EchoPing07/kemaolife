//go:build windows

package ui

import (
	"time"

	myui "github.com/egoist/mygo/ui"
)

// fluentButton 构建 Windows 11 Standard Button（4px 圆角 + 微浮雕渐变边框）。
func fluentButton(c *myui.Context, key any, label string) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	btn := myui.ButtonBase(c.Key(key)).Height(32).Padding(0, 12).Gap(6).Radius(4).
		Center().Background(p.controlBg).TextColor(t.Text).BorderWidth(1).
		BorderLinearGradient(myui.LinearGradient{
			From: p.controlBorderTop, To: p.controlBorderBottom, Angle: 180,
		})
	if btn.Pressed() {
		btn.Background(p.controlPressed).BorderLinearGradient(myui.LinearGradient{
			From: p.controlBorderTop, To: p.controlBorderTop, Angle: 180,
		})
	} else if btn.Hovered() {
		btn.Background(p.controlHover)
	}
	if label != "" {
		btn.Children(func() {
			myui.Text(c, label).FontSize(13).TextColor(t.Text).SingleLine()
		})
	}
	return btn
}

// fluentPrimaryButton 构建 Windows 11 Accent Button（强调色填充 + 顶部微高光底边微暗）。
func fluentPrimaryButton(c *myui.Context, key any, label string) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	bg := t.Accent
	fg := t.AccentText
	if bg == myui.Transparent {
		bg = myui.Hex("#0067c0")
	}
	if fg == myui.Transparent {
		fg = myui.RGB(255, 255, 255)
	}
	btn := myui.ButtonBase(c.Key(key)).Height(32).Padding(0, 14).Gap(6).Radius(4).
		Center().Background(bg).TextColor(fg).BorderWidth(1).
		BorderLinearGradient(myui.LinearGradient{
			From: p.accentBorderTop, To: p.accentBorderBottom, Angle: 180,
		})
	if btn.Pressed() {
		btn.Background(p.accentPressed).BorderLinearGradient(myui.LinearGradient{
			From: p.accentBorderTop, To: p.accentBorderTop, Angle: 180,
		})
	} else if btn.Hovered() {
		btn.Background(p.accentHover)
	}
	if label != "" {
		btn.Children(func() {
			myui.Text(c, label).FontSize(13).FontWeight(500).TextColor(fg).SingleLine()
		})
	}
	return btn
}

// fluentSubtleButton 构建 Windows 11 Subtle / Transparent 按钮（无底色，悬停出现微弱浮层）。
func fluentSubtleButton(c *myui.Context, key any, label string) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	btn := myui.ButtonBase(c.Key(key)).Height(30).Padding(0, 10).Gap(6).Radius(4).
		Center().TextColor(t.Text)
	if btn.Pressed() {
		btn.Background(p.navPressed)
	} else if btn.Hovered() {
		btn.Background(p.navHover)
	}
	if label != "" {
		btn.Children(func() {
			myui.Text(c, label).FontSize(12).TextColor(t.Text).SingleLine()
		})
	}
	return btn
}

// fluentIconButton 构建 32x32 的纯图标工具按钮（标题栏折叠按钮、命令栏箭头等）。
func fluentIconButton(c *myui.Context, key any, icon *myui.SVG, label string, subtle bool) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	btn := myui.ButtonBase(c.Key(key)).Size(32, 32).Radius(4).Center().Label(label).Tooltip(label)
	if !subtle {
		btn.Background(p.controlBg).BorderWidth(1).
			BorderLinearGradient(myui.LinearGradient{
				From: p.controlBorderTop, To: p.controlBorderBottom, Angle: 180,
			})
		if btn.Pressed() {
			btn.Background(p.controlPressed)
		} else if btn.Hovered() {
			btn.Background(p.controlHover)
		}
	} else {
		if btn.Pressed() {
			btn.Background(p.navPressed)
		} else if btn.Hovered() {
			btn.Background(p.navHover)
		}
	}
	btn.Children(func() {
		myui.Icon(c, icon).FontSize(15).TextColor(t.Text)
	})
	return btn
}

// fluentTextInput 构建 Windows 11 TextBox（4px 圆角 + 静息 1px 底线 + 聚焦 2px 主题色底线）。
func fluentTextInput(c *myui.Context, key any, value *string) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	in := myui.TextInputBase(c.Key(key), value).Height(32).Padding(6, 10).Radius(4).
		Background(p.inputBg).Border(1, p.inputBorder).FocusRing(false)
	focused := in.Focused()
	if focused {
		in.Background(p.inputFocus)
	} else if in.Hovered() {
		in.Background(p.inputHover)
	}
	bottomRest := p.inputBottomRest
	accent := t.Accent
	in.DrawOver(func(pt *myui.Painter, r myui.Rect) {
		if focused {
			pt.FillRadii(myui.Rect{X: r.X + 1, Y: r.Y + r.H - 2, W: maxFloat(0, r.W-2), H: 2}, [4]float32{0, 0, 3, 3}, accent)
		} else {
			pt.Line(r.X+2, r.Y+r.H-0.5, r.X+r.W-2, r.Y+r.H-0.5, 1, bottomRest)
		}
	})
	return in
}

// fluentSwitch 构建 Windows 11 ToggleSwitch（40x20 胶囊，关态透明+描边，开态主题色填充）。
func fluentSwitch(c *myui.Context, key any, on *bool, label string) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	sw := myui.SwitchBase(c.Key(key), on).Size(40, 20).Radius(10).FocusRing(false).Label(label)
	pos := sw.Animate("knob", boolFloat(*on), 140*time.Millisecond)
	hover := sw.Animate("hover", boolFloat(sw.Hovered()), 100*time.Millisecond)
	pressed := sw.Animate("press", boolFloat(sw.Pressed()), 80*time.Millisecond)

	offBg := myui.Transparent
	if sw.Hovered() {
		offBg = p.navHover
	}
	onBg := t.Accent
	if sw.Pressed() {
		onBg = p.accentPressed
	} else if sw.Hovered() {
		onBg = p.accentHover
	}
	bg := offBg.Mix(onBg, pos)
	borderColor := p.inputBottomRest.Mix(myui.Transparent, pos)

	sw.Background(bg)
	if pos < 0.99 {
		sw.Border(1, borderColor)
	}

	knobOff := t.TextMuted
	if sw.Hovered() {
		knobOff = t.Text
	}
	knobOn := t.AccentText
	if knobOn == myui.Transparent {
		knobOn = myui.RGB(255, 255, 255)
	}
	knobColor := knobOff.Mix(knobOn, pos)

	sw.Draw(func(pt *myui.Painter, r myui.Rect) {
		diameter := 12 + 2*hover
		width := diameter + 3*pressed
		pad := (r.H - diameter) / 2
		x := r.X + pad + pos*(r.W-2*pad-width)
		y := r.Y + (r.H-diameter)/2
		pt.Fill(myui.Rect{X: x, Y: y, W: width, H: diameter}, knobColor, diameter/2)
	})
	return sw
}

// fluentSegmented 构建 Windows 11 风格分段切换器，保留独立文本以便单测点击命中。
func fluentSegmented(c *myui.Context, key any, selected *int, labels ...string) myui.Element {
	p := mainPalette(c)
	t := c.Theme()
	parts := myui.SegmentedBase(c.Key(key), selected, len(labels))
	track := parts.Track
	track.AlignItems(myui.Stretch).Padding(3).Gap(3).Radius(6).
		Background(p.cardSubtle).Border(1, p.hairline)

	track.Children(func() {
		for i, label := range labels {
			active := *selected == i
			seg := parts.Segment(i).Height(26).Padding(0, 12).Radius(4)
			fg := t.TextMuted
			if active {
				fg = t.Text
				seg.Background(p.card).Border(1, p.cardBorder).Shadow(0, 1, 2, 0, p.cardShadow)
			} else if seg.Pressed() {
				seg.Background(p.navPressed)
				fg = t.Text
			} else if seg.Hovered() {
				seg.Background(p.navHover)
				fg = t.Text
			}
			seg.Children(func() {
				lbl := myui.Text(c, label).FontSize(12).TextColor(fg).SingleLine()
				if active {
					lbl.FontWeight(600)
				}
			})
		}
	})
	return track
}

func boolFloat(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

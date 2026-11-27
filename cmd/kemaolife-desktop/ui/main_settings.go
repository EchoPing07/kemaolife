//go:build windows

package ui

import (
	"strconv"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/desktop"
	myui "github.com/egoist/mygo/ui"
)

var themeModeValues = []string{desktop.ThemeSystem, desktop.ThemeLight, desktop.ThemeDark}

func themeModeIndex(mode string) int {
	norm := desktop.NormalizeThemeMode(mode)
	for i, v := range themeModeValues {
		if v == norm {
			return i
		}
	}
	return 0
}

func (w *mainWindow) setSettingsForm(form desktop.SettingsForm) {
	w.settingsBase = form
	w.settings = desktop.SettingsResult{
		StudentID:               form.StudentID,
		CustomerCode:            form.CustomerCode,
		CampusUser:              form.CampusUser,
		CampusFingerprint:       form.CampusFingerprint,
		AutoMode:                form.AutoMode,
		WiredOnly:               form.WiredOnly,
		AutoStart:               form.AutoStart,
		AutoUpdateCheck:         form.AutoUpdateCheck,
		ThemeMode:               desktop.NormalizeThemeMode(form.ThemeMode),
		OptimizeLocationDisplay: form.OptimizeLocationDisplay,
		PrefixLocationAt:        form.PrefixLocationAt,
		HideTimetableOverflow:   form.HideTimetableOverflow,
		PollSeconds:             strconv.Itoa(form.PollSeconds),
		CampusProbeSecs:         strconv.Itoa(form.CampusProbeSecs),
		BalanceThreshold:        form.BalanceThreshold,
		DaysThreshold:           strconv.Itoa(form.DaysThreshold),
		QuietFrom:               strconv.Itoa(form.QuietFrom),
		QuietTo:                 strconv.Itoa(form.QuietTo),
	}
	// 密码始终为空：留空表示保留已保存凭据。
	w.settings.EduPassword = ""
	w.settings.CampusPassword = ""
	w.settingsDirty = false
	w.settingsHint = ""
}

func (w *mainWindow) paintSettings(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	myui.Column(c).Fill().Children(func() {
		myui.Column(c).Shrink(0).FillWidth().Padding(22, 28, 12, 28).Children(func() {
			pageHeader(c, "设置", "外观、账号、认证、轮询与提醒；保存后立即生效", nil)
		})

		// 流式贴合内容区全宽，随窗口缩放自适应伸缩，左右与页眉及底栏严格对齐。
		myui.Scroll(c.Key("settings-scroll")).Grow(1).Height(0).FillWidth().Padding(2, 28, 22, 28).Children(func() {
			myui.Column(c).FillWidth().Gap(20).Children(func() {
				settingsGroup(c, "外观", func() {
					themeIdx := themeModeIndex(w.settings.ThemeMode)
					settingsRowCard(c, mainIcons.palette, "应用主题", "选择跟随 Windows 系统外观，或固定使用浅色 / 深色主题", func() {
						if fluentSegmented(c, "theme-mode", &themeIdx, "跟随系统", "浅色", "深色").Label("应用主题").Changed() {
							w.settings.ThemeMode = themeModeValues[themeIdx]
							w.settingsDirty = true
							w.settingsHint = ""
						}
					})
				})

				settingsGroup(c, "教务与电费", func() {
					w.settingsInputRow(c, mainIcons.user, "student-id", "学号", "用于电费与课表查询", &w.settings.StudentID, "请输入学号", false, 240)
					w.settingsInputRow(c, mainIcons.key, "edu-password", "教务密码", "仅用于课表同步，留空保持已保存密码不变", &w.settings.EduPassword, "留空保持不变", true, 240)
					w.settingsInputRow(c, mainIcons.building, "customer-code", "学校代码", "留空使用内置默认 3126", &w.settings.CustomerCode, "默认 3126", false, 240)
				})

				settingsGroup(c, "课表显示", func() {
					w.settingsSwitchRow(c, mainIcons.layout, "optimize-location-display", "课室显示优化", "在课表网格与今日课节精简冗长的课室名称，课程详情仍显示完整地点", &w.settings.OptimizeLocationDisplay)
					w.settingsSwitchRow(c, mainIcons.pin, "prefix-location-at", "课室前显示@", "在课表网格与今日课节的课室名称前加上 @ 符号，方便与课程名和教师区分", &w.settings.PrefixLocationAt)
					w.settingsSwitchRow(c, mainIcons.bookOpen, "hide-timetable-overflow", "过长隐藏", "开启后课表网格与今日课节按行数截断过长文本；关闭时完整换行展示", &w.settings.HideTimetableOverflow)
				})

				settingsGroup(c, "校园网认证", func() {
					w.settingsInputRow(c, mainIcons.user, "campus-user", "校园网账号", "独立凭据，不复用教务密码", &w.settings.CampusUser, "请输入校园网账号", false, 240)
					w.settingsInputRow(c, mainIcons.key, "campus-password", "校园网密码", "留空保持已保存密码不变", &w.settings.CampusPassword, "留空保持不变", true, 240)
					w.settingsInputRow(c, mainIcons.shield, "campus-fingerprint", "指纹参数", "可选参数，无特殊需要可留空", &w.settings.CampusFingerprint, "可留空", false, 240)
				})

				settingsGroup(c, "运行与启动", func() {
					w.settingsSwitchRow(c, mainIcons.network, "auto-mode", "未认证时自动登录", "默认手动；开启后每轮电费轮询前自动尝试校园网认证", &w.settings.AutoMode)
					w.settingsSwitchRow(c, mainIcons.cable, "wired-only", "自动认证仅限有线网络", "主出口为 Wi-Fi 时跳过自动前置，避免与宿舍路由器互顶", &w.settings.WiredOnly)
					w.settingsSwitchRow(c, mainIcons.power, "auto-start", "开机登录时自启动一次", "仅驻留系统托盘，不保活、不注册守护服务", &w.settings.AutoStart)
					w.settingsSwitchRow(c, mainIcons.update, "auto-update", "启动时自动检查更新", "仅查询公开 Releases 版本号，不自动下载或安装", &w.settings.AutoUpdateCheck)
				})

				settingsGroup(c, "轮询与提醒", func() {
					w.settingsInputRow(c, mainIcons.timer, "poll-seconds", "电费轮询间隔（秒）", "默认 14400 秒（4 小时），低于 600 秒会在保存时钳制", &w.settings.PollSeconds, "14400", false, 140)
					w.settingsInputRow(c, mainIcons.clock, "probe-seconds", "校园网探测间隔（秒）", "默认 120 秒，低于 30 秒会在保存时钳制", &w.settings.CampusProbeSecs, "120", false, 140)
					w.settingsInputRow(c, mainIcons.bell, "balance-threshold", "余额提醒阈值（度）", "剩余电费低于等于此值时触发系统通知，默认 10 度", &w.settings.BalanceThreshold, "10", false, 140)
					w.settingsInputRow(c, mainIcons.calendarEvent, "days-threshold", "可用天数阈值（天）", "预计可用天数低于等于此值时触发提醒，默认 3 天", &w.settings.DaysThreshold, "3", false, 140)
					settingsRowCard(c, mainIcons.moonClock, "静默时段（时）", "静默时段内不发送通知，也不置位告警标记", func() {
						myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
							w.settingsInputControl(c, "quiet-from", &w.settings.QuietFrom, "23", false).Width(64)
							myui.Text(c, "至").FontSize(12).TextColor(t.TextMuted)
							w.settingsInputControl(c, "quiet-to", &w.settings.QuietTo, "7", false).Width(64)
						})
					})
				})
			})
		})

		// 固定底部操作栏：透明底 + 细顶线，与通铺 Mica 一体融合。
		myui.Row(c).Shrink(0).FillWidth().Padding(12, 28).Gap(10).AlignItems(myui.Center).
			BorderWidth(1, 0, 0, 0).BorderColor(p.hairline).Children(func() {
			if fluentPrimaryButton(c, "settings-save", "保存").Clicked() {
				w.saveSettings()
			}
			if fluentButton(c, "settings-revert", "还原").Clicked() {
				w.setSettingsForm(w.state.SettingsForm)
			}
			if w.settingsDirty {
				mainBadge(c, "有未保存更改", t.Text, p.cardSubtle, true)
			}
			hint := w.settingsHint
			if hint == "" {
				hint = w.state.SettingsHint
			}
			if hint != "" {
				myui.Text(c, hint).FontSize(12).TextColor(t.TextMuted).Selectable().Grow(1).MinWidth(0).Padding(0, 4)
			}
		})
	})
}

func settingsGroup(c *myui.Context, title string, items func()) {
	myui.Column(c).FillWidth().Gap(6).Children(func() {
		myui.Text(c, title).FontSize(13).FontWeight(600).Padding(0, 0, 4, 2)
		myui.Column(c).FillWidth().Gap(4).Children(items)
	})
}

func settingsRowCard(c *myui.Context, icon *myui.SVG, title, desc string, control func()) {
	p := mainPalette(c)
	t := c.Theme()
	row := myui.Row(c).FillWidth().MinHeight(60).Padding(12, 18).Gap(14).Radius(6).Background(p.card).
		Border(1, p.cardBorder).AlignItems(myui.Center)
	row.Children(func() {
		if icon != nil {
			myui.Icon(c, icon).FontSize(18).TextColor(t.Text).Shrink(0)
		}
		myui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			myui.Text(c, title).FontSize(14).TextColor(t.Text)
			if desc != "" {
				myui.Text(c, desc).FontSize(12).TextColor(t.TextMuted)
			}
		})
		if control != nil {
			myui.Row(c).Shrink(0).AlignItems(myui.Center).Children(control)
		}
	})
}

func (w *mainWindow) settingsInputRow(c *myui.Context, icon *myui.SVG, key, title, desc string, value *string, placeholder string, password bool, width float32) {
	windowW, _ := c.Size()
	contentW := windowW - w.sidebarWidth() - 56
	if contentW < 740 && width > 200 {
		width = 200
	}
	settingsRowCard(c, icon, title, desc, func() {
		w.settingsInputControl(c, key, value, placeholder, password).Width(width)
	})
}

func (w *mainWindow) settingsInputControl(c *myui.Context, key string, value *string, placeholder string, password bool) myui.Element {
	input := fluentTextInput(c, key, value).Placeholder(placeholder)
	if password {
		input = input.Password()
	}
	if input.Changed() {
		w.settingsDirty = true
		w.settingsHint = ""
	}
	return input
}

func (w *mainWindow) settingsSwitchRow(c *myui.Context, icon *myui.SVG, key, title, desc string, value *bool) {
	t := c.Theme()
	settingsRowCard(c, icon, title, desc, func() {
		myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
			myui.Text(c, mainBoolText(*value, "开", "关")).FontSize(12).TextColor(t.TextMuted)
			if fluentSwitch(c, key, value, title).Changed() {
				w.settingsDirty = true
				w.settingsHint = ""
			}
		})
	})
}

func (w *mainWindow) saveSettings() {
	if w.handler.OnSaveSettings == nil {
		return
	}
	result := w.settings
	result.StudentID = strings.TrimSpace(result.StudentID)
	result.CustomerCode = strings.TrimSpace(result.CustomerCode)
	result.CampusUser = strings.TrimSpace(result.CampusUser)
	result.CampusFingerprint = strings.TrimSpace(result.CampusFingerprint)
	result.ThemeMode = desktop.NormalizeThemeMode(result.ThemeMode)
	result.PollSeconds = strings.TrimSpace(result.PollSeconds)
	result.CampusProbeSecs = strings.TrimSpace(result.CampusProbeSecs)
	result.BalanceThreshold = strings.TrimSpace(result.BalanceThreshold)
	result.DaysThreshold = strings.TrimSpace(result.DaysThreshold)
	result.QuietFrom = strings.TrimSpace(result.QuietFrom)
	result.QuietTo = strings.TrimSpace(result.QuietTo)
	hint, form, saved := w.handler.OnSaveSettings(result)
	w.settingsHint = hint
	if saved {
		w.setSettingsForm(form)
		w.settingsHint = hint
	}
}

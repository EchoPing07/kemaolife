//go:build windows

package ui

import (
	"strconv"
	"strings"

	"github.com/EchoPing07/kemaolife/internal/desktop"
	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) setSettingsForm(form desktop.SettingsForm) {
	w.settingsBase = form
	w.settings = desktop.SettingsResult{
		StudentID:         form.StudentID,
		CustomerCode:      form.CustomerCode,
		CampusUser:        form.CampusUser,
		CampusFingerprint: form.CampusFingerprint,
		AutoMode:          form.AutoMode,
		WiredOnly:         form.WiredOnly,
		AutoStart:         form.AutoStart,
		AutoUpdateCheck:   form.AutoUpdateCheck,
		PollSeconds:       strconv.Itoa(form.PollSeconds),
		CampusProbeSecs:   strconv.Itoa(form.CampusProbeSecs),
		BalanceThreshold:  form.BalanceThreshold,
		DaysThreshold:     strconv.Itoa(form.DaysThreshold),
		QuietFrom:         strconv.Itoa(form.QuietFrom),
		QuietTo:           strconv.Itoa(form.QuietTo),
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
		myui.Column(c).Padding(24, 24, 16, 24).Children(func() {
			pageHeader(c, "设置", "账号、认证、轮询与提醒；保存后立即生效", nil)
		})
		myui.Scroll(c.Key("settings-scroll")).Grow(1).Padding(0, 24, 18, 24).Gap(16).Children(func() {
			myui.Grid(c).Columns(2).Gap(16).AlignItems(myui.Stretch).Children(func() {
				mainCard(c).Children(func() {
					mainSectionTitle(c, "教务与电费", "教务密码只用于课表同步")
					myui.Form(c, func() {
						w.settingsInput(c, "student-id", "学号", &w.settings.StudentID, "用于电费与课表查询", false)
						w.settingsInput(c, "edu-password", "教务密码", &w.settings.EduPassword, "留空保持不变", true)
						w.settingsInput(c, "customer-code", "学校代码", &w.settings.CustomerCode, "留空使用内置默认 3126", false)
					})
				})
				mainCard(c).Children(func() {
					mainSectionTitle(c, "校园网认证", "独立凭据")
					myui.Form(c, func() {
						w.settingsInput(c, "campus-user", "账号", &w.settings.CampusUser, "校园网账号", false)
						w.settingsInput(c, "campus-password", "密码", &w.settings.CampusPassword, "留空保持不变", true)
						w.settingsInput(c, "campus-fingerprint", "指纹参数", &w.settings.CampusFingerprint, "可留空", false)
					})
				})
				mainCard(c).Children(func() {
					mainSectionTitle(c, "运行模式", "")
					myui.Column(c).Gap(12).Children(func() {
						w.settingsCheck(c, "auto-mode", "未认证时自动登录（默认手动）", &w.settings.AutoMode)
						w.settingsCheck(c, "wired-only", "自动认证仅限有线网络", &w.settings.WiredOnly)
						w.settingsCheck(c, "auto-start", "开机登录时自启动一次", &w.settings.AutoStart)
						w.settingsCheck(c, "auto-update", "启动时自动检查更新", &w.settings.AutoUpdateCheck)
					})
				})
				mainCard(c).Children(func() {
					mainSectionTitle(c, "版本更新", "只检查，不自动安装")
					label := "检查更新"
					if w.state.UpdateChecking {
						label = "检查中…"
					}
					if myui.Button(c.Key("check-update"), label).Disabled(w.state.UpdateChecking).Clicked() &&
						w.handler.OnCheckUpdate != nil {
						w.handler.OnCheckUpdate()
					}
					if w.state.UpdateHint != "" {
						myui.Text(c, w.state.UpdateHint).FontSize(12).TextColor(t.TextMuted).Selectable()
					}
				})
			})
			mainCard(c).Children(func() {
				mainSectionTitle(c, "轮询与提醒", "低于下限的间隔会在保存时钳制")
				myui.Grid(c).Columns(2).GapX(28).GapY(8).Children(func() {
					myui.Form(c, func() {
						w.settingsInput(c, "poll-seconds", "电费间隔（秒）", &w.settings.PollSeconds, "默认 14400，最低 600", false)
						w.settingsInput(c, "probe-seconds", "探测间隔（秒）", &w.settings.CampusProbeSecs, "默认 120，最低 30", false)
						w.settingsInput(c, "balance-threshold", "余额阈值（度）", &w.settings.BalanceThreshold, "默认 10", false)
					})
					myui.Form(c, func() {
						w.settingsInput(c, "days-threshold", "可用天数（天）", &w.settings.DaysThreshold, "默认 3", false)
						myui.Field(c, "静默时段（时）", func() {
							myui.Row(c).Gap(8).Children(func() {
								w.settingsInputControl(c, "quiet-from", &w.settings.QuietFrom, "23", false).Width(70)
								myui.Text(c, "至").TextColor(t.TextMuted)
								w.settingsInputControl(c, "quiet-to", &w.settings.QuietTo, "7", false).Width(70)
							})
						}).Description("静默时段内不通知，也不置位告警标记")
					})
				})
			})
		})
		myui.Column(c).Padding(14, 24).Gap(10).Background(p.canvas).
			BorderWidth(1, 0, 0, 0).BorderColor(p.border).Children(func() {
			myui.Row(c).Gap(10).Children(func() {
				if myui.PrimaryButton(c.Key("settings-save"), "保存").Clicked() {
					w.saveSettings()
				}
				if myui.Button(c.Key("settings-revert"), "还原").Clicked() {
					w.setSettingsForm(w.state.SettingsForm)
				}
				hint := w.settingsHint
				if hint == "" {
					hint = w.state.SettingsHint
				}
				if hint != "" {
					myui.Text(c, hint).FontSize(12).TextColor(t.TextMuted).Selectable().Padding(4, 4, 0, 4)
				}
			})
		})
	})
}

func (w *mainWindow) settingsInput(c *myui.Context, key, label string, value *string, placeholder string, password bool) {
	myui.Field(c, label, func() {
		w.settingsInputControl(c, key, value, placeholder, password)
	})
}

func (w *mainWindow) settingsInputControl(c *myui.Context, key string, value *string, placeholder string, password bool) myui.Element {
	input := myui.TextInput(c.Key(key), value).Placeholder(placeholder)
	if password {
		input = input.Password()
	}
	if input.Changed() {
		w.settingsDirty = true
		w.settingsHint = ""
	}
	return input
}

func (w *mainWindow) settingsCheck(c *myui.Context, key, label string, value *bool) {
	if myui.Checkbox(c.Key(key), value, label).Changed() {
		w.settingsDirty = true
		w.settingsHint = ""
	}
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

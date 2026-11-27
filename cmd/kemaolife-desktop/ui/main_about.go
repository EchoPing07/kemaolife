//go:build windows

package ui

import (
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

const (
	aboutRepoURL     = "https://github.com/EchoPing07/kemaolife"
	aboutReleasesURL = "https://github.com/EchoPing07/kemaolife/releases"
	aboutQQGroupURL  = "https://qm.qq.com/q/rf4PfFM5RA"
)

func (w *mainWindow) paintAbout(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	ver := mainTextOr(w.state.Version, "开发版")

	myui.Column(c).Fill().Children(func() {
		myui.Column(c).Shrink(0).FillWidth().Padding(22, 28, 12, 28).Children(func() {
			pageHeader(c, "关于", "应用信息、版本更新与项目说明", nil)
		})

		myui.Scroll(c.Key("about-scroll")).Grow(1).Height(0).FillWidth().Padding(2, 28, 24, 28).Children(func() {
			myui.Column(c).FillWidth().Gap(20).Children(func() {
				// 应用标识与版本更新主卡
				mainCard(c).FillWidth().Padding(20, 22).Gap(14).Children(func() {
					myui.Row(c).FillWidth().AlignItems(myui.Center).Gap(16).Children(func() {
						myui.Box(c).Size(56, 56).Radius(12).Center().Background(p.cardSecondary).
							Border(1, p.cardBorder).Shrink(0).Children(func() {
							if img := appTitleBarIcon(); img != nil {
								myui.Image(c, img).Size(36, 36)
							} else {
								myui.Icon(c, mainIcons.app).FontSize(26).TextColor(t.Text)
							}
						})

						myui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
								myui.Text(c, "科贸生活").FontSize(20).FontWeight(600).TextColor(t.Text)
								mainBadge(c, ver, t.TextMuted, p.cardSubtle, false)
							})
							myui.Text(c, "课表查询 · 宿舍电费 · 校园网认证 · 原生 Windows 桌面端").
								FontSize(13).TextColor(t.TextMuted)
							myui.Text(c, "非学校官方产品 · 仅查询本人数据 · 凭据经 Windows DPAPI 本地加密").
								FontSize(11).TextColor(t.TextMuted)
						})

						myui.Row(c).Shrink(0).AlignItems(myui.Center).Children(func() {
							label := "检查更新"
							if w.state.UpdateChecking {
								label = "检查中…"
							}
							if fluentPrimaryButton(c, "about-check-update", "").Disabled(w.state.UpdateChecking).
								Children(func() {
									myui.Icon(c, mainIcons.update).FontSize(14)
									myui.Text(c, label).FontSize(13).FontWeight(500)
								}).Clicked() && w.handler.OnCheckUpdate != nil {
								w.handler.OnCheckUpdate()
							}
						})
					})

					if w.state.UpdateHint != "" {
						myui.Box(c).Height(1).Background(p.hairline)
						mainInfoBar(c, mainIcons.info, w.state.UpdateHint, t.Text, p.cardSecondary, p.cardBorder)
					}
				})

				settingsGroup(c, "项目与交流", func() {
					w.aboutLinkRow(c, "about-repo", mainIcons.github, "开源仓库", "github.com/EchoPing07/kemaolife", "打开仓库", aboutRepoURL)
					w.aboutLinkRow(c, "about-releases", mainIcons.tag, "版本发布", "从 GitHub Releases 查看更新说明与安装包", "查看发布", aboutReleasesURL)
					w.aboutLinkRow(c, "about-qq", mainIcons.usersGroup, "QQ 交流群", "群号 1126474633 · 请勿在群内发送学号、密码或课表快照", "加入群聊", aboutQQGroupURL)
				})

				settingsGroup(c, "许可与说明", func() {
					settingsRowCard(c, mainIcons.scale, "开源许可", "MIT License · 欢迎提交 Issue 与 Pull Request", func() {
						myui.Text(c, "MIT").FontSize(12).FontWeight(500).TextColor(t.TextMuted)
					})
					settingsRowCard(c, mainIcons.shield, "隐私边界", "仅查询本人数据，不上传凭据或课表快照；本机配置与快照均通过 Windows DPAPI 加密", func() {
						myui.Text(c, "仅本地").FontSize(12).FontWeight(500).TextColor(t.TextMuted)
					})
				})
			})
		})
	})
}

func (w *mainWindow) aboutLinkRow(c *myui.Context, key string, icon *myui.SVG, title, desc, btnLabel, url string) {
	settingsRowCard(c, icon, title, desc, func() {
		if fluentButton(c, key, "").Children(func() {
			myui.Text(c, btnLabel).FontSize(12).TextColor(c.Theme().Text).SingleLine()
			myui.Icon(c, mainIcons.externalLink).FontSize(13).TextColor(c.Theme().TextMuted)
		}).Clicked() {
			w.openExternalURL(url)
		}
	})
}

func (w *mainWindow) openExternalURL(url string) {
	if w.handler.OnOpenURL != nil {
		w.handler.OnOpenURL(url)
		return
	}
	go func() {
		_ = mygo.Shell.OpenExternal(url)
	}()
}

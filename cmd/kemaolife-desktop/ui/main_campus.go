//go:build windows

package ui

import (
	"github.com/EchoPing07/kemaolife/internal/core/model"
	myui "github.com/egoist/mygo/ui"
)

func (w *mainWindow) paintCampus(c *myui.Context) {
	t := c.Theme()
	p := mainPalette(c)
	ready := !w.state.CampusLoading && !w.state.Refreshing
	canAuthenticate := ready && w.state.Campus.State != model.CampusOffline
	stateColor := campusStateColor(c, w.state.Campus.State)
	windowW, _ := c.Size()
	contentW := windowW - w.sidebarWidth() - 56
	compactActions := contentW < 760

	myui.Column(c).Fill().Children(func() {
		myui.Column(c).Shrink(0).FillWidth().Padding(22, 28, 12, 28).Children(func() {
			pageHeader(c, "校园网", "连接状态与认证操作", func() {
				if fluentButton(c, "campus-refresh", "").Disabled(!ready).Children(func() {
					myui.Icon(c, mainIcons.refresh).FontSize(14)
					myui.Text(c, "立即探测").FontSize(13)
				}).Clicked() {
					w.campusAction("refresh")
				}
			})
		})

		myui.Scroll(c.Key("campus-scroll")).Grow(1).Height(0).FillWidth().Padding(4, 28, 22, 28).Children(func() {
			myui.Column(c).FillWidth().Gap(16).Children(func() {
				// Win11 设置 Hero 状态卡片：黑白细线图标 + 状态小圆点 + 主色文字。
				mainCard(c).FillWidth().Padding(20, 22).Gap(14).Children(func() {
					renderActions := func() {
						myui.Row(c).Gap(8).AlignItems(myui.Center).Shrink(0).Children(func() {
							if !w.state.AutoMode {
								if fluentPrimaryButton(c, "campus-login", "登录校园网").Disabled(!canAuthenticate).Clicked() {
									w.campusAction("login")
								}
								if fluentButton(c, "campus-logout", "下线").Disabled(!canAuthenticate).Clicked() {
									w.campusAction("logout")
								}
							}
							if fluentButton(c, "campus-relogin", "重登").Disabled(!canAuthenticate).Clicked() {
								w.campusAction("relogin")
							}
						})
					}

					myui.Row(c).FillWidth().AlignItems(myui.Center).Gap(16).Children(func() {
						myui.Box(c).Size(44, 44).Radius(22).Center().Background(p.cardSecondary).
							Border(1, p.cardBorder).Shrink(0).Children(func() {
							myui.Icon(c, mainIcons.network).FontSize(20).TextColor(t.Text)
						})
						myui.Column(c).Grow(1).MinWidth(0).Gap(5).Children(func() {
							myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
								myui.Box(c).Size(8, 8).Radius(4).Background(stateColor).Shrink(0)
								myui.Text(c, mainStateLabel(w.state.Campus.State)).FontSize(22).FontWeight(600).
									TextColor(t.Text)
								mainBadge(c, mainBoolText(w.state.AutoMode, "自动模式", "手动模式"), t.TextMuted, p.cardSubtle, false)
							})
							myui.Text(c, "检查时间："+mainTextOr(mainTimeText(w.state.Campus.CheckedAt), "尚未检查")).
								FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
						})
						if !compactActions {
							renderActions()
						}
					})

					if compactActions {
						myui.Row(c).FillWidth().Padding(0, 0, 0, 60).Children(renderActions)
					}

					if w.state.CampusLoading || w.state.CampusError != "" || w.state.CampusAutoSkipped {
						myui.Box(c).Height(1).Background(p.hairline)
						mainLoading(c, w.state.CampusLoading, "正在执行，请稍候…")
						if !w.state.CampusLoading {
							mainError(c, w.state.CampusError)
							if w.state.CampusError == "" && w.state.CampusAutoSkipped {
								mainInfoBar(c, mainIcons.info, "自动认证已跳过：当前出口非有线", t.Text, p.cardSubtle, p.cardBorder)
							}
						}
					}
				})

				// 当前行为设置卡
				modeDesc := "全手动认证，不在后台自动登录。账号、认证与轮询选项在设置中修改。"
				if w.state.AutoMode {
					modeDesc = "未认证时按轮询策略自动登录。仅有线限制与独立凭据在设置中修改。"
				}
				settingsRowCard(c, mainIcons.shield, "当前认证行为", modeDesc, func() {
					if fluentButton(c, "campus-to-settings", "认证设置").Clicked() {
						w.page = mainPageSettings
					}
				})

				// 状态说明紧凑三栏
				myui.Column(c).FillWidth().Gap(8).Children(func() {
					myui.Text(c, "状态说明").FontSize(13).FontWeight(600).Padding(0, 0, 2, 2)
					myui.Row(c).FillWidth().Gap(10).AlignItems(myui.Stretch).Children(func() {
						campusLegendCard(c, model.CampusOnline, "已认证，可以访问网络")
						campusLegendCard(c, model.CampusUnauth, "在校园网内，但尚未认证")
						campusLegendCard(c, model.CampusOffline, "不在校园网或探测失败")
					})
					myui.Text(c, "探测失败归为「不在校园网」是正常状态，不进行失败退避。").
						FontSize(11).TextColor(t.TextMuted).Padding(2, 0, 0, 2)
				})
			})
		})
	})
}

func campusLegendCard(c *myui.Context, state model.CampusState, detail string) {
	p := mainPalette(c)
	myui.Column(c).Grow(1).MinWidth(0).Padding(12, 14).Gap(6).Radius(6).
		Background(p.card).Border(1, p.cardBorder).Children(func() {
		myui.Row(c).AlignItems(myui.Center).Gap(8).Children(func() {
			myui.Box(c).Size(8, 8).Radius(4).Background(campusStateColor(c, state)).Shrink(0)
			myui.Text(c, mainStateLabel(state)).FontSize(13).FontWeight(600)
		})
		myui.Text(c, detail).FontSize(12).TextColor(c.Theme().TextMuted)
	})
}

func (w *mainWindow) campusAction(action string) {
	if w.handler.OnCampusAction != nil {
		w.handler.OnCampusAction(action)
	}
}

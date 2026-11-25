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
	stateBg := campusStateSoftBg(c, w.state.Campus.State)

	myui.Column(c).Fill().Padding(24, 28).Gap(18).Children(func() {
		pageHeader(c, "校园网", "连接状态与认证操作", func() {
			if myui.Button(c.Key("campus-refresh"), "").Disabled(!ready).Children(func() {
				myui.Icon(c, mainIcons.refresh)
				myui.Text(c, "立即探测")
			}).Clicked() {
				w.campusAction("refresh")
			}
		})
		myui.Row(c).Grow(1).Gap(16).AlignItems(myui.Stretch).Children(func() {
			mainCard(c).Grow(1).Children(func() {
				myui.Row(c).AlignItems(myui.Center).Gap(10).Children(func() {
					myui.Text(c, "连接状态").FontSize(14).Bold()
					myui.Spacer(c)
					mainBadge(c, mainBoolText(w.state.AutoMode, "自动模式", "手动模式"), t.TextMuted, p.cardSubtle, false)
				})
				myui.Spacer(c)
				myui.Column(c).Gap(14).AlignItems(myui.Center).Children(func() {
					myui.Box(c).Size(76, 76).Radius(38).Center().Background(stateBg).Children(func() {
						myui.Icon(c, mainIcons.network).FontSize(34).TextColor(stateColor)
					})
					myui.Text(c, mainStateLabel(w.state.Campus.State)).FontSize(30).Bold().
						TextColor(stateColor)
					myui.Text(c, "检查时间："+mainTextOr(mainTimeText(w.state.Campus.CheckedAt), "尚未检查")).
						FontSize(12).TextColor(t.TextMuted).FontFeatures("tnum")
				})
				myui.Spacer(c)
				mainLoading(c, w.state.CampusLoading, "正在执行，请稍候…")
				if !w.state.CampusLoading {
					mainError(c, w.state.CampusError)
					if w.state.CampusError == "" && w.state.CampusAutoSkipped {
						mainInfoBar(c, mainIcons.info, "自动认证已跳过：当前出口非有线", t.Text, p.cardSubtle, p.cardBorder)
					}
				}
				myui.Box(c).Height(1).Background(p.hairline)
				myui.Row(c).Gap(10).Justify(myui.Center).Children(func() {
					if !w.state.AutoMode {
						if myui.PrimaryButton(c.Key("campus-login"), "登录校园网").Disabled(!canAuthenticate).Clicked() {
							w.campusAction("login")
						}
						if myui.Button(c.Key("campus-logout"), "下线").Disabled(!canAuthenticate).Clicked() {
							w.campusAction("logout")
						}
					}
					if myui.Button(c.Key("campus-relogin"), "重登").Disabled(!canAuthenticate).Clicked() {
						w.campusAction("relogin")
					}
				})
			})

			myui.Column(c).Width(316).FillHeight().Children(func() {
				mainCard(c).Grow(1).Children(func() {
					mainSectionTitle(c, "当前行为", "")
					if w.state.AutoMode {
						myui.Text(c, "未认证时按轮询策略自动登录。仅有线限制与独立凭据在设置中修改。").
							FontSize(13).TextColor(t.TextMuted)
					} else {
						myui.Text(c, "全手动认证，不在后台自动登录。账号、认证与轮询选项在设置中修改。").
							FontSize(13).TextColor(t.TextMuted)
					}
					myui.Box(c).Height(1).Background(p.hairline)
					mainSectionTitle(c, "状态说明", "")
					myui.Column(c).Gap(10).Children(func() {
						campusLegend(c, model.CampusOnline, "已认证，可以访问网络")
						campusLegend(c, model.CampusUnauth, "在校园网内，但尚未认证")
						campusLegend(c, model.CampusOffline, "不在校园网或探测失败")
					})
					myui.Spacer(c)
					myui.Text(c, "探测失败归为「不在校园网」是正常状态，不进行失败退避。").
						FontSize(11).TextColor(t.TextMuted)
					myui.Box(c).Height(1).Background(p.hairline)
					myui.Row(c).Justify(myui.End).Children(func() {
						if mainSubtleLink(c, "campus-to-settings", "认证设置") {
							w.page = mainPageSettings
						}
					})
				})
			})
		})
	})
}

func campusLegend(c *myui.Context, state model.CampusState, detail string) {
	p := mainPalette(c)
	myui.Row(c).Padding(9, 11).Radius(6).Background(p.cardSubtle).AlignItems(myui.Start).Gap(10).Children(func() {
		myui.Box(c).Size(8, 8).Radius(4).Background(campusStateColor(c, state)).Margin(5, 0, 0, 0).Shrink(0)
		myui.Column(c).Grow(1).Gap(2).Children(func() {
			myui.Text(c, mainStateLabel(state)).FontSize(13).Bold()
			myui.Text(c, detail).FontSize(11).TextColor(c.Theme().TextMuted)
		})
	})
}

func (w *mainWindow) campusAction(action string) {
	if w.handler.OnCampusAction != nil {
		w.handler.OnCampusAction(action)
	}
}

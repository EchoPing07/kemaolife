//go:build windows

package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
	"github.com/EchoPing07/kemaolife/internal/desktop"
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

// MainWindowState 是主窗口的完整只读快照。UI 只消费 DTO，不查询业务接口。
type MainWindowState struct {
	TodayDate string
	Today     model.DaySchedule
	HasToday  bool

	Week              int
	LastWeek          int
	Days              []model.DaySchedule
	TimetableSyncedAt string
	TimetableOffline  bool
	TimetableLoading  bool
	TimetableError    string

	Report             model.ElectricityReport
	HasReport          bool
	ElectricityLoading bool
	ElectricityError   string

	Campus            model.CampusStatus
	CampusLoading     bool
	CampusError       string
	CampusAutoSkipped bool
	AutoMode          bool
	Version           string
	SettingsForm      desktop.SettingsForm
	SettingsHint      string
	UpdateChecking    bool
	UpdateHint        string
	Refreshing        bool
}

// MainWindowHandler 是 main 注入的命令出口。网络动作由控制器在后台执行。
type MainWindowHandler struct {
	OnRefreshAll       func()
	OnTimetableWeek    func(int)
	OnTimetableRefresh func()
	OnElectricityMonth func(string)
	OnElectricityPoll  func()
	OnCampusAction     func(string)
	// 返回值依次是页内提示、保存后的生效表单和是否已落盘。
	OnSaveSettings func(desktop.SettingsResult) (string, desktop.SettingsForm, bool)
	OnCheckUpdate  func()
	OnOpenURL      func(string)
}

const (
	mainPageHome = iota
	mainPageTimetable
	mainPageElectricity
	mainPageCampus
	mainPageSettings
	mainPageAbout
)

const (
	MainPageHome     = mainPageHome
	MainPageSettings = mainPageSettings
	MainPageAbout    = mainPageAbout
)

const (
	sidebarWidthExpanded float32 = 212
	sidebarWidthCompact  float32 = 56
	titleBarDefaultH     float32 = 48
)

type mainWindow struct {
	native  *mygo.Window
	state   MainWindowState
	handler MainWindowHandler
	page    int

	sidebarCompact   bool
	month            string
	monthDirty       bool
	settings         desktop.SettingsResult
	settingsBase     desktop.SettingsForm
	settingsDirty    bool
	settingsHint     string
	timetableScroll  myui.ScrollState
	electricityChart int
	electricityHover int
	courseDialogOpen bool
	courseDialogDay  model.DaySchedule
	courseDialogItem model.Entry
}

var (
	mainMu           sync.RWMutex
	mainWnd          *mainWindow
	mainQuitting     bool
	titleBarIconOnce sync.Once
	titleBarIconImg  myui.ImageSource
)

func appTitleBarIcon() myui.ImageSource {
	titleBarIconOnce.Do(func() {
		if img, err := myui.DecodeImage(iconPNG); err == nil {
			titleBarIconImg = img
		}
	})
	return titleBarIconImg
}

// OpenMainWindow 创建或唤起唯一主窗口。owner 与 icon 仍由旧托盘装配传入。
func OpenMainWindow(owner, icon uintptr, state MainWindowState, h MainWindowHandler) (err error) {
	_ = owner
	_ = icon
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("创建主窗口失败: %v", r)
		}
	}()

	mainMu.Lock()
	if w := mainWnd; w != nil && w.native != nil {
		w.handler = h
		w.state = cloneMainState(state)
		w.page = mainPageHome
		w.syncStateForView()
		mainMu.Unlock()
		w.native.Update(func() {})
		w.native.Show()
		w.native.Focus()
		return nil
	}
	mainMu.Unlock()

	w := &mainWindow{
		state: cloneMainState(state), handler: h, page: mainPageHome,
		month: mainDisplayMonth(state),
	}
	w.setSettingsForm(state.SettingsForm)
	native := mygo.NewWindow(mygo.WindowOptions{
		Title:           "科贸生活",
		Width:           1180,
		Height:          760,
		MinWidth:        960,
		MinHeight:       640,
		UseContentSize:  true,
		TitleBarStyle:   mygo.TitleBarHidden,
		TitleBarHeight:  int(titleBarDefaultH),
		Vibrancy:        mygo.VibrancyMica,
		BackgroundColor: "light-dark(#f3f3f3, #202020)",
		Content:         myui.View(w.view),
	})
	w.native = native
	_ = native.SetIcon(iconPNG)
	native.OnClose(func(e *mygo.CloseEvent) {
		mainMu.RLock()
		quitting := mainQuitting
		mainMu.RUnlock()
		if !quitting {
			e.PreventDefault()
			native.Hide()
		}
	})
	mainMu.Lock()
	mainWnd = w
	mainMu.Unlock()
	return nil
}

func OpenMainPage(page int) {
	mainMu.Lock()
	w := mainWnd
	mainMu.Unlock()
	if w == nil || w.native == nil {
		return
	}
	w.native.Update(func() {
		w.page = page
		w.courseDialogOpen = false
	})
	w.native.Show()
	w.native.Focus()
}

func ShowMainWindow() {
	mainMu.Lock()
	w := mainWnd
	mainMu.Unlock()
	if w != nil && w.native != nil {
		w.native.Show()
		w.native.Focus()
	}
}

// UpdateMainWindow 原子替换展示快照。编辑中的设置表单不被后台状态覆盖。
func UpdateMainWindow(state MainWindowState) {
	mainMu.Lock()
	w := mainWnd
	mainMu.Unlock()
	if w == nil || w.native == nil {
		return
	}
	w.native.Update(func() {
		if w.state.Week != state.Week {
			w.timetableScroll.Y = 0
		}
		w.state = cloneMainState(state)
		w.syncStateForView()
	})
}

func (w *mainWindow) syncStateForView() {
	month := mainDisplayMonth(w.state)
	if w.monthDirty && w.state.HasReport && strings.TrimSpace(w.state.Report.Usage.YearMonth) == strings.TrimSpace(w.month) {
		w.monthDirty = false
	}
	if month != "" && !w.monthDirty {
		w.month = month
	}
	if !w.settingsDirty && w.state.SettingsForm != w.settingsBase {
		w.setSettingsForm(w.state.SettingsForm)
	}
}

func cloneMainState(s MainWindowState) MainWindowState {
	out := s
	out.Today.Entries = append([]model.Entry(nil), s.Today.Entries...)
	out.Days = make([]model.DaySchedule, len(s.Days))
	for i, day := range s.Days {
		out.Days[i] = day
		out.Days[i].Entries = append([]model.Entry(nil), day.Entries...)
	}
	out.Report.Usage.Days = append([]model.DayUsage(nil), s.Report.Usage.Days...)
	if s.Report.Status.DailyAvg != nil {
		v := *s.Report.Status.DailyAvg
		out.Report.Status.DailyAvg = &v
	}
	if s.Report.Status.EstimatedDays != nil {
		v := *s.Report.Status.EstimatedDays
		out.Report.Status.EstimatedDays = &v
	}
	return out
}

func (w *mainWindow) openCourseDetail(day model.DaySchedule, entry model.Entry) {
	w.courseDialogDay = day
	w.courseDialogItem = entry
	w.courseDialogOpen = true
}

func (w *mainWindow) sidebarWidth() float32 {
	if w.sidebarCompact {
		return sidebarWidthCompact
	}
	return sidebarWidthExpanded
}

func (w *mainWindow) view(c *myui.Context) {
	applyMainTheme(c)
	p := mainPalette(c)
	bar := c.TitleBar()
	titleHeight := maxFloat(bar.Height, titleBarDefaultH)
	if c.Vibrancy() {
		c.Root().Background(myui.Transparent)
	}

	root := myui.Column(c).Fill()
	if !c.Vibrancy() {
		root.Background(p.shell)
	}
	root.Children(func() {
		// Win11 透明标题栏：最左放置导航折叠按钮（不拦截拖拽），右侧整段承载 DragWindow。
		myui.Row(c).Height(titleHeight).Padding(0, bar.Right+12, 0, 12).AlignItems(myui.Center).Gap(6).Children(func() {
			toggleTip := "折叠导航栏"
			if w.sidebarCompact {
				toggleTip = "展开导航栏"
			}
			if fluentIconButton(c, "sidebar-toggle", mainIcons.menu, toggleTip, true).Clicked() {
				w.sidebarCompact = !w.sidebarCompact
			}
			myui.Row(c).Grow(1).FillHeight().AlignItems(myui.Center).Gap(8).DragWindow().Children(func() {
				titleColor := c.Theme().Text
				if !c.WindowActive() {
					titleColor = c.Theme().TextMuted
				}
				if img := appTitleBarIcon(); img != nil {
					el := myui.Image(c, img).Size(18, 18)
					if !c.WindowActive() {
						el.Opacity(0.65)
					}
				} else {
					iconColor := tAccent(c)
					if !c.WindowActive() {
						iconColor = c.Theme().TextMuted
					}
					myui.Icon(c, mainIcons.app).FontSize(16).TextColor(iconColor)
				}
				myui.Text(c, "科贸生活").FontSize(12).FontWeight(500).TextColor(titleColor)
			})
		})

		myui.Row(c).Grow(1).Height(0).AlignItems(myui.Stretch).Children(func() {
			w.paintSidebar(c)
			// 整窗通铺纯净 Mica 淡色调，内容区不加多余灰底套娃，白/深灰卡片直接浮在 Mica 上。
			myui.Column(c).Grow(1).MinWidth(0).FillHeight().Clip().Children(func() {
				myui.Column(c.Key(w.page)).Fill().Transition(myui.ElementTransition{
					Duration: 160 * time.Millisecond,
					Enter:    &myui.Motion{Y: 8, Opacity: 0},
				}).Children(func() {
					switch w.page {
					case mainPageHome:
						w.paintHome(c)
					case mainPageTimetable:
						w.paintTimetable(c)
					case mainPageElectricity:
						w.paintElectricity(c)
					case mainPageCampus:
						w.paintCampus(c)
					case mainPageSettings:
						w.paintSettings(c)
					case mainPageAbout:
						w.paintAbout(c)
					}
				})
			})
		})
	})

	w.paintCourseDialog(c)
}

type navItem struct {
	page int
	name string
	icon *myui.SVG
}

func (w *mainWindow) paintSidebar(c *myui.Context) {
	p := mainPalette(c)
	primaryNav := []navItem{
		{mainPageHome, "首页", mainIcons.home},
		{mainPageTimetable, "课表", mainIcons.calendar},
		{mainPageElectricity, "电费", mainIcons.electricity},
		{mainPageCampus, "校园网", mainIcons.network},
	}
	footerNav := []navItem{
		{mainPageSettings, "设置", mainIcons.settings},
		{mainPageAbout, "关于", mainIcons.info},
	}

	padX := float32(10)
	if w.sidebarCompact {
		padX = 8
	}
	myui.Column(c).Width(w.sidebarWidth()).Shrink(0).FillHeight().Padding(4, padX, 14, padX).Gap(3).Children(func() {
		for _, item := range primaryNav {
			w.paintNavButton(c, item, p)
		}
		myui.Spacer(c)
		myui.Box(c).Height(1).Background(p.hairline).Margin(6, 6)
		for _, item := range footerNav {
			w.paintNavButton(c, item, p)
		}
	})
}

func (w *mainWindow) paintNavButton(c *myui.Context, item navItem, p mainColors) {
	t := c.Theme()
	active := w.page == item.page
	button := myui.ButtonBase(c.Key(item.name)).Height(38).Radius(4).
		AlignItems(myui.Center).Label(item.name)
	if w.sidebarCompact {
		button.Padding(0, 6, 0, 2).Gap(4).Justify(myui.Center).Tooltip(item.name)
	} else {
		button.Padding(0, 12, 0, 4).Gap(10).Justify(myui.Start)
	}

	indicatorColor := myui.Transparent
	iconColor := t.TextMuted
	textColor := t.Text
	if active {
		indicatorColor = t.Accent
		iconColor = t.Text
		if !c.WindowActive() {
			indicatorColor = t.TextMuted
			iconColor = t.TextMuted
		}
	} else if button.Hovered() {
		iconColor = t.Text
	}

	button.Children(func() {
		// Win11 NavigationView 标志性的 3px 圆角选中指示条，切页时纵向滑移。
		ind := myui.Box(c).Width(3).Height(16).Radius(2).Background(indicatorColor).Shrink(0)
		if active {
			ind.Transition(myui.ElementTransition{
				Match:    "nav-indicator",
				Duration: 180 * time.Millisecond,
			})
		}
		myui.Icon(c, item.icon).FontSize(16).TextColor(iconColor).Shrink(0)
		if !w.sidebarCompact {
			label := myui.Text(c, item.name).FontSize(14).TextColor(textColor).SingleLine()
			if active {
				label.FontWeight(600)
			}
		}
	})

	switch {
	case button.Pressed():
		button.Background(p.navPressed)
	case active:
		button.Background(p.navActive)
	case button.Hovered():
		button.Background(p.navHover)
	}
	if button.Clicked() {
		w.page = item.page
		w.courseDialogOpen = false
	}
}

func mainDisplayMonth(s MainWindowState) string {
	if month := strings.TrimSpace(s.Report.Usage.YearMonth); month != "" {
		return month
	}
	if len(s.TodayDate) >= 7 {
		return s.TodayDate[:7]
	}
	if !s.Today.Date.IsZero() {
		if value := s.Today.Date.String(); len(value) >= 7 {
			return value[:7]
		}
	}
	return ""
}

func maxFloat(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

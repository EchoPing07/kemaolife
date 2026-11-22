//go:build windows

package ui

import (
	"fmt"
	"strings"
	"sync"

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
}

const (
	mainPageToday = iota
	mainPageTimetable
	mainPageElectricity
	mainPageCampus
	mainPageSettings
)

const (
	MainPageToday    = mainPageToday
	MainPageSettings = mainPageSettings
)

type mainWindow struct {
	native  *mygo.Window
	state   MainWindowState
	handler MainWindowHandler
	page    int

	month           string
	monthDirty      bool
	settings        desktop.SettingsResult
	settingsBase    desktop.SettingsForm
	settingsDirty   bool
	settingsHint    string
	timetableScroll myui.ScrollState
	electricityRows myui.ListState
}

var (
	mainMu       sync.RWMutex
	mainWnd      *mainWindow
	mainQuitting bool
)

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
		w.page = mainPageToday
		w.syncStateForView()
		mainMu.Unlock()
		w.native.Update(func() {})
		w.native.Show()
		w.native.Focus()
		return nil
	}
	mainMu.Unlock()

	w := &mainWindow{
		state: cloneMainState(state), handler: h, page: mainPageToday,
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
		TitleBarHeight:  40,
		BackgroundColor: "#f4f6f8",
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
	w.native.Update(func() { w.page = page })
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

func (w *mainWindow) view(c *myui.Context) {
	applyMainTheme(c)
	p := mainPalette(c)
	bar := c.TitleBar()
	titleHeight := maxFloat(bar.Height, 40)
	myui.Column(c).Fill().Children(func() {
		myui.Row(c).Height(titleHeight).Padding(0, bar.Right+16, 0, 16).
			Background(p.sidebar).DragWindow().Children(func() {
			myui.Icon(c, mainIcons.app).FontSize(18).TextColor(tAccent(c))
			myui.Text(c, "科贸生活").FontSize(13).Bold().Padding(8, 0, 0, 0)
		})
		myui.Row(c).Grow(1).AlignItems(myui.Stretch).Children(func() {
			w.paintSidebar(c)
			myui.Column(c).Grow(1).FillHeight().Background(p.canvas).Children(func() {
				switch w.page {
				case mainPageToday:
					w.paintToday(c)
				case mainPageTimetable:
					w.paintTimetable(c)
				case mainPageElectricity:
					w.paintElectricity(c)
				case mainPageCampus:
					w.paintCampus(c)
				case mainPageSettings:
					w.paintSettings(c)
				}
			})
		})
	})
}

func (w *mainWindow) paintSidebar(c *myui.Context) {
	p := mainPalette(c)
	pages := []struct {
		page int
		name string
		icon *myui.SVG
	}{
		{mainPageToday, "今天", mainIcons.today},
		{mainPageTimetable, "课表", mainIcons.calendar},
		{mainPageElectricity, "电费", mainIcons.electricity},
		{mainPageCampus, "校园网", mainIcons.network},
		{mainPageSettings, "设置", mainIcons.settings},
	}
	myui.Column(c).Width(188).FillHeight().Padding(12, 12, 16, 12).Gap(5).
		Background(p.sidebar).Children(func() {
		for _, item := range pages {
			item := item
			active := w.page == item.page
			button := myui.ButtonBase(c.Key(item.name)).Height(40).Padding(0, 12).Gap(11).Radius(7)
			button.Children(func() {
				myui.Icon(c, item.icon).FontSize(18)
				myui.Text(c, item.name).FontSize(14)
			})
			if active {
				button.Background(p.accentSoft).TextColor(tAccent(c))
				button.Border(1, p.accentBorder)
			} else if button.Hovered() {
				button.Background(p.sidebarHover)
			}
			if button.Clicked() {
				w.page = item.page
			}
		}
		myui.Spacer(c)
		if w.state.Version != "" {
			myui.Text(c, w.state.Version).FontSize(10).TextColor(c.Theme().TextMuted).
				SingleLine().Padding(8, 10, 0, 10)
		}
	})
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

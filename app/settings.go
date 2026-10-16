package app

import (
	"log"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) installMenus() {
	lang := a.config().Language
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: tr(lang, "app_menu"), Submenu: []*mygo.MenuItem{
			{Label: tr(lang, "menu_settings"), Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.showSettings()
			}},
			{Label: tr(lang, "menu_toggle"), Click: func(*mygo.MenuItem, *mygo.Window) { a.toggleVisible() }},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleWindowMenu},
	}))

	trayMenu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: tr(lang, "menu_settings"), Click: func(*mygo.MenuItem, *mygo.Window) { a.showSettings() }},
		{Label: tr(lang, "menu_toggle"), Click: func(*mygo.MenuItem, *mygo.Window) { a.toggleVisible() }},
		mygo.Separator(),
		{Label: tr(lang, "menu_quit"), Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	})
	if a.tray != nil {
		a.tray.SetMenu(trayMenu)
		return
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Title:   "snip",
		ToolTip: "snip",
		Menu:    trayMenu,
	})
	if err != nil {
		log.Printf("snip: menu bar item: %v", err)
		return
	}
	a.tray = tray
}

func (a *App) showSettings() {
	mygo.RunOnMain(a.openSettings)
}

func (a *App) openSettings() {
	if a.settingsWin != nil && a.settingsWin.IsVisible() {
		a.settingsWin.Show()
		a.settingsWin.Focus()
		return
	}
	form := config.FormFrom(a.config())
	a.settingsForm = &form
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:       "Settings",
		Width:       460,
		Height:      640,
		MinWidth:    380,
		MinHeight:   420,
		Panel:       false,
		Frameless:   false,
		AlwaysOnTop: false,
		StateKey:    "settings",
		Content:     ui.View(func(c *ui.Context) { a.settingsView(c) }),
	})
	a.settingsWin = win
	win.OnClosed(func() {
		a.settingsWin = nil
		a.settingsForm = nil
	})
	win.Show()
	win.Focus()
}

func (a *App) settingsView(c *ui.Context) {
	f := a.settingsForm
	if f == nil {
		return
	}
	dark := c.Theme().Dark
	c.SetTheme(paperTheme(dark, activeColor(f.ActiveColor, dark)))
	t := c.Theme()
	lang := f.Language
	if lang == "" {
		lang = "en"
	}
	ui.Column(c).Fill().Background(t.Background).Padding(18).Gap(12).Children(func() {
		ui.Text(c, tr(lang, "settings")).Font("Georgia").FontSize(22)
		ui.Text(c, tr(lang, "settings_hint")).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(12).Padding(0, 4, 12, 0).Children(func() {
				a.settingsSection(c, tr(lang, "language"), func() {
					ui.Select(c, &f.Language, []string{"en", "zh"})
				})
				a.settingsSection(c, tr(lang, "clipboard"), func() {
					labeledInput(c, tr(lang, "poll"), &f.ClipboardPollMs)
					labeledInput(c, tr(lang, "max_cards"), &f.MaxCards)
					ui.Text(c, tr(lang, "placement")).FontSize(12)
					ui.Select(c, &f.Placement, []string{"near-selection", "near-cursor", "fixed"})
				})
				a.settingsSection(c, tr(lang, "windows"), func() {
					ui.Checkbox(c, &f.AlwaysOnTop, tr(lang, "always_on_top"))
					ui.Checkbox(c, &f.VisibleOnAllWorkspaces, tr(lang, "all_spaces"))
					ui.Checkbox(c, &f.ContentProtection, tr(lang, "capture"))
					ui.Checkbox(c, &f.Panel, tr(lang, "panel"))
					ui.Text(c, tr(lang, "restart")).FontSize(11).TextColor(t.TextMuted)
				})
				a.settingsSection(c, tr(lang, "size"), func() {
					labeledInput(c, tr(lang, "min_w"), &f.MinWidth)
					labeledInput(c, tr(lang, "max_w"), &f.MaxWidth)
					labeledInput(c, tr(lang, "max_h"), &f.MaxHeight)
					labeledInput(c, tr(lang, "line_h"), &f.LineHeight)
					labeledInput(c, tr(lang, "max_lines"), &f.MaxLines)
					labeledInput(c, tr(lang, "pad_x"), &f.PaddingX)
					labeledInput(c, tr(lang, "pad_y"), &f.PaddingY)
					labeledInput(c, tr(lang, "font"), &f.FontSize)
					labeledInput(c, tr(lang, "char_w"), &f.CharWidth)
					labeledInput(c, tr(lang, "active_color"), &f.ActiveColor)
					labeledInput(c, "Cascade origin x", &f.CascadeOriginX)
					labeledInput(c, "Cascade origin y", &f.CascadeOriginY)
					labeledInput(c, "Cascade step x", &f.CascadeStepX)
					labeledInput(c, "Cascade step y", &f.CascadeStepY)
				})
				a.settingsSection(c, tr(lang, "motion"), func() {
					ui.Checkbox(c, &f.MotionEnabled, tr(lang, "motion_on"))
					labeledInput(c, tr(lang, "motion_ms"), &f.MotionMs)
				})
				a.settingsSection(c, tr(lang, "hotkeys"), func() {
					ui.Checkbox(c, &f.Direct, tr(lang, "direct"))
					labeledInput(c, tr(lang, "toggle"), &f.Toggle)
					labeledInput(c, tr(lang, "next"), &f.Next)
					labeledInput(c, tr(lang, "prev"), &f.Prev)
					labeledInput(c, tr(lang, "paste"), &f.Paste)
					labeledInput(c, tr(lang, "delete"), &f.Delete)
					labeledInput(c, tr(lang, "pin"), &f.Pin)
					labeledInput(c, tr(lang, "settings"), &f.Settings)
				})
				a.settingsSection(c, tr(lang, "leader"), func() {
					labeledInput(c, tr(lang, "leader_shortcut"), &f.LeaderShortcut)
					labeledInput(c, tr(lang, "leader_timeout"), &f.LeaderTimeout)
					labeledInput(c, tr(lang, "leader_toggle"), &f.LeaderToggle)
					labeledInput(c, tr(lang, "leader_next"), &f.LeaderNext)
					labeledInput(c, tr(lang, "leader_prev"), &f.LeaderPrev)
					labeledInput(c, tr(lang, "leader_paste"), &f.LeaderPaste)
					labeledInput(c, tr(lang, "leader_delete"), &f.LeaderDelete)
					labeledInput(c, tr(lang, "leader_pin"), &f.LeaderPin)
					labeledInput(c, tr(lang, "leader_settings"), &f.LeaderSettings)
					labeledInput(c, tr(lang, "double_ms"), &f.DoublePressMs)
					ui.Text(c, tr(lang, "double_map")).FontSize(12).TextColor(t.TextMuted).MaxLines(2)
					ui.TextArea(c, &f.DoubleMap).Height(56)
					ui.Checkbox(c, &f.DimOverlay, tr(lang, "dim"))
				})
				a.settingsSection(c, tr(lang, "secrets"), func() {
					ui.Checkbox(c, &f.MaskEnabled, tr(lang, "mask"))
					labeledInput(c, tr(lang, "entropy"), &f.MinEntropyBits)
					labeledInput(c, tr(lang, "secret_len"), &f.MinSecretLength)
					ui.Text(c, tr(lang, "patterns")).FontSize(12)
					ui.TextArea(c, &f.ExtraPatterns).Height(64)
					ui.Text(c, tr(lang, "markers")).FontSize(12).TextColor(t.TextMuted)
					ui.Checkbox(c, &f.HonorConcealed, tr(lang, "concealed"))
					ui.Checkbox(c, &f.SkipTransient, tr(lang, "transient"))
					ui.Checkbox(c, &f.SkipAutoGenerated, tr(lang, "auto"))
				})
				a.settingsSection(c, tr(lang, "config_path"), func() {
					ui.Text(c, a.configPath).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
				})
			})
		})
		if f.Message != "" {
			ui.Text(c, f.Message).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
		}
		if ui.Button(c, tr(lang, "save")).Clicked() {
			a.saveSettings()
		}
	})
}

func (a *App) settingsSection(c *ui.Context, title string, body func()) {
	t := c.Theme()
	ui.Column(c).Gap(8).Padding(14).Background(t.Surface).Radius(14).Border(1, t.Border).
		Shadow(0, 6, 16, 0, ui.Hex("#3B3128").Alpha(0.08)).
		Children(func() {
			ui.Text(c, title).Font("Georgia").FontSize(15)
			body()
		})
}

func labeledInput(c *ui.Context, label string, value *string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).Grow(1)
		ui.TextInput(c, value).Width(180)
	})
}

func (a *App) saveSettings() {
	f := a.settingsForm
	if f == nil {
		return
	}
	next, err := f.Parse(a.config())
	if err != nil {
		f.Message = err.Error()
		if a.settingsWin != nil {
			a.settingsWin.Invalidate()
		}
		return
	}
	if err := config.Save(a.configPath, next); err != nil {
		f.Message = err.Error()
		if a.settingsWin != nil {
			a.settingsWin.Invalidate()
		}
		return
	}
	restart := config.NeedsRestart(a.config(), next)
	a.applyConfig(next)
	lang := next.Language
	if restart {
		f.Message = tr(lang, "saved_restart")
	} else {
		f.Message = tr(lang, "saved")
	}
	if a.settingsWin != nil {
		a.settingsWin.Invalidate()
	}
}

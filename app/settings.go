package app

import (
	"log"
	"strconv"
	"strings"

	"github.com/0froq/snip/config"
	"github.com/0froq/snip/internal/macplace"
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

func atoiOr(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (a *App) showSettings() {
	mygo.RunOnMain(a.openSettings)
}

func (a *App) closeSettings() {
	if a.settingsWin != nil {
		a.settingsWin.Close()
	}
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
		Title:           "Settings",
		Width:           460,
		Height:          640,
		MinWidth:        380,
		MinHeight:       420,
		Frameless:       true,
		DisableShadow:   true,
		AlwaysOnTop:     false,
		BackgroundColor: "light-dark(#E6D9C8, #1A1613)",
		StateKey:        "settings",
		Content:         ui.View(func(c *ui.Context) { a.settingsView(c) }),
	})
	macplace.SquareWindow(win.NativeHandle())
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
	font := config.Card{FontFamily: f.FontFamily, FontSize: atoiOr(f.FontSize, config.DefaultFontSize)}.UIFont(0)
	c.SetTheme(paperTheme(dark, activeColor(f.ActiveColor, dark), font))
	t := c.Theme()
	lang := f.Language
	if lang == "" {
		lang = "en"
	}
	if c.Shortcut(ui.Cmd, ui.KeyW) || c.Shortcut(0, ui.KeyEscape) {
		a.closeSettings()
	}
	ui.Column(c).Fill().Background(t.Background).Radius(t.Radius).Padding(14).Children(func() {
		ui.Column(c).Fill().Background(t.Surface).Radius(t.Radius).Border(1, paperEdge(dark)).
			Shadow(0, 4, 14, 0, paperShadow(dark)).
			Draw(func(p *ui.Painter, r ui.Rect) { paintPaperGrain(p, r, dark, paperSeed("settings")) }).
			Children(func() {
				ui.Row(c).Shrink(0).Padding(14, 14, 10, 14).Gap(10).AlignItems(ui.Center).
					DragWindow().
					Children(func() {
						ui.Column(c).Grow(1).Gap(4).Children(func() {
							ui.Text(c, tr(lang, "settings")).FontSize(22).TextColor(paperInk(dark))
							ui.Text(c, tr(lang, "settings_hint")).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
						})
						a.settingsCloseButton(c, lang, dark)
					})
				ui.Box(c).Height(1).Shrink(0).Background(paperEdge(dark).Alpha(0.45))
				ui.Scroll(c).Grow(1).Clip().Padding(12, 14, 4, 14).Children(func() {
					ui.Column(c).Gap(12).Children(func() {
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
							labeledInput(c, tr(lang, "max_lines"), &f.MaxLines)
							labeledInput(c, tr(lang, "pad_x"), &f.PaddingX)
							labeledInput(c, tr(lang, "pad_y"), &f.PaddingY)
							labeledInput(c, tr(lang, "font_family"), &f.FontFamily)
							labeledInput(c, tr(lang, "font"), &f.FontSize)
							labeledInput(c, tr(lang, "active_color"), &f.ActiveColor)
							ui.Checkbox(c, &f.DoubleClickPaste, tr(lang, "double_click_paste"))
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
							labeledInput(c, tr(lang, "leader_shortcut"), &f.LeaderShortcut)
							labeledInput(c, tr(lang, "leader_timeout"), &f.LeaderTimeout)
							labeledInput(c, tr(lang, "double_ms"), &f.DoublePressMs)
							labeledInput(c, tr(lang, "which_key_x"), &f.WhichKeyX)
							labeledInput(c, tr(lang, "which_key_y"), &f.WhichKeyY)
							ui.Text(c, tr(lang, "which_key_align_x")).FontSize(12)
							ui.Select(c, &f.WhichKeyAlignX, []string{config.AlignStart, config.AlignCenter, config.AlignEnd})
							ui.Text(c, tr(lang, "which_key_align_y")).FontSize(12)
							ui.Select(c, &f.WhichKeyAlignY, []string{config.AlignStart, config.AlignCenter, config.AlignEnd})
							for _, item := range f.Bindings.NamedBindings() {
								settingsBinding(c, tr(lang, item.Name), item.Binding, lang)
							}
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
				ui.Column(c).Shrink(0).Gap(8).Padding(12, 14, 14, 14).Background(t.Surface).
					DrawOver(func(p *ui.Painter, r ui.Rect) {
						fade := paperEdge(dark).Alpha(0.35)
						p.Line(r.X, r.Y, r.X+r.W, r.Y, 1, fade)
					}).
					Children(func() {
						if f.Message != "" {
							ui.Text(c, f.Message).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
						}
						if ui.Button(c, tr(lang, "save")).Clicked() {
							a.saveSettings()
						}
					})
			})
	})
}

func (a *App) settingsCloseButton(c *ui.Context, lang string, dark bool) {
	t := c.Theme()
	label := tr(lang, "close")
	btn := ui.ButtonBase(c).Size(28, 28).Radius(t.Radius).Center().
		Label(label).Tooltip(label).
		Border(1, paperEdge(dark)).Background(t.SurfaceHover.Mix(t.Surface, 0.5)).
		TextColor(t.TextMuted)
	btn.Children(func() {
		ui.Text(c, "×").FontSize(18).TextColor(paperInk(dark))
	})
	if btn.Clicked() {
		a.closeSettings()
	}
}

func (a *App) settingsSection(c *ui.Context, title string, body func()) {
	t := c.Theme()
	dark := t.Dark
	ui.Column(c).Gap(8).Padding(14).Background(t.SurfaceHover.Mix(t.Surface, 0.35)).Radius(t.Radius).Border(1, paperEdge(dark).Alpha(0.7)).
		Draw(func(p *ui.Painter, r ui.Rect) { paintPaperGrain(p, r, dark, paperSeed(title)) }).
		Children(func() {
			ui.Text(c, title).FontSize(16).TextColor(paperInk(dark))
			body()
		})
}

func labeledInput(c *ui.Context, label string, value *string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).Grow(1)
		ui.TextInput(c, value).Width(180)
	})
}

func settingsBinding(c *ui.Context, label string, binding *config.ActionBinding, lang string) {
	ui.Column(c).Gap(6).Padding(8, 0).Children(func() {
		ui.Text(c, label).FontSize(14)
		labeledInput(c, tr(lang, "direct_binding"), &binding.Direct)
		ui.Checkbox(c, &binding.DirectDouble, tr(lang, "double_press"))
		labeledInput(c, tr(lang, "leader_binding"), &binding.Leader)
		ui.Row(c).Gap(12).Children(func() {
			ui.Checkbox(c, &binding.LeaderDouble, tr(lang, "double_press"))
			ui.Checkbox(c, &binding.Sticky, tr(lang, "sticky"))
		})
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

package app

import (
	"log"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) installMenus() {
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "snip", Submenu: []*mygo.MenuItem{
			{Label: "Settings…", Accelerator: "CmdOrCtrl+,", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.showSettings()
			}},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleWindowMenu},
	}))

	trayMenu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Settings…", Click: func(*mygo.MenuItem, *mygo.Window) { a.showSettings() }},
		{Label: "Show cards", Click: func(*mygo.MenuItem, *mygo.Window) { a.setAllVisible(true) }},
		{Label: "Hide cards", Click: func(*mygo.MenuItem, *mygo.Window) { a.setAllVisible(false) }},
		mygo.Separator(),
		{Label: "Quit", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	})
	if _, err := mygo.NewTray(mygo.TrayOptions{
		Title:   "snip",
		ToolTip: "snip clipboard cards",
		Menu:    trayMenu,
	}); err != nil {
		log.Printf("snip: menu bar item: %v", err)
	}
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
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Text(c, "Settings").FontSize(18)
		ui.Text(c, "Most options apply as soon as you save. The non activating panel needs a restart.").
			FontSize(12).TextColor(t.TextMuted).MaxLines(2)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(12).Padding(0, 4, 8, 0).Children(func() {
				a.settingsSection(c, "Clipboard", func() {
					labeledInput(c, "Poll interval (ms)", &f.ClipboardPollMs)
					labeledInput(c, "Max cards", &f.MaxCards)
					ui.Text(c, "Placement").FontSize(12)
					ui.Select(c, &f.Placement, []string{"near-selection", "near-cursor", "fixed"})
				})
				a.settingsSection(c, "Card windows", func() {
					ui.Checkbox(c, &f.AlwaysOnTop, "Always on top")
					ui.Checkbox(c, &f.VisibleOnAllWorkspaces, "Visible on all Spaces")
					ui.Checkbox(c, &f.ContentProtection, "Hide from screen capture")
					ui.Checkbox(c, &f.Panel, "Non activating panel")
					ui.Text(c, "Applies after restart").FontSize(11).TextColor(t.TextMuted)
				})
				a.settingsSection(c, "Card size", func() {
					labeledInput(c, "Min width", &f.MinWidth)
					labeledInput(c, "Max width", &f.MaxWidth)
					labeledInput(c, "Height", &f.Height)
					labeledInput(c, "Max lines", &f.MaxLines)
					labeledInput(c, "Padding x", &f.PaddingX)
					labeledInput(c, "Padding y", &f.PaddingY)
					labeledInput(c, "Font size", &f.FontSize)
					labeledInput(c, "Char width", &f.CharWidth)
					labeledInput(c, "Cascade origin x", &f.CascadeOriginX)
					labeledInput(c, "Cascade origin y", &f.CascadeOriginY)
					labeledInput(c, "Cascade step x", &f.CascadeStepX)
					labeledInput(c, "Cascade step y", &f.CascadeStepY)
				})
				a.settingsSection(c, "Hotkeys", func() {
					labeledInput(c, "Show cards", &f.ShowAll)
					labeledInput(c, "Hide cards", &f.HideAll)
					labeledInput(c, "Next card", &f.Next)
					labeledInput(c, "Previous card", &f.Prev)
					labeledInput(c, "Paste", &f.Paste)
					labeledInput(c, "Delete", &f.Delete)
					labeledInput(c, "Settings", &f.Settings)
				})
				a.settingsSection(c, "Secrets", func() {
					ui.Checkbox(c, &f.MaskEnabled, "Mask secrets on cards")
					labeledInput(c, "Min entropy", &f.MinEntropyBits)
					labeledInput(c, "Min secret length", &f.MinSecretLength)
					ui.Text(c, "Extra patterns, one regex per line").FontSize(12)
					ui.TextArea(c, &f.ExtraPatterns).Height(72)
				})
			})
		})
		if f.Message != "" {
			ui.Text(c, f.Message).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Save").Clicked() {
				a.saveSettings()
			}
		})
	})
}

func (a *App) settingsSection(c *ui.Context, title string, body func()) {
	t := c.Theme()
	ui.Column(c).Gap(8).Padding(12).Background(t.Surface).Radius(8).Border(1, t.Border).Children(func() {
		ui.Text(c, title).FontSize(13)
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
	if restart {
		f.Message = "Saved. The non activating panel applies after you restart snip."
	} else {
		f.Message = "Saved."
	}
	if a.settingsWin != nil {
		a.settingsWin.Invalidate()
	}
}

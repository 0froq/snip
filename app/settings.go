package app

import (
	"log"
	"strings"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/input"
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/layout"
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/text"
	"github.com/0froq/snip/config"
	"github.com/0froq/snip/internal/macpaste"
	"github.com/0froq/snip/internal/macplace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) installMenus() {
	cfg := a.config()
	lang := cfg.Language
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: tr(lang, "app_menu"), Submenu: []*mygo.MenuItem{
			{Label: tr(lang, "menu_settings"), Accelerator: strings.TrimSpace(cfg.Bindings.Settings.Direct), Click: func(*mygo.MenuItem, *mygo.Window) {
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
	a.settingsApps = macpaste.RunningApps()
	a.settingsForm = &form
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:           "Settings",
		Width:           640,
		Height:          720,
		MinWidth:        580,
		MinHeight:       480,
		Frameless:       true,
		DisableShadow:   true,
		AlwaysOnTop:     false,
		BackgroundColor: "transparent",
		// settings-tabs drops the saved one-page size, which is too narrow for the tab bar.
		StateKey: "settings-tabs",
		Content:  ui.View(func(c *ui.Context) { a.settingsView(c) }),
	})
	macplace.SquareWindow(win.NativeHandle())
	a.settingsWin = win
	win.OnClosed(func() {
		a.settingsWin = nil
		a.settingsForm = nil
	})
	win.Show()
	win.Focus()
	// makeKeyAndOrderFront lays the content view out again. Fit it after
	// that, or the title-bar inset and the window hairline come back.
	macplace.SquareWindow(win.NativeHandle())
}

func (a *App) settingsView(c *ui.Context) {
	f := a.settingsForm
	if f == nil {
		return
	}
	dark := c.Theme().Dark
	lang := f.Language
	if lang == "" {
		lang = "en"
	}
	draft := a.config()
	draft.Language = lang
	draft.Theme.Palette = f.Palette
	k := usePixel(c, draft, dark)
	if c.Shortcut(ui.Cmd, ui.KeyW) || c.Shortcut(0, ui.KeyEscape) {
		a.closeSettings()
	}
	if f.SettingsTab < 0 || f.SettingsTab > int(settingsPrivacy) {
		f.SettingsTab = int(settingsGeneral)
	}
	// The frame is the window edge. Padding sits inside it, so there is
	// no second panel and no gap inside the mask.
	pad := k.D(cardFrameDots + 3)
	ui.Column(c).Fill().FocusRing(false).Draw(func(p *ui.Painter, r ui.Rect) {
		g := k.Canvas(p, r)
		if g.W < 3 || g.H < 3 {
			return
		}
		g.Frame(0, 0, g.W, g.H, pixel.Window, pixel.Face)
	}).Children(func() {
		ui.Column(c).Fill().Padding(pad).Gap(k.D(4)).Children(func() {
			ui.Row(c).Shrink(0).Gap(k.D(4)).AlignItems(ui.Center).DragWindow().Children(func() {
				ui.Column(c).Grow(1).Gap(k.D(2)).Children(func() {
					text.Heading(c, tr(lang, "settings"), text.HeadingOptions{Level: 2, Slot: pixel.Ink})
					text.Paragraph(c, tr(lang, "settings_hint"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Faint})
				})
				if input.Button(c, "X", input.ButtonOptions{Label: tr(lang, "close")}).Clicked {
					a.closeSettings()
				}
			})
			input.Segmented(c, tr(lang, "settings"), settingsTabLabels(lang), &f.SettingsTab, input.SegmentedOptions{})
			// The scroll keeps its content height, and shrinks only when the
			// page is taller than the window. Leftover space sits below Save,
			// so a short page is not a box with a hole under the last control.
			ui.Scroll(c).Shrink(1).MinHeight(0).Children(func() {
				ui.Column(c).Shrink(0).Gap(k.D(6)).Children(func() {
					a.settingsPage(c, f, lang)
				})
			})
			if f.Message != "" {
				text.Paragraph(c, f.Message, text.ParagraphOptions{MaxLines: 3, Slot: pixel.Faint})
			}
			if input.Button(c, tr(lang, "save"), input.ButtonOptions{Variant: input.Primary}).Clicked {
				a.saveSettings()
			}
			ui.Box(c).Grow(1)
		})
	})
}

type settingsPage int

const (
	settingsGeneral settingsPage = iota
	settingsCards
	settingsCapture
	settingsShortcuts
	settingsPrivacy
)

func settingsTabLabels(lang string) []string {
	return []string{
		tr(lang, "tab_general"),
		tr(lang, "tab_cards"),
		tr(lang, "tab_capture"),
		tr(lang, "tab_shortcuts"),
		tr(lang, "tab_privacy"),
	}
}

func (a *App) settingsPage(c *ui.Context, f *config.Form, lang string) {
	switch settingsPage(f.SettingsTab) {
	case settingsGeneral:
		a.settingsGeneral(c, f, lang)
	case settingsCards:
		a.settingsCards(c, f, lang)
	case settingsCapture:
		a.settingsCapture(c, f, lang)
	case settingsShortcuts:
		a.settingsShortcuts(c, f, lang)
	case settingsPrivacy:
		a.settingsPrivacy(c, f, lang)
	default:
		panic("snip: unhandled settings tab")
	}
}

func (a *App) settingsGeneral(c *ui.Context, f *config.Form, lang string) {
	k := pixel.K(c)
	a.settingsSection(c, tr(lang, "language"), func() {
		input.Select(c, tr(lang, "language"), &f.Language, []string{"en", "zh"}, input.SelectOptions{})
	})
	a.settingsSection(c, tr(lang, "clipboard"), func() {
		labeledInput(c, tr(lang, "poll"), &f.ClipboardPollMs)
		labeledInput(c, tr(lang, "max_cards"), &f.MaxCards)
		text.Paragraph(c, tr(lang, "placement"), text.ParagraphOptions{MaxLines: 1, Slot: pixel.Ink})
		input.Select(c, tr(lang, "placement"), &f.Placement, []string{"near-selection", "near-cursor", "fixed"}, input.SelectOptions{})
	})
	a.settingsSection(c, tr(lang, "config_path"), func() {
		ui.Text(c, a.configPath).FontSize(13).TextColor(k.C(pixel.Faint)).MaxLines(3)
	})
}

func (a *App) settingsCards(c *ui.Context, f *config.Form, lang string) {
	a.settingsSection(c, tr(lang, "palette"), func() {
		pixelChoice(c, tr(lang, "palette"), &f.Palette, config.PaletteIDs, func(id string) string {
			return tr(lang, "palette_"+strings.ReplaceAll(id, "-", "_"))
		})
	})
	a.settingsSection(c, tr(lang, "windows"), func() {
		input.Checkbox(c, tr(lang, "always_on_top"), &f.AlwaysOnTop, input.CheckboxOptions{})
		input.Checkbox(c, tr(lang, "all_spaces"), &f.VisibleOnAllWorkspaces, input.CheckboxOptions{})
		input.Checkbox(c, tr(lang, "capture"), &f.ContentProtection, input.CheckboxOptions{})
		input.Checkbox(c, tr(lang, "panel"), &f.Panel, input.CheckboxOptions{})
		text.Paragraph(c, tr(lang, "restart"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Faint})
	})
	a.settingsSection(c, tr(lang, "size"), func() {
		labeledInput(c, tr(lang, "min_w"), &f.MinWidth)
		labeledInput(c, tr(lang, "max_w"), &f.MaxWidth)
		labeledInput(c, tr(lang, "max_height"), &f.MaxHeight)
		labeledInput(c, tr(lang, "pad_x"), &f.PaddingX)
		labeledInput(c, tr(lang, "pad_y"), &f.PaddingY)
		labeledInput(c, tr(lang, "font_family"), &f.FontFamily)
		labeledInput(c, tr(lang, "font"), &f.FontSize)
		input.Checkbox(c, tr(lang, "double_click_paste"), &f.DoubleClickPaste, input.CheckboxOptions{})
		labeledInput(c, "Cascade origin x", &f.CascadeOriginX)
		labeledInput(c, "Cascade origin y", &f.CascadeOriginY)
		labeledInput(c, "Cascade step x", &f.CascadeStepX)
		labeledInput(c, "Cascade step y", &f.CascadeStepY)
	})
	a.settingsSection(c, tr(lang, "motion"), func() {
		input.Checkbox(c, tr(lang, "motion_on"), &f.MotionEnabled, input.CheckboxOptions{})
		labeledInput(c, tr(lang, "motion_ms"), &f.MotionMs)
	})
}

func (a *App) settingsCapture(c *ui.Context, f *config.Form, lang string) {
	a.settingsSection(c, tr(lang, "capture_filter"), func() {
		text.Paragraph(c, tr(lang, "capture_mode"), text.ParagraphOptions{MaxLines: 1, Slot: pixel.Ink})
		a.captureModeSelect(c, lang, &f.CaptureMode)
		a.captureList(c, f, lang)
	})
}

func (a *App) settingsShortcuts(c *ui.Context, f *config.Form, lang string) {
	a.settingsSection(c, tr(lang, "hotkeys"), func() {
		labeledInput(c, tr(lang, "leader_shortcut"), &f.LeaderShortcut)
		labeledInput(c, tr(lang, "leader_timeout"), &f.LeaderTimeout)
		labeledInput(c, tr(lang, "double_ms"), &f.DoublePressMs)
		labeledInput(c, tr(lang, "which_key_x"), &f.WhichKeyX)
		labeledInput(c, tr(lang, "which_key_y"), &f.WhichKeyY)
		text.Paragraph(c, tr(lang, "which_key_align_x"), text.ParagraphOptions{MaxLines: 1, Slot: pixel.Ink})
		input.Select(c, tr(lang, "which_key_align_x"), &f.WhichKeyAlignX, []string{config.AlignStart, config.AlignCenter, config.AlignEnd}, input.SelectOptions{})
		text.Paragraph(c, tr(lang, "which_key_align_y"), text.ParagraphOptions{MaxLines: 1, Slot: pixel.Ink})
		input.Select(c, tr(lang, "which_key_align_y"), &f.WhichKeyAlignY, []string{config.AlignStart, config.AlignCenter, config.AlignEnd}, input.SelectOptions{})
		for _, item := range f.Bindings.NamedBindings() {
			settingsBinding(c, tr(lang, item.Name), item.Binding, lang)
		}
	})
}

func (a *App) settingsPrivacy(c *ui.Context, f *config.Form, lang string) {
	a.settingsSection(c, tr(lang, "secrets"), func() {
		input.Checkbox(c, tr(lang, "mask"), &f.MaskEnabled, input.CheckboxOptions{})
		labeledInput(c, tr(lang, "entropy"), &f.MinEntropyBits)
		labeledInput(c, tr(lang, "secret_len"), &f.MinSecretLength)
		text.Paragraph(c, tr(lang, "patterns"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Ink})
		input.TextArea(c, tr(lang, "patterns"), &f.ExtraPatterns, input.TextAreaOptions{Lines: 4, Width: 200})
		text.Paragraph(c, tr(lang, "markers"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Faint})
		input.Checkbox(c, tr(lang, "concealed"), &f.HonorConcealed, input.CheckboxOptions{})
		input.Checkbox(c, tr(lang, "transient"), &f.SkipTransient, input.CheckboxOptions{})
		input.Checkbox(c, tr(lang, "auto"), &f.SkipAutoGenerated, input.CheckboxOptions{})
	})
}

func (a *App) settingsSection(c *ui.Context, title string, body func()) {
	k := pixel.K(c)
	layout.Panel(c, layout.PanelOptions{Style: pixel.Raised, Padding: 4}, func() {
		ui.Column(c).Shrink(0).Gap(k.D(4)).Children(func() {
			text.Heading(c, title, text.HeadingOptions{Level: 3, Slot: pixel.Ink})
			body()
		})
	}).Shrink(0)
}

func (a *App) captureModeSelect(c *ui.Context, lang string, mode *string) {
	options := []string{config.CaptureAll, config.CaptureBlocklist, config.CaptureAllowlist}
	pixelChoice(c, tr(lang, "capture_mode"), mode, options, func(opt string) string {
		return tr(lang, "capture_mode_"+opt)
	})
}

func (a *App) captureList(c *ui.Context, f *config.Form, lang string) {
	k := pixel.K(c)
	var ids *[]string
	switch f.CaptureMode {
	case config.CaptureBlocklist:
		ids = &f.Blocklist
	case config.CaptureAllowlist:
		ids = &f.Allowlist
	case config.CaptureAll:
		text.Paragraph(c, tr(lang, "capture_all_hint"), text.ParagraphOptions{MaxLines: 3, Slot: pixel.Faint})
		return
	default:
		panic("snip: unhandled capture mode " + f.CaptureMode)
	}
	if len(*ids) == 0 {
		text.Paragraph(c, tr(lang, "capture_empty"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Faint})
	}
	removeAt := -1
	for i, id := range *ids {
		i, id := i, id
		ui.Row(c).Gap(k.D(4)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(k.D(1)).Children(func() {
				ui.Text(c, a.runningAppName(id)).FontSize(13).TextColor(k.C(pixel.Ink)).MaxLines(1)
				ui.Text(c, id).FontSize(11).TextColor(k.C(pixel.Faint)).MaxLines(1)
			})
			if input.Button(c, tr(lang, "capture_remove"), input.ButtonOptions{}).Clicked {
				removeAt = i
			}
		})
	}
	if removeAt >= 0 && removeAt < len(*ids) {
		*ids = append((*ids)[:removeAt], (*ids)[removeAt+1:]...)
	}
	choices := a.captureChoices(*ids)
	if len(choices) == 0 {
		text.Paragraph(c, tr(lang, "capture_no_apps"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Faint})
		return
	}
	text.Paragraph(c, tr(lang, "capture_pick"), text.ParagraphOptions{MaxLines: 1, Slot: pixel.Ink})
	a.captureAppSelect(c, f, choices)
	if input.Button(c, tr(lang, "capture_add"), input.ButtonOptions{}).Clicked && f.CapturePick != "" && !bundleIn(*ids, f.CapturePick) {
		*ids = append(*ids, f.CapturePick)
		f.CapturePick = ""
	}
}

func (a *App) captureAppSelect(c *ui.Context, f *config.Form, choices []macpaste.RunningApp) {
	if f.CapturePick == "" || !bundleInApps(choices, f.CapturePick) {
		f.CapturePick = choices[0].BundleID
	}
	options := make([]string, len(choices))
	for i, app := range choices {
		options[i] = app.BundleID
	}
	pixelChoice(c, "app", &f.CapturePick, options, func(id string) string {
		return a.runningAppName(id)
	})
}

// choiceDown is the select chevron. The kit's copy is unexported.
var choiceDown = pixel.MustSprite(`
	#####
	.###.
	..#..
`)

// pixelChoice is a pixel-kit select: notched face, left-aligned pixel
// text, and a chevron. textOf is the row the value shows.
func pixelChoice(c *ui.Context, label string, value *string, options []string, textOf func(string) string) {
	k := pixel.K(c)
	var sel ui.SelectParts[string]
	var shell ui.Element
	var st pixel.State
	shell, _, st = pixel.Shell(c, func() ui.Element {
		sel = ui.SelectBase(c, value)
		return sel.Trigger.Label(label)
	})
	shown := textOf(*value)
	shell.Height(k.D(pixel.LineHeight + 4)).Draw(func(p *ui.Painter, r ui.Rect) {
		g := k.Canvas(p, r)
		if g.W < 3 || g.H < 3 {
			return
		}
		ink, dy := choiceFace(g, st, sel.Open())
		g.Text(6, 2+dy, shown, ink, 1)
		g.Sprite(g.W-10, (g.H-choiceDown.H)/2+dy, choiceDown, 0, ink, 1)
		g.Finish(st)
	})
	sel.Popup(func(panel ui.Element) {
		panel.Padding(k.D(3)).Draw(func(p *ui.Painter, r ui.Rect) {
			g := k.Canvas(p, r)
			if g.W < 3 || g.H < 3 {
				return
			}
			g.Frame(0, 0, g.W, g.H, pixel.Window, pixel.Face)
		}).Children(func() {
			ui.Column(c).AlignItems(ui.Stretch).Children(func() {
				for _, opt := range options {
					opt := opt
					row := textOf(opt)
					item := sel.Item(opt).Label(row).Height(k.D(pixel.LineHeight)).MinWidth(k.D(pixel.Measure(row) + 16))
					lit := item.Highlighted()
					chosen := opt == *value
					item.Draw(func(p *ui.Painter, r ui.Rect) {
						g := k.Canvas(p, r)
						ink := pixel.Ink
						if chosen {
							ink = pixel.Accent
						}
						if lit {
							g.Rect(0, 0, g.W, g.H, pixel.Lit)
							g.Sprite(1, (g.H-pixel.Hand.H)/2, pixel.Hand, 0, pixel.Edge, 1)
						}
						g.Text(13, 0, row, ink, 1)
					})
				}
			})
		})
	})
}

// choiceFace paints the closed select the way the kit paints a normal control.
func choiceFace(g pixel.Canvas, st pixel.State, on bool) (pixel.Slot, int) {
	dy := 0
	if st.Press || on {
		dy = 1
	}
	fill := pixel.Face
	if on {
		fill = pixel.Shade
	}
	g.Frame(0, dy, g.W, g.H-dy, pixel.Notched, fill)
	switch {
	case on || st.Press:
		g.HLine(1, dy+1, g.W-2, pixel.Edge)
	default:
		g.Bevel(1, 1, g.W-2, g.H-2, pixel.Lit, pixel.Shade)
		if st.Hover {
			g.Veil(2, 2, g.W-4, g.H-4, pixel.Lit, 4)
		}
	}
	return pixel.Ink, dy
}

func (a *App) runningAppName(id string) string {
	for _, app := range a.settingsApps {
		if strings.EqualFold(app.BundleID, id) && app.Name != "" {
			return app.Name
		}
	}
	return id
}

func (a *App) captureChoices(already []string) []macpaste.RunningApp {
	var out []macpaste.RunningApp
	for _, app := range a.settingsApps {
		if app.BundleID == "" || bundleIn(already, app.BundleID) {
			continue
		}
		out = append(out, app)
	}
	return out
}

func bundleIn(ids []string, id string) bool {
	for _, item := range ids {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(id)) {
			return true
		}
	}
	return false
}

func bundleInApps(apps []macpaste.RunningApp, id string) bool {
	for _, app := range apps {
		if strings.EqualFold(app.BundleID, id) {
			return true
		}
	}
	return false
}

func labeledInput(c *ui.Context, label string, value *string) {
	k := pixel.K(c)
	ui.Row(c).Gap(k.D(4)).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			text.Paragraph(c, label, text.ParagraphOptions{MaxLines: 2, Slot: pixel.Ink})
		})
		input.TextInput(c, label, value, input.TextInputOptions{Width: 90})
	})
}

func settingsBinding(c *ui.Context, label string, binding *config.ActionBinding, lang string) {
	k := pixel.K(c)
	ui.Column(c).Gap(k.D(3)).Padding(k.D(2), 0).Children(func() {
		text.Heading(c, label, text.HeadingOptions{Level: 3, Slot: pixel.Ink})
		labeledInput(c, tr(lang, "direct_binding"), &binding.Direct)
		input.Checkbox(c, tr(lang, "double_press"), &binding.DirectDouble, input.CheckboxOptions{})
		labeledInput(c, tr(lang, "leader_binding"), &binding.Leader)
		ui.Row(c).Gap(k.D(6)).Children(func() {
			input.Checkbox(c, tr(lang, "double_press"), &binding.LeaderDouble, input.CheckboxOptions{})
			input.Checkbox(c, tr(lang, "sticky"), &binding.Sticky, input.CheckboxOptions{})
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

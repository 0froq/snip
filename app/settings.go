package app

import (
	"log"
	"strconv"
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
			{Label: tr(lang, "menu_settings"), Accelerator: settingsAccelerator(cfg), Click: func(*mygo.MenuItem, *mygo.Window) {
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
		if a.rec.On {
			a.rec = config.Recorder{}
			a.restoreShortcuts()
		}
		a.bindNotes = nil
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
	a.pollRecording(c.Now())
	// Escape is a window shortcut, so it is taken before the record catcher.
	// While a row is recording it cancels that row and leaves Settings open.
	if c.Shortcut(ui.Cmd, ui.KeyW) {
		a.closeSettings()
	} else if c.Shortcut(0, ui.KeyEscape) {
		if a.rec.On {
			a.cancelRecording()
		} else {
			a.closeSettings()
		}
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
			pixelTabs(c, tr(lang, "settings"), settingsTabLabels(lang), &f.SettingsTab)
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
			a.recordCatcher(c)
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

// pixelTabs is the settings tab bar. The kit's highlight stops one dot
// short of each segment, which leaves a Face gap before the next divider.
// A middle tab paints through its right edge. The last tab stops on the
// frame, and the first tab leaves the left frame.
func pixelTabs(c *ui.Context, label string, labels []string, selected *int) {
	k := pixel.K(c)
	v := *selected
	p := ui.SegmentedBase(c, &v, len(labels))
	w := 0
	for _, l := range labels {
		w += pixel.Measure(l) + 12
	}
	last := len(labels) - 1
	p.Track.Width(k.D(w)).Label(label).Draw(func(pt *ui.Painter, r ui.Rect) {
		g := k.Canvas(pt, r)
		if g.W < 3 || g.H < 3 {
			return
		}
		g.Frame(0, 0, g.W, g.H, pixel.Notched, pixel.Face)
	}).Children(func() {
		for i, l := range labels {
			i, l := i, l
			shell, _, st := pixel.Shell(c, func() ui.Element { return p.Segment(i).Label(l) })
			shell.Size(k.D(pixel.Measure(l)+12), k.D(pixel.LineHeight+4)).Draw(func(pt *ui.Painter, r ui.Rect) {
				g := k.Canvas(pt, r)
				if g.W < 3 || g.H < 3 {
					return
				}
				fillW := g.W - 1
				if i == last {
					fillW = g.W - 2
				}
				ink := pixel.Ink
				switch {
				case *selected == i:
					g.Rect(1, 1, fillW, g.H-2, pixel.Accent)
					ink = pixel.AccentInk
				case st.Hover:
					g.Rect(1, 1, fillW, g.H-2, pixel.Lit)
				}
				if i > 0 {
					g.VLine(0, 1, g.H-2, pixel.Edge)
				}
				g.Text(6, 2, l, ink, 1)
				g.Finish(st)
			})
		}
	})
	if v != *selected && !pixel.Current(c).Inert {
		*selected = v
	}
}

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
	issues := config.CheckBindings(f.Bindings)
	var addAction string
	removeAt := -1
	a.settingsSection(c, tr(lang, "leader"), func() {
		text.Paragraph(c, tr(lang, "leader_key_hint"), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Faint})
		a.bindingRows(c, f, lang, "leader", issues, &removeAt)
		if input.Button(c, tr(lang, "bind_add"), input.ButtonOptions{Label: tr(lang, "bind_add") + " leader"}).Clicked {
			addAction = "leader"
		}
		shortcutField(c, tr(lang, "leader_timeout"), &f.LeaderTimeout)
		shortcutField(c, tr(lang, "double_ms"), &f.DoublePressMs)
		shortcutField(c, tr(lang, "which_key_x"), &f.WhichKeyX)
		shortcutField(c, tr(lang, "which_key_y"), &f.WhichKeyY)
		aligns := []string{config.AlignStart, config.AlignCenter, config.AlignEnd}
		shortcutChoice(c, tr(lang, "which_key_align_x"), &f.WhichKeyAlignX, aligns)
		shortcutChoice(c, tr(lang, "which_key_align_y"), &f.WhichKeyAlignY, aligns)
	})
	for _, group := range []struct {
		title string
		names []string
	}{
		{"group_cards", []string{"next", "prev", "paste", "delete", "clear_cards", "pin"}},
		{"group_visibility", []string{"toggle"}},
		{"group_app", []string{"settings"}},
	} {
		title, names := group.title, group.names
		a.settingsSection(c, tr(lang, title), func() {
			text.Paragraph(c, tr(lang, "bind_hint"), text.ParagraphOptions{MaxLines: 3, Slot: pixel.Faint})
			for _, name := range names {
				name := name
				text.Paragraph(c, tr(lang, name), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Ink})
				a.bindingRows(c, f, lang, name, issues, &removeAt)
				if input.Button(c, tr(lang, "bind_add"), input.ButtonOptions{Label: tr(lang, "bind_add") + " " + name}).Clicked {
					addAction = name
				}
			}
		})
	}
	for _, name := range extraShortcutActions(f.Bindings) {
		name := name
		a.settingsSection(c, name, func() {
			a.bindingRows(c, f, lang, name, issues, &removeAt)
			if input.Button(c, tr(lang, "bind_add"), input.ButtonOptions{Label: tr(lang, "bind_add") + " " + name}).Clicked {
				addAction = name
			}
		})
	}
	if addAction != "" || removeAt >= 0 {
		if a.rec.On {
			a.cancelRecording()
		}
		a.bindNotes = nil
	}
	if addAction != "" {
		f.Bindings = insertBinding(f.Bindings, addAction)
	}
	if removeAt >= 0 {
		f.Bindings = removeBinding(f.Bindings, removeAt)
	}
}

func extraShortcutActions(rows config.BindingList) []string {
	known := map[string]bool{
		"leader": true, "toggle": true, "next": true, "prev": true, "paste": true,
		"delete": true, "clear_cards": true, "pin": true, "settings": true,
	}
	var out []string
	seen := map[string]bool{}
	for _, row := range rows {
		if known[row.Action] || seen[row.Action] {
			continue
		}
		seen[row.Action] = true
		out = append(out, row.Action)
	}
	return out
}

func (a *App) bindingRows(c *ui.Context, f *config.Form, lang, action string, issues []*config.BindingIssue, removeAt *int) {
	k := pixel.K(c)
	for i := range f.Bindings {
		if f.Bindings[i].Action != action {
			continue
		}
		i := i
		ui.Column(c).Shrink(0).Gap(k.D(1)).Children(func() {
			ui.Row(c).Gap(k.D(4)).AlignItems(ui.Center).Shrink(0).Children(func() {
				recording := a.rec.On && a.rec.Row == i
				input.TextInput(c, action+" "+strconv.Itoa(i), &f.Bindings[i].Keys, input.TextInputOptions{Width: shortcutFieldDots, Disabled: recording})
				opts := input.ButtonOptions{Label: tr(lang, "bind_record") + " " + action + " " + strconv.Itoa(i)}
				if recording {
					opts.Variant = input.Primary
				}
				if input.Button(c, tr(lang, "bind_record"), opts).Clicked {
					a.beginRecord(i)
				}
				if input.Button(c, tr(lang, "bind_remove"), input.ButtonOptions{Label: tr(lang, "bind_remove") + " " + action + " " + strconv.Itoa(i)}).Clicked {
					*removeAt = i
				}
			})
			if action != "leader" && config.HasLeaderPrefix(f.Bindings[i].Keys) {
				input.Checkbox(c, tr(lang, "sticky"), &f.Bindings[i].Stay, input.CheckboxOptions{})
			}
			issue := (*config.BindingIssue)(nil)
			if a.rec.On && a.rec.Row == i {
				issue = nil
			} else {
				if i < len(issues) {
					issue = issues[i]
				}
				if note, ok := a.bindNotes[i]; ok && note.keys == f.Bindings[i].Keys {
					issue = &config.BindingIssue{Code: note.code}
				}
			}
			if issue != nil {
				text.Paragraph(c, bindingIssueText(lang, issue), text.ParagraphOptions{MaxLines: 2, Slot: pixel.Bad})
			}
		})
	}
}

func bindingIssueText(lang string, issue *config.BindingIssue) string {
	switch issue.Code {
	case "empty":
		return tr(lang, "bind_empty")
	case "unknown":
		return tr(lang, "bind_unknown") + " " + issue.Token
	case "conflict":
		return tr(lang, "bind_conflict")
	case "leader":
		return tr(lang, "bind_leader_key")
	case "action":
		return tr(lang, "bind_action") + " " + issue.Token
	case "occupied":
		return tr(lang, "bind_occupied")
	case "leader_follow":
		return tr(lang, "bind_leader_follow")
	default:
		panic("snip: unknown binding issue " + issue.Code)
	}
}

func insertBinding(rows config.BindingList, action string) config.BindingList {
	row := config.Binding{Action: action}
	last := -1
	for i, item := range rows {
		if item.Action == action {
			last = i
		}
	}
	if last < 0 {
		return append(rows, row)
	}
	rows = append(rows, config.Binding{})
	copy(rows[last+2:], rows[last+1:])
	rows[last+1] = row
	return rows
}

func removeBinding(rows config.BindingList, index int) config.BindingList {
	if index < 0 || index >= len(rows) {
		return rows
	}
	return append(rows[:index], rows[index+1:]...)
}

func settingsAccelerator(cfg config.Config) string {
	for _, row := range cfg.Bindings {
		if row.Action != "settings" {
			continue
		}
		parsed, err := config.ParseBinding(row.Keys)
		if err != nil || parsed.Leader {
			continue
		}
		return parsed.Chord
	}
	return ""
}

// bindingGroup is the shortcuts section an action belongs to.
// A new action has to be placed, or this panics.
func bindingGroup(name string) string {
	switch name {
	case "next", "prev", "paste", "delete", "clear_cards", "pin":
		return "cards"
	case "toggle":
		return "visibility"
	case "settings":
		return "app"
	default:
		panic("snip: shortcut " + name + " has no group")
	}
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

// shortcutLabelDots is the label column. shortcutFieldDots is the key
// column. Every shortcut row uses both, so the fields share a left edge.
const (
	shortcutLabelDots = 120
	shortcutFieldDots = 150
)

func shortcutLine(c *ui.Context, label string, control func()) {
	k := pixel.K(c)
	ui.Row(c).Gap(k.D(4)).AlignItems(ui.Center).Shrink(0).Children(func() {
		ui.Box(c).Width(k.D(shortcutLabelDots)).Shrink(0).Children(func() {
			if label != "" {
				text.Paragraph(c, label, text.ParagraphOptions{MaxLines: 2, Slot: pixel.Ink})
			}
		})
		control()
	})
}

func shortcutField(c *ui.Context, label string, value *string) {
	shortcutLine(c, label, func() {
		input.TextInput(c, label, value, input.TextInputOptions{Width: shortcutFieldDots})
	})
}

func shortcutChoice(c *ui.Context, label string, value *string, options []string) {
	k := pixel.K(c)
	shortcutLine(c, label, func() {
		ui.Box(c).Width(k.D(shortcutFieldDots)).Shrink(0).Children(func() {
			pixelChoice(c, label, value, options, func(opt string) string { return opt })
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

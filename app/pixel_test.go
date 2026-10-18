package app

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/input"
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/layout"
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/text"
	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

func TestCardFrameStepsTheCorner(t *testing.T) {
	img := renderCard(t, true, false, pixel.AutumnDay)
	// The corner dot is left empty. The tester fills that with Back;
	// a real window is clear, so the desktop shows through.
	assertDot(t, img, 0, 0, pixel.AutumnDay, pixel.Back)
	assertDot(t, img, 1, 0, pixel.AutumnDay, pixel.Accent)
	idle := renderCard(t, false, false, pixel.AutumnNight)
	assertDot(t, idle, 0, 0, pixel.AutumnNight, pixel.Back)
	assertDot(t, idle, 1, 0, pixel.AutumnNight, pixel.Edge)
}

func assertDot(t *testing.T, img *image.RGBA, x, y int, pal *pixel.Palette, slot pixel.Slot) {
	t.Helper()
	got := img.RGBAAt(x*pixelDot, y*pixelDot)
	want := pal.C(slot)
	if got.R != want.R || got.G != want.G || got.B != want.B {
		t.Fatalf("dot %d,%d %+v want %s %+v", x, y, got, slot, want)
	}
}

func TestAppearancePaletteFollowsNight(t *testing.T) {
	if appearancePalette(false) != pixel.AutumnDay || appearancePalette(true) != pixel.AutumnNight {
		t.Fatalf("day %s night %s", appearancePalette(false).Name, appearancePalette(true).Name)
	}
	day := resolvePalette(config.PaletteAuto, false)
	night := resolvePalette(config.PaletteWinterNight, true)
	if day != pixel.AutumnDay || night != pixel.WinterNight {
		t.Fatalf("auto %s named %s", day.Name, night.Name)
	}
}

func TestShortcutIssuesAreInline(t *testing.T) {
	a := New(config.Default(), "")
	form := config.FormFrom(a.config())
	form.SettingsTab = int(settingsShortcuts)
	form.Bindings = append(form.Bindings,
		config.Binding{Action: "paste", Keys: ""},
		config.Binding{Action: "delete", Keys: "cmd+shift+v"},
	)
	a.settingsForm = &form
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 720)
	for _, s := range []string{"Enter a shortcut.", "Already used."} {
		if !tt.HasText(s) {
			t.Fatalf("missing %q in %v", s, tt.Texts())
		}
	}
}

func TestShortcutGroupsCoverActions(t *testing.T) {
	cfg := config.Default()
	seen := map[string]int{}
	seenAction := map[string]bool{}
	for _, row := range cfg.Bindings {
		if row.Action == "leader" || seenAction[row.Action] {
			continue
		}
		seenAction[row.Action] = true
		seen[bindingGroup(row.Action)]++
	}
	if seen["cards"] != 6 || seen["visibility"] != 1 || seen["app"] != 1 || len(seen) != 3 {
		t.Fatalf("groups %v", seen)
	}
}

func TestShortcutPageGroups(t *testing.T) {
	a := New(config.Default(), "")
	form := config.FormFrom(a.config())
	form.SettingsTab = int(settingsShortcuts)
	a.settingsForm = &form
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 720)
	texts := tt.Texts()
	for _, s := range []string{"Leader key", "Starts leader mode.", "Cards", "Visibility", "App", "Next card", "Stay in leader mode"} {
		if !tt.HasText(s) {
			t.Fatalf("missing %q in %v", s, texts)
		}
	}
	hintAt, nextAt := -1, -1
	for i, s := range texts {
		if s == "Starts leader mode." && hintAt < 0 {
			hintAt = i
		}
		if s == "Next card" && nextAt < 0 {
			nextAt = i
		}
	}
	if hintAt < 0 || nextAt < 0 || hintAt > nextAt {
		t.Fatalf("leader hint %d next card %d", hintAt, nextAt)
	}
}

func TestSelectedTabMeetsTheDivider(t *testing.T) {
	pal := pixel.AutumnDay
	face := pal.C(pixel.Face)
	for _, tab := range []settingsPage{settingsGeneral, settingsCards, settingsPrivacy} {
		a := New(config.Default(), "")
		form := config.FormFrom(a.config())
		form.SettingsTab = int(tab)
		a.settingsForm = &form
		img := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 420).Image()
		_, y, end := longestAccentRun(t, img, pal, 180)
		next := img.RGBAAt(end, y)
		if next.R == face.R && next.G == face.G && next.B == face.B {
			t.Fatalf("tab %d highlight ends on Face at %d,%d", tab, end, y)
		}
	}
}

func longestAccentRun(t *testing.T, img *image.RGBA, pal *pixel.Palette, maxY int) (x, y, end int) {
	t.Helper()
	accent := pal.C(pixel.Accent)
	best := 0
	b := img.Bounds()
	if maxY > b.Max.Y {
		maxY = b.Max.Y
	}
	for py := b.Min.Y; py < maxY; py++ {
		run, at := 0, 0
		for px := b.Min.X; px < b.Max.X; px++ {
			c := img.RGBAAt(px, py)
			if c.R == accent.R && c.G == accent.G && c.B == accent.B {
				if run == 0 {
					at = px
				}
				run++
				if run > best && px+1 < b.Max.X {
					best, x, y, end = run, at, py, px+1
				}
				continue
			}
			run = 0
		}
	}
	if best < pixelDot*8 {
		t.Fatalf("no tab highlight")
	}
	return x, y, end
}

func TestWritePixelCardShots(t *testing.T) {
	dir := os.Getenv("PIXEL_SHOTS")
	if dir == "" {
		t.Skip()
	}
	writeShot(t, dir, "card-active-day.png", renderCard(t, true, false, pixel.AutumnDay))
	writeShot(t, dir, "card-idle-night.png", renderCard(t, false, true, pixel.AutumnNight))
	writeShot(t, dir, "settings-day.png", renderSettings(t))
	writeShot(t, dir, "which-key-day.png", renderWhichKey(t))
	writeShot(t, dir, "close-button.png", renderClose(t))
}

func renderWhichKey(t *testing.T) *image.RGBA {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		k := usePixel(c, config.Config{Language: "en"}, false)
		pad := k.D(cardFrameDots + 3)
		ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			g := k.Canvas(p, r)
			if g.W >= 3 && g.H >= 3 {
				g.Frame(0, 0, g.W, g.H, pixel.Window, pixel.Face)
			}
		}).Children(func() {
			ui.Column(c).Padding(pad).Gap(k.D(2)).Children(func() {
				for _, row := range [][2]string{{"s", "show"}, {"n", "next"}, {"p", "paste"}} {
					key, name := row[0], row[1]
					ui.Row(c).Gap(k.D(4)).Children(func() {
						ui.Box(c).Size(k.D(pixel.Measure(key)), k.D(pixel.LineHeight)).Draw(func(p *ui.Painter, r ui.Rect) {
							k.Canvas(p, r).Text(0, 0, key, pixel.Accent, 1)
						})
						ui.Box(c).Size(k.D(pixel.Measure(name)), k.D(pixel.LineHeight)).Draw(func(p *ui.Painter, r ui.Rect) {
							k.Canvas(p, r).Text(0, 0, name, pixel.Ink, 1)
						})
					})
				}
			})
		})
	}, 180, 120)
	return tt.Image()
}

func renderClose(t *testing.T) *image.RGBA {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		k := usePixel(c, config.Config{Language: "en"}, false)
		ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			drawCardChrome(k.Canvas(p, r), true, false)
		}).Children(func() {
			cardCloseButton(c, true)
			ui.Column(c).Padding(float32(config.CardTitleStripPx), 16, 12, 16).Children(func() {
				ui.Text(c, "hello 你好").FontSize(14)
			})
		})
	}, 220, 72)
	return tt.Image()
}

func renderSettings(t *testing.T) *image.RGBA {
	t.Helper()
	tab := 1
	on := true
	tt := ui.NewTester(func(c *ui.Context) {
		k := usePixel(c, config.Config{Language: "en"}, false)
		pad := k.D(cardFrameDots + 3)
		ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			g := k.Canvas(p, r)
			if g.W >= 3 && g.H >= 3 {
				g.Frame(0, 0, g.W, g.H, pixel.Window, pixel.Face)
			}
		}).Children(func() {
			ui.Column(c).Fill().Padding(pad).Gap(k.D(4)).Children(func() {
				text.Heading(c, "Settings", text.HeadingOptions{Level: 2, Slot: pixel.Ink})
				input.Segmented(c, "Settings", []string{"General", "Cards", "Capture", "Shortcuts", "Privacy"}, &tab, input.SegmentedOptions{})
				layout.Panel(c, layout.PanelOptions{Style: pixel.Raised, Padding: 4}, func() {
					text.Heading(c, "Card windows", text.HeadingOptions{Level: 3, Slot: pixel.Ink})
					input.Checkbox(c, "Always on top", &on, input.CheckboxOptions{})
				})
				input.Button(c, "Save", input.ButtonOptions{Variant: input.Primary})
			})
		})
	}, 640, 300)
	return tt.Image()
}

func renderCard(t *testing.T, active, editing bool, pal *pixel.Palette) *image.RGBA {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		pixel.Use(c, pixel.Settings{Palette: pal, Dot: pixelDot})
		k := pixel.K(c)
		ui.Column(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			drawCardChrome(k.Canvas(p, r), active, editing)
		}).Children(func() {
			ui.Column(c).Padding(k.D(cardFrameDots + 4)).Children(func() {
				ui.Text(c, "hello 你好").FontSize(14)
				ui.Text(c, "func main() {}").FontSize(13)
			})
		})
	}, 220, 88)
	return tt.Image()
}

func writeShot(t *testing.T, dir, name string, img *image.RGBA) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

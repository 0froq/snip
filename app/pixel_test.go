package app

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
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
}

func TestWritePixelCardShots(t *testing.T) {
	dir := os.Getenv("PIXEL_SHOTS")
	if dir == "" {
		t.Skip()
	}
	writeShot(t, dir, "card-active-day.png", renderCard(t, true, false, pixel.AutumnDay))
	writeShot(t, dir, "card-idle-night.png", renderCard(t, false, true, pixel.AutumnNight))
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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWhichKeyCornerDefault(t *testing.T) {
	got := Default().Leader.EffectiveWhichKeyCorner()
	if got != WhichKeyBottomRight {
		t.Fatalf("default corner %q", got)
	}
	if Default().Leader.WhichKeyCorner != WhichKeyBottomRight {
		t.Fatalf("default field %q", Default().Leader.WhichKeyCorner)
	}
}

func TestWhichKeyCornerParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := "[leader]\nwhich_key_corner = \"top-left\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyTopLeft {
		t.Fatalf("parsed %q", got.Leader.WhichKeyCorner)
	}

	if err := os.WriteFile(path, []byte("[leader]\ntimeout_ms = 900\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyBottomRight {
		t.Fatalf("omitted corner %q", got.Leader.WhichKeyCorner)
	}
	if got.Leader.TimeoutMs != 900 {
		t.Fatalf("timeout %d", got.Leader.TimeoutMs)
	}

	if err := os.WriteFile(path, []byte("[leader]\nwhich_key_corner = \"CENTER\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyBottomRight {
		t.Fatalf("invalid corner %q", got.Leader.EffectiveWhichKeyCorner())
	}
}

func TestWhichKeyCornerForm(t *testing.T) {
	f := FormFrom(Default())
	if f.WhichKeyCorner != WhichKeyBottomRight {
		t.Fatalf("form default %q", f.WhichKeyCorner)
	}
	f.WhichKeyCorner = "Top-Left"
	got, err := f.Parse(Default())
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.WhichKeyCorner != WhichKeyTopLeft {
		t.Fatalf("parsed form %q", got.Leader.WhichKeyCorner)
	}
	f.WhichKeyCorner = "middle"
	if _, err := f.Parse(Default()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRepoConfigWhichKeyCorner(t *testing.T) {
	got, err := Load(filepath.Join("..", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyBottomRight {
		t.Fatalf("repo config %q", got.Leader.WhichKeyCorner)
	}
}

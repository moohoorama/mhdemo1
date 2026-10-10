package sprite

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sprites = "../../assets/sprites"

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func wantErr(t *testing.T, err error, sub string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), sub) {
		t.Errorf("error %v, want one containing %q", err, sub)
	}
}

func loadSheets(t *testing.T) map[string]*Sheet {
	sheets, err := LoadDir(sprites)
	must(t, err)
	if len(sheets) == 0 {
		t.Fatal("no sprites")
	}
	return sheets
}

func golden(t *testing.T, id string) *image.RGBA {
	f, err := os.Open(filepath.Join("testdata/units", id+".png"))
	must(t, err)
	defer f.Close()
	im, _, err := image.Decode(f)
	must(t, err)
	out := image.NewRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out
}

func TestRenderMatchesExporter(t *testing.T) {
	for id, s := range loadSheets(t) {
		t.Run(id, func(t *testing.T) {
			want, got := golden(t, id), s.Render(nil)
			if want.Bounds() != got.Bounds() {
				t.Fatalf("bounds %v, want %v", got.Bounds(), want.Bounds())
			}
			if !bytes.Equal(want.Pix, got.Pix) {
				t.Error("render differs from the exporter's PNG")
			}
		})
	}
}

func TestFactionOverride(t *testing.T) {
	s := loadSheets(t)["liubei"]
	base := s.Render(nil)
	red := Shades(color.RGBA{200, 30, 30, 255})
	got := s.Render(Palette{"faction": red[:]})
	faction := map[byte]bool{}
	for i, tg := range s.Tags {
		if tg.Group == "faction" {
			faction[byte(i+1)] = true
		}
	}
	if len(faction) != 3 {
		t.Fatalf("faction tags %v", faction)
	}
	w, h := s.Cell[0], s.Cell[1]
	changed, opaque := 0, 0
	for i, tg := range s.tag {
		c := i / (w * h)
		x, y := c%s.Columns*w+i%w, c/s.Columns*h+i/w%h
		o := (y*w*s.Columns + x) * 4
		if base.Pix[o+3] != got.Pix[o+3] {
			t.Fatalf("alpha changed at %d,%d", x, y)
		}
		if !bytes.Equal(base.Pix[o:o+4], got.Pix[o:o+4]) {
			changed++
			if !faction[tg] {
				t.Fatalf("pixel %d,%d with tag %d changed", x, y, tg)
			}
		}
		if faction[tg] && s.Colors[s.pix[i]].A == 255 {
			opaque++
		}
	}
	if opaque == 0 || changed < opaque {
		t.Errorf("changed %d pixels, want at least the %d opaque faction pixels", changed, opaque)
	}
	if !bytes.Equal(base.Pix, s.Render(Palette{"nonexistent": red[:]}).Pix) {
		t.Error("unknown group changed the render")
	}
}

func TestPalette(t *testing.T) {
	cs, err := ParseColors(json.RawMessage(`"#336699"`))
	must(t, err)
	if len(cs) != 3 || cs[1] != (color.RGBA{0x33, 0x66, 0x99, 255}) {
		t.Errorf("single colour: %v", cs)
	}
	cs, err = ParseColors(json.RawMessage(`["#ffffff","#808080","#000000"]`))
	must(t, err)
	if cs[2] != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("list: %v", cs)
	}
	for _, bad := range []string{`"red"`, `["#fff"]`, `["#ffffff","#000000"]`, `3`} {
		if _, err = ParseColors(json.RawMessage(bad)); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
	a := Palette{"faction": cs}
	b := a.Merge(Palette{"themeA": cs, "faction": {{R: 1, A: 255}}})
	if b["faction"][0] != (color.RGBA{1, 0, 0, 255}) || b["faction"][1] != cs[1] || b["themeA"] == nil {
		t.Errorf("merge: %v", b)
	}
	if a["faction"][0] != cs[0] {
		t.Error("Merge modified its receiver")
	}
	if a.Key() != (Palette{"faction": append([]color.RGBA(nil), cs...)}).Key() || a.Key() == b.Key() {
		t.Error("Key is not stable or not distinguishing")
	}
}

func TestLibraryCache(t *testing.T) {
	lib := NewLibrary(loadSheets(t), color.RGBA{10, 10, 30, 255})
	red := Shades(color.RGBA{200, 30, 30, 255})
	blue := Shades(color.RGBA{30, 30, 200, 255})
	r1 := lib.Art("liubei", Palette{"faction": red[:]})
	if r1 == nil || r1 != lib.Art("liubei", Palette{"faction": red[:]}) {
		t.Error("same palette must reuse the Art")
	}
	if r1 == lib.Art("liubei", Palette{"faction": blue[:]}) || r1 == lib.Art("liubei", nil) {
		t.Error("different palettes must cache separately")
	}
	if lib.Art("nobody", nil) != nil {
		t.Error("unknown sprite must give nil")
	}
	if r1.Frame("S", "idle", 0) == nil || r1.Shadow("S", "idle", 0) == nil || r1.TopOffset() <= 0 {
		t.Error("frame, shadow or top offset missing")
	}
	nc := lib.Art("liubei_noncombat", nil)
	if nc.FrameIndex("walk", 400) != 2 || nc.Frame("N", "walk", 3) == nil {
		t.Errorf("noncombat: walk index %d", nc.FrameIndex("walk", 400))
	}
}

func TestCorruptFiles(t *testing.T) {
	good, err := os.ReadFile(filepath.Join(sprites, "liubei_noncombat.spr"))
	must(t, err)
	_, err = Decode(good)
	must(t, err)
	for _, n := range []int{0, 3, 5, 6, 10, len(good) / 2, len(good) - 1} {
		if _, err := Decode(good[:n]); err == nil {
			t.Errorf("truncated to %d bytes accepted", n)
		}
	}
	_, err = Decode(append([]byte("XXXX"), good[4:]...))
	wantErr(t, err, "magic")
	bad := append([]byte(nil), good...)
	bad[4] = 9
	_, err = Decode(bad)
	wantErr(t, err, "version")

	z, err := zlib.NewReader(bytes.NewReader(good[6:]))
	must(t, err)
	payload, err := io.ReadAll(z)
	must(t, err)
	repack := func(p []byte) []byte {
		var b bytes.Buffer
		b.Write(good[:6])
		w := zlib.NewWriter(&b)
		w.Write(p)
		w.Close()
		return b.Bytes()
	}
	_, err = Decode(repack(payload))
	must(t, err)
	_, err = Decode(repack(payload[:len(payload)-1]))
	wantErr(t, err, "pixel data")
	_, err = Decode(repack(append(append([]byte(nil), payload...), 0)))
	wantErr(t, err, "pixel data")
	bad = append([]byte(nil), payload...)
	bad[len(bad)-1] = 255
	_, err = Decode(repack(bad))
	wantErr(t, err, "out of range")
	_, err = Decode(repack(payload[:20]))
	wantErr(t, err, "header")

	p := filepath.Join(t.TempDir(), "broken.spr")
	must(t, os.WriteFile(p, good[:len(good)/2], 0o644))
	_, err = Load(p)
	wantErr(t, err, "broken.spr")
	_, err = LoadDir(filepath.Dir(p))
	wantErr(t, err, "broken.spr")
}

func TestShadow(t *testing.T) {
	for id, s := range loadSheets(t) {
		t.Run(id, func(t *testing.T) {
			sd := s.Shadow()
			if want := image.Rect(0, 0, sd.Cell[0]*s.Columns, sd.Cell[1]*len(s.Rows)); sd.Image.Bounds() != want {
				t.Fatalf("bounds %v, want %v", sd.Image.Bounds(), want)
			}
			if sd.Pivot != [2]int{s.Pivot[0] + shadowPad, shadowPad} {
				t.Errorf("pivot %v", sd.Pivot)
			}
			cells := len(s.Rows) * s.Columns
			filled := 0
			for c := 0; c < cells; c++ {
				x0, y0, has := c%s.Columns*sd.Cell[0], c/s.Columns*sd.Cell[1], false
				for y := 0; y < sd.Cell[1]; y++ {
					for x := 0; x < sd.Cell[0]; x++ {
						p := sd.Image.RGBAAt(x0+x, y0+y)
						if p.R|p.G|p.B != 0 {
							t.Fatal("shadow is not black")
						}
						has = has || p.A == core
					}
				}
				if has {
					filled++
				}
			}
			if filled < cells*3/4 {
				t.Errorf("only %d of %d cells have a shadow", filled, cells)
			}
		})
	}
}

func TestShadowMatchesPython(t *testing.T) {
	want := golden(t, "infantry.shadow")
	got := loadSheets(t)["infantry"].Shadow().Image
	if want.Bounds() != got.Bounds() || !bytes.Equal(want.Pix, got.Pix) {
		t.Error("shadow differs from ver3/shadows.py output")
	}
}

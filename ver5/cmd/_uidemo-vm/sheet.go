package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// sheetCell is the width each screen is shrunk to on a sheet; two cells make a row.
const sheetCell = 960

// runSheet plays the demo once per reference screen, each run in a child process of this
// tool (Ebitengine runs one game per process), then lays the -at frames out two by two.
func runSheet() error {
	frameIDs := atFrames()
	if len(frameIDs) == 0 {
		return fmt.Errorf("-sheet needs the frames to compare in -at")
	}
	sort.Ints(frameIDs)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	dirs := make([]string, len(screens))
	for i, sc := range screens {
		dirs[i] = filepath.Join(*out, fmt.Sprintf("screen-%d", i))
		cmd := exec.Command(self, "-ui", *ui, "-frames", strconv.Itoa(*frames), "-at", *at,
			"-stage", *stage, "-duel", *duel, "-node", *node, "-out", dirs[i],
			"-size", fmt.Sprintf("%dx%d", sc.w, sc.h), "-uiscale", strconv.Itoa(sc.uiScale))
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		fmt.Fprintf(os.Stderr, "%s: ", sc.name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", sc.name, err)
		}
	}
	for _, f := range frameIDs {
		name := filepath.Join(*out, fmt.Sprintf("sheet-%04d.png", f))
		if err := writeSheet(name, f, dirs); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, name)
	}
	return nil
}

// writeSheet puts frame f of every screen on one image, each under its name.
func writeSheet(name string, f int, dirs []string) error {
	const label = 24
	cells := make([]image.Image, len(dirs))
	rowH := make([]int, (len(dirs)+1)/2)
	for i, dir := range dirs {
		file, err := os.Open(filepath.Join(dir, fmt.Sprintf("frame-%04d.png", f)))
		if err != nil {
			return err
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			return err
		}
		b := img.Bounds()
		h := b.Dy() * sheetCell / b.Dx()
		cell := image.NewRGBA(image.Rect(0, 0, sheetCell, h))
		draw.ApproxBiLinear.Scale(cell, cell.Bounds(), img, b, draw.Src, nil)
		cells[i] = cell
		rowH[i/2] = max(rowH[i/2], h+label)
	}
	total := 0
	for _, h := range rowH {
		total += h
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 2*sheetCell, total))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.RGBA{40, 40, 44, 255}), image.Point{}, draw.Src)
	y := 0
	for i, cell := range cells {
		x := (i % 2) * sheetCell
		sc := screens[i]
		d := font.Drawer{Dst: sheet, Src: image.White, Face: basicfont.Face7x13, Dot: fixed.P(x+8, y+17)}
		d.DrawString(fmt.Sprintf("%s  %dx%d", sc.name, sc.w, sc.h))
		draw.Draw(sheet, cell.Bounds().Add(image.Pt(x, y+label)), cell, image.Point{}, draw.Src)
		if i%2 == 1 {
			y += rowH[i/2]
		}
	}
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, sheet)
}

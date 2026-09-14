// spritetool provides general-purpose PNG sprite processing and fixed-grid atlas packing.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
)

type options struct {
	crop        rectValue
	size        sizeValue
	frames      int
	frame       int
	trim        bool
	outline     int
	atlas       atlasValue
	atlasPivot  pointValue
	spritePivot anchorValue
}

func main() {
	var o options
	flag.Var(&o.crop, "crop", "crop rectangle x,y,width,height")
	flag.Var(&o.size, "size", "output size WIDTHxHEIGHT for each frame")
	flag.IntVar(&o.frames, "frames", 1, "number of horizontal source frames")
	flag.IntVar(&o.frame, "frame", -1, "zero-based frame to extract; default processes every frame")
	flag.BoolVar(&o.trim, "trim", false, "trim transparent margins, shared across all source frames")
	flag.IntVar(&o.outline, "outline", 0, "opaque black outline width in pixels")
	flag.Var(&o.atlas, "atlas", "atlas CELL_WIDTHxCELL_HEIGHT,COLUMNSxROWS")
	flag.Var(&o.atlasPivot, "atlas-pivot", "destination pivot x,y; defaults to cell center,bottom")
	flag.Var(&o.spritePivot, "sprite-pivot", "source pivot, e.g. center,bottom-2")
	flag.Parse()

	args := flag.Args()
	if o.atlas.set {
		if len(args) < 2 {
			log.Fatal("atlas mode needs at least one input PNG and one output PNG")
		}
		if !o.atlasPivot.set {
			o.atlasPivot.point = image.Pt(o.atlas.cell.X/2, o.atlas.cell.Y)
		}
		if !o.spritePivot.set {
			o.spritePivot = anchorValue{set: true, x: axisAnchor{center: true}, y: axisAnchor{end: true}}
		}
		if err := packAtlas(args[:len(args)-1], args[len(args)-1], o.atlas, o.atlasPivot.point, o.spritePivot); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: spritetool [options] input.png output.png")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if o.frames < 1 || o.frame >= o.frames || o.frame < -1 || o.outline < 0 {
		log.Fatal("invalid frame or outline option")
	}
	if err := processFile(args[0], args[1], o); err != nil {
		log.Fatal(err)
	}
}

func decodePNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return im, nil
}

func encodePNG(path string, im image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, im); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func parsePair(s, separator string) (int, int, error) {
	var a, b int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d"+separator+"%d", &a, &b); err != nil {
		return 0, 0, fmt.Errorf("invalid pair %q", s)
	}
	return a, b, nil
}

type sizeValue struct {
	set  bool
	size image.Point
}

func (v *sizeValue) String() string { return fmt.Sprintf("%dx%d", v.size.X, v.size.Y) }
func (v *sizeValue) Set(s string) error {
	w, h, err := parsePair(s, "x")
	if err != nil || w < 1 || h < 1 {
		return fmt.Errorf("invalid size %q", s)
	}
	v.set, v.size = true, image.Pt(w, h)
	return nil
}

type rectValue struct {
	set  bool
	rect image.Rectangle
}

func (v *rectValue) String() string { return fmt.Sprint(v.rect) }
func (v *rectValue) Set(s string) error {
	var x, y, w, h int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d,%d,%d,%d", &x, &y, &w, &h); err != nil || w < 1 || h < 1 {
		return fmt.Errorf("invalid crop %q", s)
	}
	v.set, v.rect = true, image.Rect(x, y, x+w, y+h)
	return nil
}

type pointValue struct {
	set   bool
	point image.Point
}

func (v *pointValue) String() string { return fmt.Sprintf("%d,%d", v.point.X, v.point.Y) }
func (v *pointValue) Set(s string) error {
	x, y, err := parsePair(s, ",")
	if err != nil {
		return err
	}
	v.set, v.point = true, image.Pt(x, y)
	return nil
}

type atlasValue struct {
	set        bool
	cell, grid image.Point
}

func (v *atlasValue) String() string {
	return fmt.Sprintf("%dx%d,%dx%d", v.cell.X, v.cell.Y, v.grid.X, v.grid.Y)
}
func (v *atlasValue) Set(s string) error {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid atlas %q", s)
	}
	cw, ch, err := parsePair(parts[0], "x")
	if err != nil {
		return err
	}
	cols, rows, err := parsePair(parts[1], "x")
	if err != nil || cw < 1 || ch < 1 || cols < 1 || rows < 1 {
		return fmt.Errorf("invalid atlas %q", s)
	}
	v.set, v.cell, v.grid = true, image.Pt(cw, ch), image.Pt(cols, rows)
	return nil
}

type axisAnchor struct {
	center bool
	end    bool
	offset int
}

func (a axisAnchor) resolve(length int) int {
	switch {
	case a.center:
		return length/2 + a.offset
	case a.end:
		return length + a.offset
	default:
		return a.offset
	}
}

type anchorValue struct {
	set  bool
	x, y axisAnchor
}

func (v *anchorValue) String() string { return "center,bottom" }
func (v *anchorValue) Set(s string) error {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid sprite pivot %q", s)
	}
	parse := func(s string, horizontal bool) (axisAnchor, error) {
		a := axisAnchor{}
		base := s
		if i := strings.LastIndexAny(s[1:], "+-"); i >= 0 {
			i++
			base = s[:i]
			if _, err := fmt.Sscanf(s[i:], "%d", &a.offset); err != nil {
				return a, fmt.Errorf("invalid anchor offset %q", s)
			}
		}
		switch base {
		case "center", "middle":
			a.center = true
		case "right":
			if !horizontal {
				return a, fmt.Errorf("right is only valid on x axis")
			}
			a.end = true
		case "bottom":
			if horizontal {
				return a, fmt.Errorf("bottom is only valid on y axis")
			}
			a.end = true
		case "left", "top":
		default:
			if _, err := fmt.Sscanf(base, "%d", &a.offset); err != nil {
				return a, fmt.Errorf("invalid anchor %q", s)
			}
		}
		return a, nil
	}
	var err error
	if v.x, err = parse(parts[0], true); err != nil {
		return err
	}
	if v.y, err = parse(parts[1], false); err != nil {
		return err
	}
	v.set = true
	return nil
}

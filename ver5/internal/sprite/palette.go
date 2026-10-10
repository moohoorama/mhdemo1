package sprite

import (
	"encoding/json"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
)

// Palette maps a tag group id to its colours; element i is shade i+1 (light, mid, dark).
type Palette map[string][]color.RGBA

// Shades derives light, mid and dark from one base colour; mid is the base itself.
func Shades(base color.RGBA) [3]color.RGBA {
	mix := func(c color.RGBA, to uint8, n int) color.RGBA {
		f := func(v uint8) uint8 { return uint8((int(v)*(100-n) + int(to)*n) / 100) }
		return color.RGBA{f(c.R), f(c.G), f(c.B), 255}
	}
	base.A = 255
	return [3]color.RGBA{mix(base, 255, 30), base, mix(base, 0, 30)}
}

func parseHex(s string) (color.RGBA, error) {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	if len(s) != 7 || s[0] != '#' || err != nil {
		return color.RGBA{}, fmt.Errorf("bad colour %q (want #rrggbb)", s)
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, nil
}

// ParseColors reads a group's colours from JSON: "#rrggbb" (shades derived) or a list of 1 or 3 such strings.
func ParseColors(raw json.RawMessage) ([]color.RGBA, error) {
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		var one string
		if json.Unmarshal(raw, &one) != nil {
			return nil, fmt.Errorf("colours %s: want \"#rrggbb\" or a list", raw)
		}
		list = []string{one}
	}
	if len(list) != 1 && len(list) != 3 {
		return nil, fmt.Errorf("colours %s: want 1 or 3 entries, got %d", raw, len(list))
	}
	out := make([]color.RGBA, len(list))
	for i, s := range list {
		c, err := parseHex(s)
		if err != nil {
			return nil, err
		}
		out[i] = c
	}
	if len(out) == 1 {
		sh := Shades(out[0])
		return sh[:], nil
	}
	return out, nil
}

// Merge returns p with o applied on top, shade by shade (entries with zero alpha are unset); neither input changes.
func (p Palette) Merge(o Palette) Palette {
	out := Palette{}
	for g, cs := range p {
		out[g] = append([]color.RGBA(nil), cs...)
	}
	for g, cs := range o {
		if len(out[g]) < len(cs) {
			out[g] = append(out[g], make([]color.RGBA, len(cs)-len(out[g]))...)
		}
		for i, c := range cs {
			if c.A != 0 {
				out[g][i] = c
			}
		}
	}
	return out
}

// Key is a stable string for caching: equal palettes give equal keys.
func (p Palette) Key() string {
	groups := make([]string, 0, len(p))
	for g := range p {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	var b strings.Builder
	for _, g := range groups {
		b.WriteString(g)
		for _, c := range p[g] {
			fmt.Fprintf(&b, ":%02x%02x%02x", c.R, c.G, c.B)
		}
		b.WriteByte(';')
	}
	return b.String()
}

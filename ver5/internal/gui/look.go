package gui

import (
	"encoding/json"
	"image/color"
	"strings"

	"srpg/internal/content"
	"srpg/internal/sprite"
)

// character finds a character definition by id, troop id (<template>#<n>) or display name.
func (g *Game) character(ref string) (string, content.Character, bool) {
	d := g.S.Data
	if c, ok := d.Characters[ref]; ok {
		return ref, c, true
	}
	if id, ok := g.names[ref]; ok {
		return id, d.Characters[id], true
	}
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		c, ok := d.Characters[ref[:i]]
		return ref[:i], c, ok
	}
	return "", content.Character{}, false
}

// nameIndex maps display names to the characters they belong to (troop templates excluded).
func nameIndex(d *content.Data) map[string]string {
	out := map[string]string{}
	for id, c := range d.Characters {
		if !c.Template {
			out[c.Name] = id
		}
	}
	return out
}

// name is the shown name of a unit or character; a troop's number follows its template's name.
func (g *Game) name(id string) string {
	if def, c, ok := g.character(id); ok {
		if i := strings.IndexByte(id, '#'); i >= 0 && def != id {
			return c.Name + id[i+1:]
		}
		return c.Name
	}
	return id
}

func colorsOf(c []string) []color.RGBA {
	b, _ := json.Marshal(c)
	out, _ := sprite.ParseColors(b)
	return out
}

// palette is the faction's colours with the character's own over them.
func (g *Game) palette(faction string, look content.Look) sprite.Palette {
	p := sprite.Palette{}
	if f, ok := g.S.Data.Factions[faction]; ok && len(f.Colors) > 0 {
		p["faction"] = colorsOf(f.Colors)
	}
	own := sprite.Palette{}
	for group, c := range look.Palette {
		own[group] = colorsOf(c)
	}
	return p.Merge(own)
}

// art draws a sprite in a faction's colours over the character's palette.
func (g *Game) art(id, faction string, look content.Look) *sprite.Art {
	return g.assets.Lib.Art(id, g.palette(faction, look))
}

// battleArt is the character's battle sprite, or its class's.
func (g *Game) battleArt(def, faction string) *sprite.Art {
	c := g.S.Data.Characters[def]
	if a := g.art(c.Look.Sprite, faction, c.Look); a != nil {
		return a
	}
	return g.art(g.S.Data.Classes[c.Class].Sprite, faction, c.Look)
}

// sceneArt is the sprite a character stands on a scene in: the cast's own pick, else the
// character's scene sprite, battle sprite or class sprite.
func (g *Game) sceneArt(def, pick, faction string) *sprite.Art {
	c := g.S.Data.Characters[def]
	for _, id := range []string{pick, c.Look.Scene, c.Look.Sprite, g.S.Data.Classes[c.Class].Sprite} {
		if a := g.art(id, faction, c.Look); id != "" && a != nil {
			return a
		}
	}
	return nil
}

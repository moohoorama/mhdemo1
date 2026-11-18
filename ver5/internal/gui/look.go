package gui

import (
	"encoding/json"
	"image/color"
	"maps"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/sprite"

	"github.com/hajimehoshi/ebiten/v2"
)

// portraitArt is the character's own portrait, or else that of the class it serves as (its
// own class when class is empty).
func (g *Game) portraitArt(c content.Character, class string) *ebiten.Image {
	c.Look = g.laterLook(c.Look)
	if src := g.assets.portrait(c.Look.Portrait); src != nil {
		return src
	}
	if class == "" {
		class = c.Class
	}
	return g.assets.portrait(g.S.Data.Classes[class].Portrait)
}

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

// officerMark follows a faction colour's name when the sprite takes the faction's officer shades.
const officerMark = "*"

// officerColor is the faction colour of a unit: the officer shades for a named officer or a template led as an
// officer, commander or boss, the troop shades for the rest.
func (g *Game) officerColor(def, faction, rank string) string {
	c := g.S.Data.Characters[def]
	if strings.HasSuffix(faction, officerMark) || c.Template && rank != "officer" && rank != "commander" && rank != "boss" {
		return faction
	}
	return faction + officerMark
}

// unitColor is the faction colour a battle unit is drawn in.
func (g *Game) unitColor(u core.UnitView) string { return g.officerColor(u.Def, u.Color, u.Rank) }

// palette is the faction's colours (its officer shades for a colour marked with officerMark) with the character's own over them.
func (g *Game) palette(faction string, look content.Look) sprite.Palette {
	p := sprite.Palette{}
	name, officer := strings.CutSuffix(faction, officerMark)
	if f, ok := g.S.Data.Factions[name]; ok && len(f.Colors) > 0 {
		cols := f.Colors
		if officer && len(f.OfficerColors) > 0 {
			cols = f.OfficerColors
		}
		p["faction"] = colorsOf(cols)
	}
	own := sprite.Palette{}
	for group, c := range look.Palette {
		own[group] = colorsOf(c)
	}
	p = p.Merge(own)
	if f, ok := g.S.Data.Factions[name]; ok && officer {
		for group, c := range f.OfficerPalette {
			p[group] = colorsOf(c)
		}
	}
	return p
}

// art draws a sprite in a faction's colours over the character's palette.
func (g *Game) art(id, faction string, look content.Look) *sprite.Art {
	return g.assets.Lib.Art(id, g.palette(faction, look))
}

// battleArt is the character's battle sprite, or its class's.
func (g *Game) battleArt(def, faction string) *sprite.Art {
	c := g.S.Data.Characters[def]
	return g.battleClassArt(def, c.Class, g.startingWeapon(c), "", faction)
}

// startingWeapon is the type of the weapon a character's definition equips.
func (g *Game) startingWeapon(c content.Character) string {
	for _, id := range c.Equipment {
		if e := g.S.Data.Equipment[id]; e.Slot == "weapon" {
			return e.Type
		}
	}
	return ""
}

// battleClassArt is the sprite of a character serving in class with a weapon of the given type: its entry for
// that class and weapon, for the class, its default sprite in its own class, or else the class's own (recoloured).
// A mount (an equipped horse of its own colour, UnitView.Mount) lays its colours over the character's.
func (g *Game) battleClassArt(def, class, weapon, mount, faction string) *sprite.Art {
	c := g.S.Data.Characters[def]
	faction = g.officerColor(def, faction, "")
	c.Look = g.mountLook(c.Look, mount)
	ids := []string{c.Look.Sprites[class]}
	if class == g.S.Data.ClassLine(c.Class)[0] {
		ids = append(ids, c.Look.Sprite)
	}
	if weapon != "" {
		ids = append([]string{c.Look.Sprites[class+":"+weapon]}, ids...)
	}
	for _, id := range ids {
		if a := g.art(id, faction, c.Look); id != "" && a != nil {
			return a
		}
	}
	return g.art(g.S.Data.Classes[class].Sprite, faction, c.Look)
}

// mountLook is the character's look with the colours of the horse it rides laid over it.
func (g *Game) mountLook(look content.Look, mount string) content.Look {
	pal := g.S.Data.Equipment[mount].Palette
	if len(pal) == 0 {
		return look
	}
	own := maps.Clone(look.Palette)
	if own == nil {
		own = map[string]content.Colors{}
	}
	maps.Copy(own, pal)
	look.Palette = own
	return look
}

// shootArt is the look a unit takes while shooting from range: the character's sprite for the
// shoot class, else that class's own (never the sprite the character carries into every class).
func (g *Game) shootArt(def, class, mount, faction string) *sprite.Art {
	c := g.S.Data.Characters[def]
	faction = g.officerColor(def, faction, "")
	c.Look = g.mountLook(c.Look, mount)
	if id := c.Look.Sprites[class]; id != "" {
		if a := g.art(id, faction, c.Look); a != nil {
			return a
		}
	}
	return g.art(g.S.Data.Classes[class].Sprite, faction, c.Look)
}

// sceneArt is the sprite a character stands on a scene in: the cast's own pick, else the
// character's scene sprite, the scene sprite of its class (an officer on foot instead of a horseman), battle sprite
// or class sprite, with the colours of the horse it rides (mount, an equipment id) when it rides one.
func (g *Game) sceneArt(def, pick, faction, mount string) *sprite.Art {
	c := g.S.Data.Characters[def]
	faction = g.officerColor(def, faction, "")
	c.Look = g.mountLook(g.laterLook(c.Look), mount)
	first := g.S.Data.ClassLine(c.Class)[0]
	class := g.S.Data.Classes[first]
	for _, id := range []string{pick, c.Look.Scene, class.Scene, c.Look.Sprites[first], c.Look.Sprite, class.Sprite} {
		if a := g.art(id, faction, c.Look); id != "" && a != nil {
			return a
		}
	}
	return nil
}

// laterLook is the look with the portrait and scene sprite of the last later look whose conditions hold.
func (g *Game) laterLook(l content.Look) content.Look {
	if g.S == nil || g.S.Engine == nil {
		return l
	}
	for _, later := range l.Later {
		if g.S.Engine.Holds(later.If) {
			if later.Portrait != "" {
				l.Portrait = later.Portrait
			}
			if later.Scene != "" {
				l.Scene = later.Scene
			}
		}
	}
	return l
}

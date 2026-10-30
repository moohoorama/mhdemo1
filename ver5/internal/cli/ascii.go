package cli

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"srpg/internal/content"
	"srpg/internal/core"
)

// A map is drawn one cell to four columns: a three-letter glyph and a side mark. Units are
// abbreviated from their character id ("Liu" for liubei), troops from their class ("lin" for
// light_infantry); the player's units are capitalized, the enemy's lowercase, and the mark after
// a glyph tells the side: ' ' the player, '+' an AI ally, '!' the enemy.

var groundGlyph = map[byte]string{'g': "   ", 'd': " . ", 'l': " ' ", 's': " ; ", 'w': "~~~", 'c': "###", 'i': " _ ", 'h': "^^^", 'm': "[ ]", 'u': " n ", 'n': "MMM"}
var structureGlyph = map[byte]string{'t': "TRE", 'v': "VIL", 'k': "CMP", 'o': "CST", 'p': "PAS", 'b': "BRG", 'z': "FNC"}
var stateGlyph = map[byte]string{'e': "EMB", 'F': "FIR", 'x': "xxx", 'O': "[O]"}

// cellGlyph is a battle cell's glyph: its state, else its structure, else its ground. Soot and
// damage leave the structure in lowercase.
func cellGlyph(c content.Cell) string {
	if g, ok := stateGlyph[c.State]; ok {
		return g
	}
	if c.Structure != content.NoLayer {
		g := structureGlyph[c.Structure]
		if g == "" {
			g = " " + string(c.Structure) + " "
		}
		if c.State == 's' || c.State == 'd' {
			g = strings.ToLower(g)
		}
		return g
	}
	if g, ok := groundGlyph[c.Ground]; ok {
		return g
	}
	return " " + string(c.Ground) + " "
}

// classCode shortens a class id to three letters: light_infantry → lin, archer → arc.
func classCode(class string) string {
	words := strings.Split(class, "_")
	last := words[len(words)-1]
	code := ""
	for _, w := range words[:len(words)-1] {
		code += w[:1]
	}
	code += last
	if len(code) < 3 {
		code += "   "
	}
	return code[:3]
}

// abbreviations gives every unit its three-letter glyph: an officer the first three letters of his
// id; when two officers on the field share them, his family name's initial and the first two letters
// of his given name (zhangfei → Zfe, zhangliang → Zli); when those are shared too, the initials of
// his syllables (xiahoudun → Xhd, xiahouyuan → Xhy).
func abbreviations(d *content.Data, units []core.UnitView) map[string]string {
	ways := []func(string) string{
		func(id string) string { return (id + "   ")[:3] },
		func(id string) string {
			s := syllables(id)
			return (s[0][:1] + strings.Join(s[1:], "") + "   ")[:3]
		},
		func(id string) string {
			code := ""
			for _, s := range syllables(id) {
				code += s[:1]
			}
			return (code + id[len(id)-1:] + "   ")[:3]
		},
	}
	glyph := map[string]string{}
	for _, way := range ways {
		count := map[string]int{}
		for _, u := range units {
			if !d.Characters[u.Def].Template && glyph[u.Def] == "" {
				count[way(u.Def)]++
			}
		}
		for _, u := range units {
			if c := way(u.Def); !d.Characters[u.Def].Template && glyph[u.Def] == "" && count[c] == 1 {
				glyph[u.Def] = c
			}
		}
	}
	out := map[string]string{}
	for _, u := range units {
		g := glyph[u.Def]
		switch {
		case d.Characters[u.Def].Template:
			g = classCode(u.Class)
		case g == "":
			g = ways[2](u.Def)
		}
		if u.Faction == "enemy" {
			g = strings.ToLower(g)
		} else {
			r := []rune(strings.ToLower(g))
			r[0] = unicode.ToUpper(r[0])
			g = string(r)
		}
		out[u.ID] = g
	}
	return out
}

var pinyin = regexp.MustCompile(`^(zh|ch|sh|[bpmfdtnlgkhjqxrzcsyw])?(iang|iong|uang|ang|eng|ing|ong|ian|iao|uai|uan|ai|ei|ao|ou|an|en|in|un|ia|ie|iu|ua|uo|ui|ue|er|a|o|e|i|u|v)`)

// syllables splits a pinyin id into its syllables; what does not read as pinyin stays in one piece.
func syllables(id string) []string {
	var out []string
	for rest := id; rest != ""; {
		m := pinyin.FindString(rest)
		if m == "" {
			return append(out, rest)
		}
		out = append(out, m)
		rest = rest[len(m):]
	}
	return out
}

func sideMark(u core.UnitView) string {
	switch {
	case u.Faction == "enemy":
		return "!"
	case u.AI:
		return "+"
	}
	return " "
}

// grid writes a width × height board whose cell (x, y) is drawn by cell as four columns.
func grid(out io.Writer, w, h int, cell func(x, y int) string) {
	fmt.Fprint(out, "    ")
	for x := range w {
		fmt.Fprintf(out, "%-4d", x)
	}
	fmt.Fprintln(out)
	for y := range h {
		fmt.Fprintf(out, "%2d |", y)
		for x := range w {
			fmt.Fprint(out, cell(x, y))
		}
		fmt.Fprintf(out, "| %d\n", y)
	}
}

// Board draws the battle map with its units; marks are cells drawn as " o " (where the selected
// unit may move). It returns the glyph of every unit for the roster.
func Board(d *content.Data, o core.Observation, out io.Writer, marks map[content.Point]bool) map[string]string {
	if o.Map == nil {
		return nil
	}
	abbr := abbreviations(d, o.UnitViews)
	at := map[content.Point]core.UnitView{}
	for _, u := range o.UnitViews {
		if u.HP > 0 {
			at[content.Point{X: u.X, Y: u.Y}] = u
		}
	}
	grid(out, o.Map.Width, o.Map.Height, func(x, y int) string {
		p := content.Point{X: x, Y: y}
		if u, ok := at[p]; ok {
			return abbr[u.ID] + sideMark(u)
		}
		if marks[p] {
			return " o  "
		}
		return cellGlyph(o.Map.Cell(x, y)) + " "
	})
	fmt.Fprintln(out, "방위: 북=왼쪽 위(0,0) · 동=오른쪽 위 · 서=왼쪽 아래 · 남=오른쪽 아래 | 표시: ' '우리 '+'동맹 '!'적")
	fmt.Fprintln(out, "지형: TRE 나무 FNC 울타리(소문자=그을음) CMP 주둔지 CST 성 ### 성벽 _ 성내 [ ] 성문 [O] 열린 성문 ~~~ 물 ^^^ 절벽 FIR 화염 EMB 불씨")
	return abbr
}

// roster lists the living units of one side with their numbers, as the battle screen shows them.
func roster(out io.Writer, o core.Observation, abbr map[string]string, keep func(core.UnitView) bool) {
	for _, u := range o.UnitViews {
		if u.HP <= 0 || !keep(u) {
			continue
		}
		fmt.Fprintf(out, "  %s%s %s Lv%d (%d,%d) HP %d/%d MP %d/%d\n", abbr[u.ID], sideMark(u), u.Name, u.Level, u.X, u.Y, u.HP, u.Stats.MaxHP, u.MP, u.Stats.MaxMP)
	}
}

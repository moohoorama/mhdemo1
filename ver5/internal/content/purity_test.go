package content

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestEngineSourceNamesNoGameContent keeps the engine and its screens free of the game's own names: no string literal in
// non-test Go code under internal/ or cmd/ may equal an id or display name from the real data bundle.
func TestEngineSourceNamesNoGameContent(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	engine := map[string]bool{"ally": true, "enemy": true, "fire": true, "water": true, "earth": true, "weapon": true, "armor": true, "accessory": true,
		"dialogue": true, "choice": true, "preparation": true, "join": true, "grant": true}
	for k := range Statuses {
		engine[k] = true
	}
	for k := range SkillEffects {
		engine[k] = true
	}
	for k := range TraitEffects {
		engine[strings.TrimSuffix(k, ":")] = true
	}
	// Korean words the screens use as labels of engine vocabulary (statuses, the equipment tab) may coincide with data names.
	for _, w := range []string{"장비", "신속", "원시", "반격", "혼란", "봉책", "봉격", "약화"} {
		engine[w] = true
	}
	banned := map[string]string{}
	add := func(kind, s string) {
		if s != "" && !engine[s] && len(s) > 1 {
			banned[s] = kind
		}
	}
	for id, c := range d.Characters {
		add("character", id)
		add("character name", c.Name)
	}
	for id, c := range d.Classes {
		add("class", id)
		add("class name", c.Name)
	}
	for id, c := range d.Equipment {
		add("equipment", id)
		add("equipment name", c.Name)
	}
	for id, c := range d.Items {
		add("item", id)
		add("item name", c.Name)
	}
	for id, c := range d.Skills {
		add("skill", id)
		add("skill name", c.Name)
	}
	for id, c := range d.Traits {
		add("trait", id)
		add("trait name", c.Name)
	}
	for id, f := range d.Factions {
		add("faction", id)
		add("faction name", f.Name)
	}
	for _, s := range d.Stages {
		add("stage", s.ID)
	}
	for _, n := range d.Nodes {
		add("node", n.ID)
	}
	for id := range d.Terrain {
		add("terrain group", id)
	}
	for _, root := range []string{"../../internal", "../../cmd"} {
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if l, ok := n.(*ast.BasicLit); ok && l.Kind == token.STRING {
					if s, err := strconv.Unquote(l.Value); err == nil {
						if kind, bad := banned[s]; bad {
							t.Errorf("%s: %q is a %s of the data bundle", p, s, kind)
						}
					}
				}
				return true
			})
			return nil
		})
	}
}

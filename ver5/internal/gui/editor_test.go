package gui

import (
	"image"
	"regexp"
	"slices"
	"strings"
	"testing"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/mapedit"
	"srpg/internal/session"
)

func TestEditorFacingOnlyDiagonals(t *testing.T) {
	dir := "NW"
	for range 8 {
		dir = nextFacing(dir)
		if !slices.Contains([]string{"NE", "SE", "SW", "NW"}, dir) {
			t.Fatal(dir)
		}
	}
	check(t, dir, "NW")
}

func TestEditorOfficerFiltersAndCounts(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range d.MapEditor.Families {
		for _, army := range d.MapEditor.Armies {
			all := editorOfficers(d, family.ID, "", army.ID, "")
			count := 0
			for _, class := range family.Classes {
				count += len(editorOfficers(d, family.ID, class.ID, army.ID, ""))
			}
			if count != len(all) {
				t.Fatalf("class counts do not sum: %s %s", family.ID, army.ID)
			}
			for _, id := range all {
				c := d.Characters[id]
				if c.Template || d.ClassFamily(c.Class) != family.ID || d.MapEditor.CharacterArmies[id] != army.ID {
					t.Fatal("filter leaked")
				}
				if !slices.Contains(editorOfficers(d, family.ID, c.Class, army.ID, c.Name), id) {
					t.Fatal("search lost officer")
				}
			}
		}
	}
}

func TestEditorTraitChoicesBelongToOfficer(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{S: &session.Session{Data: d}, editor: &mapEditor{doc: mapedit.Clone(d.Maps[d.Stages[0].ID])}}
	for id, c := range d.Characters {
		if c.Template {
			continue
		}
		base := content.Spawn{Character: id}
		traits := g.editorPreview(base).Traits
		owned := append([]string{}, c.Traits...)
		for _, eq := range c.Equipment {
			owned = append(owned, d.Equipment[eq].Traits...)
		}
		for _, cl := range d.Classes {
			for _, lt := range cl.Plan {
				owned = append(owned, lt.Trait)
			}
		}
		for _, trait := range traits {
			if !slices.Contains(owned, trait) {
				t.Fatal("unowned choice")
			}
		}
		removed := []string{}
		base.Traits = &removed
		v := core.SpawnPreview(d, "", base)
		for _, trait := range v.Traits {
			if !slices.Contains(owned, trait) {
				t.Fatal("unowned grant")
			}
		}
	}
}

func TestEditorEdgeScrolling(t *testing.T) {
	r := editorViewport
	for _, tc := range []struct {
		p    image.Point
		x, y int
	}{
		{image.Pt(r.Min.X, r.Min.Y), -1, -1}, {image.Pt(r.Max.X-1, r.Max.Y-1), 1, 1},
		{image.Pt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2), 0, 0}, {image.Pt(r.Max.X, r.Min.Y), 0, 0},
	} {
		x, y := editorEdgeScroll(tc.p, r)
		check(t, x, tc.x)
		check(t, y, tc.y)
	}
}

func TestTerrainWithoutUnitDoesNotClaimBlocked(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range []content.Cell{{Ground: 'g', Structure: content.NoLayer, State: content.NoLayer}, {Ground: 'w', Structure: content.NoLayer, State: content.NoLayer}, {Ground: 'i', Structure: 'p', State: content.NoLayer}, {Ground: 'g', Structure: 't', State: 'F'}} {
		text := strings.Join(terrainInfoLines(d, cell, false, "", ""), " ")
		if strings.Contains(text, "진입 불가") || regexp.MustCompile(`(^|[^0-9])0%`).MatchString(text) {
			t.Fatal(text)
		}
	}
}

func TestTerrainRecoverySeparatesHPAndMP(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{"", d.Classes[d.Characters[d.Campaign.Roster[0]].Class].Terrain} {
		t.Run(family, func(t *testing.T) {
			lines := strings.Join(terrainInfoLines(d, content.Cell{Ground: 'i', Structure: content.NoLayer, State: content.NoLayer}, false, family, "보병"), "\n")
			if !strings.Contains(lines, "지형 회복 효과 없음") {
				t.Fatal(lines)
			}
			lines = strings.Join(terrainInfoLines(d, content.Cell{Ground: 'g', Structure: 'v', State: content.NoLayer}, false, family, "보병"), "\n")
			if !strings.Contains(lines, "병력 회복 +10%") || !strings.Contains(lines, "병법치 회복 ×2") {
				t.Fatal(lines)
			}
			lines = strings.Join(terrainInfoLines(d, content.Cell{Ground: 'g', Structure: 'o', State: content.NoLayer}, false, family, "보병"), "\n")
			if !strings.Contains(lines, "병력 회복 +20%") || !strings.Contains(lines, "병법치 회복 ×3") {
				t.Fatal(lines)
			}
			lines = strings.Join(terrainInfoLines(d, content.Cell{Ground: 'g', Structure: 'v', State: 'e'}, false, family, "보병"), "\n")
			if !strings.Contains(lines, "차례 시작 병력 -10%") || strings.Contains(lines, "병력 회복 +") {
				t.Fatal("embers on a village:", lines)
			}
		})
	}
}

package core

import (
	"math"
	"srpg/internal/content"
)

// Matchup compares identical base attributes at equal level and equipment
// quality. It excludes critical/guard rolls so effective hits can be compared.
type Matchup struct {
	Attacker, Defender, Terrain, Skill                              string
	Level, HP, Damage, HitsToRetreat, MP, Cost, CastsBeforeRecovery int
	Hit                                                             float64
	Move, Range, Recovery                                           int
}

func Matchups(d *content.Data, level int) []Matchup {
	d = copyOf(d)
	out := []Matchup{}
	classes := []string{"경기병", "경보병", "궁병", "사마", "적병"}
	for _, a := range classes {
		for _, b := range classes {
			for _, tile := range []string{".", "f", "c"} {
				da := d.Officers["유비"]
				db := d.Officers["장각"]
				da.Class = a
				db.Class = b
				da.Stats = [6]int{70, 70, 70, 70, 70, 70}
				db.Stats = da.Stats
				da.Traits = nil
				db.Traits = nil
				da.Equipment = nil
				db.Equipment = nil
				d.Officers["유비"] = da
				d.Officers["장각"] = db
				e := New(d, 1)
				e.officer("유비").Class = a
				e.officer("유비").Level = level
				e.officer("유비").Equipment = map[string]string{}
				e.addOfficer("장각")
				e.officer("장각").Class = b
				e.officer("장각").Level = level
				e.officer("장각").Equipment = map[string]string{}
				d.Stages[0].Tiles[5] = ".................."
				d.Stages[0].Tiles[5] = d.Stages[0].Tiles[5][:4] + tile + tile + d.Stages[0].Tiles[5][6:]
				e.spawn("유비", "ally", 4, 5)
				e.spawn("장각", "enemy", 5, 5)
				u, v := e.unit("유비"), e.unit("장각")
				c := Command{Kind: "attack"}
				skill := "attack"
				if a == "사마" {
					c = Command{Kind: "skill", Skill: "초열"}
					skill = "초열"
				}
				sk, _ := e.skillDef(c)
				n := max(1, int(math.Floor(e.damage(u, v, sk))))
				cost := e.cost(u, sk)
				st := e.stats(u)
				casts := 0
				if cost > 0 {
					casts = st.MaxMP / cost
				}
				out = append(out, Matchup{a, b, tile, skill, level, v.HP, n, (v.HP + n - 1) / n, st.MaxMP, cost, casts, e.hitChance(u, v, sk), d.Classes[a].Move, d.Classes[a].Range, st.Recovery})
			}
		}
	}
	return out
}

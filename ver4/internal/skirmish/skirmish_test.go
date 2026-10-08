package skirmish

import (
	"encoding/json"
	"srpg/internal/content"
	"strings"
	"testing"
)

func campaign(t *testing.T) *content.Data {
	t.Helper()
	d, err := content.Load("../../assets/content/campaign.json")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func terrains(t *testing.T) []content.Stage {
	t.Helper()
	maps, err := LoadMaps("../../assets/content/simulator-maps.json")
	if err != nil {
		t.Fatal(err)
	}
	return maps
}

func config() Config {
	return Config{Terrain: "초원", Seed: 1, MaxRounds: 30,
		Ally:  Side{Level: 10, Officers: []Member{{Officer: "관우"}, {Officer: "장비"}}},
		Enemy: Side{Level: 10, Officers: []Member{{Officer: "관우", Level: 5}, {Officer: "여포", Class: "중기병"}}}}
}

func TestBuild(t *testing.T) {
	t.Run("sides, levels and overrides", func(t *testing.T) {
		c := config()
		c.Classes = map[string]json.RawMessage{"경기병": json.RawMessage(`{"Bonus": [20, 8, 4, 9, 8, 16]}`)}
		d, err := Build(campaign(t), terrains(t), c)
		if err != nil {
			t.Fatal(err)
		}
		st := d.Stages[0]
		if st.Goal != "rout" || len(st.Party) != 2 || len(st.Enemies) != 2 {
			t.Fatalf("stage %+v", st)
		}
		ally, enemy := d.Officers[st.Party[0].Officer], d.Officers[st.Enemies[0].Officer]
		if ally.Name != "관우" || enemy.Name != "관우" || ally.ID == enemy.ID || ally.Level != 10 || enemy.Level != 5 {
			t.Fatalf("ally %+v, enemy %+v", ally, enemy)
		}
		if d.Officers[st.Enemies[1].Officer].Class != "중기병" || d.Classes["경기병"].Bonus[0] != 20 || d.Classes["경기병"].HPFactor != 3 {
			t.Fatal("class override")
		}
	})
	t.Run("forest needs a cost", func(t *testing.T) {
		c := config()
		c.Terrain = "숲"
		if _, err := Build(campaign(t), terrains(t), c); err == nil || !strings.Contains(err.Error(), "이동 비용") {
			t.Fatalf("got %v", err)
		}
		c.Terrains = map[string]map[string]json.RawMessage{"기병": {"f": json.RawMessage(`{"Cost": 3}`)}}
		d, err := Build(campaign(t), terrains(t), c)
		if err != nil {
			t.Fatal(err)
		}
		if tr := d.Terrain["기병"]["f"]; tr.Cost != 3 || tr.Factor != .8 {
			t.Fatalf("terrain %+v", tr)
		}
	})
	t.Run("campaign data untouched", func(t *testing.T) {
		base := campaign(t)
		c := config()
		c.Classes = map[string]json.RawMessage{"경기병": json.RawMessage(`{"Move": 9}`)}
		if _, err := Build(base, terrains(t), c); err != nil {
			t.Fatal(err)
		}
		if base.Classes["경기병"].Move == 9 || len(base.Stages) != 3 {
			t.Fatal("Build changed its base")
		}
	})
	t.Run("unknown terrain", func(t *testing.T) {
		c := config()
		c.Terrain = "바다"
		if _, err := Build(campaign(t), terrains(t), c); err == nil {
			t.Fatal("accepted")
		}
	})
}

func TestOverridesRoundTrip(t *testing.T) {
	base := campaign(t)
	work := Clone(base)
	cl := work.Classes["궁병"]
	cl.Bonus[0], cl.Range = 11, 4
	work.Classes["궁병"] = cl
	tr := work.Terrain["보병"]["f"]
	tr.Cost = 2
	work.Terrain["보병"]["f"] = tr
	c := config()
	SetOverrides(&c, base, work)
	if len(c.Classes) != 1 || len(c.Terrains) != 1 || len(c.Terrains["보병"]) != 1 {
		t.Fatalf("overrides %v %v", c.Classes, c.Terrains)
	}
	again := Clone(base)
	if err := Apply(again, c); err != nil {
		t.Fatal(err)
	}
	if again.Classes["궁병"].Bonus[0] != 11 || again.Classes["궁병"].Range != 4 || again.Terrain["보병"]["f"] != tr {
		t.Fatal("overrides did not round-trip")
	}
	b, err := json.Marshal(c.Ally.Officers)
	if err != nil || string(b) != `["관우","장비"]` {
		t.Fatalf("members %s %v", b, err)
	}
}

func TestPlay(t *testing.T) {
	d, err := Build(campaign(t), terrains(t), config())
	if err != nil {
		t.Fatal(err)
	}
	out, err := Play(d, 1, 30)
	if err != nil {
		t.Fatal(err)
	}
	if out.Winner == "draw" || out.Rounds < 1 {
		t.Fatalf("outcome %+v", out)
	}
	standing := map[string]bool{}
	for _, u := range out.Units {
		if u.HP > 0 {
			standing[u.Faction] = true
		}
		if u.Level != d.Officers[u.Officer].Level || u.XP != 0 {
			t.Errorf("%s gained experience", u.ID)
		}
	}
	if standing["ally"] == standing["enemy"] || standing["ally"] != (out.Winner == "ally") {
		t.Fatalf("winner %s, standing %v", out.Winner, standing)
	}
}

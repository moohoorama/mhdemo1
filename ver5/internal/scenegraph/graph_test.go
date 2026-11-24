package scenegraph

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/scenemap"
)

func load(t *testing.T) *content.Data {
	t.Helper()
	d, err := content.LoadScenario("../../assets/data", "hero")
	require.NoError(t, err)
	return d
}

var (
	catalogOnce sync.Once
	catalog     *scenemap.Catalog
)

// env reads the scene maps of assets/graphics/maps and their rules, as the admin does.
func env() Env {
	catalogOnce.Do(func() {
		var err error
		if catalog, err = scenemap.LoadCatalog("../../assets/graphics"); err != nil {
			panic(err)
		}
	})
	return Env{Catalog: catalog, Portrait: func(name string) bool {
		_, err := os.Stat("../../assets/graphics/portraits/" + name + ".png")
		return err == nil
	}, Map: func(id string) *scenemap.Map {
		b, err := os.ReadFile("../../assets/graphics/maps/" + id + ".json")
		if err != nil {
			return nil
		}
		m, ok, err := scenemap.Parse(b)
		if err != nil || !ok {
			return nil
		}
		return m
	}}
}

func TestFirstChapterFollowsTheWalkInOrder(t *testing.T) {
	d := load(t)
	g := Build(d, env(), "c01-walk", Options{Battles: 1})
	var order []string
	for _, s := range g.Steps {
		order = append(order, s.Vertex)
	}
	assert.Equal(t, []string{"event:c01-e-notice", "event:c01-e-tavern", "event:c01-e-oath", "event:c01-sortie", "battle:c01-battle"}, order)
	for _, i := range g.Issues {
		assert.NotEqual(t, "error", i.Level, i.Msg)
	}
}

func TestEveryChapterStartsWithoutErrors(t *testing.T) {
	d := load(t)
	for _, c := range Outline(d) {
		if c.Start == "" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			g := Build(d, env(), c.Start, Options{Battles: 1, Assume: AssumeBefore(d, c.Start)})
			for _, i := range g.Issues {
				assert.NotEqual(t, "error", i.Level, i.Msg)
			}
			assert.NotEmpty(t, g.Steps)
		})
	}
}

func TestLayoutGivesEveryVertexItsOwnCell(t *testing.T) {
	d := load(t)
	g := Build(d, env(), "c02-walk", Options{Battles: 2, Assume: AssumeBefore(d, "c02-walk")})
	seen := map[[2]int]string{}
	for _, v := range g.Vertices {
		p := [2]int{v.X, v.Y}
		assert.NotContains(t, seen, p, "%s and %s share a cell", v.ID, seen[p])
		seen[p] = v.ID
	}
}

// findings are the rule findings of every chapter walked from its start, once each, in "rule where subject" form.
func findings(t *testing.T, d *content.Data, rule, level string) []string {
	t.Helper()
	seen := map[string]bool{}
	var out []string
	for _, c := range Outline(d) {
		if c.Start == "" {
			continue
		}
		g := Build(d, env(), c.Start, Options{Battles: 99, Assume: AssumeBefore(d, c.Start)})
		for _, i := range g.Issues {
			if i.Rule != rule || i.Level != level {
				continue
			}
			if k := i.Where + " " + i.Subject; !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	slices.Sort(out)
	return out
}

func TestEveryExitIntoARoomHasAWayBack(t *testing.T) {
	assert.Empty(t, findings(t, load(t), RuleExitReturn, "error"))
}

// gapsOnlyShrink holds the findings of a warning rule to the list in testdata/<file>: a new finding fails, and so does
// one that was closed but is still listed (the list is a ratchet: it only gets shorter).
func gapsOnlyShrink(t *testing.T, rule, file, hint string) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + file)
	require.NoError(t, err)
	var wantPairs []string // a line each: where, a space, who or what (which may hold spaces)
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			wantPairs = append(wantPairs, l)
		}
	}
	slices.Sort(wantPairs)
	got := findings(t, load(t), rule, "warn")
	if os.Getenv("UPDATE_BASELINES") != "" { // rewrite the list to what is found now, to take a closed gap off it
		require.NoError(t, os.WriteFile("testdata/"+file, []byte(strings.Join(append(got, ""), "\n")), 0o644))
		return
	}
	var added, gone []string
	for _, g := range got {
		if !slices.Contains(wantPairs, g) {
			added = append(added, g)
		}
	}
	for _, w := range wantPairs {
		if !slices.Contains(got, w) {
			gone = append(gone, w)
		}
	}
	assert.Empty(t, added, hint)
	assert.Empty(t, gone, "closed, but still listed in testdata/"+file+"; remove these lines")
}

func TestWhereaboutsGapsOnlyShrink(t *testing.T) {
	gapsOnlyShrink(t, RuleWhereabouts, "whereabouts_baseline.txt",
		"a core ally is in the party but is neither in the walk nor in its Whereabouts; add them to the walk or record where they are")
}

func TestPopulationGapsOnlyShrink(t *testing.T) {
	gapsOnlyShrink(t, RulePopulation, "population_baseline.txt",
		"a settlement has more or fewer people than its Scale; add or remove people, or say why in PopulationNote")
}

func TestAboutTwoInThreePeopleOfASettlementCanBeTalkedTo(t *testing.T) {
	gapsOnlyShrink(t, RuleTalkRatio, "talk_ratio_baseline.txt", "too few of a settlement's people can be talked to; give them something to say, or say why in PopulationNote")
}

func TestShopsWithNewStockAreAnnouncedNearTheirDoor(t *testing.T) {
	gapsOnlyShrink(t, RuleShopNotice, "shop_notice_baseline.txt", "a shop with new stock has nobody near its door telling so (Notice \"shop:<id>\")")
}

func TestTheNewsOfTheTimeIsToldInEveryWalk(t *testing.T) {
	gapsOnlyShrink(t, RuleNews, "news_baseline.txt", "a piece of the scenario's News is told by nobody in a walk of its time (Notice \"news:<id>\")")
}

func TestRulersGuestsAndCourtsHaveHomesOfTheirSize(t *testing.T) {
	gapsOnlyShrink(t, RuleResidence, "residence_baseline.txt", "a lord, guest or court lacks a home of the right size (Residence)")
}

func TestLinesHaveNoHanja(t *testing.T) {
	gapsOnlyShrink(t, RuleHanja, "hanja_baseline.txt", "a line has Chinese characters; the Korean edition says everything in Korean")
}

func TestOnlyTheSwornBrothersCallLiuBeiHyung(t *testing.T) {
	gapsOnlyShrink(t, RuleHyung, "hyung_baseline.txt", "someone other than 관우 or 장비 calls 유비 형/형님; use his title (공, 주공, 현령, 사군 …)")
}

func TestTalkLinesHaveTextAndRealSpeakers(t *testing.T) {
	gapsOnlyShrink(t, RuleTalkLine, "talk_line_baseline.txt", "a walk's talk line is empty, a stand-in, or spoken by an id that is no character")
}

func TestSpeakersHavePortraits(t *testing.T) {
	gapsOnlyShrink(t, RulePortrait, "portrait_baseline.txt", "a named character speaks without a portrait of their own; draw one or set Look.NoPortrait")
}

func TestLineRulesCatchHanjaHyungAndStandIns(t *testing.T) {
	d := rulesData()
	d.Characters["mifang"] = content.Character{Name: "미방"}
	walk := func(p *content.ExplorePerson) *content.Explore {
		return &content.Explore{Hero: "liubei", Areas: map[string]*content.ExploreArea{"street": {Name: "거리", People: map[string]*content.ExplorePerson{"사람": p}}}}
	}
	rules := func(p *content.ExplorePerson) []string { return issueRules(exploreLineIssues(d, Env{}, "x", walk(p))) }
	say := func(who, text string) *content.ExplorePerson {
		return &content.ExplorePerson{Talk: []content.Line{{who, text}}}
	}

	assert.Empty(t, rules(say("zhangfei", "형, 채비는 다 됐슈.")), "장비 calls him 형")
	assert.Empty(t, rules(say("guanyu", "형님, 다녀오십시오.")), "관우 calls him 형님")
	assert.Empty(t, rules(say("관우", "형님, 다녀오십시오.")), "a speaker by shown name")
	assert.Equal(t, []string{"warn hyung"}, rules(say("mifang", "형님, 쌀이 모자랍니다.")))
	assert.Empty(t, rules(say("mifang", "형님은 그냥 나눠 주라 하시지만……")), "his own brother, not a call")
	assert.Equal(t, []string{"warn hanja"}, rules(say("", "曹操가 왔다")))
	assert.Equal(t, []string{"warn hanja"}, rules(&content.ExplorePerson{Chatter: "攻!"}))
	assert.Equal(t, []string{"warn talk-line"}, rules(say("", "  ")))
	assert.Equal(t, []string{"warn talk-line"}, rules(say("", "TODO 대사")))
	assert.Equal(t, []string{"warn talk-line"}, rules(say("nobody", "안녕하시오")))
	assert.Empty(t, rules(say("행인", "안녕하시오")), "a shown name that is no character is a label")

	d.Campaign.SwornBrothers = nil
	assert.Empty(t, rules(say("mifang", "형님, 쌀이 모자랍니다.")), "a scenario without sworn brothers is not checked")
}

func TestPortraitRuleSkipsTemplatesAndNoPortrait(t *testing.T) {
	d := rulesData()
	d.Characters["soldier"] = content.Character{Name: "병사", Template: true}
	d.Characters["lady"] = content.Character{Name: "부인", Look: content.Look{NoPortrait: true}}
	d.Characters["scout"] = content.Character{Name: "척후"}
	d.Characters["yuan"] = content.Character{Name: "원소", Look: content.Look{Portrait: "yuanshao"}}
	env := Env{Portrait: func(name string) bool { return name == "guanyu" || name == "yuanshao" }}
	for id, want := range map[string]int{"guanyu": 0, "soldier": 0, "lady": 0, "scout": 1, "yuan": 0, "nobody": 0} {
		assert.Len(t, portraitIssues(d, env, id), want, id)
	}
	assert.Empty(t, portraitIssues(d, Env{}, "scout"), "no portrait check without the portraits folder")
	run := Run{Nodes: []content.Node{{ID: "a", Kind: "say", Speaker: "guanyu·scout", Text: "張飛"}, {ID: "b", Kind: "narrate", Text: "밤이 깊었다"}}}
	assert.ElementsMatch(t, []string{"warn hanja", "warn portrait"}, issueRules(nodeLineIssues(d, env, "event:a", run)))
}

func rulesData() *content.Data {
	return &content.Data{
		Campaign: content.Campaign{CoreAllies: []string{"guanyu", "zhangfei"}, SwornBrothers: []string{"guanyu", "zhangfei"}},
		Characters: map[string]content.Character{
			"liubei":   {Name: "유비"},
			"guanyu":   {Name: "관우"},
			"zhangfei": {Name: "장비"},
		},
	}
}

func issueRules(is []Issue) []string {
	var out []string
	for _, i := range is {
		out = append(out, i.Level+" "+i.Rule)
	}
	return out
}

func TestExitRuleWantsAWayBackUnlessOneWay(t *testing.T) {
	room := func(to string, oneWay bool) *content.Explore {
		x := &content.Explore{Areas: map[string]*content.ExploreArea{
			"street": {Name: "거리", Exits: []content.ExploreExit{{Name: "주점", To: "inn", OneWay: oneWay}}},
			"inn":    {Name: "주점", Exits: []content.ExploreExit{{Name: "거리로", To: to}}},
			"market": {Name: "시장"},
		}}
		if to == "market" { // the inn leads on to the market, which leads back: only the street's way to the inn is one way
			x.Areas["market"].Exits = []content.ExploreExit{{Name: "주점", To: "inn"}}
		}
		return x
	}
	d := rulesData()
	assert.Empty(t, ExploreIssues(d, Env{}, "x", room("street", false)))
	assert.Equal(t, []string{"error exit-return"}, issueRules(ExploreIssues(d, Env{}, "x", room("market", false))))
	assert.Empty(t, ExploreIssues(d, Env{}, "x", room("market", true)), "a road meant to be one way is allowed")
}

func TestPeopleRulesWantSomethingToSay(t *testing.T) {
	x := &content.Explore{Areas: map[string]*content.ExploreArea{"street": {Name: "거리", People: map[string]*content.ExplorePerson{
		"행인":   {},                                   // nothing to say
		"장꾼#1": {},                                   // a numbered extra may be silent
		"관우":   {Chatter: "흠"},                       // named, only mutters
		"장비":   {Talk: []content.Line{{"장비", "어이"}}}, // fine
		"노인":   {Chatter: "허허"},                      // fine: a mutter is something
	}}}}
	got := issueRules(ExploreIssues(rulesData(), Env{}, "x", x))
	assert.ElementsMatch(t, []string{"warn silent-npc", "warn named-mute"}, got)
}

func TestOnePersonAsksForTheSortie(t *testing.T) {
	sortie := &content.ExploreSortie{Ask: "출진?"}
	x := &content.Explore{Areas: map[string]*content.ExploreArea{"camp": {Name: "진영", People: map[string]*content.ExplorePerson{
		"관우":       {Sortie: sortie},
		"관우@night": {Sortie: sortie}, // the same person under another flag
		"장비":       {Sortie: sortie},
	}}}}
	assert.Equal(t, []string{"warn sortie-count"}, issueRules(ExploreIssues(rulesData(), Env{}, "x", x)))
	delete(x.Areas["camp"].People, "장비")
	assert.Empty(t, ExploreIssues(rulesData(), Env{}, "x", x))
}

func TestWhereaboutsRuleAccountsForCoreAllies(t *testing.T) {
	d := rulesData()
	walk := func(people map[string]*content.ExplorePerson, wh map[string]content.Whereabout) *content.Explore {
		return &content.Explore{Hero: "liubei", Whereabouts: wh,
			Areas: map[string]*content.ExploreArea{"street": {Name: "거리", People: people}}}
	}
	talk := &content.ExplorePerson{Talk: []content.Line{{"", "…"}}}
	party := []string{"liubei", "guanyu", "zhangfei"}

	t.Run("an ally in the party who stands nowhere needs a record", func(t *testing.T) {
		got := WhereaboutsIssues(d, "w", walk(map[string]*content.ExplorePerson{"관우": talk}, nil), party)
		require.Len(t, got, 1)
		assert.Equal(t, "zhangfei", got[0].Subject)
		assert.Equal(t, "warn", got[0].Level)
	})
	t.Run("a record accounts for the absence", func(t *testing.T) {
		wh := map[string]content.Whereabout{"zhangfei": {Where: "elsewhere", Place: "서주", Reason: "성을 지키는 중"}}
		assert.Empty(t, WhereaboutsIssues(d, "w", walk(map[string]*content.ExplorePerson{"관우": talk}, wh), party))
	})
	t.Run("someone not yet in the party needs nothing", func(t *testing.T) {
		assert.Empty(t, WhereaboutsIssues(d, "w", walk(map[string]*content.ExplorePerson{"관우": talk}, nil), []string{"liubei", "guanyu"}))
	})
	t.Run("a record that contradicts the walk is an error", func(t *testing.T) {
		wh := map[string]content.Whereabout{"zhangfei": {Where: "absent", Reason: "없음"}}
		got := WhereaboutsIssues(d, "w", walk(map[string]*content.ExplorePerson{"관우": talk, "장비": talk}, wh), party)
		assert.Equal(t, []string{"error whereabouts"}, issueRules(got))
	})
	t.Run("a record needs a reason, and a place when elsewhere", func(t *testing.T) {
		for _, w := range []content.Whereabout{{Where: "absent"}, {Where: "elsewhere", Reason: "r"}, {Where: "gone", Reason: "r"}} {
			got := WhereaboutsIssues(d, "w", walk(map[string]*content.ExplorePerson{"관우": talk}, map[string]content.Whereabout{"zhangfei": w}), party)
			assert.Equal(t, []string{"error whereabouts"}, issueRules(got), "%+v", w)
		}
	})
	t.Run("a record for someone not in the party yet is stale", func(t *testing.T) {
		wh := map[string]content.Whereabout{"zhangfei": {Where: "absent", Reason: "아직 만나지 않음"}}
		got := WhereaboutsIssues(d, "w", walk(map[string]*content.ExplorePerson{"관우": talk}, wh), []string{"liubei", "guanyu"})
		assert.Equal(t, []string{"warn whereabouts"}, issueRules(got))
	})
}

func roomEnv(reach, cells int) Env {
	return Env{Map: func(string) *scenemap.Map { return &scenemap.Map{Size: [2]int{cells, 1}} }, Reach: func(string, [2]int) int { return reach }}
}

func TestPopulationRuleHoldsASettlementToItsDensity(t *testing.T) {
	people := func(n int) map[string]*content.ExplorePerson {
		m := map[string]*content.ExplorePerson{}
		for i := 0; i < n; i++ {
			m[fmt.Sprintf("백성#%d", i)] = &content.ExplorePerson{Talk: []content.Line{{"", "…"}}}
		}
		return m
	}
	// 1000 walkable tiles: the normal band (0.5 to 1.2 for every 100) asks 5 to 12 people
	issues := func(density, note string, n int) []string {
		return issueRules(populationIssues(roomEnv(1000, 400), "w", &content.ExploreArea{Name: "성", Density: density, PopulationNote: note, People: people(n)}))
	}
	assert.Empty(t, issues("", "", 5))
	assert.Empty(t, issues("", "", 12))
	assert.Equal(t, []string{"warn population"}, issues("", "", 4), "too few, no reason")
	assert.Equal(t, []string{"warn population"}, issues("", "", 13), "too many, no reason")
	assert.Empty(t, issues("", "사막이라 사람이 드물다", 2), "a reason accounts for it")
	assert.Equal(t, []string{"warn population"}, issues("", "이유", 8), "a reason where none is needed")
	assert.Empty(t, issues("sparse", "", 3), "a sparse place asks 2 to 5")
	assert.Equal(t, []string{"error population"}, issues("crowded", "", 8))
	assert.Empty(t, populationIssues(roomEnv(40, 42), "w", &content.ExploreArea{Name: "방", People: people(1)}), "a room is not held to a density")
	assert.Equal(t, []string{"warn talk-ratio"}, issueRules(populationIssues(roomEnv(1000, 400), "w", &content.ExploreArea{Name: "성", People: map[string]*content.ExplorePerson{"a#1": {}, "a#2": {}, "a#3": {}, "a#4": {}, "a#5": {}}})), "a crowd nobody can talk to")
	assert.Empty(t, populationIssues(Env{}, "w", &content.ExploreArea{Name: "성"}), "no map, nothing to say")
}

func TestTalkShareRuleCountsWhoCanBeTalkedTo(t *testing.T) {
	area := func(note string, talking, mute int) *content.ExploreArea {
		a := &content.ExploreArea{Name: "성", PopulationNote: note, People: map[string]*content.ExplorePerson{}}
		for i := 0; i < talking; i++ {
			a.People[fmt.Sprintf("말하는 이#%d", i)] = &content.ExplorePerson{Talk: []content.Line{{"", "…"}}}
		}
		for i := 0; i < mute; i++ {
			a.People[fmt.Sprintf("백성#%d", i)] = &content.ExplorePerson{Chatter: "음"}
		}
		return a
	}
	// 1000 tiles: 5 to 12 people fit; 10 people need 6 to talk (0.6)
	run := func(a *content.ExploreArea) []string { return issueRules(populationIssues(roomEnv(1000, 400), "w", a)) }
	assert.Empty(t, run(area("", 6, 4)))
	assert.Equal(t, []string{"warn talk-ratio"}, run(area("", 5, 5)), "half is too few")
	assert.Empty(t, run(area("항구라 구경꾼이 많다", 5, 5)), "a reason accounts for it")
	assert.Empty(t, populationIssues(roomEnv(500, 400), "w", area("", 1, 2)), "three people are not asked for a share")
	assert.Equal(t, []string{"warn population"}, run(area("이유", 6, 4)), "a reason where none is needed")
}

func shopData() *content.Data {
	d := rulesData()
	d.Shops = map[string]content.Shop{"iron": {Name: "대장간", Mark: "red"}, "plain": {Name: "대장간"}}
	return d
}

func TestAShopWithNewStockIsAnnouncedNearItsDoor(t *testing.T) {
	walk := func(shop string, crier *content.ExplorePerson) *content.Explore {
		street := &content.ExploreArea{Name: "거리", Exits: []content.ExploreExit{{Name: "대장간", At: [2]int{10, 10}, To: "smithy"}}, People: map[string]*content.ExplorePerson{}}
		if crier != nil {
			street.People["외치는 사람"] = crier
		}
		return &content.Explore{Areas: map[string]*content.ExploreArea{"street": street,
			"smithy": {Name: "대장간", Exits: []content.ExploreExit{{Name: "거리로", To: "street"}}, People: map[string]*content.ExplorePerson{"주인": {Shop: shop}}}}}
	}
	near := &content.ExplorePerson{At: [2]int{12, 9}, Notice: "shop:iron", Chatter: "새 쇠가 들어왔소!"}
	far := &content.ExplorePerson{At: [2]int{40, 9}, Notice: "shop:iron", Chatter: "새 쇠가 들어왔소!"}
	mute := &content.ExplorePerson{At: [2]int{12, 9}, Notice: "shop:iron"}
	wrong := &content.ExplorePerson{At: [2]int{12, 9}, Notice: "shop:other", Chatter: "허허"}
	run := func(x *content.Explore) []Issue { return shopIssues(shopData(), "w", x.Areas["street"], x) }
	assert.Empty(t, run(walk("iron", near)))
	assert.Len(t, run(walk("iron", nil)), 1, "nobody tells")
	assert.Len(t, run(walk("iron", far)), 1, "too far from the door")
	assert.Len(t, run(walk("iron", mute)), 1, "a notice with no bubble")
	assert.Len(t, run(walk("iron", wrong)), 1, "telling of something else")
	assert.Empty(t, run(walk("plain", nil)), "a shop with no new stock needs no crier")
}

func newsData() (*content.Data, *content.Explore) {
	d := rulesData()
	d.Chapters = []content.Chapter{{ID: "01", Nodes: []string{"a-walk"}}, {ID: "02", Nodes: []string{"b-walk"}}, {ID: "03", Nodes: []string{"c-walk"}}}
	d.Nodes = []content.Node{{ID: "a-walk", Kind: "explore", Explore: "xa"}, {ID: "b-walk", Kind: "explore", Explore: "xb"}, {ID: "c-walk", Kind: "explore", Explore: "xc"}}
	d.Campaign.News = []content.NewsItem{{ID: "war", Text: "전쟁이 났다", From: "02", Until: "02"}}
	return d, &content.Explore{Areas: map[string]*content.ExploreArea{"town": {Name: "성", Map: "m", People: map[string]*content.ExplorePerson{}}}}
}

func TestTheNewsOfTheTimeNeedsSomeoneToTellItInTheWalk(t *testing.T) {
	big := Env{Map: func(string) *scenemap.Map { return &scenemap.Map{Size: [2]int{20, 20}} }}
	d, x := newsData()
	assert.Empty(t, newsIssues(d, big, "xa", x), "before its time")
	assert.Empty(t, newsIssues(d, big, "xc", x), "after its time")
	assert.Len(t, newsIssues(d, big, "xb", x), 1, "in its time and nobody tells")
	x.Areas["town"].People["소문꾼"] = &content.ExplorePerson{Notice: "news:war", Chatter: "전쟁이 났대!"}
	assert.Empty(t, newsIssues(d, big, "xb", x))
	x.Areas["town"].People["소문꾼"].Chatter = ""
	assert.Len(t, newsIssues(d, big, "xb", x), 1, "a notice with no bubble")
	small := Env{Map: func(string) *scenemap.Map { return &scenemap.Map{Size: [2]int{5, 5}} }}
	assert.Empty(t, newsIssues(d, small, "xb", x), "a walk with no settlement has nobody to tell it to")
	d.Campaign.News[0].Until = "99"
	assert.Equal(t, []string{"error news"}, issueRules(newsIssues(d, big, "xb", x)))
}

func TestResidencesFollowWhoLivesThere(t *testing.T) {
	rooms := func(cells int) Env {
		return Env{Map: func(string) *scenemap.Map { return &scenemap.Map{Size: [2]int{cells, 1}} }}
	}
	build := func(lord string, guest, court bool, exits ...content.ExploreExit) (*content.ExploreArea, *content.Explore) {
		a := &content.ExploreArea{Name: "성", Lord: lord, Guest: guest, Court: court, Exits: exits}
		return a, &content.Explore{Areas: map[string]*content.ExploreArea{"town": a, "house": {Name: "집", Map: "h"}}}
	}
	run := func(env Env, a *content.ExploreArea, x *content.Explore) []string {
		return issueRules(residenceIssues(env, "w", a, x))
	}

	a, x := build("liubiao", false, false)
	assert.Equal(t, []string{"warn residence"}, run(rooms(100), a, x), "a lord without a large home")
	a, x = build("liubiao", false, false, content.ExploreExit{Name: "저택", To: "house", Residence: "large"})
	assert.Empty(t, run(rooms(100), a, x))
	assert.Equal(t, []string{"warn residence"}, run(rooms(48), a, x), "a large home on a small interior")
	a, x = build("", true, false)
	assert.Equal(t, []string{"warn residence"}, run(rooms(48), a, x), "a guest without a small home")
	a, x = build("", true, false, content.ExploreExit{Name: "거처", To: "house", Residence: "small"})
	assert.Empty(t, run(rooms(48), a, x))
	a, x = build("", false, true, content.ExploreExit{Name: "궁궐", Node: "event", Residence: "palace"})
	assert.Equal(t, []string{"warn residence"}, run(rooms(200), a, x), "a palace that is only an event")
	a, x = build("", false, true, content.ExploreExit{Name: "궁궐", To: "house", Residence: "palace"})
	assert.Empty(t, run(rooms(200), a, x))
	assert.Equal(t, []string{"warn residence"}, run(rooms(100), a, x), "a palace on an interior too small")
	a, x = build("", false, false, content.ExploreExit{Name: "집", To: "house", Residence: "mansion"})
	assert.Equal(t, []string{"error residence"}, run(rooms(100), a, x), "a size that does not exist")
}

func TestAPersonUnderSeveralFlagsCountsOnce(t *testing.T) {
	a := &content.ExploreArea{People: map[string]*content.ExplorePerson{"백성@pre": {}, "백성@post": {}, "상인#1": {}, "상인#2": {}, "관우": {}}}
	assert.Equal(t, 4, peopleCount(a))
}

// nullPaths lists where decoded JSON holds a null; the admin page maps over every list it gets, so a nil slice
// serialised as null blanks the whole page.
func nullPaths(x any, path string, out *[]string) {
	switch v := x.(type) {
	case nil:
		*out = append(*out, path)
	case map[string]any:
		for k, e := range v {
			nullPaths(e, path+"/"+k, out)
		}
	case []any:
		for _, e := range v {
			nullPaths(e, path+"[]", out)
		}
	}
}

func jsonNulls(t *testing.T, v any) []string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var x any
	require.NoError(t, json.Unmarshal(b, &x))
	var out []string
	nullPaths(x, "", &out)
	return out
}

// TestAdminJSONHasNoNulls builds the graph of every chapter, one and three battles deep, and the detail of every
// vertex in it, and requires that none of their JSON carries a null.
func TestAdminJSONHasNoNulls(t *testing.T) {
	d := load(t)
	e := env()
	for _, c := range Outline(d) {
		for _, battles := range []int{1, 3} {
			g := Build(d, e, c.Start, Options{Battles: battles, Assume: AssumeBefore(d, c.Start)})
			where := fmt.Sprintf("%s (%d battles)", c.ID, battles)
			assert.Empty(t, jsonNulls(t, g), "graph of "+where)
			for _, v := range g.Vertices {
				det, err := Detail(d, e, v.ID)
				if !assert.NoError(t, err, where+" "+v.ID) {
					continue
				}
				assert.Empty(t, jsonNulls(t, det), "detail of "+v.ID+" in "+where)
			}
		}
	}
}

package scenegraph

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"

	"srpg/internal/content"
	"srpg/internal/scenemap"
)

// The scenario rules a walk is held to. Each breaks as an Issue carrying its Rule; the scenario test fails on every
// error, and the warnings are the list still to be worked through.
const (
	RuleExitReturn  = "exit-return"  // an exit into a room has a way back to where it was entered from (error), unless OneWay
	RuleSilent      = "silent-npc"   // a person (not a numbered extra) has something to say: a talk, an event, a shop, the sortie, or a mutter
	RuleMute        = "named-mute"   // a person who is a named character can be talked to
	RuleSortie      = "sortie-count" // one person asks for the sortie
	RuleWhereabouts = "whereabouts"  // a core ally in the party stands in the walk, or the walk says where they are instead
	RulePopulation  = "population"   // the people in a settlement fit the ground the hero can walk, or PopulationNote says why not
	RuleTalkRatio   = "talk-ratio"   // about two in three people of a settlement can be talked to
	RuleShopNotice  = "shop-notice"  // a shop with new stock has someone near its door telling so
	RuleNews        = "news"         // each piece of the world's news of the time is told by someone in the walk
	RuleResidence   = "residence"    // the lord, the guest and the emperor each have a home of the right size
	RuleHanja       = "hanja"        // no Chinese characters in what is said or shown (the Korean edition is Korean only)
	RuleHyung       = "hyung"        // only the hero's sworn brothers (Campaign.SwornBrothers) call him "형" or "형님"
	RuleTalkLine    = "talk-line"    // a walk's talk line has text, and a speaker written as an id is a character
	RulePortrait    = "portrait"     // a named character who speaks has a portrait of their own, or says NoPortrait
)

// Density bands are how many people stand for every 100 small tiles the hero can walk in a settlement.
var Bands = map[string][2]float64{
	"sparse": {0.2, 0.5},
	"normal": {0.5, 1.2},
	"dense":  {1.2, 2.5},
}

// TalkShare is the least share of a settlement's people who have something to say when talked to (a talk, an event,
// a shop, the sortie): the rest are the crowd.
const TalkShare = 0.6

// SettlementCells is how many game cells a map has at least when it shows a settlement, not a room.
const SettlementCells = 100

// Residence sizes, by the game cells an interior map has at least.
var residenceCells = map[string]int{"small": 0, "large": 90, "palace": 160}

// charOf is the character a shown name belongs to: the character with that id, else the one with that name.
func charOf(d *content.Data, name string) string {
	if _, ok := d.Characters[name]; ok {
		return name
	}
	char := ""
	for id, c := range d.Characters {
		if !c.Template && c.Name == name && (char == "" || id < char) {
			char = id
		}
	}
	return char
}

func speaks(p *content.ExplorePerson) bool {
	return len(p.Talk) > 0 || len(p.Event) > 0 || p.Sortie != nil || p.Shop != ""
}

func peopleOf(x *content.Explore) []string {
	var keys []string
	for name, a := range x.Areas {
		for k := range a.People {
			keys = append(keys, name+"\x00"+k)
		}
	}
	sort.Strings(keys)
	return keys
}

// ExploreIssues checks one explore file against the rules that need no story state: exits, people, the sortie.
func ExploreIssues(d *content.Data, env Env, file string, x *content.Explore) []Issue {
	var out []Issue
	add := func(level, rule, where, msg string) {
		out = append(out, Issue{Level: level, Where: where, Msg: msg, Rule: rule})
	}
	areas := make([]string, 0, len(x.Areas))
	for name := range x.Areas {
		areas = append(areas, name)
	}
	sort.Strings(areas)
	sortieBy := map[string][]string{}
	for _, name := range areas {
		a := x.Areas[name]
		where := AreaID(file, name)
		for _, e := range a.Exits {
			t := x.Areas[e.To]
			if e.To == "" || t == nil || e.OneWay || slices.ContainsFunc(t.Exits, func(o content.ExploreExit) bool { return o.To == name }) {
				continue
			}
			back := "(나가는 출구가 없음)"
			if len(t.Exits) > 0 {
				switch o := x.Areas[t.Exits[0].To]; {
				case t.Exits[0].To == "":
					back = "이벤트 " + t.Exits[0].Node
				case o != nil:
					back = o.Name
				default:
					back = t.Exits[0].To
				}
			}
			add("error", RuleExitReturn, where, fmt.Sprintf("%s 의 출구 %q 로 들어가면 %s 에서 %s 로 돌아오는 출구가 없습니다 (나오면 %s). 일부러 돌아올 수 없는 길이면 출구에 OneWay 를 적습니다", a.Name, e.Name, t.Name, a.Name, back))
		}
		out = append(out, populationIssues(env, where, a)...)
		out = append(out, shopIssues(d, file, where, a, x)...)
		out = append(out, residenceIssues(env, where, a, x)...)
		keys := make([]string, 0, len(a.People))
		for k := range a.People {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := a.People[k]
			who := content.PersonName(k)
			if p.Sortie != nil {
				sortieBy[who] = append(sortieBy[who], name)
			}
			extra := strings.Contains(k, "#")
			switch {
			case !speaks(p) && p.Chatter == "" && !extra:
				add("warn", RuleSilent, where, fmt.Sprintf("%s 은(는) 말을 걸어도 할 말이 없고 혼잣말도 없습니다", k))
			case !speaks(p) && charOf(d, who) != "" && !extra:
				add("warn", RuleMute, where, fmt.Sprintf("%s 은(는) 이름 있는 인물인데 말을 걸어도 할 말이 없습니다 (혼잣말만 있음)", k))
			}
		}
	}
	out = append(out, newsIssues(d, env, file, x)...)
	out = append(out, exploreLineIssues(d, env, file, x)...)
	if len(sortieBy) > 1 {
		names := make([]string, 0, len(sortieBy))
		for n := range sortieBy {
			names = append(names, n)
		}
		sort.Strings(names)
		add("warn", RuleSortie, "", fmt.Sprintf("출진을 묻는 사람이 %d명입니다 (%s)", len(names), strings.Join(names, ", ")))
	}
	return out
}

// WhereaboutsIssues checks a walk's core allies against who is in the party as it begins. An ally in the party must
// stand somewhere in the walk, or the walk's Whereabouts must say where they are; a record must match the walk.
func WhereaboutsIssues(d *content.Data, where string, x *content.Explore, roster []string) []Issue {
	var out []Issue
	add := func(level, subject, msg string) {
		out = append(out, Issue{Level: level, Where: where, Msg: msg, Rule: RuleWhereabouts, Subject: subject})
	}
	name := func(id string) string {
		if c, ok := d.Characters[id]; ok && c.Name != "" {
			return c.Name
		}
		return id
	}
	present := map[string]bool{}
	for _, key := range peopleOf(x) {
		area, k, _ := strings.Cut(key, "\x00")
		if c := charOf(d, content.PersonName(k)); c != "" {
			present[c] = true
		}
		_ = area
	}
	for _, id := range d.Campaign.CoreAllies {
		if id == x.Hero || !slices.Contains(roster, id) || present[id] {
			continue
		}
		if _, ok := x.Whereabouts[id]; !ok {
			add("warn", id, fmt.Sprintf("%s 은(는) 부대에 있는데 이 탐험에 나오지 않고, 어디에 있는지 기록도 없습니다 (Whereabouts)", name(id)))
		}
	}
	ids := make([]string, 0, len(x.Whereabouts))
	for id := range x.Whereabouts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		w := x.Whereabouts[id]
		switch {
		case d.Characters[id].Name == "":
			add("error", id, fmt.Sprintf("Whereabouts 의 %q 는 없는 인물입니다", id))
		case w.Where != "elsewhere" && w.Where != "absent" || w.Reason == "" || w.Where == "elsewhere" && w.Place == "":
			add("error", id, fmt.Sprintf("%s 의 Whereabouts 는 Where(elsewhere 또는 absent)와 Reason 이 있어야 하고, elsewhere 는 Place 도 있어야 합니다", name(id)))
		case present[id]:
			add("error", id, fmt.Sprintf("%s 은(는) Whereabouts 에 이곳에 없다고 적혀 있는데 실제로 서 있습니다", name(id)))
		case !slices.Contains(roster, id):
			add("warn", id, fmt.Sprintf("%s 은(는) 이 시점에 부대에 없어서 Whereabouts 기록이 필요 없습니다", name(id)))
		}
	}
	return out
}

// peopleCount is how many people stand in an area: a person under several flags (a name with @) counts once, a
// numbered extra (a name with #) each.
func peopleCount(a *content.ExploreArea) int {
	seen := map[string]bool{}
	for k := range a.People {
		if i := strings.Index(k, "@"); i >= 0 && !strings.Contains(k, "#") {
			k = k[:i]
		}
		seen[k] = true
	}
	return len(seen)
}

// talkers is how many of the area's people (counted as peopleCount does) can be talked to.
func talkers(a *content.ExploreArea) int {
	talk := map[string]bool{}
	for k, p := range a.People {
		if i := strings.Index(k, "@"); i >= 0 && !strings.Contains(k, "#") {
			k = k[:i]
		}
		talk[k] = talk[k] || speaks(p)
	}
	n := 0
	for _, t := range talk {
		if t {
			n++
		}
	}
	return n
}

// reachOf is how many small tiles the hero can walk in an area, from where he arrives; 0 when it cannot be told.
func reachOf(env Env, a *content.ExploreArea) (reach, cells int) {
	if env.Map == nil {
		return 0, 0
	}
	m := env.Map(a.Map)
	if m == nil {
		return 0, 0
	}
	cells = m.Size[0] * m.Size[1]
	if env.Reach != nil {
		return env.Reach(a.Map, a.At), cells
	}
	if env.Catalog == nil {
		return 0, cells
	}
	w := scenemap.Walkable(m, env.Catalog)
	w.Within(a.At[0], a.At[1])
	for v := m.Origin[1]; v < m.Origin[1]+m.H(); v++ {
		for u := m.Origin[0]; u < m.Origin[0]+m.W(); u++ {
			if w.Open(u, v) {
				reach++
			}
		}
	}
	return reach, cells
}

// populationIssues holds a settlement-sized area to its density: people for every 100 small tiles the hero can walk
// (the rest of a big map is scenery he cannot go to). A density outside the band needs a reason in PopulationNote.
func populationIssues(env Env, where string, a *content.ExploreArea) []Issue {
	reach, cells := reachOf(env, a)
	if cells < SettlementCells || reach == 0 {
		return nil
	}
	add := func(level, msg string) []Issue {
		return []Issue{{Level: level, Where: where, Msg: msg, Rule: RulePopulation, Subject: RulePopulation}}
	}
	name := a.Density
	if name == "" {
		name = "normal"
	}
	band, ok := Bands[name]
	if !ok {
		return add("error", fmt.Sprintf("%s 의 밀도 %q 는 없는 값입니다 (sparse, normal, dense)", a.Name, a.Density))
	}
	n := peopleCount(a)
	lo, hi := int(math.Ceil(float64(reach)*band[0]/100)), int(math.Floor(float64(reach)*band[1]/100))
	var out []Issue
	outOfBand := n < lo || n > hi
	quiet := n >= 4 && float64(talkers(a)) < TalkShare*float64(n)
	if outOfBand && a.PopulationNote == "" {
		out = append(out, add("warn", fmt.Sprintf("%s: 걸을 수 있는 칸 %d, 사람 %d명 (100칸당 %.1f명). %s 밀도(%.1f~%.1f)는 %d~%d명입니다. 사람을 맞추거나, 이유를 PopulationNote 에 적거나, Density 를 sparse/dense 로 정합니다",
			a.Name, reach, n, 100*float64(n)/float64(reach), name, band[0], band[1], lo, hi))...)
	}
	if quiet && a.PopulationNote == "" {
		out = append(out, Issue{Level: "warn", Where: where, Rule: RuleTalkRatio, Subject: RuleTalkRatio,
			Msg: fmt.Sprintf("%s: 사람 %d명 중 말을 걸 수 있는 사람은 %d명(%.0f%%)뿐입니다. %.0f%% 이상이어야 합니다. 대화를 더하거나 이유를 PopulationNote 에 적습니다",
				a.Name, n, talkers(a), 100*float64(talkers(a))/float64(n), 100*TalkShare)})
	}
	if !outOfBand && !quiet && a.PopulationNote != "" {
		out = append(out, add("warn", fmt.Sprintf("%s 는 밀도와 대화 비율이 범위 안인데 PopulationNote 가 있습니다. 지우세요", a.Name))...)
	}
	return out
}

// noticeNear is the person who mutters (Chatter) and tells topic, standing within dist small tiles of at; nil if none.
func noticeNear(a *content.ExploreArea, topic string, at [2]int, dist int) *content.ExplorePerson {
	for _, k := range sortedPeople(a) {
		if p := a.People[k]; p.Notice == topic && p.Chatter != "" && max(abs(p.At[0]-at[0]), abs(p.At[1]-at[1])) <= dist {
			return p
		}
	}
	return nil
}

func sortedPeople(a *content.ExploreArea) []string {
	keys := make([]string, 0, len(a.People))
	for k := range a.People {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// shopNoticeDist is how close to a shop's door the person telling of its new stock stands, in small tiles.
const shopNoticeDist = 6

// unspaced is text without spaces, for finding an item's name in a line however the line breaks the name.
func unspaced(text string) string { return strings.ReplaceAll(text, " ", "") }

// missingNames are the names none of the texts mention.
func missingNames(names []string, texts ...string) []string {
	var out []string
	for _, n := range names {
		if !slices.ContainsFunc(texts, func(t string) bool { return strings.Contains(unspaced(t), unspaced(n)) }) {
			out = append(out, n)
		}
	}
	return out
}

// productNumberRe is a numeral the text states as a value: a digit, or a Sino-Korean numeral with a unit of
// money or share ("이천 냥", "삼십 퍼센트"). A comparison like "한 급 위", a measure like "한 줌" and words that contain
// a numeral syllable like "만만치 않소" are not.
var productNumberRe = regexp.MustCompile(`[0-9０-９]+|[일이삼사오육칠팔구십백천만]+\s*(?:냥|푼|퍼센트|%)`)

// productNumber is the first product value a shop advert states, or "".
func productNumber(text string) string { return productNumberRe.FindString(text) }

// shopIssues holds the telling of new stock to what the story says is new. Where a shop sells something no walk
// before it sold (content.Data.NewStock), a person outdoors near its door tells so in bubbles and, talked to, tells
// each new good by name, standing in front of the building and turned to the door or to the street. A person who
// tells of a shop's stock (Notice "shop:<id>") but not at its door, or of a shop with nothing new, is wrong: old
// goods are not cried again, and a crier does not stand inside the shop.
func shopIssues(d *content.Data, file, where string, a *content.ExploreArea, x *content.Explore) []Issue {
	var out []Issue
	warn := func(shop, msg string) {
		out = append(out, Issue{Level: "warn", Rule: RuleShopNotice, Where: where, Subject: shop, Msg: msg})
	}
	doors := map[string]bool{}
	for _, e := range a.Exits {
		room := x.Areas[e.To]
		if room == nil {
			continue
		}
		for _, k := range sortedPeople(room) {
			shop := room.People[k].Shop
			if shop == "" {
				continue
			}
			doors[shop] = true
			stock := d.NewStock(file, shop)
			if len(stock) == 0 {
				continue
			}
			var names []string
			for _, id := range stock {
				names = append(names, d.StockName(id))
			}
			crier := noticeNear(a, "shop:"+shop, e.At, shopNoticeDist)
			if crier == nil {
				warn(shop, fmt.Sprintf("%s (%s)에 새 물건(%s)이 들어왔는데, 입구 근처 %d칸 안에 이를 말풍선으로 알리는 사람(Notice %q, Chatter)이 없습니다", e.Name, d.Shops[shop].Name, strings.Join(names, ", "), shopNoticeDist, "shop:"+shop))
				continue
			}
			var talk []string
			for _, l := range crier.Talk {
				talk = append(talk, l[1])
			}
			if len(talk) == 0 {
				warn(shop, fmt.Sprintf("%s 앞에서 새 물건을 알리는 사람에게 말을 걸어도 할 말이 없습니다. 새 물건(%s)을 설명해야 합니다", e.Name, strings.Join(names, ", ")))
			}
			if miss := missingNames(names, strings.Join(talk, " ")); len(talk) > 0 && len(miss) > 0 {
				warn(shop, fmt.Sprintf("%s 앞에서 말을 걸었을 때 새 물건 %s 의 설명이 없습니다", e.Name, strings.Join(miss, ", ")))
			}
			for _, text := range append(append([]string{}, talk...), strings.Split(crier.Chatter, "|")...) {
				if num := productNumber(text); num != "" {
					warn(shop, fmt.Sprintf("%s 앞의 안내 %q 에 상품 수치 %q 이(가) 있습니다. 값·회복량·능력치 숫자는 데이터가 바뀌면 어긋나므로 적지 않습니다", e.Name, text, num))
				}
			}
			if miss := missingNames(names, strings.Split(crier.Chatter, "|")...); len(miss) > 0 {
				warn(shop, fmt.Sprintf("%s 앞의 말풍선에 새 물건 %s 이(가) 없습니다", e.Name, strings.Join(miss, ", ")))
			}
			front := e.At[1] - 2
			if e.Object != nil {
				front = e.Object[1]
			}
			toDoor := content.Toward(content.Point{X: crier.At[0], Y: crier.At[1]}, content.Point{X: e.At[0], Y: e.At[1]})
			switch {
			case crier.At == e.At || crier.At[1] < front:
				warn(shop, fmt.Sprintf("%s 앞의 알리는 사람(%v)이 문 자리이거나 건물 뒤에 서 있습니다. 건물 앞 길에 세웁니다", e.Name, crier.At))
			case crier.Face != toDoor && crier.Face != "SW":
				warn(shop, fmt.Sprintf("%s 앞의 알리는 사람이 %s 를 봅니다. 문(%s)이나 길(SW) 쪽을 봐야 합니다", e.Name, crier.Face, toDoor))
			}
		}
	}
	for _, k := range sortedPeople(a) {
		shop, ok := strings.CutPrefix(a.People[k].Notice, "shop:")
		switch {
		case !ok:
		case !doors[shop]:
			warn(shop, fmt.Sprintf("%s 은(는) 상점 %s 의 물건을 알리지만 그 가게 입구가 있는 바깥이 아닙니다. 가게 안이 아니라 입구 앞 길에 세웁니다", k, shop))
		case len(d.NewStock(file, shop)) == 0:
			warn(shop, fmt.Sprintf("%s 은(는) 상점 %s 의 물건을 알리는데, 이 탐험 앞에서 이미 팔던 물건뿐입니다. 이미 판 물건은 다시 알리지 않습니다", k, shop))
		}
	}
	return out
}

// chapterIndexOf is the position in the story of the chapter whose walk uses explore file; -1 when none does.
func chapterIndexOf(d *content.Data, file string) int {
	for i, c := range d.Chapters {
		for _, id := range c.Nodes {
			if n := d.Node(id); n.Kind == "explore" && n.Explore == file {
				return i
			}
		}
	}
	return -1
}

func chapterIndex(d *content.Data, id string) int {
	return slices.IndexFunc(d.Chapters, func(c content.Chapter) bool { return c.ID == id })
}

// newsIssues asks, for each piece of news of the walk's chapter, someone in the settlements of the walk to tell it.
func newsIssues(d *content.Data, env Env, file string, x *content.Explore) []Issue {
	at := chapterIndexOf(d, file)
	if at < 0 || len(d.Campaign.News) == 0 {
		return nil
	}
	var settled []*content.ExploreArea
	for _, name := range sortedAreas(x) {
		if _, cells := reachOf(env, x.Areas[name]); cells >= SettlementCells {
			settled = append(settled, x.Areas[name])
		}
	}
	if len(settled) == 0 {
		return nil
	}
	var out []Issue
	for _, n := range d.Campaign.News {
		from, until := chapterIndex(d, n.From), chapterIndex(d, n.Until)
		if from < 0 || until < 0 {
			out = append(out, Issue{Level: "error", Rule: RuleNews, Subject: n.ID, Msg: fmt.Sprintf("소식 %q 의 From/Until 장 번호가 없는 장입니다", n.ID)})
			continue
		}
		if at < from || at > until || slices.ContainsFunc(settled, func(a *content.ExploreArea) bool {
			return slices.ContainsFunc(mapValues(a.People), func(p *content.ExplorePerson) bool { return p.Notice == "news:"+n.ID && p.Chatter != "" })
		}) {
			continue
		}
		out = append(out, Issue{Level: "warn", Rule: RuleNews, Where: AreaID(file, x.Start.Area), Subject: n.ID,
			Msg: fmt.Sprintf("소식 「%s」(%s)을 말풍선으로 알리는 사람(Notice %q, Chatter)이 이 탐험의 마을이나 성에 없습니다", n.Text, n.ID, "news:"+n.ID)})
	}
	return out
}

func mapValues(m map[string]*content.ExplorePerson) []*content.ExplorePerson {
	out := make([]*content.ExplorePerson, 0, len(m))
	for _, p := range m {
		out = append(out, p)
	}
	return out
}

func sortedAreas(x *content.Explore) []string {
	names := make([]string, 0, len(x.Areas))
	for n := range x.Areas {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// residenceIssues asks the lord of a place for a large home, a guest for a small one, the court for a palace, and holds
// each home to the size of the interior it leads to.
func residenceIssues(env Env, where string, a *content.ExploreArea, x *content.Explore) []Issue {
	var out []Issue
	add := func(level, subject, msg string) {
		out = append(out, Issue{Level: level, Where: where, Msg: msg, Rule: RuleResidence, Subject: subject})
	}
	have := map[string]bool{}
	for _, e := range a.Exits {
		if e.Residence == "" {
			continue
		}
		min, ok := residenceCells[e.Residence]
		if !ok {
			add("error", e.Name, fmt.Sprintf("출구 %q 의 Residence %q 는 없는 값입니다 (small, large, palace)", e.Name, e.Residence))
			continue
		}
		have[e.Residence] = true
		room := x.Areas[e.To]
		switch {
		case e.Residence != "small" && room == nil:
			add("warn", e.Name, fmt.Sprintf("%s 은(는) %s 인데 들어갈 수 있는 실내가 없고 이벤트만 있습니다", e.Name, e.Residence))
		case room != nil && env.Map != nil:
			if m := env.Map(room.Map); m != nil && m.Size[0]*m.Size[1] < min {
				add("warn", e.Name, fmt.Sprintf("%s 은(는) %s 로 정했는데 실내 지도 %s 는 %d칸뿐입니다 (%s 는 %d칸 이상)", e.Name, e.Residence, room.Map, m.Size[0]*m.Size[1], e.Residence, min))
			}
		}
	}
	for _, need := range []struct {
		on         bool
		kind, what string
	}{{a.Lord != "", "large", "이곳을 다스리는 사람(Lord)의 큰 저택"}, {a.Guest, "small", "이곳에 의탁한 유비의 작은 저택"}, {a.Court, "palace", "황제가 있는 곳의 황궁"}} {
		if need.on && !have[need.kind] {
			add("warn", need.kind, fmt.Sprintf("%s 에 %s 으로 가는 출구(Residence %q)가 없습니다", a.Name, need.what, need.kind))
		}
	}
	return out
}

// hanja finds Chinese characters (CJK ideographs) in a line.
var hanja = regexp.MustCompile(`[\x{3400}-\x{4DBF}\x{4E00}-\x{9FFF}\x{F900}-\x{FAFF}]`)

// hyungCall is a line that opens by calling its listener 형 or 형님.
var hyungCall = regexp.MustCompile(`^\s*(형님|형)\s*[,!~…?.]`)

// placeholder is a line left as a stand-in for text still to write.
var placeholder = regexp.MustCompile(`(?i)TODO|FIXME|XXX|자리표시|대사 없음`)

// idLike is a speaker written as a character id rather than a shown name.
var idLike = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func lineIssue(rule, where, subject, msg string) Issue {
	return Issue{Level: "warn", Rule: rule, Where: where, Subject: subject, Msg: msg}
}

func heroName(d *content.Data, x *content.Explore) string {
	if c := d.Characters[x.Hero]; c.Name != "" {
		return c.Name
	}
	return x.Hero
}

// exploreLineIssues holds what the people of a walk say to the line rules: no hanja, 형 from the hero's sworn
// brothers only (a walk's talk is said to its hero), real text from real speakers.
func exploreLineIssues(d *content.Data, env Env, file string, x *content.Explore) []Issue {
	var out []Issue
	for _, name := range sortedAreas(x) {
		a := x.Areas[name]
		where := AreaID(file, name)
		keys := make([]string, 0, len(a.People))
		for k := range a.People {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := a.People[k]
			if hanja.MatchString(p.Chatter) {
				out = append(out, lineIssue(RuleHanja, where, k, fmt.Sprintf("%s 의 혼잣말에 한자가 있습니다: %q", k, p.Chatter)))
			}
			for i, l := range p.Talk {
				subject := fmt.Sprintf("%s#%d", k, i+1)
				who := l[0]
				if who == "" {
					who = content.PersonName(k)
				}
				char := charOf(d, who)
				switch {
				case strings.TrimSpace(l[1]) == "" || placeholder.MatchString(l[1]):
					out = append(out, lineIssue(RuleTalkLine, where, subject, fmt.Sprintf("%s 의 %d번째 대사가 비었거나 자리표시 글입니다: %q", k, i+1, l[1])))
				case idLike.MatchString(l[0]) && d.Characters[l[0]].Name == "":
					out = append(out, lineIssue(RuleTalkLine, where, subject, fmt.Sprintf("%s 의 %d번째 대사의 화자 %q 는 없는 인물입니다", k, i+1, l[0])))
				}
				if hanja.MatchString(l[1]) {
					out = append(out, lineIssue(RuleHanja, where, subject, fmt.Sprintf("%s 의 대사에 한자가 있습니다: %q", k, l[1])))
				}
				if len(d.Campaign.SwornBrothers) > 0 && !slices.Contains(d.Campaign.SwornBrothers, char) && hyungCall.MatchString(l[1]) {
					out = append(out, lineIssue(RuleHyung, where, subject, fmt.Sprintf("%s 이(가) %s 를 형이라 부릅니다: %q. 형·형님이라 부르는 사람은 의형제(SwornBrothers)뿐입니다", who, heroName(d, x), l[1])))
				}
				out = append(out, portraitIssues(d, env, char)...)
			}
		}
		for _, e := range a.Exits {
			for _, l := range e.Shut {
				if hanja.MatchString(l[1]) {
					out = append(out, lineIssue(RuleHanja, where, e.Name, fmt.Sprintf("출구 %s 의 막힘 대사에 한자가 있습니다: %q", e.Name, l[1])))
				}
			}
		}
	}
	return out
}

// nodeLineIssues holds the lines of an event run (say, narrate) to the line rules that need no listener.
func nodeLineIssues(d *content.Data, env Env, where string, run Run) []Issue {
	var out []Issue
	for _, n := range run.Nodes {
		if n.Kind != "say" && n.Kind != "narrate" {
			continue
		}
		if hanja.MatchString(n.Text) {
			out = append(out, lineIssue(RuleHanja, where, n.ID, fmt.Sprintf("%s 에 한자가 있습니다: %q", n.ID, n.Text)))
		}
		if n.Kind == "say" {
			for _, id := range strings.Split(n.Speaker, "·") {
				out = append(out, portraitIssues(d, env, id)...)
			}
		}
	}
	return out
}

// portraitIssues asks a named character who speaks for a portrait of their own (graphics/portraits), unless they
// stand for many (Template) or are shown without one on purpose (NoPortrait); without it the class's is shown.
func portraitIssues(d *content.Data, env Env, id string) []Issue {
	c, ok := d.Characters[id]
	if !ok || env.Portrait == nil || c.Template || c.Look.NoPortrait {
		return nil
	}
	name := c.Look.Portrait
	if name == "" {
		name = id
	}
	if env.Portrait(name) {
		return nil
	}
	return []Issue{lineIssue(RulePortrait, "character:"+id, RulePortrait, fmt.Sprintf("%s(%s)는 대사가 있는데 자기 초상(portraits/%s.png)이 없어 병종 초상으로 보입니다. 초상을 그리거나 Look.NoPortrait 를 적습니다", c.Name, id, name))}
}

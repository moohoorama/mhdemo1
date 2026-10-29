package content

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Curve [][2]float64

// Chance is a percentage that starts at Base for an even ratio and grows by Per for each extra multiple.
type Chance struct{ Base, Per, Max float64 }

// Rules are the balance numbers of the engine's formulas.
type Rules struct {
	HitCurve              Curve
	StatNumerator         float64 // an attribute a gives stat = k*(level+StatLevelBase) * (StatNumerator/(StatDenominator-a) + bonus)
	StatDenominator       float64
	StatLevelBase         int
	BasicAttack           Skill
	HitFloor, HitCeil     float64 // physical hit chance clamp
	SpellFloor, SpellCeil float64
	PhysicalLevelScale    float64 // physical damage = attack - defense + (level+StatLevelBase)*PhysicalLevelScale
	MagicLevelScale       float64 // spell damage = mind - target mind + (level+StatLevelBase)*MagicLevelScale
	ManaMaxFactor         float64 // max spell points = base + ManaMaxFactor*StatNumerator/(StatDenominator-command)
	ManaRecoveryFactor    float64 // spell points per turn = base + ManaRecoveryFactor*StatNumerator/(StatDenominator-intellect)
	Crit, Double          Chance
	CritMultiplier        float64
	CounterShare          float64 // a counter from a stance or trait deals this share of a normal blow
	PlainCounterShare     float64 // any other unit strikes back in its reach with this share of a normal blow
	DoubleShare           float64 // a second strike deals this share of a normal blow
	PoisonPercent         int
	BurnPercent           int
	BuffPercent           int     // the size of an attack/defense/morale/agility buff or debuff
	BuffTurns             int     // how long a trait's automatic buff lasts
	ExpGrowth             float64 // action XP is scaled by this per level the target is above the actor
	ExpAction, ExpSpell   float64 // action XP in percent of the actor's level bar at equal levels, and its spell multiplier
	ExpDefenseShare       float64 // received strike or successful defense XP, relative to action XP
	ExpDefeat             float64
	ExpLevelStep          float64
	ExpMinFactor          float64
	ExpMaxFactor          float64
	ExpSupportPercent     int
	PointsPerLevel        int
	MaxLevel              int
	DeputyShare           int   // percent of the main officer's experience a deputy also earns
	DeputyWeight          int   // deputy attribute blend: ratio = (own charm*DeputyWeight + deputy charm) / DeputyDivisor
	DeputyDivisor         int   // also the percent scale of charm
	PromoteLevels         []int // minimum level to promote from tier i
	StageLevelBase        int   // the n-th battle of the story is fought at StageLevelBase + StageLevelStep×(n−1), rounded; 0 leaves spawns at their characters' levels
	StageLevelStep        float64
	SideShareMax          float64            // the most damage share a blow deals to anyone it strikes besides its target (splash, sweep, pierce)
	RankLead              map[string]float64 // a rank's lead over the stage level, as a share of (stage level + StatLevelBase) so it means the same strength at any level
	EnemyStock            map[string]int
	AllyStock             map[string]int
	EdgeTile              string                    // the cell (ground, structure, state) that lies beyond the map's edge
	CritMarks             map[string]string         // the state a critical spell of the element leaves on each critically struck target's cell
	CritMarkTurns         int                       // how many rounds that state lasts
	ElementTerrain        map[string]map[string]int // percent more a spell of the element deals to a target on the structure or, without one listed, the ground letter
	AuraHeal              int
}
type Campaign struct {
	Start      string // first scenario node
	Roster     []string
	Deployment []string
	Inventory  map[string]int
	Money      int
	Loadout    map[string][]string `json:",omitempty"` // equipment each officer wears in a game started at a waypoint
}

// Chapter is one story file of the scenario, in play order. Nodes lists the ids of the story's flow; Events
// those of the explore events, which the story reaches only by talking to someone (each ends with a back node).
type Chapter struct {
	ID, File string
	Nodes    []string `json:",omitempty"`
	Events   []string `json:",omitempty"`
}

// ScenarioRef is one entry of assets/scenario/scenario.json.
type ScenarioRef struct{ ID, Name, Path string }

// ScenarioFile is a scenario's own file: where the campaign starts and the chapters in order.
type ScenarioFile struct {
	Campaign
	Chapters []Chapter
}
type Faction struct {
	Name   string
	Colors []string // light, middle, dark
}

type Class struct {
	Name, Desc                      string
	Family                          string `json:",omitempty"`
	Terrain, Attack                 string // movement group; melee or ranged
	Weapon, Armor                   string // equipment types it carries
	Move, Range, MinRange, HPFactor int    // MinRange: the nearest cell a ranged class can attack
	Bonus                           [6]float64
	MPBase, RecoveryBase            int
	Skills, Pool, Tags              []string
	Traits                          []string // held while in this class (not learned, lost on class change)
	Promotes                        []string
	Tier                            int
	Sprite                          string
	Scene                           string `json:",omitempty"` // non-combat sprite for scenes of officers in this class that have none of their own
	Shoot                           string // class whose look a unit takes when FxMountedRange lets it shoot from range
	Portrait                        string // stands in for officers without a portrait of their own
	pool                            []string
}
type Colors []string

func (c *Colors) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*c = Colors{one}
		return nil
	}
	return json.Unmarshal(b, (*[]string)(c))
}

// Look is how a character is drawn.
type Look struct {
	Sprite   string            // battle sprite for any class Sprites does not name; the class's when empty
	Sprites  map[string]string `json:",omitempty"` // class id, or class id:weapon type -> battle sprite while the character serves in that class (with that weapon)
	Scene    string            // non-combat sprite for scenes; Sprite when empty
	Portrait string
	Faction  string
	Palette  map[string]Colors // tag group -> one base colour or three shades
}
type Character struct {
	Name, Class, Biography   string
	Stats                    [6]int
	Level                    int
	Points                   int `json:",omitempty"` // trait points the character starts with
	Lord, Playable, Template bool
	Traits, Learn, Equipment []string
	Look                     Look
}
type Equipment struct {
	Name, Desc, Slot, Type string
	Icon                   string // the UI sheet's item_<Icon> picture
	Price                  int    `json:",omitempty"` // what a shop asks in money; it buys it back for half
	Bonus                  [6]float64
	Traits, Learn, Skills  []string
}
type Item struct {
	Name, Desc string
	Icon       string // the UI sheet's item_<Icon> picture
	Effect     string
	Amount     int
	Class      string // the class a promotion item makes
	Price      int    `json:",omitempty"` // what a shop asks in money; it buys it back for half
}
type Trait struct {
	Name, Desc string
	Kind       string // strategy (책략), tactic (전법) or talent (특기): the icon's shape
	Tone       string // attack, heal, buff, debuff, guard, move or craft: the icon's colour
	Cost       int    // learning cost; 0 = given only
	Requires   []string
	Effects    map[string]float64
}
type Skill struct {
	Name, Desc    string
	Kind, Mode    string
	Shape, Effect string
	Element       string
	Condition     string
	Item          string // the item a steal takes
	Requires      []string
	Cost, Hit     int
	Min, Max      int
	Duration      int
	Length        int // cells of a line
	Coeff         float64
	Weapon        string  // the weapon type that must be equipped
	Pierce        float64 // share of damage the enemy behind the target takes
	Strikes       int     // 2 makes every use strike twice
	Each          bool    // strikes every target with an attack of its own, one after another
	Reach         int     // cells the skill adds to the attacker's range
	MagicCoeff    float64 // a physical skill also deals spell damage at this coefficient
	Deprecated    bool
}

// Terrain is what a class pays and gets standing on a cell, after its layers are put together.
type Terrain struct {
	Factor float64
	Cost   int
}
type Point struct{ X, Y int }

// Toward is the facing (NE, SE, SW, NW) of someone at from looking at to: on screen map +x runs to
// the lower right (SE) and +y to the lower left (SW). The longer axis wins, the x axis on a tie
// (docs/story-writing.md).
func Toward(from, to Point) string {
	dx, dy := to.X-from.X, to.Y-from.Y
	switch {
	case max(dx, -dx) >= max(dy, -dy) && dx > 0:
		return "SE"
	case max(dx, -dx) >= max(dy, -dy) && dx < 0:
		return "NW"
	case dy > 0:
		return "SW"
	default:
		return "NE"
	}
}

type Spawn struct {
	AI                bool      `json:",omitempty"`
	Policy            string    `json:",omitempty"` // how the unit behaves: hold, react, assault, march, flee; every AI-driven unit has one
	Goal              *Point    `json:",omitempty"` // march and flee: the cell the unit makes for
	GoalArea          int       `json:",omitempty"` // march and flee instead of Goal: any cell of this stage area, the nearest one
	DuelFirst         bool      `json:",omitempty"` // the AI seeks a pending duel partner before anything else
	Class, Facing     string    `json:",omitempty"`
	Group             string    `json:",omitempty"` // name events use to address this spawn with others
	Side              string    `json:",omitempty"` // event spawns only: "ally" (an AI ally, the default) or "enemy"
	HP, MP            *int      `json:",omitempty"`
	Traits, Equipment *[]string `json:",omitempty"`
	Character         string
	X, Y              int
	Level             int     `json:",omitempty"` // overrides the character's level; only where Rules set no stage level
	Rank              string  `json:",omitempty"` // with a stage level: troop, officer, commander or boss; none is troop for a template, officer for a named one
	Shift             float64 `json:",omitempty"` // added to the rank's lead: -0.07 for a crowd one level under at the start
	Faction           string  `json:",omitempty"` // overrides the stage's and the character's faction color
}

// Beat is one step of a duel's staging: Kind names the motion, Side who acts (0 the ally, 1 the
// enemy, 2 and 3 the supports), Speech what is said and Duration how long it lasts in seconds.
// Title is a note for whoever writes the duel.
type Beat struct {
	Kind, Title, Speech string
	Side                int
	Duration            float64
}

// DuelVariant is the staging used when First is the officer who moved to meet the other.
type DuelVariant struct {
	First string
	Beats []Beat
}

// Duel is a pending fight between two officers: it starts when they stand next to each other,
// its result is fixed here and Beats (with Supports who join in) stage it.
type Duel struct {
	ID, Ally, Enemy, Outcome, AllyResult, EnemyResult, Text string
	Supports                                                []string      `json:",omitempty"`
	Beats                                                   []Beat        `json:",omitempty"`
	Variants                                                []DuelVariant `json:",omitempty"`
}
type Speech struct{ Actor, Text string }

// Area is a numbered rectangle of the stage that events and objectives point at by id; Name and
// Description are for people reading the data.
type Area struct {
	ID                int
	Rect              MapRect
	Name, Description string
}

// Action is one effect of a battle event. Kind picks the fields that matter: say (Lines: the
// speakers shown together), spawn (Spawns), remove (Targets leave the map), flag (Flags).
type Action struct {
	Kind     string
	Lines    []Speech       `json:",omitempty"`
	Spawns   []Spawn        `json:",omitempty"`
	Targets  []string       `json:",omitempty"`
	Flags    map[string]int `json:",omitempty"`
	Stat     string         `json:",omitempty"` // buff: attack, defense, mind, morale, agility (percent) or speed, range (cells)
	Value    int            `json:",omitempty"` // buff: how much, negative for a debuff
	Moves    []Move         `json:",omitempty"` // move: who goes where, all at once
	Steps    [][]Step       `json:",omitempty"` // cutscene: groups of steps; a group's steps run together, the next group waits for them
	Duel     string         `json:",omitempty"` // duel: the stage duel to hold now, wherever its officers stand
	Area     int            `json:",omitempty"` // terrain: the area whose cells change
	State    string         `json:",omitempty"` // terrain: the state they get; NoLayer clears it
	Turns    int            `json:",omitempty"` // terrain: for how many rounds; none is for good (the state then turns into its Then)
	Policy   string         `json:",omitempty"` // ai: the new policy of the Targets
	Goal     *Point         `json:",omitempty"` // ai: its goal cell, for march and flee
	GoalArea int            `json:",omitempty"` // ai: instead of Goal, any cell of this area
	Victory  []Goal         `json:",omitempty"` // goals: the win goals from now on; left out, they stay
	Defeat   []Goal         `json:",omitempty"` // goals: the loss goals from now on; left out, they stay ([] clears them)
}

// Move sends a unit (or each unit of a group, to cells around To) somewhere, ignoring move range and
// terrain cost, and turns it to Facing there (none: the way its last step went).
type Move struct {
	Unit   string
	To     Point
	Facing string `json:",omitempty"`
}

// Step is one thing that happens in a cutscene: Unit moves to To (a group fills the cells around it)
// and turns to Facing there (none: the way its last step went), plays Motion (attack, hit, block, exhausted), turns to Facing, says Text,
// leaves the field across the nearest edge as if fled, or the scene waits Seconds.
type Step struct {
	Kind, Unit, Motion, Facing, Text string
	To                               *Point  `json:",omitempty"`
	Seconds                          float64 `json:",omitempty"`
}

// BuffStats are the stats a buff can change; speed and range count cells, the rest percent.
var BuffStats = []string{"attack", "defense", "mind", "morale", "agility", "speed", "range"}

// Policies are the behaviours an AI-driven unit can be given: hold stands its ground and strikes what
// comes within reach, react stands until an opponent is within its move and reach and then goes for
// it, assault always advances, march goes for a cell fighting what it meets, flee goes for a cell
// without fighting and leaves the field when it gets there, and run does the same blindly: it only
// walks, never stopping to use an item.
var Policies = []string{"hold", "react", "assault", "march", "flee", "run"}

// Leaves reports whether a unit with the policy leaves the field when it reaches its goal.
func Leaves(policy string) bool { return policy == "flee" || policy == "run" }

// ValidPolicy checks a policy and its goal against each other: march, flee and run take either a goal
// cell inside the stage or one of its areas with a cell inside, the other policies neither.
func (s Stage) ValidPolicy(policy string, goal *Point, area int) bool {
	if policy != "" && !contains(Policies, policy) {
		return false
	}
	if policy == "march" || Leaves(policy) {
		return (goal == nil) != (area == 0) && len(s.GoalCells(goal, area)) > 0
	}
	return goal == nil && area == 0
}

// GoalCells are the cells a marching or fleeing unit makes for: its goal cell, or every cell of its
// goal area inside the stage.
func (s Stage) GoalCells(goal *Point, area int) []Point {
	var out []Point
	if goal != nil {
		if goal.X >= 0 && goal.Y >= 0 && goal.X < s.Width && goal.Y < s.Height {
			out = append(out, *goal)
		}
		return out
	}
	for _, a := range s.Areas {
		if a.ID != area {
			continue
		}
		for y := max(0, a.Rect.Y); y < min(s.Height, a.Rect.Y+a.Rect.H); y++ {
			for x := max(0, a.Rect.X); x < min(s.Width, a.Rect.X+a.Rect.W); x++ {
				out = append(out, Point{X: x, Y: y})
			}
		}
	}
	return out
}

// BattleEvent runs its Do once (or each time its If turns true again when Repeat) as soon as all of If hold.
type BattleEvent struct {
	ID     string
	Repeat bool     `json:",omitempty"`
	If     []string `json:",omitempty"`
	Do     []Action
}

// Goal is one way to win or lose a battle: all of If hold. Text is what the briefing says.
type Goal struct {
	If   []string
	Text string
}

// Goals are a battle's win and loss goals: the stage's at first, until a goals action of a battle
// event puts others in place.
type Goals struct{ Victory, Defeat []Goal }

// Objective is a bonus goal judged when the battle is won: if its If holds then, the player gets
// Reward and the story flag Flag.
type Objective struct {
	ID, Text, Flag string
	If             []string
	Reward         map[string]int `json:",omitempty"`
}

// VictoryTargets are the units a victory goal asks to defeat (defeated:X), whom the AI goes after.
func (s Stage) VictoryTargets() []string { return DefeatTargets(s.Victory) }

// DefeatTargets are the units the goals ask to defeat (defeated:X).
func DefeatTargets(goals []Goal) []string {
	var out []string
	for _, g := range goals {
		for _, cond := range g.If {
			if id, ok := strings.CutPrefix(cond, "defeated:"); ok {
				out = append(out, id)
			}
		}
	}
	return out
}

// DuelGroups numbers each of the stage's duels by the officers it joins, from 0 in the order the duels
// first appear: duels that share an officer (one enemy met by several, a support) get the same number.
func (s Stage) DuelGroups() []int {
	parent := map[string]string{}
	var root func(string) string
	root = func(n string) string {
		if p, ok := parent[n]; ok && p != n {
			parent[n] = root(p)
			return parent[n]
		}
		parent[n] = n
		return n
	}
	for _, d := range s.Duels {
		for _, n := range append([]string{d.Enemy}, d.Supports...) {
			parent[root(n)] = root(d.Ally)
		}
	}
	number := map[string]int{}
	out := make([]int, len(s.Duels))
	for i, d := range s.Duels {
		r := root(d.Ally)
		if _, ok := number[r]; !ok {
			number[r] = len(number)
		}
		out[i] = number[r]
	}
	return out
}

// RoutVictory reports whether routing every enemy wins the battle.
func (s Stage) RoutVictory() bool {
	for _, g := range s.Victory {
		for _, cond := range g.If {
			if cond == "alldefeated:enemy" {
				return true
			}
		}
	}
	return false
}

type Stage struct {
	TurnLimit          int           `json:",omitempty"`
	Events             []BattleEvent `json:",omitempty"`
	Areas              []Area        `json:",omitempty"`
	ID, Name           string
	Order              int         `json:"-"` // which battle of the story this is, from 1; 0 when no battle node leads here
	Width, Height, Max int         // Max is how many officers may sortie
	Required           []string    `json:",omitempty"` // officers who must sortie, if the player has them
	Victory            []Goal      // the battle is won when any goal holds
	Defeat             []Goal      `json:",omitempty"` // extra ways to lose; the fall of the lord or of every ally always loses
	Sub                []Objective `json:",omitempty"` // bonus objectives judged at the victory
	Tiles              []string
	Allies             []Point
	Enemies            []Spawn
	Reward             map[string]int
	Duels              []Duel
	Party              []Spawn `json:",omitempty"` // a skirmish's allies
	Map                string  `json:",omitempty"` // map document under maps/ that supplies the tiles; none means Tiles is inline
	Faction            string  `json:",omitempty"` // color of this stage's enemies without a faction of their own
	Forced             bool    `json:",omitempty"` // the officers sortie straight onto the Allies cells in order: no deployment screen
	Night              bool    `json:",omitempty"` // the battle is drawn at night, lit only round fires and Lights
	Boats              bool    `json:",omitempty"` // water is crossed in boats (terrain.json Boats); without it water stops every unit
	Lights             []Point `json:",omitempty"` // cells that light the night around them (torches)
	Armies             []Army  `json:",omitempty"` // the stage's armies in turn order; colors not listed follow in order of appearance
}

// Army is one side's force in a stage, by its faction color: the player's army moves first, then
// each AI army on the player's side (Ally) in this order, then each enemy army. Name is what the turn
// is called on screen (유비군, 조조군, 황건군).
type Army struct {
	Faction, Name string
	Ally          bool `json:",omitempty"`
}

// NodeCase is one arm of a branch node: the first whose conditions all hold is taken.
type NodeCase struct {
	If   []string
	Next string
}
type Node struct {
	ID, Kind, Text, Next, Event, Character, Item, Stage string
	Speaker                                             string         `json:",omitempty"` // say: character ids joined by ·
	Place                                               string         `json:",omitempty"` // scene: where the story is now
	Else                                                string         `json:",omitempty"` // branch: where no case matched; none means the next node
	Cases                                               []NodeCase     `json:",omitempty"`
	Flags                                               map[string]int `json:",omitempty"` // set: flag values to store
	Count                                               int
	Choices                                             map[string]string
	Explore                                             string  `json:",omitempty"` // explore: the explore file the hero walks
	Opening                                             string  `json:",omitempty"` // explore: the event that plays as the walk first opens
	Area                                                string  `json:",omitempty"` // back: the area the hero stands in afterwards; where he talked when empty
	At                                                  *[2]int `json:",omitempty"` // back: his small tile there
	Face                                                string  `json:",omitempty"` // back: the way he faces there
	Leave                                               bool    `json:",omitempty"` // back: the walk is over; the story goes on after the explore node
}
type EditorChoice struct{ ID, Name string }
type EditorFamily struct {
	ID, Name string
	Classes  []EditorChoice
}
type EditorCatalog struct {
	Families        []EditorFamily
	Armies          []EditorChoice
	CharacterArmies map[string]string
	Brushes         []string
}

type Data struct {
	MapEditor      EditorCatalog `json:",omitempty"`
	Title, Version string
	Hash           string `json:"-"`
	Rules          Rules
	Campaign       Campaign
	ScenarioID     string    `json:",omitempty"`
	Chapters       []Chapter `json:",omitempty"`
	Factions       map[string]Faction
	Grounds        map[string]Ground
	Structures     map[string]Structure
	States         map[string]State
	Terrain        map[string]TerrainSet
	Boats          map[string]map[string]TerrainRule `json:",omitempty"` // movement group -> water ground -> its rule on a stage with boats
	Classes        map[string]Class
	Characters     map[string]Character
	Equipment      map[string]Equipment
	Items          map[string]Item
	Skills         map[string]Skill
	Traits         map[string]Trait
	Maps           map[string]*MapDocument `json:"-"`
	Shops          map[string]Shop
	Explores       map[string]*Explore `json:"-"` // explore file name without .json -> its areas
	Stages         []Stage
	Nodes          []Node
}

// Load reads the bundle with the first scenario of the scenario folder; see LoadScenario.
func Load(dir string) (*Data, error) { return LoadScenario(dir, "") }

// LoadScenario reads the data bundle under dir (every top-level json file holds sections of Data) and the
// scenario of that id (the first when id is empty) under the scenario folder next to it (or inside
// it): the campaign start, the chapters' story nodes, and the stage files with their maps. Names
// are free; a section may be split across files.
func LoadScenario(dir, id string) (*Data, error) {
	var d Data
	h := sha256.New()
	var files []string
	err := filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err == nil && e.IsDir() && p != dir && e.Name() == "scenario" {
			return filepath.SkipDir
		}
		if err == nil && !e.IsDir() && strings.HasSuffix(p, ".json") {
			files = append(files, p)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		h.Write([]byte(filepath.ToSlash(p[len(dir):])))
		h.Write(b)
		if err = strictDecode(b, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	if err = d.loadScenario(dir, id, h); err != nil {
		return nil, err
	}
	d.Hash = fmt.Sprintf("%x", h.Sum(nil))
	d.link()
	if err := d.Validate(); err != nil {
		return nil, err
	}
	for _, id := range sortedKeys(d.Maps) {
		if err := d.Maps[id].Validate(&d); err != nil {
			return nil, fmt.Errorf("map %s: %w", id, err)
		}
	}
	return &d, nil
}

// link derives the learn pools: a class's own pool and those of the classes it promotes from.
func (d *Data) link() {
	for id, c := range d.Classes {
		c.pool = nil
		seen := map[string]bool{}
		var up func(string)
		up = func(target string) {
			for from, f := range d.Classes {
				for _, next := range f.Promotes {
					if next == target && !seen[from] {
						seen[from] = true
						up(from)
					}
				}
			}
		}
		up(id)
		var out []string
		have := map[string]bool{}
		for _, k := range append(sortedKeys(seen), id) {
			for _, t := range d.Classes[k].Pool {
				if !have[t] {
					have[t] = true
					out = append(out, t)
				}
			}
		}
		c.pool = out
		d.Classes[id] = c
	}
}
func sortedKeys[V any](m map[string]V) []string {
	k := make([]string, 0, len(m))
	for id := range m {
		k = append(k, id)
	}
	sort.Strings(k)
	return k
}

// Pool is every trait the class and the classes below it teach.
func (c Class) AllPool() []string { return c.pool }
func (d *Data) Node(id string) Node {
	for _, n := range d.Nodes {
		if n.ID == id {
			return n
		}
	}
	return Node{}
}

// StageIndex is the position of the stage with the given id, or -1.
func (d *Data) StageIndex(id string) int {
	for i, s := range d.Stages {
		if s.ID == id {
			return i
		}
	}
	return -1
}

// StageOrder is which battle of the story the stage with the given id is, from 1; 0 when none.
func (d *Data) StageOrder(id string) int {
	if i := d.StageIndex(id); i >= 0 {
		return d.Stages[i].Order
	}
	return 0
}

// NextBattleOrder is the order of the first battle the story reaches after node id; 0 when none follows.
func (d *Data) NextBattleOrder(id string) int {
	for seen := map[string]bool{}; id != "" && !seen[id]; id = d.After(id) {
		seen[id] = true
		if n := d.Node(id); n.Kind == "battle" {
			return d.StageOrder(n.Stage)
		}
	}
	return 0
}

// Spawns lists a spawn's character definition.
func (d *Data) Has(character string) bool { _, ok := d.Characters[character]; return ok }

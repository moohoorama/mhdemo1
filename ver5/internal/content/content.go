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
	CounterShare          float64 // a counter deals this share of a normal blow
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
	EnemyStock            map[string]int
	EdgeTile              string // what lies beyond the map's edge
	AuraHeal              int
}
type Campaign struct {
	Start      string // first scenario node
	Roster     []string
	Deployment []string
	Inventory  map[string]int
}

// Chapter is one story file of the scenario, in play order. Nodes lists the ids the file holds.
type Chapter struct {
	ID, File string
	Nodes    []string `json:",omitempty"`
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

// Tile describes one map glyph.
type Tile struct {
	Name           string
	Surface        int  `json:",omitempty"`
	Water          bool // beyond it lies the map edge; spells that need water look for it beside the caster or target
	Ambush         bool
	Castle         bool
	RestHP, RestMP int // percent of max HP restored at turn start; multiplier of spell point regeneration (0 = 1)
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
	Lord, Playable, Template bool
	Traits, Learn, Equipment []string
	Look                     Look
}
type Equipment struct {
	Name, Desc, Slot, Type string
	Bonus                  [6]float64
	Traits, Learn, Skills  []string
}
type Item struct {
	Name, Desc string
	Effect     string
	Amount     int
	Class      string // the class a promotion item makes
}
type Trait struct {
	Name, Desc string
	Cost       int // learning cost; 0 = given only
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
	MagicCoeff    float64 // a physical skill also deals spell damage at this coefficient
	Deprecated    bool
}
type Terrain struct {
	Factor float64
	Cost   int
}
type Point struct{ X, Y int }
type Spawn struct {
	AI                bool      `json:",omitempty"`
	Policy            string    `json:",omitempty"` // how the unit behaves: hold, assault, march, flee; none is the default advance
	Goal              *Point    `json:",omitempty"` // march and flee: the cell the unit makes for
	DuelFirst         bool      `json:",omitempty"` // the AI seeks a pending duel partner before anything else
	Class, Facing     string    `json:",omitempty"`
	Group             string    `json:",omitempty"` // name events use to address this spawn with others
	Side              string    `json:",omitempty"` // event spawns only: "ally" (an AI ally, the default) or "enemy"
	HP, MP            *int      `json:",omitempty"`
	Traits, Equipment *[]string `json:",omitempty"`
	Character         string
	X, Y              int
	Level             int    `json:",omitempty"` // overrides the character's level
	Faction           string `json:",omitempty"` // overrides the stage's and the character's faction color
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
	Kind    string
	Lines   []Speech       `json:",omitempty"`
	Spawns  []Spawn        `json:",omitempty"`
	Targets []string       `json:",omitempty"`
	Flags   map[string]int `json:",omitempty"`
	Stat    string         `json:",omitempty"` // buff: attack, defense, mind, morale, agility (percent) or speed, range (cells)
	Value   int            `json:",omitempty"` // buff: how much, negative for a debuff
	Moves   []Move         `json:",omitempty"` // move: who goes where, all at once
	Steps   [][]Step       `json:",omitempty"` // cutscene: groups of steps; a group's steps run together, the next group waits for them
	Duel    string         `json:",omitempty"` // duel: the stage duel to hold now, wherever its officers stand
	Area    int            `json:",omitempty"` // terrain: the area whose cells change
	Tile    string         `json:",omitempty"` // terrain: the tile they become
	Turns   int            `json:",omitempty"` // terrain: for how many rounds; none is for good
	Policy  string         `json:",omitempty"` // ai: the new policy of the Targets
	Goal    *Point         `json:",omitempty"` // ai: its goal cell, for march and flee
}

// Move sends a unit (or each unit of a group, to cells around To) somewhere, ignoring move range and terrain cost.
type Move struct {
	Unit string
	To   Point
}

// Step is one thing that happens in a cutscene: Unit moves to To (a group fills the cells around it),
// plays Motion (attack, hit, block, exhausted), turns to Facing, says Text, or the scene waits Seconds.
type Step struct {
	Kind, Unit, Motion, Facing, Text string
	To                               *Point  `json:",omitempty"`
	Seconds                          float64 `json:",omitempty"`
}

// BuffStats are the stats a buff can change; speed and range count cells, the rest percent.
var BuffStats = []string{"attack", "defense", "mind", "morale", "agility", "speed", "range"}

// Policies are the behaviours an AI-driven unit can be given: hold stands its ground and strikes what
// comes within reach, assault always advances, march goes for a cell fighting what it meets, and
// flee goes for a cell without fighting and leaves the field when it gets there.
var Policies = []string{"hold", "assault", "march", "flee"}

// ValidPolicy checks a policy and its goal against each other; the goal's cell must lie inside w×h.
func ValidPolicy(policy string, goal *Point, w, h int) bool {
	if policy != "" && !contains(Policies, policy) {
		return false
	}
	if policy == "march" || policy == "flee" {
		return goal != nil && goal.X >= 0 && goal.Y >= 0 && goal.X < w && goal.Y < h
	}
	return goal == nil
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

// Objective is a bonus goal judged when the battle is won: if its If holds then, the player gets
// Reward and the story flag Flag.
type Objective struct {
	ID, Text, Flag string
	If             []string
	Reward         map[string]int `json:",omitempty"`
}

// VictoryTargets are the units a victory goal asks to defeat (defeated:X), whom the AI goes after.
func (s Stage) VictoryTargets() []string {
	var out []string
	for _, g := range s.Victory {
		for _, cond := range g.If {
			if id, ok := strings.CutPrefix(cond, "defeated:"); ok {
				out = append(out, id)
			}
		}
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
	Tiles          map[string]Tile
	Terrain        map[string]map[string]Terrain
	Classes        map[string]Class
	Characters     map[string]Character
	Equipment      map[string]Equipment
	Items          map[string]Item
	Skills         map[string]Skill
	Traits         map[string]Trait
	Maps           map[string]*MapDocument `json:"-"`
	Stages         []Stage
	Nodes          []Node
}

// Load reads the data bundle under dir (every top-level json file holds sections of Data) and the
// first scenario under the scenario folder next to it (or inside it): the campaign start, the
// chapters' story nodes, and the stage files with their maps. Names are free; a section may be
// split across files.
func Load(dir string) (*Data, error) {
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
	if err = d.loadScenario(dir, h); err != nil {
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

// Spawns lists a spawn's character definition.
func (d *Data) Has(character string) bool { _, ok := d.Characters[character]; return ok }

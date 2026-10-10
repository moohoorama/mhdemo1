package content

import (
	"bytes"
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
	DamageCurve, HitCurve  Curve
	StatNumerator          float64 // an attribute a gives stat = k*(level+StatLevelBase) * (StatNumerator/(StatDenominator-a) + bonus)
	StatDenominator        float64
	StatLevelBase          int
	BasicAttack            Skill
	HitFloor, HitCeil      float64 // physical hit chance clamp
	SpellFloor, SpellCeil  float64
	MagicPower, MagicGuard float64 // spell damage = MagicPower*mind - MagicGuard*target mind
	Crit, Double           Chance
	CritMultiplier         float64
	CounterShare           float64 // a counter deals this share of a normal blow
	PoisonPercent          int
	BurnPercent            int
	BuffPercent            int // the size of an attack/defense/morale/agility buff or debuff
	BuffTurns              int // how long a trait's automatic buff lasts
	ExpBase                float64
	ExpGrowth              float64
	ExpAction, ExpSpell    float64 // action XP in percent of the target's level cost, and its spell multiplier
	ExpDefeat              float64
	ExpLevelStep           float64
	ExpMinFactor           float64
	ExpMaxFactor           float64
	ExpSupportPercent      int
	PointsPerLevel         int
	MaxLevel               int
	DeputyShare            int   // percent of the main officer's experience a deputy also earns
	DeputyWeight           int   // deputy attribute blend: ratio = (own charm*DeputyWeight + deputy charm) / DeputyDivisor
	DeputyDivisor          int   // also the percent scale of charm
	PromoteLevels          []int // minimum level to promote from tier i
	DeputyMinTier          int   // lowest class tier that may take a deputy
	EnemyStock             map[string]int
	EdgeTile               string // what lies beyond the map's edge
	AuraHeal               int
}
type Campaign struct {
	Start      string // first scenario node
	Roster     []string
	Deployment []string
	Inventory  map[string]int
}
type Faction struct {
	Name   string
	Colors []string // light, middle, dark
}

// Tile describes one map glyph.
type Tile struct {
	Name           string
	Water          bool // beyond it lies the map edge; spells that need water look for it beside the caster or target
	Ambush         bool
	Castle         bool
	RestHP, RestMP int // percent of max HP restored at turn start; multiplier of spell point regeneration (0 = 1)
}
type Class struct {
	Name, Desc                      string
	Terrain, Attack                 string // movement group; melee or ranged
	Weapon, Armor                   string // equipment types it carries
	Move, Range, MinRange, HPFactor int    // MinRange: the nearest cell a ranged class can attack
	Bonus                           [6]float64
	MPBase, RecoveryBase            int
	MPCoeff, RecoveryCoeff          float64
	Grants, Skills, Pool, Tags      []string
	Promotes                        []string
	Tier                            int
	Sprite                          string
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
	Sprite   string // battle sprite; the class's when empty
	Scene    string // non-combat sprite for scenes; Sprite when empty
	Portrait string
	Faction  string
	Palette  map[string]Colors // tag group -> one base colour or three shades
}
type Character struct {
	Name, Class              string
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
	Deprecated    bool
}
type Terrain struct {
	Factor float64
	Cost   int
}
type Point struct{ X, Y int }
type Spawn struct {
	Character string
	X, Y      int
	Level     int    `json:",omitempty"` // overrides the character's level
	Faction   string `json:",omitempty"` // overrides the stage's and the character's faction color
}
type Duel struct {
	ID, Ally, Enemy, Outcome, AllyResult, EnemyResult, Text string
	AttackDown, DefenseDown                                 int `json:",omitempty"` // an ally win cuts every enemy's attack or defense by this percent for the battle
}
type Stage struct {
	ID, Name, Boss                  string
	Width, Height, Limit, Threshold int
	Tiles                           []string
	Allies                          []Point
	Enemies                         []Spawn
	Reward                          map[string]int
	Duels                           []Duel
	Goal                            string  `json:",omitempty"` // "rout": a skirmish won by routing the other side
	Party                           []Spawn `json:",omitempty"` // a skirmish's allies
	Charge                          bool    `json:",omitempty"` // the enemy marches on the allies from the first turn
	Next                            string  `json:",omitempty"` // scenario node after a victory; none ends the campaign
	Faction                         string  `json:",omitempty"` // color of this stage's enemies without a faction of their own
}
type Node struct {
	ID, Kind, Text, Next, Event, Character, Item, Stage string
	Count                                               int
	Choices                                             map[string]string
}
type Data struct {
	Title, Version string
	Hash           string `json:"-"`
	Rules          Rules
	Campaign       Campaign
	Factions       map[string]Faction
	Tiles          map[string]Tile
	Terrain        map[string]map[string]Terrain
	Classes        map[string]Class
	Characters     map[string]Character
	Equipment      map[string]Equipment
	Items          map[string]Item
	Skills         map[string]Skill
	Traits         map[string]Trait
	Stages         []Stage
	Nodes          []Node
}

// Load reads every JSON file under dir: files at the top hold sections of Data, files under
// stages/ one Stage each. Names are free; a section may be split across files.
func Load(dir string) (*Data, error) {
	var d Data
	h := sha256.New()
	var files []string
	err := filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
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
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if filepath.Base(filepath.Dir(p)) == "stages" {
			var s Stage
			if err = dec.Decode(&s); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			d.Stages = append(d.Stages, s)
		} else if err = dec.Decode(&d); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}
	d.Hash = fmt.Sprintf("%x", h.Sum(nil))
	d.link()
	return &d, d.Validate()
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

package core

import "srpg/internal/content"

const CoreVersion = "2"
const AIPolicyVersion = "greedy-8"
const RulesVersion = "rules-13"

// Character is a persistent record: a party member, or a troop made for one battle (Temp).
type Character struct {
	ID, Def, Class, Deputy string
	Temp, Dead             bool
	XPShareRemainder       int
	Level, XP, Points      int
	Learned                []string
	TraitsOverride         *[]string `json:",omitempty"`
	Placement              bool      `json:",omitempty"`
	Equipment              map[string]string
}
type Status struct {
	Turns, Value int
	Source       string
}
type Unit struct {
	AI, DuelFirst                        bool           `json:",omitempty"`
	Policy                               string         `json:",omitempty"` // hold, assault, march or flee; none is the default advance
	Goal                                 *content.Point `json:",omitempty"` // the cell a marching or fleeing unit makes for
	ID, Character, Faction, Color        string
	Group                                string `json:",omitempty"`
	Facing                               string `json:",omitempty"`
	X, Y, HP, MP                         int
	Moved, Acted, Attacked, Done, Spared bool
	Status                               map[string]Status
	Items                                map[string]int
}

func (u Unit) ActsIn(turn string) bool {
	if u.AI && u.Faction == "ally" {
		return turn == "support"
	}
	return u.Faction == turn
}

type State struct {
	Format                            int
	Core, Rules, Content, ContentHash string
	AI, Difficulty                    string
	Revision                          uint64
	Seed, RNG                         uint64
	Phase, Node, Turn, Result, Place  string
	Stage, Round                      int
	Characters                        []Character
	Units                             []Unit
	Inventory                         map[string]int
	Warehouse                         map[string]int
	Deployment                        []string
	Executed                          map[string]bool
	Flags                             map[string]int    // story flags set by set nodes
	PendingFlags                      map[string]int    `json:",omitempty"` // flags a battle sets; they join Flags when it is won
	TileOverrides                     []TileOverride    `json:",omitempty"` // cells an event changed, for Turns more rounds (0: for good)
	Choices                           map[string]string // choice node -> the option taken
	DamagePct, XPPct                  int               `json:",omitempty"` // damage dealt and experience gained in percent; 0 is 100
	Learning                          []string          // characters whose learning window is open, in order
	Checkpoint                        *State            `json:",omitempty"`
}
// TileOverride makes a stage cell another tile (fire on grass) until it runs out.
type TileOverride struct {
	X, Y  int
	Tile  string
	Turns int
}
type Stats struct{ Attack, Defense, Mind, Agility, Morale, MaxHP, MaxMP, Recovery int }
type UnitView struct {
	Unit
	Name, Class, Def      string
	Shoot                 string `json:",omitempty"` // class whose look it takes when its attack shoots from range
	Weapon                string `json:",omitempty"` // type of the equipped weapon
	Level, XP, XPRequired int
	XPProgress            float64
	Stats                 Stats
	Attributes            [6]int // 통솔/무력/지력/민첩/운/매력 after the deputy blend
	Points                int
	Traits, Skills        []string
}

// Learner is what a learning window shows for one character.
type Learner struct {
	ID, Name string
	Points   int
	Options  []string // traits the character may learn now, whether affordable or not
}
type Observation struct {
	State
	Dialogue  content.Node
	Map       *content.Stage `json:",omitempty"`
	UnitViews []UnitView
	Learners  []Learner
}
type Command struct {
	Kind, Actor, Target, Skill, Item, Trait, Option string
	X, Y                                            int
	Deployment                                      []string
}
type Event struct {
	Speech                    []content.Speech `json:",omitempty"`
	Unit                      *Unit            `json:",omitempty"`
	DuelID                    string           `json:",omitempty"`
	Outcome                   string           `json:",omitempty"`
	First                     string           `json:",omitempty"` // duel: the officer who moved to meet the other
	Kind, Actor, Target, Text string
	Amount                    int
	Path                      []content.Point `json:",omitempty"` // move: cells walked, both ends included
}
type Result struct {
	Revision uint64
	Events   []Event
}
type Preview struct {
	Cost, MinDamage, MaxDamage int
	Healing, MPRecovery        int
	Effect                     string
	Hit, Crit, Double          float64 // percent; Crit and Double are 0 for non-physical actions
	XP, CritXP                 int     // the actor's experience in 1/100 percent of a level if it lands, normally and critically
	Targets                    []string
}
type Options struct{ Commands []Command }

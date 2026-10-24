package core

import "srpg/internal/content"

const CoreVersion = "2"
const AIPolicyVersion = "greedy-11"
const RulesVersion = "rules-14"

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
	Policy                               string         `json:",omitempty"` // hold, react, assault, march or flee; none only on a player unit
	Goal                                 *content.Point `json:",omitempty"` // the cell a marching or fleeing unit makes for
	GoalArea                             int            `json:",omitempty"` // or, instead, the stage area whose nearest cell it makes for
	ID, Character, Faction, Color        string
	Group                                string `json:",omitempty"`
	Facing                               string `json:",omitempty"`
	X, Y, HP, MP                         int
	Moved, Acted, Attacked, Done, Spared bool
	Status                               map[string]Status
	Items                                map[string]int
}

// NPC reports whether the unit keeps its own items instead of drawing on the party Inventory.
func (u Unit) NPC() bool { return u.Faction != "ally" || u.AI }

// ActsIn reports whether the unit moves in the turn: the player's units in the ally turn, an AI ally in
// the support turn of its army, an enemy in the enemy turn of its army. An empty army is the whole side.
func (u Unit) ActsIn(turn, army string) bool {
	if u.AI && u.Faction == "ally" {
		return turn == "support" && (army == "" || u.Color == army)
	}
	return u.Faction == turn && (turn == "ally" || army == "" || u.Color == army)
}

type State struct {
	Format                            int
	Core, Rules, Content, ContentHash string
	AI, Difficulty                    string
	Revision                          uint64
	Seed, RNG                         uint64
	Phase, Node, Turn, Result, Place  string
	Army                              string `json:",omitempty"` // in an AI turn (support, enemy): the faction color of the army moving now
	Stage, Round                      int
	Characters                        []Character
	Units                             []Unit
	Inventory                         map[string]int
	Warehouse                         map[string]int
	Money                             int
	Explore                           string       `json:",omitempty"` // the explore node the hero walks, while there or in one of its events
	Spot                              content.Spot `json:",omitempty"` // where the hero stands in it
	Deployment                        []string
	Executed                          map[string]bool
	Flags                             map[string]int    // story flags set by set nodes
	PendingFlags                      map[string]int    `json:",omitempty"` // flags a battle sets; they join Flags when it is won
	TileStates                        []TileState       `json:",omitempty"` // states an event put on cells, for Turns more rounds (0: for good)
	Choices                           map[string]string // choice node -> the option taken
	DamagePct, XPPct                  int               `json:",omitempty"` // damage dealt and experience gained in percent; 0 is 100
	CounterAll                        bool              `json:",omitempty"` // every unit strikes back at a physical attacker in reach
	Learning                          []string          // characters whose learning window is open, in order
	Checkpoint                        *State            `json:",omitempty"`
}

// CutStep is one step of a played cutscene as the replay shows it: a move comes with the cells
// walked, and a group of units has been told apart into one step per unit.
type CutStep struct {
	Kind, Unit, Motion, Facing, Text string
	Path                             []content.Point `json:",omitempty"`
	Seconds                          float64         `json:",omitempty"`
}

// TileState is the state an event put on a cell (fire on grass) until it runs out. A cell without one
// has the state its stage gave it; NoLayer clears that.
type TileState struct {
	X, Y  int
	State string
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
	Count                                           int `json:",omitempty"` // buy, sell: how many
	Deployment                                      []string
}
type Event struct {
	Speech                    []content.Speech `json:",omitempty"`
	Cutscene                  [][]CutStep      `json:",omitempty"` // cutscene: what the groups of steps came to, for the replay
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

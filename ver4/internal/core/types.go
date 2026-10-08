package core

import "srpg/internal/content"

const CoreVersion = "1"
const AIPolicyVersion = "greedy-4"
const RulesVersion = "opening-rules-4"

type Officer struct {
	ID, Class, Deputy string
	Dead              bool
	XPShareRemainder  int
	Level, XP, Points int
	Learned           []string
	Equipment         map[string]string
}
type Status struct {
	Turns, Value int
	Source       string
}
type Unit struct {
	ID, Officer, Faction                       string
	X, Y, HP, MP                               int
	Moved, Acted, Attacked, Done, FirstHitUsed bool
	Status                                     map[string]Status
	Items                                      map[string]int
}
type State struct {
	Format                            int
	Core, Rules, Content, ContentHash string
	AI, Difficulty                    string
	Revision                          uint64
	Seed, RNG                         uint64
	Phase, Node, Turn, Result         string
	Stage, Round                      int
	Officers                          []Officer
	Units                             []Unit
	Inventory                         map[string]int
	Warehouse                         map[string]int
	Deployment                        []string
	Executed                          map[string]bool
	Checkpoint                        *State `json:",omitempty"`
}
type Stats struct{ Attack, Defense, Mind, Agility, Morale, MaxHP, MaxMP, Recovery int }
type UnitView struct {
	Unit
	Name, Class           string
	Level, XP, XPRequired int
	XPProgress            float64
	Stats                 Stats
	Attributes            [6]int // 통솔/무력/지력/민첩/운/매력 after the deputy blend
	Points                int
	Traits, Skills        []string
	Learnable             []string // traits the officer may still learn, owned ones left out
}
type Observation struct {
	State
	Dialogue  content.Node
	Map       *content.Stage `json:",omitempty"`
	UnitViews []UnitView
}
type Command struct {
	Kind, Actor, Target, Skill, Item, Trait, Option string
	X, Y                                            int
	Deployment                                      []string
}
type Event struct {
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
	XP, CritXP                 int     // the actor's raw experience if it lands, normally and critically
	Targets                    []string
}
type Options struct{ Commands []Command }

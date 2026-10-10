package content

// The engine's vocabulary. A data bundle names things freely (characters, classes, items,
// skills, traits); it can only combine these identifiers, and new ones need engine code.

// Trait effects. A trait's Effects maps one of these keys to a number; a flag is 1.
// Effects of the same key from several traits (own, deputy, equipment, class grants) add up.
const (
	FxCritWeaker     = "crit_vs_weaker"   // physical attacks on a lower-might target always crit
	FxCritAlways     = "crit_always"      // physical attacks always crit
	FxSpellCrit      = "spell_crit"       // landed attack spells always crit
	FxCritAmbush     = "crit_ambush"      // physical attacks always crit from an ambush tile
	FxDoubleAlways   = "double_always"    // second strike chance is 100%
	FxCounterForce   = "counter_force"    // counters a lower-might attacker in reach; value is the damage multiplier
	FxAssist         = "assist"           // follows an ally's physical attack on a foe in reach; value is the damage share
	FxSplashCross    = "splash_cross"     // physical attack also hits the cross; value is the damage share
	FxSplashSquare   = "splash_square"    // physical attack also hits the 3x3; value is the damage share
	FxPierce         = "pierce"           // the enemy behind the target takes this share
	FxIgnoreTerrain  = "ignore_terrain"   // target's terrain defense is ignored
	FxPoison         = "poison_on_hit"    // landed physical hits poison
	FxLifeSteal      = "life_steal"       // heals this share of damage dealt
	FxMissHalf       = "miss_half"        // a missed physical blow still deals this share
	FxBulwark        = "bulwark"          // blows above this share of max HP deal half above it
	FxCritImmune     = "crit_immune"      // a blow that would crit is blocked
	FxRangedImmune   = "ranged_immune"    // ranged attackers always miss
	FxRiposte        = "riposte"          // strikes back at a physical attacker that misses; value is the damage multiplier
	FxReflect        = "reflect"          // returns an attack spell to its caster
	FxFortify        = "fortify"          // on a castle tile, a defense buff of this percent each turn
	FxMove           = "move"             // movement points
	FxIgnoreMoveCost = "ignore_move_cost" // terrain never slows
	FxIgnoreZOC      = "ignore_zoc"       // zone of control never stops
	FxHaste          = "haste"            // each turn, a movement buff of this many points
	FxFarSight       = "far_sight"        // each turn, a range buff of this many cells
	FxMountedRange   = "mounted_range"    // mounted classes may attack at range 1..value
	FxLightMove      = "light_move"       // light classes gain this movement
	FxSpellRange     = "spell_range"      // spell reach
	FxSpellSure      = "spell_sure"       // spells on lower-mind targets never fail
	FxChain          = "chain"            // attack spells repeat on one neighbour
	FxSure           = "sure:"            // + element: spells of the element never fail
	FxCritElement    = "crit:"            // + element: spells of the element always crit
	FxDispel         = "dispel"           // spells from lower-mind casters fail
	FxStatusImmune   = "status_immune"    // harmful statuses never land
	FxHealBonus      = "heal_bonus"       // healing amounts grow by this share
	FxAura           = "aura_heal"        // adjacent allies recover this share at turn start
	FxDeputy         = "deputy_rate"      // the deputy's attributes count at this rate
	FxItemShare      = "item_share"       // items also affect one more adjacent ally
	FxManaRegen      = "mana_regen"       // spell points regenerate this many times faster
	FxBloodCost      = "blood_cost"       // missing spell points are paid in HP at this ratio
	FxRegen          = "regen"            // recovers this share of max HP at turn start
	FxBuffExtend     = "buff_extend"      // buffs it casts last this many turns longer
	FxLastStand      = "last_stand"       // survives its first fall with 1 HP
	FxNoMinRange     = "no_min_range"     // ranged attacks have no minimum range
	FxCostHalf       = "cost_half"        // skill costs are halved
	FxCleanse        = "cleanse"          // harmful statuses end at turn start
	FxFirstHit       = "first_hit_half"   // the first hit taken each enemy turn deals half
)

// TraitEffects lists every trait effect key; keys ending in ':' take an element after it.
var TraitEffects = map[string]bool{}

func init() {
	for _, k := range []string{FxCritWeaker, FxCritAlways, FxSpellCrit, FxCritAmbush, FxDoubleAlways, FxCounterForce, FxAssist,
		FxSplashCross, FxSplashSquare, FxPierce, FxIgnoreTerrain, FxPoison, FxLifeSteal, FxMissHalf, FxBulwark, FxCritImmune,
		FxRangedImmune, FxRiposte, FxReflect, FxFortify, FxMove, FxIgnoreMoveCost, FxIgnoreZOC, FxHaste, FxFarSight,
		FxMountedRange, FxLightMove, FxSpellRange, FxSpellSure, FxChain, FxSure, FxCritElement, FxDispel, FxStatusImmune,
		FxHealBonus, FxAura, FxDeputy, FxItemShare, FxManaRegen, FxBloodCost, FxRegen, FxBuffExtend, FxLastStand,
		FxNoMinRange, FxCostHalf, FxCleanse, FxFirstHit} {
		TraitEffects[k] = true
	}
}

// Skill kinds, modes, shapes and effects.
const (
	KindPhysical = "physical"
	KindMagic    = "magic"
	KindStatus   = "status"
	KindBuff     = "buff"
	KindHeal     = "heal"

	ModeMelee  = "melee"
	ModeRanged = "ranged"
	ModeSelf   = "self"
	ModeAny    = "any"

	ShapeSingle = "single"
	ShapeCross  = "cross"
	ShapeSquare = "square"
	ShapeLine   = "line"
	ShapeAll    = "all"

	AttackMelee  = "melee"
	AttackRanged = "ranged"

	ItemHP      = "hp"
	ItemMP      = "mp"
	ItemPromote = "promote"
)

// Skill effects: a heal's kind, or the status a buff or status skill applies.
var SkillEffects = map[string]bool{
	"heal": true, "cure": true, "dispel": true, "steal": true, "eightfold": true, "counter": true,
	"mind": true, "attack": true, "defense": true, "morale": true, "agility": true, "speed": true, "range": true,
	"poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "disarm": true, "refill": true,
}

// Harmful statuses, ended by status immunity and cleansing.
var Harmful = map[string]bool{"poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "disarm": true, "weak": true}

// Statuses lists every status id a unit can carry.
var Statuses = map[string]bool{
	"poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "disarm": true, "weak": true,
	"speed": true, "range": true, "counter": true, "eightfold": true,
	"attack": true, "defense": true, "morale": true, "agility": true, "attack-down": true, "defense-down": true,
}

// Class tags the engine reads.
const (
	TagMounted = "mounted" // rides: gains ranged attacks from FxMountedRange
	TagLight   = "light"   // gains movement from FxLightMove
)

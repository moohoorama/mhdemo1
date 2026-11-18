package content

// The engine's vocabulary. A data bundle names things freely (characters, classes, items,
// skills, traits); it can only combine these identifiers, and new ones need engine code.

// Trait effects. A trait's Effects maps one of these keys to a number; a flag is 1.
// Effects of the same key from several traits (own, deputy, equipment) add up. Keys ending in ':' take
// a status (or an element, or a weapon type) after them.
const (
	FxDiagonal           = "diagonal"             // the melee reach also takes the four diagonal cells
	FxLongReach          = "long_reach"           // the melee reach runs two cells along the four straight lines
	FxRetreatSweep       = "retreat_sweep"        // defeats trigger one extra strike on surrounding foes
	FxCritAlways         = "crit_always"          // physical attacks always crit
	FxCritBoost          = "crit_boost"           // added to the critical multiplier of its blows
	FxSpellCrit          = "spell_crit"           // landed attack spells always crit
	FxCritAmbush         = "crit_ambush"          // physical attacks always crit from an ambush tile (woods, mountains)
	FxCritScholar        = "crit_vs_scholar"      // physical attacks on a scholar class always crit
	FxCritMounted        = "crit_vs_mounted"      // physical attacks on a mounted class always crit
	FxCritWounded        = "crit_vs_wounded"      // attacks on a target at half HP or less always crit
	FxLoneCritLeader     = "lone_crit_leader"     // with no ally around, attacks on a commander or boss always crit
	FxDoubleAlways       = "double_always"        // second strike chance is 100%
	FxAssist             = "assist"               // follows an ally's physical attack on a foe in reach; value is the damage share
	FxRetaliate          = "retaliate"            // counters every physical attack, from any distance
	FxGuardian           = "guardian"             // counters at this share a melee attacker of an adjacent ally that it can reach
	FxSplashCross        = "splash_cross"         // physical attack also hits the cross; value is the damage share
	FxSplashCrossRanged  = "splash_cross_ranged"  // ranged physical attack also hits the cross; value is the damage share
	FxSplashSquare       = "splash_square"        // physical attack also hits the 3x3; value is the damage share
	FxSplashSquareRanged = "splash_square_ranged" // ranged physical attack also hits the 3x3; value is the damage share
	FxPierce             = "pierce"               // the enemy behind the target takes this share
	FxFlank              = "flank"                // a melee strike also hits the two cells beside the target; value is the damage share
	FxHeavy              = "heavy"                // added to the share of its counters, second strikes and side blows
	FxCounterBoost       = "counter_boost"        // added to the share of its counters
	FxKnockback          = "knockback"            // a landed first blow pushes the target one cell back; where it cannot go, the blow deals this share more
	FxTerrainCap         = "terrain_cap"          // the target's terrain factor counts at most 1
	FxTerrainFloor       = "terrain_floor"        // its own terrain factor counts at least 1
	FxGrassFactor        = "grass_factor"         // its own terrain factor grows by this much on bare grassland
	FxTerrainFactor      = "terrain_factor"       // its own terrain factor grows by this much on any ground
	FxWetFactor          = "wet_factor"           // its own terrain factor grows by this much in water, a ford or a wet patch
	FxAmbushFactor       = "ambush_factor"        // its own terrain factor grows by this much on an ambush cell (woods, mountains)
	FxWard               = "ward:"                // + element: spells of the element on it deal this share less
	FxDeputyPlus         = "deputy_plus"          // its deputy's attributes count this much higher when they are blended in
	FxPoison             = "poison_on_hit"        // landed physical hits poison
	FxPoisonBoost        = "poison_boost"         // the poison it puts on deals this many times more
	FxLifeSteal          = "life_steal"           // heals this share of damage dealt
	FxPlunder            = "plunder"              // each landed physical blow of a player's unit brings the party this much money
	FxMissHalf           = "miss_half"            // a missed physical blow still deals this share
	FxOutnumbered        = "outnumbered"          // attack and defense grow by this share for each foe in the eight cells around
	FxCompanions         = "companions"           // attack and defense grow by this share for each ally in the eight cells around
	FxLone               = "lone"                 // attack and defense grow by this share with no ally in the eight cells around
	FxChargePower        = "charge_power"         // attack grows by this share for each cell walked this turn
	FxChargeShock        = "charge_shock"         // a melee attack after walking this many cells lowers the target's defense
	FxChargeAccuracy     = "charge_accuracy"      // physical hit chance grows by this many percentage points for each cell walked this turn
	FxOpening            = "opening"              // its blows deal this share more of the target's HP share above half
	FxDesperate          = "desperate"            // at half HP or less its blows deal this share more
	FxGrit               = "grit"                 // at half HP or less blows on it deal this share less
	FxHoldGround         = "hold_ground"          // ending its own turn without walking, blows on it deal this share less until its next turn
	FxRangedSure         = "ranged_sure"          // its ranged attacks always hit
	FxRout               = "rout:"                // + status: defeating a foe lowers that stat of the foes around it
	FxHitDebuff          = "hit_debuff:"          // + status: a landed physical hit lowers that stat of the target
	FxTurnBuff           = "turn_buff:"           // + status: a buff of this percent on itself each own turn
	FxCrush              = "crush"                // felling a foe lowers every stat of the foes around it
	FxScorn              = "scorn"                // physical blows from a foe at least 10 below its 무력 deal this share less
	FxImmune             = "immune:"              // + element: spells of the element deal it no damage, no critical, no mark
	FxAuraBuff           = "aura_buff:"           // + status: a buff of this percent on the allies around it each own turn
	FxBulwark            = "bulwark"              // blows above this share of max HP deal half above it
	FxCritImmune         = "crit_immune"          // a blow that would crit is blocked
	FxRangedImmune       = "ranged_immune"        // ranged attackers always miss
	FxRangedWard         = "ranged_ward"          // ranged physical blows on it deal this share less
	FxMagicWard          = "magic_ward"           // enemy attack spells on it deal this share less
	FxCounterWard        = "counter_ward"         // counters struck back at it lose this much of their share
	FxStartMana          = "start_mana"           // starts a battle with this share of its max spell points
	FxReflect            = "reflect"              // returns an attack spell to its caster
	FxReflectStatus      = "reflect_status"       // returns a harmful status spell to its caster
	FxMove               = "move"                 // movement points
	FxIgnoreMoveCost     = "ignore_move_cost"     // terrain never slows
	FxIgnoreZOC          = "ignore_zoc"           // zone of control never stops
	FxHaste              = "haste"                // each turn, a movement buff of this many points
	FxAuraHaste          = "aura_haste"           // each turn, a movement buff of this many points on the allies around it
	FxRangedReach        = "ranged_reach"         // ranged attacks reach this many cells farther
	FxMountedRange       = "mounted_range"        // mounted classes may shoot at this distance
	FxSpellRange         = "spell_range"          // spell reach
	FxSpellSure          = "spell_sure"           // its spells never fail
	FxSpellWard          = "spell_ward"           // enemy spells on it succeed this many percentage points less often
	FxChain              = "chain"                // single-target spells also reach the weakest of the target's side in the eight cells around it
	FxSure               = "sure:"                // + element: spells of the element never fail
	FxCritElement        = "crit:"                // + element: spells of the element always crit
	FxPower              = "power:"               // + element: spells of the element deal this share more
	FxPatience           = "patience"             // spells deal (spell points − 50)% more above 50 spell points
	FxFinisher           = "finisher"             // spells deal this share of the target's missing HP fraction more
	FxDispelBuffs        = "dispel_buffs"         // its attack and status spells strip the target's buffs
	FxSpellLeech         = "spell_leech"          // casting an attack spell restores this share of its max HP
	FxDispel             = "dispel"               // enemy attack and status spells never land on it
	FxStatusImmune       = "status_immune"        // harmful statuses never land
	FxStatusWard         = "status_ward"          // harmful statuses never land on the allies next to it
	FxHealBonus          = "heal_bonus"           // heal spells it casts grow by this share
	FxAura               = "aura_heal"            // adjacent allies recover this share at turn start
	FxAuraMana           = "aura_mana"            // adjacent allies recover this share of their max spell points at turn start
	FxItemShare          = "item_share"           // items also affect the weakest ally in the eight cells around the target
	FxItemPower          = "item_power"           // items it uses restore this share more
	FxManaRegen          = "mana_regen"           // spell points regenerate this many times faster
	FxBloodCost          = "blood_cost"           // missing spell points are paid in HP at this ratio
	FxRegen              = "regen"                // recovers this share of max HP at turn start
	FxWoundRegen         = "wound_regen"          // recovers this share of its missing HP at turn start
	FxBuffExtend         = "buff_extend"          // buffs it casts last this many turns longer
	FxSelfGuardHalf      = "self_guard_half"      // a defense buff it casts on itself costs half
	FxLastStand          = "last_stand"           // survives its first fall with 1 HP
	FxCostHalf           = "cost_half"            // skill costs are halved
	FxCleanse            = "cleanse"              // harmful statuses end at turn start
	FxSpellCure          = "spell_cure"           // its spells on its own side (heal, buff, ...) also end the target's harmful statuses
	FxAbsorbMana         = "absorb_mana"          // defeating a foe takes the spell points the foe had left
	FxStatusDouble       = "status_double"        // the debuffs and ailments it casts last twice as long
	FxCritAiling         = "crit_vs_ailing"       // attacks and spells on a foe with an ailment always crit
	FxPassAilments       = "pass_ailments"        // at turn start its harmful statuses pass to an adjacent ally
	FxBlendBonus         = "blend_bonus"          // the lower of the class attack and mind bonuses rises to their average
	FxDeputyAppoint      = "deputy_appoint"       // may name a deputy
	FxWield              = "wield:"               // + weapon type: may also equip weapons of that type
	FxEvade              = "evade"                // physical attacks on it hit this many percentage points less often
	FxAccuracy           = "accuracy"             // its physical attacks hit this many percentage points more often
	FxPouchHeal          = "pouch_heal"           // after an attack hurts it and leaves it standing, it recovers this many HP
	FxHitStatus          = "hit_status:"          // + harmful status: a landed physical hit puts that status on the target for this many turns
	FxDoubleWard         = "double_ward"          // the second strike of an attack on it is blocked
	FxCharm              = "charm"                // its 매력 counts this much higher (max HP, deputy blend)
	FxManaMax            = "mana_max"             // its max spell points grow by this many
	FxManaPlus           = "mana_plus"            // it recovers this many more spell points each turn
)

// TraitEffects lists every trait effect key; keys ending in ':' take a status, element or weapon type after it.
var TraitEffects = map[string]bool{}

func init() {
	for _, k := range []string{FxDiagonal, FxLongReach, FxRetreatSweep, FxCritAlways, FxCritBoost, FxSpellCrit, FxCritAmbush, FxCritScholar,
		FxCritMounted, FxCritWounded, FxLoneCritLeader, FxDoubleAlways, FxAssist, FxRetaliate, FxGuardian, FxSplashCross, FxSplashCrossRanged, FxSplashSquare, FxSplashSquareRanged,
		FxPierce, FxFlank, FxHeavy, FxCounterBoost, FxKnockback, FxTerrainCap, FxTerrainFloor, FxGrassFactor, FxTerrainFactor, FxWetFactor, FxAmbushFactor, FxWard, FxDeputyPlus, FxPoison, FxPoisonBoost, FxLifeSteal, FxPlunder,
		FxMissHalf, FxOutnumbered, FxCompanions, FxLone, FxChargePower, FxChargeShock, FxChargeAccuracy, FxOpening,
		FxDesperate, FxGrit, FxHoldGround, FxRangedSure, FxRout,
		FxHitDebuff, FxTurnBuff, FxCrush, FxScorn, FxImmune, FxAuraBuff, FxBulwark, FxCritImmune, FxRangedImmune, FxRangedWard, FxMagicWard, FxCounterWard, FxStartMana, FxReflect, FxReflectStatus,
		FxMove, FxIgnoreMoveCost, FxIgnoreZOC, FxHaste, FxAuraHaste, FxRangedReach, FxMountedRange, FxSpellRange,
		FxSpellSure, FxSpellWard, FxChain, FxSure, FxCritElement, FxPower, FxPatience, FxFinisher, FxDispelBuffs, FxSpellLeech, FxDispel,
		FxStatusImmune, FxStatusWard, FxHealBonus, FxAura, FxAuraMana, FxItemShare, FxItemPower, FxManaRegen,
		FxBloodCost, FxRegen, FxWoundRegen, FxBuffExtend, FxSelfGuardHalf, FxLastStand, FxCostHalf, FxCleanse, FxSpellCure, FxAbsorbMana, FxStatusDouble, FxCritAiling, FxPassAilments,
		FxBlendBonus, FxDeputyAppoint, FxWield, FxEvade, FxAccuracy, FxPouchHeal,
		FxHitStatus, FxDoubleWard, FxCharm, FxManaMax, FxManaPlus} {
		TraitEffects[k] = true
	}
}

// StatusEffects are the trait effects a unit holds while it carries the status: the stances a buff puts on it.
var StatusEffects = map[string]map[string]float64{
	"stance_pierce": {FxPierce: .5},
	"stance_cross":  {FxSplashCross: .5},
	"stance_square": {FxSplashSquare: .5},
}

// Skill kinds, modes, shapes and effects.
const (
	KindPhysical = "physical"
	KindMagic    = "magic"
	KindStatus   = "status"
	KindBuff     = "buff"
	KindHeal     = "heal"
	KindMixed    = "mixed" // helps the caster's side and harms the other in the same area

	TabMartial = "martial" // a Skill.Tab: listed with the physical skills in the command menu

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
	ItemCure    = "cure" // lifts every harmful status
)

// Skill effects: a heal's kind, the status a buff or status skill applies, or what a physical skill adds to its blow.
// A buff, status or mixed skill may join several stat statuses with '+' ("attack+defense").
var SkillEffects = map[string]bool{
	"heal": true, "cure": true, "dispel": true, "counter": true,
	"mind": true, "attack": true, "defense": true, "morale": true, "agility": true, "speed": true, "range": true,
	"poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "disarm": true, "refill": true,
	"guard": true, "disrupt": true, "invert": true, "sure_crit": true, "meditate": true,
	"stance_pierce": true, "stance_cross": true, "stance_square": true, "clear_marks": true,
}

// StatStatuses are the statuses that raise or lower a battle stat by a percentage.
var StatStatuses = map[string]bool{"attack": true, "defense": true, "mind": true, "morale": true, "agility": true}

// Harmful statuses, ended by status immunity and cleansing.
var Harmful = map[string]bool{"disrupt": true, "poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "disarm": true, "weak": true}

// Statuses lists every status id a unit can carry.
var Statuses = map[string]bool{
	"poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "disarm": true, "weak": true,
	"speed": true, "range": true, "counter": true, "sure_crit": true,
	"stance_pierce": true, "stance_cross": true, "stance_square": true,
	"attack": true, "defense": true, "mind": true, "morale": true, "agility": true,
	"guard": true, "disrupt": true, "hold_ground": true,
}

// Class tags the engine reads.
const (
	TagMounted = "mounted" // rides: gains ranged attacks from FxMountedRange, and 대기병 전술 crits it
	TagSiege   = "siege"   // a siege engine: the screens show its attacks as stones lobbed along an arc
	TagCivil   = "civil"   // a civil officer's class: 낙봉 crits it
)

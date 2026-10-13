# assets/data — ver5 game data bundle

`content.Load("assets/data")` reads every `.json` here. Top-level files hold sections of `content.Data`
(split freely); `maps/<ID>.json` holds one shared map document each, ordered by file name (B01, B02, B03).
Stage grids and graphics are derived in memory. Simple `stages/*.json` definitions remain supported for test/custom bundles.
Scenes live in `assets/scenes.json`, not here.

| File | Section |
|---|---|
| `game.json` | Title, Version, Rules, Campaign, Factions (3 shades: light, mid, dark), Tiles legend |
| `terrain.json` | Terrain tables per movement group: `cavalry infantry archer scholar bandit` |
| `classes.json` | Classes (Tier/Promotes chain, Grants = given traits, Pool = learnable traits of that tier, Skills = given skills) |
| `characters.json` | Officers and `Template: true` soldier templates (`troop_*`) |
| `traits.json`, `skills.json` | Trait effects (vocab.go keys) and skills; spells and special moves open through `Requires` traits |
| `equipment.json`, `items.json` | Equipment (slots weapon/armor/accessory) and items (`hp`/`mp`/`promote`) |
| `script.json` | Scenario nodes |
| `maps/*.json` | Shared editor/game map: small tiles, resolved terrain, exterior, spawns, rewards, duels, `Next` node |

## Id conventions

- Every map key and reference is an English snake_case id; Korean appears only in `Name`, `Desc`, `Text`, `Biography` (and choice labels).
- Characters use pinyin matching portrait/sprite stems (`liubei`, `lubu`, ...). Soldier templates: `troop_turban`, `troop_footman`, `troop_archer`.
- Classes: English names (`light_infantry`, `scholar`, `staff_officer`, `sorcerer`, ...). Equipment types: `sword spear bow fan` / `armor robe` / `any`.
- Skills: English (`scorch`, `minor_supply`, `sweep`); traits: English (`valor`, `fire_boost`). Spell family roots (`fire water earth relief exhort hinder stratagem`) are given-only (Cost 0).
- Promotion items: `promote_<class id>`. Scenario events: `join-<id>`, `grant-<id>`.

## Officer biographies

`Character.Biography` is optional display text for the battle detail panel’s fourth tab. Named officers use the biographies in `docs/officers.md`; coalition officers added for the scenario describe their in-game story. Soldier templates can omit it.

## Trait rules

- Given traits (class Grants, equipment Traits) have Cost 0 and are never learnable. Learnable traits (Cost 300-1500) sit in Class.Pool or Character.Learn.
- Player characters start with `Traits: []` and a `Learn` list; enemies and NPCs get `Traits` directly.
- Tiered spell families: root + `<x>_boost` / `<x>_spread`; top spells need all three.

`tools/migrate/gen_data.py` regenerated this bundle from ver4 once; after that, edit the JSON directly.

## Map editor overrides

`mapeditor.json` defines terrain brush order, class families (including disabled choices),
army tabs and officer-to-army membership. These labels remain content data, separate from faction colors.
`Tiles.Surface` selects the ground atlas surface (0 water, 1 grass, 2 wasteland).
`Class.Family`, when present, overrides the movement group used for editor family selection.

Stage spawns can override `Class`, `Level`, `Facing` (`NE SE SW NW`),
current `HP`/`MP`, `Traits` and `Equipment`. Omitted HP starts at the derived maximum and omitted MP keeps the existing initial value (0);
explicit MP 0 is supported. Maxima are calculated by the engine, not stored overrides.
An omitted trait/equipment list uses the character definition; an explicit empty list removes those choices.
Class and equipment grants still apply. Officers must be unique; soldier templates create independent instances.
Class changes must remain in the same family, and equipment must fit the resulting class and slots.

The GUI saves exactly one map document. The game reads that same file; see
[the map format](../../docs/maps.md). Defense XP is controlled by `Rules.ExpDefenseShare`.

## Battle events and allied AI

`Spawn.Guard` holds a unit at its spawn cell. `Spawn.AI` makes an allied spawn act in the automatic `support` phase between player and enemy phases. Such reinforcements are temporary and cannot be commanded by the player.

`Stage.Events` defines one-shot events with a stage-local `ID`. `Trigger: "boss-alone"` fires while the boss survives and all other enemies are gone. `Trigger: "duel"` fires after any ID listed in `Duels` resolves. `Spawns` contains allied AI spawns; occupied entry cells use the closest usable free cell. `Dialogue` is an ordered array of groups of `{Actor, Text}` lines. Lines in the same group appear simultaneously in battlefield speech bubbles. Executed events and spawned units survive save/load.

Equipment can grant `retreat_sweep` (damage share for one additional strike on foes in the defeated target’s surrounding eight cells), `counter_always` (a counter against every physical attack in reach), and `turn_defense` (defense percentage renewed at the start of each own phase). The effects follow the wearer’s faction. Additional sweep blows do not cascade.

`Stage.TurnLimit` sets the battle round deadline. Zero or omitted means unlimited. A positive value allows the entire final round (including the enemy phase); entering the next allied round loses the battle. B01–B03 currently use 30 rounds.

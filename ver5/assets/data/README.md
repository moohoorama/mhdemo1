# assets/data — ver5 game data bundle

`content.Load("assets/data")` reads every `.json` here. Top-level files hold sections of `content.Data`
(split freely); `stages/<ID>.json` holds one `Stage` each, ordered by file name (B01, B02, B03).
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
| `stages/*.json` | Maps, spawns (`Character` + optional `Level`), rewards, duels, `Next` node |

## Id conventions

- Every map key and reference is an English snake_case id; Korean appears only in `Name`, `Desc`, `Text` (and choice labels).
- Characters use pinyin matching portrait/sprite stems (`liubei`, `lubu`, ...). Soldier templates: `troop_turban`, `troop_footman`, `troop_archer`.
- Classes: English names (`light_infantry`, `scholar`, `staff_officer`, `sorcerer`, ...). Equipment types: `sword spear bow fan` / `armor robe` / `any`.
- Skills: English (`scorch`, `minor_supply`, `sweep`); traits: English (`valor`, `fire_boost`). Spell family roots (`fire water earth relief exhort hinder stratagem`) are given-only (Cost 0).
- Promotion items: `promote_<class id>`. Scenario events: `join-<id>`, `grant-<id>`.

## Trait rules

- Given traits (class Grants, equipment Traits) have Cost 0 and are never learnable. Learnable traits (Cost 300-1500) sit in Class.Pool or Character.Learn.
- Player characters start with `Traits: []` and a `Learn` list; enemies and NPCs get `Traits` directly.
- Tiered spell families: root + `<x>_boost` / `<x>_spread`; top spells need all three.

`tools/migrate/gen_data.py` regenerated this bundle from ver4 once; after that, edit the JSON directly.

---
name: duel-authoring
description: Writes or reworks a duel (일기토) of the ver5 SRPG — how many 합 the pair should fight from their 무력 gap, the flow of the exchange (pressing, being pushed back, dodging, countering, clashing, backing off), the lines between exchanges in each officer's voice, and the draw/stay rule — as Stage.Duels[].Beats in assets/scenario/hero/stageNN.json built with tools/duel_flow/flow.py, then checked by tests and one game capture. Use for "일기토 만들어줘/늘려줘/다시", "합 수 맞춰", "공방이 단조로워", "일기토 대사", or when a new duel is added to a stage.
---

# Duel authoring

Work from `ver5/`. Read `docs/presentation-duel.md` (beat kinds, what each does on screen), `docs/logic-duel.md`
(results, XP), `docs/data-scenario-node.md` §5 (fields) and `docs/data-speech-style.md` (each officer's voice) first.
Report in Korean.

## 1. How many 합

A duel's 합 are its exchange beats: every `block`, `rush-block`, `yielding-block`, `critical-block`, `dodge`,
`critical-dodge`, `hit`, `critical` (talk, approach, press, close, backstep, retreat, win, draw are not).

```
합 = max(5, 30 − |아군 무력 − 적 무력|)        (무력 = Stats[1])
```

- 무력이 비슷하면 30합, 10 차이면 20합, 20 차이면 10합. 아무리 차이가 나도 최소 5합.
- Use the *effective* 무력 when the story says so (수공에 지친 장료는 86, 파재는 60대). The data keeps the printed
  value; only the 합 count follows the story.
- 1장 두 판(등무·정원지)은 첫 일기토를 다양하게 보여 주는 튜토리얼이라 예외다.
- 목순·방열처럼 짧게 끝나는 판은 기준 밖의 의도된 예외로 두되, 최소 5합 적용 여부는 사용자에게 묻는다.
- The attitude of the characters may bend the count (예: 말없이 베는 고순은 빠른 쪽, 서로 베지 못하는 관우·장료는 긴 쪽).
  Say so when a duel deviates from the line.

## 2. Rules that must hold

- `Outcome: draw` → `AllyResult` and `EnemyResult` are both `stay` (validate.go rejects anything else). The scene may
  still end with someone riding off; the map result is stay.
- The last beat matches `Outcome` (`win` side 0/1, `withdrawn` side 1/0, `draw`). `retreat` → `withdrawn` for a retreat.
- `Side` of an exchange beat is the **attacker**. The defender is the other side. Speech on `block`, `yielding-block`,
  `dodge`, `critical-dodge` is the **defender's** line; on `critical-block`, `critical`, `hit` the attacker's. A `dodge`
  whose title says "B가 피하다" must have Side = A (the one who swings).
- `Speech` is spoken text, `Title` is the caption shown at the top of the screen. Never put the caption in Speech.
- Do not chain `rush-*` (the other fighter keeps being pushed back) and do not chain `press` from one side more than
  twice; alternate who presses so the pair stays centred.
- The log `Text` must not contradict the count ("세 합", "단칼에" → "수십 합", "십여 합" or remove it).
- Weapons: 장비 사모, 관우 청룡도, 고순·조운·악진·장무 창, 기령 삼첨도, 서황 도끼, 적병 칼. Check
  `data-speech-style.md` and `characters*.json` before naming one (고순은 창이다).

## 3. Flow, not trading blows

Alternating plain `block`s reads as two people hitting each other. Give the duel phases where the initiative moves:

| token | beats | 합 |
|---|---|---|
| `probe(a)` | a tests, b turns it aside and answers | 2 |
| `trade(a)` | strike and counter-strike | 2 |
| `press(a)` | a drives forward, b backs up (`press`), `rush-block`, `yielding-block` | 2 |
| `evade(a)` | a swings big, b dodges | 1 |
| `counter(a)` | a slips in on an opening | 1 |
| `clash(a)` | two `critical-block`s, sparks | 2 |
| `lock(a)` / `push(a)` | blades locked / a shoves b back (`yielding-block`) | 1 |
| `rush(a)` | a charges on horseback | 1 |
| `cb(a)` / `edodge(a)` | strong blow blocked / barely dodged | 1 |
| `stag(a)` | `hit`, b is knocked back (not in draws or even matches) | 1 |
| `kill(a)` | `critical`, the finishing blow | 1 |
| `gap(a)` / `close(a)` | a rides back to open the distance / closes again | 0 |

Shape: probe → one side presses → the other evades and counters → trade → clash → a breather (`gap`, talk) → the
other side presses → … → the closing beat. The stronger fighter wins the last phases; in a near-even match the
initiative changes hands every two or three exchanges; in a draw nobody gets a `stag`/`kill`.

## 4. Lines

- A line (or a quiet situation beat) after every one or two exchanges; never more than three exchanges in silence.
  Put them as `talk` beats with the speaker's `Side`, plus a few short `Speech` values on dodges/criticals.
- What goes in: what is at stake in this fight (a road to hold, a horse to steal, a debt, 하비의 은혜), what the
  place does to the fight (진흙에 빠진 말, 물이 불어남, 날이 저뭄), what the loser feels (속말은 괄호), and the
  attitude between the two (존중, 비웃음, 아쉬움). Lines come from the story the duel sits in
  (`docs/fullstory.md`, 정사·연의); do not invent facts the story contradicts.
- Voice per officer from `data-speech-style.md` (장비 충청 방언 한두 군데, 관우 짧은 단정형, 조운 하오체·반말,
  장료 하오체, 하후은·장무·진손 거친 반말). 장비는 유비를 "큰형", 관우를 "둘째 형"이라 부르고 "형님"은 쓰지 않는다.
- Keep each line short (one or two bubble lines); the existing opening and closing lines stay unless the user asks.

## 5. Build

`tools/duel_flow/flow.py` turns a plan into beats, checks the 합 count against the target and rewrites only the
`Beats` (and nothing else) of the named duels, keeping the file's formatting.

```sh
python3 tools/duel_flow/flow.py tools/duel_flow/example_plans.py          # dry run, prints target vs got
python3 tools/duel_flow/flow.py tools/duel_flow/example_plans.py --write  # writes stageNN.json
```

A plan file calls `plan(duel_id, target合, [name0, name1], [weapon0, weapon1], tokens)`; tokens are the rows above
plus `t(side, title, text)` (talk), `ap(side, title)` (approach) and `E(kind, side, title, speech)` for explicit
beats (the closing `win`/`retreat`/`withdrawn`/`draw`/`backstep`). See `tools/duel_flow/example_plans.py`.
Stage files are shared with other sessions: touch only the duels you were asked about.

## 6. Check

```sh
go test -p 1 ./internal/duel
go test -p 1 ./internal/content -run 'Duel|Scenario|Stage|Load'
go test -p 1 ./internal/gui -run Duel
go run ./cmd/srpg-gui -duel <duel-id> -duel-capture <dir>      # one duel at a time, then read a few frames
```

Look at the frames for variety (press, clash flash, distance, bubbles) and report with the 합 table
(target vs got, 무력 gap) so the user can compare against the line.

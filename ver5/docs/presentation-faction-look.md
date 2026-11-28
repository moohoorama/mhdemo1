# 세력의 모습 — 한눈에 그 세력으로 읽히게

기준: `tools/doteditor/tools/civ3d.py`, `tools/doteditor/tools/story_heads.py`, `tools/doteditor/heroes/<병종>/grid/`,
`assets/data/game.json`(`Factions`), `internal/gui/look.go`·`scene.go`·`scene_shout.go`, `tools/title_reel/*.py`.

어떤 세력(황건적, 서량군, 남만, 산월 …)이 화면에서 그 세력으로 읽히려면 다섯 겹이 함께 맞아야 한다. 황건적을
예로 각 겹에서 무엇을 하는지, 어떤 도구를 쓰는지, 어디서 걸리는지를 적는다. 장면 형식은
[presentation-scene.md](presentation-scene.md), 도트 시트와 색 묶음은 [presentation-sprite.md](presentation-sprite.md)에 있다.

| 겹 | 황건적에서 한 것 | 고친 곳 |
|---|---|---|
| 1. 이름 있는 인물의 비전투 도트 | 장각·장량·장보: 풀어헤친 머리 위 노란 띠 한 줄, 등 뒤로 늘어진 머리, 누런 도포; 장각은 두 팔을 치켜드는 동작 | `civ3d.py`, `story_heads.py`, `characters*.json` `Look.Scene` |
| 2. 병사 시트의 표식 | 산적·무뢰배 머리띠를 한 줄 굵게, 뒤로 날리는 매듭 꼬리 두 가닥(192프레임씩 손으로) | `heroes/bandit`·`ruffian` 격자, `.spr`, 기준 그림 |
| 3. 세력 색 | 진영색 노랑; 장면 엑스트라도 병사 색으로(`PlainExtras`) | `game.json`, `look.go`, `scene.go` |
| 4. 무대 | 진채: 제단·향로·노란 깃발·화로, 문 난 목책·망루·군막 | `tools/title_reel/turban_camp.py` → `S_title_turban_camp` |
| 5. 목소리 | 구호를 머리 위 큰 글자로 외침(`Shout`), 외치기 전 군중이 조용해짐(`Chatter "-"`) | 장면 데이터 |

결과 화면: `~/orca/mhdemo1/result/20261128/turban-look/`(컷·전투·엑스트라·세 형제), 시안
https://claude.ai/artifact/8pddAmQH5uVYw4kNxTM6wP, 머리띠 전후 `~/orca/mhdemo1/result/20261128/turban-headband/`.

## 1. 이름 있는 인물의 비전투 도트

전투 시트가 없는 장수의 장면 도트는 3D 평복 모델을 변환해서 만든다(`civ3d.py`; 손으로 그린 초안은 쓰지 않는다).
`STORY`에 항목을 넣는다: `(이름, scene(key, n, 머리, **kit), 'light_infantry', 색 태그들)`.

- **머리**: `story_heads.py`의 2D 머리 격자(13×12, NW·SW만 렌더). 기존 머리를 바꿔 만든다. 황건 형제는
  `loose()`: `Hd.TOPKNOT`의 상투를 지우고 흐트러진 머리털 줄과 삐친 털을 올리고, 세 줄짜리 띠를 가운데 한 줄만
  남겼다(세 줄 그대로면 두건처럼 보였다). 수염은 `bearded(머리, 'long'|'goatee'|'bristle', 'h'|'g')`.
- **몸(kit)**: `cloth`·`robe`·`guard`·`tabard`(띠)·`arms`·`shoulder`·`cape`·`apron`·`fan`·`mane`. `mane`은 등 뒤로 늘어진
  머리털이다(머리 뒤 상자 + 어깨 위 두 가닥). 몸통 표면이 y .27이라 그보다 뒤(y .33)에 놓아야 보인다.
- **동작 칸**: 기본은 포권. 다른 자세는 `work(look, 이름, 포즈함수)`와 `ACTION[key] = (이름, 0)`. 장각의 `preach`는
  두 팔을 머리 위로 든다.
- **색 묶음과 `Look.Palette`**: 장면 도트에도 그 인물의 `Look.Palette`(전투 도트용)가 그대로 칠해진다.
  팔레트가 바꾸는 묶음(장각: themeA·hair, 장량: themeC·iron …)에 머리털이나 띠를 두면 엉뚱한 색이 된다.
  황건 형제는 띠·허리띠(금 키 k/l/m)를 안장 태그로, 머리털·늘어진 머리·신(가죽 키 f/g/h)을 말갈기 태그로 옮겼다
  (`KEY_TAGS[key] = TURBAN_KEYS`, 색은 `tags('saddle', …)`·`tags('horseMane', …)`). 말이 없는 시트라 이 묶음은 비어 있다.
- 만든 뒤: `python3 civ3d.py <key>` → `python3 render.py sheet <key>_noncombat --zoom 4`로 보고 →
  `python3 tools/spritebuild/build.py tools/doteditor/units/<key>_noncombat.yaml` → `Look.Scene` 연결 →
  `python3 tools/spritebuild/golden.py`(기준 그림; 전체 시트를 다시 뽑는다) → 게임 안 2배율 캡처(아래 확인).

## 2. 병사 시트의 표식

병사가 그 세력으로 읽히는 건 머리 표식(두건·띠·투구 장식)이 1배율에서 보일 때다. 진영색 태그(faction j/k/l)로
칠한 부분이 세력 색을 받는다.

- 표식은 프레임마다 손으로 고친다(규칙으로 모든 프레임을 바꾸지 않는다). 시트는 8방향 × 6동작 × 4프레임 = 192장이고
  E·NE·SE도 W·NW·SW의 반전이 아니라 따로 그려져 있다. 병종 하나에 에이전트 하나를 붙여 동작마다 8방향 시트를 렌더해
  보며 고치게 하면 15~20분 걸린다. 작업 전 격자를 백업하고 전후 비교 그림을 남긴다.
- 띠를 굵게: 기존 띠 바로 위 머리털 한 줄을 진영색으로. 매듭 꼬리: 뒤통수가 보이는 방향에 3~5px 두 가닥, k/l, 1px 외곽선.
  팔·칼이 뒤통수를 가리는 프레임은 꼬리를 줄이거나 뺀다.
- 투구를 쓴 병종(황건의 경보병)은 띠를 둘 자리가 없다. 군중은 표식이 있는 병종(산적·무뢰배)으로 맞춘다.
- 끝나면 `grid.py import <병종>`, `.spr` 빌드, `golden.py`, `go test ./internal/sprite/`.

## 3. 세력 색

`game.json` `Factions.<id>`: `Colors`(병사), `OfficerColors`(이름 있는 장수와 지휘관; 병사와 구분되게), `PlainExtras`.

- 전투: 병사는 `Colors`, 장수급(이름 있는 장수, 계급 officer·commander·boss)은 `OfficerColors`(`look.go` `officerColor`).
- 장면: 배우는 기본으로 `OfficerColors`를 받는다. 이름에 `#`이 붙은 엑스트라(`황건#3`)도 그래서, 황건 병사의 노란
  띠가 장면에선 흰빛(장수 색)으로 보였다. `PlainExtras: true`인 세력은 장면 엑스트라가 `Colors`를 받는다
  (`scene.go` `actorArt` → `look.go` `sceneArtIn`). 다른 세력 장면은 그대로다.
- 세력 색을 바꿀 땐 `internal/content/faction_colors_test.go`가 장수 색과 병사 색이 구분되는지 본다.

## 4. 무대

배경이 그 세력의 자리로 읽히게 소품을 둔다. 황건 진채의 그림은 이미 있다: `altar`, `incense`, `banner_yellow`,
`brazier`, `tent`, `tent_round`, `watchtower`, `antler_barrier`, `sfence_NN`(울타리 조각, 이음 번호 1 −u·2 −v·4 +u·8 +v:
v 방향으로 늘어선 줄은 `sfence_10`).

- **맵 오브젝트로 넣는다.** 장면 소품(`Prop` 단계)은 밤·해질녘 어둠 위에 그려져 혼자 밝게 뜨고, 화로의 빛도 나지 않는다.
  맵 사본을 만드는 스크립트에 오브젝트를 더한다(`tools/title_reel/turban_camp.py`, `guandu_camp.py`). 오브젝트 좌표는
  작은 타일(한 칸 = 2), 오브젝트가 선 칸은 걸을 수 없게 된다.
- 넣은 뒤 `go run ./cmd/_walkmap -fmt <맵>`, `assets/graphics/index.json` 등록(`extend_map.register`).
- 화면 칸은 64×32 px(1배율)다. 원·화면에 들어오는 범위를 셀 때 이 값으로 계산한다(32×16으로 잡으면 두 배 넓게 착각한다).

## 5. 목소리: 외침 글자

`Shout`(문구), `ShoutSize`(UI px, 기본 30): 말풍선 대신 머리 위에 굵은 고딕(Noto Sans KR 800) 노란 글자에 어두운 테두리를
크게 찍고, 툭 커졌다 줄며 떠오르고 1.6초에 사라진다(`scene_shout.go`). 타이틀 영상에서는 필름 색보다 먼저 그려 함께 바랜다.

- 문구는 장면 데이터의 문자열이다. 번역본은 문구만 바꾸면 된다(중국어판은 "蒼天已死 黃天當立"). 그때 글꼴은 한·중·일을
  다 담은 것(Noto Sans CJK)으로 바꾼다.
- 크기: 지도자 34, 그 곁 24, 군중 15 정도. 군중은 몇 명만 외치게 하고(전원이면 화면을 덮는다) 앞줄부터 시차를 둔다.
- 외치기 전에 군중의 `Chatter`를 `"-"`로 멈춘다. 멈출 때 떠 있던 웅성거림 말풍선도 같이 사라진다.

## 확인

- 장면 시험: 시험용 장면 파일을 `uidemo-vm -ui scenefile -stage <file.json>`로 찍는다(2배율). 엑스트라 색을 볼 땐 배우 이름에
  `#`을 붙인다(안 붙이면 엑스트라가 아니라 장수 색이 나온다).
- 장면 하나: `-ui scenenode -stage <노드>`. 타이틀 컷: `-ui titlereel -node <컷>`. 전투: `-ui battleview -stage B01`(스테이지 ID).
- 테스트: `go test ./internal/sprite/`(기준 그림), `./internal/content/`(엔진 코드에 데이터 이름이 섞였는지; 효과 이름이
  스킬 ID와 겹쳐 걸린 적이 있다), `./internal/gui/ -run 'TestTitleReel|TestScene'`.

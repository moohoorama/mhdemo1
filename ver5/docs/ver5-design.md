# ver5 설계

ver4를 정리해 만든 판이다. 목표는 셋이다.

1. **엔진과 데이터의 완전 분리.** Go 소스에는 유비·관우 같은 고유명사, 병종·아이템·스킬 이름, 스테이지 ID가 하나도 없다.
   게임 하나는 `assets/data/` 아래 데이터 묶음이다. 엔진은 그 묶음의 *속성*(`Lord`, `Effect` 등)만 읽는다.
2. **신형 스프라이트만 쓴다.** 도트 에디터 YAML(픽셀별 색 태그)을 빌드한 바이너리(`.spr`)를 게임이 읽고,
   진영색·옷색·말색을 캐릭터마다 태그 단위로 바꾼다. 옛 PNG 시트와 RGB 치환은 없다.
3. **특성 개편 반영.** `docs/traits-rework.md`의 규칙형 특성을 엔진 효과(effect)로 구현한다.

ver4에서 가져온 것: 규칙 코어 구조, 클레임 리플레이, Ebitengine 전장, 시나리오 장면.
가져오지 않은 것: ver2·ver3의 옛 스프라이트(`tools/spritetool`, `tools/unit3d`)와 그 빌드 체인, 도트 작업 노트(`heroes/<id>/`).

## 1. 디렉터리

```
ver5/
├─ cmd/srpg-{gui,cli,sim}
├─ internal/
│  ├─ content   데이터 묶음 로더·검증 (assets/data/*)
│  ├─ core      규칙 엔진 (고유명사 없음)
│  ├─ ai, session, storage, cli, sim, replay, skirmish
│  ├─ sprite    .spr 로더, 태그 팔레트 적용, 그림자, 시트 캐시
│  └─ gui       Ebitengine 화면
├─ assets/
│  ├─ data/        게임 데이터 (아래 2절)
│  ├─ sprites/     <id>.spr (tools/spritebuild 결과)
│  ├─ graphics/    타일셋·구조물·소품·맵 그림 데이터·초상 (정적 에셋)
│  └─ fonts/
├─ tools/
│  ├─ doteditor/   도트 에디터와 units/*.yaml 원본
│  ├─ spritebuild/ yaml → .spr 빌드·검증
│  ├─ build_assets.py  초상 축소·맵 그림 데이터
│  └─ smoke_cli.py, battlesimulator/
└─ docs/
```

## 2. 데이터 묶음 `assets/data/`

로더는 디렉터리의 JSON을 모아 `content.Data` 하나로 만든다. 해시는 모든 파일을 이름순으로 이은 바이트의 SHA-256이다.

| 파일 | 내용 |
|---|---|
| `game.json` | 제목, 버전, `Rules`(곡선·계수·경험치·승급 레벨 등 수치), `Campaign`(시작 노드·시작 명단·시작 편성·시작 소지품), `Factions`, `Tiles` 범례 |
| `classes.json` | 병종: 이동·사거리·공격 방식(`Attack`: `melee`/`ranged`)·장비 종류·지형군(`Terrain`)·스킬·특성·승급·기본 `Sprite` |
| `characters.json` | 인물 정의(장수와 병사 템플릿). `Look` 포함 |
| `equipment.json`, `items.json`, `skills.json`, `traits.json` | 각 정의 |
| `terrain.json` | 지형군별 타일 계수·이동비용 |
| `stages/<ID>.json` | 스테이지(맵, 배치, 보상, 일기토, 다음 노드) |
| `script.json` | 시나리오 노드 |
| `scenes.json` | 장면 연출(GUI 전용) |

### 2.1 Character (구 Officer)

```json
"liubei": {
  "Name": "유비", "Class": "light_infantry", "Level": 1, "Stats": [91, 76, 82, 75, 87, 99],
  "Lord": true, "Playable": true,
  "Traits": [...], "Learn": [...], "Equipment": [...],
  "Look": { "Sprite": "liubei", "Scene": "liubei_noncombat", "Portrait": "liubei", "Faction": "shu",
            "Palette": { "themeA": "#c04a3a" } }
}
```

- 키는 ID. 이름은 `Name`이고 코드는 ID나 이름 문자열을 비교하지 않는다.
- `Playable`: 편성·장비·부관 대상이 되는 아군 인물. `Lord`: 쓰러지면 패배하고 편성에서 뺄 수 없으며 부관이 될 수 없다.
- `Template: true`: **병사 템플릿.** 스테이지가 같은 템플릿을 여러 번 배치하면 엔진이 개체마다 임시 인물(`<템플릿>#<번호>`)을 만든다.
  임시 인물은 그 전투 동안만 있고 저장 명단에 오르지 않는다. 이름은 `Name` 뒤에 번호를 붙인다.
- `Look`: `Sprite`(전투 도트 키, 없으면 병종의 `Sprite`), `Scene`(비전투 도트, 없으면 `Sprite`), `Portrait`(없으면 이름표만),
  `Faction`(진영색 키, 스테이지가 `Faction`을 주면 그쪽이 우선), `Palette`(태그 그룹 → 색).

### 2.2 스프라이트와 팔레트

- 도트 키는 `assets/sprites/<키>.spr`의 파일 이름이다.
- 태그 그룹: `iron`, `skin`, `hair`, `faction`, `themeA`, `themeB`, `themeC`, `horseSkin`, `horseMane`, `saddle`(각 1·2·3번 = 밝음·중간·어두움),
  `outline`, `effect`. 태그가 없는 픽셀은 원본 팔레트 색을 그대로 쓴다. 투명도는 항상 유지한다.
- 최종 색 우선순위: 스프라이트 기본 → 진영(`faction` 그룹) → 캐릭터 `Look.Palette`. 같은 태그에 겹치면 뒤의 것이 이긴다.
- `Palette` 값은 `"#rrggbb"` 하나(중간색, 밝음·어두움은 자동 계산) 또는 세 개짜리 배열.
- 시트는 `(스프라이트, 최종 팔레트)`로 캐시한다. 그림자는 시트 알파에서 로드할 때 만든다.
- 데이터에 없는 장수·병사는 병종의 `Sprite`로 그린다. 신형 도트는 경보병, 경기병, 궁병, 문사(사마)와 유비·관우·장비뿐이고,
  나머지 장수는 가장 가까운 병종 도트에 진영색과 `Palette`로 구분한다(여포는 경기병 도트).
  비전투 도트(`*_noncombat`)가 없는 인물은 전투 도트의 `idle`·`walk`로 장면에 서고 `action` 자리에는 `idle`을 쓴다.

### 2.3 엔진이 아는 어휘

엔진은 영어 식별자만 안다. 화면에 보이는 이름은 모두 데이터의 `Name`이다.

- 공격 방식 `melee`/`ranged`/`self`/`any`, 스킬 종류 `physical`/`magic`/`status`/`buff`/`heal`, 모양 `single`/`cross`/`all`/`square`
- 상태 `poison burn confusion seal root weak speed range charge counter attack defense morale attack-down defense-down` 등
- 스킬·아이템·특성의 `Effect`(효과 종류)와 매개변수. 새 효과 종류가 필요하면 엔진에 추가하고, 이름·수치·조합은 데이터가 정한다.
- 타일 글자는 데이터의 `Tiles` 범례가 정의한다(`Water`, `Wall`, `RestHP`, `RestMP` 등 속성).

## 3. 엔진 변경 요약

| ver4 | ver5 |
|---|---|
| 유비·관우·장비·간옹 하드코딩(편성·패배·저장 검증) | `Lord`, `Playable`, `Campaign.StartRoster/StartDeployment` |
| 시작 노드 `oath`, 전투 후 노드 표 | `Campaign.Start`, 스테이지 `Next`, 준비 노드의 `Stage` |
| 아이템 이름 판별(`소군량`, `승급:`) | `Items[].Effect`(`hp`/`mp`/`promote`) |
| 스킬 이름 판별(`탈취`) | 스킬 `Effect: steal`과 `Item` |
| 특성 이름 판별 | `Traits[].Effects`(효과 키 → 매개변수) |
| 병종 이름·`Weapon == 활` | `Attack: ranged`, `Terrain` 지형군 |
| 곡선·계수 상수 | `Rules` |
| 병사 = 스테이지마다 복제한 Officer | `Template` + 임시 인물 |
| 진단(`Matchups`)이 유비·장각을 수정 | 데이터를 받아 임의의 두 인물을 쓰는 테스트 도우미 |
| GUI 하드코딩 표(`officers`, `heroArt`, `classArt`, `stageFaction`, `traitNotes`) | `Look`, 병종 `Sprite`, 스테이지 `Faction`, 특성 `Desc` |

`core` 이름 변경: `Officer`→`Character`, `Unit.Officer`→`Unit.Character`, JSON 키 `Officers`→`Characters`.

회귀 방지: `internal/` 소스(테스트 제외)에서 `assets/data`의 인물 ID·이름·병종 이름·아이템 이름·스킬 이름·특성 이름이
따옴표 문자열로 나오면 실패하는 테스트(`internal/content/purity_test.go`). 엔진 테스트는 `testdata/` 최소 데이터 묶음으로 돌리고,
실제 데이터로는 완주 스모크만 돌린다.

## 4. 단계

1. 뼈대 복사, 이 문서 (완료)
2. 스프라이트: `.spr` 포맷, 빌드, Go 로더·팔레트·그림자 (`docs/sprite-binary.md`)
3. 데이터 묶음 분리와 엔진 일반화(고유명사 제거, 병사 템플릿)
4. 특성 개편 (`docs/traits-rework.md`)
5. GUI: `Look` 연결, 비전투 도트 `action`, 장면 출연자 진영색, 초상 연결 (`docs/todo-connect-art.md`)
6. 문서 갱신, 순도 테스트, 전체 검증

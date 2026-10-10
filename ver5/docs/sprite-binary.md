# 스프라이트 바이너리 (`.spr`)

`tools/doteditor/units/*.yaml`(편집 원본) → `tools/spritebuild/build.py` → `assets/sprites/<id>.spr`(게임이 읽는 파일).
픽셀마다 팔레트 인덱스와 태그 번호를 함께 저장하므로, 게임이 태그 그룹별로 색을 바꿀 수 있다.
설계 의도는 `sprite-binary-intent.md`, 팔레트 우선순위는 `ver5-design.md` 2.2.

## 빌드

```
make -C tools/doteditor apply            # units/*.yaml 전부 (약 35초; YAML 파싱이 대부분)
python3 tools/spritebuild/build.py tools/doteditor/units/liubei.yaml
```

- PyYAML만 필요하다. `comparison_layers`는 버린다.
- `<id>_noncombat.yaml`(NW·SW만 있음)은 빌드 때 펼친다. NE·SE는 NW·SW를 좌우 반전한 행(픽셀과 태그 모두)이고,
  셀은 기준점 기준 좌우 대칭이 되게 넓히며, `walk`는 idle·왼발·idle·오른발 4칸이다. 런타임은 반전하지 않는다.
- 파일 이름은 `unit.id`이고 게임의 스프라이트 키가 된다.
- `tools/spritebuild/golden.py`는 `internal/sprite/testdata`의 비교용 PNG(기존 ver4 `apply.py`와 ver3 `shadows.py` 출력)를
  다시 만든다. Pillow가 필요하고 오래 걸리며, 테스트에는 필요 없다.

## 파일 구조

모든 정수는 little-endian.

```
magic    4B  "MHSP"
version  u16 = 1
payload  zlib(deflate) 전체
```

payload (`str` = u16 길이 + UTF-8):

| 필드 | 형식 |
|---|---|
| id, name | str, str |
| cell w, h / pivot x, y | u16 x4 |
| rows(방향) | u8 개수, str x개수 (순서가 시트의 행) |
| animations | u8 개수, 각각 `name str`, `first_column u16`, `frames u16`, `loop u8`, `n u8`, `ms u32 x n` |
| tags | u8 개수, 각각 `index u8`(1..개수), `group str`, `shade u8`(1..3, outline은 1), `color RGBA 4B` (기본색) |
| palette | u16 개수(1..256), RGBA 4B x개수 (0번은 투명) |
| columns | u16 |
| 프레임 | 행 x 열 순서(행 우선)로 셀마다 `팔레트 인덱스 평면 w*h B` + `태그 평면 w*h B` (태그 0 = 없음) |

프레임 평면의 위치는 셀 (행 r, 열 c)에서 `r*columns + c`번째다. 애니메이션 `a`의 i번째 프레임은 열 `first_column + i`.
YAML에 없는 셀은 전부 0(투명)이다.

## 검증 (`Load`)

오류에는 파일 이름이 붙는다(`path: ...`). magic·version 불일치, zlib 오류, 헤더·본문 잘림,
빈 시트, 기준점이 셀 밖, 애니메이션의 `ms` 개수·열 범위 오류, 태그 인덱스 중복·범위 밖,
픽셀 데이터 크기 불일치(잘림 또는 뒤에 남은 바이트), 팔레트 인덱스·태그 번호 범위 밖.

## 색 규칙

- 태그 > 0인 픽셀: 그 태그의 현재 색의 RGB + 팔레트 픽셀의 알파.
- 태그 0: 팔레트 색 그대로. 투명 팔레트 항목은 항상 투명.
- 현재 색 = 기본색(태그 표) 위에 `Palette[group][shade-1]`를 덮어쓴 값(알파 0인 칸은 "없음").
- 기본 색으로 렌더하면 기존 `apply.py`가 만든 PNG와 픽셀 단위로 같다(`internal/sprite` 테스트).

## Go API (`srpg/internal/sprite`)

- 순수 Go (`sheet.go`, `palette.go`, `render.go`): `Load`, `LoadDir`, `Decode`, `Sheet.Render`, `Sheet.Groups`, `Sheet.Shadow`,
  `Palette`, `Shades`, `ParseColors`, `Palette.Merge`, `Palette.Key`.
- Ebiten (`art.go`): `Art`, `Library`.
- 그림자는 `ver3/shadows.py`의 알고리즘(SLOPE·GROW·CORE·HALO·BLUR 동일)을 옮긴 것으로 검은색 + 알파이고,
  시트와 같은 행·열에 셀 크기만 다르다(`Shadow.Cell`, `Shadow.Pivot`). 팔레트와 무관하게 스프라이트당 한 번만 만든다.
- 기본색 그룹 기본값(모든 유닛 공통): `faction` 1·2·3 = `#5a8fd0 #3567a6 #1b3358`.
  옛 `team_keys`/`ramp`와의 대응: `ramp[3]`=faction1, `ramp[2]`=faction2, `ramp[0]`=faction3(`ramp[1]`은 쓰지 않는다).
  `saddle` 기본색도 같은 파랑이므로 안장은 진영색과 별개로 정해야 한다.

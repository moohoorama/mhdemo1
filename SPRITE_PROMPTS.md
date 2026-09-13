# 타일맵 유닛 이미지 생성 가이드

이 파일을 대화에서 `@SPRITE_PROMPTS.md`로 참조하고 병종, 방향, 동작만 지정한다. 아래 예시를 복사해서 사용해도 된다. 별도 플러그인이나 명령 설치 없이 사용하는 프로젝트용 프롬프트 가이드다.

## 빠른 호출

```text
@SPRITE_PROMPTS.md
풋맨의 전체 스프라이트 시트를 만들어줘.
SE와 NE 각각 대기·걷기·공격·피격·탈진 4프레임씩,
4열 × 10행의 투명 PNG 한 장으로 만들어줘.
```

```text
@SPRITE_PROMPTS.md
궁병을 기존 풋맨과 같은 화풍으로 추가해줘.
공격은 활 공격 규칙을 적용하고, 공통 40프레임 시트로 만들어줘.
초원과 황무지 배치 시안도 별도 PNG로 만들어줘.
```

```text
@SPRITE_PROMPTS.md
기병의 전체 40프레임 시트를 만들어줘. 장비는 검과 방패.
탈진은 말에서 내려 주저앉아 숨쉬는 모습으로, 말도 셀 안에 유지해줘.
```

전체 시트가 기본 산출물이다. 외형 검토만 필요하면 “단독 외형 시안 1장”을 명시한다. 이는 애니메이션 프레임 수 변경이 아니다.

## 참조 이미지와 우선순위

- `README.md`: 타일 크기, 투영, 외곽선, 렌더링 규칙의 기준.
- `map-export.png`: 지형의 색감, 픽셀 밀도, 오브젝트와 유닛 크기 비교 기준.
- `footman.png`: 원본 풋맨의 정체성. 빨간 깃털, 철갑옷, 둥근 방패, 휘어진 칼.
- `output/imagegen/footman-sprite-concept.png`: 승인된 풋맨 외형·화풍 시안.
- `output/imagegen/footman-terrain-preview.png`: 승인된 맵 배치 분위기 참고.

사용자의 현재 요청이 최우선이다. 다른 병종에는 풋맨의 장비를 그대로 강제하지 않는다. 기존 병종의 방향·동작을 바꿀 때는 승인된 해당 병종 이미지를 외형 기준으로 사용한다. 참조 이미지를 실제로 열어 확인하고 생성 도구에 전달한다.

현재 승인 이미지는 생성형 시안이다. 정확한 24×24 픽셀, 일정한 픽셀 격자, 프레임 정합성까지 검증된 런타임 에셋은 아니다.

## 요청할 때 바꿀 항목

| 항목 | 기본값 또는 예시 |
|---|---|
| 병종 | 풋맨, 궁수, 창병, 마법사, 기병 |
| 외형 기준 | 승인된 해당 병종 이미지. 새 병종이면 풋맨은 화풍만 참고 |
| 특징 | 무기, 방어구, 머리장식, 소속 색상, 탈것 |
| 방향 | 기본 전체 시트는 SE(우하단), NE(우상단). 아래 표의 **화면 기준** |
| 동작 | 대기(idle), 걷기(walk), 공격(attack), 피격(hit), 탈진(exhausted)의 5종만 |
| 프레임 수 | 모든 동작 각각 정확히 4프레임 |
| 반복 | 대기·걷기·탈진 반복, 공격·피격 비반복 |
| 목표 논리 크기 | 보병 대기 24×24, 점유 영역 약 18×22px를 출발점으로 사용 |
| 출력 종류 | 병종당 40프레임 통합 시트 PNG 1장. 단독 외형·맵 배치는 요청 시 추가 |
| 배치 지형 | 초원, 황무지, 바위 주변, 숲 가장자리 |
| 저장 위치 | `output/imagegen/` 아래 새 이름으로 저장 |

지정하지 않은 항목은 위 기본값과 병종 특성으로 판단한다. 이미 지정된 내용을 다시 확인받지 않는다. 큰 무기·탈것·공격 동작에는 캔버스를 넓혀도 되지만 캐릭터 본체의 축척은 유지한다.

## 방향 규약

N/E 등의 표기는 지도 좌표축이나 세계 방위가 아니라 화면에서 바라보는 방향이다. 카메라를 돌리는 대신 캐릭터만 회전시킨다.

| 코드 | 화면 방향 | 보이는 면 |
|---|---|---|
| S | 아래 ↓ | 정면, 투구 윗면도 보임 |
| SW | 왼쪽 아래 ↙ | 정면 3/4 |
| W | 왼쪽 ← | 왼쪽을 보는 측면 |
| NW | 왼쪽 위 ↖ | 후면 3/4 |
| N | 위 ↑ | 뒷모습 |
| NE | 오른쪽 위 ↗ | 후면 3/4 |
| E | 오른쪽 → | 오른쪽을 보는 측면 |
| SE | 오른쪽 아래 ↘ | 정면 3/4 |

아이소메트릭 지형을 내려다보는 카메라 각도는 모든 방향에서 고정한다. 무기와 방패가 장착된 **캐릭터의 해부학적 좌우**도 유지한다. 단순 좌우 반전으로 손을 바꾸지 않는다. 뒷방향에서 얼굴을 억지로 드러내지 않는다.

## 공통 화풍 프롬프트

아래 내용을 모든 생성 요청의 공통 조건으로 사용한다.

```text
첨부한 map-export.png에 어울리는 저해상도 아이소메트릭 전략 게임 유닛을 그린다.
승인된 유닛 참조의 화풍을 유지한다. 카메라는 위에서 내려다보는 고정 시점이다.
짧은 다리와 큰 머리의 약 2등신 캐릭터, 작은 크기에서도 읽히는 실루엣을 사용한다.
논리 해상도 기준 1px 검은 외곽선, 재질별 2~3단계 명암, 차분한 색상을 사용한다.
부드러운 그라데이션, 안티앨리어싱, 흐림, 미세한 장식, 사실적인 질감은 피한다.
보병 대기 자세는 24×24 논리 픽셀 캔버스에 약 18×22px 점유를 목표로 한다.
확대 시안에서도 정수배 최근접 확대처럼 각진 픽셀을 유지한다.
장비와 몸통 사이를 구분해 무기·방패·병종 특징이 작은 크기에서 읽히도록 한다.
단독 스프라이트와 시트는 실제 투명 배경으로 만들고 체크무늬를 그려 넣지 않는다.
그림자는 캐릭터에 포함하지 않는다. 글자, 라벨, 프레임 테두리, 워터마크를 넣지 않는다.
```

## 단독 스프라이트 요청 템플릿

```text
출력: 단독 스프라이트 시안 PNG 1장.
병종: {병종}
외형 기준 이미지: {경로}
참조 역할: {캐릭터 정체성 / 화풍만 참고}
필수 특징: {무기, 방어구, 색상, 머리장식}
방향: 화면 기준 {코드와 한글 방향}
동작: {동작과 자세}
목표 논리 캔버스: {가로×세로}px
공통 화풍 프롬프트를 적용한다.
전체 몸과 장비가 잘리지 않도록 하고 발밑 기준점은 양발 사이 지면에 둔다.
파일명: {병종}-{방향}-{동작}-concept-v01.png
```

## 방향별 외형 검토 시트 요청 템플릿

```text
승인된 {병종 이미지}의 동일 캐릭터를 {방향 목록} 방향으로 그린다.
애니메이션 전체 시트와 별개인 외형 검토용이다. 모든 칸에서 대기 1프레임의 기본 자세로 통일한다.
시트 배치: {열}열 × {행}행. 읽는 순서는 왼쪽부터 오른쪽, 위에서 아래.
각 칸의 논리 크기는 {가로×세로}px, 배율과 발밑 기준점은 모든 칸에서 동일하다.
캐릭터 키, 장비, 팔레트, 조명, 카메라 각도를 유지하고 몸의 방향만 바꾼다.
장착된 손을 바꾸지 않고 방향에 따른 무기·몸·방패의 가림 관계를 반영한다.
공통 화풍 프롬프트를 적용한다. 시트 안에는 방향명을 쓰지 않는다.
방향 순서는 별도 MD에 기록한다.
```

## 공통 40프레임 시트 규격

병종마다 투명 PNG 한 장을 사용한다. **4열 × 10행 = 40셀**이며, 열은 프레임 1–4, 행은 아래 순서로 고정한다. PNG 안에는 번호·이름·격자선을 그리지 않는다.

| 행(0부터) | 방향 | 동작 | 열 0–3 |
|---|---|---|---|
| 0 | SE 우하단 | idle 대기 | 프레임 1–4 |
| 1 | SE 우하단 | walk 걷기 | 프레임 1–4 |
| 2 | SE 우하단 | attack 공격 | 프레임 1–4 |
| 3 | SE 우하단 | hit 피격 | 프레임 1–4 |
| 4 | SE 우하단 | exhausted 탈진 | 프레임 1–4 |
| 5 | NE 우상단 | idle 대기 | 프레임 1–4 |
| 6 | NE 우상단 | walk 걷기 | 프레임 1–4 |
| 7 | NE 우상단 | attack 공격 | 프레임 1–4 |
| 8 | NE 우상단 | hit 피격 | 프레임 1–4 |
| 9 | NE 우상단 | exhausted 탈진 | 프레임 1–4 |

### 셀 크기와 기준점

- 공통 게임용 셀은 **48×48 논리 픽셀**, 전체 PNG는 **192×480px**를 기준으로 한다. 보병 본체는 기존 약 18×22px 축척을 유지한다. 셀을 꽉 채우려고 캐릭터를 확대하지 않는다.
- 셀 내부 공통 지면 피벗은 왼쪽 위에서 **(24, 36)**이다. 걷기의 상하 운동과 탈진 자세는 이 기준 주변에서 그리며, 프레임마다 이미지를 중앙 정렬하거나 잘라내지 않는다.
- 여백은 무기 궤적, 뒤로 젖힘, 탈것을 수용하기 위한 공간이다. 모든 장비·효과는 셀 안에 들어와야 하고 이웃 셀로 넘어가면 안 된다.
- 기병 등에서 공간이 부족하면 본체만 임의 축소하지 않는다. 공통 규격을 확대하거나 동일 엔진이 읽는 병종별 `cell_width`, `cell_height`, `pivot` 메타데이터로 처리한다.
- 생성형 확대 시안은 정확한 게임용 규격과 구분한다. 정수 배율 k의 검증된 확대 시트라면 셀은 48k×48k, 전체는 192k×480k이며 피벗도 k배다. 생성 결과가 이 격자를 따르는지 실제 확인하기 전에는 런타임용이라고 하지 않는다.
- 파일명 예: `footman-sheet-v01.png`, `archer-sheet-v01.png`. 별도 메타데이터에는 실제 셀 크기, 피벗, 방향/동작 순서, 프레임 시간, 반복 여부를 기록한다. 프레임별 PNG는 기본 산출물이 아니다.

공통 렌더러는 다음 규칙으로 원하는 사각형 하나를 잘라 그릴 수 있다. 아래는 구현 규약이며 이번 문서 수정에서 게임 엔진을 구현한 것은 아니다.

```text
# All indices are zero-based.
# directionIndex: SE=0, NE=1
# actionIndex: idle=0, walk=1, attack=2, hit=3, exhausted=4
row = directionIndex * 5 + actionIndex
sourceX = frameIndex * cellWidth      # frameIndex = 0..3
sourceY = row * cellHeight
sourceRect = (sourceX, sourceY, cellWidth, cellHeight)
drawOrigin = projectedGroundPosition - pivot
```

추가 병종도 이 규칙을 따르면 이미지와 메타데이터만 바꾸어 같은 렌더러를 사용할 수 있다. 다른 방향을 추가할 때는 방향당 5행을 덧붙이고 방향 목록을 메타데이터에 기록한다. 기본 두 방향으로 모든 방향이 표현되는 것은 아니며, 비대칭 장비를 단순 반전하면 장착 손이 바뀌므로 자동 반전은 기본 규칙에 넣지 않는다.

## 동작 규약 — 모두 정확히 4프레임

프레임 번호는 아래 설명에서 1부터 센다. 기본 자세는 해당 방향의 대기 1프레임이다.

| 동작 | 프레임 1 | 프레임 2 | 프레임 3 | 프레임 4 |
|---|---|---|---|---|
| 대기 | 기본 | 들숨, 가볍게 몸 상승 | 기본(1과 동일) | 날숨, 가볍게 몸 하강 |
| 걷기 | 왼발 반쯤 전진, 몸 상승 | 왼발 완전히 전진·접지, 몸 하강 | 오른발 반쯤 전진, 몸 상승 | 오른발 완전히 전진·접지, 몸 하강 |
| 공격: 근접 | 무기를 뒤로 당김 | 힘차게 휘두름 + 무기 궤적 | 무기가 타격 지점에 도달 | 관성으로 타격 지점을 조금 지나 멈춤 |
| 공격: 활 | 화살을 건 채 시위를 당김 | 발사 + 화살 출발 궤적, 시위가 발사 방향으로 살짝 넘어감 | 시위가 중앙으로 돌아오며 양옆에 작은 곡선 잔진동 효과 | 시위 정지, 출발 궤적·진동 효과 없음 |
| 피격 | 기본 | 기본(1과 동일) | 과장되게 뒤로 젖힘 + 반투명 피격 효과 | 3과 정확히 같은 몸·장비 자세, 피격 효과만 제거 |
| 탈진 | 주저앉아 헉헉 들숨 | 같은 위치에서 헉헉 날숨 | 1과 동일 | 2와 동일 |

공격의 근접/활은 같은 `attack` 동작의 병종별 표현이며 별도 동작 행을 추가하지 않는다. 보병·기병 등은 장착 무기에 맞게 팔을 움직인다. 검·도끼는 호를 그리며 휘두르고, 창은 찌르는 궤적을 사용한다. 다른 무기도 당김 → 빠른 실행 → 명중 → 여운이라는 4단계를 따른다.

걷기는 제자리 사이클로 표현한다. 해부학적 왼발·오른발을 기준으로 하며 화면 방향에 따라 뒤바뀌지 않는다. 기병의 이동은 말의 보행과 기수의 상하 운동으로 해당 4박을 표현한다. 탈진은 사망이 아니라 주저앉아 숨을 고르는 상태다. 기병은 하마한 기수와 말의 배치를 셀 안에서 유지한다.

### 공격과 피격 동기화

- 공격자와 피격자는 같은 시작 시각과 같은 프레임별 시간을 사용해 1→2→3→4를 함께 재생한다.
- **모든 무기의 명중 순간은 프레임 3**이다. 엔진의 0 기반 인덱스로는 2다. 피해 적용과 피격 효과도 여기에 맞춘다.
- 활은 프레임 2에 발사하여 프레임 3에 매우 빠르게 도착하는 연출이다. 거리에 따른 비행시간은 생략한다. 별도 화살 이동을 구현하더라도 프레임 3 도착에 맞춘다.
- 시위는 활에 걸린 줄이다. 오른쪽으로 발사한다면 프레임 1에서 왼쪽으로 당겨지고, 2에서 오른쪽으로 살짝 넘어가며, 3에서 중앙으로 돌아오되 `()`처럼 작은 곡선으로 잔진동을 표시한다. 다른 방향도 발사축을 기준으로 회전한다. 과장은 작은 픽셀 변위로 제한한다.
- 피격 1·2의 기본 자세는 공격 준비 시간과 맞추기 위한 의도적인 유지다. 피격을 명중 뒤에 1부터 새로 시작하면 반응이 늦어지므로 그렇게 재생하지 않는다.
- 프레임 시간은 개수와 별도로 지정할 수 있지만 공격/피격 쌍은 동일한 시간표를 공유한다. 명중 이벤트는 3 진입 시 한 번만 발생한다.
- 공격 4는 대기 복귀가 아니라 여운 자세다. 피격 4도 젖힌 자세를 유지한다. 이후 상태 전환은 재생 측에서 처리하고 임의의 5번째 복귀 프레임을 추가하지 않는다.

### 효과와 중복 프레임

무기 궤적, 화살 출발 효과, 시위 진동, 반투명 피격 효과는 이번 규격의 필수 표현이다. 기본 통합 PNG의 해당 셀에 포함한다. 캐릭터 본체를 통째로 반투명하게 만들지 않고 피격 효과에만 반투명 알파를 사용한다. 효과도 흐림 없이 픽셀 형태를 유지한다.

대기 1=3, 피격 1=2, 탈진 1=3 및 2=4의 중복은 의도적이다. 피격 3과 4는 효과를 제외한 본체가 동일하다. 생성 도구의 근사 재현을 픽셀 단위 동일성으로 간주하지 말고 게임용 정리 단계에서 검증한다.

## 전체 시트 생성 템플릿 (영문)

한국어 호출 내용을 아래 변수에 넣고 공통 화풍 및 승인된 참조 이미지와 함께 전달한다.

```text
Create one transparent PNG sprite sheet for {unit_class}, using {approved_reference}
as the character identity reference and map-export.png as the terrain style reference.
Use exactly 4 columns and 10 rows, 40 equally sized cells, with no gutters or labels.
Rows 0-4 face screen lower-right (SE); rows 5-9 face screen upper-right (NE).
Within each direction, rows are IDLE, WALK, ATTACK, HIT, EXHAUSTED, in that order.
Columns are frames 1, 2, 3, 4 in chronological order.

Target logical cell size: 48x48 pixels. Target logical sheet size: 192x480 pixels.
Keep the ground pivot at (24,36) in every cell. Infantry body size stays approximately
18x22 logical pixels; do not enlarge it to fill the cell. Keep weapons, mounts and
all effects inside their cells. An enlarged concept must preserve this grid and scale.
Use the shared pixel-art style specification: fixed elevated isometric camera,
black one-logical-pixel outlines, muted colors, 2-3 shades per material, no smoothing.
Keep identity, equipment handedness, body scale, palette and lighting consistent.
NE is a rear three-quarter view, not a mirrored SE pose. Rotate the character only.

IDLE: (1) neutral, (2) inhale with a slight rise, (3) identical to 1,
(4) exhale with a slight lowering. Feet remain planted.
WALK: (1) anatomical left foot halfway forward, body rising,
(2) left foot fully forward and contacting ground, body lowering,
(3) right foot halfway forward, body rising,
(4) right foot fully forward and contacting ground, body lowering.
Animate in place. For cavalry, adapt the four beats to the mount and rider.

ATTACK uses the unit's equipped weapon: {weapon}.
For melee: (1) wind-up, (2) forceful swing with a visible pixel weapon trail,
(3) weapon reaches the impact point, (4) weapon stops slightly beyond it due to momentum.
Use a thrust for a spear instead of an inappropriate sword swing.
For bows: (1) draw the string with an arrow nocked,
(2) release with a short arrow departure streak and slight forward string overshoot,
(3) string returns toward rest with small curved vibration marks on both sides,
(4) string rests with no vibration or departure effect.
Use only the appropriate attack variant in the ATTACK row.

HIT: (1) neutral, (2) identical neutral, (3) exaggerated backward recoil with a
semi-transparent pixel impact effect, (4) exactly the same recoil pose without that effect.
Keep the character opaque; only the impact effect is semi-transparent.
EXHAUSTED: seated and slumped, visibly panting; (1) inhale, (2) exhale,
(3) identical to 1, (4) identical to 2. This is exhaustion, not death.
For cavalry, show the dismounted rider seated with the mount remaining within the cell.

Attack and hit share the same four-beat timeline. Impact always occurs on frame 3.
Arrows launch on frame 2 and reach the target on frame 3 regardless of distance.
Idle, walk and exhausted loop. Attack and hit play once. Do not add recovery frames.
Preserve intentional duplicate frames. Include required effects in their cells.
Use actual transparent background, not a painted checkerboard. No cast shadows,
text, borders, grid lines, frame numbers or watermarks.
```

## 지형 배치 시안 요청 템플릿

```text
편집 대상: map-export.png
삽입할 유닛: {승인된 유닛 이미지}
원본 맵의 지형 배치, 해안선, 나무, 바위, 팔레트, 바깥 배경을 유지한다.
동일한 유닛을 초원, 황무지, 바위 옆 빈 땅, 숲 가장자리의 걸을 수 있는 땅에 배치한다.
배치 수: {수}. 방향: {방향 또는 위치별 방향}.
물이나 바위 위에 세우지 않는다. 숲에서는 앞뒤 관계에 맞게 일부가 가려질 수 있다.
원본 맵의 실제 4배 PNG에서 보병 높이 약 88px를 출발점으로 삼는다.
출력 크기가 바뀌면 유닛도 같은 비율로 조정하고 지형과 유닛의 픽셀 밀도를 맞춘다.
작고 납작한 접지 그림자를 추가한다. UI와 글자는 넣지 않는다.
전체 맵 배치 시안 PNG를 별도 저장한다. 원본 map-export.png는 덮어쓰지 않는다.
이 이미지는 생성형 배치 시안이며 게임 엔진의 실제 렌더 결과라고 표시하지 않는다.
```

## 생성 및 검토 규칙

1. 이미지 생성 도구로 작업하고, 요청된 각 산출물을 별도 파일로 저장한다. 기존 파일은 덮어쓰지 않고 버전을 올린다.
2. 저장한 파일 경로, 사용한 프롬프트, 참조 역할, 방향·프레임 순서를 함께 MD로 남긴다.
3. 생성 후 외형 일관성, 방향, 손과 장비, 프레임 수, 잘림, 배경 투명도를 확인한다. 실제 검증하지 않은 속성은 보장하지 않는다.
4. 시안과 게임용 에셋을 구분한다. 이미지 생성만으로 정확한 픽셀 격자·캔버스·알파·프레임 정렬이 보장되지는 않는다.
5. 사용자가 게임용 에셋 제작까지 요청하면 실제 크기, 시트 셀 크기, 발밑 피벗, 팔레트, 알파, 프레임 정합성을 별도로 정리하고 검증한다. 프로젝트의 자동 외곽선을 적용할 경우 이미 그린 외곽선이 중복되지 않게 한다.
6. 이번 요청이 PNG 시안만이면 게임 코드나 맵 데이터는 수정하지 않는다.

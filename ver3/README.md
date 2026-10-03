# ver3

정리된 에셋(캐릭터별 이미지 1장, 맵 타일셋 1장, 이를 설명하는 YAML)과 그 에셋으로 돌아가는 pygame 전투 데모.

```sh
python3 ver3/demo.py                       # 데모 (pygame)
python3 ver3/demo.py --wave 8 --fps 30     # 증원 8기씩, 30 FPS로 시작
python3 ver3/demo.py --shots /tmp/demo 5 20  # 화면 없이 5초·20초 장면 저장
python3 ver3/tools/build_assets.py         # 에셋 다시 묶기 (Pillow, PyYAML, Go)
go run . -map ver3/asset/maps/demo-map.json  # 데모 맵을 기존 에디터로 편집
```

화면 아래 막대에서 **증원 규모**(턴마다 1·2·4·8·16기)와 **FPS**(15·30·60·120, 기본 60)를 고른다.
움직임은 실제 경과 시간(ms)으로 진행하므로 FPS는 부드러움만 바꾼다. 막대에 실제 FPS와 진영별 병력이 보인다.

조작: Space 일시정지 · F 빨리 감기 · R 다시 시작 · 1–5 증원 규모 · `[` `]` FPS · Esc 종료

## 구조

```
ver3/
├─ asset/
│  ├─ assets.yaml          모든 이미지와 그 안의 스프라이트·타일 정보, 진영색 표, 맵 목록
│  ├─ units/<병종>.png      캐릭터별 1장: 행 = 8방향, 열 = 5동작 × 4프레임
│  ├─ map/tileset.png       맵 타일 1장: 지형 타일(16×8) + 장식(바위·잔디·나무, 40×48)
│  └─ maps/
│     ├─ demo-map.json      맵 원본 (에디터 형식 v5)
│     └─ demo-map.yaml      그리기용 데이터: 셀 격자, 지형 레이어, 장식 배치
├─ assets.py               로더: 프레임 자르기, 진영색 치환(캐시)
├─ demo.py                 전투 데모
└─ tools/
   ├─ build_assets.py      ver2 병종 시트·지형 아틀라스·맵을 위 구조로 묶고 YAML 생성
   └─ exportmap/main.go    에디터 지형 규칙(자동 경계·장식 배치)으로 맵을 그리기용 데이터로 내보냄
```

### 캐릭터 이미지 읽는 법 (`assets.yaml` → `units`)

- 행: `rows` 순서의 바라보는 방향(화면 기준, SW = 좌하단).
- 열: `animations.<동작>.first_column + 프레임`. `ms`는 프레임별 시간, `loop`는 반복 여부.
- 프레임 영역 = `(열 × cell[0], 행 × cell[1], cell[0], cell[1])`. `pivot`(발밑)을 유닛 위치에 맞춰 그린다.
- 진영색 부위는 `factions.team_keys` 4색으로 그려져 있다. 이 4색과 정확히 같은 픽셀만 진영의 `ramp`로 바꾼다.

### 맵 읽는 법 (`maps/*.yaml`)

- 셀 `(u, v)`는 32×16 마름모, 화면 좌표 `((u − v) × 16, (u + v) × 8)`. `cells`: `.` 초원 · `:` 황무지 · `~` 물 · `R` 바위 · `T` 숲. 이동 가능 = 초원·황무지·바위.
- `ground`: `[x, y, [타일 id …]]`를 순서대로 겹쳐 그린다. id 0 = 현재 물 프레임(`animations.water`).
- `decorations`: `[x, y, 애니메이션, 위상]`, 유닛과 함께 발밑 y로 정렬해 그린다.

## 데모 규칙

- 촉(노랑)과 위(청)가 번갈아 턴을 가진다. 턴 시작마다 자기 쪽 맵 끝(서쪽 / 동쪽 가장자리)에서 증원이 나온다.
  가장자리가 차면 안쪽 줄에서 나온다. 진영당 최대 인원은 증원 규모 × 3(최소 6), 영웅은 한 명씩.
- 같은 진영 유닛은 0.25초 간격으로 출발해 동시에 움직인다. 각자 가장 가까운 적에게 닿는 칸까지 길을 찾아
  이동(보병 3칸, 기병·영웅 5칸)하고, 사거리(근접 1, 궁병 3) 안이면 공격한다.
- 공격 3프레임(타격 순간)에 피해가 들어간다. 맞은 유닛은 피격 모션과 함께 좌우로 흔들린다. 왼쪽에서 맞으면 오른쪽으로 먼저 밀린다.
- 체력이 절반 이하면 대기 대신 탈진 모션. 0이 되면 깜빡이다 사라진다.
- 병종 능력치와 진영 구성은 `demo.py`의 `STATS`, `SIDES`.

## 원본과 갱신

에셋 원본은 저장소의 기존 위치에 있고 `build_assets.py`가 읽기만 한다.

- 병종: `tools/spritetool/assets/ver2-units/` (생성: `python3 tools/unit3d/build.py --full`, 기준: `tools/spritetool/assets/UNIT_ART_GUIDE.md`)
- 지형·장식: `assets/terrain.png`, `assets/objects.png`, `assets/catalog.json` (`make assets`)

병종이나 지형을 바꾸면 `build_assets.py`를 다시 실행한다.

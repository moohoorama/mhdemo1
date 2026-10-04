# ver2 병종 세트

[병종 기준](../UNIT_ART_GUIDE.md)과 [공통 기준](../SPRITE_ART_GUIDE.md)을 따른다. 구현은 [`tools/unit3d/`](../../../unit3d/DESIGN.md).

8방향 × 6동작(대기·걷기·공격·피격·탈진·막기) × 4프레임. 병종 폴더마다 `{N,NE,E,SE,S,SW,W,NW}-pixel.png`(4열 × 5행)과
`frames.json`(셀·기준점·프레임 좌표·재생 시간·팔레트). 진영색은 `factions.json`.

```sh
python3 tools/unit3d/build.py --full                 # 전체 재생성 (약 10분)
python3 tools/unit3d/build.py --full --unit guanyu   # 한 병종만
python3 tools/unit3d/build.py                        # 8방향 대기 검토 시트만 (output/ver2-units.png)
python3 tools/unit3d/attack_candidates.py            # 공격 후보 비교 (output/ver2-attack-candidates.*)
make preview-knights                                 # http://127.0.0.1:8765/ver2-units-preview.html
python3 ver3/tools/build_assets.py                   # ver3/asset 재패키징 (캐릭터별 1장 + assets.yaml)
```

## 병종

| 폴더 | 병종 | 외형 | 공격 (`attacks.SELECTED`) | 셀 · 기준점 |
|---|---|---|---|---|
| `infantry` | 경보병 | 철투구, 칼, 원형 방패 | 횡베기 `sweep` | 56×46 · (28,38) |
| `bandit` | 황건 적병 | 상투 + 황건, 삼베 옷, 칼, 원형 방패 | 내려베기 `overhead` | 58×54 · (29,41) |
| `spearman` | 창병 | 원뿔 투구, 창 | 찌르기 `thrust` | 104×50 · (52,36) |
| `archer` | 궁병 | 두건, 활·화살통 | 강궁 `power` | 94×48 · (47,39) |
| `cavalry` | 경기병 | 철투구, 창, 밤색 말 | 돌진 찌르기 `charge` | 102×53 · (51,50) |
| `guanyu` | 관우 | 녹건·녹포, 붉은 얼굴, 긴 수염, 청룡언월도, 적토마 | 언월도 횡베기 `glaive_sweep` | 70×60 · (35,57) |
| `zhangfei` | 장비 | 검은 투구, 진영색 옷·붉은 띠, 검은 철갑, 거뭇한 얼굴, 덥수룩한 갈색 수염, 장팔사모, 흑마 | 휘두르기 `swing` | 68×53 · (34,50) |
| `strategist` | 사마(문관) | 검은 관모, 진영색 도포, 깃털부채 | 횡베기 `sweep` | `frames.json` |
| `liubei` | 유비 | 금빛 관모, 작은 귀, 진영색 도포·금빛 어깨, 쌍검 | 횡베기 `sweep` | `frames.json` |
| `jianyong` | 간옹 | 흰 윤건, 진영색 옷, 활 | 강궁 `power` | `frames.json` |
| `zhangjiao` | 장각 | 황건 띠, 긴 수염, 삼베 도포, 금고리 지팡이 | 횡베기 `sweep` | `frames.json` |
| `huaxiong` | 화웅 | 검은 투구·갑옷, 언월도, 흑마 | 언월도 횡베기 `glaive_sweep` | `frames.json` |
| `lubu` | 여포 | 꿩깃 금관, 붉은 갑옷, 방천화극, 적토마 | 휘두르기 `swing` | `frames.json` |
| `zhangshiping` | 장세평(상단) | 검은 관모, 진영색 도포, 큰 등짐, 무기 없음(비전투 우군) | — (공격 행은 쓰지 않음) | `frames.json` |
| `sushuang` | 소쌍(상단) | 맨상투, 장세평과 같은 몸 | — | `frames.json` |
| `chengyuanzhi` | 정원지 | 노란 두건, 긴 수염, 삼베 도포, 지팡이 | 횡베기 `sweep` | `frames.json` |
| `dengmao` | 등무 | 노란 두건, 가죽 갑옷, 칼·방패 | 횡베기 `sweep` | `frames.json` |

ver4에서 추가한 6종(사마~여포)과 B01의 장세평·소쌍·정원지·등무는 `tools/unit3d/looks.py`의 후보 중 사용자가 고른 외형이다(`SELECTED`, 나머지는 백업).
후보 비교: `python3 tools/unit3d/look_candidates.py [--unit KEY]` → `output/ver4-look-candidates*.png`.

셀은 병종마다 전 프레임의 합집합으로 자동으로 정해지므로 다시 생성하면 바뀔 수 있다. 값은 `frames.json`이 기준이다.
재생 시간(ms): 대기 180, 걷기 120, 공격 180·280·240·200, 피격 120, 탈진 220, 막기 100·140·260·200.

## 막기

공격이 빗나가면 나오는 동작이다. 후보 4안(막아서기·쳐내기·몸 비켜 피하기·뒤로 물러서기) 중 **막아서기**를 골랐다
(`tools/unit3d/blocks.py`의 `SELECTED`, 나머지는 백업). 칼 병종은 칼과 방패, 창병은 창대, 궁병은 굵게 그린 활,
기마는 무기를 가로로 들어 막는다. 후보 비교: `python3 tools/unit3d/block_candidates.py` → `output/ver4-block-candidates.png`.

## 장면용 평복 (`../ver4-civilians/`)

시나리오 장면에서 쓰는 도보·무기 없는 차림. 같은 형식이고 동작만 다르다: 대기·걷기·말하기·포권·건배·놀람·끄덕임.
**방향은 좌상(NW)·좌하(SW) 2개만 만든다**(2026-10-04 사용자 결정). 우상(NE)·우하(SE)는 게임에서 좌우반전하고, 장면에서는 정방향(N·E·S·W)을 쓰지 않는다.
반전하면 빛 방향도 뒤집히고 비대칭 장식이 반대편으로 가므로, 평복은 좌우 대칭에 가깝게 디자인한다.

| 키 | 장수 | 외형 |
|---|---|---|
| `guanyu` | 관우 | 녹건 · 녹색 도포 (후보 2) |
| `zhangfei` | 장비 | 맨상투 · 진영색 도포 · 붉은 띠 · 덥수룩한 갈색 수염 (후보 2를 밝게 고침) |
| `liubei` | 유비 | 금빛 관모 · 진영색 도포 (후보 2) |
| `jianyong` | 간옹 | 흰 윤건 · 진영색 옷 (전투 외형에서 활만 뺌) |
| `sunqian` | 손건 | 검은 관모 · 진영색 도포 (사마 외형에서 부채만 뺌) |
| `zhangshiping` | 장세평 | 검은 관모 · 녹색 도포 (후보 4) |
| `sushuang` | 소쌍 | 맨상투 · 녹색 옷 · 큰 등짐 (후보 4) |
| `yuanshao` | 원소 | 금빛 관모 · 흰 도포 · 금빛 어깨 (후보 1) |
| `yuanshu` | 원술 | 금관 · 금빛 도포 · 뻣뻣한 수염 (후보 2) |
| `gongsunzan` | 공손찬 | 철 투구 · 흰 깃털 · 은빛 옷 · 흰 갑옷 (후보 4) |
| `dongzhuo` | 동탁 | 검은 관모 · 덥수룩한 수염 · 검은 도포 · 금빛 어깨 · 비대 (후보 1) |
| `liru` | 이유 | 검은 관모 · 긴 수염 · 회색 도포 (후보 4) |
| `caocao` | 조조 | 검은 관모 · 염소수염 · 흰 속옷 · 밝은 푸른 전포 · 금빛 띠 (3차 후보 4, `caocao3`; 1·2차 검정·붉은 후보는 백업) |

촉 이외 인물의 평복은 진영 키 대신 고정색을 쓴다(장면은 모든 인물을 한 진영색으로 그린다).
조조의 푸른색은 진영 키가 아니라 기사 v6 청색 램프(`7 8 9 a`, 밝은 전포는 `8 9 a o`)다.

```sh
python3 tools/unit3d/civilian_candidates.py   # 외형 후보 → output/ver4-civilian-candidates.png
python3 tools/unit3d/civilian_build.py        # 전체 → ver4-civilians/, 검토 output/ver4-civilians.png
```

## 공격 후보 (백업)

선택되지 않은 후보는 `tools/unit3d/attacks.py`에 정의를 남겼다. `SELECTED`만 바꾸면 다시 적용된다.

- 칼 `SWORD`: 내려베기 `overhead`, 횡베기 `sweep`, 올려베기 `rising`, 찌르기 `thrust`
- 창(기마) `SPEAR`: 돌진 찌르기 `charge`, 내려찍기 `downstab`, 휘두르기 `swing`, 도약 찌르기 `leap`, 언월도 횡베기 `glaive_sweep`
- 활 `BOW`: 강궁 `power`, 낮게 쏘기 `low`, 하늘로 쏘기 `sky`, 속사 `rapid`

## 진영색

진영색 부위는 키 램프 `W X Y Z`(`#1b3358 #264d80 #3567a6 #5a8fd0`)로 그려져 있다. `factions.json`의
`team_keys`와 정확히 같은 픽셀을 각 진영의 `ramp`로 바꾼다. 진영은 `tools/unit3d/factions.py`에서 색조·채도·밝기로 정의한다.

위(청, 키 그대로) · 촉(금빛 노랑) · 오(빨강) · 황건적(주황빛 황색) · 동탁·여포(곤색) · 공손찬(흰색) · 원소·원술(레몬빛 노랑) · 이민족(보라)

## 이력

- 초기 2D 후보(기사 v6 몸 + 머리 교체)는 [`../ver2-unit-candidates/`](../ver2-unit-candidates/README.md)에 보관.
- 기사 프레임 머리 교체 방식은 몸이 잘리고 공격·피격에서 머리를 추적할 수 없어 3D 렌더로 바꿨다.
- 피격은 넉백·베임·웅크림·흔들림 4안 중 베임의 자세만 채택했다(색 효과는 런타임).

---
name: towngen
description: Makes or remakes a town, castle or village scene map (성·마을 장면맵) of the ver5 SRPG by rule — keeps what the scenario needs (walls, gates, the explore areas' people, exits and signs, the story scenes' places and walks), wipes the old ground and buildings inside, lays straight avenues from the gates, splits blocks into zones (관아·상업·부유층·민가·밭) and fills them with the regional pixel buildings (화북·중원·회남·강남·파촉, facing either road side), compound walls and fences as wall-line tiles, props and trees, then checks that nothing covers a kept place and everything is still walkable, renders the map and shows old and new on a page. Uses ver5/tools/towngen (towngen.py, batch.sh). Use for "성 맵 새로", "마을 맵 만들어", "장면맵 재설계", "성안 다시 깔아", "지역 건물로 바꿔", "<도시> 맵 이상해", "길이 성문이랑 어긋나", "건물이 성벽에 붙어", or for a new town or castle scene map.
---

# 성·마을 장면맵 생성 (towngen)

사용자가 정한 원칙(2026-11-28)으로 성·마을 장면 맵을 다시 짠다. 옛 건물을 제자리에서 바꿔 끼우는 '치환'은 거절됐다(담장 크기·길·바닥이 안 맞음). 지킬 것만 남기고 성 안을 새로 깐다.

## 원칙 (사용자)

1. 맵 하나씩. 탐험 맵만이 아니라 스토리 장면이 쓰는 성·마을 맵도 (실내·진영·벌판은 제외).
2. 유비가 방문할 건물(관청·주점·도구점·대장간·저택·황궁·조조의 저택 …)이 그 맵에 있었는지 역할과 이름을 기록한다 (`work/root/roles/<맵>.json`).
3. 성·울타리가 있으면 건물은 반드시 그 안. 모자라면 성을 키운다(앞쪽으로, 사람이 걷는 범위 Size도 함께).
4. 성·마을 바닥은 새로. 옛 바닥 글자는 지운다.
5. 성문에서 직선 대로, 그와 교차하는 대로. 황성이면 황궁에서 대로.
6. 황궁 둘레에는 건물을 두지 않는다.
7. 구역을 뜻으로: 상업·민가·부유층·관아(·피난민).
8. 구역에 맞게 바닥·건물·소품. 초가는 울 없는 쪽이 기본, 울타리 초가는 많아야 한둘.
9. 앞쪽(왼쪽 아래 +v, 오른쪽 아래 +u) 성벽 옆에는 중요 건물을 두지 않는다 (내려다보는 시점에서 가림).
10. 구도가 안 맞으면 배치부터 다시.

이후 피드백으로 정한 것:
- 길은 한 종류(고을 흙길 `d`, 황성 포석 `i`). 예외: 업은 가운데 십자 대로만 포석(CFG `avenue`).
- 길은 판 끝까지 곧게, 성문 밖으로도 곧게. **성문 길은 성문 칸 -1·0** — 성문 그림의 통로, 조각 배치(gate_offset (k-5)/2), 엔진 걷기(gate_passage)가 모두 이 두 칸이다. 내려다보는 원근에서 통로 한쪽 옆벽이 입구를 가려 한 칸 밀려 보일 수 있으니 눈대중으로 옮기지 말 것(11-28 -2·-1로 옮겼다가 되돌림). 통로 바닥을 길 색으로 그린 성문 시안: result/20261128/gate-v2(적용은 선택).
- 건물·담장은 성벽에서 3칸, 서로 2칸 띄운다. 나무는 성벽 3칸·건물 2칸·나무끼리 3칸 밖. 빽빽하게 하지 않는다.
- 건물은 길을 향한다: 왼쪽 아래 길 → 원래 그림, 오른쪽 아래 길 → 반전 그림 `_r`(빛을 거울로 놓고 그린 뒤 뒤집은 것).
- 관청 구역은 크게: 담장 마당 옆 2·뒤 1·앞 4칸, 담장 안 앞마당까지 전부와 대문 밖 광장을 돌 포석(건물 바닥에만 깔면 거절됨). 안 들어가면 옆 1칸 → 담장 없는 포석 마당(yard) → 작은 저택 순으로 물러선다. 관청 대문 양옆 깃발·등. 매대는 교차로 가까운 대로변 블록 하나를 **시장 구역**으로 정해 그 안에 줄지어(대로변에 흩뿌리기는 거절), 가장자리에 수레·자루·물항아리. 주점 앞 술항아리. 마을은 원래 있던 만큼만.
- 원래 성 안에 있던 나무 종류(복숭아·매화 등)를 살린다.
- 대장간은 앞이 트인 바깥 헛간. 성 안 대장간 앞 출구 → 지역별 바깥 마당 맵 `S_smithy_yard_<지역>`(smithy_yard.py, 옛 실내와 같은 좌표틀이라 대장장이·견습공·c02-e-smith 동선 그대로). 대장장이를 성 안으로 옮기는 안은 버림(11-28).
- **장면 무대 지키기**(stage.py, 11-28 "3번 싹 고쳐"): 장면에서 사람이 서는 칸·걷는 길(u 먼저 v), 선 자리 앞(화면 아래) 네 칸은 비운다. 끝으로 화면 상자로 사람을 가리는 물건을 치우고, 사람 바로 뒤에 있던 옛 집은 그 자리에 지역 그림으로 되살리고(ANCHOR_SWAP), 모여 선 곳은 네모 광장(옛 맵 그 자리 바닥 글자).
- 관청·주점·도구점·저택 출구는 새 건물 문(담장이 있으면 대문) 앞으로.

## 맵별 설정 (`towngen.py` CFG)

| 맵 | 설정 | 뜻 |
|---|---|---|
| S_ye | `avenue: 'i'` | 가운데 십자 대로만 포석 |
| S_xuzhou_gate | `flip_u`, `outside` | 성벽 오른쪽 아래가 성 안, 왼쪽 위 성 밖은 넓은 대로·나무·밭만 (여포군 등장) |
| S_xiapi_inner | `ruin` | 수공 직후: **맵 전체** 흙탕·웅덩이·젖은 자국·쓰레기 줄, 흙탕물 자국 진 집(`_mud`), 집 절반 (동서로 나누면 거절) |
| S_mangdang | `drop_water` | 끊긴 물줄기 지움 |
| S_vil_peace | `mode: keep`, `rename tavern→cottage_3`, `thin_trees: 2` | 융중: 원래 배치 그대로(초려 앞 제갈량 장면), 집만 지역 그림, 길 2칸 안 나무 솎기. 외딴 초가(hermit)는 장면과 안 맞아 버림 |
| S_jiangxia_harbor | `mode: keep`, `erase_d` | 나루: 원래 배치 그대로, 집만 지역 그림, 대각선 흙 띠만 지움 |
| S_danyang_village | `mode: keep`, `cliff_forest`, `fence_around` | 원래 배치(큰 나무·우물 광장) 그대로, 원래 절벽 언덕(높이 층)을 그대로 두고 위는 빈 풀밭 없이 숲, 울타리는 사람이 걷는 범위 둘레(절벽에 닿으면 멈춤, 큰길만 틈). 전투맵 언덕(`hills`)으로 바꾸는 안은 거절됨 |

새 맵에 특별한 사연(피난민, 불탄 뒤, 성 밖 장면 …)이 있으면 CFG에 모드를 더하고 표에 적는다.

## 도구 (ver5/tools/towngen, 생성물은 `work/` — git 밖)

- `town_dot2.py` 지역 건물 그림(A안: 상자 면에 도트, gen_pixel_castle 렌더러). 집만 남기고(`yard_from`) 바닥을 다시 잡음, `_r` 반전판, 상자 투영 그림자 마스크. `town_dot.py` 공용 페인트, `shade.py` 처마 밑 그늘·해 그림자·땅 그림자 마스크.
- `town_lines.py` 줄 타일: `<지역>_twall`(기와 담장)·`_ewall`(흙담)·`_fence`(울타리) 16조각, 대문 `_tgate_u/v`·`_egate_u/v`.
- `build_root.py` 미리보기 루트(ver5/assets 복사, data 등은 **복사** — 심볼릭 링크면 scenario-admin이 "rules incomplete"로 꺼짐), 지역 그림·줄 타일·언덕/산을 extra 시트와 scene.json에.
- `towngen.py ROOT MAP REGION` 한 맵 생성, `batch.sh [--art] [MAP:REGION ...]` 전체(빌드·생성·렌더·걷기 점검), `page_all.py` 비교 페이지, `check_cells.py` 지킬 칸 덮임 검사.
- `stage.py` 장면 무대(선 자리·걷는 길·가리는 물건·기대던 옛 집·광장), towngen이 생성 때와 끝(`stage_pass`)에 쓴다. `smithy_yard.py ROOT 지역…` 대장간 마당 맵과 탐험·장면 Map 바꾸기(batch.sh가 끝에 부름). `scene_compare.sh` 장면을 옛/새 맵에 그려 `work/cap/pair/<장면>.png`, 장면 동선 테스트 명령도 알려 줌.
- 엔진: `cmd/_walkcheck`(탐험 지역마다 시작 칸에서 사람·출구까지, 순찰 지점까지 걸어서 닿는지), 줄 조각 마스크 그림자(internal/gui·maplight, 기존 맵 101개 픽셀 동일 확인).

## 절차

1. 대상 맵과 지역을 정한다(화북·중원·회남·강남·파촉). 장면(chNN.json)·탐험(explore*.json)이 그 맵을 어떻게 쓰는지 본다. 좌표: 탐험 At은 소타일, 장면은 게임 칸(×2).
2. 특별한 사연이 있으면 CFG에 넣는다.
3. `tools/towngen/batch.sh MAP:REGION` (그림을 바꿨으면 `--art`). 결과 줄: 검사 통과·누락·덮임·성 키움·울타리 초가 수·역할 건물.
4. **렌더를 직접 연다** (`work/render/gen/MAP.png`, 현재판 `work/render/gencur/`). 숫자 점검만으로 보고하지 않는다: 길·성문 정렬, 성벽과의 간격, 겹친 나무, 빽빽함, 구역이 맞는지.
5. 걷기 점검 `work/walk_gen.txt`가 0인지(탐험 순찰 지점 포함). `scene_compare.sh`로 장면을 옛/새 맵에 그려 쌍마다 직접 본다(사람이 풀밭 한가운데·나무 뒤·지붕 밑에 있지 않은지). 장면 동선 테스트는 스크립트가 알려 주는 명령으로 사본에서. 대장간처럼 탐험 이벤트로 여는 장면은 임시 데모로 게임에서 돌려 캡처.
6. `page_all.py`로 결과 폴더(result/YYYYMMDD/…)에 페이지, Artifact로 올리고 URL을 보고. 미리보기 어드민은 `work/root`에서 18766.
7. 게임 데이터 반영은 사용자 승낙 뒤: 그림을 assets_candidate로, build_extra, scene.json(walls·footprints), 맵·탐험 파일. 반영 전엔 미리보기 루트에만 쓴다.

## 하지 않는 것

- 치환(옛 자리 바꿔 끼우기), 길 두 종류 섞기, 성벽·건물에 붙은 배치, 담장이 성벽 줄과 맞닿는 배치(엔진은 종류가 달라도 이웃 줄을 이어 그려 성가퀴가 빠진다).
- 저장소 맵·시나리오를 승낙 없이 덮어쓰기. 공유 작업 트리라 남의 변경도 건드리지 않는다.

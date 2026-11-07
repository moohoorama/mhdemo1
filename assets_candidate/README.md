# assets_candidate

게임에 넣기 전에 눈으로 보고 고르기 위한 후보 그림. 이 폴더의 그림은 원본이고, 게임이 읽는 것은 `ver5/assets/graphics/`다. `python3 ver5/tools/build_extra.py`가 옛 기둥 성문(`gate_*`)과 숲 속 덤불(`cover_bush_*`)을 뺀 나머지를 `graphics/map/extra.png`로 묶고 불·연기 줄을 `graphics/effects/`로 옮긴다(2026-10). 장면 14개는 `ver5/tools/scene_maps/`가 `graphics/maps/S_*.json`으로 만든다.

## 열어 보기

`viewer.html`을 브라우저로 연다(외부 파일 없음). 장면 20개(전투맵 화공·성문 A·C·다리, 시나리오 마을 방화·군막 내부·길·언덕·진영·성 밖·성 안·성루·관아·궁궐·동승의 집·조조의 후원)와 후보 목록, 울타리 16종이 있다.
`python3 tools/shot.sh <장면> <상태번호> out.png`는 헤드리스 Chrome으로 같은 화면을 저장한다.

## 후보

전투맵은 데포르메되어 있어(장소 그림이 캐릭터의 1/4 안팎) 그림을 **전투맵용**과 **시나리오 장면용**으로 나눈다.

| 쓰임 | 종류 | 파일 | 만든 방법 |
|---|---|---|---|
| 전투맵 | 성문 A(기둥 문루, 열림·닫힘) | `tiles/gate_floor_{u,v}.png`, `gate_{open,closed}_{u,v}_{back,front}.png`, `gate_roof_{u,v}.png` | 처음부터 새로 그림. 성벽 사이 한 칸을 채우는 문루: 높은 기둥 넷이 슬레이트 지붕을 받쳐 말 탄 기병이 밑으로 지나갈 높이(기둥 32px), 통로 앞뒤 끝에 문이 하나씩. 열림은 문이 안쪽으로 젖혀져 양옆에 붙고 닫힘은 두 문이 통로를 막는다. 층: 바닥(ground), 뒤 문틀(정렬 y−3), 앞 문틀(y+3), 지붕(y+5) — 병사·기병이 문틀 사이로 지나가고 지붕 밑에 들어가게 함. `_u`는 성벽이 u축을 따라갈 때(SW·NE면), `_v`는 좌우 반전. 주둔 가능한 관문 타일(`sites.png`)과는 별개 |
| 전투맵 | 성문(문루) | `tiles/gatehouse_{closed,open}_{u,v}_{0..3}.png` | `tools/gen_pixel_kit.py`: 게임 칸 하나(작은 칸 2×2)를 덮는 마당형 문루를 도트로 직접(성벽 높이 16px, 양옆 담과 앞뒤 문 담, 마당 폭 거의 끝까지 넓은 아치, 문 위 성벽에 붉은 기둥 넷과 검은 평판). 닫힌 것은 문짝이 아치를 막고, 열린 것(관문 `p` 포함)은 문짝이 마당 양옆 담에 붙음. 칸 중심선(u+v=2) 뒤는 조각 0, 앞은 조각 3, 1·2는 빈 그림 — 성문 칸의 유닛이 그 사이에 그려짐. `_u`는 성벽 줄이 u 방향 |
| 전투맵 | 성벽 | `tiles/wall_clean_00..15.png` | `tools/gen_pixel_kit.py`: 작은 칸 하나의 성벽을 도트로 직접(벽돌 무늬를 가로 2·세로 1로 쌓음), 트인 면(nw,ne,se,sw = 비트 0~3)에 성가퀴 |
| 장면 | 실내 벽·바닥 | `tiles/wall_{plaster,lacquer,canvas}_{nw,ne}.png`, `floor_{wood,carpet,palace,tent}.png` | `tools/gen_pixel_kit.py`: 같은 방식. props.png의 같은 이름 그림을 대신한다 |
| 전투맵 | 마을·주둔지와 불탄 판 | `tiles/village_0..3.png`, `camp.png`, `burnt_village_0..3.png`, `burnt_camp.png` | `tools/gen_pixel_kit.py`: 게임 칸 하나를 도트로 직접. 마을 네 판 — 0 초가·기와집·짚더미·우물·울짱, 1 ㄱ자 초가 둘·장독대·텃밭, 2 큰 기와집·기와 곳간·기와 얹은 흙담, 3 오두막 셋·빨래 장대·장작더미 — 가운데는 유닛 자리. 주둔지는 끝 깎은 말뚝 목책(오른쪽 앞 출입구)·지휘 막사·병사 막사·보급 꾸러미·깃대(깃발은 scenery 오버레이). 불탄 판은 지붕·천막 없이 들쭉날쭉한 벽 껍데기·말뚝만 남고 숯 색. sites.png의 같은 이름 그림을 대신한다 |
| 전투맵 | 불탄 숲 | `tiles/burnt_forest_0..3.png` | 기존 소나무 숲(`forest_*`)을 같은 방식으로 |
| 전투맵 | 울타리 | `tiles/fence_00..15.png` | `tools/gen_pixel_kit.py`: 가로대 두 줄 나무 울타리, 칸 중심 기둥에서 이어진 면의 가운데까지, 맵 배율 그대로. 성벽과 같은 이음 마스크(nw,ne,se,sw = 비트 0~3) |
| 전투맵 | 그을음 | `tiles/ash_{1,2,3}_{0,1,2}.png` | 32×16 마름모 반투명 덧씌움, 3단계 × 변형 3 |
| 전투맵 | 손상·파괴 다리 | `tiles/bridge_{u,v}_{damaged,broken}.png` | 기존 `bridge_*`에서 구멍 / 가운데를 지운 토막 |
| 전투맵 | 불길 | `effects/flame_t.png`(10×18), `flame_t_small.png`(7×11) | 6프레임 좁은 불꽃 한 줄기. 한 칸에 3~5개, 2×2칸 구조물에는 10개 안팎을 타원(가로가 세로의 2배)으로 놓아 바닥이 평평해 보이지 않게 함 |
| 전투맵 | 연기 | `effects/smoke_t.png` | 16×16 × 8프레임, 초당 12px 상승 |
| 장면 | 불길·연기 | `effects/flame.png`(14×26), `flame_small.png`, `smoke.png` | 같은 방식의 큰 크기 |
| 장면 | 허리 높이 울타리 | `tiles/sfence_00..15.png` | 시나리오 마을 등 |
| 장면 | 불탄 주막·큰 나무·초가 | `tiles/burnt_tavern.png`, `burnt_oak_0..2.png`, `burnt_hut.png` | 주막과 큰 나무는 장면 전용 (전투맵은 소나무뿐) |
| 장면 | 화로 | `tiles/brazier.png` | 군막 내부용 |
| 장면 | 새 소품 | `tiles/screen_{u,v}.png`(병풍), `lantern.png`(등롱), `curtain_{nw,ne}.png`(휘장), `stairs_{u,v}.png`(계단), `throne_{u,v}.png`(옥좌 단상, 3×2칸: 벽에 붙여 u는 SW, v는 SE를 향함), `milestone.png`(이정표), `incense.png`(향로) | 시나리오 장면 소품. 장면 크기(실제 크기)로 그렸고 기준점은 바닥 중심(벽 조각은 성벽 소품처럼 변의 위쪽 꼭짓점) |
| 장면 | 3D 모델로 만든 소품 | `tiles/hall_{u,v}.png`(기와지붕 관청: 돌 기단, 회벽, 붉은 기둥, 문 달린 정면), `plum_tree.png`(매화: 굽은 줄기와 가지 끝의 흰·분홍·붉은 꽃송이), `taihu_rock.png`(태호석: 높이 다른 뾰족한 돌 다섯에 구멍), `pavilion.png`(정자, 2×2칸: 붉은 기둥 네 개, 기와지붕) | 게임의 3D 셀 렌더러(`tools/unit3d`: 같은 카메라·팔레트·1px 외곽선, 주막·복숭아나무를 만든 방식)로 모델을 만들어 그림. `tools/gen_3d_props.py`가 `tools/unit3d`를 읽기 전용으로 불러 쓴다. `_u`는 정면이 SW면, `_v`는 SE면 |
| 장면(성 밖) | 큰 성벽 세트(3D 모델) | `tiles/swall_{u,v}.png`(성가퀴 달린 높은 성벽, 한 칸 길이·16px 폭 조각, 칸 안에서 끝나는 성가퀴 하나·활구멍, 두께 한 칸), `sgate_{closed,open}_{first,second}_{u,v}.png`(성문 누각: 네 칸 폭, 통로 두 칸, 기와지붕 2층. 가로로 반씩 두 조각: first(화면 위쪽 반)는 유닛 앞에, second(아래쪽 반, 지붕 포함)는 유닛 뒤에 그린다. 한 모델을 통째로 그린 뒤 화소만 나눠 자름 — 잘린 곳에 외곽선이 없다) | 전투맵 성벽은 데포르메되어 낮아서 성 밖 장면만 큰 성벽을 쓴다. 성 안·성루 장면은 기존 전투맵 성벽(`wall_clean_*`)이 더 어울려 그것을 쓴다. 다리는 전투맵 `bridge_v` 한 칸 그대로(해자 두 칸 폭) |
| 숲 속 병력 | 발치 덤불 `tiles/cover_bush_{0,1,2}.png` | 숲 칸에 선 병력이 숲에 있다는 느낌을 내는 아이디어 시연(`viewer.html`의 "숲 속 병력"): 현재(나무 앞에 온전히 섬) / ① 발치 덤불 / ② 숲을 병력 앞에 그려 반투명 / ③ 덤불 + 녹색 물들임 + 바닥 그늘 | 게임 코드는 아직 안 바뀜. 현재는 숲 그림 깊이값이 칸 중심보다 0.1 앞이라 칸 중심의 병력이 숲보다 나중에 그려짐 |
| 장면 | 성내 큰길 | `tiles/avenue_{u,v}.png` | 바닥층 32×16 칸. 따뜻한 사암 포석 길(양옆 연석, 가운데 선). 바닥 기본은 게임의 성내 타일(`structures.png` floor) |
| 전투맵 | 땅 장식과 소나무 변형(소프트 장식 + 외곽선 나무, 물·절벽 윗면·구조물 움직임) | `scenery/scenery.png`, `scenery.json`, `preview_*.png` | 전투에 영향을 주는 것(나무)은 게임의 외곽선·팔레트를 그대로, 장식(풀포기·꽃·자갈·얼룩·갈라짐·이끼·낙엽·덤불·그림자, 물가 갈대·거품·연잎·반짝임, 절벽 윗면의 돌·마른 덤불)은 외곽선 없이 땅과 가까운 톤으로 반투명하게. 소나무는 하나의 원뿔 더미 모델에서 10종 × 잎 흔들림 4프레임. 주둔지 깃발(원본 천을 지우고 4프레임), 성 깃발, 마을 연기는 `scenery.json`의 `sites`로 구조물에 얹는다. `python3 tools/gen_scenery.py [--install]` — 단독으로 돌려도 `tiles.json`을 건드리지 않는다. 설명은 `ver5/docs/temp/PLAN_map-scenery.md` |

규격은 `tiles/tiles.json`(기준점 `pivot`, 층 `layer`), `effects/effects.json`에 있다. 기준점은 바닥에 닿는 점이고, 칸 하나 그림은 게임의 구조물과 같은 40×48 칸에 기준점 (21,39)다.

## 유닛(미리보기용)

`units/*.png|json`은 `ver5/assets/sprites/*.spr`을 기본색으로 펼친 것이다(`tools/spr.py`). 시연 화면에서만 쓴다.

## 다시 만들기

```
cd tools
python3 spr.py light_infantry light_cavalry liubei_noncombat bandit guanyu_noncombat   # 유닛 미리보기(처음 한 번)
python3 build_all.py      # 모든 후보 그림 + tiles.json + viewer.html (한 프로세스로 돌려야 목록이 한 곳에 모인다)
```

`gen_all.py`(문·불탄 것·그을음·장면 울타리·다리·불길·연기), `gen_interior.py`(화로), `gen_gate.py`(성문 A), `gen_scene_props.py`(2D 장면 소품), `gen_3d_props.py`(3D 모델 소품), `gen_3d_castle.py`(장면용 큰 성벽 세트)는 `build_all.py`가 불러 쓴다. `gen_pixel_castle.py`·`gen_pixel_kit.py`는 따로 `--write`로 돌린다(`build_all.py`가 tiles.json을 새로 쓰므로 그 뒤에 다시). 따로 돌리면 `tiles/tiles.json`에 일부만 남아 시연 화면이 깨진다.

Pillow가 필요하다. 그림은 게임 그림을 읽어 바탕으로 쓰므로 `ver5/assets/graphics`가 있어야 한다.

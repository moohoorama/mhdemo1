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
| 전투맵 | 성문 C(낮은 문루) | `tiles/gatehouse_closed_{u,v}.png`, `gatehouse_floor_{u,v}.png` | 게임의 성벽 블록(`wall_15`) 위에 만든 한 칸짜리 문루라 벽돌·외곽선·높이(면 17px)가 옆 성벽과 같다. 성가퀴 윗면을 지우고, 한쪽 면에 반원 머리의 아치와 닫힌 문을 내고, 처마가 성벽 윗선에 놓이는 낮은 지붕을 얹음. 열리면 이 그림이 사라져 성벽 사이가 뚫린 길(바닥 `gatehouse_floor_*`만 남음)이 된다. `_u`는 성벽이 u축(문이 SW면), `_v`는 v축(SE면) |
| 전투맵 | 이음새 지운 성벽 | `tiles/wall_clean_00..15.png` | 게임의 성벽 그림에서 칸이 이어지는 쪽의 끝 외곽선(성벽 따라 반복되는 세로 눈금)을 지움. 성벽을 이어 그릴 때 이 그림을 쓰면 눈금이 없다 |
| 전투맵 | 불탄 마을·주둔지 | `tiles/burnt_village.png`, `burnt_camp.png` | 마을·주둔지 구조물(100×110)을 숯 색으로, 일부를 지우고 불씨를 얹음 |
| 전투맵 | 불탄 숲 | `tiles/burnt_forest_0..3.png` | 기존 소나무 숲(`forest_*`)을 같은 방식으로 |
| 전투맵 | 낮은 울타리 | `tiles/fence_00..15.png` | 무릎 아래 높이, 성벽과 같은 이음 마스크(nw,ne,se,sw = 비트 0~3) |
| 전투맵 | 그을음 | `tiles/ash_{1,2,3}_{0,1,2}.png` | 32×16 마름모 반투명 덧씌움, 3단계 × 변형 3 |
| 전투맵 | 손상·파괴 다리 | `tiles/bridge_{u,v}_{damaged,broken}.png` | 기존 `bridge_*`에서 구멍 / 가운데를 지운 토막 |
| 전투맵 | 불길 | `effects/flame_t.png`(10×18), `flame_t_small.png`(7×11) | 6프레임 좁은 불꽃 한 줄기. 한 칸에 3~5개, 2×2칸 구조물에는 10개 안팎을 타원(가로가 세로의 2배)으로 놓아 바닥이 평평해 보이지 않게 함 |
| 전투맵 | 연기 | `effects/smoke_t.png` | 16×16 × 8프레임, 초당 12px 상승 |
| 장면 | 불길·연기 | `effects/flame.png`(14×26), `flame_small.png`, `smoke.png` | 같은 방식의 큰 크기 |
| 장면 | 허리 높이 울타리 | `tiles/sfence_00..15.png` | 시나리오 마을 등 |
| 장면 | 불탄 주막·큰 나무·초가 | `tiles/burnt_tavern.png`, `burnt_oak_0..2.png`, `burnt_hut.png` | 주막과 큰 나무는 장면 전용 (전투맵은 소나무뿐) |
| 장면 | 화로·지도 탁자 | `tiles/brazier.png`, `map_table.png` | 군막 내부용 |
| 장면 | 새 소품 | `tiles/screen_{u,v}.png`(병풍), `bed.png`(침상), `lantern.png`(등롱), `curtain_{nw,ne}.png`(휘장), `stairs_{u,v}.png`(계단), `throne_{u,v}.png`(옥좌 단상, 3×2칸: 벽에 붙여 u는 SW, v는 SE를 향함), `milestone.png`(이정표), `incense.png`(향로) | 시나리오 장면 소품. 장면 크기(실제 크기)로 그렸고 기준점은 바닥 중심(벽 조각은 성벽 소품처럼 변의 위쪽 꼭짓점) |
| 장면 | 3D 모델로 만든 소품 | `tiles/hall_{u,v}.png`(기와지붕 관청: 돌 기단, 회벽, 붉은 기둥, 문 달린 정면), `plum_tree.png`(매화: 굽은 줄기와 가지 끝의 흰·분홍·붉은 꽃송이), `taihu_rock.png`(태호석: 높이 다른 뾰족한 돌 다섯에 구멍), `pavilion.png`(정자, 2×2칸: 붉은 기둥 네 개, 기와지붕) | 게임의 3D 셀 렌더러(`tools/unit3d`: 같은 카메라·팔레트·1px 외곽선, 주막·복숭아나무를 만든 방식)로 모델을 만들어 그림. `tools/gen_3d_props.py`가 `tools/unit3d`를 읽기 전용으로 불러 쓴다. `_u`는 정면이 SW면, `_v`는 SE면 |
| 장면(성 밖) | 큰 성벽 세트(3D 모델) | `tiles/swall_{u,v}.png`(성가퀴 달린 높은 성벽, 한 칸 길이·16px 폭 조각, 칸 안에서 끝나는 성가퀴 하나·활구멍, 두께 한 칸), `sgate_{closed,open}_{first,second}_{u,v}.png`(성문 누각: 네 칸 폭, 통로 두 칸, 기와지붕 2층. 가로로 반씩 두 조각: first(화면 위쪽 반)는 유닛 앞에, second(아래쪽 반, 지붕 포함)는 유닛 뒤에 그린다. 한 모델을 통째로 그린 뒤 화소만 나눠 자름 — 잘린 곳에 외곽선이 없다) | 전투맵 성벽은 데포르메되어 낮아서 성 밖 장면만 큰 성벽을 쓴다. 성 안·성루 장면은 기존 전투맵 성벽(`wall_clean_*`)이 더 어울려 그것을 쓴다. 다리는 전투맵 `bridge_v` 한 칸 그대로(해자 두 칸 폭) |
| 숲 속 병력 | 발치 덤불 `tiles/cover_bush_{0,1,2}.png` | 숲 칸에 선 병력이 숲에 있다는 느낌을 내는 아이디어 시연(`viewer.html`의 "숲 속 병력"): 현재(나무 앞에 온전히 섬) / ① 발치 덤불 / ② 숲을 병력 앞에 그려 반투명 / ③ 덤불 + 녹색 물들임 + 바닥 그늘 | 게임 코드는 아직 안 바뀜. 현재는 숲 그림 깊이값이 칸 중심보다 0.1 앞이라 칸 중심의 병력이 숲보다 나중에 그려짐 |
| 장면 | 성내 큰길 | `tiles/avenue_{u,v}.png` | 바닥층 32×16 칸. 따뜻한 사암 포석 길(양옆 연석, 가운데 선). 바닥 기본은 게임의 성내 타일(`structures.png` floor) |

규격은 `tiles/tiles.json`(기준점 `pivot`, 층 `layer`), `effects/effects.json`에 있다. 기준점은 바닥에 닿는 점이고, 칸 하나 그림은 게임의 구조물과 같은 40×48 칸에 기준점 (21,39)다.

## 유닛(미리보기용)

`units/*.png|json`은 `ver5/assets/sprites/*.spr`을 기본색으로 펼친 것이다(`tools/spr.py`). 시연 화면에서만 쓴다.

## 다시 만들기

```
cd tools
python3 spr.py light_infantry light_cavalry liubei_noncombat bandit guanyu_noncombat   # 유닛 미리보기(처음 한 번)
python3 build_all.py      # 모든 후보 그림 + tiles.json + viewer.html (한 프로세스로 돌려야 목록이 한 곳에 모인다)
```

`gen_all.py`(문·불탄 것·그을음·울타리·다리·불길·연기), `gen_interior.py`(화로·지도 탁자), `gen_gate.py`(성문 A), `gen_gatehouse.py`(성문 C), `gen_walls.py`(이음새 지운 성벽), `gen_scene_props.py`(2D 장면 소품), `gen_3d_props.py`(3D 모델 소품), `gen_3d_castle.py`(장면용 큰 성벽 세트)는 `build_all.py`가 불러 쓴다. 따로 돌리면 `tiles/tiles.json`에 일부만 남아 시연 화면이 깨진다.

Pillow가 필요하다. 그림은 게임 그림을 읽어 바탕으로 쓰므로 `ver5/assets/graphics`가 있어야 한다.

# 결정 기록

이 문서가 맡는 범위: 설계 결정과 그 이유. 현재 규칙은 각 분야 문서(`logic-*`, `data-*`, `presentation-*`, `tools-*`, `system-*`)가 말하고, 여기에는 **왜 바꿨는가**만 적는다.
변경할 때마다 적는다. 변경한 파일 이름, 수정 내역, 의도(또는 이유)를 `파일이름: 수정내역 — 의도` 형식으로 남긴다. 새 항목을 위에 쌓는다. 설계 결정은 `결정 · 이유 · 영향받는 문서` 항목으로 쓴다.

---

## 2026-10-22 도주 유닛의 전장 이탈 연출

결정 · 도주 유닛은 목표 칸에 닿는 즉시 전장 밖으로 걸어 나가며 어두워지고 사라진다 · 이유: 사용자 "동탁 추격전에서 원술을 비롯 동맹군이 후퇴하는게 매우 어색. 병력이 타일 한곳에 겹쳐서 배치되는 증상" — 이동 재생이 같은 명령의 `retreat` 이벤트를 무시해 퇴각한 유닛이 목표 칸에 그대로 보이고, 다음 유닛이 그 칸으로 걸어 들어가 겹쳤다 · 영향받는 문서: presentation-replay.md
- `internal/core/engine.go`: `leaveByFlight`의 `retreat` 이벤트에 `Text: "flight"` — 화면이 쓰러짐과 이탈을 구분하게
- `internal/gui/player.go`: `unitVis.leave`·`leaving`, `leaveField`(가장 가까운 가장자리 바깥으로 1.5칸, 모서리면 걸어온 방향 `heading`), `perform`이 어떤 명령이든 `flight` 퇴각에 이탈 박자를 깔고 `strike`는 깜빡임을 하지 않는다
- `internal/gui/field.go`, `field_outside.go`: 이탈 중인 유닛은 `edgeShade`(가장자리를 넘은 첫 칸에 걸쳐 바깥 땅 밝기 `exteriorDim`까지)로 어둡게, 진행의 제곱으로 투명하게, 그림자도 함께 옅게
- `internal/gui/view.go`: `syncUnit`이 살아 있는 유닛의 `leave`를 지운다
- `internal/gui/uidemo_flee.go`(새): AI 아군 넷을 동쪽 끝 목표로 도주시키는 데모(`-ui flee -stage B04`)

## 2026-10-22 전원 반격 기본 켬, _balance가 게임 기본 설정으로 측정

결정 · 전원 반격을 기본으로 켠다 · 이유: 사용자 "전원반격 모드를 기본으로 하자" · 영향받는 문서: logic-combat.md, presentation-ui.md, system-architecture.md, stage_lv_balance.md
- `internal/storage/storage.go`: `DefaultSettings().CounterAll = true`, `counter_all`의 `omitempty`를 빼서 끈 값이 저장되게(키가 없는 예전 설정 파일은 기본값 켬으로 읽힘)
- `internal/storage/storage_test.go`: 예전 설정 파일은 켬, 끈 값은 끈 채로 저장
- `internal/session/coalition_test.go`: 요청 전 설정을 엔진에 옮기는 줄에 `SetCounterAll`
- `cmd/_balance`: `-damage`(기본 80)·`-xp`(기본 80)·`-counter`(기본 켬)로 게임 기본 설정과 같은 조건에서 재고, `-offset`으로 주력 레벨을 L에서 띄워 레벨별로 잰다. 전에는 데미지·경험치 100%, 반격 끔으로 쟀다
- `internal/core/engine.go`, `battle_events.go`: 합류 보정(`raiseTo`)을 `spawn` 안에서 빼 이벤트 `join`과 이야기 `join` 노드에서만 — `spawn`에 두었더니 출전할 때마다 모든 주력이 L까지 올라갔다. `internal/core/story_test.go`: 출전 때는 올리지 않는다

## 2026-10-22 스테이지 레벨을 순번·직급으로 계산

결정 · 스테이지 파일에서 절대 레벨을 없애고, 엔진이 스테이지 순번(이야기 속 몇 번째 전투인지)의 기준 레벨 L = 3 + 2(n − 1)에 직급 몫을 더해 계산한다. 직급 몫은 레벨 차가 아니라 `(L + 10)`에 대한 비율이다. 합류 장수는 합류할 때 L까지 올린다 · 이유: 사용자 "아군 레벨 10인데 여포가 20, 간옹은 2로 합류, 번궁은 레벨 12인데 3으로 합류", "현재의 스테이지를 기록해두고 그거에 맞춰 강제로 보정", "lv1과 5는 엄청난 차이지만 lv50과 55는 차이가 적지 않아?" — 능력치·피해가 `(레벨 + 10)`에 비례해서 같은 +5가 Lv3에서 38%, Lv50에서 8%다 · 영향받는 문서: stage_lv_balance.md, data-scenario-node.md, logic-growth.md, AGENTS.md
- `internal/content/content.go`, `spawn.go`, `scenario.go`, `validate.go`: `Rules.StageLevelBase`·`StageLevelStep`·`RankLead`, 스폰 `Rank`·`Shift`, 로드 때 `battle` 노드 순서로 `Stage.Order`, `StageLevel`·`SpawnLevel`·`StageOrder`·`NextBattleOrder`. 스테이지 레벨을 쓰는 규칙에서 스폰 `Level`은 검증 실패(시험용 mini 묶음은 규칙이 없어 예전처럼 `Level`)
- `internal/core/engine.go`, `growth.go`: 임시 인물(적·템플릿·AI)은 `SpawnLevel`, 합류하는 플레이 장수는 `raiseTo`로 L까지 올리고 오른 레벨마다 특성치 100. 이야기 `join` 노드는 다음 전투의 L. `SpawnPreview`가 스테이지 ID를 받는다
- `assets/data/game.json`: `StageLevelBase` 3, `StageLevelStep` 2, `RankLead` troop 0 · officer 0.08 · commander 0.15 · boss 0.30(B03 여포 12를 그대로 두는 값)
- `assets/scenario/hero/stage01~07.json`: `Level`을 모두 지우고 `Rank`·`Shift`로. B01~B03은 계산 결과가 예전 레벨과 같다. B04~B07은 정책 레벨로 올라간다(B04 여포 20 → 15, 서영 8 → 12, 복병 5 → 8, 조조군·제후군 장수 11; B05 순우경 14, 병력 10, 번궁 13; B06 안량 16, 장합 15, 병력 11; B07 원소 19, 장수 17, 병력 13)
- `assets/data/characters.json`, `characters_story.json`: 간옹·조운·번궁 `Points`를 정의 레벨에 맞춰 `100 × (레벨 − 1)` — 합류 보정 뒤 주력과 같은 특성치(간옹 Lv11 1000, 번궁 Lv13 1200, 조운 Lv15 1400)
- `internal/gui/editor.go`, `editor_ui.go`: 맵 편집기의 `LV ±`를 직급 순환 버튼으로(선택한 병력은 `장수 · LV 17`처럼 계산된 레벨도 보임)
- `cmd/_balance`: 주력 레벨을 `Rules.StageLevel`로(`-base`·`-step` 플래그 삭제)
- `internal/content/stage_level_test.go`(새), `internal/core/story_test.go`, `spawn_override_test.go`: 순번·직급 계산, 절대 레벨 거부, 간옹 합류 Lv11·특성치 1000

## 2026-10-22 장비 아이콘 19종 모두 그림에서 변환

- `tools/spritetool/assets/ver5-ui/sources/item_*.png`(16장 새), `settings.json`(`"source"`, 칼·창 `"solid": .35`, 활 .2), `pixels.json`, `assets/graphics/ui/ui.{png,json}`: 무기·말·장신구도 codex 그림에서 변환 — 사용자 "다 같은 방식으로. 이제 아이템 도트는 이런 식으로 하자"
- `tools/pixelize_icon.py`: 그림의 짙은 선을 외곽선 `k`로 세고, 칸의 4분의 1을 덮는 색이 이를 이김(가는 활시위·칼날이 외곽선에 묻히던 것), 짙은 선뿐인 듬성한 칸은 비움(활 안쪽이 검게 메워지던 것), 칠할 덮임 비율 `"solid"` — 칸 절반 미만인 칼날이 지워지던 것. 내부 선이 살아 갑옷도 또렷해짐
- `tools/spritetool/assets/ver5-ui/README.md`: 장비 도트는 손으로 찍지 않고 이 방식으로
- `.claude/skills/item-icon/SKILL.md`(새): 이 과정(이름·팔레트 → codex 그림 → 512px 원본 → 변환 → 확인 → 게임 캡처)을 스킬로 — 사용자 "스킬로 등록했어?"

## 2026-10-22 갑옷 아이콘을 그림에서 도트로 변환

결정 · 장비 아이콘이 손 도트로 각지고 무거우면, codex로 그린 그림을 팔레트 도트로 변환한다 · 이유: 사용자 "갑옷류·보조류가 너무 각져", 손 도트 재작업도 "나아지긴 했지만 별로", 사용자 제안 "지금 그린 거 주면서 codex로 그리고, 축소·지정한 색으로 색을 줄인 후(색이 많으면 그라데이션이 부드러워 도트처럼 안 보이니) 외곽선" → 결과 "훨씬 좋아" · 영향받는 문서: tools/spritetool/assets/ver5-ui/README.md
- `tools/pixelize_icon.py`(새): 마젠타 빼기 → 30도트 맞춤 → 큰 그림에서 픽셀마다 팔레트 색(Lab) → 칸마다 최다 색 → 외톨이 점 정리 → 1px 외곽선 → `pixels.json`. 평균 내어 줄이면 강철 위 붉은 끈이 흙탕색이 되고, 줄인 뒤 색을 줄이면 질감이 얼룩이 되어서 이 순서로
- `tools/spritetool/assets/ver5-ui/sources/item_{leather,robe,diamond}.png`(새, codex 그림 512px), `settings.json`(`"source"`), `pixels.json`, `assets/graphics/ui/ui.{png,json}`: 갑옷 3종을 3/4 각도의 둥근 실루엣으로. 가죽갑옷·금강갑은 섬세한 그림이 줄이면 얼룩져 "32x32 도트 아이콘, 큰 덩어리, 3단 명암"으로 다시 그림
- 장신구 4종(지도·병법입문·화계서·옥쇄)은 손 도트를 기울이고 둥글게 다시 그림
- `internal/gui/equipment_window.go`: 창고에 입은 것과 같은 장비가 또 있으면 `착용 중` 행이 두 번 나오던 것 — 입은 장비는 한 번만. `equipment_window_test.go`: `TestPickerListsTheWornPieceOnce`
- `internal/gui/uidemo_equip.go`: `EQUIP_SLOT`(기본 `weapon`)으로 고르기 창의 칸을 고름

## 2026-10-22 특성 분류 아이콘(책략·전법·특기 × 효과 색)

결정 · 특성 아이콘을 특성마다 그리지 않고, 모양 3종(책략 두루마리·전법 교차한 무기·특기 메달)을 효과 색 7종(공격 빨강·회복 흰색·강화 초록·약화 검정·방어 파랑·이동 노랑·책략술 보라)으로 물들인다 · 이유: 사용자 "책략, 전법, 특기로 분류해 단색 아이콘… 공격은 빨간색, 회복은 흰색, 버프는 초록색, 디버프는 검은색", 초록이 31개로 몰려 방어·이동·책략술 색을 더함. `병법`은 책략과 무기 기술을 함께 부르는 명령 이름이라 무기 기술 쪽 분류는 `전법`(공격력으로 적을 치는 기술), 늘 적용되는 특성은 `특기`로 · 영향받는 문서: logic-traits.md, presentation-ui.md
- `assets/data/traits.json`, `internal/testdata/mini/traits.json`, `internal/content/content.go`, `validate.go`: 특성 `Kind`(`strategy`·`tactic`·`talent`)·`Tone`(`attack`·`heal`·`buff`·`debuff`·`guard`·`move`·`craft`), 없거나 모르는 값이면 검증 실패. 대비·경계는 책략이지만 `guard`, 교란은 디버프 책략, 도발은 전법, 화신은 책략술
- `internal/gui/trait_icon.go`(새): 아이콘 그리기(약화의 검정은 옅은 테를 둘러 어두운 창에서 보이게), 기술은 종류로 모양·색을 정함
- `internal/gui/equipment_window.go`: 장비 특성 표식을 분류 아이콘으로, 배울 수 있는 특성은 이름 뒤 `배움`, 설명 창 제목 `특기 · 방어` 식. 메달 `trait_badge`(첫 글자) 폐기
- `internal/gui/hud.go`(특성 탭), `windows.go`(특성 모달), `learning.go`(학습 창), `scenario.go`(출전 준비 습득 특성): 특성 이름 앞에 아이콘
- `tools/spritetool/assets/ver5-ui/{settings,pixels}.json`, `README.md`, `assets/graphics/ui/ui.{png,json}`: `trait_strategy`·`trait_tactic`·`trait_talent`(26×26 회색 도트, 에이전트가 그리고 7색으로 검수)
- `internal/gui/equipment_window_test.go`: 모든 특성의 분류 아이콘과 색이 있는지

## 2026-10-22 장비 아이콘과 장비 창 개편

결정 · 장비 창을 명령 목록(`해제 철검`, `장착 청공검 ×1`, 페이지 넘김)에서 칸 카드 + 고르기 팝업으로 바꾸고, 장비마다 아이콘을 둔다 · 이유: 사용자 "장수가 장착하는 장비의 아이콘이 있어야함", "무기·방어구·보조 셋만 나오고 각 장비의 능력치·특성, 특성 아이콘을 누르면 설명… 장비를 누르면 목록 팝업, 고른 뒤 확인/취소… 스크롤" · 영향받는 문서: presentation-ui.md, data-item.md
- `internal/gui/equipment_window.go`(새): 무기·방어구·장신구 카드(아이콘·이름·그 장수에게 더하는 능력치·특성 표식), 카드를 누르면 뜨는 고르기 팝업(장비 벗기·착용 중·장착 가능·병종에 안 맞는 장비 순, 지금 장비와 견준 변화, 휠·끌기 스크롤, 확인/취소), 특성 표식을 누르면 뜨는 설명 창, 전투 HUD 장비 탭(`detailEquipment`) — 능력치는 게임 사본에 해제·장착을 적용해 계산(`itemEffect`, 캐시)해 장비 `Bonus`(성장 배율)가 아니라 실제 오르는 수치를 보이려고
- `internal/gui/scenario.go`: 출전 준비 상세의 장비 줄에 아이콘, 옛 `equipmentWindow`는 부관 전용 `deputyWindow`로
- `internal/gui/windows.go`, `game.go`: 장비 창 크기, 고르기·설명 창을 장비 창 위에 그림, `Esc`가 설명 → 고르기 → 장비 창 순으로 닫음, `updateEquipPicker` 입력
- `internal/gui/hud.go`: 장비 탭을 글 목록에서 아이콘·특성 표식으로
- `internal/gui/events.go`: `IncompatibleEquipment`·`MissingEquipment`·`InvalidActor` 한국어 문구 — 고르기 목록에 사유로 보이므로
- `internal/content/content.go`, `assets/data/equipment.json`: 장비 `Icon` 필드, 19종 모두 자기 그림
- `tools/spritetool/assets/ver5-ui/{settings,pixels}.json`, `seed.py`, `README.md`, `assets/graphics/ui/ui.{png,json}`: 장비 아이콘 19종(36×36 수제 도트, 에이전트가 그리고 x8·x2·x1로 검수)
- `tools/build_effects.py`: 스프라이트 `"dot": k` — 칸보다 k배 작은 격자를 최근접으로 키워 굽기(도트 아이콘을 UI 4배 그림 규격에 맞추려고)
- `internal/gui/uidemo_equip.go`(새): 데모 `equip`, `internal/gui/equipment_window_test.go`: 모든 장비 아이콘이 시트에 있는지

## 2026-10-21 설정에 전원 반격 옵션

- `internal/storage/storage.go`: `Settings.CounterAll`(`counter_all`, 기본 끔) — 사용자 요청 "100% 확률로 모든 캐릭터가 데미지 배율 50%로 반격, 설정에서 켜고 끄기"
- `internal/core/types.go`, `api.go`, `combat.go`: `State.CounterAll`·`SetCounterAll`, 켜면 반격 태세·특성이 없는 유닛도 사거리 안의 물리 공격에 일반 공격 ×0.5(`counterAllShare`)로 한 번 반격(연타 판정 없음). 태세·특성 반격은 기존 규칙(×0.7, 연타) 그대로. `CounterDamage` 예측도 같은 비율 — 데미지 배율처럼 엔진 상태에 실어 예측·시험 복사본·AI가 같은 규칙을 쓰게
- `internal/session/session.go`: 요청마다 설정을 엔진으로 동기화
- `internal/gui/windows.go`: 설정 창에 `전원 반격` 켜기/끄기 줄, 창 높이 604 → 668
- `internal/cli/play.go`: CLI 설정 화면에 같은 토글
- `internal/core/core_test.go`: `TestCounterAllAnswersEveryAttack`

## 2026-10-22 장면 사람·소품 손질(사용자 검수 반영)

- `../tools/unit3d/scene_props3d.py`(새), `tools/scene_maps/scene_props.py`, `tools/scene_maps/pixel_props.py`(삭제), `../assets_candidate/tiles/*`, `assets/graphics/map/extra.png`, `assets/graphics/index.json`: 장면 소품 14종(병기가·갑옷대·촛대·녹각·망루·방패 벽·거적·피 묻은 천·약탕관·부러진 창·쓰러진 깃발·솥·군량 자루·술독)을 손으로 찍은 정면 그림에서 3D 모델 렌더(소품 카메라·셀 셰이딩·외곽선, 손 리터칭 표 `RETOUCH`)로 바꿈. 망루는 사람 키에 맞춰 줄이고 초가 지붕을 세움, 솥·약탕관은 위가 열린 그릇(`bowl`)과 납작한 원통(`disc`) — 사용자 "2d 판대기 같아서 별로, 3d modeling 후 리터칭". 새·반짝임은 `scene_props.py`로 옮김
- `../tools/unit3d/scene_people.py`(`throne`, `POSE_KEY_TAGS`, `POSE_COLORS`), `tools/doteditor/tools/civ3d.py`(`sheet_head`, `to_tags`), `assets/sprites/xiandi_noncombat.spr`, `internal/sprite/testdata/units/xiandi_noncombat.png`: 헌제 옥좌를 사람보다 높은 등받이·금 장식 관·팔걸이·디딤단으로 키우고, 나무는 시트가 안 쓰는 말 색 칸에 붉은 옻칠로(헌제 팔레트가 themeA를 검게, themeB를 금색으로 바꾸므로). 시트 머리는 맨 윗 픽셀 대신 렌더된 머리 픽셀 위에 붙여 등받이에 머리가 가거나 얼굴이 둘 되던 것을 고침. 안 쓰던 `POSE_TAG_SWAP`은 지움 — 사용자 "헌제옥좌에 앉아있는거 같지가 않아"
- `../tools/unit3d/scene_people.py`, `assets/scenes/ch05.json`, `docs/presentation-scene.md`: 부축 2인(`helped_noncombat`)을 없앰(시트·스프라이트·비교 PNG 삭제, 5장 패잔병 장면에서 뺌) — 사용자 "부축은 그냥 없애죠". `tools/doteditor/heroes/helped_noncombat/` 작업 폴더는 남아 있다
- `AGENTS.md`: 장면 데이터 위치를 `assets/scenes/chNN.json`으로(옛 `scenes.json`이 남아 있었다)
- `internal/gui/field.go`(`propShadow`, `eachProp`이 소품 이름을 넘김): 이야기 장면 맵의 서 있는 소품마다 부드러운 땅 그림자(벽·휘장·병풍 제외), 소품보다 조금 넓게 오른쪽 아래로 — 사용자 "소품에도 그림자가 필요". 처음엔 너비의 70%로 소품 밑에 가려 안 보였다
- `tools/doteditor/tools/robe_colors.py`(새), `units/official_{crimson,purple,blue,green}_noncombat.yaml`, `assets/sprites/*.spr`, 비교 PNG, `assets/scenes/ch02.json`: 품계별 조복(삼공 진홍·구경 자주·대부 청·낭관 녹, 원로 검정)과 궁궐에서 옥좌에 가까울수록 높은 품계 — 사용자 "중신들이 계급별로 색이 다르잖아, 황제와 가까울수록 계급이 높다"
- `../tools/unit3d/scene_people.py`, `tools/doteditor/tools/civ3d.py`(`turned`): 눕기와 들것 위 부상병은 3D 작은 머리 대신 시트의 2D 머리를 옆으로 눕혀 찍고 흰 붕대, 부축은 둘 사이를 벌리고 부상병이 허리 숙여 어깨에 팔을 걸침 — 사용자 "머리가 안 보여", "오른쪽 아래 사람이 왼쪽 위 사람을 들고 가는 느낌"
- 같은 파일, `assets/scenes/ch09.json`: 엎드린 강노병을 빼고(`crossbow_prone_noncombat`의 시트·스프라이트·비교 PNG 삭제) 계교 방패 벽 뒤에는 선 강노병 — 사용자 "끔찍해, 누운 표현은 생략". `tools/doteditor/heroes/crossbow_prone_noncombat/` 작업 폴더는 남아 있다
- 같은 파일(`POSE_KITS`): 유비 앉기를 녹색 외투(시트 theme B) + 진영색 안옷으로 다시 그림 — 진영색 칸을 통째로 녹색으로 바꿔 외투가 사라져 보였다. 사용자가 본 관우(잿빛)·헌제(면류관 없음)는 고치기 전 첫 시도를 페이지에 올린 것이었다
- `tools/scene_maps/backdrops.py`: 원경을 다시 그림(구름, 세 겹 능선의 햇빛·그늘 면과 안개, 나무 실루엣, 관문·성벽의 망루·처마·깃발, 불타는 낙양의 두 겹 지붕·불규칙한 불길과 연기·불티) — 사용자 "전경은 좀 부족해"
- `tools/scene_maps/pixel_props.py`: 새 그림을 9×4로 키움(5×3은 화면에서 안 보였다). `internal/gui/scene.go`: 회상 색을 바랜 잿빛 갈색으로(노을과 구분되지 않았다)

## 2026-10-22 장면을 채우는 사람·자세·소품·맵·원경

결정 · 이야기 장면에는 앉은 자세를 그린다(2026-10-05의 "앉기 빼기"는 전투 모드 이야기였다) · 이유: 사용자 확인 · 영향받는 문서: presentation-scene.md
- `internal/scenes/scenes.go`, `internal/gui/scene.go`, `player.go`(`unitVis.pose`·`rest`): 쉬는 자세 `Pose`(배우·단계), 보초 왕복 `Patrol`, 웅성거림 `Chatter`, 장면 `Backdrop`, 시간대 `memory`(회상), 효과 `birds`·`glint`, 장면이 열릴 때 0.5초 암전 — 사용자 "인물 나와서 대사만", "진영인데 병사가 너무 안 보인다", 거록 매복의 복선
- `internal/gui/field_night.go`(`night.flat`), `assets.go`(`Backdrops`): 회상은 불빛 없이 고른 색, 원경 그림 읽기
- `internal/gui/field.go`: 이야기 장면 맵은 캔버스를 바닥색으로 채우지 않고 비움 — 채우면 그 뒤에 깐 원경이 가려졌다(그림 밖의 색은 장면 화면이 같은 바닥색으로 먼저 칠한다)
- `tools/scene_maps/scene_format.py`: `Backdrop`을 쓰고, 모르는 장면 키가 있으면 멈춤 — 처음엔 `Backdrop`을 조용히 버려 원경이 사라졌다
- `internal/gui/scenario_test.go`: 자세 이름이 그 배우 시트에 있는지 검사(오타면 그냥 서 있어 눈에 안 띄므로)
- `../tools/unit3d/scene_people.py`(새), `tools/doteditor/tools/civ3d.py`, `grid.py`(`poses`), `tools/spritebuild/build.py`(비전투 시트의 임의 동작): 3D 변환으로 문관(진현관·넓은 소매·홀, 조복은 진영색)·중신(검은 조복)·환관·부상병(누움·기대 앉음)·부축 2인·들것 3인·엎드린 강노병, 원술·공손찬 평복 시트, 장수 시트에 `sit`(헌제는 `throne`)만 덧붙임(다른 프레임은 다시 그리지 않음; 유비는 진영색 칸을 녹색 칸으로, 관우·헌제는 시트의 머리를 붙임) — 사용자 "신하가 신하 같은 모습이 아니다", "누워 있는 병사", 막사의 제후가 서 있기만 한 것. 픽셀은 스크립트가 만들었다
- `tools/doteditor/units/*_noncombat.yaml`(새 9·자세 추가 10), `light_cavalry_white.yaml`(새, 말 색만 흰색), `assets/sprites/*.spr`, `internal/sprite/testdata/units/*.png`: 위 시트와 비교용 PNG
- `assets/data/characters.json`: 원술·공손찬 `Look.Scene` — 회의·막사 장면에서 평복으로 앉게
- `tools/scene_maps/pixel_props.py`(새), `scene_props.py`, `../assets_candidate/tiles/*`, `assets/graphics/map/extra.png`·`index.json`: 병기가·갑옷대·촛대·녹각·망루·방패 벽·거적·피 묻은 천·약탕관·부러진 창·쓰러진 깃발·솥·금빛 병풍, 새·반짝임 그림
- `tools/scene_maps/compose.py`: 맹주의 막사 12×8(상석·좌석 8·수행 장수 자리), 장수의 군막·원술의 사치 막사(6×5), 계교 강과 다리, 거록 협곡 길목, 굶주린 진영, 포위된 성루의 부상병 자리, 궁궐의 옥좌 상자를 금빛 병풍·촛대로 — 사용자 "막사가 너무 작다", "거록 길목", "전풍과 매복을 논하는 장면"
- `tools/scene_maps/backdrops.py`(새), `assets/graphics/backdrops/*.png`: 원경 먼 산·관문·성벽·불타는 낙양
- `assets/scenes/ch01~09.json`: 위를 장면에 적용 — 궁궐(옥좌의 헌제, 동탁 곁 이유, 읍하다 동탁이 웃자 부복하는 문관들, 환관), 맹주의 막사 넷(앉은 제후와 뒤에 선 하후돈·정보·안량·문추, 웅성거림), 원술의 사치 막사, 손견의 굶주린 진영(거적의 부상병, 사수관 성벽 원경), 포위된 성루(누운 부상병, 신음, 오가는 보초), 패잔병 귀환(들것·부축), 공손찬의 군막(침상 곁에 앉은 공손찬), 거록 협곡(새 떼와 창끝 반짝임 뒤 번궁의 경고), 계교 다리(방패 벽 뒤 엎드린 강노병, 건너편 백마), 낙양 길 회상(`memory`), 원경
- `docs/presentation-scene.md`, `tools-build.md`, `.claude/skills/scene-staging/SKILL.md`, `check_scenes.py`: 위 형식·맵·사람·자세, 바닥 소품은 서 있을 수 있음

## 2026-10-21 장면 연출 개편(장면별 데이터·새 비트·새 맵), 전투와 이야기의 어긋남 수정

### 장면
결정 · 장면 데이터를 노드별 `assets/scenes.json`에서 장(章)별 `assets/scenes/chNN.json`의 장면(구간)별 항목으로 바꾼다 · 이유: 노드 238개가 같은 배우를 복사해 들고 있어 배우가 걸어간 자리·불이 노드마다 초기화되고(불이 한 줄 만에 꺼짐), 자동 배치(0.8칸 한 줄)가 겹침·바닥 밖·엉뚱한 인물 구성을 만들었다 · 영향받는 문서: presentation-scene.md, tools-build.md, system-architecture.md, system-save.md
- `internal/scenes/scenes.go`(새 패키지): 장면 파일 읽기, 노드 → 장면 색인, 엑스트라 키(`병사#2`) — GUI와 CLI가 같은 읽기를 쓰게
- `internal/gui/scene.go`: 장면 안에서는 배우 자리·효과·시간대를 노드가 바뀌어도 유지, 중간 노드로 들어오면 앞 노드 비트를 한 번에 맞춤(`settleNode`). 새 단계 `Enter`·`Exit`·`Speed`·`After`·`Look`·`Say`(맵 위 말풍선)·`Shake`·`Cam`/`CamAt`·`Zoom`·`Time`·`Sprite`·`Prop`, 효과 `smallfire`·`smoke`·`steam`, 화자가 말할 때 주변 배우가 화자 쪽으로 돌아봄(`faceSpeaker`, `Hold`로 막음), 엑스트라 대기 동작 어긋남 — 사용자 "인물 나와서 대사만 하는 경우가 대부분… 이동·액션·뒤돌아보기·말풍선", "불 모양이 있었다가도 갑자기 사라져"
- `internal/gui/field_night.go`: 밤 어둠의 색(`night.color`, 노을·새벽은 옅게)과 장면의 소수 칸 불빛(`night.fires`) — 장면의 `Time`이 전투맵 밤 렌더를 그대로 쓰게
- `internal/gui/game.go`, `internal/cli/scenes.go`: `scenes.Load(assets/scenes)`로 읽기. CLI는 장면이 바뀔 때 배우를 새로 세우고 노드의 비트(등장·이동·퇴장·말풍선)를 글로 보임
- `internal/gui/scenario_test.go`, `scene_maps_test.go`, `officer_intro_test.go`: 새 형식 검사(배우 이름·스프라이트·진영·단계 이름·소품·비트 노드가 장면에 속함)
- `assets/scenes/ch01~04.json`: 1~4장을 콘티대로 다시 연출(황건 진영 밤 술판과 꺼지지 않는 불, 포위된 성루의 수비병·성 밖 화톳불, 새벽 진영에 말 타고 들어오는 조조·하후돈·하후연과 지나가는 손견·황개, 관우·장비와 함께인 유비, 낙양 길의 조조 기병, 대신이 두 줄로 선 궁궐, 상석·좌석·입구가 있는 맹주의 막사, 사수관 들판에서 멀리 나타나는 화웅과 동쪽에서 달려오는 유비군, 관우의 출진·북소리·귀환) — 사용자 지적(동탁·헌제만 있는 궁궐, 조조·유비 단둘, 병사 없는 진영, 겹침, 2~4장 반동탁 연합 장면)
- `assets/scenes/ch05~09.json`: 5~9장도 같은 방법으로 연출(산조 원소의 막사를 `S_council` 밤 술자리로, 공손찬의 막사를 `S_tent`로, 조조의 영채·패잔병 귀환, 하북 길에 말 타고 오는 간옹, 신도성 안의 병사들, 거록 길목의 안량 진과 양옆 매복, 계교 원소가 언덕에서 다리 건너 백마를 내려다봄). 산조 언덕의 '붉은 서쪽 하늘'은 하늘에 뜬 불 대신 노을(`dusk`)과 먼 연기로
- `assets/scenes.json`: 삭제. 런타임이 읽지 않던 옛 형식 맵 원본(`maps`)도 함께 지움(맵 파일 `S_peach`·`S_village`·`S_camp`는 그대로)
- `tools/scene_maps/compose.py`(새): viewer에 없는 방 `S_council`(9×6 맹주의 막사), `S_hall`(7×9 궁궐 전각)과 변형 `S_turban_camp`(누런 깃발·술독·군량), `S_rampart_siege`(군량·화로·깃발) — 4×4 막사·5×5 궁궐에는 제후 회의·대신 줄이 들어가지 않고, 황건 진영이 관군 붉은 깃발 맵을 쓰던 문제
- `tools/scene_maps/scene_props.py`(새), `../assets_candidate/tiles/{banner_yellow,command_flag_yellow,grain_sack,wine_jar}.png`, `tiles.json`, `assets/graphics/map/extra.png`, `index.json`: 누런 깃발(붉은 깃발 색만 바꿈)·군량 자루·술독(아스키 픽셀) — 새 그림은 대체할 것이 없는 소품만
- `tools/scene_maps/wire_story.py`: 장면이 없는 구간에 빈 장면(맵·노드)만 만들고 연출한 장면은 건드리지 않음(전에는 모든 노드의 배우를 자동 한 줄 배치로 덮어씀). 장소 표에 새 맵 반영
- `tools/scene_maps/scene_format.py`(새): 장면 파일 줄 배치(배우 하나, 비트 그룹 하나가 한 줄) — 비트 diff가 읽히게
- `tools/subset_fonts.py`: 글자 수집에 `assets/scenario`, `assets/scenes` 포함(전에는 이야기 파일이 빠져 있었다)
- `.claude/skills/scene-staging/check_scenes.py`: 새 형식, 비트를 차례로 재생하며 그룹마다 겹침·바닥·소품 검사, 야외 맵은 그림 범위까지 설 수 있음

### 전투와 이야기
- `story02.json`, `assets/scenes/ch02.json`: 급보 앞에 두 장면 추가 — 남양 원술의 막사(손견이 호진을 벴다는 첩보에 "늑대를 쫓고 범을 들이는 꼴"이라며 군량 수레를 막음), 사수관 앞 손견의 진영 밤(군량이 끊긴 지 열흘, 그날 밤 서량 기병의 야습, 동쪽으로 퇴각) — 사용자 "손견이 왜 화웅이랑 싸우게 되었는지, 왜 위기에 빠졌는지 설명이 없다". 연의 5회의 원술 참소·군량 단절
- `story03.json`, `stage02.json`, `assets/scenes/ch03.json`: 3장 들판 대화(손견 퇴각·황개의 경고·화웅의 외침·유비 도착·작전)를 B02 첫 차례 컷신(`field-meeting`)으로 옮기고, 전투 전 이야기는 유비가 서쪽의 패잔병을 보고 서두르는 두 줄만 남김 — 사용자 "전투씬 첫부분이랑 똑같다, 그 대사는 전투씬 처음에 나오는 게 맞다"
- `stage02.json`: 사수관 관우 대 화웅 일기토를 화웅 퇴각으로(`EnemyResult: retreat`, 기획서 대사 "수염 긴 놈, 죽기 전에 이름이나 대라!" / "이 원한은 잊지 않겠다!") — 일기토에서 화웅이 죽고도 막사 장면에서 다시 관우에게 베이던 이중 사망, 일기토가 막사 장면의 "소장이 나가 화웅의 목을…"을 미리 쓰던 중복. 기획서는 처음부터 퇴각이었다. 시험 `internal/duel/scripts_test.go`도 "화웅이 막히고 달아난다"로(퇴각 걸음, 관우는 제자리, 쓰러지지 않고 사라짐)
- `story09.json`, `stage07.json`: 전투 전 원소 장면을 다리 앞 매복을 자신하는 대화로 바꾸고, "공손찬을 놓쳤다·백마의종을 꺾었다·원군이 왔다" 자축과 총공격 명령은 B07 2라운드 `all-out` 이벤트의 전장 대사로 옮김 — 개전 컷신이 보여 주기 전에 결과를 말하던 순서 뒤집힘 · 영향받는 문서: conceptual_scenario.md(구현 메모)
- `story09.json`, `conceptual_scenario.md`: 공손찬 "포로는 돌려보내지" → "사자는 돌려보내지"(잡은 포로가 없고 돌려보낼 사람은 사자 진진)
- `story02.json`, `conceptual_scenario.md`: 손견 "도적의 북쪽 진" → "남쪽 진"(1장은 북쪽이 강, 유비가 불 놓은 곳은 남쪽)
- `story05.json`: 의천검 `grant`를 조조가 칼을 건네는 지문·대사 뒤로(획득 표시가 먼저 뜨던 순서)
- `stage01.json`: 파재·유비의 부르는 대사가 일기토 첫 줄과 겹치지 않게("불을 놓은 놈, 썩 나와라!" → "썩 나와라!", "탁현의 유비가 여기 있다!" → "그대의 상대는 나다!"), 장비 일기토 첫 줄이 돌격 컷신의 "연인 장익덕이 여기 있다"를 되풀이하지 않게
- `stage01·03·04·05·06·07.json`: 장비 일기토 대사의 "~랑께"를 이야기 말씨 "~유·~슈"로 — 화면마다 다른 지방 말을 쓰던 것

## 2026-10-21 장면(scene) 용어, 장면 캡처 데모

- `AGENTS.md`: 이야기 화면에서 전투 사이에 보이는 장면 맵·배우·엑스트라·비트·카메라를 "장면(scene)"이라 부른다고 적음 — 사용자 "스토리 영역을 구성하는 모습을 장면, scene이라고 칭한다"
- `internal/gui/uidemo_scenenode.go`, `uidemo.go`(`Demo.replace`): `-ui scenenode -stage <노드,...>`가 지정한 이야기 노드의 장면을 줄마다 재생 — 장면 개선안을 실제 화면 캡처로 비교하려고. 엔진에 노드로 건너뛰는 길이 없어 관찰값의 노드만 바꿔 그린다
- `.claude/skills/scene-staging/SKILL.md`, `check_scenes.py`: 장면 연출 스킬(이야기를 감독처럼 읽기 → 맵 고르기 → 콘티 → scenes.json → 배치 검사 → scenenode 캡처)과 배치 검사 스크립트(배우 1칸 미만·바닥 밖·소품 위·맵 밖). 지금 데이터에서 714건 — 사용자 "다음에 스토리 할 때도 비슷한 기조로 할 수 있도록"
- `.claude/skills/scene-staging/SKILL.md`: 장면 앞뒤 전투의 이벤트·일기토 대사를 이야기와 이어 읽고 중복·순서·생사·지급 순서·말투를 맞추는 단계 추가 — 사용자 "전투 시의 스토리랑 장면에서의 스토리랑 안 맞는다"(B02 화웅 이중 사망, 9장 순서 뒤집힘 등)

## 2026-10-20

### 유비 대 파재 일기토, 지원 병법 경험치 레벨 보정
- `stage01.json`: 유비 대 파재 일기토(파재 퇴각) 추가, 파재 `DuelFirst` — 사용자 제안. 일기토 경험치가 관우·장비에게만 가서 유비가 기준 레벨보다 처지던 문제(B01 유비 +1.0 → +2.0)
- `internal/core/growth.go`(`supportXP(u, q)`), `api.go`, `combat.go`: 회복·버프 경험치에 행동 경험치와 같은 레벨차 보정(대상 레벨 − 시전자 레벨) — 사용자 "아군 레벨이 높으면 보정해 경험치도 높여줬으면" · 시험: `core/defense_xp_test.go` · 영향받는 문서: logic-growth.md
- 서툰 AI 캠페인 시험은 5판 중 3판 승(기준 4판 미달). stage_lv_balance.md 3.2에 원인을 적음

### 일기토 이동, 전장 대사 넘기기, B02 손견군, B03 샛길
- `internal/gui/command.go`: 일기토 상대 옆 칸을 고르면 확인 창 없이 바로 이동하고 걸음과 일기토를 재생(`moveIntoDuel`이 `request`로) — 사용자 "일기토 메시지가 떴는데 정작 일기토는 없었어", "팝업 지워죠". 원인: 확인 뒤 명령만 보내고(`send`) 재생하지 않아 엔진에선 일기토가 났지만 화면엔 안 보였다
- `internal/ai/ai.go`: `DuelFirst`는 이번 차례에 닿는 일기토 칸이 있을 때만 그리로, 아니면 평소대로 — 사용자 "일기토 위해 무작정 달려오는 거 이상해" · 영향받는 문서: logic-battle-event.md 5.4, data-scenario-node.md
- `internal/gui/battle_speech.go`, `player.go`, `cutscene.go`: 전장 대사는 클릭하면 빨리, 다 나온 뒤 클릭하면 다음 줄로. 끝난 말풍선은 1초에 걸쳐 흐려짐(`dropSpeech`, `Player.Fading`, `nextLine`) — 사용자 요청 · 영향받는 문서: presentation-replay.md
- `stage02.json`, `story03.json`: 손견군이 퇴각(`march`)하지 않고 함께 싸움, 대사도 그렇게. 화웅 공세 3 → 6라운드 — 사용자 요청
- `characters.json`: 이각을 경보병으로(자기 그림 없이 경기병 병종 그림을 쓰던 장수, 이동이 길어 달려오던 문제) — 사용자 제안
- `tools/mapgen/layouts/hero/map03.txt`, `stage03.json`: 호로관 15×26, 동쪽 샛길. 이유 무리는 관 앞을 지키다 이각이 쓰러지면 출진, 전진 병력이 전멸하면 목순·무안국이 샛길로 우회해 나타나고 여포는 다음 라운드에 진격(`lubu-ready`), 동맹군 일기토 뒤 유비·장비·관우 대사 교체 — 사용자 요청 · 영향받는 문서: logic-battle-event.md 8절, conceptual_scenario.md 4장

### 진영별 차례, 등장 카메라, 재생 시점의 기록
- 결정: 차례를 플레이어의 군 → 아군 편 AI 진영 하나씩 → 적 진영 하나씩으로. 진영은 스테이지 `Armies`(진영색, 이름, 플레이어 편 여부)로 정한다 · 이유: 사용자 "조조&손견&황보숭군이 동시에 움직이면 동맹군을 포커싱해도 움직임이 안 들어옴, 진영색도 그럴싸하게" · 영향받는 문서: logic-battle-event.md 1절, data-scenario-node.md, presentation-ui.md
- `internal/content`(`Stage.Armies`, 검증), `internal/core`(`State.Army`, `Unit.ActsIn(turn, army)`, `armyOrder`·`nextArmy`, `turn` 이벤트 `Text`), `internal/session`(`ResolvePhase`는 한 진영), `internal/ai`, `assets/scenario/hero/stage01~07.json`(`Armies`): 위 결정의 구현. `Army`가 빈 AI 차례(옛 저장·시험)는 그 편 전체
- `internal/gui/turn_banner.go`, `hud_panel.go`: 차례 배너와 HUD 머리를 진영 이름·진영색으로(황보숭군의 차례, 조조군의 차례)
- `internal/gui/enemy_camera.go`: AI 진영 카메라를 동맹군 차례에도 — 사용자 "동맹군이 움직이는 것도 자동으로 포커싱"
- `internal/gui/player.go`(`perform`), `field_state.go`(`areaCenter`): 병력 등장·지형 변화가 이어지면 그 장면 전에 카메라가 먼저 그곳으로 — 사용자 "병력이 나오는 걸 몰라서 화면이 멈춰 있는 줄"
- `internal/gui/battle_log.go`, `player.go`, `cutscene.go`: 대사·병력 등장 기록을 재생이 그 장면에 닿을 때 씀 — 사용자 "조조가 나오지 않았는데 조조 대사가 일람에 찍힘"
- `stage01.json`: 컷신에서 유비가 남쪽 울타리 통로 앞(17,21)까지 나아감, 출격 5라운드 / `stage02.json`: 화웅만 나서고 이숙은 관문을 지킴 — 진영별 차례로 B01이 짧아지며 유비 성장이 다시 처져서(단순 AI 캠페인 시험: 5판 중 4판 이상 승, stage_lv_balance.md 기준과 맞춤)

### 레벨 정책, 텍스트 CLI, B01·B03 다시 짜기
- 결정: 스테이지 레벨 정책 — 주력은 스테이지 n에 3 + 2×(n − 1), 적 병사 L 또는 L − 1, 적 장수 L + 1, 총대장 L + 2~3(거물 L + 5까지), 동맹 장수 L + 1. 한꺼번에 붙는 적 수와 무리 나누기까지 정했다 · 이유: 사용자 기획 · 영향받는 문서: stage_lv_balance.md(새), AGENTS.md(스테이지 편집 지침)
- 결정: B01 이야기를 플레이어 조작 없이 이벤트로 끝까지(등장 → 작전 대화 → 관우·장비가 울타리로 가서 불 → 번짐 → 조조·손견 → 출격 → 퇴각). 등무·정원지 일기토는 그대로 살림 · 이유: 사용자 "a~f까지는 조작이 아니라 이벤트로", 옛 B01 일기토 살리기 · 영향받는 문서: logic-battle-event.md 8절, conceptual_scenario.md 1장
- 결정: 가장 가까운 적만 치는 서툰 플레이어(`ai.RushNext`)로 B01~B03을 이기게 · 이유: 사용자 "단순 AI로 3스테이지 정도는 쉽게" · 시험: `ai/rush_test.go`(시드 1~5, 성장 이어받음)
- `assets/data/characters.json`, `internal/content`(`Character.Points`), `internal/core/engine.go`: 유관장 Lv3·특성치 200으로 시작 — 사용자 제안
- `tools/mapgen/layouts/hero/map01.txt`, `stage01.json`: 26×26으로 넓혀 성과 울타리 사이 띠를 여섯 칸으로, 울타리마다 통로 두세 곳(사용자: 울타리가 좁아 동맹군이 의미 없음). 울타리 불은 꺼지지 않고(사용자: 화공이 너무 빨리 꺼짐) 진영 안에 불씨(차례 시작 시 병력 20% 손실)를 군데군데. 불이 난 진영은 사기·순발·방어 약화(사용자 제안). 레벨은 정책대로(조조 Lv14 등 동맹 장수 Lv4로), 조조·손견군은 장수를 앞줄에 세운 세 줄 진형(사용자: 삼각형으로 빼곡함). 불에 쫓긴 남쪽 병사 셋이 유관장에게 덤벼 유비도 싸울 상대가 생김, 출격은 4라운드
- `tools/mapgen/layouts/hero/map03.txt`, `stage03.json`: 호로관을 좁고 긴 골짜기(11×26)로, 서량 전진 병력을 선봉·이각·이유 세 무리로 나눠 `react`로 길목에 세움. 여포가 나서면 이유 무리는 관으로 물러남 — 사용자 제안(맵을 좁고 길게, 닿으면 공격, 이야기 이벤트)
- `stage02.json`: 정책 레벨에 서량 기병 셋(L − 1) 돌격, 보병 넷(L − 2) 화웅 곁 대기 — 측정해서 정함(stage_lv_balance.md 3.2)
- `internal/content/content.go`, `internal/ai/ai.go`: 정책 `react` — 이동+사거리 안에 상대가 들어올 때만 움직여 친다. `AIPolicyVersion` `greedy-10` · 영향받는 문서: logic-battle-event.md 5.4, data-scenario-node.md, system-save.md
- `internal/ai/ai.go`: `DuelFirst` 장수는 일기토 상대가 살아 있는 동안 상대 쪽으로만 걷고 아무도 치지 않음 — 화웅이 관우 대신 유비를 쳐서 군주가 퇴각하던 문제
- `internal/core/formation.go`: 일기토의 `Supports`가 맞닿아도 일기토가 벌어지고, 주역이 퇴각했으면 첫 지원군이 대신 치름 — 호로관의 관우·유비 선공 연출이 데이터에만 있던 문제, 장비가 먼저 퇴각하면 여포를 이길 길이 없던 문제 · 영향받는 문서: presentation-duel.md, data-scenario-node.md
- `internal/gui/duel.go`: 먼저 나선 장수를 일기토 화면 앞에 세우고 주역은 지원군 자리로 — 변형 연출의 대사와 그림이 어긋나던 문제(전부터 있던 것)
- `internal/gui/game.go`(`replayOpening`), `view.go`: 전투를 연 명령(출전·배치 완료)이 낸 첫 차례 이벤트도 재생하고, 첫 작전 개요 창을 닫을 때까지 재생을 멈춤 — 사용자 "불이 붙는 걸로 시작하는 게 아니라 모션이 있어야지". 원인은 이 경우 GUI가 재생 없이 끝 상태를 바로 그리던 것 · 영향받는 문서: presentation-replay.md
- `internal/gui/view.go`, `game.go`(`briefed`): 첫 차례 재생이 기다리는 작전 개요 창은 재생이 남아 있어도 그리고 클릭으로 닫게 함 — 사용자 "1스테이지 전투 시작하면 아무것도 안 하고 가만히 있음". 창은 재생이 끝나야 그려지고, 재생은 창이 닫히길 기다려 서로 멈춰 있었다. 앞서 확인 데모가 창을 코드로 닫아서 놓쳤고, 이번엔 창이 등록한 클릭 버튼을 눌러 확인함
- `internal/gui/player.go`, `field_state.go`: 재생 중 지형 상태는 재생 전 모습을 들고 있다가 `terrain` 이벤트 차례에 바꿈, 전투 대사 여러 줄은 한 줄씩 — 사용자: 조조·하후돈, 손견·정보가 동시에 말함
- `internal/gui/field_night.go`: 밤에 유닛 둘레도 밝힘 — 사용자: 너무 어두워 누가 있는지 모름 · 영향받는 문서: presentation-ui.md 밤
- `internal/gui/dialogue_scroll.go`, `officer_intro.go`: 이야기(대화창·장수 첫 등장)는 인물 자신의 일러스트만, 병사·일러스트 없는 장수는 얼굴 없음(`storyPortrait`) — 사용자 요청 · 영향받는 문서: presentation-scene.md 초상
- `internal/cli/play.go`·`ascii.go`·`scenes.go`(새), `cmd/srpg-cli`: 번호 메뉴로 하는 텍스트 플레이. 장면·전투를 칸당 네 칸 글자 지도로, 대상은 번호와 능력치·예상 피해로. `-slot`으로 실행 사이를 이어 감 — 사용자 요청(에이전트가 창 없이 배치·이야기·난이도를 확인) · 영향받는 문서: system-architecture.md
- `cmd/_balance`(새, 임시): 정책 레벨로 스테이지를 시드마다 돌려 승률·성장·퇴각을 냄. `internal/ai/rush.go`(새): 서툰 플레이어 AI
- `.claude/skills/stage-debug`(새): 수치(`_balance`) → 텍스트 플레이(`srpg-cli`) → 화면 캡처 세 단계 스테이지 점검 스킬 — 사용자 요청, 직접 써 보고 등록

### B01을 "포위를 불로 뚫는 밤"으로 다시 짬
- 결정: 장사성을 맵 가운데 두고 남·서·동 세 진영이 에워싼 밤(북쪽은 강)으로 바꾸고, 유관장의 화공 → 조조·손견의 화공 → 성문 출격 → 황건 붕괴를 이벤트로 잇는다. 마을·서복(서서 복선)은 뺀다 · 이유: 사용자 기획. 옛 B01은 성이 서쪽 끝에 있어 포위가 보이지 않았고 마을 구도와 이야기가 따로 놀았다(약탈대 구도도 사용자가 어색하다고 봄) · 영향받는 문서: conceptual_scenario.md 1장, logic-battle-event.md 8절, logic-campaign.md 4절, data-map.md, data-scenario-node.md, presentation-ui.md(밤)
- `internal/content`(`Stage.Forced`·`Night`·`Lights`, `State.Grounds`), `internal/core/engine.go`, `deployment.go`: 강제 출전은 `start`가 `DeploymentPlan`대로 놓고 전투로 들어감, 전투 중의 `DeploymentPlan`은 빈 명령열(호출처 20여 곳이 그대로 동작) — 사용자: 강제출전 유관장, 배치 없이
- `assets/data/game.json`·`terrain.json`: 상태 `O` 열린 성문(성문 땅에만, 비용 1) — 땅을 바꾸지 않고 상태만으로 성문을 열려고. 맨 위 층이 비용을 정하는 규칙 덕에 엔진 변경 없이 지나갈 수 있다. 탄 울타리는 울타리 비용 0이 그대로라 못 지나간다(사용자 결정)
- `internal/gui/field_night.go`(새), `field.go`, `field_native.go`: 밤 어둠과 불·횃불 빛, 밤에는 HP 막대를 어둠 위에, 열린 성문은 문루 없이 바닥(`gatehouse_floor_*`)
- `assets/scenario/hero/maps/map01.json`(`tools/mapgen/layouts/hero/map01.txt`), `stage01.json`, `story01.json`, `assets/scenes.json`, `characters_story.json`: 새 맵(20×20)과 이벤트, 성루·황건 진영 장면, 병서는 늘 받음, 조무 추가·서복 삭제. 승리는 붕괴 플래그와 파재 격파 — 파재가 진입로 가까이 있어 불을 놓기 전에 끝나 버리지 않게
- 시험: `core/deployment_test.go`(강제 출전), `core/campaign_content_test.go`(`TestChangsheSiegeUnfolds`: 불 → 번짐 → 조조·손견 → 출격·퇴각 → 승리)

### 대화창을 줄이고 대사를 한 글자씩
- `internal/gui/dialogue_scroll.go`: 두루마리 도트 배율 4→3, 글자 34→26, 높이 UI 24%(150~190), 초상 최대 200, 이름패도 함께 줄임 — 사용자: 대화창과 글씨가 너무 크다
- `internal/storage/storage.go`, `gui/scenario.go`, `gui/windows.go`: 게임 화면용 `InstantText`(기본 꺼짐)를 새로 두고 대사를 45ms에 한 글자씩. `TextDelayMS`는 CLI의 글자 사이 대기로만 남김 — 사용자: 글씨가 팍 뜨지 말고 말하듯 한 글자씩. 저장된 설정의 `text_delay_ms: 0`이 기본값으로 쓰여 있어 기본값만 바꿔서는 반영되지 않아 새 항목으로 뺐다
- `internal/gui/scenario.go`, `game.go`: 줄 시작 시각을 게임 틱 대신 UI 초(`lineAt`)로 — 말풍선과 같은 시계를 쓰고, 틱을 올리지 않는 데모에서도 찍히게
- `internal/gui/scenario.go`(`typing`, `hurry`), `battle_speech.go`(`rushSpeech`), `game.go`: 찍히는 중에 클릭하면 남은 글자를 8배 빠르게(장면 대사는 한 번 더 누르면 다 나옴) — 사용자: 아무 데나 클릭하면 8배쯤 빨리. 한 번에 채우던 것보다 말하는 느낌을 남기려고
- `internal/duelview/bubble.go`(`DrawTyped`), `gui/battle_speech.go`: 전투맵 말풍선도 한 글자씩, 크기는 전체 문장에 맞춰 고정. 일기토 말풍선은 그대로 한 번에 — 사용자: 일기토는 실시간·급박한 연출이라 예외

### 벡터로 그리던 그림을 수제 픽셀 시트로
- 결정: 그림 역할을 하는 것(효과·입자·아이콘·장식 테두리·무늬·맵 장식)은 수제 픽셀 시트로 그리고, 맵 칸 도형·능력치 육각형·단색 채우기·단순 테두리·글자는 벡터로 둔다 · 이유: 도트 유닛 사이에서 안티앨리어싱된 매끈한 도형이 화풍이 따로 놀았다. 칸 마스크는 마름모 PNG로 찍어 봤으나 소수 배율에서 경계가 들쭉날쭉해(이미지 확대의 한계) 화면 해상도로 칠하는 도형으로 되돌렸다 · 영향받는 문서: tools-build.md, 각 원본 README
- `tools/build_effects.py`(새, `build_projectiles.py` 대신), `tools/pixelgrid.py`(새), `../tools/spritetool/assets/ver5-fx`·`ver5-outside`·`ver5-ui`(새): 스프라이트를 레이어(아스키 격자 또는 `gen.py`의 부드러운 반투명 그림)로 쌓고 레이어마다 투명도·합성(보통/빛)을 줘 보통 시트와 빛 시트로 굽는다 — 사용자 제안: 불투명으로 찍고 레이어별 알파, 부드러워야 하는 것만 Python
- `internal/gui/shots.go`(새), `effects.go`, `damage_popup.go`, `projectile.go`: 책략(회복·물·불·버프·디버프)·전투(베기·화살 16방향·크리티컬·방어막)·피격(베기 궤적·먹물·치명 고리·회복 줄기) 효과와 모든 입자, 투석 그림자를 시트로 그림. 버프·불똥처럼 색이 바뀌는 것은 흰색을 물들이고 방어막은 좌우를 뒤집어 씀 — 화살은 도트를 돌리지 않으려고 방향을 프레임으로
- `field.go`, `field_deployment.go`: 하이라이트와 배치 음영을 칸 경로(`cellPath`) 하나로 칠함, 배치 음영은 배치 칸만 남기고 화면 전체를 어둡게 — 벡터 마름모 이미지·화면 크기 마스크와 캐시를 없애고 소수 배율의 칸 이음매를 없앰
- `scene.go`, `duelview`, `field_outside.go`, `unit_bar.go`, `skin.go`, `menu_panel.go`, `hud_units.go`, `dialogue_scroll.go`: 장면 모닥불(불길 줄 + 빛)·꽃잎, 일기토 섬광·바닥 반점, 맵 밖 소나무·돌, HP바, 창 테두리(17×17을 9조각), 메뉴 아이콘·삼각 표시, 두루마리 말림 그늘·나뭇결을 시트로 그림. 창 테두리는 그림 1px = UI 1px 최근접(4배 그림을 선형 축소하면 밉맵으로 1px 선이 번졌다), 아이콘·표시는 4배 그림을 선형 축소하고 8px 여백(이웃 그림이 섞여 들었다)
- 확인: 매 단계 scratchpad 사본 두 벌의 실제 렌더를 같은 프레임으로 캡처해 비교. `_uidemo-vm` 기본 크기는 UI를 절반 버퍼에 그려 키우므로 1px 비교는 `-size 2560x1700`

## 2026-10-18

### 후보 그림을 게임에 연결: 성문·울타리·불·그을음·다리 손상, 장면 맵 14개
- 결정: 후보 그림(`assets_candidate/`) 중 옛 기둥 성문 A(`gate_*`)와 숲 속 덤불 시안은 넣지 않는다. 성문 C는 "진입 불가 성문 타일"로, 성벽 줄의 한 칸을 대신하는 새 땅 `m`(성문)으로 둔다(구조물이 아니라 땅인 까닭: 성벽 위에는 구조물이 없다는 규칙을 지키고, 방향은 이웃 성벽이 정한다). 울타리는 구조물 `z`(진입 불가), 다리의 손상·파괴는 `On: "b"`인 상태 `d`·`x`로 둔다(이벤트로 다리를 끊을 수 있다)
- `tools/build_extra.py`(새): 후보 그림 108장(+ 짚집 `hut`)을 `assets/graphics/map/extra.png` 한 장으로 묶고 불·연기 줄 6개를 `graphics/effects/`로 옮겨 `index.json`의 `extra`·`effects`에 올림 — 게임이 후보를 바로 읽게
- `assets/graphics/index.json`, `internal/gui/assets.go`: `extra`를 소품으로, `effects`를 `Assets.Flames`로 읽고 `wall_clean_NN`을 전투맵 성벽(`wall_NN`)으로 대신 씀(이음새 눈금 없는 성벽) — 성벽이 이어진 줄이 깨끗이 보이게
- `assets/data/game.json`, `terrain.json`, `mapeditor.json`: 땅 `m` 성문, 구조물 `z` 울타리, 상태 `d` 손상·`x` 파괴(`On`, `Look`)와 병종군별 비용(성문·울타리·파괴 0, 손상 2) — 위 결정
- `internal/content/terrain.go`, `internal/core/battle_events.go`: 상태의 `On`·`Look`, `terrain` 동작이 놓을 수 없는 칸(다리가 아닌 칸의 손상)을 건너뜀 — 풀밭에 다리 파괴가 놓이지 않게
- `internal/gui/field_state.go`(새), `field_native.go`, `field.go`: 칸의 상태 모양(`Looks`)을 필드에 알려 불씨·화염은 후보 불꽃 줄과 연기, 그을음은 바닥 덧씌움·불탄 마을·주둔지·숲으로, 다리는 손상·파괴 그림으로 그리고 성문은 성문 누각, 울타리는 이웃과 이은 울타리(칸 두 배 크기)로 그림. 옛 절차 불꽃(`fire.go`)은 지움 — 그림 불꽃으로 바꿨고 둘이 겹치지 않게
- `internal/gui/uidemo_mapfx.go`, `uidemo_scenemap.go`(새), `uidemo.go`: 데모 `mapfx`(전투맵 새 요소 모음)·`scenemap`(장면 맵 보기), `battle`이 맵을 바꾼 채 시작하는 `battleWith`, `uidemo_fire.go`가 맵 밖 칸에 불을 놓아 `CorruptSave`가 나던 것을 고침
- `tools/scene_maps/export.mjs`, `build.py`(새): 후보 viewer의 JS 장면을 Node로 돌려 그림 배치를 뽑아 장면 맵 14개(`graphics/maps/S_vil_*`·`S_tent`·`S_road`·`S_hill`·`S_army_camp`·`S_castle_out*`·`S_courtyard`·`S_rampart`·`S_office`·`S_palace`·`S_house`·`S_garden`)를 만듦 — 시나리오에서 `scenes.json`의 `map`으로 쓰게. 후보의 한 칸이 게임 소타일과 같은 크기라 네이티브 맵으로 만들어 그림 픽셀을 그대로 둠. 유닛·불꽃·하늘·밤 색은 뺌
- `internal/gui/assets.go`, `field.go`: 맵 파일의 `native`·`visual`·`small_tiles`·`scene`을 읽고 `scene` 맵은 외곽 그늘·바닥 무늬 채움 없이 그림 — 장면 맵이 끝나는 곳까지만 그려지게
- `tools/scene_maps/wire_story.py`(새), `assets/scenes.json`: 이야기의 `scene` 노드 장소 22곳을 맵에 대응시켜 대화 노드 235개를 장면 맵 위에 올리고 그 구간에서 말하는 인물을 맵의 빈 땅에 마주 보게 세움(배우, 비트는 비워 둠) — 장면 맵을 만들어 두기만 하고 `scenes`가 비어 있어 게임에서 하나도 안 쓰이던 것을 고침
- `internal/gui/scene.go`: 배우가 없는 장면의 카메라를 맵 가운데로 — 배우 평균 위치를 쓰던 계산이 배우 없이는 맵 모서리를 비추어서
- 시험: `content/terrain_test.go`(성문·울타리·다리 상태), `core/fire_test.go`(다리 상태는 다리에만), `gui/field_state_test.go`·`scene_maps_test.go`(그림 로드, 장면 맵 14개 로드·그리기)
- 영향: 이전 저장은 `Rules` 변경 없이도 콘텐츠 해시가 달라 호환되지 않는다

### 지형을 땅·구조물·상태 세 층으로 나눔
- 결정: 한 칸(대타일)이 규칙의 근간이고 `{땅, 구조물, 상태}`다. 소타일은 그림을 매끈하게 하는 표현이라 저장하지 않고 땅을 2×2로 펼쳐 그림 쪽에서 만든다(1~7장 맵의 소타일은 전부 2×2로 균일해서 잃는 정보가 없었다). 우선순위는 땅 < 구조물 < 상태이고, 이동 비용·배율·회복은 값을 말하는 가장 위층이 이기며 `Water`·`Castle`·`Ambush`는 어느 층이든 가지면 참인 조건이다. 글자 하나가 땅·구조물·상태를 한꺼번에 나타내 마을·숲 위에 불을 붙일 수 없고 그을음을 더할 수 없던 한계를 풀려고
- 결정: 구조물은 놓을 수 있는 땅이 정해져 있다(관문은 성내, 다리는 물, 나무는 초원·황무지, 마을·주둔지는 초원·황무지·성내, 성은 초원·성내). 성벽·절벽은 구조물이 없다. 불씨(영구, 배율 0.8, 차례 시작 병력 -20%, 비용은 아래층)와 화염(진입 불가, 라운드마다 줄어 끝나면 그을음)과 그을음(규칙 효과 없음)이 상태이며 이벤트로만 생긴다. 구조물을 없애는 `Burns`는 두지 않는다
- `assets/data/game.json`: `Tiles`를 `Grounds`·`Structures`·`States`로 나누고 `EdgeTile`을 3글자(`w--`)로 바꿈 — 위 결정
- `assets/data/terrain.json`: 병종군마다 `Ground`·`Structure`·`State` 표로 바꿈. 값을 말하는 칸만 적음 — 위층이 정의하면 이기고 아니면 아래층을 따르게
- `assets/data/mapeditor.json`: 브러시를 층+글자 두 글자(`gg`·`st`·`ee`…)로 바꿈 — 브러시가 자기 층만 칠하게
- `assets/scenario/hero/maps/map01~07.json`, `tools/mapgen/layouts/hero/*.txt`: 3글자 칸 형식(`MapVersion` 2)으로 다시 만듦. 옛 `Resolved` 글자와 칸마다 대조해 모두 같음을 확인함. 관문 4칸(맵 1·3·5)은 땅이 초원에서 성내로 바뀜
- `assets/scenario/hero/stage01.json`: `terrain` 동작의 `Tile`을 `State`로 바꿈 — 황건적이 놓는 불(4라운드)은 화염 `F`, 마을이 타는 불은 불씨 `e`
- `tools/mapgen/mapgen.py`: 공백으로 구분한 3글자 칸을 받고 한 글자 옛 표기는 별칭으로 풂 — 새 형식, 옛 배치 호환
- `internal/content/terrain.go`(새): `Ground`·`Structure`·`State`·`Cell`과 합성 조회(`Move`·`Props`·`CellName`·`Glyph`·`ValidCell`) — 규칙 합성을 한 곳에 둠
- `internal/content/content.go`, `validate.go`, `battle_events.go`: `Tile`·`Tiles`를 새 표로, 동작 `Tile` → `State`, 표 검증 — 새 형식
- `internal/content/map_document.go`, `map_graphics.go`, `scenario.go`: 3글자 `Cells`, 층별 `Paint`(구조물이 설 수 있는 땅만, 땅을 바꾸면 설 수 없는 구조물은 지움), `Resolved`·다수결·소타일 저장 제거, 그림용 소타일과 글자 행은 칸에서 만듦 — 소타일은 표현
- `internal/core/*.go`: `Cell`·`CellProps`로 합친 칸을 읽고, `TileOverrides`를 `TileStates`(상태만, 끝나면 `Then`)로 바꿈, `Format` 4·`rules-14`, 물 위 배치 금지는 구조물 없는 물만, `NextTurnRecovery`의 불 피해 표시(`RestHP != 0`) — 상태 규칙
- `internal/ai/ai.go`: 회복 칸 선호를 글자 비교(`k`·`v`)에서 `RestHP > 0`으로 바꿈, `AIPolicyVersion` `greedy-9` — 성·관문도 회복 칸이라서(주석이 원래 "village or castle floor"라 했다)
- `internal/gui/*.go`: 불꽃을 상태의 `Blaze`로, 정보창을 칸 기준으로(불 피해 줄 `차례 시작 병력 -N%` 추가), 에디터를 칸 단위 칠하기·층별 브러시·세 층 표시로 바꿈, `uidemo_fire`의 복원 오류(맵 밖 칸) 고침
- `internal/cli/cli.go`: 지도 출력이 칸의 구조물/땅 글자와 상태(`*`)를 보임
- `internal/testdata/mini/*`, 각 패키지 시험: 새 형식으로 옮기고 합성·상태 흐름·구조물 검증 시험(`content/terrain_test.go`, `core/fire_test.go`)을 더함
- `docs/data-map.md`, `logic-combat.md`, `logic-battle-event.md`, `data-scenario-node.md`, `tools-editor.md`, `system-save.md`, `system-architecture.md`, `todo/engine.md`, `assets/data/README.md`: 현재 동작으로 고침 — 문서가 코드와 맞게
- 영향: 이전 저장은 모두 호환되지 않는다. 숲 비용(다른 세션의 미커밋 변경)은 그대로 옮겼다

### 1~9장 일기토를 이야기와 성격에 맞춰 다시 씀
- `assets/scenario/hero/stage01.json`: 영천에 등무(북쪽 진영)와 정원지(등무가 쓰러지면 등장)를 놓고 옛 캠페인의 두 일기토(장비·관우 승, 둘 다 전사)를 되살림 — 새 시나리오로 옮기며 빠졌던 연의 1회의 두 장면이라서. 정원지의 "등무가 죽었다" 대사가 맞도록 등장 순서를 이벤트로 묶음
- `assets/scenario/hero/stage04.json`: 변수에 여포 대 하후돈(하후돈 열세·퇴각)과 여포 대 장비(무승부, 장비가 호로관보다 수세에 몰리지만 버팀)를 더하고 여포에 `DuelFirst`, 부목표를 "조조 생존"으로 줄이고 승리를 "유비만 동쪽 끝에 도달", 패배에서 조조 퇴각을 뺌 — 하후돈이 퇴각하면 "조조군 장수 전원 생존"을 못 이루고, 조조 퇴각이 패배면 "조조 생존"이 늘 참이라서
- `assets/scenario/hero/stage05·06·07.json`: 신도성 순우경(술기운·코 복선), 거록 안량(결과를 무승부에서 장비 압승으로 바꾸고 관우 대 화웅 정도의 격차), 계교 문추(조운을 언급한 오기, 장비 승)의 `Beats`를 새로 씀. 계교 오프닝은 컷신 근사를 버리고 일기토 공손찬 대 문추(무승부, 둘 다 생존)를 `duel` 동작으로 치르게 함
- `internal/content/battle_events.go`, `validate.go`, `map_document.go`: 일기토의 아군은 `Party`에 있어도, 적은 이벤트가 `Side: "enemy"`로 만들어도 되게 풂 — 하후돈·공손찬(처음부터 있는 NPC)과 여포·정원지(이벤트 등장)를 일기토에 쓰려고
- `internal/core/campaign_content_test.go`: 이벤트로 나타난 호위병을 치워 일기토 상대 옆자리를 비움
- `docs/data-speech-style.md`: 장비는 형들을 `형`(관우는 `형님`)으로 부르고 방언은 살짝만, 하후돈·순우경·안량·문추·공손찬 말투와 여포의 `애꾸` 호칭을 더함
- 영향: `docs/conceptual_scenario.md`, `data-scenario-node.md`, `todo/data.md`

### 숲을 지나갈 수 있게 하고 병력이 나무 사이에 서도록 그림
- `assets/data/terrain.json`: 숲(`f`)의 비용을 0에서 보병·궁병·서생·적병 그룹 1, 기병 그룹 2로 바꿈. 발석거는 궁병 그룹에 묶여 있어서 `machine` 그룹을 새로 만들고(궁병 표 복사, 숲 비용 2·배율 1.1) `classes.json`의 `catapult`가 쓰게 함. 그룹 이름은 `siege`로 하려다 엔진 소스에 게임 내용을 쓰지 않는 시험(`purity_test`)이 병종 태그 `siege`와 겹쳐 막아서 `machine`으로 함
- `internal/gui/field.go`·`field_native.go`: 숲 그림의 깊이값을 칸 중심보다 0.1 뒤로 옮겨 숲 칸의 병력보다 나중에 그림 — `assets_candidate`의 "숲 속 병력" 시연 중 ② 나무 사이(반투명)를 고름. 가려진 부분의 반투명은 기존 규칙
- `internal/content/forest_terrain_test.go`: 숲 비용이 병종 태그(`mounted`·`siege`)와 맞는지 시험
- 영향: 숲 비율이 높은 맵 2·4·6은 길이 크게 열림. 매복·스테이지 설계는 아직 안 맞춰 봄

### 스테이지 번호를 1부터 이어서 다시 매김
- `assets/scenario/hero/stage03~09.json`·`maps/map03~09.json`·`tools/mapgen/layouts/hero/map03~09.txt`: 이야기 번호(1·3·4·5·7·8·9)를 따르던 이름과 ID를 `B01~B07`·`stage01~07`·`map01~07`로 바꿈, 전투 노드(`story03·04·05·07·08·09`)의 `Stage`와 변수 추격전의 `flag-b05-lubu`를 `flag-b04-lubu`로 맞춤 — 스테이지 2가 없는 번호 구멍을 없애려고(스토리 번호는 그대로)
- `internal/*/coalition_test.go`·`gui/*_render_test.go`·`core/campaign_content_test.go`·`cmd/_m8check`·`docs/data-map.md`: 새 ID에 맞춤
- 영향: 옛 저장은 `ContentHash`로 어차피 거부됨

### 시나리오 1~9장을 데이터로 만들고 옛 캠페인을 교체함
- 도원결의·사수관·호로관 세 판짜리 옛 캠페인을 지우고 `conceptual_scenario.md`의 1~9장(영천~계교)을 장 9개·스테이지 7개·맵 7개로 만듦. 맵은 글자 배치에서 생성(`tools/mapgen`). 새 인물 26명(`characters_story.json`)과 화계서(`fire_manual`)를 더함. 책략 「화계」 보상은 부목표 플래그를 전투 뒤 `branch`가 읽고 `grant`로 장비를 주는 방식이라 새 노드 종류 없이 해결함. 일기토 연출은 새로 쓴 것 대부분이 기본 연출이고 호로관·사수관의 것만 옛 연출을 이어씀.
- 호로관의 삼영전을 일기토 하나(`hulao-brothers`)와 `Variants`(먼저 맞닿은 장수)로 합침 — 일기토가 인접하는 순간 자동으로 벌어져 셋으로 나눠 두면 셋이 따로 벌어질 수 있어서.
- 영향: `assets/scenario/hero/`, `assets/data/characters_story.json`, `internal/core/campaign_content_test.go`, `tools/mapgen/`, `docs/todo/data.md`

### 전투 연출용 이벤트 동작(`buff`·`join`·`move`·`cutscene`·`duel`)을 둠
- 전투 도중 능력치 보정(손견군 약화)·조작 가능한 인물의 합류(조운)·이동력과 무관한 여럿 동시 이동(백마의종)·전투 위의 연출(`cutscene`: 묶음별로 동시/순차, 병력·병법은 바꾸지 않고 끝 위치만 남김)·인접하지 않은 일기토를 지금 치르기. 연출은 전투 화면 위에서 기존 병종 동작(`attack` `hit` `block` `exhausted`)으로 하고 새 스프라이트가 필요 없음. 일기토를 명령 종류와 상관없이 앞 동작 뒤에 보이도록 재생을 고침(자동 일기토가 이동 뒤에 나오도록).
- 영향: `internal/core/battle_events.go`, `internal/gui/player.go`·`cutscene.go`, `docs/logic-battle-event.md`·`presentation-replay.md`

### 불을 타일 두 종류와 `terrain` 동작으로 표현함
- 불붙은 땅(`F`: 모든 병종 배율 0.8, 자기 차례가 시작될 때 선 병력은 최대 병력의 20%를 잃음, 지나가기만 하면 피해 없음)과 불길(`W`: 진입 불가). 불이 붙는 것은 이벤트 `terrain` 동작이 칸의 타일을 바꾸는 것으로만 일어나고, 변경은 상태에 저장되며 `Turns`로 꺼지게 할 수 있음(병법이 나중에 같은 수단을 쓸 수 있게). 불이 번지는 규칙은 두지 않음.
- 영향: `internal/core/battle_events.go`·`combat.go`·`rules.go`, `internal/gui/fire.go`, `assets/data/game.json`·`terrain.json`

### 유닛 행동 정책(`hold`·`assault`·`march`·`flee`)과 `ai` 동작을 둠
- 스폰의 `Guard`(제자리)와 스테이지 `Charge`(적이 처음부터 진군)를 유닛별 `Policy`로 바꾸고 `march`(목표 칸으로 가며 교전)·`flee`(목표 칸으로 달아나 맵을 떠남)를 더함. 이벤트 `ai` 동작이 정책을 바꿈. `charge`라는 이름은 데이터의 특성 ID와 겹쳐 엔진 소스에 못 써서 `assault`로 했고, `flee`가 맵을 떠나는 것은 AI가 아니라 엔진 규칙(`retreat`)이라 `defeated` 조건으로 볼 수 있음. 정책을 이벤트가 바꿀 수 있어 저장 검증은 정책의 모양만 봄.
- 영향: `internal/ai/ai.go`, `internal/core/engine.go`·`battle_events.go`, `docs/logic-battle-event.md`·`data-scenario-node.md`

### 일기토를 인접하는 순간 자동으로 벌이고 연출을 스테이지 파일에 합침
- 공격이 일기토로 바뀌던 방식을 없애고, 지정한 두 장수가 인접한 상태가 되면(이동·소환·개전 어느 경우든) `settle()`의 첫 단계에서 벌어지게 함. 결과는 지금처럼 데이터로 고정. 미리보기 이동에서는 일기토를 보일 수 없어서 일기토가 벌어지는 칸으로 이동할 때 확인 창을 한 번 거침(`DuelAt`). 이동해서 맞닿은 장수(`First`)에 따라 연출 변형(`Variants`)을 고름.
- `assets/duels.json`을 없애고 연출(`Beats`·`Supports`)을 스테이지 파일의 일기토 항목에 합침 — 규칙 결과와 연출이 한 곳에 있어 어긋나지 않게. 연출의 마지막 장면이 `Outcome`과, 떠나는 장수의 결과(`retreat`)와 맞는지 로드 때 확인함.
- 영향: `internal/core/formation.go`, `internal/duel/scripts.go`, `internal/gui/command.go`·`duel.go`, `docs/logic-duel.md`·`data-scenario-node.md`·`presentation-duel.md`

### 승패를 목표 목록으로 바꾸고 부목표와 필수 출전을 둠
- `Boss`·`Threshold`·`Goal`·`Limit`를 없애고 스테이지가 `Victory`·`Defeat`(`[{If, Text}]`)·`Sub`·`Required`·`Max`를 가지게 함. 목표의 조건은 전투 이벤트 조건 문자열을 그대로 써서 도달(`at`/`allat`)·점령(`at:enemy:구역`)·전멸(`alldefeated:enemy`)·버티기(`round>=N`)에 별도 어휘를 만들지 않음. 군주 퇴각과 아군 전멸은 `Defeat`와 무관하게 늘 패배. `TurnLimit`은 "마지막 라운드까지 이길 수 있다"는 규칙이 달라 그대로 둠.
- 부목표는 승리 순간 판정해 보상과 이야기 플래그를 줌. 책략 습득 같은 보상은 플래그를 읽는 전투 뒤 노드가 맡음.
- 영향: `internal/core/engine.go`, `internal/content/battle_events.go`, `internal/gui/hud.go`·`windows.go`, `docs/logic-battle-event.md`·`data-scenario-node.md`

### 전투 이벤트를 조건과 동작(`If`/`Do`)으로 통합함
- 옛 `Trigger`(`boss-alone`·`duel`)와 `Spawns`·`Dialogue` 필드를 없애고, 이벤트가 상태 조건(`round` `turnstart` `defeated` `at` `adjacent` `hp` `duel` `flag` `any` …)과 동작(`say` `spawn` `remove` `flag`)을 갖게 함. 스폰에 `Group`·`Side`, 스테이지에 번호 붙은 `Areas`. 시점 필드 없이 `turnstart:진영` 조건으로 "한 차례를 버텼다"를 표현하고, 전투 중 `flag`는 승리 때 합쳐지는 `PendingFlags`에 적음. 반복 이벤트(`Repeat`)는 조건이 거짓에서 참이 될 때만 발동하고 `spawn`은 쓰지 못하게 함 — 저장 검증이 이벤트 스폰을 기준으로 삼기 때문.
- 영향: `internal/core/battle_events.go`·`cond.go`, `internal/content/battle_events.go`·`cond.go`, `docs/logic-battle-event.md`·`data-scenario-node.md`

### 민중 도트 3종(남자·여자·노인) 추가
- `tools/doteditor/tools/mounted_kits/commoners.py`(신규): 서생 몸체에 베 튜닉·바지/치마·맨발·등짐을 입힌 무기 없는 민중 키트 3종(`commoner_man`·`commoner_woman`·`commoner_elder`). 남자는 수건 두른 상투, 여자는 쪽머리 없이 어깨까지 내려오는 머리, 노인은 회색 줄무늬 백발에 흰 수염과 지팡이 — 진영색을 쓰지 않아 스테이지 진영에 물들지 않게 하고, 셋을 한눈에 구분하려고
- `tools/doteditor/units/commoner_{man,woman,elder}.yaml`(신규): idle·walk·hit·exhausted·block 변환 결과(attack 없음)와 태그 색 — 비전투 시트도 같은 파일을 쓰려고
- 결정: 아이 손잡기·업기와 할머니는 뺌 — 아이가 5px 기둥처럼 읽혀 어색했고 할머니는 여자와 구분이 안 돼서
- `assets/sprites/commoner_{man,woman,elder}.spr`, `internal/sprite/testdata/units/commoner_*.png`(신규): 위 도트를 빌드한 결과와 비교용 PNG — 스프라이트 시험이 .spr마다 PNG를 요구해서
- `assets/data/characters.json`: 템플릿 캐릭터 `troop_commoner_man`·`troop_commoner_woman`·`troop_commoner_elder`(민중, 경보병 병종, 맨손)를 추가 — 스테이지·장면에서 `Character`로 바로 세울 수 있게

### 시나리오를 `assets/scenario/`로 옮기고 장 파일과 스테이지·맵 파일로 나눔
- `assets/data/script.json`·`maps/B0x.json`을 `assets/scenario/<시나리오>/`의 `scenario.json`(`Campaign`+`Chapters`)·`storyNN.json`(장의 노드)·`stageNN.json`(스테이지)·`maps/mapNN.json`(지형)으로 나눔 — 장 흐름이 한 곳에서 보이고, 큰 `Cells`가 사람이 고치는 스테이지 규칙과 섞여 diff를 해치지 않게. 스테이지 ID는 그대로(`B01`)이고 파일은 접두어를 뗀 번호. `Stage.Next`는 없애고 노드 `Next`와 장 순서(`Data.After`)가 대신함. 옛 데이터와 같은 시드의 AI 완주 상태 해시가 같음을 확인.
- 영향: `internal/content/scenario.go`, `internal/mapedit/store.go`, `docs/data-scenario-node.md`·`logic-campaign.md`·`data-map.md`

### 스토리 노드를 늘리고 플래그를 둠
- `say`·`narrate`·`scene`·`set`·`branch`·`end`를 더하고 `preparation`을 `battle`로, 문자열 안의 `이름:` 파싱(`dialogue`)을 `Speaker` 필드로 바꿈. `State`에 `Flags`·`Choices`·`Place`(저장 `Format 3`). 분기 조건은 문자열(`flag:` `alive:` `choice:노드=옵션` `won:` …)로 적고 로드 때 파싱·검증함.
- 영향: `internal/core/engine.go`·`cond.go`, `internal/content/cond.go`, `docs/todo/engine.md`

### 후보 그림(assets_candidate) 첫 묶음
- `assets_candidate/`: 새 폴더(게임 자산은 건드리지 않음). 닫힌 관문, 불탄 숲·나무·마을·주막, 그을음 3단계, 울타리 16종, 손상·파괴 다리, 화로·지도 탁자, 불길·연기 프레임과 장면 4개 시연(`viewer.html`) — 불·연기·그을음·울타리·관문·다리로 시나리오 장면을 꾸미려면 무엇이 필요한지 눈으로 보고 고르려고. 기존 그림을 바탕으로 만들어 도트 스타일을 맞춤. 이어서 전투맵(데포르메)·장면 크기를 나눔: 성문은 주둔 가능한 관문 타일과 구분되게 새로 그린 한 칸 성문으로(바닥 + 통로 앞뒤 끝의 문틀 두 개, 문 위는 비움 — 열리면 병사가 걸어 지나가도 어색하지 않게). 문짝에 두께와 외곽선을 주고 안쪽으로 열려 옆 성벽에 붙게 한 뒤, 다시 슬레이트 지붕을 얹되 기둥을 32px로 높여 기병이 지나갈 높이로 함(바닥·뒤 문틀·앞 문틀·지붕 4층). 도개교 성문은 어색해서 접고, 처음 형태(돌 블록 + 아치 + 지붕)를 주변 성벽 높이로 낮춘 성문 C로 대체(열리면 그림이 사라짐). 성문 C는 성벽 블록 그림 위에 만들고 아치를 반원 머리로 고침 — 바닥 외곽선이 안쪽으로 들어가 보이고 문 오른쪽 위가 눌려 보이던 것을 바로잡음. 성벽 칸 이음새의 세로 눈금을 지운 성벽 `wall_clean_*`도 추가. 시나리오 장면 후보 9개(길·언덕·진영·성 밖·성 안·성루·관아·궁궐·동승의 집·조조의 후원)와 새 소품 11종(병풍·침상·등롱·휘장·계단·정원석·매화·정자·이정표·향로)을 추가 — 시나리오 52장면의 장소를 전부 덮기 위해. 성루는 바닥 전체를 돌 통로로 하고 오른쪽 위 가장자리만 성가퀴로, 길은 길 방향을 따라 걷게, 관아·궁궐은 단상과 융단 중심으로 배치를 다시 짜고 휘장을 벽 방향에 맞춤. 이어서 모든 실내·마당·진영 배치를 isometric의 두 축(오른쪽 아래·왼쪽 아래 대각선)에 맞춰 다시 짬: 옥좌 단상 `throne_*`를 새로 그려 오른쪽 위 벽에 붙이고 융단·기둥·신하를 단상에서 왼쪽 아래로 줄세움, 병풍은 방향이 벽과 맞게 고침 — 화면 세로 대칭 구도가 어색했음. 성 안 관청(`hall_*`)·매화·태호석·정자는 2D로 그린 것이 어색해서 `tools/unit3d` 셀 렌더러로 3D 모델링해 다시 그림(`tools/gen_3d_props.py`). 야외 장면은 지형 규칙을 화면 끝까지 이어 그리고 둘레에 나무를 흩어 검은 여백을 없앴으며, 성루·성 안에는 하늘(낮·저녁·밤) 배경과 화면 끝까지 이어진 돌바닥을 넣고, 후원은 정자를 가운데로 옮김. 성 밖은 성벽을 화면 끝까지 늘리고 앞에 해자·나무다리, 뒤에 성 안(돌바닥·관청들)이 보이게 다시 짬. 이어서 장면용 큰 성벽 세트(성가퀴 달린 성벽·성문 누각)를 3D 모델로 새로 만들어 성 밖에 쓰고(성벽 두께는 한 칸 전체로 성문과 면을 맞추고 성가퀴는 칸 안에서 온전히 끝나게(칸마다 하나·활구멍), 조각은 그 칸의 앞면 기준으로 잘라 성문과 이어지는 곳이 비지 않게, 성문은 네 칸 폭에 통로 두 칸·가로로 반씩 두 조각(왼쪽 반은 유닛 앞, 오른쪽 반은 유닛 뒤에 그려 통로 안 병사가 오른쪽 기둥에 가림; 통째로 그린 그림을 화소 단위로 나눠 잘린 곳에 외곽선이 없음), 다리는 전투맵 한 칸 그대로에 해자 두 칸 폭), 성 안·성루는 기존 전투맵 성벽이 어울려 되돌림, 성내 큰길 타일과 격자로 가지런한 관청 배치를 넣음 — 전투맵 성벽은 데포르메되어 낮고 성문 열림 쪽 끝 면도 깨졌음, 불꽃은 좁은 불꽃 여러 개 타원 배치로, 울타리는 전투맵용 낮은 것과 장면용 두 벌로 다시 만듦. 숲 칸에 선 병력의 표현 시연 "숲 속 병력"(현재 / 발치 덤불 / 나무 사이 반투명 / 그늘+덤불)과 덤불 `cover_bush_*` 3종을 추가 — 현재는 병력이 자기 칸의 나무 앞에 온전히 서서 숲에 있다는 느낌이 없었음.

### 설정에 경험치 배율을 추가
- `internal/storage/storage.go`: `Settings.XPPct`(60~100, 기본 80)와 검증, 선택지 목록을 `ScalePcts`로 데미지와 공용 — 레벨업 속도를 사용자가 조절하게.
- `internal/core/types.go`, `api.go`, `growth.go`: `State.XPPct`·`SetXPPct`, `actionXPAmount`·`supportXP`에서 곱함(최소 1) — 전투 행동·방어·지원 경험치와 예측 창이 같은 값을 쓰게. 이벤트 보상으로 한 칸을 주는 경우(`formation.go`)는 곱하지 않음.
- `internal/session/session.go`, `internal/gui/windows.go`: 요청마다 엔진에 동기화, 설정 창에 `경험치 배율` 줄 추가(창 높이 604).
- 테스트와 `docs/system-architecture.md`, `presentation-ui.md` 갱신.

### 초열 병법 소모 40, 이유 통솔 70, 문관 계열 회복 기본값 8
- `assets/data/skills.json`: 초열 `Cost` 50 → 40 — 첫 화계를 더 자주 쓰게.
- `assets/data/characters.json`: 이유 통솔 61 → 70 — 병법 최대치 68 → 73.
- `assets/data/classes.json`: 서생·참모·군사·방술사·선술사 `RecoveryBase` 5 → 8 — 문관 계열의 병법치 회복을 올림(이유 17 → 20). `docs/logic-combat.md` 갱신.

### 데미지 배율 선택지를 60~100%로, 기본 80%
- `internal/storage/storage.go`, `session/session.go`: 선택지 30·50·70·100 → 60·70·80·90·100, 값이 없으면 80(`DefaultDamagePct`). 세션은 `DamageScale()`을 엔진에 동기화 — 30%는 너무 느려져 쓸모가 적고 기본을 100%보다 낮게 두고 싶다는 요청.
- 테스트와 `docs/system-architecture.md`, `presentation-ui.md` 갱신.

### 2회 공격의 두 번째 타격을 ×0.7로
- `internal/content/content.go`, `assets/data/game.json`, `internal/testdata/mini/game.json`: `Rules.DoubleShare`(0.7) 추가 — 연타가 첫 타와 같은 피해를 내면 치명과 겹쳐 한 대상 3.0배까지 가서 과함.
- `internal/core/combat.go`: `attackOn`과 `counterBack`의 두 번째 타격에 곱함. 요극 추가 공격은 처음 해석(후속타 = 요극)이 틀려 되돌렸고 비율이 없다. 한 대상 최대는 1.5 + 1.5×0.7 = 2.55배(반격은 ×0.7 더).
- `internal/core/double_share_test.go`: 비율 검사. `docs/logic-combat.md` 갱신.

### 설정에 데미지 배율을 추가하고 선택지를 버튼 줄로 바꿈
- `internal/storage/storage.go`: `Settings.DamagePct`(30·50·70·100, 0은 100)와 검증 — 전투 속도를 사용자가 조절하게.
- `internal/core/types.go`, `combat.go`, `api.go`: `State.DamagePct`와 `SetDamagePct`, `damage()`에서 곱하고 최소 1 — 설정을 엔진 상태에 실어야 `Restore`한 예측·시험 복사본이 같은 피해를 내고 정렬·세이브가 어긋나지 않음. 지속 피해·회복은 곱하지 않음.
- `internal/session/session.go`: 요청 처리 끝에 설정값을 엔진에 동기화 — 불러온 세이브의 값이 아니라 현재 설정이 이기게.
- `internal/gui/windows.go`, `skin.go`: 설정 창을 `choiceRow`(이름 위·선택지 버튼 줄 아래, 고른 것은 눌린 채 금색)로 바꿈. 적 AI·전투 기록·대사 속도·재생 속도·UI 크기·데미지 배율 모두. 순환 버튼(`nextUIScale`)은 삭제 — 여러 값 중 지금 값을 한눈에 보고 바로 고르려고.
- `internal/storage/storage_test.go`, `session/session_test.go`, `core/damage_pct_test.go`: 설정 검증, 엔진 동기화, 배율 적용 테스트.

### 일람 탭을 간소화하고 병력일람 창을 추가
- `internal/gui/hud_units.go`: 일람 탭을 장수명·병력·병법치·명령 한 줄 목록으로 줄이고(보기 전환 제거), 정렬 순서를 `무정렬 → 내림 → 오름`으로 바꿈. 목록 상태를 `unitList`, 열 배치를 `unitTable`로 묶어 탭과 팝업이 같은 표 코드를 씀 — 우상단 패널(폭 320)에 6칸 두 줄은 너무 좁아서.
- `internal/gui/hud_roster.go`: 새 파일. `기타 › 병력일람` 오버레이(`roster`). 화면을 거의 채우는 창에 한 줄·큰 글씨(17)로 기본(병종·레벨·병력·병법치·명령)·전투력·열전·장비 네 보기를 보이고 같은 정렬을 씀. 행을 누르면 창을 닫고 그 장수로 이동.
- `internal/gui/hud_panel.go`, `windows.go`, `game.go`, `uidemo_hudpanel.go`, `hud_units_test.go`: 기타 탭에 `병력일람` 버튼(높이 180→220), 오버레이 크기·제목, 필드 `unitsList`/`rosterList`, 데모 프레임, 테스트 갱신.
- `docs/presentation-ui.md`: 일람 탭·기타 탭 설명을 새 동작에 맞춤.
- `internal/gui/hud_roster.go`: 필터·보기 버튼 폭을 창 폭에 맞춰 줄임 — UI 160%에서 `아군/적군/전체`와 `기본/전투력/열전/장비`가 겹쳤음.
- `internal/gui/layout_overlap_test.go`: 새 파일. 화면 5종 × UI 크기 5종 × 화면 13종(HUD 탭·메뉴·설정·작전 개요·병력일람 보기)을 그려 버튼끼리 겹치거나 화면을 벗어나면 실패 — 그리는 동안 `g.buttons`에 쌓이는 사각형만 읽으므로 네이티브 렌더러 없이 `go test`로 돎.
- `internal/gui/hud_panel.go`: 탭 순서를 미니맵·장수·일람·기록·기타로 바꾸고 `장수` 탭을 선택 없이도 고를 수 있게 함(선택 해제 뒤 돌아갈 기본 탭이 될 수 있음, 비면 안내 문구) — 장수를 기본 화면으로 두고 싶다는 요청.
- `internal/gui/events.go`: `deployment_finish` 이벤트를 기록에서 숨김 — 배치 종료마다 `deployment_finish  0`이 기본 형식으로 새어 나왔음.
- `internal/gui/hud_units.go`, `hud_roster.go`: 병력일람 창의 머리·값을 칸 가운데 정렬, 전투력의 `회복`을 `병법회복`으로 바꿈 — `회복`이 병력 회복으로 읽혔지만 실제로는 턴당 병법치 회복량(`Stats.Recovery`).

### 헌제 캐릭터 추가
- `assets/data/characters.json`: 헌제(`xiandi`)를 추가 — 경보병(전투 모습을 따로 만들 필요가 없어 일반 보병), 레벨 1, 능력치 28/14/72/33/40/78(통솔·무력은 낮고 지력·매력이 높은 황제), 장비 없음, `Look.Scene`은 `xiandi_noncombat`, 진영은 `han`. 초상은 아직 없어 병종 초상이 나온다. 사용자 요청.
- `assets/sprites/xiandi_noncombat.spr`, `internal/sprite/testdata/units/xiandi_noncombat.png`: 헌제의 장면용 비무장 도트를 빌드 — 면류관과 바닥에 끌리는 도포.
- `docs/todo/data.md`, `docs/data-character.md`: 헌제·손견이 `characters.json`에 있다고 적음. 하비 전투의 여포 백마·`DuelFirst`는 이미 `todo/data.md`에 적혀 있어 그대로 둠.

### 조조의 시작 장비를 의천검으로
- `assets/data/characters.json`: 조조의 `Equipment`를 철창에서 의천검(+가죽갑옷)으로 바꿈 — 사용자 결정. 조조가 `다재다능`이라 검을 장착할 수 있고, 검을 들었으니 검 든 도트(`caocao_sword`)로 보이며 적의 물리 공격 명중률이 15%p 낮다.

### 청공검(부채꼴 5칸)과 의천검(회피 15%p) 추가
- `assets/data/equipment.json`: 청공검(`qinggang_sword`, 검, 공격 +3)과 의천검(`yitian_sword`, 검, 공격 +4)을 추가 — 조운이 청공검을 빼앗아 드는 이야기와 조조의 보검을 장비로 만들려고. 무기 종류가 검이라 장착하면 조조·조운의 검 든 도트가 나온다.
- `internal/content/vocab.go`, `internal/core/combat.go`: 효과 키 `sweep_arc`(근접 일반 공격이 앞 칸을 중심으로 공격자를 둘러싼 5칸의 적도 비율만큼 침, `arcAhead`)와 `evade`(방어자의 회피, 물리 명중률에서 %p만큼 뺌)를 추가 — 사용자가 정한 장비 효과. 부채꼴은 화면 방향 8칸 고리(N NE E SE S SW W NW)에서 앞을 중심으로 좌우 둘씩이며 앞이 E면 N·NE·E·SE·S다. 사용자가 처음 적은 E 예시에서 N이 빠진 것은 실수라 정정함.
- `assets/data/traits.json`: 장비 전용(비용 0) 특성 `qinggang_edge`(청공: `sweep_arc` 0.5)와 `heaven_blade`(의천: `evade` 15)를 추가. 처음에 임시로 넣었던 찔러공격·사방격 효과는 이것으로 바꿈.
- `internal/core/traits_test.go`, `internal/testdata/mini/traits.json`: 부채꼴(NE·SW·E 앞에서 맞는 칸, 거리가 있으면 안 닿음)과 회피(명중률이 정확히 15%p 내려감) 시험을 추가.
- `docs/logic-traits.md`, `logic-combat.md`, `data-item.md`, `todo/data.md`: 장비 전용 특성, 물리 명중식, 청공·의천 효과 설명, 아이템 할 일(육도만 남음)을 갱신.
- 누가 언제 얻는지(`grant` 노드, 시작 장비)는 시나리오 데이터에서 정한다. 지금 조조·조운의 시작 장비는 철창 그대로.

### 장비한 무기에 따라 도트가 바뀜(조조·조운), 검 색을 어두운 회색으로
- `internal/content/content.go`, `validate.go`: `Look.Sprites`의 키를 `병종` 또는 `병종:무기 종류`로 넓히고 검증도 맞춤 — 청공검을 빼앗아 드는 조운처럼 무기를 바꿔 들면 모습이 바뀌게 하려고.
- `internal/core/types.go`, `engine.go`: `UnitView.Weapon`(장착 무기 종류) 추가 — 화면이 무기 종류를 알아야 해서.
- `internal/gui/look.go`, `view.go`, `deployment.go`, `duel.go`, `duel_gallery.go`, `editor.go`, `editor_ui.go`: `battleClassArt`와 `duelFighter`가 무기 종류를 받아 `병종:무기` → `병종` → `Look.Sprite` → 병종 스프라이트 순으로 고름. 정의만 있는 곳(일기토 보조·갤러리)은 정의의 무기 장비로 판정(`startingWeapon`).
- `assets/data/characters.json`: 조조·조운의 `Look.Sprite`를 창 든 `caocao`·`zhaoyun`으로 두고, 경기병·중기병·친위대가 검을 들면 `caocao_sword`·`zhaoyun_sword`가 나오게 함 — 사용자 결정(무기에 따라 다르게).
- `tools/doteditor/tools/mounted_kits/heroes.py`: 검 날을 흰색(철1)에서 어두운 회색(철3·철2 램프, 날 가장자리만 철1에 가까움)으로 바꾸고 `caocao_sword`·`zhaoyun_sword`를 다시 변환 — 사용자 요청("좋은 검이 너무 하얗다"). 날 색은 도트의 철 태그를 따르므로 조운은 은빛이라 조조보다 밝은 회색이다.
- `assets/sprites/{caocao,zhaoyun}.spr`와 PNG: 창 든 도트를 빌드, `caocao_sword`·`zhaoyun_sword`는 다시 빌드.
- `internal/gui/look_test.go`: 무기에 따른 선택(창·검·없음·다른 무기) 시험을 추가.
- `docs/presentation-scene.md`, `data-character.md`, `todo/data.md`: 무기별 스프라이트 규칙, 조조·조운 열전, 청공검 아이템 할 일을 갱신.

### 검 든 조운(zhaoyun_sword)
- `tools/doteditor/tools/mounted_kits/heroes.py`: 검 든 조운 유닛 추가. 조조의 검 설정(관우의 `언월도 횡베기` 휘두르기에 짧은 곧은 검)을 `swordsman`으로 뽑아 두 유닛이 같이 쓰고, 은빛 갑옷 기병은 `silver_rider`로 뽑아 창 든 조운과 나눔 — 사용자 요청(조운도 조조처럼 검). 같은 코드를 쓰므로 조조의 검은 그대로.
- `tools/doteditor/units/zhaoyun_sword.yaml`, `assets/sprites/zhaoyun_sword.spr`, `internal/sprite/testdata/units/zhaoyun_sword.png`: 새 파일(기반 중기병, 태그 색은 `zhaoyun`과 같음). 조운의 `Look.Sprite`는 아직 창 든 `zhaoyun`이고, 검 든 도트를 쓸지는 사용자 결정을 기다림.

### 일기토 우선 AI 설정, 여포의 백마는 하비에서, 나머지 장수 도트 연결
- `internal/content/content.go`, `internal/core/types.go`, `engine.go`, `api.go`: 스폰 설정 `DuelFirst`를 `Guard`와 같은 경로(스폰 → 유닛 → 저장 검증)로 추가하고 AI 정책 버전을 `greedy-8`로 올림 — 백마로 활을 쏘는 여포가 일기토 상대를 멀리서 죽여 일기토가 사라지는 문제를, 호로관만 특성을 덮어쓰는 대신 AI 설정으로 풀려고(옛 저장·리플레이는 버전 검사로 거부됨).
- `internal/core/formation.go`: `pendingDuel`을 `duelFor`에서 떼어 내고 `DuelPending(a, b)`를 공개 — 위치와 상관없이 치르지 않은 일기토 상대인지 AI가 물을 수 있게.
- `internal/ai/ai.go`: `DuelFirst` 유닛은 일기토 상대와 맞닿는 이동 칸을 1500점으로 먼저 고르고, 상대를 맞닿기 전의 공격(원거리 포함)은 건너뜀. 일기토가 꺼져 있으면(`duels` 거짓) 무시.
- `internal/ai/coalition_test.go`: 호로관 테스트를 `coalitionDuels`로 나누고, 여포에게 백마를 줘도 일기토 두 건이 일어나는지 보는 시험을 추가 — `DuelFirst`를 끄면 무안국이 먼저 쓰러져 실패함을 확인.
- `assets/data/maps/B03.json`: 호로관의 여포와 연합 장수(목순·무안국)에 `DuelFirst`를 켬. 앞서 넣은 여포 특성 덮어쓰기(`Traits: ["valor"]`)는 뺌.
- `assets/data/characters.json`: 여포의 기본 특성에서 `white_horse`를 뺌 — 사용자가 마지막 전투(하비) 즈음에 얻게 하려고 함. 하비 스테이지가 아직 없어 `docs/todo/data.md`에 "하비 스폰에 `Traits`와 `DuelFirst`" 할 일로 적음.
- `assets/data/characters.json`, `assets/sprites/{yuanshao,zhaoyun,yuanshu,sunjian}.spr`, `internal/sprite/testdata/units/*.png`: 원소·조운·원술은 `Look.Sprite`를 새 도트로 연결하고(원소의 `themeA` 노랑과 조운의 말 색 `Look.Palette`는 도트에 이미 있어 지움), 손견은 열전 수치로 새 캐릭터를 추가(경기병·오 진영, 초상은 아직 없어 병종 초상) — 사용자 요청.
- `internal/gui/scenario_test.go`: 캐릭터 스프라이트 검사가 `Look.Sprites`의 값도 확인하도록 넓힘 — 병종별 스프라이트 ID가 틀려도 잡히게.
- `docs/data-scenario-node.md`, `logic-battle-event.md`, `system-save.md`, `todo/data.md`, `data-character.md`: `DuelFirst`, 저장 AI 버전, 하비의 여포, 열전을 갱신.

### 여포의 백마와 활 든 도트, 병종별 스프라이트, 근접 판정을 8칸으로
- `internal/content/content.go`, `validate.go`: `Look.Sprites`(병종 ID → 스프라이트 ID)와 `Class.Shoot`(`mounted_range`로 쏠 때 취하는 병종) 추가, 검증은 없는 병종과 원거리가 아닌 `Shoot`를 막음 — 한 인물이 병종마다 다른 모습을 갖게 하고(병종을 바꾸면 보이는 모습이 바뀜), 근접 병종이 쏠 때의 모습을 데이터로 정하려고.
- `internal/core/combat.go`, `rules.go`, `types.go`, `engine.go`: 근접·원거리 판정을 `|dx|<=1 && |dy|<=1`(`adjacent`)로 바꾸고, `UnitView.Shoot`(`shootClass`)을 추가 — 사선으로 붙은 적을 칠 때 창을 휘두르는데 규칙은 원거리가 되는 어긋남을 없애려고(사거리는 그대로 `|dx|+|dy|`).
- `internal/gui/look.go`, `player.go`, `view.go`: 전투·장면 스프라이트가 `Look.Sprites[병종]`을 먼저 보고, 기본 공격이 8칸 밖이면 공격 동안 `shoot` 모습으로 바꿔 화살을 쏨(`shootArt`, `shootsFromRange`) — 여포가 평소엔 화극을 들고 거리가 벌어지면 활을 쏘게 하려고.
- `assets/data/classes.json`(경기병·중기병·친위대 `Shoot: horse_archer`), `traits.json`(`white_horse`의 `mounted_range` 2→3, 설명 갱신), `characters.json`(여포 `Traits`에 `white_horse`, `Look.Sprite`=`lubu`, `Look.Sprites`=경기병 `lubu`·궁기병 `lubu_bow`, 도트에 구워 넣은 `Look.Palette` 제거) — 백마의 최대 사거리를 궁기병의 사거리(3)로 맞추고 여포에게 줌.
- `assets/data/maps/B03.json`: 호로관의 여포 스폰에 `Traits: ["valor"]`를 덮어씀 — 백마를 주자 여포가 3칸 밖에서 무안국을 쏴서 일기토(`hulao-wuanguo`)가 일어나지 않아 호로관 AI 통합 테스트가 깨졌고, 연출된 일기토 흐름을 지키려고.
- `tools/doteditor/tools/mounted_kits/heroes.py`, `units/lubu_bow.yaml`, `lubu.yaml`: 활 든 여포(`lubu_bow`, 궁기병 키트의 활·화살통·사격 포즈에 여포의 중갑·마갑·꿩깃 머리)를 추가하고 `lubu`를 전 동작으로 확정, `assets/sprites/lubu.spr`·`lubu_bow.spr`와 스프라이트 비교용 PNG를 빌드 — 여포의 두 모습.
- `internal/core/traits_test.go`, `internal/gui/look_test.go`, `internal/testdata/mini/classes.json`: 시험 추가 — 8칸 판정과 `Shoot`, 병종별 스프라이트와 활 모습 전환 조건.
- `docs/logic-combat.md`, `logic-traits.md`, `presentation-scene.md`, `data-character.md`: 근접 판정, 백마, 병종별 스프라이트와 활 모습 전환, 여포 열전을 현재형으로 갱신.

### 조조·조운에게 다재다능(검·창 겸용), 조조는 검 든 도트로
- `internal/content/vocab.go`: 효과 키 `wield:<무기 종류>`(`FxWield`) 추가 — 병종 무기와 다른 무기를 장착하는 규칙을 데이터(특성)로 주려고. 접두 키라 `validate.go`의 키 검사가 그대로 통과한다.
- `internal/content/spawn.go`: `Data.Carries(class, traits, equipment)`와 `Data.SpawnTraits(spawn)` 추가, 스폰 장비 검증이 `Carries`를 씀 — 장비 호환 판정이 엔진·저장·스폰·편집기 여러 곳에 복사돼 있어서 한 함수로 모음.
- `internal/content/validate.go`, `internal/core/engine.go`, `internal/core/api.go`, `internal/gui/editor.go`, `internal/gui/editor_ui.go`: 각자의 `Weapon`/`Armor` 비교를 `Data.Carries`로 교체(저장 검증은 장수 정의·학습·덮어쓴 특성으로 판정) — 특성으로 무기를 얻은 장수가 장착·저장·편집기 목록에서 막히지 않게.
- `assets/data/traits.json`: `versatile`(다재다능, 비용 0: `wield:spear`, `wield:sword`) 추가. `assets/data/characters.json`: 조조 `Traits`에 `versatile`, `Look.Sprite`를 `caocao_sword`로, 조운 `Traits`에 `versatile` — 사용자 결정(조조는 검 든 도트를 쓰고 조조·조운이 검과 창을 모두 든다). 조운의 `Look.Sprite`는 아직 그대로(새 도트는 `.spr` 빌드 전).
- `assets/sprites/caocao_sword.spr`, `internal/sprite/testdata/units/caocao_sword.png`: 새 파일. `build.py`로 `.spr`만 빌드(`make apply` 전체는 돌리지 않음)하고, 스프라이트 테스트가 시트마다 비교용 PNG를 요구해 이 유닛만 `golden.py`와 같은 방식으로 PNG를 만듦.
- `internal/testdata/mini/traits.json`, `internal/core/core_test.go`: 시험용 특성 `lancer`(`wield:lance`)와 `TestEquipmentRules` 확장 — 특성으로 다른 무기는 장착되고 그 밖의 종류는 여전히 거부됨을 확인.
- `docs/logic-traits.md`, `docs/data-item.md`, `docs/data-character.md`: 다재다능 규칙과 장착 규칙, 조조·조운 열전을 현재형으로 갱신.

### 장수 7명의 전용 도트를 3D 변환(class-sprite-3d)으로 시작
- `tools/doteditor/tools/mounted_kits/heroes.py`: 새 파일. 조조(친위대 + 관모·검은 수염), 여포(중기병 + 방천화극 + 꿩깃 금관), 원소(친위대 + 금투구, 장식 높이 절반), 조운(은빛 갑옷 기병 + 붉은 깃), 손견(두건 쓴 경기병), 헌제(바닥에 끌리는 도포의 서생 + 면류관), 원술(가는 몸 + 금 문양 + 원소에서 옮긴 긴 장식)을 기존 키트의 머리·갑옷·몸 변형으로 등록 — 병종 도트 위에 팔레트만 바꾸면 한 병종의 장수들이 구별되지 않아서. S 방향 얼굴은 양쪽 1px씩 좁혀 경기병 폭에 맞춤. 금투구 하이라이트 키 `n`은 말 가죽 키와 겹쳐 `m`으로 합침.
- `tools/doteditor/units/{caocao,yuanshao,lubu,zhaoyun,sunjian,yuanshu}.yaml`: 새 파일. 대기(idle) 8방향 × 4프레임만 3D 변환이고 나머지 동작은 기반 병종(친위대·중기병·경기병·경보병)의 프레임 그대로 — 사용자가 먼저 대기 모션만 요청했기 때문.
- `tools/doteditor/units/xiandi_noncombat.yaml`: 새 파일. 헌제는 비무장(NW·SW, 대기 1 + 걷기 2 + 동작 1)만 — 장면용 인물이라 전투 모션이 필요 없다는 결정.
- 태그 색은 `grid.py colors`로 장수별로 지정: 여포 적토마(붉은 말), 조운 백마·은빛 철 색·붉은 깃(themeC), 조조·손견·헌제 검은 머리카락(수염이 가장 밝은 hair1), 원술·헌제 금 장식(themeC).
- `tools/doteditor/units/lubu.yaml`: 말 피부·갈기 태그 색을 이전 `Look.Palette` 값(`#e06050/#b03020/#741a10`, `#901818/#601010/#300808`)으로 되돌림 — 미리보기용 어두운 적토마 색보다 기존 색이 더 선명해서 사용자가 기존 색을 선택.
- `tools/doteditor/units/lubu.yaml`: 철(`#6a6a78/#3a3a46/#1a1a22`)과 themeA(`#d04848/#9a2424/#5a1010`)도 이전 `Look.Palette` 값으로 맞춤 — 기존 여포의 어두운 철과 붉은 가죽 톤을 유지하려고.
- 여포 진영색은 검정으로 바꾸지 않고 기존 `dong`(동탁과 같은 색)을 유지 — 검정은 3D 미리보기에서만 쓴 임시 색이었고, `game.json`의 `dong`을 바꾸면 동탁까지 바뀌기 때문. 변경한 파일 없음.
- `tools/doteditor/units/{caocao,yuanshao,lubu,zhaoyun,sunjian,yuanshu}.yaml`: 걷기·공격·피격·탈진·막기까지 3D 변환(+`--denoise`)해 8방향 전 동작을 채움 — 대기만 바꾸면 움직일 때 기반 병종으로 보이기 때문. 원술 막기 행에서 `Bl.foot`의 `style` 인자를 빠뜨려 실패해 `heroes.py`에서 `Bl.SELECTED`를 넘기도록 고침.
- `assets/data/game.json`: 진영 `han`(한 황실, `#ffd85a/#e0a81e/#9a6a10`)을 추가 — 헌제의 옷(진영색 도포)을 황색으로 보이게 하려는데, 진영색 태그는 도트에서 장수별로 못 바꾸고 `yuan`(원소·원술의 레몬 노랑)과 구분되는 황금색이 필요해서. 헌제 캐릭터를 만들 때 `Look.Faction: han`으로 쓴다.
- `tools/doteditor/tools/mounted_kits/heroes.py`: `caocao_sword` 유닛 추가(검 든 조조, 대기·걷기·공격·피격·탈진·막기 8방향). 관우의 기마 코드를 재사용해 `units.glaive_at` 자리에 곧은 검(손잡이·금 날밑·길이 1.35 날)을 그리고, `언월도 횡베기` 휘두르기와 궤적을 그대로 따르되 무기 도달 거리를 언월도의 약 2/3(`MOUNT_REACH['glaive']` = (.45, 1.5))로 줄임 — 창대 달린 언월도보다 짧되 적에게 닿지 않아 보이지 않게.
- `tools/doteditor/units/caocao_sword.yaml`: 새 파일(기반 친위대, 머리카락 태그 색은 `caocao`와 같음). 창 든 `caocao`는 그대로 두고 둘 중 게임에서 쓸 것을 나중에 고른다.

- 게임 반영(`make apply`, `Look.Sprite` 연결, 여포 진영색 검정)은 하지 않음 — 스킬의 정지선이며 `dong` 진영색 변경은 동탁에도 영향을 주므로 사용자 결정을 기다린다.

### 소비 아이템을 영걸전 방식으로 바꿈
- 병력 회복은 콩·보리·쌀(소·중·대), 병법치 회복은 술·특급주·빼갈(30·60·120). 군량은 화폐로만 쓴다.
- 이유: 화폐 군량과 소비 아이템 "소군량·중군량"의 이름이 겹쳤다. 영걸전의 회복 아이템 이름(콩·보리·쌀, 술 계열)을 따랐다.
- 영걸전의 술 계열은 사기 회복(PC판)이지만 여기서는 병법치 회복으로 쓴다. ver5에 병법치가 있고 소병법단이 이미 병법치 회복이었기 때문이다. 회복량은 기존 소병법단 15가 낮다고 보아 30·60·120으로 올렸다.
- ID는 `bean`, `barley`, `rice`, `liquor`, `fine_liquor`, `baijiu`. 쌀 2200은 임시 값이다. 상약·서 계열은 나중에 정한다.
- 영향: `data-item.md`, `todo/data.md`

## 2026-10-17

### 변경 기록 규칙 추가
- `AGENTS.md`: 변경할 때마다 `history.md`에 `파일이름: 수정내역 — 의도`를 적는다는 규칙을 추가 — 무엇을 왜 바꿨는지 한곳에서 추적하려고.
- `docs/history.md`: 머리말의 "커밋에 남는 것은 적지 않는다"를 새 규칙에 맞게 고침 — 규칙이 서로 어긋나지 않게.

### 비무장 스프라이트와 장수 도트 범위 확정
- `docs/todo/graphics.md`: "비무장(장면용)"을 신규 제작에서 기존 병종 도트 + 옷색·진영색 재사용으로 바꿈(서복·선비·헌제는 서생, 병사는 경보병, 피난민·촌민·노인은 서생/경보병, 전령은 경보병/경기병) — 장면 배우가 `cast`·`Look.Scene`·병종 스프라이트 순으로 스프라이트를 찾으므로 새로 그릴 필요가 없다.
- `docs/todo/graphics.md`: 전용 도트 장수를 조조·여포·조운·손견·원소로 확정, 동탁·공손찬은 보류, 나머지는 병종 도트 + 팔레트 — 시나리오 등장 비중이 큰 인물만 그려 작업량을 줄이려고. `_noncombat` 시트는 장면에서 어색할 때만 추가.

### 전용 도트 장수에 공손찬·동탁 추가
- `docs/todo/graphics.md`: 전용 도트 장수를 조조·여포·조운·손견·원소·공손찬·동탁(배불뚝이)로 확정, 보류 표기 삭제 — 공손찬은 언급 49회로 비중이 크고, 동탁은 배불뚝이 체형으로 병종 도트와 달라 전용이 필요하다.

### 전용 도트 장수를 12명으로 확장
- `docs/todo/graphics.md`: 전용 도트 장수에 고순·원술·헌제·허저·문추를 추가(조조·여포·원소·조운·공손찬·손견·동탁 포함 12명), 비무장 목록에서 헌제를 뺌 — 언급 횟수가 많거나 체형·역할이 병종 도트와 달라 팔레트만으로는 부족한 인물이라서.

### 문서 구조를 `<분류>-<기능>.md`로 바꿈
- 분류: logic(어떻게 돌아가는가) / data(그 위에 어떤 데이터를 태우는가) / presentation / tools / system.
- 이유: 한 사실이 여러 문서에 되풀이되어 어긋났다(경험치·부관·일기토가 `battle.md`와 `design.md`에 중복 등). 한 사실은 한 문서에만 두고 나머지는 링크한다.
- 본문은 현재 코드·데이터가 하는 일만 현재형으로 쓴다. 미구현 설계는 `todo/`, 작업 중 plan은 `temp/`(git ignore). 구현이 끝나면 분야 문서로 옮기고 todo는 지운다.
- 정리 전 문서는 `docs_old/`에 보존한다(`design.md` 원본, 후반 시나리오 계획 `scenario.md`, `script-c1.md`, 옛 `scenario-system.md`, 특성 문서 3종, `ui-design.md`, `plans/`).

### 시나리오 v1(1~20)을 ver5의 기준 시나리오로 채택
- 원본(`scenario_v1.txt`)은 현재 게임 시스템 이전에 쓴 것이라, 큰 줄기에 맞으면 수정을 허용한다. 문서 이름은 `conceptual_scenario.md`(뼈대 이야기).
- 기존 B01~B03은 폐기하고 새로 만든다. 도원결의로 시작하던 흐름을 영천(노식)으로 바꾼다.
- 범위는 1~20 전체. "이벤트 스테이지"는 별도 시스템이 아니라 전투 없이 장면만 나오는 스테이지(2 낙양, 6 산조, 12 서주 계승, 16 허도)다.

### 군량을 화폐로 삼고 출전을 필수/정원/증원 3단계로 함
- 금 보상은 군량으로 바꾼다. 출전 비용, 상점 매매, 보상이 한 화폐로 정리된다.
- 필수(반드시 출전), 정원(무료로 채우는 인원), 증원(정원을 넘어 최대까지, 군량을 내고 출전). 어려우면 군량을 들여 병력을 늘리고, 아낄 수 있으면 아낀다. 시나리오의 "4기부터 병량 소모"는 이 증원 비용이 대신한다.
- 출전 전 회의장(조조전식)에서 대화와 군수물자 매매를 한다.
- 영향: `todo/engine.md`, `todo/data.md`

### 쓰러뜨릴 수 없는 적(`Unyielding`)을 두지 않음
- 여포는 그냥 강하게 만든다. 일기토 외의 이유로 퇴각해도 그대로 둔다. HP 임계 회복(`hp`+`heal`)도 쓰지 않는다.
- 영향: `todo/engine.md`

### 일기토 디버프를 없애고 이벤트로 처리함
- `AttackDown`/`DefenseDown` 필드와 `attack-down`/`defense-down` 상태를 지웠다. 디버프는 스킬 디버프 하나뿐이다. 일기토 승리 뒤의 효과는 `duel:ID` 조건 이벤트의 `buff` 동작으로 건다(이벤트 시스템 구현 전까지 옛 B01의 두 효과는 비어 있다).
- 영향: `logic-duel.md`, `data-scenario-node.md`, `todo/engine.md`

### 일기토는 인접하는 순간 자동 발생
- 공격이 일기토로 바뀌던 방식을 버린다. 지정한 두 장수가 인접한 상태가 되면(이동 뒤 대기·공격·소환·합류 어느 경우든) `settle()`의 첫 단계에서 벌어진다. 결과는 지금처럼 데이터로 고정이다.
- 영향: `todo/engine.md` 2.5

### 이벤트 트리거는 상태 조건 하나로 통일함
- 엣지 트리거(`enter`·`stay`·`adjacent`·`hp`·`defeated`)를 없애고 상태 조건(`at`·`adjacent`·`hp`·`defeated`…)으로 줄인다. 검사 시점 필드는 두지 않고, 시점이 필요하면 `turnstart:진영` 조건을 쓴다. 방화병의 "1턴 생존"은 `turnstart:enemy`(적 차례 시작, 아군이 한 번 행동한 뒤)에 `at`을 검사하는 것으로 한다.
- 영향: `todo/engine.md` 2.2

### 불은 타일 두 종류로 표현함
- 불길(진입 불가)과 불붙은 땅(배율 0.8, 차례를 맞으면 최대 병력의 20% 손실). 지나가기만 하면 피해가 없다. 불이 붙는 것은 `terrain` 동작이 타일을 바꾸는 것으로만 일어난다.
- 영향: `todo/engine.md` 2.6

### 스테이지는 전투, 스토리는 장면·대사
- 스테이지는 전투이고 스토리는 장면·대사 노드의 흐름이다. 전투 앞뒤의 장면(1 영천의 노식 대사 등)도 스토리이고, 2 낙양, 6 산조, 12 서주 계승, 16 허도, 24 삼고초려는 스토리만 있는 번호다("전투 없는 장"이라는 별도 범주는 없다). 전투 중 이벤트와 구별한다.
- 영향: `conceptual_scenario.md`, `todo/engine.md`, `todo/data.md`

### 호로관의 연합 장수는 목순으로 통일
- 방열과 목순이 비슷하고, ver5에 이미 목순·무안국과 `hulao-mushun` 일기토가 있다.

### 병종 도트는 새로 그리지 않음
- 백마의종은 기병+흰색 진영색, 쇠뇌 노병대는 기존 노병, 함진영은 색을 바꾼 창병으로 쓴다. 장면 배경 맵은 막사류·성내류·궁궐류로 합쳐 쓴다.
- 영향: `todo/graphics.md`

### `design.md`를 방향과 원칙만 남기고 줄임
- 옛 `design.md`(ver2/ver4, 5437줄)는 `docs_old/`에 보존한다. 경험치·부관·일기토 규칙은 `logic-*`가 맡는다.
- 옛 설계서의 다음 서술은 현재 코드·데이터와 달라 버렸다: 레벨별 필요 경험치 곡선(실제는 모든 레벨 10,000 고정), "피격·방어 경험치 없음"(실제는 `ExpDefenseShare` 0.5), 부관 임명 특성을 병종이 준다는 서술(실제는 학습 특성).
- 특성 문서 3종은 `logic-traits.md` 하나로 합쳤다. 병종 `Grants` 폐지, 승급 레벨 15·30(옛 문서는 20·40) 등 데이터를 따랐다.

## 2026-10 (옛 `battle.md` 개정 이력에서 옮김)

현재 값은 `logic-combat.md`, `logic-growth.md`와 `assets/data/`가 말한다. 아래는 바꾼 이유만 남긴다.

- **물리 피해식**을 `max(1, 공격−방어+(레벨+10)×20)`으로 바꿨다(`PhysicalLevelScale` 20). 공방비 보정(0.8~1.2배)과 `DamageCurve` 규칙값은 없앴다. 이유: 같은 수치끼리면 레벨이 높은 쪽이 강하도록.
- **책략 피해식**을 `max(1, 정신차+(레벨+10)×10)`으로 바꿨다(`MagicLevelScale` 10). 이유: 물리와 같은 레벨 기반 구조로 통일.
- **최대 MP와 MP 회복**을 `Rules`의 곡선(`ManaMaxFactor`, `ManaRecoveryFactor`)으로 바꿨다. `MPCoeff`·`RecoveryCoeff`는 없앴다. 이유: 능력치 곡선을 다른 전투 수치와 맞추려고.
- **상위 병종 HP 보너스**를 직접 올리고 별도 승급 배율은 폐기했다. 이유: 매력 80 기준으로 2단계 약 +10%, 3단계 약 +20%를 맞추려고.
- **초열·업화 재조정**: 책략 피해식 개정에 맞춰 MP와 계수를 조정했다. 업화 MP와 물리 병법 MP, 저장 MP 정책은 유지했다.
- **`troop_scholar` 지력 30→60, 조조의 병종 경로를 경기병→중기병→친위대로, 사마의 운을 100으로** 바꿨다.
- **문무겸비**(`civil_martial`/`blend_bonus`)를 신설했다. 부여 대상은 조조·강유·등애뿐이고 학습 비용은 정하지 않았다(부여만 가능). 이유: 문무 양쪽 병종 보너스를 평균으로 올려 조조 같은 총대장형을 표현하려고.
- **규칙 버전을 올려 옛 저장을 불러오지 못한다.** 추가 행동(`free_skill`·`free_item`)과 국사무쌍·부호를 없앴다. 병종 `Grants`를 없애고 책략 권한을 학습 특성으로 옮겼다. 이유: 규칙을 단순화하고 병종과 특성의 역할을 분리하려고.
- **부관 보정률에 상한이 없다**는 점을 문서에 못박았다. 이유: 보정률이 1을 넘으면 부관이 앞선 능력치가 부관 수치를 넘을 수 있어서.
- **밸런스 점검 항목**(동급 전투의 격퇴까지 타격 수, 병종 간 대결, 문관의 지속 병법 횟수, 부관 유무별 성장 차)은 `design.md`의 밸런스 축에 있다. 수치를 바꿀 때는 같은 시드·정책·편성으로 전후를 비교하고, 코어를 바꾼 뒤에는 `go test ./... && go vet ./...`를 돌린다.

## 2026-10-18 모바일 조작·일기토 컷신 정리
- `internal/gui/unit_bar.go`: 흰 trail이 남은 상태에서 회복하면 `trailRate`가 음수가 되어 trail이 끝없이 늘어나던 버그를 절댓값 속도와 `hold=0`으로 고침 — 간옹 HP바 하얀 막대가 무한히 길어짐. 회귀 테스트 `unit_bar_test.go` 추가.
- `internal/gui/hud.go`: 행동 메뉴(menu/sub) 단계에서는 지형 `brief`를 그리지 않음 — 메뉴 4개 항목을 가림.
- `internal/gui/command.go`: 터치 입력에서 이동 목적지·공격/병법 대상을 처음 누르면 노란 하이라이트로만 표시하고(`order.armed`, `arm`, `marks`) 같은 칸을 다시 눌러야 명령 — 롤오버가 없어 지형 정보를 못 보이고 오터치가 잦아서.
- `internal/gui/duel_cutin.go`: 컷인 초상을 확대(1.35배, 크게 오프셋)하지 않고 패널 폭에 맞춤, 이름 글씨 크기를 밴드 높이에 비례해 줄임 — 얼굴이 너무 커지고 모바일에서 이름이 밴드를 넘침.
- `internal/gui/duel_decor.go`, `battle_speech.go`: 낮은 화면(`smallScreen`)에서 하단 장수 카드를 줄이고 말풍선을 키움 — 모바일에서 카드는 크고 대사는 작음.
- `internal/gui/duel_cutin.go`: 컷신 이름과 "아군/적군 장수" 표기를 각 초상 패널의 보이는 영역 중앙(x 25%·76%)에 가운데 정렬 — 이름이 화면 가운데에 몰려 있었음.
- `internal/duelview/view.go`, `internal/gui/duel.go`: 일기토 말풍선에 `BubbleScale` 추가, 낮은 화면에서 1.6배 — 모바일에서 대사가 작음. `DrawText`가 좌표에도 배율을 곱하므로 x/k, y/k로 넘겨 위치를 유지.
- `internal/duelview/bubble.go`, `view.go`, `internal/gui/battle_speech.go`: 말풍선 글자를 화면 해상도로 다시 그리는 `DrawSharp` 추가, 일기토는 캔버스를 확대한 뒤 그 위에 그림(`drawBubbles`) — 1x로 그린 글자를 확대해서 모바일에서 뭉개져 보였음.
- `internal/duelview/decor.go`: 일기토 진영의 깃발을 3개에서 6개(군기 4·지휘 깃발 2)로 늘리고, 기존 `banner`/`command_flag` 그림을 열 단위로 물결치게 비튼 4프레임 애니메이션(`wave`, 170ms)을 런타임에 만들어 씀 — 새 그림 없이 펄럭임을 내려고. 깃발마다 위상이 달라 같이 움직이지 않음.
- `internal/gui/duel.go`: 낮은 화면의 일기토 말풍선 배율 1.6→2 — 대사가 아직 작다는 의견.
- `internal/gui/duel_gallery.go`: 메뉴의 일기토 모드 뷰어에도 `Decor: g.duelDecor()`를 넘김 — 깃발·바위·풀이 없는 빈 경기장이었고, 깃발 펄럭임이 전투 중 일기토에만 보였음.
- `tools/doteditor/units/{caocao,sunjian,yuanshao,zhaoyun,dongzhuo,infantry}_noncombat.yaml`, `assets/sprites/*_noncombat.spr`, `internal/sprite/testdata/units/*_noncombat.png`: 조조·손견·원소·조운·동탁의 장면용 비무장 도트와 일반 보병 시트를 새로 빌드 — 장면에서 기병 장수만 말을 타고 나와 어색했음. 에이전트가 손으로 그린 초안은 원본 스프라이트와 달라 버리고, `tools/unit3d`의 평복 3D 모델을 도트로 변환하는 방식으로 만들었다.
- `tools/doteditor/tools/civ3d.py`, `model3d.py`, `model3d.html`: 3D 평복 모델에서 `<id>_noncombat`을 만드는 도구와 3D 모델 뷰어(머리는 방향별 2D 도트를 붙임).
- `tools/unit3d/civilian.py`: 평복 걷기에서 팔이 무기 든 병사 자세로 파닥거리던 것을 앞발과 같은 쪽 팔이 뒤로 가는 팔 흔들기(`walk_arms`)로 바꿈.
- `assets/data/characters.json`: 조조·손견·원소·조운·동탁의 `Look.Scene`을 `<id>_noncombat`으로 연결.
- `assets/data/classes.json`, `internal/content/content.go`, `internal/gui/look.go`: 병종에 `Scene` 필드를 추가하고 기마 병종(경기병·중기병·궁기병·노기병·친위대·전차)에 `infantry_noncombat`을 지정, `sceneArt`가 캐릭터 `Look.Scene` 다음에 병종 `Scene`을 쓰게 함 — 장면용 시트가 없는 공손찬·화웅 같은 기병 장수가 걸어서 서도록.
- `assets/data/characters_story.json`: 번궁을 경기병에서 무도가(`brawler`)로 바꾸고 장비를 평복으로, 말 팔레트를 뺌 — 사용자 요청.
- `tools/doteditor/units/bandit.yaml`, `assets/sprites/bandit.spr`, `internal/sprite/testdata/units/bandit.png`: 적병 시트를 3D 변환으로 다시 만듦 — 기존 시트는 경보병 기본형에서 방패만 지워서 팔이 방패·칼 든 자세 그대로(팔꿈치가 바깥으로 꺾인 L자, 걷기에서 팔이 안 움직임)이고 칼 궤적이 칼과 어긋났음. 머리는 기존 손그림 머리를 유지한다.
- `tools/doteditor/tools/grid3d.py`: `--kit bandit` 추가(맨팔 삼베옷, 허리띠, 칼, 방패 없음) — 방패 없는 병종은 양팔을 몸 옆에 내리고 칼 든 손을 허리에 두는 자세(`relaxed_pose`)를 쓰고, 걷기에서 빈 팔이 다리와 반대로 흔들린다. 머리 상자를 기존 시트에서 복사할 때 상자 안의 옛 칼 픽셀과 칼에만 붙은 외곽선을 빼고(칼이 두 자루로 보이고 목에서 잘리던 원인), 머리 앞에 있는 3D 칼 픽셀은 머리 위에 다시 그림(`front_weapon`), 공격 궤적은 기존 시트에서 붙이지 않고 3D 칼 경로로 생성(`TRAILS`), 정면(S)·후면(N) 내려베기는 칼이 점이 되므로 몸 앞을 가로지르는 베기(`overhead_s`)와 칼 납작한 면을 카메라 쪽으로 돌리는 처리(`face_camera`)를 씀, 궤적 폭을 1.3→1.1로 줄여 칸 밖으로 잘리던 것을 없앰.

## 2026-10-21 전투맵 배경 장식과 숲 변형
- `internal/gui/field_scenery.go`(신규): 장식 정의(`scenery.json`) 로더, 맵 ID·좌표 해시와 값 잡음으로 소타일마다 풀포기·꽃·자갈·얼룩(황무지·성내·숲 바닥은 각자의 세트)을 정해 한 장에 굽는 `makeScenery`, 숲 칸마다 소나무 4~8그루를 변형·위치·위상을 달리해 놓는 `nativeForest`(그루마다 깊이 정렬, 프레임 4장 흔들림, 그림자는 장식 층에 같이 구움), 보이는 부분만 그리는 `drawBaked` — 땅 타일은 그대로 두고 그 위 장식만 늘려 이음새가 드러나지 않게 하고, 같은 소나무 묶음이 격자로 반복되던 숲을 없애려고. 장식 위치는 저장하지 않고 계산한다.
- `internal/gui/field.go`: `Field`에 `decor`·`treeShadow`·`clockMS` 추가, `NewField`가 `makeScenery`/`finishScenery`를 부르고 `Draw`가 땅·바닥 다음에 장식 층을 그림 — 성내 바닥이 장식을 덮지 않게 바닥 뒤에 그림.
- `internal/gui/field_native.go`: `nativeObjects`가 장식 자료가 있으면 `nativeForest`를 쓰고 없으면 예전 `forest_N` 묶음 그대로 — 자료가 없어도 게임이 도는 안전장치.
- `internal/gui/assets.go`: `Assets.Scenery`와 `loadScenery` 호출 추가(`graphics/map/scenery.json`이 없으면 nil).
- `internal/gui/editor.go`: `dispose`가 `decor`·`treeShadow`도 해제.
- `internal/gui/uidemo_scenery.go`(신규): `scenery` 데모 — 환경변수 `SCENERY_TOUR`의 칸 좌표·배율 목록대로 카메라를 30프레임마다 옮김 — 전후 그림을 같은 카메라로 뽑으려고.
- `assets/graphics/map/scenery.png`, `scenery.json`(신규): `assets_candidate/tools/gen_scenery.py --install`이 만든 장식·소나무 10종×4프레임·그림자 도장과 놓는 밀도.
- `assets_candidate/tools/gen_scenery.py`(신규): 장식(외곽선 없음, 땅과 가까운 톤, 반투명)과 소나무(기존 숲의 외곽선·팔레트, 원뿔 더미를 왼쪽 빛으로 래스터화, 흔들림 4프레임)를 만듦.
- `tools/mapshots/shots.py`, `compare.py`(신규): 맵 지형에서 촬영 지점을 골라 창 없이 `cmd/_uidemo-vm`으로 PNG·GIF를 뽑고 전후 비교 HTML을 만듦 — 눈으로 확인하는 반복을 프로그램으로 하려고.
- `internal/gui/field_scenery.go`: 물(`sceneWater`: 물가 거품 줄기·갈대, 물 안쪽 연잎, 깊이 얼룩, 반짝임·잔물결 반복 효과 `waterFX`/`drawWaterFX`), 절벽 윗면(`cliffTop`: 윗면 높이 `cliff_rise`만큼 올려 구워 절벽 블록과 같은 깊이 순서로 `drawCliffTop`), 구조물 생기(`siteLife`/`drawSite`: 주둔지 깃발 천을 지운 그림 위에 펄럭임 4프레임, 마을 일부 칸 지붕의 연기, 성 망루 깃발; 위상은 나무와 같은 바람) 추가 — 물과 절벽 윗면이 평평한 단색으로 남아 있었고 구조물은 정지해 있었음.
- `internal/gui/field_native.go`: 절벽 블록 그리기에 윗면 장식 추가, 구조물이 `siteLife`가 있으면 `drawSite`로 그림.
- `internal/gui/assets.go`: `Assets.CliffRise`(index.json `cliff_rise`) 읽기.
- `assets_candidate/tools/gen_scenery.py`, `assets/graphics/map/scenery.{png,json}`: 물·절벽 윗면 소재와 주둔지 깃발·성 깃발·연기 프레임, `sites`·`anim` 설명 추가, 절벽 금 밀도 조정.
- `tools/mapshots/shots.py`, `compare.py`: 절벽·주둔지·마을·성 촬영 지점과 GIF 종류(`--gifkind`) 추가.
- `internal/gui/field_deployment.go`, `field_outside.go`: `shadeScenery`가 전투 영역 밖에 선 물체(나무·절벽·성벽·구조물)를 `exteriorShade`로 어둡게 함(영역 안은 그대로, 밖은 `exteriorDim` 0.55에 비네트 곡선으로 멀수록 더 어둡게, 발 밑 칸 기준이라 경계의 나무가 반으로 갈리지 않음) — 바닥만 어두워서 숲과 절벽이 떠 보였음. 배치 화면의 기존 어둡게 하기와 함께 쓰며 편집기의 외곽 어둡게 보기 전환도 따른다.

## 2026-10-21 B01 장사의 밤 다시 짜기

- `assets/scenario/hero/stage01.json`: 정원지를 처음부터 남쪽 진영에 둠(옛 `chengyuanzhi-comes` 스폰 삭제). 오프닝을 이벤트 하나로 — 유관장 화공 → 숲에 숨기 → 황건 당황 → 남쪽 병사가 바깥·안쪽으로 뛰쳐나옴 → 유관장 급습 → 파재 돌격 명령 → 조조군·손견군 화공 → 황보숭 출성 → 파재가 사방을 돌아보고 당황 → 황건 전원 공격·방어 −20%. 정원지·파재는 앞 장수 일기토 뒤 차례로 출진, 파재 병력 70% 아래면 유비와 일기토. 옛 불 번짐·측면 화공·도주 이벤트와 `b01-routed` 플래그 삭제, 승리는 파재 격파 — 사용자가 정한 진행 순서대로, 일반병 → 등무 → 정원지 → 파재 순으로 자연스럽게 만나게.
- `tools/mapgen/layouts/hero/map01.txt`, `assets/scenario/hero/maps/map01.json`: 26×26 → 32×32. 성, 두 칸 띠, 안쪽 울타리(성을 가둠), 다섯 칸 황건 진영 띠(주둔지 12곳 여섯 칸 간격, 황무지 조각), 바깥 울타리, 바깥 네 칸(남쪽 숲, 북쪽 강). 울타리 틈은 모두 두 칸 — 울타리 사이 길이 한 칸이라 너무 좁았음.
- `internal/core/battle_events.go`: `shift`(동작·컷신 `move`)의 길을 직선에서 걸을 수 있는 땅의 최단 길(`walk`, 다른 유닛 무시, 길 없으면 직선)로 — 컷신에서 병사가 울타리를 뚫고 걸어 나갔음. `buff` 동작이 대상마다 `status` 이벤트를 냄 — 이벤트 디버프가 화면에 아무것도 안 보였음.
- `internal/gui/player.go`, `internal/gui/events.go`: 이어진 같은 능력치 `status` 이벤트를 한 박자에 모든 대상 위 버프·디버프 효과와 "공격 -20%" 수치로 보임, 기록 줄은 내지 않음.
- `internal/cli/play.go`: `status` 이벤트를 능력치·값마다 한 줄(대상 이름 나열)로 출력.
- `internal/core/campaign_content_test.go`: `TestChangsheSiegeUnfolds`를 새 흐름(세 곳 화공, 성문, 병사 뛰쳐나옴, 디버프, 정원지·파재 차례 출진)으로.
- `docs/conceptual_scenario.md`, `docs/logic-battle-event.md`: B01 전투 중 흐름, `buff`의 `status` 이벤트, 컷신 길 찾기.
- 측정(10 시드): rush 10/10(6.0라운드, 관우 +2.6 · 유비 +2.3 · 장비 +2.3), ai 10/10(7.3라운드, 관우 +2.8 · 유비 +1.4 · 장비 +2.3, 장비 퇴각 2). 남쪽 바깥 병사를 여섯에서 넷으로 줄임 — 관우가 +3.1로 기준을 넘고 퇴각했음.

- `assets/scenario/hero/story01.json`: `c01-05` 서술을 두 겹 울타리 진형으로 — 새 맵과 맞춤.

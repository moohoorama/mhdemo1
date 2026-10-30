# 결정 기록

이 문서가 맡는 범위: 설계 결정과 그 이유. 현재 규칙은 각 분야 문서(`logic-*`, `data-*`, `presentation-*`, `tools-*`, `system-*`)가 말하고, 여기에는 **왜 바꿨는가**만 적는다.
변경할 때마다 적는다. 변경한 파일 이름, 수정 내역, 의도(또는 이유)를 `파일이름: 수정내역 — 의도` 형식으로 남긴다. 새 항목을 위에 쌓는다. 설계 결정은 `결정 · 이유 · 영향받는 문서` 항목으로 쓴다.

---

## 2026-10-29 모든 화면의 위쪽 두 모서리에 같은 판: 왼쪽은 장소·장·돈과 배속·확대·자동 넘김·장면 넘기기·전투 목표, 오른쪽은 부대·메뉴, 부대 창에 소지품 탭

- 결정: 화면마다 제각각이던 왼쪽 위(전투 `×1`·`전투 목표` 92×60·140×60, 장면 `자동 넘김 꺼짐/켜짐`·`장면 넘기기` 150×44, 탐험 지역 이름·돈 글자와 대화 중 (24,84)의 자동 넘김)와 오른쪽 위(`메뉴` 96×29, 탐험 `부대`)를 접을 수 있는 판 두 개로 통일. 칸은 늘 같은 순서·자리, 화면에 없는 칸은 빈칸. 켜고 끄는 것은 `목표 안내`와 같은 토글 · 이유: 사용자 "좌상단에 보이는 버튼들이 제각각이야", "자동넘김 꺼짐 같은걸 토글로", "그 화면에 해당 안하는건 비워두는 식으로", "좌상단의 버튼들, 돈 이런것들도 다 확장/축소", "좌상단에 화면 확대 배율도", "우상단의 부대, 매뉴도 마찬가지로", "부대쪽에는 아이템 목록도" — 시안 https://claude.ai/artifact/8ZXyLF1BuWS1J8FsjJ6Txu · 영향받는 문서: presentation-ui.md("모서리 판"), presentation-explore.md, presentation-scene.md
- `internal/gui/hud_corner.go`(새 파일): `leftPanel`(폭 540, 좁으면 칸을 여러 줄로)·`rightPanel`·`cornerButtons`·`foldButton`, 칸 `speedSlot`·`zoomSlot`·`autoSlot`·`skipSlot`·`goalSlot`, 화면별 `storyPanels`·`battleLeftPanel` — 칸 정의를 한곳에 모아 화면마다 자리가 같게
- `internal/gui/skin.go`: `btn`의 몸통을 `btnFace`로 떼고 `centeredBtn`·`toggleBtn`·`emptySlot` 추가. `windows.go`: `objectiveButtons`가 `toggleBtn`·`centeredBtn`을 씀 — 토글 모양을 하나로
- `internal/gui/scenario.go`: `autoButton`·`skipButton` 삭제(`skipGuard`는 hud_corner.go로). `explore.go`: 지역 이름·돈 글자와 `부대`·자동 넘김 버튼 삭제. `game.go`: `speedButton` 삭제, 이야기 화면은 `storyPanels`, 전투는 `battleLeftPanel`, 탐험에서 판 위 클릭은 걷기로 넘기지 않음, `leftFolded`·`party` 필드 — 옛 버튼들을 판으로 대체
- `internal/gui/hud_panel.go`: 머리줄 34px·위 10px로 진행판과 맞추고 접기 버튼을 오른쪽 끝으로, 머리줄 아래 `부대`(병력일람)·`메뉴` 줄, 접으면 머리줄만(전에는 탭 줄도 남음), `기타` 탭에서 `전투 목표`·`메뉴` 뺌, `menuButtonRect` 삭제. `battle_log.go`·`battle_log_test.go`: 기록 높이 한도에서 버튼 줄을 뺌 — 전투에서도 같은 자리에 부대·메뉴
- `internal/gui/deployment.go`: 40px 제목 띠와 `메뉴` 대신 두 판, 미니맵은 오른쪽 판 아래(`hud.go` `minimap`이 위치를 받음)
- `internal/gui/party.go`(새 파일, `shop.go`에서 `partyWindow` 옮김): 부대 창에 `장수`·`소지품` 탭. 소지품은 `Inventory`·`Warehouse`·장착 장비를 한 목록(`belongings`)으로 모아 종류별로 거르고, 고른 것의 설명·가진 수·장착·쓰는 곳, 장비는 `장비 바꾸기`
- `internal/gui/uidemo_party.go`(새 파일): `party` 데모(부대 창 두 탭) — 캡처 확인용
 위쪽 가장자리로 스크롤되고 가장자리 띠는 화면 폭의 5%, 보조목표의 도망 목적지가 보이며, 데이터가 바뀐 뒤의 저장도 묻고 불러온다

- `internal/gui/view.go`: 가장자리 스크롤이 HUD 위인지(`onMap`)가 아니라 전장 화면(`battleViewport`) 안인지로 판단, 띠 두께를 8 전장 px에서 화면 폭의 5%로 — 사용자 "병력 배치할때 스크롤이 위쪽으로 안되", "화면폭 한 5%로 하자" — 배치 화면 위쪽 40px 제목 띠가 가장자리 8px을 덮어 위로 밀 수 없었다
- `internal/core/goal_marks.go`: 보조목표의 부정 조건(`!fled:carts`)도 표시해 도망 목적지 구역을 회색으로 그림. 같은 유닛·구역이 두 번 표시되지 않게 합침. 승패 목표의 부정 조건은 여전히 빼는데, B01 "장세평 퇴각"(`defeated`, `!fled`)의 목적지는 달아나면 지지 않는 곳이라 패배 색이 틀린 뜻이 되기 때문. 부정 깃발 조건도 대기(pending)로 바르게 판단 — 사용자 "팽성에서 마차 도망갈때, 어디로 도망가는지 방향을 몰라 … 보조목표와 관련된거, 가령 목적지 같은거, 그것도 다 표현되도록". 유닛 옆 방향 화살표도 그려 봤으나 "화살표는 지워. 그냥 목적지 구역 표시면 되"로 뺌
- `internal/core/battle_goals_test.go`: B04 수레 표시·퇴로 구역 테스트
- `internal/core/api.go`: 데이터 해시(`ContentHash`)만 다른 저장은 `IncompatibleSave` 대신 `ChangedContent`, `RestoreChanged`가 지금 해시로 바꿔 불러옴(버전 상수가 다르면 여전히 거부). `internal/session/session.go`: `load`의 `Changed`(`load_changed`). `internal/gui/game.go`·`events.go`: `ChangedContent`면 "게임 데이터가 바뀌기 전의 저장입니다. 그래도 불러오시겠습니까?"를 묻고 확인 시 다시 불러옴. `cmd/srpg-cli/main.go`: 경고를 찍고 불러옴. `internal/session/session_test.go`: 테스트 — 사용자 "지금은 테스트 용이니까... sha256 다르면 경고 후 불러오기 되는걸로 하자" — json 한 글자만 바뀌어도 모든 저장이 깨졌다

## 2026-10-29 영천 황건 진영 대화는 언덕 맵 성벽 밑에서 하고 카메라가 언덕으로 흐르며, 하진의 망설임은 지문 대신 1초 쉼

- 결정: 2장 `c02-camp` 장면(숲길 옆 작은 진영 `S_turban_camp`)을 없애고 그 대사(c02-camp, t1~t3)를 `c02-hill`(`S_cliff_camp`) 앞에 붙임. 성벽 밑 진영에 파재·장량·도적 9을 두고, 나레이션(c02-05)에서 카메라가 벼랑 위 유관장으로 흐른다 · 이유: 사용자 "황건적 진영이 장면에 나오는데, 너무 작고 상황에 안맞는거 같아. 성을 포위하고 있다는 느낌이 안들어 … 거기서 슥 스크롤 해서 언덕위의 유관장을 보여주는거지" · 영향받는 문서: presentation-scene.md
- `assets/scenes/ch02.json`: `c02-camp` 삭제, `c02-hill`에 황건 배역·잡담·비트와 시작 카메라(`Ambient` `CamAt [2,5]`), c02-05 `CamAt [10,3]`. 파일을 `scene_format.py` 배치로 다시 씀 — 한 장면으로 이어 진영과 언덕이 한 맵에 있음을 보이기
- `assets/scenes/ascii/c02-*.txt`: 다시 뽑음, `c02-camp.txt` 삭제 — ch02 배치 그림을 지금 데이터에 맞춤
- `internal/gui/scene.go`: 장면을 열 때 배우 가운데를 `Ambient`보다 먼저 잡아, `Ambient`의 `CamAt`이 시작 카메라가 됨 — 이름 있는 배우가 진영과 언덕에 나뉘어 있으면 가운데 허공에서 시작했음
- `assets/scenario/hero/story03.json`(c03-f13), `docs/fullstory.md`: 하진 대사의 "(잠시 생각하다)" 지문 삭제, `assets/scenes/ch03.json`: c03-f13 앞에 `Wait 1.0` — 사용자 "(잠시생각하며)이런 지문 없애고, 실제로 딜레이 주도록 … 1초 정도"

## 2026-10-29 1장 상인은 민중 병종으로 맹목적으로 성문까지 달아난다

- 결정: 새 행동 정책 `run`(목표로 걷기만 하고 공격·기술·도구를 쓰지 않음, 목표에 닿으면 `flee`처럼 맵을 빠짐)과 새 병종 `commoner`(민중: 경보병과 같고 사거리 0이라 공격도 반격도 못 함)를 두고, B01의 장세평·소쌍을 민중·`run`으로 · 이유: 사용자 "1탄에서 장세평,소쌍이 안도망가. 도구로 회복하고 있어. 그냥 맹목적으로 도망가는 ai 추가하고, 그 ai를 쓰게 해죠", "민중은 경보병이랑 비슷한 대신 사거리 0. 즉 공격도 반격도 안하도록" — `flee`는 공격·기술만 빼서, 다친 상인은 도구 회복(회복량×0.85)이 한 칸 다가가기(3)보다 점수가 높아 제자리에서 회복만 했다 · 영향받는 문서: logic-battle-event.md 5.4, data-scenario-node.md
- `internal/content/content.go`: `Policies`에 `run`, `Leaves`(목표에 닿으면 빠지는 정책), `ValidPolicy`가 `run`에도 목표를 요구. `internal/content/validate.go`: 병종 `Range` 0 허용
- `internal/ai/ai.go`: `run`은 도구도 쓰지 않음. `internal/core/engine.go`(`leaveByFlight`), `internal/core/goal_marks.go`: `flee` 대신 `content.Leaves`
- `assets/data/classes.json`: `commoner`(민중). `assets/data/characters_story.json`: 장세평·소쌍을 `commoner`로. `assets/scenario/hero/stage01.json`: 둘의 `Policy`를 `run`으로
- `internal/ai/policy_test.go`: 다친 도주 유닛이 `flee`는 도구, `run`은 이동. `internal/core/core_test.go`: 사거리 0 병종은 공격 명령이 없고 반격하지 않음
- 확인: `_balance -stage B01 -player ai` 5/5 승. `TestRushWinsTheFirstStages`의 B01 패배(시드 3·4)는 이 변경 전에도 같음

## 2026-10-29 5장의 조조는 삿갓 쓴 베옷 차림으로 나온다

- 결정: 주막 구석의 사내와 이튿날 관문의 조조는 삿갓을 눌러쓰고 수수한 베옷을 입는다(`caocao_hat_noncombat`). 주막에서 "삿갓을 벗으며"(`c05-c5`) 삿갓만 벗고 베옷 차림은 그대로(`caocao_plain_noncombat`), 관문에서는 다시 삿갓을 쓴다 · 이유: 사용자 "조조가 삿갓 쓰고 있다고 하는데, 삿갓이 없으니 아쉽네", "삿갓 쓴 조조 그려죠" · 영향받는 문서: 없음(fullstory.md에 이미 삿갓 연출이 적혀 있음)
- `tools/doteditor/tools/civ3d.py`: `satgat_mask`·`satgat`·`satgat_heads`(낮고 넓은 원뿔 삿갓을 윤곽으로 계산해 찍음 — 꼭짓점에서 테 끝까지 옆선이 곧게, 23칸 폭, 짚색은 themeC), `POSE_HEADS`(앉으면 삿갓을 앞으로 기울여 눈가를 가림), `caocao_hat`·`caocao_plain`. 처음 그린 손 도안은 원뿔과 테가 ㄴ자로 꺾여 보여 바꿈 — 사용자 "꼭대기랑 끝이 일자로", "낮고 펑퍼짐한" `tools/doteditor/units/caocao_hat_noncombat.yaml`, `caocao_plain_noncombat.yaml`, `assets/sprites/*.spr`
- `assets/scenes/ch05.json`(`c05-inn`, `c05-gate`), `assets/scenario/hero/explore05_town.json`(삿갓 쓴 사내)
- 대사는 장비@town 안 1, 장비@inn 안 2("간이 배 밖에 나왔구먼유"), 장꾼 안 1, 유비 안 2("무도한 자일세")로 정함

## 2026-10-29 장면의 시간 전환은 어두워졌다 밝아지고, 팽성 수레는 되찾았을 때만 들판에 선다

- 결정: 장면 연출 묶음에 `Time`이 있으면 0.5초 검게 어두워진 뒤 시간을 바꾸고 다시 밝아지며, 대사는 다 밝아진 뒤 뜬다. 장면 `Ambient` 단계에 `If`(이야기 조건)를 붙일 수 있다. 이름 있는 탐험 인물은 대사·이벤트·상점·출진 중 하나를 가져야 한다 · 이유: 사용자 "술집에서 이야기하고 어두워지는데, 뭔가 깜빡하고 번쩍 하는거같아", "수레 있는 장면, 없는 장면 둘", "장비가 대사 가능처럼 표시는 되는데 눌러도 말을 안해", "단양 마을 망가진거 보여주는 용도로 울타리 바깥에 짓밟힌 밭", "스승님 그래버리니 개인적인 원한때문에 토벌하는 느낌" · 영향받는 문서: fullstory.md
- `internal/gui/scene.go`: `nightfall`, `changesTime`. `internal/scenes/scenes.go`: `Step.If`. `internal/core/cond.go`: `Engine.Holds`. `internal/gui/scenario_test.go`: `If` 조건 검사, 비트에서는 금지
- `internal/content/explore.go`: 할 일 없는 이름 있는 인물을 막음. `internal/core/explore_test.go`
- `assets/graphics/maps/S_pengcheng_field.json`: 곡식 수레·자루를 맵에서 뺌. `assets/scenes/ch05.json`(`c05-reunion`): `flag-pengcheng-carts`일 때만 수레·자루를 소품으로 놓음
- `tools/unit3d/scene_props3d.py`: `field_trampled`(말이 가로지른 조밭). `assets/graphics/maps/S_danyang_village.json`: 울타리 밖 두 곳. `assets/graphics/map/extra.png`, `index.json` 다시 묶음
- `explore05_town.json`(장비@town, 장비@inn), `explore08.json`(장꾼), `explore03.json`(조조@in) 대사. `story05.json`(`c05-d14`) 유비 대사를 "무도한 자일세…"로. `docs/fullstory.md` 반영

## 2026-10-29 간옹은 4장 팽성에서 합류하고, 1·2장 탁현에 미리 나온다

- 결정: fullstory의 간옹 합류(4장 S#2 팽성, 장세평의 말 장사 행렬)를 게임에 옮긴다. 1장·2장 탁현 거리에 간옹 NPC(복선 대사), 4장 팽성 성문 앞에 군량 수레와 장세평·간옹이 걸어 들어와 합류. 외상값 생색 대사 "탁현 외상값은 한 푼도 안 남기고 갚았네. 술집 주인이 울더군, 단골 하나 잃는다고." 8장의 임시 합류(`c08-join`)는 지운다. B04~B07 출전 최대 3 → 4 · 이유: 사용자 "난 당연히 된줄 알았는데 막상 해보니 등장 안해서 놀랐네", "진행해" · 영향받는 문서: fullstory.md
- `assets/scenario/hero/story04.json`(`c04-c1`~`c04-c6`, `c04-b5`), `assets/scenes/ch04.json`(`c04-peng`: 행렬 입장, 전령 자리), `explore01.json`·`explore02.json`(탁현 거리의 간옹), `story08.json`(`c08-join` 삭제), `stage04~07.json`(`Max` 4)
- `internal/content/stage_level_test.go`: 간옹이 B04 앞에서 합류(`c04-c6`)
- 측정(ai, 10시드, 간옹 포함): B04 10/10, B05 9/10, B06 10/10, B07 10/10(간옹 없을 때와 거의 같음)

## 2026-10-29 조운은 계교부터 유비군으로 싸운다

- 결정: 10장 진영 장면 끝에 공손찬이 "자룡, 내일은 현덕 곁에서 싸워라" 하고 조운이 합류한다. B10에서 조운은 필수 출전(최대 여섯), 11장 강가에서 "공손 장군께서 당분간 공의 곁에 있으라 하셨습니다"로 북해·서주까지 함께한다 · 이유: 사용자 "조운을 유비 진영으로 놓으면 스토리가 많이 이상할까? 어차피 이후, 북해 공융에 대한 원군, 서주 팽성 전투까지 같이할텐데", "조운 같은편 되게 고쳐죠". 정사 「조운별전」에서 공손찬이 유비를 전해에게 보낼 때 조운이 따라가 유비의 기병을 맡았다 · 영향받는 문서: fullstory.md
- `assets/scenario/hero/story10.json`(`c10-18`~`c10-20`), `story11.json`, `explore10.json`(조운 대사), `assets/scenes/ch10.json`. `assets/scenario/hero/stage10.json`: 조운을 아군 AI에서 빼고 `Required`·`Max` 6, 조운 첫 대사, 조운이 들어온 만큼 문추 기병 하나·궁병 하나를 되돌림
- `internal/core/campaign_content_test.go`: 조운 합류 확인
- 측정(ai, 10시드, 간옹·번궁·조운 포함): 거록 부목표 실패(안량 출현) 5/10, 달성 9/10

## 2026-10-29 신도성은 번궁이 성 안에서 호응해 나오고, 거록은 벼랑과 숲, 책략은 속성 지형에서 더 아프다

- 결정: B08 번궁은 성 안에 있다가 3턴에 "우리도 호응하자" 하고 성문을 열고 나온다(패배 조건 번궁 퇴각 삭제). 전투맵의 마을·진채는 붙여 두지 않는다(B08 진채 하나·마을 둘 흩어서, B09·B10 진채 한 칸씩). 거록은 계교와 같은 강·다리 대신 벼랑 능선의 길목과 남쪽 숲길, 숲이 많은 지형으로 바꾸고 길목 북서쪽에 회복용 성을 둔다. 신도성 장면 맵은 탁현·평원과 달리 성벽이 북서·남서쪽(성문 남서쪽 벽)인 새 맵. 9장 S#1 대사를 줄이고 번궁이 "더 중요한 것은 계교의 대결, 계교로 원군을 가자"고 말한다. 번궁은 지력 73, 고유 학습 풀에 화계. 책략은 속성에 맞는 지형에서 피해가 늘어난다(화계 초원 +15%·숲 +30%, 수계 여울 +15%·물 +30%, 지계 황무지 +15%) · 이유: 사용자 "신도 성 전투는 왜 신도성주가 나와있어", "마을은 있다면 클러스터링 되어 있지 않게", "신도성이나 탁군이나 평원이나 너무 비슷", "계교랑 거록이랑 지형이 똑같네", "번궁의 화계가 은근 쏠쏠히 쓸만하게" · 영향받는 문서: fullstory.md, logic-combat.md, logic-traits.md
- `internal/content/content.go`: `Rules.ElementTerrain`. `internal/core/combat.go`: `elementTerrain`(구조물 값이 있으면 그 값, 없으면 땅의 값), 공격 책략(`KindMagic`)에만. `assets/data/game.json`: `ElementTerrain`. `internal/core/core_test.go`: `TestSpellGainsOnItsElementsTerrain`
- `assets/data/characters_story.json`: 번궁 지력 32 → 73, `Learn: ["fire"]`, 소개글
- `tools/scene_maps/xindu.py`(새 파일), `assets/graphics/maps/S_xindu.json`, `index.json`. `assets/scenes/ch09.json`, `explore09.json`: 새 맵 자리
- 장면의 마지막 대사 뒤 퇴장(8장 현청, 9장 전해)을 대사 다음 줄로 옮김: 같은 줄에 두면 대사가 나오기 전에 나가 버림
- 측정(ai/rush, 10시드): B08 10/10 · 7/10, B09 10/10 · 1/10, B10 4/10(안량 출현) · 7/10(부목표 달성)

## 2026-10-29 8~11장: 신도성·거록·계교 전투와 그 사이 이야기

- 결정: 8장 평원에서 이어 신도성 전투(B08, 원소 별동대 고람, 번궁 구원) → 9장 신도성(번궁 합류, 전해의 영) → 거록 전투(B09, 안량·장합, 부목표 관우 대 안량 일기토) → 10장 계교 공손찬의 진영(백마의종 패배, 조운 첫 만남) → 계교 전투(B10, 국의의 노병·문추·원소 총공격) → 11장 첫 장면(백마, 조운과의 강가 대화)까지 새로 만든다. 판마다 보상: 번궁 합류, 계교에 안량이 오지 않음, 백마 · 이유: 사용자 "아예 전반적으로 계교전투까지… 신도,거록,계교의 전투와 이야기를 만든 다음, 장면장면의 스샷과 함께 대본, 설명, 구도 해서 html로". 정사·연의: 원소의 기주 탈취 뒤 계교에서 국의의 노병이 백마의종을 꺾음, 조운은 이 무렵 공손찬 휘하에서 유비를 만남. 영걸전 오마주로 신도 번궁, 거록 안량·장합 · 영향받는 문서: fullstory.md, presentation-explore.md, stage_lv_balance.md
- 부목표를 "10턴 안에"에서 "관우가 안량을 일기토로 물리친다"로 바꿈: 빨리 이기기만 하면 저절로 달성되어 무엇을 해야 하는지 보이지 않음. 안량이 관우에게 꺾여 체면을 잃는 것은 훗날 백마의 복선
- `tools/mapgen/layouts/hero/map08~10.txt`, `assets/scenario/hero/maps/map08~10.json`: 새 전투맵(신도성 20×16, 거록 22×16 강·다리·여울, 계교 24×18 계수·다리·여울·양쪽 진채). `assets/scenario/hero/stage08~10.json`: 새 스테이지
- `assets/scenario/hero/story08.json`(`c08-battle`), `story09~11.json`, `explore09~10.json`, `scenario.json`(9~11장), `assets/scenes/ch09~11.json`
- `tools/scene_maps/gongsun_camp.py`(새 파일), `assets/graphics/maps/S_gongsun_camp.json`, `index.json`: 계교 공손찬의 진영 장면 맵
- `assets/data/equipment.json`: `baima`(백마, 보조, 이동 +2·질주 습득, 값 임시)
- `cmd/_balance/main.go`: `-roster`(합류한 장수를 명단·출전에 더해 잰다)
- `internal/core/campaign_content_test.go`, `internal/content/stage_level_test.go`: 캠페인 B01~B10, 거록 부목표·번궁 합류·백마·계교 안량 출현 여부 확인
- 측정(ai, 10시드, 합류 장수 포함): B08 8/10, B09 10/10, B10 5/10(거록 부목표 실패, 안량 출현) · 7/10(달성)

## 2026-10-29 8장 S#3 평원현: 부임과 기주는 해설로, 이야기는 현청의 급보에서 시작한다

- 결정: 연합이 흩어진 뒤 몇 달을 건너뛴다. 평원 거리 위 해설로 평원현령 부임과 원소의 기주 탈취(한복의 죽음, 공손찬과 원소의 대립)를 알리고, 현청에서 전령이 원소 별동대의 신도 공격을 알리면서 장면이 시작된다. 평원현은 하북의 탐험 거점(현청·훈련장·상점·운하 나루)이고 관우에게 말하면 신도성으로 출진한다 · 이유: 사용자 "어차피 삼국지 영걸전 오마주야", "현승 맞이하는 장면 없애자… 시간이 지났다는 걸로", "사자가 문으로 뛰어 들어와 엎드립니다부터 시작하자". 옛 9장 첫머리(길에서 간옹이 서신을 가져옴)는 간옹이 4장부터 함께 있어 맞지 않음 · 영향받는 문서: fullstory.md, presentation-explore.md
- `tools/scene_maps/pingyuan.py`(새 파일), `assets/graphics/maps/S_pingyuan.json`(새 파일), `index.json`: 평원현 장면 맵. 북문을 탁현(서쪽 문)과 다른 오른쪽 벽에, 운하로 하북 평원 고을 느낌
- `assets/scenario/hero/story08.json`: `c08-p1`~`c08-p12`(해설·현청 급보), `c08-join`(간옹 합류 — 4장 팽성 합류를 만들기 전까지 B08이 간옹을 요구해서 여기 둠), `c08-walk`(explore08). `assets/scenario/hero/explore08.json`(새 파일). `assets/scenes/ch08.json`: `c08-p1`(평원 거리, 행인·현병), `c08-p5`(현청, 전령이 들어와 엎드림, 끝에 모두 걸어 나감)
- 8장 S#3 이후의 옛 구현을 모두 삭제(git에 남음): `assets/scenario/hero/story09~11.json`, `stage08~10.json`, `maps/map08~10.json`, `assets/scenes/ch09~11.json`, `assets/scenes/ascii/c09-*·c10-*·c11-*`, `tools/mapgen/layouts/hero/stage08~10.txt`, `docs/plan-yuanshao-retreat.md`, `scenario.json`의 9~11장. 8장은 평원 탐험에서 끝나고 관우에게 출진하면 캠페인이 끝난다 · 이유: 사용자 "여기 이후의 전투맵이나 그런거 다 이상하거든. 그냥 지워버려. 햇갈리지 않게". 시험: 캠페인 끝까지 B01~B07, 계교 원소 퇴각 시험 삭제, 합류 레벨 시험은 B07 앞에 둔 시험용 합류 노드로
- `../tools/unit3d/scene_people.py`: 보병 비전투 도트에 `prostrate`(전령이 엎드림). `tools/doteditor/units/infantry_noncombat.yaml`, `assets/sprites/infantry_noncombat.spr`
- `internal/content/stage_level_test.go`, `internal/core/story_test.go`: 간옹 합류 노드 `c09-11` → `c08-join`, 합류 뒤 다음 노드가 탐험이라 `start` 명령 없이 확인

## 2026-10-29 전투맵 땅은 반 칸마다 합성한 바닥을 깊이만큼 내려 그리고, 높은 칸 앞에 벽을 세운다

- 결정: 전투맵 땅을 맵 전체 한 장으로 굽지 않고, 반 칸마다 땅 조각·여울 돌을 32×16 한 장으로 합성(그리는 목록을 열쇠로 캐시)해 깊이(땅 0, 여울 5, 물 9)만큼 내려 먼 칸부터 그린다. 그다음 뒤 칸이 높은 변에만 뒤 칸 깊이부터 이 칸 깊이까지 벽을 먼 칸부터 그린다. 경계 번짐은 같은 깊이끼리만, 벽은 옆 칸 바닥보다 깊은 줄을 그 칸 마름모 위에서 지운다. 벽·여울 돌 원본은 128×128 틀, 반 칸 중심 (64, 64) · 이유: 층 분해(https://claude.ai/artifact/6Th8NszPxW4r6uTSDHVwz9)로 물 덧칠이 물가 번짐을 지우고, 모든 둑 조각이 위쪽을 땅 높이로 가정해 여울·물 사이 턱과 여울 뒤 경계가 5px 올라가며, 둑 조각이 먼 칸부터 그려지지 않음이 드러남. 사용자 "타일방식으로 매번 출력", "128x128 png… 64x64가 기준점", "합칠만한건 없으려나", "가리면 안될걸 가릴까봐 걱정", 시험 렌더러(https://claude.ai/artifact/3UWtDaSk9hvBimRcHCvGsB)를 보고 "훨씬 자연스러워졌네", "'같은 높이끼리만'이 맞을거 같고. 벽 가림 처리 켜는것도 맞고" · 영향받는 문서: plan-tile-floors.md(새 파일), plan-naval-water.md, data-map.md
- `internal/gui/field_floors.go`(새 파일): `floorTile`, `makeFloors`, `floorPicture`, `wallsOf`, `drawFloors`, `drawWalls`. `internal/gui/field.go`: 전투맵은 `f.ground`·`f.clip`을 굽지 않음, 그리는 순서 바닥 → 성 바닥 → 벽 → 장식 → 물 반짝임 → 다리. `internal/gui/field_water.go`: `makeBanks` 삭제. `internal/gui/field_native.go`: `drawNativeWater`는 전투맵이 아닌 맵만. `internal/gui/field_scenery.go`: 장식·물결을 놓인 칸 깊이만큼 내리고 더 가까운 높은 바닥에 덮이면 뺌(`lowerOnFloor`). `internal/content/map_graphics.go`: `GroundLevel`
- `../assets_candidate/tools/gen_banks.py`: `wall_{grass,soil,stone,weir}_{nw,ne}_{위}{아래}_{옆 칸 깊이}`, 128 틀의 `ford_bed_*`. 옛 `bank_*`·`ford_*_{grass,soil}`·`ford_back_*` 삭제. `tools/build_extra.py`: `"frame": 128` 조각은 투명 테두리를 잘라 묶고 기준점을 계산
- 확인: `_uidemo-vm -ui rivercross`·`rivercross-b07`·`naval` 캡처, `go test ./internal/gui/`

## 2026-10-29 간옹은 4장 팽성에서 합류, 콩주머니는 호로관 부목표로 공융이 준다

- 결정: 간옹을 4장 팽성에서 합류시킨다. 탁현에서 의병의 외상값을 정리하고 장세평의 군량을 챙기다가 그의 말 장사 행렬에 끼어 온다. 유비의 어머니는 다루지 않는다(사용자: "간옹이 왔다면 어머니는 어떻게 하고 왔다는건가"). 1장 탁현 거리에 NPC로 미리 세운다 · 이유: 사용자 "7탄까지면 꽤 긴데, 유관장만 나오는게 아쉬워. 대충 4탄쯤 간옹을 등장", 안 A(상인 행렬) 선택, "1장 탁군 스토리에서도 npc로 간옹 있으면 좋지 않을까? 복선 식으로" · 영향받는 문서: fullstory.md
- 결정: 판마다 다음 판의 동기(새 장수·아이템·특성)를 하나씩: 1 청룡언월도·장팔사모, 2 화계서, 3 간옹, 4 보리, 5 고정도, 6 콩주머니, 7 절영 · 이유: 사용자 "다음판,다음판을 이어갈 수 있는 원동력으로 새로운 특성/인물/아이템" · 영향받는 문서: fullstory.md
- 결정: 자웅일대검은 간옹 합류 때가 아니라 더 나중 · 이유: 사용자 "자웅일대검은 더 나중에 주도록" · 영향받는 문서: fullstory.md
- 결정: 콩주머니(조조전식, 피격마다 콩을 먹음)는 6장 호로관 부목표(상당·하내 병사가 한 부대라도 산 채로 여포를 물리침) 보상으로 공융이 준다. 콩 회복량 220 → 400 · 이유: 사용자 "콩주머니가 너무 일찍나오긴 한다. 호로관에서 부목표", "공융 어떄? 공융은 좀 인격자로", 공융의 은혜가 11장 북해 구원의 복선 · 영향받는 문서: fullstory.md, data-item.md
- `docs/fullstory.md`: 출전 원칙(4장부터 넷)과 판마다 받는 것, 1장 간옹 복선, 4장 S#2 간옹 합류, 5~7장 아군에 간옹, 6장 부목표와 전투 뒤 S#5 공융, 8장 개요의 간옹 합류 삭제

## 2026-10-29 플레이어 AI는 기대전투력으로 고르고, 쓰러질 수 있는 칸만 피하며, 목표 구역으로 간다

- 결정: 플레이어 유닛(자동 진행·측정)은 기대전투력 `max(공격, 지력) × (r^0.5 + 1)/2`(퇴각 0, 군주 ×5, 격파 대상 ×2)의 아군 합 − 적 합 증가로 공격·책략·회복·도구·버프·이동을 함께 매긴다. 군주와 장수는 쓰러질 수 있는 칸만 피한다. 승리 목표가 보내는 구역으로 남은 라운드에 맞춰 가고, 치르고 남는 일기토를 연다. 적·동맹 AI는 그대로. 정책 `greedy-12` · 이유: 옛 점수는 피해량만 보아 버프를 안 쓰고, 회복 점수(회복량 × 0.85)가 다가가기(40점 안팎)를 늘 이겨 호로관에서 궁수 둘 앞에 18라운드를 머물렀고, 도달 목표를 몰라 B07을 늘 졌다. 사용자 "HP비율^n × 공격력(문관은 정신력)을 기대전투력이라 하고… 최대화", "(hp비율^n+m)/(m+1) 같은게 더 좋지 않아?", "3번에서 난 군주만 두는게 아니라 장수도", "몇가지 안으로 시뮬레이션 돌려봐", "d안+목표 이게 맞다는 거였어. 즉 바꾸라는 거" · 영향받는 문서: system-architecture.md, logic-battle-event.md, stage_lv_balance.md, system-save.md
- 시뮬레이션(B01~B07 각 10 시드, 승리 수 합 / B06 라운드): 옛 AI 56 / 21.8. n 0.5·m 1에 위험을 장수까지 닿는 공격 전부·가중 0.3 56 / 21.4, 센 둘만 55 / 18.5, 가중 0.5 49 / 31(반응형 적 앞에서 멈춤), 가중 1은 B06에서 30라운드 맴돎, 군주만 50 / 18.5, n 1 54(B01 6/10), m 0 54, m 2 55, **쓰러질 수 있는 칸만 56 / 13.9(B06 9/10)** — 이것을 택함. 목표 이동을 더해 B07 0/10 → 10/10
- `internal/ai/eval.go`(새): `Eval`·`Player`·`ParseEval`·`chooseEval`. `ai.go`: 플레이어 유닛은 `chooseEval`. `internal/core/api.go`: `Engine.Strike`(어디서든 한 대의 기대 피해), `Preview.Damage`(대상별 피해). `cmd/_balance`·`cmd/_campaign`: `-eval`
- 테스트: internal/ai/eval_test.go(B07에서 먼저 후위를 치고, 목표가 바뀌면 유비가 빠져나감)
- 결정: 군주 ×5는 서 있는 몫에만, 패배 조건이 지켜야 하는 아군도 군주처럼, 그들을 칠 수 있는 적은 ×2 · 이유: 사용자 "b03은 왜 지는거야?" — 값 전체에 ×5를 곱하니 멀쩡한 관우가 77%인 유비에게 콩을 주는 게 황건을 치는 것보다 8배 커서 2~4라운드를 콩만 먹었고, 고순(패배 조건)을 지킬 이유가 없어 고순 혼자 본대 여섯과 싸우다 6~7라운드에 쓰러졌다. 고친 뒤 B03 7/10 → 10/10, B01~B07 69/70
- 측정(`_campaign`, 시드 1~3): B01~B08을 이김(B03에서 한두 번 다시). B07 뒤 관우 15 · 유비 14~15 · 장비 14~15(B08 기준 14). B07은 4~5라운드에 빠져나가 장비 대 여포 일기토는 열리지 않음

## 2026-10-29 전투 목표는 이벤트가 바꿀 수 있고, B07은 후위 전멸에서 여포가 나타나면 동쪽 끝 도달로 바뀐다

- 결정: 전투 이벤트 동작 `goals`(`Victory`·`Defeat`)가 그 전투의 승패 목표를 바꾼다. 바뀐 목표는 상태 `Goals`에 남고 승패 판정·목표 안내·출현 방향·`Observe`의 지도·AI가 모두 그것을 쓴다. 바뀌면 `전투 목표` 창이 다시 열린다 · 이유: 사용자 "전투 목표가 중간에 바뀌게 하는건 어때?… b07하나만을 위해서가 아니라 이벤트 중에 전투 목표가 바뀌는 걸로 엔진에서 지원" — B07은 "여포가 나타난 뒤 동쪽 끝 도달" 한 목표라, 1라운드에 동쪽 끝에 가서 기다리면 3라운드에 이기는 빈틈이 있었고 측정 AI는 도달 목표를 몰라 늘 졌다 · 영향받는 문서: logic-battle-event.md, logic-combat.md, presentation-ui.md
- 결정: B07 처음 목표는 동탁군 후위 전멸, `lords-and-lubu`(3라운드) 끝에 유비가 동쪽 끝(구역 1)으로 빠져나가기로 바뀜 · 이유: 사용자 "적 전멸 목표였다가 남동쪽 도달로 바꾸면? 여전히 3턴에 여포 나오는걸로"
- `internal/content`: `Goals`, `Action.Victory`·`Defeat`, `DefeatTargets`, `validateGoals`(스테이지 목표와 이벤트 목표가 함께 씀). `internal/core`: `State.Goals`, `battleGoals`, `goals` 동작과 이벤트, `clearBattle`이 지움, `Observe`의 지도에 반영. `internal/gui`: 목표 창이 `Observe`의 지도를, `goals` 이벤트에서 목표 창, 일기토 시험의 격파 대상
- 테스트: battle_goals_test.go(바뀐 목표·안내·저장·승리·다음 전투), campaign_content_test.go `winStage`가 지금 목표로, objective_test.go
- 측정(B07, 10 시드): 지금 측정 AI 0/10(도달 목표를 모름), 목표 이동을 아는 시험 평가식 10/10·4.6라운드. 여포가 서쪽 끝에 나타나 빠져나가기가 쉬움 — 스테이지 조정은 남김

## 2026-10-29 동탁 추격전 뒤: 새벽 앞마당과 벼랑의 작별, 조조가 패도로 돌아서는 자리

- 결정: 추격전 뒤 선물은 의천검이 아니라 절영 · 이유: 사용자 "앞에서 고정도를 얻었어. 둘다 칼이야. 칼 쓰는 사람은 유비밖에 없거든" · 영향받는 문서: fullstory.md, data-item.md
- 결정: 원소의 밤 술자리를 빼고, 원소가 새벽 앞마당에 와서 동군을 맡기며 "본초" 소리에 맹주를 내세우고 조조가 읍한다 · 이유: 사용자 제안("원소가 오는거야… 니 처지가 감히 나랑 맞먹을 수 있겠느냐"), 정사에서 형양 패전 뒤 조조가 원소에게 반쯤 기댔다가 동군태수가 됨. 원소의 하대 이유는 조조의 헛소문에 제후들이 망신당한 것 · 영향받는 문서: fullstory.md
- 결정: 절영은 새벽이 아니라 벼랑의 작별에서 준다("이제 나한테는 어울리지 않는 말이오" 다음 "천하를 버릴지언정") · 이유: 사용자 "이 작별 자리 이거 좋은거같아" · 영향받는 문서: fullstory.md
- 결정: 제후들은 술판이 아니라 몸을 사렸을 뿐(비파·술 소리 뺌), 말은 그리지 않고 대사로만, 일행은 동시에 걸어 들어오고, 말리기는 서술 대신 위협 자세·말풍선·읍 · 이유: 사용자 지적(첫 영상) · 영향받는 문서: presentation-scene.md
- 결정: 벼랑 장면은 2장 벼랑의 사본이 아니라 진영이 벼랑 남서쪽 아래에 펼쳐진 새 맵 · 이유: 진영이 벼랑 뒤(북서쪽)에 있으면 벽이 카메라 반대편이라 "절벽이란 느낌이 안 들어", 벼랑 위 풀 칸은 "초록색 초원 같은게 점점점점" · 영향받는 문서: presentation-scene.md
- `assets/scenario/hero/story07.json`·`assets/scenes/ch07.json` `c07-scene-2`: 새벽 앞마당(S_coalition_camp 조조 진영 앞마당, 거적 덮인 주검, 원소와 호위, 부목표 실패 분기). `story08.json`·`ch08.json`: 옛 원소 막사·언덕 대신 벼랑의 작별(S_cliff_coalition, 봉화대 연기, 절영 grant)
- `assets/data/equipment.json`: `jueying`(절영, 장신구, 민첩 +2·사기 +1, 질주; 수치 임시). `characters_story.json`: 하후돈 `Look.Scene` `xiahoudun_noncombat`. `internal/core/campaign_content_test.go`: 의천검 → 절영
- `internal/scenes/scenes.go`·`internal/gui/scene.go`: `Via`(Move 전에 거치는 모퉁이), 소품 이름 `#` 뒤는 같은 소품의 다른 한 개(`straw_mat#2`), 장면 효과가 높은 땅 높이만큼 올라감. `internal/scenemap/plan.go`: 높은 땅 윗면에도 그 땅의 지면 장식(`topStamps`)
- `tools/unit3d/scene_people.py`·`tools/doteditor/tools/civ3d.py`: 자세 `fist`(하후돈)·`point`(장비)·조조 `bow`. `tools/unit3d/scene_props3d.py`: `beacon`(봉화대), `rock_pile_0~2`(만들었으나 사용자 "어색해"로 맵에서 뺌). `tools/scene_maps/cliff_coalition.py`: S_cliff_coalition

## 2026-10-29 스테이지 레벨은 전투마다 +1.5, 도구도 경험치를 주고, 유비는 화계서를 든다

- 결정: 기준 레벨을 `3 + round(1.5 × (n − 1))`로(`StageLevelStep` 1.5, 실수). 경험치·일기토·특성치·승급 레벨은 그대로 · 이유: 사용자 "전투당 +1.5 정도로", "promote level은 15,30으로 그대로… 기존이 promote가 너무 빨리 와서 조정하자고 한거", "특성치도 100으로 그대로", "경험치를 24로" — 공통부 24·연의 13·가상 16 전투로 +2면 끝이 Lv75~81(거물 Lv95 안팎). +1.5면 장판파 38, 연의 끝 57, 가상 끝 62. B01부터 성장을 이어 가는 캠페인 측정에서 경험치 24로 평균이 기준 레벨을 따라감(경험치는 레벨차로 스스로 조절됨) · 영향받는 문서: stage_lv_balance.md, logic-campaign.md
- 결정: 도구로 병력·병법치를 실제로 채우면 지원 병법과 같은 경험치(대상마다 `ExpSupportPercent`, 레벨차 보정) · 이유: 사용자 "유비는 도구, 버프를 통해 경험치를 좀 획득했으면… 도구 사용도 경험치 올라가게" · 영향받는 문서: logic-growth.md
- 결정: 황보숭이 준 화계서를 유비가 곧바로 찬다(`grant` 노드의 `Character`). 병법입문은 창고로 · 이유: 사용자 "이게 유비에게 가고, 유비가 화계 책략을 쓰는게 기본", 장신구 칸이 하나라 버프와 화계 중 "화계서로 교체"를 고름 · 영향받는 문서: data-scenario-node.md
- `internal/content/content.go`: `StageLevelStep` float64. `spawn.go`: `StageLevel` 반올림. `validate.go`: `grant`의 `Character`는 장비·플레이 장수만
- `internal/core/engine.go`: `wear`(창고에서 꺼내 차고 옛 장비는 창고로) — 장비 명령·`grant`·장면 이동 `Loadout`이 함께 씀. `grant` 이벤트의 대상은 아이템 ID만. `combat.go` `useItem`: 경험치. `api.go`: 도구 미리보기에 경험치
- `assets/data/game.json`: `StageLevelStep` 1.5. `story02.json` `c02-grant`: `Character: liubei`
- `cmd/_campaign`(새, 임시): B01부터 이야기를 따라 게임 AI로 싸우며(지면 난수를 바꿔 다섯 번까지) 유관장 레벨과 유비의 행동을 낸다
- 테스트: stage_level_test.go·coalition_test.go·story_test.go 기대 레벨, campaign_content_test.go 화계서는 유비가 참, damage_pct_test.go 도구 경험치
- 측정(`_campaign`, 시드 1~3, B07 기준 Lv12 앞): 고치기 전 관우 14~15 · 장비 11~13 · 유비 7~8 → 고친 뒤 관우 14 · 장비 13~14 · 유비 13~14. 유비는 B01 견고, B03부터 초열을 판마다 2~7번, 콩을 1~7번 씀. 셋 평균이 기준보다 +1.5~2 높음. B07은 고치기 전에도 늘 지던 스테이지

## 2026-10-29 다리 밑으로 둑이 잇고 교각이 서며, 여울은 땅과 물 사이 높이의 돌바닥

- 결정: 낮은 물의 건넘목을 그림으로 잇는다. 다리는 땅 높이 그대로 두되 다리 밑에도 둑(끝은 돌 축대)을 잇고 옆판·교각·난간을 단다. 여울은 땅보다 5px 낮고 물보다 4px 높은, 맑은 얕은 물 아래 돌바닥으로, 땅 쪽에 흙 턱, 아래쪽 물 쪽에 울퉁불퉁한 돌 사이로 물이 흘러내리는 턱, 뒤쪽 물과는 물빛이 번져 섞임 · 이유: 사용자 "물타일이 내려가면서 절벽생긴게, 다리등에는 적용되지 않으니까 좀 어색해", 시안 네 가지 중 "1안이 훨씬 좋은거같아", 여울은 "평지와 강의 중간 정도… 돌이랑 진흙이랑 징검돌다리 이런거 있어서 살짝 솟았다는 느낌", "물 담그기는 냅두고", 첫 구현을 보고 "여울이랑 물이 너무 칼같이 구별", "땅에 가깝게 높아보여", "울퉁불퉁한 돌이 있어서 거기로 물이 흘러내린다는 느낌", "흙탕물처럼 보여서", 강둑 절벽이 여울과 만나는 곳에서 "b가 a를 끊어버리잖아", 층 분해 페이지(https://claude.ai/artifact/6Th8NszPxW4r6uTSDHVwz9)를 보고 "3번이 덮어버리면서", "7번 턱의 높이가 여울의 높이-물이 높이", "11이 밑에가 깍여있어", "위쪽 찍고 그 다음에 아래쪽" · 영향받는 문서: plan-naval-water.md, data-map.md
- `../assets_candidate/tools/gen_banks.py`: `bank_*_stone`·`bank_*_ford`·`ford_*_{grass,soil}`·`ford_back_*`·`ford_bed_0~7`·`bridge_edge_{se,sw}`·`bridge_rail_{nw,ne}`, 옆 칸 여울로 넘어가는 부분은 5px만 남는 강둑 `bank_*_cutford`. `tools/build_extra.py`로 `extra.png`에 묶음
- `internal/gui/field_water.go`: `fordDepth`, `makeBanks`가 다리 칸·여울 칸에도 면을 깖, `bridgeEdges`, 여울 칸에는 배를 띄우지 않음. `internal/gui/field.go`: 여울 칸 `Lift` −5, 다리는 둑 다음에 그림(`bridges`). `internal/gui/field_native.go`: 다리를 `bridges`로. 앞에 여울이 있는 물 칸엔 어두운 선을 긋지 않음, 둑 조각을 먼 칸부터(같은 칸에선 턱 → 절벽 → 선) 그림, 여울 턱은 옆 칸이 땅일 때만 자름, `Field.Layers`(디버그 층 스위치). `internal/gui/field_native.go`: 낮은 물 위에도 여울 쪽 경계 번짐(`content.FordTile`)은 다시 얹음 `internal/gui/field_wade.go`: −5에서도 발이 잠김
- 확인: `_uidemo-vm -ui rivercross`(폭 2칸 강에 다리·여울)와 `-ui rivercross-b07`(B07 맵) 캡처

## 2026-10-29 외길을 지나는 AI 유닛이 한 칸 간격으로 함께 걷는다

- 결정: 클레임 리플레이에서 두 행동이 둘 다 걸어서 지나가기만 하는 칸은, 뒤 행동이 앞 행동 전체가 아니라 앞 유닛이 그 칸을 벗어날 때까지만 기다린다(`Trail`) · 이유: 사용자 "좁은 지형이 있고, 거기에 지나갈 때는 무조건 한번에 1기씩 애니메이션이야? AI 특유의 동시에 전개되는게 갑자기 안되네" — B07 여포군 보병이 3턴부터 움직이며 다리 앞에서 한 기씩 건너는 게 드러남 · 영향받는 문서: presentation-replay.md
- `internal/replay/replay.go`: `Claim`(키, 걷는 칸인지, 머무는 걸음 `In`~`Out`), `Action.walk`(첫 명령이 이동일 때의 길), `Action.Trail`·`Lead`. `Link`가 걷는 칸끼리는 `After` 대신 `Trail`에 출발 지연(걸음 수)을 적음
- `internal/gui/player.go`: 재생 시계 `clock`, 행동별 출발 시각 `started`. `Trail`의 앞 행동이 출발하고 `걸음 × stepTime`이 지나야 출발
- `internal/replay/replay_test.go`: 한 줄로 걷는 셋이 한 걸음씩 뒤따름, 멈춘 자리를 지나가는 행동은 도착 뒤, 싸우는 칸을 지나가는 행동은 끝까지 기다림
- 확인: B07 3턴부터 끝내기를 반복하는 GUI 캡처에서 4턴 동탁군 차례에 기병 둘·셋이 한 칸 간격으로 다리 위를 함께 건넘

## 2026-10-29 동탁 추격전: 제후군은 이벤트 안에서 물러나고, 여포군은 서쪽 끝 세 줄, 여포는 가운데

- 결정: 컷신 단계 `leave`(가장 가까운 가장자리로 걸어가 도주 퇴장)를 두고, B07 제후군의 퇴각을 `ai flee` 대신 이벤트 안의 `leave`로. 여포군은 x 0~2 세 줄, 여포는 가운데 (1,5). 보병은 나오자마자 `assault` · 이유: 사용자 "여포가 나온 후 아군을 이동시키면 한참 메뉴가 안 뜸… 우하단 동맹군이 후퇴하는 모션이 나오더라", "여포가 나타난 후 동맹군이 여포다! 하더니 바로 후퇴하는 모션이 이벤트에 나온 후, 조조군과 유비군만 남아 대화", "여포가 병력들 가운데 있어야", "같이 나온 보병들은 멀뚱멀뚱", "좌상단 끝 3,2,1칸" · 영향받는 문서: logic-battle-event.md, data-scenario-node.md, presentation-replay.md
- 원인: 제후군은 동쪽 끝(구역 2) 바로 옆에 나와 일부가 이미 구역 안에 서 있었고, 도주 유닛의 퇴장(`leaveByFlight`)은 다음 정산, 곧 플레이어의 다음 행동 뒤에 일어났다. 그래서 이동 직후 제후군 열세 명의 퇴장 연출이 끝날 때까지 행동 메뉴가 나오지 않았다
- `internal/core/battle_events.go`: 컷신 단계 `leave` — `leave`가 가장 가까운 가장자리 칸(걸을 수 있는 땅)까지의 길을 내고 HP 0, `fled-` 기록. `internal/gui/cutscene.go`: 그 길을 걸은 뒤 `leaveField`. `internal/content/battle_events.go`·`content.go`: 검증·설명. `internal/cli/play.go`: `[컷신] X 퇴장`
- `assets/scenario/hero/stage07.json`: `lords-and-lubu`의 `ai lords flee`를 `cutscene [[leave lords]]`로. 여포군 스폰을 앞줄 x 2(보병 (2,3) · 기병 (2,4~7) · 보병 (2,8)), 가운데 줄 x 1(보병 (1,3)(1,4) · 여포 (1,5) · 보병 (1,6)(1,7)), 뒷줄 x 0(보병 (0,3)(0,5~7))으로. 화면에서 여포 바로 위 칸 (0,4)는 비워 보병이 여포의 HP 막대·말풍선을 가리지 않게 함. 보병 그룹 `lubu-foot`을 `lubu`에 합치고 `assault`로 나오게 해 `lubu-foot-follows`를 지움. 여포는 그대로 `react` → 4턴 `assault`(`lubu-charges`). 스폰 순서가 기병 → 여포 → 보병이고 여포가 다리 줄(y 5)에 있어, 텍스트 추적에서 4턴에 여포가 보병보다 먼저 다리를 건넘을 확인

## 2026-10-29 여울·평지 지형, 여울에 잠기는 발, 탁군 길은 평지로

- 결정: 땅에 여울(`s`, 강가의 얕게 물이 고인 땅)과 평지(`l`, 초원과 황무지 사이)를 더하고, 물·황무지처럼 경계가 번지게 한다. 여울은 청회색에 늪 수풀, 여울에 선 병력은 발이 잠긴다. 탁군 전투의 황무지 길은 평지로 · 이유: 사용자 "강근처, 약간 축축한 지형", "초원보다는 흙길이지만 황무지만큼 척박하진 않음", "여울을 좀 더 청회색 계열로… 늪지대처럼 수풀", "병력이 들어갔을 때 발쪽이 여울에 잠긴 스프라이트", 시험 화면에 "아주 좋아" · 영향받는 문서: data-map.md, tools-build.md
- `assets/data/game.json`: 땅 `s` 여울·`l` 평지, `Surface`를 층 번호로(물 0, 여울 1, 초원·성벽·성문 2, 평지 3, 황무지·성내·절벽 4). 나무·울타리는 여울·평지에, 마을·주둔지·성은 평지에도 선다
- `assets/data/terrain.json`: 평지는 모든 병종군 배율 1·비용 1, 여울은 기병만 0.9·2(말이 빠짐), 나머지 1·1
- `internal/content/map_graphics.go`: 땅 조각 층을 물·초원·흙 세 층 고정 표에서 층 목록(`groundSets`)으로 — 조각마다 가장 높은 꽉 찬 층을 바탕으로 그 위 층들의 경계를 얹음
- `tools/build_ground_tiles.py`(새 파일): 여울(70~84)·평지(86~100) 경계 조각을 `tileset.png`에 추가. 경계 모양과 음영은 물가(풀→물)·흙길(흙→풀) 조각에서 빌려 이음이 기존 땅과 같고, 여울은 얕은 물빛 청회색에 물 반짝임·진흙 점
- `../assets_candidate/tools/gen_scenery.py`: 여울 늪 수풀 네 종(갈대 덤불·부들·사초·가는 갈대, 아스키 도트), 여울·평지 장식 묶음과 밀도. `internal/scenemap/scenery.go`: `s`는 ford, `l`은 plain 장식, 늪 수풀은 모여 남
- `internal/gui/field_wade.go`(새 파일), `field.go`, `player.go`, `field_render.go`: 여울 칸의 병력은 0.2초에 걸쳐 잠김. 유닛을 따로 그린 판에서 수면선(보병 5px, 기병 8px) 아래를 물빛으로 물들이고 반쯤 비치게 하며, 수면선에 물결 고리(뒤 반은 유닛 뒤, 앞 반은 앞, 걸을 때 빨리 퍼짐), 그림자는 옅게. 새 스프라이트 없이 모든 병종·8방향에 맞음
- `internal/gui/uidemo_wading.go`·`uidemo_terrainmix.go`(새 데모): 여울을 걸어 드나드는 장수 둘과 8방향 보병·기병 줄 / 여울·평지가 다른 땅·성벽·절벽·나무·마을·울타리와 맞붙는 견본 20개
- `internal/gui/hud.go` 미니맵 색, `internal/cli/ascii.go`·`tools/mapgen/stage_ascii.py` 배치 글자
- `assets/scenario/hero/maps/map01.json`, `tools/mapgen/layouts/hero/stage01.txt`: 탁군의 황무지 28칸(가운데 길)을 평지로

## 2026-10-29 1장 상인 한 칸 성문 쪽으로

- `assets/scenario/hero/stage01.json`, `tools/mapgen/layouts/hero/stage01.txt`: 장세평 (7,5)→(7,6), 소쌍 (8,5)→(8,6) — 사용자 "1탄 난이도를 낮추려고 상인을 성과 가깝게 한 칸씩 좌하로". 등무와 한 칸 벌어지고 성문까지 한 칸 줄어듦. `_balance -seeds 20`: rush 11/20 → 15/20(유비 퇴각 9 → 5), ai 20/20 그대로, 두 방식 모두 상인은 전후 한 번도 쓰러지지 않음.
- `cmd/_balance/main.go`: 퇴각 집계에 AI 아군(상인·수비대)이 쓰러진 것도 `npc:` 접두어로 넣음(성문으로 빠져나간 `flight`는 뺌) — 호위 대상이 쓰러지는지 볼 수단이 없었음.

## 2026-10-29 전투 목표 안내

- `internal/core/goal_marks.go`(새 파일, `Engine.GoalMarks`·`DuelMarks`): 승리·패배 조건에서 표시할 유닛·지역을 뽑음 — `defeated`는 그 유닛, `at`은 지역과(아군이면) 그 유닛, 군주는 늘 패배 쪽, `flag`·`round`가 아직이면 `Pending`. 남은 일기토의 장수(대신 나설 `Supports` 포함)와 그 묶음 번호 — 사용자 "특정 위치 이동, 특정 세력 전멸이 뭔지 인식이 안 됨", "장비랑 등무가 싸우는지 정원지랑 싸우는지 헷갈림".
- `internal/content/content.go`(`Stage.DuelGroups`, 테스트 `duel_groups_test.go`): 장수를 공유하는 대진을 한 묶음으로 — 사용자 "한 장수가 여럿이랑 걸리면 같은 색".
- `internal/gui/field_goals.go`(새 파일), `field.go`: 바닥 색칠 뒤·지형물과 유닛 앞에 초록 실선(승리, 칸 크기)·빨강 실선(패배, 작게)·묶음 색 점선(일기토, 그 사이) 마름모와 지역 테두리, 미충족 목표는 흐리게. 일기토 색은 초록·빨강과 먼 다섯 색을 돌려 씀 — 사용자 안(겹친 유닛은 큰 초록·작은 빨강, 표시는 배경 타일 쪽 발밑, 일기토는 좀 작게, 먼 색). 조건 줄에 올리면 밝아지는 연결은 "정 필요하면 나중에"로 뺌.
- `internal/gui/game.go`(`speedButton`·`openObjective`), `hud_panel.go`, `windows.go`: 속도 버튼 오른쪽 `전투 목표` 버튼이 옛 `작전 개요` 모달(이름을 `전투 목표`로)을 염. 모달 아래에 `목표 안내` 토글(켜지면 금빛으로 눌림)과 `확인`, 아무 데나 클릭해 닫던 것은 확인으로만 — 사용자 "그냥 '목표'라고 쓰니 투박", 토글을 모달 안에 두자는 안. 일기토 줄 앞에 그 색의 점선 마름모. `internal/storage/storage.go`(`NoGoalGuide`): 기본 켜짐, 설정에 저장. `scene.go`: 장면은 안내를 지움.
- `internal/gui/windows.go`(`objectiveDuels`): 일기토 줄에 대신 나설 장수(`Supports`)도 `장비·관우·유비 대 여포`처럼 함께 — 사용자 "호로관은 관우 대 여포, 유비 대 여포도 있는데 전투 목표에 안 나옴".
- `internal/gui/uidemo_goals.go`(새 파일): 표시 중심으로 카메라를 맞추고 100프레임에 모달을 열어 150프레임에 안내를 끄는 데모. `docs/presentation-ui.md`·`presentation-replay.md`: 설명.

## 2026-10-29 5장 주막 조조 사연·관문 대사

- `assets/scenario/hero/story05.json`: 주막 c05-c6~c05-c8을 동탁의 폐립·처형·약탈·궁녀 희롱·도굴로, c05-c8b("칼을 품고 그자를 찾아갔으나…… 실패했소")·c05-c8c(관문마다 붙은 방 → 짚신) 추가, 옛 "벼슬을 내리겠다" 줄은 뺌 — 조조가 쫓기는 몸임을 S#3에서 밝히라는 사용자 요청(연의의 칼 품은 암살 시도).
- `assets/scenario/hero/story05.json`: 관문 나레이션 c05-d1·c05-d2·c05-d5를 지우고 관리(official)·유비의 범양 대화 c05-d1~d1d, 관리의 "지나가시오" c05-d2로 — 지문을 나레이션으로 두지 않음(사용자 승낙 대본).
- `assets/scenes/ch05.json` `c05-inn`·`c05-gate`, `ascii/c05-gate.txt`: 일행이 동쪽 길에서 걸어와 방문 앞에서 조조가 삿갓을 눌러씀, 관리 놀람(!), 유비·관리의 장날 말풍선 사이 조조가 관리 곁을 지나 문 안으로, 일행이 문을 지나 서쪽 길을 걷다 이정표(`Prop` milestone) 앞에 멈춤, 관문 병사 둘.

## 2026-10-28 주막 이벤트 뒤 바로 관문

- `internal/content/content.go`(`Node.Leave`), `internal/core/engine.go`(`back`), `internal/content/explore.go`(검사): `back` 노드에 `Leave`를 두면 탐험을 마치고 탐험 노드 다음으로 이야기가 이어진다 — 탐험은 출진(`Sortie`)으로만 끝났는데, 사용자가 주막 이벤트가 끝나면 바로 관문으로 가길 원함.
- `assets/scenario/hero/story05.json` `c05-c-back`: `Leave`. `explore05_town.json`: 관우의 "그만 쉬시겠습니까?"(`Sortie`)를 뺌 — 승낙 안 된 줄.
- `docs/presentation-explore.md`: `Leave` 설명, 5장 흐름.

## 2026-10-28 전투맵 성문을 마당형으로, 관문은 열린 성문으로, 성벽 뒤 유닛은 30%로

- 결정: 전투맵 성문을 옛 관문의 옹성 구성(바깥 문 → 담 두른 마당 → 안쪽 문)으로 다시 그려 닫힌 성문과 열린 성문 둘 다 이 그림으로, 관문(`p`)은 열린 성문과 같은 그림으로 한다. 성벽·성문 뒤 유닛은 30%로 비친다 · 이유: 사용자 "관문이랑 성문이랑 사실상 같은 것", 옛 관문은 높이·색·느낌이 성벽과 달랐고, 문루 지붕 시안은 성문 칸의 유닛을 가렸다. 시안 셋 중 "붉은 기둥 네 개와 검은 평판" 안을 엇갈리지 않게 맞춰 고름, "문을 타일이 허용하는 거의 최대로", "디자인 딱 좋아". 반투명은 끄고 보니 "병사 30, 원래꺼 70 정도로" · 영향받는 문서: data-map.md, tools-build.md
- `../assets_candidate/tools/gen_pixel_kit.py`: `battle_gate(orient, closed)` — 양옆 담, 앞뒤 문 담에 마당 폭 거의 끝까지 넓은 아치(높이 13px), 문 위 성벽에 붉은 기둥 넷과 검은 평판. 닫힌 것은 문짝 두 장이 아치를 막고 열린 것은 마당 양옆 담에 붙음. 우진각 지붕 문루 삭제. `cut`이 칸 중심선(u+v=2) 뒤를 조각 0, 앞을 조각 3으로 — 작은 칸으로 자르면 유닛과 같은 깊이인 옆 조각이 앞 문 담을 유닛 발밑에 깔았다
- `internal/gui/field_native.go`: 관문(`g`)과 열린 성문을 `gatehouse_open_*`로 그리고 그 아래 성 안 바닥, 옛 `sites.png`의 `gate_u`·`gate_v`는 쓰지 않음. 성벽 조각은 셋 다(닫힘·열림·관문)를 이어지는 벽으로 봄(`sameBlock`) — 열린 성문 옆 성벽이 끝면 테두리를 그려 세로 선이 생겼다
- `internal/gui/field.go`: 성벽·성문 조각(`drawItem.hides`)이 유닛을 덮으면 유닛을 30%로 덧그림(나무·집은 50%), `drawUnit`이 투명도를 받음. 반투명은 유닛 단위라 50%가 성문 담 위로도 비쳤다
- `assets/scenario/hero/maps/map01.json`·`stage01.json`: 탁군 관문 두 칸 → (7,10) 한 칸, 상인이 피하는 구역도 한 칸 — 성문 둘이 나란히 있어 어색했다
- `assets/scenario/hero/maps/map04.json`: 팽성 성문 (15,11)을 열린 상태로 — 닫힌 성문 위에 도겸이 서 있었고, "적이 성 안에 들어옴" 패배 조건이 일어날 수 없었다
- 시도 후 거절: 성(`o`) 칸을 성벽 안이면 관청, 들판이면 네 방향 문의 작은 성으로 — "성 별로야, 관청도 너무 높고 뾰족해". 성은 옛 그림 그대로

## 2026-10-28 전투맵 성벽·성문과 실내 벽·바닥을 2:1 도트로 (스킬 pixel-iso-kit)

결정 · 장면 성벽(10-24)과 같은 방식 — 칸 단위 상자의 면에 아스키 도트 무늬를 놓아 가로 2·세로 1로 쌓는 렌더 — 으로 전투맵 성벽 16종·성문, 실내 벽 3종·바닥 4종을 다시 그린다. 성문은 작은 칸마다 서던 작은 집 네 채 대신 게임 칸 하나를 덮는 문루 하나를 작은 칸 넷으로 자른다 · 이유: 사용자 "훨씬 깨끗하다. 적용하고 이거 스킬로 만들어죠" · 영향받는 문서: tools-build.md
- `../assets_candidate/tools/gen_pixel_kit.py`(새 파일): `gen_pixel_castle.py`의 렌더러로 전투 성벽 `wall_clean_00..15`(높이 16px, 트인 면에 성가퀴, 위 길에 판석, 닫힌 면·그 사이 대각선 이웃을 문맥으로 그려 이음에 잉크 점 없음), 성문 `gatehouse_closed_{u,v}_{0..3}`(아치·닫힌 문·우진각 지붕, 작은 칸 위치 du+2dv), 실내 벽 `wall_{plaster,lacquer,canvas}_{nw,ne}`(높이 34px), 바닥 `floor_{wood,carpet,palace,tent}`(64×32, 도트 하나가 화소 하나). `--compare`로 지금 그림과 나란히, `--write`로 tiles에. 실내 벽·바닥은 props.png와 같은 이름이라 extra.png가 덮음
- `internal/gui/field_native.go`: `gateSprite`가 게임 칸의 이웃으로 성벽 방향을 정하고 칸 안 위치로 조각을 고름 — 작은 칸 이웃으로 정하면 v 방향 성문의 한쪽 조각이 옆 성문 칸을 성벽으로 보아 u로 잘못 골랐음
- `internal/gui/field_state_test.go`: 성문 그림 이름
- `../assets_candidate/tools/gen_walls.py`·`gen_gatehouse.py` 삭제, `build_all.py`·`viewer_template.html`(성문 C 시연)에서 뺌, 옛 `gatehouse_closed_{u,v}`·`gatehouse_floor_{u,v}` 삭제 — 새 그림을 덮어쓰지 않게
- `assets/graphics/map/extra.png`·`index.json`: `build_extra.py`로 다시 묶음
- `.claude/skills/pixel-iso-kit`(새 스킬): 방법·문맥 몸체·자르기·비교·적용 순서

## 2026-10-28 동탁 추격전 조정

- 결정: B07 조조군 일반병을 줄이고(장수 다섯은 그대로), 여포를 `boss`에서 `commander`로, 여포군을 기병 앞줄 → 여포 → 보병 뒷줄로 세워 한 칸 다리를 줄지어 건너오게 한다 · 이유: 사용자 "조조군이 너무 많다, 장수는 두고 병사를 줄여", "여포 등장 뒤의 다리가 두 칸, 한 칸으로", "여포 병사 조금 줄여", "여포는 대장 직급 유지, 기병 앞줄·여포·보병 뒷줄로, 여포가 먼저 돌진하지 않게" · 영향받는 문서: fullstory.md
- `assets/scenario/hero/stage07.json`: 조조군 일반병 여섯 → 셋(보병 (15,4)(15,6)·궁병 (16,5), x=16 줄의 보병 둘·궁병 하나를 뺌) — 동맹군이 후미를 다 가져가 주력 몫이 없었다.
- `assets/scenario/hero/stage07.json`: 여포 `Rank` `boss` → `commander`(Lv15 스테이지에서 Lv23 → Lv19, 주력 Lv15). 레벨 표(3절)의 '거물'에서 벗어남 — 사용자가 "대장" 직급을 원했고, 이 전투는 여포를 이기는 길이 아니라 물러나는 전투라 L+8은 한 대에 관우 HP의 60%를 깎았다. 일기토 둘(하후돈 패배·장비 무승부)은 결과가 정해져 있어 레벨과 무관하게 그대로다.
- `assets/scenario/hero/stage07.json`: 여포 원군 경기병 여덟 → 기병 넷 + 보병 열하나. 앞줄 기병 (5,3~6), 그 뒤 줄에 여포 (4,3)과 보병 다섯 (4,4~8), 맨 뒷줄 보병 여섯 (3,3~8), 모두 `SE` — 사용자 "여포 줄에 보병 다섯, 뒷줄 보병을 여섯으로". 서쪽 둑은 x 0~2가 y 4~7 네 칸뿐이라 진형을 x 3~5로 옮김. 여포를 줄 가운데 (4,5)에 두면 뒷줄 (3,4)의 보병이 화면에서 여포 바로 위에 겹쳐 HP 막대가 여포 머리를 가리고 말풍선이 보병을 가리켰다 — 여포를 줄의 북쪽 끝 (4,3)에 두어 그 칸 (3,2)가 나무가 되게 함. 스폰 순서는 기병 → 여포 → 보병(같은 진영은 스폰 순서대로 움직인다). 여포와 보병(그룹 `lubu-foot`)은 `react`로 나와 3턴에는 기병만 달려오고, `lubu-charges`(`flag:flag-b04-lubu`, `round>=4`)가 기병·여포(그룹 `lubu`)를, `lubu-foot-follows`(`round>=5`)가 보병을 `assault`로 — 보병을 여포와 같은 턴에 풀면 여포가 다리 앞에서 멈춘 사이 보병이 다리에 먼저 올라 여포를 가로막았다. 텍스트 추적으로 4~6턴 내내 기병 → 여포 → 보병 순으로 다리를 건넘을 확인.
- `tools/mapgen/layouts/hero/map07.txt`, `assets/scenario/hero/maps/map07.json`(`mapgen.py`로 다시 만듦): 서쪽 강의 다리 (7~8,5~6) 두 줄 → (7~8,5) 한 줄, (7~8,6)은 물. 이 칸을 가리키는 구역·이벤트 좌표는 없다.
- `assets/scenario/hero/stage07.json`: `Night: true`, `Lights` 횃불 셋 — 다리 양쪽 머리 (6,5)(9,5)와 동쪽 연합 진영 (27,7) — 사용자 "앞 이야기가 계속 밤이니 밤 전투로". 그림만 바뀌고 규칙은 같다. 유닛마다 작은 빛이 있어 양 군이 보이고, 다리 머리의 횃불이 여포군이 건너올 한 칸 다리를 밝힌다.
- `tools/mapgen/layouts/hero/stage07.txt`: `stage_ascii.py dump B07`로 다시 찍음(횃불 `Trc` 포함).
- `assets/data/terrain.json`: 기병 무리(경기병·중기병·궁기병·노기병·친위대·전차대)의 다리(`b`) 배율 1.1 → 1.0 — 사용자 요청. 다리가 있는 전투맵은 B07(2칸)과 B10 계교(2×2)뿐이라 B10의 다리 위 기병(백마의종·원소 기병 양쪽)이 10% 약해진다.
- 측정(Lv15, 10~20 시드): `_balance` rush 0/10 · ai 0/10(전·후 같음 — 두 플레이어 모두 동쪽 끝으로 물러나지 않고 끝까지 싸워 이 전투에는 맞지 않는 잣대). 임시 도구로 "여포가 나온 뒤 N라운드부터 동쪽으로 물러나는" 플레이어를 돌린 승률: 4라운드 10/10 → 20/20, 5라운드 5/10 → 15/20, 6라운드 2/10 → 8/20(앞은 바꾸기 전). 조조군 병사를 둘까지 줄이면 4라운드 퇴각도 18/20으로 떨어진다(서영 Lv19의 치명타 두 번에 유비가 쓰러지는 판) — 그래서 셋에서 멈춤. 보병 아홉을 더한 뒤(20 시드): 3라운드 20/20, 4라운드 20/20, 5라운드 15/20, 6라운드 7/20 — 보병은 다리 뒤에 줄을 서서 퇴각하는 쪽에 거의 닿지 않는다.
- `assets/scenario/hero/stage06.json`: 호로관 여포 `Rank` `boss` → `commander`(Lv13 스테이지에서 Lv20 → Lv16) — 사용자 "호로관 여포도 대장으로". 레벨 표의 '거물'에서 벗어남(같은 이유). `_balance` 10 시드: rush 2/10 · ai 5/10, 바꾸기 전과 같음 — 진 판은 여포에 닿기 전 진채·서량 기병에서 무너진다. 목순·방열 일기토와 삼영전(`hulao-brothers`)은 그대로 열리고 삼영전 뒤 여포가 퇴각해 승리함을 확인.
- `tools/mapgen/layouts/hero/stage06.txt`: `stage_ascii.py dump B06`로 다시 찍음.
- `internal/gui/events.go`: 이벤트로 나타난 유닛의 기록 줄 — 아군 편이 아니면 "○○ 등장"(CLI와 같은 말), 아군 편은 그대로 "○○ 동맹군 합류". 적 원군도 "동맹군 합류"로 찍혔다(B07 여포군). `internal/gui/battle_log_test.go`: `TestUnitArrivalLogNamesTheSide`.
- B10 다리 위 기병 10% 약화(위 `terrain.json`)는 사용자가 받아들임.

## 2026-10-28 7장 낙양이 타는 밤 대본

- 결정: 7장 S#1~S#6을 밤으로 잇고, 제후들은 여포가 무서워 추격을 꺼리고, 조조가 조용히 나간 뒤 유비가 따라 나와 자청한다. 승전 술자리는 대화창 없이 말풍선으로, 진영에는 보물 소문이 돈다 · 이유: 사용자 7장 설계("여포를 보고 겁난 연합군이 추격을 꺼려", "삼형제도 같이 쫓겠습니다", "동탁추격전 직전은 밤", "술자리는 말풍선 식으로") · 영향받는 문서: fullstory.md(7장, 먼저 고침)
- `assets/scenario/hero/story07.json`: 나레이션 노드 c07-a1(옛)·a2~a5·b5·b7·c2·c6·d4·e1을 지우고, S#1은 조조의 "(잔을 내려놓으며) ……." 한 줄로, S#3에 원소·원술의 여포 걱정과 조조의 두 번째 설득·원소 "오늘은 그만합시다"(c07-c4a·c4b), S#4는 유비의 자청·장비의 병력 걱정·조조 두 줄·장비·관우(c07-d3a·d6) — 지문을 대사·연출로
- `assets/scenes/ch07.json`: c07-start는 원소·원술·교모·한복·공융·장비·관우의 말풍선과 조조의 "…"로, c07-out·c07-out2는 맹주 막사 앞(조조·유비가 막사 문에서 걸어 나옴, 전령은 동쪽 길에서 달려옴, 조조는 서쪽 길로 떠나고 형제는 "?"), c07-council은 조조·유비가 문으로 들어오고 제후들 웅성임, c07-march는 밤 — 사람은 걸어 들고 나게
- `assets/scenario/hero/story07.json`·`assets/scenes/ch07.json` c07-start: S#1 자리표시 줄 "(잔을 내려놓으며) ……"를 승낙받은 대본으로 — 말풍선 뒤 공손찬이 자리에서 일어나 삼형제에게 걸어와 두 마디, 장비의 "검은 놈으로 주슈", 관우가 잔을 들고(동작), 조조가 자리에서 걸어와 유비에게 잔을 권하고 "바람이나 쐽시다". 조조 자리를 (7,1)로 옮기고 하후돈은 그 뒤 (8,0), 삼형제는 (9,6)(10,6)(9,7), 공손찬은 (7,5)에 앉힘 — 걸어오는 길이 탁자·사람에 막히지 않게
- `assets/scenario/hero/explore07.json`: 진영을 밤("반동탁 연합 · 진영, 밤", `Time` night)으로, 보물 소문(흘리며 간다·무기를 버렸다·천자의 보물·조조가 혼자 차지하러 갔다), 원술 진영 병사·맹주 막사 위병·순찰병·조조 진영 남은 병사(조련 동작 뺌)·교모·공융·공손찬·원소군 병졸·장비·관우 출진 말 — 조조가 흘린 소문이 진영을 움직이는 밤

## 2026-10-28 5장 고을 탐험·주막·맹주의 막사 대사

- 결정: 5장 소문 나레이션(S#2)을 작은 고을 탐험으로, 주막(S#3)·맹주의 막사(S#6·S#7)의 나레이션을 대사·연출·장면 전환으로 바꾼다. 장면이 바뀔 때 앞 장면이 어두워진다 · 이유: 사용자 승낙 대본(fullstory.md 5장 S#2·S#3·S#6·S#7), "지문이 나레이션이 되어 버렸다", "손견이 나가자마자 전령이 오는 게 광속 같다, 페이드" · 영향받는 문서: fullstory.md, presentation-scene.md, presentation-explore.md
- `internal/gui/scene.go`·`scenario.go`: `scene.leaving`과 `nextNode` — 다음 노드가 다른 장면을 열면 지금 장면을 0.5초 동안 검게 어둡게 한 뒤 넘어간다(새 장면은 전처럼 검은 화면에서 밝아진다). 탐험·전투로 넘어갈 때는 그대로 — 손견이 나간 뒤 며칠 뒤 장면이 바로 붙어 보였다.
- `assets/graphics/maps/S_road_town.json`(새 파일), `index.json` `maps`, `plan/draft_maps/road_town/gen.py`(새 파일): 낙양 가는 길의 작은 고을 — 사용자 "고을맵은 새로 작은걸로".
- `assets/scenario/hero/explore05_town.json`(새 파일): 고을(소문 다섯, 노인은 이벤트로 `c05-hejin`, 장비의 하진 소문 뒤 대사, 관우, 행인·장꾼 혼잣말)과 주막(주인 상점, 삿갓 쓴 사내 = 조조 이벤트, 이벤트 뒤 관우 `Sortie` "그만 쉬시겠습니까?"). 주막 앞 아이는 아이 그림이 없어 주막 머슴으로 — 탐험은 출진으로만 끝나서 관문으로 가는 길을 관우의 `Sortie`로 둠.
- `assets/scenario/hero/story05.json`: `c05-rumor`·`c05-b1~b5`(소문 나레이션)와 본문의 `c05-inn`을 탐험 노드 `c05-walk-town`(Opening `c05-e-town`)으로, 주막은 `Events`(`c05-e-town`·`c05-e-elder`·`c05-inn`)로. 주막의 나레이션 `c05-c1`·`c12`·`c17`을 빼고 `c05-c5`(삿갓을 벗으며)·`c17`(먼저 눕겠소)·`c17b` 주인·`c20` 장비를 대사로. 맹주의 막사: 조조의 귀띔 `c05-f1·f1b·f1c`, 유비 `f3` "그렇군요. 알겠습니다.", 공융과 통성명 `f3a~f3c`, 나레이션 `f7`·`f11`·`f16`·`f21`·`f24`·`f28` 삭제, `f16` 조조 "자, 자, 공로. 고정하시오.", 화웅 급보 `f21`·원소 `f21a`·손견 자청 `f22`·원소의 명 `f22a`·보급 `f22b`, 새 장면 노드 `c05-council2`("며칠 뒤 · 맹주의 막사").
- `assets/data/characters_story.json`: 대사용 `innkeeper`(주인), `town_elder`(노인).
- `assets/scenes/ch05.json`: `c05-rumor` 삭제, `c05-e-town`(형제가 동쪽 길에서 걸어 들어옴), `c05-e-elder`, `c05-inn`(유비가 사내 앞에 서고 장비·관우가 문에서 따라 들어옴, 삿갓 쓴 사내를 조조로 바꿔 세움, 조조가 일어나 안쪽 방으로 걸어 나감), `c05-council`(조조와 형제가 입구에서 귀띔을 나눈 뒤 조조는 자리로, 원술이 잔을 놓고, 장비가 일어서려 하고, 전령, 손견이 걸어 나감), `c05-council2`(전령, 제후들의 웅성임, 유비가 한 걸음 나섬). `assets/scenes/ascii/c05-*.txt`: 다시 찍음(`c05-inn`은 탐험 칸 자리라 그림 없음).
- `assets/scenario/hero/story05.json`·`assets/scenes/ch05.json`·`assets/scenes/ascii/c05-start.txt`: 연합 진영 출진 뒤의 `c05-start`(`c05-n1` 나레이션, `c05-s1` 유비)를 지워 출진이 바로 사수관 출전 준비로 — 사용자 "불필요한 장면". 손견군 퇴각과 서량 기병 추격은 B05 개전 컷신이 이미 보인다.
- `docs/fullstory.md`: 5장 S#2의 주막 앞 아이 → 주막 머슴.

## 2026-10-28 팽성 전투 수레 추격

- 결정: 장순과 오환 기병을 북서쪽으로 옮기고, 동쪽 들의 수레가 장순 쪽 북서 모퉁이로 달아나게, 장순은 혼자 다른 시트·색으로, 곁에 `hold` 호위, 공손찬은 남서쪽에서 · 이유: 사용자 "수레 되찾기가 너무 어려움", "유비가 포위를 각오하고 북쪽으로 바로 올라가야", "장순이 일반 기병과 구분이 안 됨", "장순이 공손찬 등장 위치와 가까워졌으니 공손찬은 남서쪽에서" · 영향받는 문서: fullstory.md
- `assets/scenario/hero/stage04.json`: 장순 (9,2) → (3,2), 호위 기병 셋 `hold`(그룹 `guards`, (4,2)(3,3)(2,3), `SE`), 오환 기병 여섯(모두 `assault`) → 다섯: `assault` 둘 (7,5)(6,7)과 수레 길목의 `react` 셋 (9,2)(11,4)(8,4). 약탈 수레 셋(16~18,7~8) → 둘(17,4)(17,6), `Facing` `NW`. 구역 2 "북쪽 끝"(y=0 한 줄) → "북서쪽 퇴로" (0~1,0~1). 공손찬 3턴 등장 (0,1)(0,0)(1,2) `SE` → (0,13)(0,12)(1,14) `NE`(장순 쪽). 개전 대사: 장순 "수레는 이리로 몰고 와라. 기병은 성문을 들이쳐라!", 유비 "장순에게 닿기 전에 막아야 하네!" — 수레가 장순 쪽으로 가니 유비가 곧장 북쪽으로 올라가 기병 사이로 들어가야 막을 수 있다. 수레 둘·`react` 기병으로 한꺼번에 붙는 수를 줄여 되찾기는 쉬워지되 길목의 기병이 긴장을 남긴다.
  측정(10 시드, Lv9): `ai` 10/10 승(이전 8/10), 수레 둘을 모두 되찾음 0/10(게임 AI는 장순을 먼저 노림, 한 대 9/10). 관우·장비가 수레만 쫓는 플레이(임시 도구) 9/10 승·되찾기 10/10, 셋 다 쫓으면 10/10 승·10/10. `rush` 2/10(이전 1/10, B04는 rush 기준 밖 — 유비가 맨 앞에서 기병에 에워싸여 퇴각).
- `assets/data/characters_story.json`: 장순 `Look.Sprite` `heavy_cavalry`(마갑 두른 기존 시트), `Palette` 검은 말(`horseSkin` #202026)·붉은 안장·금빛 쇠(`iron` #e0c060, 투구·마갑 판) — 같은 경기병 시트·같은 이민족 보라색이라 호위·일반 기병과 구별이 안 됐다. 새 그림 없이 화웅(`Sprite: cavalry`)처럼 기존 시트와 팔레트로.
- `tools/mapgen/layouts/hero/map04.txt`: 머리 주석의 배치 설명(장순·수레·공손찬 자리)만 고침. 지형은 그대로.
- `tools/mapgen/layouts/hero/stage04.txt`: `stage_ascii.py dump B04`로 다시 찍음.
- `docs/fullstory.md`: 팽성 전투의 맵·NPC·적·흐름 줄을 새 배치에 맞춤(공손찬 남서쪽, 수레 둘이 북서 모퉁이로, 장순의 외형).

## 2026-10-28 관문 장면 넓힘

- `assets/graphics/maps/S_checkpoint.json`: 두 절벽 사이 골짜기를 작은 칸 13줄에서 28줄로 넓힘(남쪽 절벽을 v=30부터로 내리고 `size` 12×8 → 12×15), 성벽 줄을 새 남쪽 절벽까지 늘이고(v 2~29), 넓어진 풀밭에 나무·덤불 일곱 — 앞 절벽(높이 146px)의 윗면이 골짜기 안 사람을 덮어 `c05-gate`의 조조·관우·장비가 반투명으로 보였다(사용자 지적).
- `assets/scenes/ascii/c05-gate.txt`: 넓어진 맵으로 다시 찍음. 배우 자리는 그대로.

## 2026-10-28 5장 연합 진영 사람들

- `assets/scenario/hero/explore05.json`: 연합 진영에 원소·교모·한복·원술·하후돈·조홍·무안국·공손찬과 원소군·원술 진영·북해·하내·상당 병사, 손견 진영 부상병, 백마 기병을 세우고, 말이 없던 맹주 막사 위병·순찰병·조조군 병사·장터 손님에게 대사를 줌 — 진영이 넓은데 사람이 없다는 사용자 지적. 술판·비방은 줄이고 화웅·사수관·전장 이야기로(사용자 요청).
- `docs/fullstory.md` 5장 S#7 탐험 목록: 같은 대사.
- `assets/scenario/hero/explore06.json`, `docs/fullstory.md` 6장 S#4: 6장 진영에 교모·한복·무안국(5장 큰소리와 짝)·정보·황개·한당·공손찬·조홍과 병사들, 여포·호로관 이야기 — 같은 지적.

## 2026-10-28 단양 전투 아군 진영 이름, 단양 마을·팽성 들판 맵 시안

- `assets/scenario/hero/story04.json`, `assets/scenes/ch04.json`·`ascii/c04-start.txt`: 나레이션 "도겸이 군사를 이끌고 다가온다."(c04-a7)를 지우고, 도겸 일행이 걸어 들어오는 비트를 도겸의 첫 대사 c04-a8로 옮김 — 걸어오는 모습이 보이므로(사용자 결정).
- `assets/scenario/hero/stage03.json`: NPC 진영 이름 "고순·도겸" → "단양병" — 차례 표시에 도겸 이름이 먼저 나와 3턴 등장이 예고되었다(사용자 지적).
- `plan/draft_maps/danyang_village/`(새 폴더): 단양 고순의 마을 세 안(어귀 앞 `dy_gate`, 마을 마당 `dy_square`, 산자락 마을 전체 `dy_whole`)과 `gen.py` — 지금 `S_vil_peace`(6×5)가 좁다는 사용자 지적. 비교 페이지 https://claude.ai/artifact/4sa3t5CVtM2NnsdgFk5C7N
- `plan/draft_maps/pengcheng_field/`(새 폴더): 팽성 싸움 뒤 공손찬과 만나는 들판 두 안(북문 밖 `pc_wall`, 북쪽 언덕 아래 `pc_hills`)과 `gen.py` — 성문 맵 `S_castle_out`이 너무 자주 쓰여 새 맵으로(사용자 요청, 전투 맵의 북동쪽 풀밭).
- `assets/graphics/maps/S_danyang_village.json`(새 파일), `index.json` `maps`: 사용자가 고른 C안(산자락 마을 전체, 16×12) — 절벽 아래 십자 골목·우물·고목 마당, 부서진 집, 남동 목책의 어귀, 개울 낀 동쪽 계곡 길. 마당 앞(남동)의 초가 둘과 문가 평상은 뺐다(마당에 모인 사람을 가렸다).
- `assets/graphics/maps/S_pengcheng_field.json`(새 파일), `index.json` `maps`: 나안을 바탕으로 절벽을 빼고 북동쪽에 구비 도는 강(버드나무·덤불), 가운데는 모양이 제각각인 짓밟힌 흙자리, 되찾은 군량 수레 넷과 부린 자루, 모닥불, 부러진 창·쓰러진 깃발(14×10) — 사용자 "적당한 들판, 가운데가 단조로움, 북동쪽에 강이 구비구비". 원본은 `plan/draft_maps/pengcheng_field/gen.py`의 `pc_field`.
- `assets/scenes/ch04.json` `c04-start`: 맵을 `S_danyang_village`로. 마을 사람 일곱이 마당에서 고순을 둘러싸 웅성이고(Chatter), 유관장은 계곡 길에서 어귀로 걸어 들어와 고순 앞에 선다. 도겸과 서주병 셋은 같은 길로 맵 밖에서 들어와 어귀 안쪽에 서고, 유관장은 그쪽으로 돌아선다.
- `assets/scenes/ch05.json` `c05-reunion`: 맵을 `S_pengcheng_field`로. 도겸·고순·유관장은 수레 곁, 서주병 둘은 수레를 지킨다. 공손찬은 백마기 둘과 남서쪽(맵 밖)에서 말을 타고 들어와(`light_cavalry`, 백마 팔레트) 유비를 알아본 뒤 말에서 내리고(`gongsunzan_noncombat`), 잔을 나눌 때 도겸 곁으로 간다 — 공손찬이 전투에서 남서쪽으로 들어오게 바뀐 것에 맞춤.
- `assets/scenes/ascii/c04-start.txt`·`c05-reunion.txt`: 다시 찍음. `plan/draft_maps/danyang_village/README.md`·`pengcheng_field/README.md`(새 파일): 고른 안.

## 2026-10-28 낙양 거리·술집·대장군부 회상 다시 연출

- 결정: 3장 낙양 저잣거리를 탁현과 다른 새 맵으로, 장비를 술집에 관우와 함께 앉히고 이야기 뒤 둘이 걸어 나간 뒤 밤이 되게, 대장군부 회상에 하진의 부하를 채우고 하진이 입을 열기 전까지 웅성거리게 · 이유: 사용자 "낙양맵이 탁군이랑 너무 똑같음", "장비가 술집에 관우랑 같이 있는게 더 자연스러울듯", "팡 하고 사라져서 어색함", "조조에게 말을 거니 확 밤이 되는게 어색함. 술집에서 이벤트 후 나오면 밤", "하진의 군영에 사람이 너무 없어서 어색함" · 영향받는 문서: presentation-explore.md, presentation-scene.md
- `internal/content/explore.go`: 지역의 `Times`(`ExploreTime{If, Time}`)와 조건 검사 — 플래그에 따라 시간대가 바뀌는 지역
- `internal/core/explore.go`: `AreaTime` / `internal/gui/explore.go`: 지역을 펼칠 때 `AreaTime`
- `tools/scene_maps/luoyang.py`(새 파일) → `assets/graphics/maps/S_luoyang.json`, `index.json`: 궁성 담·궁문·포석 광장과 대로·버드나무와 등롱·조조의 저택 마당(소나무·괴석)·관아와 방문·기와집 골목·장터·술집(초가 주막 그림)·도구점
- `assets/scenario/hero/explore03.json`: 저잣거리를 `S_luoyang`으로, 출구·방문 자리 새로, 낮에는 장사꾼·손님·행인·선비·수레꾼, 밤에는 순라군. 거리·저택·술집이 `c03-rumor` 뒤로 밤. 장비를 거리에서 술집 관우 맞은편 자리로(둘 다 앉음), 관우에게 말을 걸어도 손님 이야기가 열림
- `assets/scenario/hero/story03.json`: `c03-r9` 해설 "관우와 장비가 먼저 자리를 털고 일어나 술집을 나선다."
- `assets/scenes/ch03.json`: `c03-e-rumor` 유비가 탁자 곁으로 걸어가 손님 쪽을 봄, `c03-r8`에 밤이 되며 관우·장비가 일어섬, `c03-r9`에 장비·관우가 차례로 문까지 걸어 나감. `c03-e-flash` 관리 다섯·무관 넷·호위 둘로 좌석을 채우고(조조는 말석), 첫 줄부터 웅성임·혼잣말, 급보 뒤 단양 이야기로 바뀌고 하진이 말할 때 그침
- `internal/gui/uidemo_explore.go`: 비트가 재생되는 동안 대사를 넘기지 않음(캡처에 걸어 나가는 장면이 보이게)

## 2026-10-28 공격 책략 소모·계수 정리

- 결정: 화·수·지 공격 책략의 MP를 기본 24, 강화·확산 48, 강화+확산 64로 낮추고, 화계만 세던 초열(×1.5→1.0)·업화(×2.25→1.5)를 수·지와 맞춤. 강화+확산 책략(폭염·해일·거암·용)의 계수는 계통마다의 개성으로 그대로 둠 · 이유: 사용자 "데미지가 약한 대신, 크리티컬이면 지형변화를 노리는 식", "초열, 업화만 나머지랑 데미지 맞추면" · 영향받는 문서: logic-traits.md
- `assets/data/skills.json`: 15개 공격 책략의 `Cost`, 초열·업화의 `Coeff`
- `docs/logic-traits.md`: 계통별 계수 설명

## 2026-10-28 책략 치명타와 바닥 흔적

- 결정: 공격 책략도 일반 공격처럼 사기비로 치명타를 굴리고, 치명타가 난 대상 칸에 속성별 흔적을 3라운드 남긴다. 화계 불씨(턴 시작 병력 −10%, 배율은 그대로), 수계 물웅덩이(배율 −0.1), 지계 돌무더기(들어가는 비용 3) — 체력 피해·지형 보정·이동력 중 무엇을 깎을지 책사를 고르게. 상태의 배율은 아래층 값에 더한다(120→110, 80→70). 여울 지형과 원소 피해 ±30%는 넣지 않았다 · 이유: 사용자 "책략도 일반 공격 마냥 사기에 의해 치명타", "체력적 피해냐, 지형보정이냐, 이동력 저하냐를 선택", "-10%를 더했으면 하는거야" · 영향받는 문서: logic-combat.md, data-map.md, presentation-ui.md
- `internal/content/terrain.go`: `State.Factor`(서 있는 병종의 배율에 더함), `Move`가 상태의 `Factor`를 더함, `CritMarks`가 없는 상태를 가리키면 데이터 오류
- `internal/content/content.go`: `Rules.CritMarks`(속성 → 상태)·`CritMarkTurns`
- `assets/data/game.json`, `internal/testdata/mini/game.json`: `CritMarks {fire:e, water:w, earth:r}`, `CritMarkTurns 3`, 불씨 `RestHP` −20→−10·`Then s`·`Grounds gdi`, 새 상태 `w` 물웅덩이(`Factor` −0.1, Look `wet`)·`r` 돌무더기(Look `rubble`)
- `assets/data/terrain.json`, `internal/testdata/mini/terrain.json`: 상태 `e`의 배율 0.8을 지우고 `r`의 비용 3을 모든 병종에
- `internal/core/combat.go`: 책략도 치명타 확률을 굴리고 치명타 면역이 막음, `strike`가 치명타 여부도 돌려줌, 책략 치명타면 `markCell`, `critChance`가 책략의 확정 조건을 봄
- `internal/core/battle_events.go`: `markCell`(놓을 수 있는 칸에만, 이벤트 상태는 덮지 않음, `mark` 이벤트)·`markState`
- `internal/core/api.go`, `types.go`: 미리보기가 공격 책략의 치명 확률도 보임
- `internal/core/crit_marks_test.go`(새 파일), `fire_test.go`, `defense_xp_test.go`, `core_test.go`, `internal/content/terrain_test.go`, `internal/gui/editor_test.go`: 새 규칙과 바뀐 불씨 수치
- `../assets_candidate/tools/gen_ground_marks.py`(새 파일) → `../assets_candidate/tiles/{charred,wet,rubble}_0..4.png`·`tiles.json`, `tools/build_extra.py`로 `assets/graphics/map/extra.png`·`index.json`: 그을린 땅(불씨 밑)·젖은 땅·돌무더기 그림 다섯 가지씩. 번짐은 가로 2 : 세로 1 블러 — 사용자와 시안 7번을 보며 고름(웅덩이·자갈은 거절, 마름모 반듯함 거절, 돌 반 크기와 둘레 어두운 흙)
- `internal/gui/field_state.go`: 흔적 칸마다 그림 하나(칸으로 고른 변형), `markLook`
- `internal/gui/player.go`, `events.go`, `internal/cli/play.go`: `mark` 이벤트를 재생·기록·CLI에 반영
- `internal/gui/uidemo_marks.go`(새 파일): `-ui marks`

## 2026-10-28 탐험에서 성문 밖으로 걸어 나가지 않기

- 결정: 탐험의 걸을 수 있는 칸을 지역의 `At`에서 성문을 지나지 않고 닿는 곳과 성문 통로로 한정한다. 성 밖으로는 성문의 출구로만 나간다 · 이유: 사용자 "탁군 장면에서 유비가 바깥으로 갈 수 있잖아" — 열린 성문 통로가 성벽 줄을 뚫어 성 밖 풀밭까지 걸을 수 있었다(낙양 저잣거리도 같음) · 영향받는 문서: presentation-explore.md
- `internal/scenemap/walk.go`: `Walk`가 성문 통로 칸을 기억하고, `Within(u, v)`가 그 점에서 성문을 지나지 않고 닿는 칸과 성문 통로만 남긴다. `plan_test.go`: `TestWalkWithin`(탁현 성문 통로는 열리고 문 밖·북벽 밖 풀밭은 닫힘)
- `internal/gui/explore.go`: 지역을 펼칠 때 `Within(a.At)`
- `assets/graphics/maps/S_zhuo2.json`: 서쪽 성벽 밖 나무 셋(소나무 둘·활엽수)을 u −4·−3에서 −7로 — 그림자가 오른쪽 아래로 드리워 성벽을 뚫고 성 안 바닥에 비쳤다(사용자 지적)
- `assets/graphics/maps/S_castle_town.json`: 낙양 저잣거리도 같은 자리의 나무 셋을 u −7로 — 같은 그림자
- `assets/scenario/hero/story02.json`: 추정 대화 끝(`c02-z9`)에 "그날 성문으로 들인 중산 상인들이 그대를 찾더이다. 저택에 묵고 있으니 떠나기 전에 들러 보시오.", 뒤 노드를 `c02-z10`·`z11`로. `assets/scenes/ch02.json`: 그 줄에서 유비가 추정 쪽(SE)을 다시 본다. `docs/fullstory.md` S#1 — 사용자 "대화 마지막에 구해준 상인들이 너 부른다더라. 저택으로 가봐라": 관청에서 다음 갈 곳을 바로 알려 준다

## 2026-10-28 무도가 무기 철권갑

- 결정: 무도가 계열(무도가·권사·권성)의 무기 종류 `fist`에 맞는 장비가 하나도 없어 무기 칸이 늘 비었다. 기본 무기들과 같은 급의 `iron_knuckles`(철권갑, 무력 +1, 300냥)를 더하고 대장간에서 팔며 번궁이 처음부터 찬다 · 이유: 사용자 "무도가대용 무기를 만들자"
- `assets/data/equipment.json`: `iron_knuckles`(Type `fist`, Icon `knuckles`)
- `assets/data/shops.json`: 대장간(`weapons`)에 철권갑
- `assets/data/characters_story.json`: 번궁 `Equipment`에 철권갑
- `../tools/spritetool/assets/ver5-ui/settings.json`·`sources/item_knuckles.png`·`pixels.json`, `assets/graphics/ui/ui.png`·`ui.json`: `item_knuckles` — codex 그림(가죽을 감은 주먹에 쇠 덧댐) → `pixelize_icon.py` → `build_effects.py`
- `internal/gui/uidemo_waypoints.go`: 출전 준비 뒤 첫 장수의 장비 창(210)과 무기 고르기(240)까지 보임

## 2026-10-28 장면 이동(검증용 지점 시작)

- 결정: 시작 메뉴에 "장면 이동"을 두어 각 장의 시작과 각 전투를 스크롤 목록으로 고르면 바로 그곳에서 새 게임을 연다. 레벨은 1 + 2×(다음 전투 순번), 아이템·장비는 모두 가진 채, 관우 청룡언월도·장비 장팔사모를 찬다(적토마·방천화극은 아직 여포의 것이라 창고에만) · 이유: 사용자 "스토리, 전투를 검증하기가 어려운데, 편하게 왔다갔다", "모든 아이템은 얻은걸로", "관우는 청룡언월도, 장비는 사모" · 영향받는 문서: logic-campaign.md, data-scenario-node.md
- `internal/content/scenario.go`: `Waypoint`, `Data.Waypoints`(장마다 첫 노드와 전투 노드, 장소는 첫 장면의 `Place` 또는 첫 탐험의 시작 지역 이름) — 목록과 엔진이 같은 지점을 쓰게
- `internal/content/content.go`·`validate.go`: `Campaign.Loadout`과 그 검사 — 엔진에 인물·장비 이름을 두지 않으려고
- `internal/core/waypoint.go`(새 파일), `engine.go`: `NewAt`(앞 이야기를 기본 길로 지나감, 레벨·학습 창, 아이템, `Loadout`), `New`를 `begin`+`enter`로 나누고 `join`·`grant` 처리를 `once`로 뽑아 함께 씀
- `internal/core/waypoint_test.go`(새 파일): 모든 지점이 열리고 저장 복원되는지, 앞 전투 승리·합류, `Loadout`, 없는 지점 거부
- `internal/session/session.go`: `jump` 요청(`Request.Node`) — 새 게임처럼 버릴지 묻고 엔진을 바꿈
- `internal/gui/waypoints.go`(새 파일), `game.go`·`windows.go`: 시작 메뉴 "장면 이동"과 목록 창(휠·끌기로 스크롤, 끌지 않고 놓은 줄로 이동, 줄마다 레벨)
- `internal/gui/scenario.go`: 출전 준비의 "군량 · 도구"가 창 아래를 넘으면 들어가는 줄까지만 보이고 "…" — 아이템을 모두 가지면 넘쳤음
- `internal/gui/uidemo_waypoints.go`(새 파일): `-ui waypoints` — 목록, 끝까지 스크롤, 마지막 전투로 이동해 학습 창과 출전 준비

## 2026-10-28 수전: 낮은 물과 배

- 결정: 전투맵의 물은 한 종류로, 수면을 땅보다 9px 낮게 그리고 좌상·우상 강둑에 흙 절벽을 세운다. 스테이지의 `Boats`가 물을 진입 불가로 둘지 배로 건널지 정하고, 배 전투의 물 칸은 보병류 우대(보병·적병 1·1.1, 궁병·서생 1·1.0, 기병 2·0.9, 병기 불가). 기병은 말 탄 채 긴 배에 탄다. 배는 들린 배 · 이유: 사용자가 시안(https://claude.ai/artifact/NwKVGU8HAq8mZP5UFovE36)을 보고 "느낌 되게 좋다", 얕은물·깊은물 구분 대신 "물은 이것으로 고정", "전투에 설정이 있어서 ... 둘 중 하나만", 배 밑 점선 물결 대신 기존 그림자 생성기로 · 영향받는 문서: plan-naval-water.md(새 문서), data-scenario-node.md, data-map.md, logic-combat.md
- `internal/content/terrain.go`·`content.go`: `Stage.Boats`, `Data.Boats`(병종군 → 물 땅 → 규칙), `Move(group, cell, boats)`가 배 전투에서 `Boats` 규칙을 땅 바로 위 층으로 덧씌움(다리·상태는 그 위라 부서진 다리는 그대로 막힘), `validateTerrain`이 `Boats`의 병종군·물 땅을 검사 — 스테이지 하나의 설정으로 물의 쓰임을 바꾸려고
- `assets/data/terrain.json`: `Boats` 표
- `internal/core/rules.go`·`battle_events.go`·`api.go`, `internal/content/map_document.go`, `internal/gui/editor.go`·`hud.go`: `Move` 호출처에 배 여부를 넘김(`Engine.Boats()`), 저장 검사가 배 전투에서는 물 위 유닛을 허용, `waterNear`가 자기 칸도 봄(배 위에서 물 병법) — 배 전투에서 물 위에 설 수 있게
- `internal/gui/field_water.go`(새 파일): `makeBanks`(물 작은 칸의 NW·NE 변에 땅 이웃이면 강둑 면, 그 쪽 아래 변의 이웃도 땅이면 잘린 면, SE·SW 변에 1px 선), `boat`·`pose`·`drawAfloat`(물 위 유닛을 배 → 병사(걷기 대신 대기 프레임) → 앞 뱃전으로 조립해 출렁이고, 앞쪽 땅 칸의 발자국으로 지운 뒤 캔버스에) — 물을 실제로 내리지 않고 낮아 보이게: 물 텍스처가 이음새 없이 반복되므로 강둑 면만 덮으면 됨
- `internal/gui/field.go`·`field_native.go`·`field_render.go`·`assets.go`·`player.go`·`view.go`: `MapArt.SunkWater`(전투맵만), 물 칸 `Lift` −9(유닛·바·효과가 따라 내려옴), 하이라이트는 땅 높이, 낮은 물에서는 물가 블렌드를 그리지 않음, 강둑을 물 장식 다음에 그림, 배 그림자(배 소품 그림자 + 대기 프레임 그림자), `unitVis.mounted`·`boatAxis`
- `internal/gui/uidemo_naval.go`(새 파일): `-ui naval` — 가장 큰 맵에 강을 칠하고 `Boats`를 켜, 물 위 아군과 강을 건너는 AI를 보임
- `../tools/unit3d/boats3d.py`(새 파일): 들린 배 3D 모델(뱃전이 양 끝으로 휘어 오르고 이물·고물 기둥), 보병용 `boat_{u,v}`·기병용 `boat_l_{u,v}`와 앞 뱃전 `_rim`(그림자 없음), `--review`
- `../assets_candidate/tools/gen_banks.py`(새 파일): 강둑 면 `bank_{nw,ne}_{grass,soil}[_cut]`(9px 흙 절벽, 풀·흙 테두리, 발치 물거품)과 `bank_line_{se,sw}`, `--review`
- `../assets_candidate/tiles/boat*.png`·`bank_*.png`, `tiles.json`, `assets/graphics/map/extra.png`·`index.json`(extra): 위 그림, `build_extra.py`로 다시 묶음
- `internal/content/terrain_test.go`: 배 전투 규칙 시험, 다른 `Move` 시험은 배 없음으로

## 2026-10-28 진영 맵

- 결정: 탐험 진영 셋을 넓게 다시 짓는다. 진영마다 두 안을 실제 형식 2 맵으로 그려 비교 페이지로 보이고, 사용자가 의병 A·노식 B·연합 A를 골랐다 · 지금 진영은 60×60 풀밭 가운데 게임 칸 5×5에 천막 일곱 개뿐이라 초라했고(사용자 지적), 연합군 진영은 제후마다 막사가 있고 들어갈 수 있어야 했다.
- `assets/graphics/maps/S_militia_camp.json`(다시 씀): 탁현 성벽·성문 아래 길가 주둔지 — 모병소, 장비의 훈련장(허수아비 갑옷걸이), 의병 천막과 거적 자리, 삼형제의 막사, 취사, 짓다 만 목책·망루(38×34 작은 칸). 성벽에 붙은 나무는 잘려 보여 뺐다(사용자 지적).
- `assets/graphics/maps/S_luzhi_camp.json`(새 파일): 개울 건너 광종성(닫힌 성문)을 따라 녹각, 전영(방패벽·망루)·중군(노식의 큰 막사)·후영(군량 수레 셋, 사신의 막사, 취사)이 한 줄, 길 하나가 셋을 꿰어 동쪽 끝 탁군으로(64×34) — 출구에서 들어오면 사신과 군량부터 지나치게.
- `assets/graphics/maps/S_coalition_camp.json`(새 파일): 바깥 목책 안 십자 대로, 가운데 맹주의 큰 막사, 제후 진영 여덟(원소군·원술(큰 막사, 군량 수레 넷)·조조(큰 막사, 조련장)·공융·손견(약솥·피 묻은 천)·하내 왕광·상당 장양·공손찬(삼형제 막사)), 종군 상인 거리, 망루(80×72). 진영 사이 길은 세 칸, 문은 모두 큰길 쪽.
- `assets/graphics/maps/S_tent2.json`·`S_tent_lux2.json`·`S_council2.json`(새 파일, `scene_v2.py --copy 2`): 군막·원술의 막사·맹주 회의 막사의 형식 2 사본 — 탐험은 형식 2 맵만 걷기 때문에. 옛 맵은 장면이 그대로 쓴다.
- `assets/graphics/maps/S_army_camp2.json`(지움), `index.json` `maps`: 쓰는 곳이 없어짐(노식·연합 진영이 새 맵으로).
- `assets/scenario/hero/explore01.json`·`explore02.json`·`explore05~07.json`: 진영 지역을 새 맵으로, 사람·출구 자리를 다시 잡고, 막사 출구(`Object` 막사)와 실내 지역(`camp_tent`, `luzhi_tent`, `meng_tent`·`yuanshu_tent`·`cao_tent`·`our_tent`)을 더함. 엑스트라(훈련하는 의병, 노식 막사 위병·순찰병, 맹주 막사 위병, 대로 순찰병, 조조군 조련, 장터 손님)를 더함. 사람은 대본의 진영 자리에(조조는 영채 문, 공융은 북해 진영 문, 원소·원술은 막사 앞). 막사 안 인물·대사는 아직 없음.
- `assets/graphics/maps/S_coalition_camp.json`: 제후 진영 문 옆 세 칸 안의 천막 다섯을 뺌 — 게임 캡처에서 영채 문 앞의 조조가 천막에 겹쳐 보였다.
- `assets/scenes/ch01.json` `c01-sortie`: 정찰병이 대흥산 길(남쪽)에서 달려오고, 형제가 그쪽을 본다. `ch02.json` `c02-e-luzhi`: 맵을 `S_luzhi_camp`로, 사신과 노식이 마주 보고 사신은 동쪽 문으로 나가며 유비는 노식 옆으로. `ch03.json` `c03-start`: 맵을 `S_luzhi_camp`로, 함거와 호송병은 후영 길 위, 형제는 동쪽 문에서 들어온 자리. `assets/scenes/ascii/`: 다시 찍음.
- `plan/draft_maps/camps/`(새 폴더): 고르지 않은 세 안(`militia_b`·`luzhi_a`·`coalition_b`)과 여섯 안을 다시 쓰는 `gen_camps.py`, `README.md` — 사용자가 고르지 않은 안도 지우지 말라고 함.
- `docs/presentation-explore.md`: 1부 탐험 표의 진영 맵과 막사 출구 설명.
- `../tools/unit3d/scene_props3d.py`: 새 소품 `grain_cart`(군량 자루를 싣고 끈으로 묶은 두 바퀴 수레, u 방향)·`grain_cart_v`(같은 모델을 v 방향으로 돌림)·`command_tent`(큰 막사: 기둥 위 천 벽, 용마루 있는 우진각 지붕, 붉은 물결 처마띠, 앞(SW)면에 열어 묶은 문과 두 기둥 차양, 용마루 양끝 깃발) — 사용자 요청으로 노식 진영 사신이 세는 수레·원술의 떠나지 않는 수레와, 맹주·노식·조조처럼 들어가는 막사를 다른 천막과 한눈에 가르기 위해. 3D 모델 → 셀 렌더(기존 소품 경로).
- `../assets_candidate/tiles/grain_cart.png`·`grain_cart_v.png`·`command_tent.png`, `tiles.json`: 위 렌더(50×45, 50×45, 113×102).
- `assets/graphics/map/extra.png`·`index.json`(extra): `build_extra.py`로 다시 묶음.
- `assets/graphics/scene.json` `footprints`: `grain_cart` [-1, -0.55, 2.2, 1.1], `grain_cart_v` [-0.55, -1, 1.1, 2.2], `command_tent` [-1.75, -1.45, 3.5, 2.9](천 벽까지, 문 앞 차양 아래는 걸을 수 있게) — 걷기와 그리는 순서.

## 2026-10-28 1장 주점·도원 대사, 영천 새벽 골목 장면, 노모 도트

결정 · 관우가 주점에 며칠째 있던 설정을 "무섭게 마셔 둘레에 사람이 없다"로 바꾸고, 결의는 유비가 "가족"을 꺼내고 장비가 "진짜 형제"를 제안하는 흐름으로. 영천 불을 끈 뒤 새벽 골목에서 서복의 인사와 화계서 전달을 보이고, 광종으로 돌아가는 동기를 둔다 · 이유: 사용자 "관우가 며칠째 있는건 ... 의병 되기 무서워서 저러는거 같잖아", "불끄고 돌아오니 진영에 노숙(노식)이 있다니", "불이 꺼진 마을을 보여주고, 서복이 감사합니다" · 영향받는 문서: fullstory.md
- `assets/scenario/hero/story01.json`·`explore01.json`·`assets/scenes/ch01.json`: S#1~S#3 대사, 주점·도원 노드 번호를 새로 매기고 연출을 옮김(관우가 일어서고 다시 앉는 자세 추가). 결의 전 관우는 "현덕" + 부드러운 존댓말, 유비는 형제에게 하게체
- `assets/scenario/hero/story02.json`·`assets/scenes/ch02.json`: 새 장면 `c02-dawn`(S_castle_town_burnt, 새벽, 서복·노모·병사). 화계서 지급과 "새벽녘에야 불이 꺼졌다"를 빈 진영 장면에서 이 장면으로 옮김. 조조·하후돈 퇴장은 `c02-e9`로
- 불은 성 안이 아니라 장사성 남동쪽, 유비 진영 가까운 마을에 났다(성 안에 불이 났는데 황보숭이 손견·조조를 치하하는 것은 이상하다는 지적). `story02.json`·`characters_story.json`(서복 소개)·fullstory의 "성 안 민가·골목"을 마을로, `c02-dawn` 지도는 아래 `S_village_burnt`. 서복·노모·병사는 화면 밖에서 걸어 들어오고 걸어 나감
- `story03.json`·`ch03.json`: 3장 S#1을 광종 진영 앞이 아니라 광종으로 돌아가는 길(`S_road`)에서 쉬고 있는 함거와 마주치는 장면으로(연의 2회). 형제가 화면 밖에서 걸어 들어오고, 장비 "저 안에, 스승님 아녀유?" 한 줄 추가, 노드 번호 다시 매김.
- `assets/graphics/maps/S_village_burnt.json`(새 형식 2 맵)·`index.json`: 옛 `S_vil_burnt`(6×5)는 좁고 다닥다닥해서 10×8 불탄 마을을 새로 — 불탄 집 셋, 성한 초가 둘, 흙길·갈림길, 재·물통·우물·울타리·나무. `c02-dawn`에 정리하는 백성 셋(웅성임)을 둠. 유비의 마지막 줄은 "황보 장군께 인사 올리고, 광종 스승님께 돌아가세"(황보숭을 만나는 장면은 생략)
- `internal/gui/scene.go`·`internal/scenes/scenes.go`·`scenario_test.go`: 장면 비트 `Prop: [이름, 배우]` — 놓인 소품이 그 배우를 따라 움직이고 배우가 퇴장하면 함께 사라진다. 3장 함거가 그 자리에서 사라지고 호송병이 형제를 뚫고 지나가던 것을, 형제가 길가로 비키고 함거가 노식과 함께 끌려 나가며 호송병이 뒤따르게 고침
- `tools/doteditor/tools/mounted_kits/commoners.py`: `commoner_oldwoman`(노파) — 긴 치마, 쪽진 흰머리(테두리 없는 덩어리, 뒷모습에서 눈처럼 보이지 않게), 허리 위만 앞으로 꺾는 `hunch`(lean은 무릎부터 몸 전체를 기울여 넘어지는 것처럼 보임), 땅까지 곧은 지팡이, 뒷짐. 노인과는 치마·수염 없음·굽은 허리로 구별
- `tools/doteditor/units/commoner_oldwoman.yaml`, `assets/sprites/commoner_oldwoman.spr`, `internal/sprite/testdata/units/commoner_oldwoman.png`: 새 시트

## 2026-10-28 영천: 조조·손견도 울타리로 걸어가 불을 놓고, 원군은 장수만, 성 안 수비병은 넷

결정 · 개전 재생에서 이벤트로 도착하는 군대의 컷신 걸음·모션이 빠지던 것을 고치고, 조조군·손견군은 장수만, 황보숭군 수비병은 절반으로 · 이유: 사용자 "조조,손견은 가만히 있는데 갑자기 불이 나는 형세라 어색해", "조조,손견군이 너무 많은데, 주요 장수들만 나오고", "황보순군은... 절반 정도로 줄여죠". `_balance` B02 ai 10판은 바꾸기 전과 같음(10/10, 장비 퇴각 5) · 영향받는 문서: fullstory.md
- `internal/gui/cutscene.go`: `walkTimeline`·`cutsceneTimeline`이 유닛을 타임라인을 만들 때가 아니라 각 박자가 올 때 찾음(`onUnit`) — `replayOpening`이 도착 전 군대를 판에서 빼 두므로, 만들 때 찾으면 조조·손견의 이동·공격이 통째로 빠지고 불만 났음
- `assets/scenario/hero/stage02.json`: 조조군 spawn에서 보병·궁병 여섯, 손견군 spawn에서 보병·궁병 여섯을 뺌(조조·하후돈·하후연, 손견·정보·황개·조무만). 성 안 수비병 여덟 중 (8,13)·(8,16)·(9,14)·(9,15) 넷을 뺌 — 황보숭·주준 곁에 둘씩
- `tools/mapgen/layouts/hero/stage02.txt`: 다시 dump
- `docs/fullstory.md`: 영천 NPC 줄에 수비병 넷과 원군 장수 이름

## 2026-10-27 위압을 "줄지 않는 타격"으로, 옛 위압은 급소로

결정 · 청룡언월도의 위압을 "연격·반격·주변 타격이 100%"인 규칙 변경형 특성으로 바꾸고, 빗나감 50% 효과는 이름을 급소로 바꿔 화웅에게 준다 · 이유: 사용자 "청룡언월도는 반격,이연격,사방격,팔방격,청공 등 데미지 100% 미만으로 들어가는걸 전부 100%로", "위압으로 하고, 기존 위압의 이름을 바꾸면? 무게와 위압감으로 어떤 상황에서도 적에게 충분한 피해를 준다", "기존의 위압은 ... 적의 약점을 찌른다는 식", "급소로 하고, 화웅은 급소를 주자". 응사·협동·급소 빗나감은 범위에서 뺌 · 영향받는 문서: logic-combat.md, logic-traits.md
- `assets/data/traits.json`: `intimidate`(위압)의 효과를 `full_share`로, 새 `vital_strike`(급소, `miss_half` 0.5)
- `assets/data/characters.json`: 화웅 `intimidate` → `vital_strike` — 화웅의 효과는 그대로
- `internal/content/vocab.go`: `FxFullShare`. `internal/core/combat.go`: `full_share`면 `attackOn`이 두 번째 타격에 `DoubleShare`를 곱하지 않고, `counterShare`가 1을 주고, `victims`의 주변·뒤 칸(사방격·팔방격·청공·찔러공격)이 상한 없이 1. 찔러공격의 뒤 칸도 같은 `side` 규칙을 쓰므로 함께 1이 됨
- `internal/core/combat_followup_test.go`: `TestFullShareStrikesInFull`, `internal/testdata/mini/traits.json`에 `full_share_t`
- `docs/logic-traits.md`: 고정도의 특성을 위압으로 적어 둔 것은 틀렸음(실제 `keen_edge` 예리). 바로잡음

## 2026-10-27 전원 반격을 설정에서 빼고 규칙으로 고정

결정 · 모든 유닛이 사거리 안의 물리 공격자에게 반격하는 것을 끌 수 없는 기본 규칙으로 한다 · 이유: 사용자 "지금 전원반격이 기본이거든. 전원 반격인걸로 그냥 코드나 그런거 고정해줄레?" · 영향받는 문서: logic-combat.md, data-item.md, presentation-ui.md, system-architecture.md, stage_lv_balance.md
- `assets/data/game.json`, `internal/testdata/mini/game.json`, `internal/content/content.go`: `Rules.PlainCounterShare`(0.5) — 태세·특성 없는 반격 배율을 코드 상수 `counterAllShare`에서 규칙 데이터로 옮김
- `internal/core/combat.go`, `api.go`: `counterShare`가 반격 배율과 투신 여부를 정하고 `counterBack`·`CounterDamage`가 같이 씀. `State.CounterAll`·`SetCounterAll`과 관찰 캐시의 비교를 지움
- `internal/storage/storage.go`: `Settings.CounterAll`(`counter_all`) 삭제. 예전 설정 파일의 키는 읽을 때 무시됨
- `internal/gui/windows.go`: 설정 창의 `전원 반격` 줄 삭제, 창 높이 732 → 668. `internal/cli/play.go`의 설정 줄, `internal/session`의 엔진 동기화, `cmd/_balance`의 `-counter`도 삭제
- `internal/core/core_test.go`: `TestEveryUnitCountersAndStancesCounterHarder` — 태세 없는 반격, 반격 예측이 보통 < 태세 < 투신
- 문서의 "전원 반격은 연타 없음"은 코드와 달랐다(전원 반격도 `strikeBack`으로 연타를 굴렸음). 코드대로 연타를 굴린다고 고침

## 2026-10-27 맵 바깥 테두리와 미니맵을 벡터 경로 대신 이미지로

- `internal/gui/field_outside.go`: 바깥 풀밭·장식·안개·비네트를 화면 크기 이미지 하나에 모아 전투 영역을 삼각형 두 개(`cutBattleArea`)로 한 번 오려 내고 한 번 합성함. 안개와 비네트는 맵 중심에서의 거리만으로 정해지므로 둘을 겹친 색을 텍스처 하나(`hazeTexture`)에 미리 구움. 경계의 어두운 그림자와 깜빡이는 테두리 선은 1픽셀 폭 세로 띠 이미지(`dropBand`, `rimBand`)를 네 변의 사각형에 늘려 찍고, 이웃한 변은 모서리 이등분선에서 만나 겹치지 않게 함(`drawBorder`). 띠 두께는 찍을 때 확대 배율에 맞추므로 ×4에서도 선명함 — 매 프레임 안티에일리어싱 경로 7개(오려 내기 3·선 4)와 화면 전체 합성 3번이 들던 것을 경로 0·합성 1로. 월드 크기로 미리 굽는 안은 확대하면 선이 계단지고 배율마다 구우면 메모리가 커서 쓰지 않음. 비교 페이지: https://claude.ai/artifact/26wMvxb3wVzYZyzaoECgEC
- `internal/gui/hud.go`, `game.go`: 미니맵 지형을 맵·크기·기기 픽셀 위치가 같은 동안 이미지 한 장(`miniGround`)에 두고 매 프레임 그 한 장만 찍음.
- `internal/gui/skin.go`: `fillCircle`이 원 경로 대신 미리 만든 원 이미지를 찍음. 마름모 오려 내기용 `whitePixel`을 둠 — B02(유닛 51)에서 벡터 경로 처리가 CPU의 12~15%였고, 거의 전부 미니맵 유닛 점(유닛마다 안티에일리어싱 원)이었음. ebiten은 경로를 모아 두었다 다음 그리기에서 처리하므로 프로파일에는 그 자리(미니맵 카메라 틀)가 비싸 보였음.
- `internal/gui/editor.go`: 해제할 이미지 목록을 새 텍스처로 바꿈.
- 측정(Mac, B02 불 붙은 상태, 실제 게임 루프): Draw CPU 중앙값 5.2ms→4.0ms, 벡터 경로 처리 12~15%→0%, 미니맵 11~13%→1.2%, 바깥 테두리 5.3%→1%. 줌 1·4 캡처 비교에서 바뀐 픽셀은 전체의 0.15% 미만(최대 차이 52/255, 테두리 선·미니맵 점 가장자리·안개 반올림).

## 2026-10-27 Observe가 상태를 복사하지 않고 공유함

- `internal/core/engine.go`: `Observe`가 상태·장수·유닛·대화 노드·스테이지를 JSON으로 왕복 복사하던 것을 그만두고 공유함(칸 상태를 쓰는 맵 줄만 새로 만듦). 상태를 모으는 부분을 `current`로 떼어 `Snapshot`은 그것을 한 번에 깊은 복사함 — 명령은 복사본에서 실행해 통째로 갈아 끼우므로(Apply) 확정된 상태는 그 자리에서 바뀌지 않아, 화면이 읽기만 하면 복사가 필요 없음. B02(유닛 51) `Observe` 한 번 1~2ms·419KB→0.20ms·122KB.
- `internal/core/observation_cache_test.go`: 받은 결과가 이후 명령·설정 변경 뒤에도 그대로인지 확인.

## 2026-10-27 화면이 읽는 엔진 상태를 바뀔 때만 만듦

- `internal/core/engine.go`: `ObservationCache` 추가 — 엔진이 바뀌거나 상태가 바뀔 때(명령의 Revision, Revision을 올리지 않는 설정 세 가지)만 `Observe`를 다시 부르고, 그 사이에는 같은 결과를 돌려줌. 결과는 함께 쓰므로 읽기만 함. 장사의 밤(B02) 실제 게임 루프 측정: 프레임마다 `Observe`를 8번(팝업 확인 2·Update·전투 기록 2·키 입력·advance·그리기) 불러 상태 전체를 JSON으로 왕복 복사했고(한 번 1~2ms, 419KB) CPU의 44%를 차지 — 사용자: 안드로이드에서 전투가 끝난 뒤 스크롤이 버벅임. 바꾼 뒤 Update 8.1ms→0.06ms, 할당 4.9→1.4MB/프레임, 프레임 간격 p95 32~40ms→17~19ms.
- `internal/session/session.go`: `Session.View()`가 캐시된 결과를 줌. session 안의 `Engine.Observe()`도 이것으로 바꿈. 설정 변경에 Revision을 올리는 방법은 JSON 줄 프로토콜의 Revision 약속(명령마다 하나)을 깨서(`StaleRevision`) 쓰지 않음.
- `internal/gui/*.go`: `g.S.Engine.Observe()` 63곳을 `g.S.View()`로, 임시 이동 뒤 sandbox를 읽는 곳은 `g.view()`(sandbox 전용 캐시)로 바꿈. 한 번만 쓰는 sandbox·데모 준비 코드는 그대로 둠.
- `internal/core/observation_cache_test.go`: 상태가 그대로일 때 재사용, 명령·설정·엔진 교체 뒤 다시 만듦을 확인.

## 2026-10-26 유비 호칭: '유씨' 대신 돗자리 장수, 관우가 종친을 말함

- `assets/scenario/hero/explore01.json`, `explore02.json`, `story01.json`, `story02.json`: 백성·술꾼·소쌍이 유비를 "유씨 댁 아들"·"짚신 장수" 대신 "돗자리 장수"로 부르고, 유비 대사에 돗자리를 넣음. 도원 술자리의 관우 주공 대사를 "황실 종친이시며, 우리 셋 중 가장 사람을 잘 이끌 수 있는 이"로 바꿈 — 사용자: 황제의 성인 유씨를 백성이 들먹이는 건 무도해 보임. 유비는 짚신과 함께 돗자리도 팔았음.
- `assets/data/characters.json`, `docs/data-character.md`: 유비 소개에 돗자리를 더함.
- `docs/fullstory.md`: 유비 설정을 "종친임은 형제만 안다(도원에서 관우가 한 번 말함), 세상이 아는 건 17장"으로 고치고, 백성은 이름·자를 모른 채 "돗자리 장수"라 부른다고 적음. 같은 대사들을 맞춤.

## 2026-10-25 탐험 이름표 색: 색조는 뜻, 밝기는 가리킴

- `internal/gui/walkmap.go`: 이름표 색을 보통 회색·본편 가이드 초록·곁이야기 가이드 노랑으로 바꾸고, 가리키면 같은 색조에서 바탕이 밝아지고(밝기 약 64→128) 글자가 흰색이 되게 함. `plateHover`를 없애고 `namePlate`가 가리킴을 따로 받음 — 사용자: 가이드 노랑·보통 갈색·가리킴 검붉은색이 섞여 고르고 있는지 헷갈림. 가리키면 가이드 색이 사라지던 문제도 없앰. 초록은 유비의 색.
- `internal/gui/explore.go`, `tour.go`: 가리킴을 색 대신 따로 넘기고, 가리킨 이름표를 맨 위에 그림.
- `internal/gui/uidemo_explore.go`: 나레이션이 사라진 뒤 다음 노드로 넘기는 처리(게임 Update의 것)를 데모에도 넣음 — 나레이션 페이드가 들어간 뒤 탐험 데모가 1장 오프닝에서 멈췄음.

## 2026-10-25 탁현 관아 앞 태호석 치움

- `assets/graphics/maps/S_zhuo2.json`, `S_zhuo.json`, `tools/scene_maps/zhuo.py`: 관아 앞 태호석을 뺌 — 사용자: 관아로 가는 길을 막음.

## 2026-10-25 장면 소품 그림자를 실루엣 투영으로

- `internal/sprite/render.go`: 유닛 그림자의 투영·키움·흐림을 `castShadow`로 떼어 내고 그림 한 장용 `ImageShadow` 추가(시트 그림자는 결과 그대로, 파이썬 골든 일치) — 소품도 유닛과 같은 규칙으로 그림자를 만들려고.
- `internal/sprite/art.go`: 그림자 색 입히기를 `Tint`로 떼어 냄.
- `internal/gui/assets.go`: 서 있는 소품마다 `Prop.Shadow`, 흔들리는 나무는 프레임마다 `SwayShadow`를 로드할 때 만듦.
- `internal/gui/field_scene.go`, `field.go`, `scene.go`: 장면 소품·비트 소품의 폭 맞춤 타원(`propShadow`)을 지우고 실루엣 그림자를 깖. 흔들리는 나무는 잎과 같은 프레임(`swayFrame`)의 그림자를 고름 — 사용자 요청: 가옥·나무·우물·바위에 모양대로 그림자, 나무가 흔들리면 그림자도 흔들리게. 탁군·도원 캡처로 확인.

## 2026-10-25 장면 나무 잎 흔들림(F3)을 탁군·복숭아 숲·의병 주둔지에

- `tools/build_tree_sway.py`(새): 잎 있는 장면 나무 13종에 F3(밝은 잎·꽃 조각 1px 들썩 + 테두리 3px 살랑)를 8프레임·160ms로 만들어 `../assets_candidate/tiles/sway/`에 쓰고 `assets/graphics/map/sway.png`·`index.json` `"sway"`로 묶음. 줄기 옆 두 픽셀 이하의 초록 음영은 줄기로, 테두리 규칙 밖의 어두운 선(매화 잔가지)은 고정, 움직이는 조각은 잎 면적의 25%까지(밝은 흰 꽃이 많은 매화가 전체로 들끓지 않게), 버드나무는 가지 끝이 더 흔들림 — 사용자가 1차·2차 시안 중 F3를 골라 세 장면에 시험 적용.
- `internal/gui/assets.go`: `"sway"` 표를 `Assets.Sway`(나무별 `FrameStrip`)·`SwayMaps`로 읽음.
- `internal/gui/field_scene.go`: `drawSway` — `SwayMaps`에 든 형식 2 맵의 나무를 위치 해시로 정한 시작 프레임·시간에서 재생(숲이 한꺼번에 움직이지 않게).
- 적용 맵: `S_zhuo2`(탁군 성 안), `S_peach_grove2`(복숭아 숲), `S_militia_camp`(의병 주둔지) — 1장의 세 장면. 맵 파일 형식은 바꾸지 않고 표에 적어, 맵을 다시 쓰는 도구가 지우지 않게 함.
- `docs/presentation-scene.md`: 잎 흔들림 항목 추가.

## 2026-10-25 활엽수 잎 살랑임 2차 시안: C·D·E 조합과 3D 정보 이용

- `tools/tree_sway_variants.py`: 조합 F1~F5(가장자리 살랑을 바탕으로 3px 잎 반짝임·수관보다 넓은 바람 물결·5px 이하 밝은 잎 조각의 1px 들썩임, F5는 12프레임 고요→돌풍→고요)와 3D 이용 G1~G4(활엽수 3D 모델의 잎덩이 45개를 도트 크기에 맞춰 렌더해 앞 덩이의 움직임대로 도트를 옮김, 3D 면 방향 기준으로 잎 조각을 기울여 명암 한 단계, 둘의 합, 가장자리·돌풍 주기까지) 추가 — 사용자: C는 울렁거려 잎 느낌이 안 살고, E는 국소적이라 어색, D는 가운데가 그대로라 이상. 큰 덩이 대신 작은 잎 조각이 수관 전체에서 움직이게 함.
- `docs/tree-sway-variants.html`: 2차 조합·3D·1차 묶음으로 나누고 안마다 프레임 수(8·12)를 따름.

## 2026-10-25 활엽수 잎 살랑임 다섯 안

- `tools/tree_sway_variants.py`(새): `stree_broad_0` 원본 도트에서 수관째 휘청·물결·잎덩이별·가장자리만·바람 띠 다섯 방식의 8프레임 시트를 만듦. 원본 8색만 쓰고 줄기는 고정, 외곽선은 원본과 같은 상하좌우 규칙으로 다시 그림, 캔버스 좌우 3px 여유 — 3D·그림 재생성 없이 기존 도트를 살린 움직임을 비교하기 위해.
- `../tools/spritetool/assets/tree-sway-variants/stree_broad_0/`(새): `original.png`, 다섯 안 시트, `variants.json` — 시안 결과 보관.
- `docs/tree-sway-variants.html`(새): 원본·다섯 안의 확대 반복·여러 그루(시작 프레임 엇갈림)·프레임 띠, 간격·배율 조절 — 사용자가 고르게 하기 위해. 게임 에셋은 바꾸지 않음.

## 2026-10-25 나무 움직임 두 가지 비교 시안

- `docs/tree-motion-experiments-plan.md`: 표면 법선 햇살과 ⅓ 진폭 잎 움직임의 비교 범위 기록 — 두 방식을 분리해 검토하기 위해.
- `tools/tree_motion_experiments.py`: 활엽수 법선 맵·햇살 4프레임과 버드나무 0.04 진폭 3D·도트 렌더 추가 — 위치와 기둥을 고정한 약한 변화량을 비교하기 위해.
- `tools/spritetool/assets/tree-motion-experiments/README.md`, `stree_broad_0/normal-map.png`, `stree_broad_0/sunlight-sheet.png`, `stree_willow/reference-3d.png`, `stree_willow/dot-sheet.png`: 실험 설정과 결과 원화 추가 — 재생성과 시각 검토를 위해.
- `docs/tree-motion-experiments.html`: 원본·법선·햇살 변화와 버드나무의 기존·약한 흔들림을 250ms로 비교하는 페이지 추가 — 게임 적용 전에 두 시안을 검토하기 위해.

## 2026-10-25 장면 나무 전체 4프레임 산들바람 시안

- `docs/tree-sway-all-plan.md`: 잎·꽃이 있는 13개 변형을 4프레임·250ms로 만드는 범위와 검수 기준 기록 — 나무마다 같은 제작 규칙을 적용하기 위해.
- `tools/render_scene_tree_sway.py`: 기존 3D 모델의 줄기·가지 면을 고정하고 잎·꽃 면만 네 자세로 렌더 — 기둥이 흔들리지 않는 공통 참조를 만들기 위해.
- `tools/prepare_scene_tree_sway.py`: 원화 시트를 나무별 원본 캔버스·뿌리 기준점·팔레트로 도트화하고 줄기 색 픽셀을 고정 — 네 프레임의 배치와 화풍을 맞추기 위해.
- `tools/spritetool/assets/scene-tree-sway/README.md`: 대상·프레임 설정·제작 명령·원화 생성 요청 기록 — 시안의 출처와 재생성 방법을 남기기 위해.
- `tools/spritetool/assets/scene-tree-sway/stree_pine_0`, `stree_pine_1`, `stree_broad_0`, `stree_broad_1`, `stree_willow`, `stree_old`, `stree_bush_0`, `stree_bush_1`, `stree_plum_0`, `stree_plum_1`, `peach_bloom_0`, `peach_bloom_1`, `peach_bloom_2`의 `reference-3d.png`·`painted-sheet.png`·`dot-sheet.png`: 13개 장면 나무의 3D 참조·원화·최종 도트 시트 추가 — 모든 잎나무를 같은 4프레임 방식으로 비교하기 위해.
- `docs/tree-sway-all-preview.html`: 13개 나무의 원본·250ms 애니메이션·3D 참조·네 프레임과 불탄 나무 정지 그림을 보여주는 HTML 추가 — 게임 적용 전에 한 화면에서 검토하기 위해.

## 2026-10-25 줄거리 문서를 fullstory.md 하나로

- `docs/fullstory.md`: 사용자의 원안 「한실중흥 설정과 시놉시스」에 구현된 1~2장, 1부 대본(3~7장), 구현 계획의 결정(화웅 사망, 목순 유지, 장료 추가, 7장 허리끈 장면 삭제, 화계서 무조건 지급)을 합침 — 여기저기 흩어진 줄거리 문서를 한 기준으로 모으기 위해.
- `docs/fullstory.md` 17장: 허도에서 헌제가 족보로 유비가 종친임을 확인하고 황숙이라 부르는 장면을 논영회 앞에 둠 — 2장에서 유언이 빠져 종친 설정을 처음 꺼낼 자리가 없어졌기 때문.
- `docs/fullstory.md` 22장: 단복은 영천에서 노모를 구해 준 서복이고, 유비가 알아보고도 가명의 사정을 묻지 않음 — 영천의 밤에 서복이 먼저 나오기 때문.
- `docs/fullstory.md` 23장: 박망파 뒤 영천 불 이야기("형님의 불은 백성을 태우지 않았습니다") 삭제 — 지금 영천에서는 유비의 불이 민가로 번지므로.
- `story_2026_10_23.md`, `docs/conceptual_scenario.md`, `../scenario_v1.txt`: 삭제(내용은 git 기록에 남음). 링크는 `docs/README.md`, `design.md`, `data-character.md`, `story-writing.md`, `todo/*.md`, `plan/*.md`, scene-staging·stage-debug 스킬에서 `fullstory.md`로 바꿈.
- `assets/scenario/hero/stage02.json`, `stage06.json`: 일기토 문구의 "쌍고검으로"를 "검으로" — 유비는 쌍고검을 받은 적이 없으므로.

## 2026-10-25 복숭아나무 줄기 고정, 꽃·잎만 흔들림

- `docs/tree-sway-plan.md`: 줄기·가지 고정과 꽃·잎 움직임으로 시안 기준 수정 — 나무 기둥이 흔들리는 이전 결과를 바로잡기 위해.
- `tools/render_peach_sway.py`, `tools/spritetool/assets/scene-peach-sway/reference-3d.png`: 갈색 줄기·가지의 3D 면은 그대로 두고 꽃·잎 면만 여덟 자세로 렌더 — 줄기 움직임을 없애기 위해.
- `tools/spritetool/assets/scene-peach-sway/painted-sheet.png`, `README.md`: 새 3D 자세를 바탕으로 꽃 표현을 다시 그리고 중간 원화의 역할과 프레임 규칙 기록 — 3D의 둥근 꽃 모양을 최종 도트에 남기지 않기 위해.
- `tools/prepare_peach_sway.py`, `tools/spritetool/assets/scene-peach-sway/dot-sheet.png`: 원화 칸을 뿌리 기준으로 정렬해 팔레트·외곽선 처리하고 줄기·가지 픽셀을 첫 프레임으로 통일 — 최종 도트에서도 나무 몸통을 고정하기 위해.
- `docs/tree-sway-preview.html`: 원화 중간 화면을 빼고 기존 나무·고정 줄기 3D·최종 도트만 비교 — 실제 검토 대상인 꽃·잎 움직임에 집중하기 위해.

## 2026-10-25 복숭아나무 3D 자세 기반 8프레임 시안

- `docs/tree-sway-plan.md`: 3D 자세·한 장 원화·팔레트 처리·HTML 검수 순서를 기록 — 시안 제작 범위와 기준을 분명히 하기 위해.
- `tools/render_peach_sway.py`: 기존 복숭아나무 모델의 가지를 여덟 자세로 굽혀 같은 카메라와 뿌리 자리에서 렌더 — 픽셀 이동 대신 실제 모델의 움직임을 참조하기 위해.
- `tools/spritetool/assets/scene-peach-sway/README.md`, `reference-3d.png`, `painted-sheet.png`, `dot-sheet.png`: 프레임 설정과 3D 참조·다시 그린 원화·팔레트 적용 도트 시트 추가 — 제작 경로를 보존하고 비교하기 위해.
- `tools/pixelize_tree.py`: 입력 이미지와 고정된 프레임 범위를 받도록 확장 — 여덟 칸에 기존 팔레트·외곽선 처리를 동일하게 적용하기 위해.
- `tools/prepare_peach_sway.py`: 원화 시트를 뿌리 기준으로 정렬한 뒤 93×96 도트 시트로 변환 — 프레임마다 크기와 기준점이 흔들리지 않게 하기 위해.
- `docs/tree-sway-preview.html`: 기존 그림·3D 자세·다시 그린 원화·최종 도트를 나란히 재생하고 여덟 프레임을 보여줌 — 게임 적용 전에 움직임과 그림체를 검토하기 위해.

## 2026-10-25 영천 전투까지의 유비 인물 초상 후보

- `tools/illustrations/portraits/story-preview/guanyu-early-portrait.png`, `zhangfei-early-portrait.png`, `caocao-early-portrait.png`, `xiahoudun-early-portrait.png`: 관우·장비·조조·하후돈의 초기 인물 초상 후보 추가 — 유비와 같은 시점의 성격을 얼굴과 태도로 비교하기 위해.
- `tools/illustrations/portraits/story-preview/early-portraits.prompts.md`: 네 인물의 묘사와 공통 수채화 프롬프트 기록 — 후보를 재생성하거나 수정할 수 있도록.
- `tools/illustrations/portraits/story-preview/liubei-early-portrait.png`: 짚신 장수 출신 의병장의 온화함과 걱정, 조용한 결단을 얼굴과 자세로 표현한 초상 후보 추가 — 1~2장의 인물상에 맞추기 위해.
- `tools/illustrations/portraits/story-preview/liubei-early-portrait.prompt.md`: 줄거리에서 뽑은 인상과 사용한 수채화 프롬프트 기록 — 초상을 재생성하거나 수정할 수 있도록.

## 2026-10-25 계교 전투 원소의 일반 퇴각

- `docs/plan-yuanshao-retreat.md`: 자동 퇴각 삭제 범위와 확인 방법 기록 — 원소 격파 승리 경로를 분명히 하기 위해.
- `assets/scenario/hero/stage10.json`: 인접하거나 병력이 절반 아래로 내려가면 원소를 즉시 제거하던 `yuanshao-retreat` 이벤트 삭제 — 원소의 병력이 0이 되었을 때 일반 퇴각으로 승리하도록 함.
- `internal/core/campaign_content_test.go`: 실제 계교 전투에서 원소가 인접·병력 49%에도 남고, 병력 0에서 승리하는 회귀 시험 추가 — 자동 퇴각이 다시 들어오지 않게 함.

## 2026-10-25 영천 일기토의 거리와 대사 순서

- `docs/plan-yingchuan-duel.md`: 강제 일기토 원인, 수정 범위와 확인 방법을 기록 — 전투 데이터와 화면 재생을 함께 고치기 위해.
- `assets/scenario/hero/stage02.json`: 파재 체력만으로 유비를 불러 일기토를 강제하는 이벤트를 제거 — 유비가 파재와 인접할 때만 기존 자동 일기토가 시작되게 함.
- `internal/gui/player.go`: 일기토 재생을 이벤트 순서에 놓고 일기토 결과를 종료 뒤에 적용 — 같은 행동의 대사가 일기토 뒤에 나오거나 파재가 일기토 전에 사라지던 오류 수정.
- `internal/gui/duel_scene.go`: 공격 연출 뒤 일기토 화면이 열릴 때까지 미리보기가 기다림 — 순서를 바로잡은 재생 방식에 맞춤.
- `internal/gui/duel_render_test.go`: `SRPG_DUEL_ID`로 특정 일기토만 캡처 가능 — 영천 연출을 독립적으로 확인하기 위해.
- `internal/core/campaign_content_test.go`: 관우가 파재를 공격하고 유비가 떨어져 있는 영천 배치를 검사 — 먼 거리의 유비 일기토 재발 방지.

## 2026-10-25 안드로이드 업데이트 뒤 옛 스테이지 파일 정리

- `android/src/com/orca/srpg/ver5/MainActivity.java`: 새 런타임 압축을 푼 뒤 압축에 없는 옛 파일을 런타임 폴더에서 제거 — 업데이트 후 남은 B03 스테이지가 여포의 옛 `Level` 값으로 시작을 막지 않게 함. 저장 파일은 별도 `saves` 폴더에 둔다.
- `docs/history.md`: 안드로이드 런타임 파일 정리 변경을 기록 — 업데이트 오류의 원인과 수정 의도를 남김.

## 2026-10-25 전투맵 배치 그림과 ascii-layout 스킬

- 결정: 장면뿐 아니라 전투맵 유닛 배치도 한 칸 3글자 그림으로 뽑고 고쳐서 되돌린다. 순서는 스킬 `ascii-layout`으로 묶는다 · 앞으로 배치 작업에 이 방식을 쓰기로(사용자 결정) · `data-map.md` 유닛 배치 그림, `presentation-scene.md`.
- `tools/mapgen/stage_ascii.py`(새 파일): `dump`·`load`. 지형 글자는 srpg-cli를 따르되 빈 땅은 `...`. 장수 코드는 `scene_ascii.py`의 `OFFICER`를 같이 쓴다. 평범한 들여쓰기 JSON이 아닌 스테이지 파일은 다시 쓰지 않는다.
- `tools/mapgen/layouts/hero/stage01.txt`~`stage10.txt`(새 파일): 지금 배치 그림.
- `tools/scene_maps/scene_ascii.py`: 같은 장수가 여럿일 때 코드 예비 후보(`Xu1`…).
- `../.claude/skills/ascii-layout/SKILL.md`(새 파일).

## 2026-10-25 스토리 장면 전부 정수 칸, 아스키 배치 그림

- 결정: 2장·3장 이후 모든 스토리 장면의 사람 위치와 카메라를 정수 칸으로 옮긴다. 겹치거나 막힌 사람은 가장 가까운 빈 정수 칸으로 · 정수 칸 규칙을 옛 장에도(사용자 결정. 전후 비교에서 앉은 사람이 의자·옥좌에서 벗어나는 것을 보고도 우선 전부 바꾸고 고칠 곳은 그림으로 고치기로) · `presentation-scene.md`.
- 결정: 장면 배치를 한 칸 3글자 아스키 그림으로 뽑고, 그 그림을 고쳐 다시 넣는다 · 사용자 제안(조조 `Cao`, 탁자 `Tbl`, 벽 `###`). JSON 좌표보다 한눈에 보고 고치기 쉽다. 그림은 첫 무대(배역 자리)만 고치고 걸음은 주석으로 보인다.
- `tools/scene_maps/scene_ascii.py`(새 파일): `whole`·`dump`·`load`. 서 있는 물체 목록은 `check_scenes.py`와 같게.
- `assets/scenes/ch02.json`·`ch03`·`ch05`~`ch11.json`: 정수 칸. 반올림한 칸이 막혀 옆으로 옮긴 곳이 88곳(회의 장면 제후들, 누운 부상병 등). `assets/scenes/ascii/*.txt`(새 파일 32개): 2장·3장 이후 장면 그림. 1장은 다른 작업이 진행 중이라 그대로.
- `../.claude/skills/scene-staging/check_scenes.py`: 자리 없는 배역(`Stay`, 탐험에서 이어받는 사람)을 건너뛴다 — 1·2장 탐험 이벤트 장면에서 멈췄다.

## 2026-10-25 성문 앞 화로, 스토리 장면은 정수 칸

- 결정: 스토리 장면의 위치는 1칸 단위 정수로만 잡는다 · 반 칸씩 옮겨 겹침을 피하는 배치를 사용자가 원하지 않음(사용자 결정) · `presentation-scene.md` 장면 맵.
- 결정: 성문 장면의 화톳불은 땅 위에 뜬 불 대신 화로 소품 위의 불로 · 땅 위의 불이 소나무 꼭대기에 붙어 보였다(사용자 제안).
- `tools/scene_maps/build.py`(`EXTRA`), `assets/graphics/maps/S_castle_out_open.json`: 다리 앞 길 양쪽 (2, 3)·(4, 3)에 화로. 맵 소품이라 사람과 앞뒤가 맞게 그려진다(장면 `Prop`은 사람 위에 그려진다).
- `assets/scenes/ch02.json`: `c02-night`·`c02-gate`·`c02-camp-empty`의 모든 위치를 정수 칸으로. 황보숭은 두 화로 사이 (3, 3)에 선다 — (3, 4)이면 왼쪽 화로와 화면 같은 줄이라 불이 머리 위에 겹쳤다. 진영 맵은 막사·깃발이 소수 자리에 있어 비어 있는 정수 칸을 검사로 찾아 썼다.

## 2026-10-25 장면의 등장·퇴장을 서서히, 성문 앞 다리, 파재 전사

- 결정: 장면 배우의 `Exit`는 갑자기 사라지지 않고 전투 퇴각처럼 한 칸 반 더 걸어 나가며 흐려지고, `Enter`는 서서히 나타난다. 들고 나는 자리는 화면 밖(맵 끝 쪽)에 둔다 · 형제가 불 끄러 나갈 때, 조조가 오고 갈 때 그 자리에서 나타나고 사라져 어색했다(사용자 지적) · `presentation-scene.md`.
- `internal/gui/player.go`: `unitVis.arriving`(나타나는 남은 시간), `leaving()`이 나타나는 중에도 흐림 정도를 준다. `internal/gui/scene.go`: 경로 끝의 `Exit`에 `leave`(바라보는 쪽 `sceneStep`), `Enter`에 `arriving`, 건너뛰기·중간 진입은 바로 끝 상태.
- `assets/scenes/ch02.json`: 서복은 남서쪽 화면 밖 (2.5, 23)에서 들어오고 형제·의병은 그쪽으로 나간다. 조조·하후돈은 북동쪽 화면 밖 (2.5, −18)에서 오고 그리로 돌아간다. 황보숭은 성문 통로 (3, 0.4)에서 나타난다.
- `tools/scene_maps/build.py`, `assets/graphics/maps/S_castle_out.json`·`S_castle_out_open.json`: 다리 소품이 놓인 물 칸을 `b`로 — 다리가 바닥과 함께 구워져 움직이는 물에 덮여, 황보숭이 물 위를 건너는 것처럼 보였다(사용자 지적). 게임은 `b` 칸의 다리를 물 다음에 그린다.
- `assets/scenario/hero/stage02.json`: 유비와 파재의 일기토에서 파재가 죽는다(`EnemyResult: death`, 마지막 비트 `win`) · 사용자 결정.
- `assets/scenario/hero/story01.json`, `assets/scenes/ch01.json`: 탁군 출진 장면의 관우 대사 "쫓는 데 정신이 팔린 놈들은 등이 비는 법이오."를 뺀다 — 형제가 등 뒤가 아니라 성문 앞에서 맞는 새 배치와 어긋났다.

## 2026-10-25 영천의 밤: 2장 끝을 서복·황보숭·빈 진영의 조조로

- 결정: 영천 전투 뒤 장면을 통째로 바꾼다. 서복이 성 안 불을 알리고 형제가 달려감 → 성문 앞에서 손견·조조(말에서 내린 채)가 말 탄 황보숭을 맞음 → 조조·하후돈이 빈 진영에서 형제의 목소리만 듣고 책을 병사에게 맡김 · 조조와 유비가 여기서 마주치지 않아 3장 낙양의 첫 대면이 산다. 연출 다섯 안을 찍어 보고 사용자가 고른 6안(밤, 남서쪽 불빛, 화면 밖 말풍선) · `../story_2026_10_23.md` 2장 끝, `presentation-scene.md`.
- 결정: 대본은 장면 1·3 다판(서복의 어머니, 바람 탓 불), 장면 2 나판(손견의 호방함·조조의 "물동이"). 화계서는 조건 없이. 하후돈은 조조를 "맹덕"이라 부르며 평어(사용자 결정) · `data-speech-style.md`.
- 결정: 황보숭이 말에서 내리는 연출은 없앤다. 말 탄 사람은 장면 배역 `Sprite: light_cavalry`로 그 인물의 색을 입힌 기병 시트를 쓴다 · 말→보병 전환이 어색했다(사용자 지적).
- `internal/gui/scene.go`·`field_night.go`: 효과 `bigfire`(불꽃 셋·연기, 밤에 반지름 440·느리고 큰 흔들림·짙은 불빛). 말풍선이 화면 밖 사람이면 화면 가장자리(대화창 위)에 붙고 꼬리가 그쪽을 가리킴. `scenario_test.go`: 효과 목록. 이름 `blaze`는 데이터의 기술 이름과 겹쳐 순수성 시험에 걸렸다.
- `assets/scenario/hero/story02.json`: `c02-dawn`~`c02-end`를 `c02-night`(n1~n7)·`c02-gate`(g1~g8)·`c02-camp-empty`(e1~e9)·`c02-grant`·`c02-end`로. `assets/scenes/ch02.json`: 세 장면(진영·성 밖 성문 열림·진영, 모두 밤).
- `assets/data/characters_story.json`: 서복(`xufu`, 비전투 보병 도트, 초상 없음). 하후돈·하후연 초상 연결(그림은 있었는데 `Portrait`가 빠져 대화창에 안 나왔다).
- `assets/scenario/hero/stage07.json`: 하후돈 일기토 말 둘을 "맹덕"으로.
- `../story_2026_10_23.md`: 3장의 옛 S#1(영천 성문 앞)·S#2(유비의 진영)를 지우고, 노식의 죄목을 "재물을 빼돌렸다"로(2장에서 백성에게 돌려준 재물).

## 2026-10-25 탁군 전투를 한 줄 배치로

- 결정: B01을 남쪽부터 탁현(추정 혼자) → 유관장 → 상인 → 등무 → 정원지의 한 줄로 세우고 성 밖 정원지 주둔지를 없앤다 · 형제가 상인과 성문 사이에 서야 상인을 지키기 쉽다(사용자 요청) · `../story_2026_10_23.md` 탁군 전투.
- `assets/scenario/hero/stage01.json`: 아군 (6,9)(6,8)(9,8), 개전 걸음 한 칸 북쪽. 상인 (7,5)(8,5)는 2라운드에 성문에 닿는다(6행이면 1라운드라 지킬 것이 없었다). 등무 조 4행, 정원지 조(`camp`→`rear`) (7,0)(6,1)(8,1) — 정원지를 병졸 뒤에 둔다. 길가 무리 `road1`·`road2`를 성문 옆 (4,7)(11,8)에서 등무 줄 양옆 (4,4)(11,4)로 — 성문 옆이면 개전에 유비를 둘러싸 막 나가는 플레이어가 2라운드에 유비를 잃었다. 수비병 삭제. 적 수·직급은 그대로.
- `tools/mapgen/layouts/hero/map01.txt`·`assets/scenario/hero/maps/map01.json`: 주둔지 칸 (2,4)(1,5)를 초원으로.
- `internal/core/spawn_override_test.go`: 덧붙이는 시험 병력을 빈 칸 (12,2)로 — (9,4)에 추격조가 선다.
- 수치(`cmd/_balance`, 20판/10판): 막 나가는 플레이어 20/20(전 10/10), 게임 AI 10/10(장비가 대부분 한 번 퇴각, 관우 +2.7), `TestRushWinsTheFirstStages` 통과. 정원지를 병졸 앞 (7,1)에 두거나 줄을 더 북쪽으로 몰면 B01이 3라운드로 짧아져 성장이 모자라 B05에서 졌다.

## 2026-10-25 1부 줄거리 문서를 1·2장 대본 B에 맞춤

- 결정: `story_2026_10_23.md`의 1장(도원결의·탁군 전투)과 2장 스토리(영천 전투 직전까지)를 구현된 대본 B로 고쳐 쓴다 · 구현이 문서와 달라졌는데 문서는 그대로였다(사용자 요청). 영천 전투와 3장 이후는 그대로 둔다.
- 원안과 달라진 점과 이유: 첫 장면을 방문 앞 장비로(탐험이 열리자마자 시작) · 결의 전에 유비가 주점의 술꾼을 설득해 첫 지원자로 만들고 관우가 그걸 보고 "주공"을 꺼낸다(관계에는 근거 장면이 먼저) · 탁군 전투는 성 포위 대신 장세평·소쌍을 쫓는 등무를 치고, 상인이 2장 저택의 감사와 대장간의 청룡언월도·장팔사모로 이어진다(일화를 여러 장에 엮기) · 치하와 출진 명은 유언 대신 추정이 맡고, 탁군을 지킬 사람이 있어 유비가 떠날 수 있다(동기를 앞에서 깔기) · 노식 진영은 포위 중이 아니라 싸움이 끝난 뒤, 노식이 약탈품을 달라는 좌풍을 물리치고 재물을 백성에게 돌려준다(말보다 행동).
- `../story_2026_10_23.md`.

## 2026-10-25 이야기 쓰는 법 문서

- 결정: 확정한 1·2장 대본 B의 기조(연의의 정조 위에 생활감과 웃음, 지위·관계는 근거 장면 먼저, 말보다 행동, 판본 2~3개를 html로 비교해 고르기)를 문서로 남기고 AGENTS.md에서 가리킨다 · 새 세션도 같은 기조로 쓰게 하기 위해(사용자 요청) · `story-writing.md`.
- `docs/story-writing.md`(새 파일), `docs/README.md`, `../AGENTS.md`.

## 2026-10-25 용무늬 대화창과 타이틀

- 결정: 대나무 두루마리 대화창을 용무늬 배경 + 검은 테두리 상자로 바꾸고, 첫 화자의 진영색으로 물들인다 · 사용자 요청(HTML 시안 확인 후). 진영색은 또렷이 갈릴 필요 없는 장식 · `presentation-scene.md` 대화창.
- 결정: 진영마다 그림을 미리 만들지 않고 회색조 한 장에 색 배율 · 파일 하나로 끝나고 진영이 늘어도 할 일이 없으며 화자가 바뀔 때 색을 넘기기 쉽다. 원본(1254px)을 1/4로 줄여 75KB.
- `internal/gui/dialogue_box.go`(`dialogue_scroll.go`에서 이름 바꿈): 두루마리 부품 대신 `drawDragon`(한 사각형에 `AddressRepeat` — 타일을 여러 장 찍으면 선형 필터가 가장자리에서 투명을 읽어 이음선이 보였다), `factionTint`, 검은 테두리·펼침 끝 막대, 밝은 글자, 초상·이름패 검은 테두리. `scrollAnim`→`dialogueAnim`(쓰지 않던 닫힘 동작 삭제).
- `internal/gui/game.go`: 타이틀 배경을 용무늬로(천천히 흐름, 가장자리 어둡게). `internal/gui/assets.go`: `Assets.Scroll`→`Dragon`. `internal/gui/uidemo_title.go`(새 파일): `-ui title`.
- `assets/graphics/ui/dragon.png`·`tools/ui/dragon_tile.py`(새 파일). `assets/graphics/ui/scroll/`·`tools/ui/scroll_parts.py` 삭제. UI 시트의 `scroll_curl`·`scroll_grain`은 이제 쓰지 않는다(`../tools/spritetool/assets/ver5-ui`에 남아 있음).

## 2026-10-24 탐험 모드와 1부 1~2장 (계획 2026_10_24_explore_mode)

- 결정: 전투 사이의 이야기를 유비가 걸어 다니는 탐험으로 진행한다. 사람에게 말을 걸면 이벤트(기존 장면 연출), 심복에게 말을 걸면 출진, 무기점·도구점·주점은 실제 상점, 재화는 "돈" · 사용자가 영걸전처럼 마을·주둔지·진영을 돌며 이야기를 보고 싶어 함(1부 계획의 "엔진 신규 기능 없음"을 바꿈) · `presentation-explore.md`, `plan/2026_10_24_explore_mode.md`.
- 결정: 이벤트 뒤 사람의 자리는 저장하지 않고 늘 플래그로 다시 세운다 · 저장은 지역·유비의 칸·방향·돈·플래그만으로 되게 · `presentation-explore.md`.
- `internal/content/explore.go`(새 파일)·`content.go`·`scenario.go`·`validate.go`: 탐험 파일 `exploreNN.json`, 노드 `explore`·`back`, 스토리 파일의 `Events` 목록(흐름 밖 노드, 이벤트끼리만 이어짐), 상점 `Shops`, `Price`, `Campaign.Money`, 보상·지급의 `money`. 이벤트가 모든 갈래에서 `back`으로 끝나는지 검사.
- `internal/core/explore.go`(새 파일)·`engine.go`·`api.go`·`types.go`: `State.Money`·`Explore`·`Spot`, 명령 `walk`·`talk`·`sortie`·`buy`·`sell`, `Visitors`·`OpenExits`·`ExploreNext`(완주 시험·자동 진행). 탐험 중에도 장비·부관 변경. 저장 형식 5(옛 저장은 받지 않음). 보상 장비는 창고로.
- `internal/cli/explore.go`(새 파일)·`cli.go`·`play.go`: 탐험 번호 메뉴와 상점.
- `internal/gui/explore.go`·`shop.go`·`walkmap.go`(새 파일), `tour.go`·`scene.go`·`scenario.go`·`game.go`·`windows.go`: 탐험 화면(사람의 순찰·배회·모션, 대화, 출진 확인, 이벤트는 같은 무대에서 이어받기 `adoptScene`, 상점·부대 창), 작은 칸 걷기는 둘러보기와 한 코드. `internal/scenes`: 배역의 `Stay`.
- `cmd/_town`, `internal/town`, `internal/gui/town.go`·`uidemo_town.go`, `assets/towns/zhuo.json` 삭제 — 옛 마을 데모(사용자 결정).
- 장·전투 번호를 1부 최종 번호로: 이야기 1→2, 2→3, 3→5 … 9→11, 전투 B01→B02, B02→B05 … B07→B10. 1·3·4장과 B01·B03·B04는 탁군·단양·팽성 자리.
- `assets/scenario/hero/explore01.json`·`explore02.json`·`story01.json`(새 파일), `story02.json`, `assets/scenes/ch01.json`(새 파일)·`ch02.json`: 1장(방문 앞 장비 → 주점 → 복숭아 숲 결의 → 의병 주둔지 출진)과 2장(관청 추정 → 노식의 진영 → 출진). 노식 진영 첫 장면은 이벤트로 옮김.
- `assets/scenario/hero/stage01.json`·`maps/map01.json`(새 파일), `tools/mapgen/layouts/hero/map01.txt`(초안에서 옮김): 탁군 전투. 정원지·등무와 두 일기토를 영천에서 옮김. 3턴에 등무 조가 성문에서 돌아서 본진 쪽으로.
- `assets/scenario/hero/stage02.json`: 영천에서 정원지·등무·두 일기토·`chengyuanzhi-sorties`·부목표를 뺌, 파재 출격은 5라운드. 보상에 돈.
- `assets/data/characters_story.json`: 추정, 좌풍. `assets/data/shops.json`(새 파일), `equipment.json`·`items.json`의 `Price`, `scenario.json`의 `Money`.
- `assets/graphics/maps/S_peach_grove2.json`·`S_army_camp2.json`(형식 2 사본), `S_militia_camp.json`(새 파일, 진영 사본에서 막사를 줄이고 무기대·모닥불): 탐험 지역.
- `cmd/_walkmap`(새 임시 도구), `internal/gui/uidemo_explore.go`(`-ui explore`), `internal/gui/explore_test.go`.
- 앞에 전투가 하나 들어와 뒤 전투의 스테이지 레벨이 한 단계씩 올랐다(시험 기대값 갱신). 9장 이후 난이도는 다시 볼 일.

## 2026-10-24 장면 편집기 2~4단계: 물체·성벽·성문·검사 (`cmd/_scene`)

- `internal/scenemap/plan.go`: 물체 그림(`Draw`)에 맵 물체 번호(`Object`, JSON `o`) — 편집기가 클릭한 그림의 물체를 알기 위해. 게임은 쓰지 않는다.
- `internal/scenemap/check.go`(새 파일)·`check_test.go`: `Check` — 성벽 줄 위에 선 물체, 같은 자리에 같은 물체 둘, 한 칸 이상 넓은 물체끼리 겹침. 작은 물체가 모여 선 것과 얇은 것(벽·휘장)이 붙은 것, 격자 밖 물체는 일부러 그렇게 놓은 맵이 있어 묻지 않는다(지금 맵들에는 문제 0개).
- `cmd/_scene/main.go`: `POST /api/check`(문제와 걷는 칸).
- `cmd/_scene/static/editor.js`·`viewer.js`·`index.html`·`style.css`: 물체(선반·놓기·고르기·끌기·화살표·지우기·칸에 맞춤·U V dx dy 고치기·그림 바꾸기), 성벽(끌어서 줄, 오른쪽 끌기로 칸 빼기), 성문(성벽 위 클릭, 열림/닫힘, 지우기), 검사 표시. 커서 칸을 맵의 점 변환(`point`)의 역으로 다시 계산하고 높은 칸 윗면을 잡는다 — 1단계의 땅 높이 한계를 고침.

## 2026-10-24 장면 편집기 1단계: 바닥·높이 (`cmd/_scene`)

- 결정: 장면 맵 편집기는 게임 안이 아니라 브라우저 뷰어(`cmd/_scene`)를 키워 만든다. 규칙 계산은 서버의 `internal/scenemap`이 하고 브라우저는 입력과 그리기만 · 같은 규칙을 Go 한 곳에 두면서 입력 화면(붓 목록·썸네일·팝업)을 ebiten으로 새로 짜는 비용을 피함(사용자와 합의) · `tools-scene-viewer.md`.
- `internal/scenemap/format.go`: `Map.Encode` — 맵 파일 모양(키 한 줄, 격자 한 줄, 물체·성벽 줄 한 줄, 숫자 붙여 씀) 그대로 쓰기. `format_test.go`: 형식 2 맵 전부가 읽고 다시 쓰면 글자 그대로.
- `cmd/_scene/main.go`: `POST /api/preview`(보낸 맵의 그리기 목록), `POST /api/save/<id>`(읽기·그려 보기가 되면 덮어쓰기).
- `cmd/_scene/static/editor.js`(새 파일)·`viewer.js`·`index.html`·`style.css`: 편집 줄(보기·바닥·높이, 붓, 붓 크기, 격자, 되돌리기·다시·저장), 칠하기·집기, 편집 중 맵은 회전·다시 읽기에도 유지, 저장 안 한 편집 경고.

## 2026-10-24 성문 통로 안쪽 면을 어둡게

- `../assets_candidate/tools/gen_pixel_castle.py`: 아치 어깨의 면 중 줄 방향을 보는 면(통로 안쪽)과 통로 옆벽 전체를 통로 색으로 — 열린 문에서 아치 위 어깨 면만 바깥 벽돌색으로 밝게 남아 어색하던 것(사용자 지적). 성벽 바깥면에 보이는 어깨는 돌 그대로. 성문 조각 40장, `map/extra.png`·`index.json` 다시 묶음.

## 2026-10-24 성벽·성문을 도트로 직접 (형식 2)

- 결정: 형식 2 성벽·성문은 3D 렌더 대신 작은 칸 격자에 도트를 직접 찍는다 · 3D 키트는 칸 정렬을 지키려 모든 부품을 성벽 면에 붙이자 성문이 "구멍 낸 성벽에 집을 얹은" 꼴이 되고 기둥이 떠 보였다. 사용자가 아스키 벽돌 무늬를 2:1로 기울여 쌓자고 제안 · `presentation-scene.md`, `tools-build.md`.
- `../assets_candidate/tools/gen_pixel_castle.py`(새 파일): 칸 단위 상자·볼록 다면체를 화면 투영으로 광선 판정해 면마다 무늬로 칠함. 성벽 몸통 `swall2`, 성가퀴 `sparapet_{mu,mv,pu,pv}`, 밑동 그림자 `swall2_shade_{pu,pv,pc}`, 성문 `sgate2_{open,closed}_{u,v}_{0..9}`(v 방향도 반전 없이 직접 그려 +v면 밝게·+u면 그늘). 돌은 사용자가 고른 문루 돌, 문루는 성벽과 같은 높이, 지붕은 처마·위 두 단 처진 기와(기와골·막새·용마루·치미). 열린 문 통로는 반투명 검정 — 밝은 바닥이 비쳐 어색하던 것(사용자 지적). `swall2_u`·`swall2_v` 삭제.
- `internal/scenemap/catalog.go`·`plan.go`, `assets/graphics/scene.json`: `walls`는 `{block, parapet, shade}`. 성벽 칸은 몸통 + 보기 기준으로 옆 칸이 성벽·성문이 아닌 옆면의 성가퀴(뒤쪽 먼저), 칸마다 앞이 트인 면에 바닥 그림자 먼저. 줄이 겹치는 칸은 하나로(성문 우선). 모서리 그림자는 줄 방향 대신 앞에 성벽 칸이 없는 면에서(`wallAxis` 삭제) — 두 줄이 만나는 모서리에서 성가퀴가 교차하던 것(사용자 지적), 성벽 밑 바닥이 너무 밝던 것(사용자 지적).
- `../assets_candidate/tools/gen_3d_castle.py`, `../tools/unit3d/render.py`: 블록 키트(`--write2`)와 `owner` 인자를 되돌림 — 쓰는 곳이 없어짐.
- `internal/scenemap/plan_test.go`(성가퀴·그림자 규칙, 모서리 그림자 높이 72), `internal/gui/scene_maps_test.go`: 새 그림 이름. `map/extra.png`·`index.json` 다시 묶음.

## 2026-10-24 성벽·성문 부품을 칸 안으로 (블록 키트)

- `../assets_candidate/tools/gen_3d_castle.py`: 블록 키트의 모든 부품을 자기 작은 칸 안에 둔다(`FLUSH`): 기둥은 성벽과 같은 두께의 1×1 탑으로 성벽보다 1.4칸 높게, 받침돌·윗단 띠·성문 처마·기둥 받침은 칸 밖으로 나가지 않게. 성가퀴는 걷는 길 앞뒤 양쪽에 낮게(`parapets`). 성문은 양옆에 성벽 블록을 붙여 렌더해 끝에 바깥선이 생기지 않게 하고, 그 둘의 픽셀은 버린다. 성벽 블록은 뒤 이웃 위의 바깥선만 뺀다 — 장면 규칙은 칸 단위로 앞뒤를 정하므로, 칸 밖으로 튀어나온 부품(두께 1.4칸 기둥)은 앞쪽 끝에서 앞 칸 성벽에 덮이고 뒤쪽 끝에서는 튀어나와 두 기둥이 가이드선에서 다르게 보였다(사용자 지적). 성가퀴가 앞쪽에만 높게 있어 성벽이 한 칸 두께로 보이지 않던 것도.
- `../tools/unit3d/render.py`: `owner`는 보이는 면에서 시선 방향으로 아주 조금 안쪽 점으로 판정 — 칸 경계에 걸친 면이 바깥 칸 몫으로 잡혀 기둥 면이 비던 것.
- 그림 44장, `map/extra.png`·`index.json` 다시 묶음.

## 2026-10-24 장면 성벽·성문을 3D로 다시 (형식 2)

- 결정: 형식 2 맵의 성벽은 작은 칸 하나를 통째로 그린 블록, 성문은 작은 칸 10개를 칸마다 나눈 조각으로 바꾼다 · 옛 성벽 그림은 앞면 16px 띠만 잘라 써서 걷는 길과 뒤쪽이 잘려 얇게(성가퀴가 붙은 듯) 보였고, 성문 기둥이 칸 밖으로 넘쳐 이어지는 성벽과의 앞뒤가 어색했다(사용자 지적) · `presentation-scene.md`.
- `../tools/unit3d/render.py`: `render(..., owner=, owners=)` — 모델 좌표로 각 픽셀을 차지한 부분을 알려 준다(기존 호출은 그대로) — 그림을 칸 단위로 나누기 위해.
- `../assets_candidate/tools/gen_3d_castle.py`: 블록 키트(`--write2`, `--preview2`): `swall2_u`·`swall2_v`, `sgate2_{open,closed}_{u,v}_{0..9}`. 높이 3.2·두께 1칸·성문 구조는 옛 것과 같다. `../assets_candidate/tiles/tiles.json`과 새 그림 44장, `build_extra.py`로 `map/extra.png`·`index.json`을 다시 묶음.
- `internal/scenemap/plan.go`·`catalog.go`·`walk.go`, `assets/graphics/scene.json`: 성문 조각은 모두 그 칸 성벽 블록을 대신한다(가장자리 조각 예외와 앞뒤 비교 삭제), 회전 시 조각 순서는 뒤집기, `gates`는 물체 이름 → 조각 이름, 통로 칸은 `gate_passage`.
- `assets/graphics/maps/S_castle_town.json`: u 방향 성벽 줄을 뒤 모서리 칸(−1, −1)부터 — 두꺼운 성벽에서 모서리가 비지 않게.
- `internal/scenemap/plan_test.go`, `internal/gui/scene_maps_test.go`: 새 그림 이름과 성문 조각 자리.

## 2026-10-24 성문 앞쪽 기둥과 성벽의 앞뒤

- `internal/scenemap/plan.go`: 성문 가장자리 조각(기둥 끝)과 성벽 조각이 한 칸에 있으면 깊이로 순서를 정한다: 그 칸이 성문 중심보다 가까우면 성벽을 나중에 — 성 안 민가·탁군처럼 성문 앞쪽(화면 왼쪽 아래)으로 성벽이 이어지는 곳에서 성문 기둥이 더 가까운 성벽 위에 그려지던 것. 옛 `seat_gates`의 "가장자리 조각은 성벽 조각 뒤에" 규칙은 뒤쪽 끝에만 맞았다. 성 안 민가 비교 차이 14,391 → 15,198픽셀(의도).
- `docs/presentation-scene.md`: 위 규칙.

## 2026-10-24 형식 2 둘러보기 데모 `cmd/_town2`

- `cmd/_town2/main.go`·`internal/gui/tour.go`·`uidemo_tour.go`·`game.go`(새 파일·연결): 유비가 형식 2 장면 맵을 작은 칸 단위로 걸어 다니고, 출구(이름판·건물·출구 자리)를 누르면 다음 맵으로 가는 데모 — 새 규칙으로 그린 맵을 직접 걸으며 보기 위해. 사람·대사는 없다.
- `internal/scenemap/walk.go`(새 파일), `assets/graphics/scene.json`(`walkable`): 걸을 수 있는 작은 칸을 땅 글자·높이·물체 footprint·성벽 줄(성문 통로는 열림)로 정하는 규칙 — 마을 파일의 `Footprints`·`Ground` 목록 같은 따로 둔 데이터 없이 맵 규칙에서 바로 정하기 위해.
- `assets/towns/tour2.json`(새 파일): 탁군 사본 6곳·관문·성 안 민가·군막과 출구.
- `assets/graphics/maps/S_zhuo2.json` 외 5개(새 파일), `index.json`: 옛 탁군 맵의 형식 2 사본 — 원본과 `cmd/_town`을 그대로 두기 위해.
- `tools/scene_maps/scene_v2.py`: `--copy <접미어>`로 사본을 만들고 `index.json`의 `maps`에 올린다.
- `internal/scenemap/plan_test.go`(`TestWalkable`), `internal/gui/scene_maps_test.go`(`TestTourAreas`): 관문의 길·절벽·성벽·성문 통로·망루 아래, 둘러보기 지역마다 출구에 닿는지.
- `docs/presentation-town.md`: 둘러보기 사용법과 데이터.

## 2026-10-24 장면 렌더링 규칙과 장면 맵 형식 2 (계획 2026_10_24_scene_render 1~3단계)

- 결정: 장면 맵의 그리기 규칙을 Go 패키지 `internal/scenemap` 한 곳에 두고, 뷰어(`cmd/_scene`)와 게임(`internal/gui`)이 같은 그리기 목록을 쓴다 · 계획은 뷰어(JS)와 게임(Go)에 같게 두 번 구현하고 비교 시험으로 맞추는 것이었으나, 한 곳에 두면 어긋날 일이 없다 · `presentation-scene.md`, `tools-scene-viewer.md`.
- `internal/scenemap/format.go`·`catalog.go`·`plan.go`·`order.go`: 형식 2(땅 글자·높이·물체·성벽 줄)를 읽어 평면층 → 입체층 순서의 그리기 목록을 만든다. 바닥은 입체층을 덮지 않고, 입체층은 footprint 앞 모서리·높이·종류·화면 x로, 여러 칸 물체는 footprint 앞뒤로 정렬한다. 높은 땅 블록·성문 조각·모서리 그림자·절벽 밑동 띠를 규칙으로 고르고, 회전은 데이터를 돌린 뒤 같은 규칙을 쓴다 — 맵마다 붙던 예외(앞 절벽을 반 칸 늦게 그리기, 성벽 `_up`, 회전 시 성문 조각·그림자 숨김)를 없애기 위해.
- `internal/scenemap/scenery.go`, `internal/gui/field_scenery.go`: 바닥 장식(풀·꽃·돌·물가)의 흩뿌리기 계산을 패키지로 옮겨 전투 맵·장면 맵(게임)·뷰어가 같은 함수를 쓴다 — 장면 맵 장식을 뷰어와 게임이 같게 그리기 위해. 전투 맵의 결과는 그대로다.
- `assets/graphics/scene.json`(새 파일): 땅 글자의 이음 성질, 실내 바닥 글자(`T`·`W`·`P`·`C`)와 포석 `i`, 높이 단계 2(높은 절벽), 성벽 계열, 성문, 모서리 그림자 위치, 여러 칸 물체의 footprint — 물체·규칙 정보를 맵이 아니라 목록에 두기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_tent_mid.json`·`S_castle_town.json`: 형식 2로 옮김(197KB→7.4KB, 8KB→1KB, 114KB→4KB) — 시험 맵 셋. `S_checkpoint_closed`·`S_castle_town_burnt`는 아직 옛 형식.
- `tools/scene_maps/scene_v2.py`(새 파일): 옛 맵(파일 또는 git 판)을 형식 2로 옮김. 물체 위치는 옛 뷰어가 그리던 픽셀에 맞춰 반올림한다 — 4단계에서 나머지 맵에도 쓰기 위해.
- `tools/scene_maps/build.py`: `floor_half`가 음수 짝수를 한 칸 틀리게 내림하던 버그 수정 — 맵 바깥 가장자리에 물 테두리가 생기던 원인.
- `tools/scene_maps/compose.py`: 이미 형식 2인 맵은 덮어쓰지 않음 — 4단계에서 도구를 형식 2로 바꾸기 전까지 시험 맵을 지키기 위해.
- `tools/build_extra.py`, `assets/graphics/index.json`: 성문 조각의 `gate_offset`을 넘김. `../assets_candidate/tiles/tiles.json`(휘장·병풍)과 `../tools/unit3d/props.py`·`index.json`(방 벽 `wall_*`)에 `"shadow": false` — 그림자를 이름으로 거르던 게임 코드의 예외를 데이터로 옮기기 위해.
- `internal/gui/field_scene.go`(새 파일)·`field.go`·`assets.go`·`editor.go`: 형식 2 맵은 평면층을 한 번 구워 두고, 입체층 조각(여러 그림인 블록은 미리 한 장으로 합침)을 사람과 함께 `scenemap.Order`로 정렬해 그린다. 높은 땅 위의 사람은 그 높이만큼 올린다. 물체 그림자의 `wall_`·`curtain_`·`screen_` 이름 예외를 지움 — 뷰어와 같은 그림을 내기 위해.
- `internal/gui/scene_maps_test.go`: 옛 관문 형식 시험을 지우고, 게임 Field의 입체층 목록이 규칙 패키지 목록과 같은지·관문 성벽 앞뒤의 사람 순서를 시험. 관문 그리기 벤치마크: 맵 전체 1.25ms, 1280×720 화면 약 1.0ms(옛 형식 0.65ms; 높은 땅 2,340칸이 각자 정렬되는 비용) — 성능 기록.
- `internal/scenemap/plan_test.go`(새 파일): 회전, 정렬, 성문 조각, 모서리 그림자, 실내 바닥 시험.
- `internal/cli/scenes.go`, `../.claude/skills/scene-staging/check_scenes.py`: 형식 2 맵을 읽음(실내 바닥 글자는 설 수 있는 곳) — 텍스트판 장면 지도와 배치 검사가 계속 돌게 하기 위해. 배치 검사 0건.
- `cmd/_scene/main.go`·`static/viewer.js`·`index.html`·`style.css`·`legacy.js`(새 파일): 형식 2는 규칙 목록으로 그리고, 층 단계·새 규칙/기존/비교 보기·장식·footprint·정렬 번호 표시를 더함. 옛 형식은 옛 코드(`legacy.js`)로 그대로 — 기존 데이터와 픽셀로 비교하기 위해. 비교 기준은 태그 `scene-render-base`(백업 커밋 46282a6).
- 비교 결과(장식 없이, 0°): 관문 6,922·군막 6,423·성 안 민가 14,391픽셀이 다르고, 모두 이유가 있다: 바닥이 덮던 물체 발치·절벽 밑동 띠, 마루 틈의 풀, 옛 `floor_half` 버그의 가장자리 물 테두리, 맵 바깥에 지던 밑동 띠, 포석, 모서리 그림자 세로 1px(두 맵의 옛 위치가 서로 달랐다), 윗면 외곽선 끝 반투명 픽셀. 회전하면 미세 위치를 정수 픽셀로 적은 탓에 소품이 1px 밀리는 곳이 있다.
- `docs/presentation-scene.md`·`tools-scene-viewer.md`·`tools-build.md`: 형식 2와 그리기 규칙, 뷰어 사용법, 변환 도구.

## 2026-10-24 공통 장면 맵 뷰어

- `cmd/_scene/main.go`: `S_` 장면 맵 40개와 게임 그림 자료를 제공하는 로컬 뷰어 서버 추가 — 관문 외 장면도 같은 화면에서 고르기 위해.
- `cmd/_scene/static/index.html`·`style.css`·`viewer.js`: 바닥·장식·소품을 공통 순서로 그리고 배율·스크롤·회전·바닥 기준 커서와 하이라이트를 제공; 자유 소품은 작은 칸 중심, 면 소품은 피벗 보정, 높은 절벽은 한 번 출력 — 확정한 관문 그리기 규칙을 비교 스위치 없이 여러 장면에 적용하기 위해.
- `docs/tools-scene-viewer.md`: 실행법과 기존 `props` 장면의 좌표 처리 범위를 설명 — 맵 자료 형식에 따른 화면 의미를 기록하기 위해.
- `docs/README.md`: 공통 장면 맵 뷰어 문서 연결 — 도구를 찾기 쉽게 하기 위해.
- `docs/history.md`: 변경 파일과 이유를 기록 — 공통 뷰어 추가 이력을 남기기 위해.

## 2026-10-24 관문 장면 기본 배치 확정

- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: `small_snap: true`, `cliff_repeat: false`를 저장 — 자유 소품을 작은 칸 중심에 놓고 절벽을 한 번만 그리기 위해.
- `tools/scene_maps/compose.py`: 관문 맵을 다시 생성해도 같은 기본 설정을 기록 — 재생성 후 장면 출력이 달라지지 않게 하기 위해.
- `internal/gui/assets.go`·`field.go`: 맵의 두 설정을 읽어 자유 소품 위치와 절벽 출력 횟수에 적용 — 실제 장면이 선택한 배치를 사용하게 하기 위해.
- `internal/gui/scene_maps_test.go`: 관문 두 상태의 설정과 소품 좌표를 검사하고 절벽 한 번 그리기 벤치마크를 기록 — 장면 출력 회귀를 확인하기 위해.
- `docs/S_checkpoint.html`: 기본 선택을 `끄기`·`반올림 O`·`작은 칸 기준`으로 맞추고 장면 설정을 표시 — 비교 화면의 첫 상태를 실제 장면과 일치시키기 위해.
- `docs/scene-tile-faces.md`·`presentation-scene.md`: 관문 장면의 배치 설정과 절벽 출력 횟수를 설명 — 자료와 출력 규칙을 분명히 하기 위해.
- `docs/history.md`: 변경 파일과 이유를 기록 — 관문 장면 기본 설정의 이력을 남기기 위해.


## 2026-10-24 관문 회전 시 바닥·절벽 논리 좌표 정렬

- `docs/S_checkpoint.html`: 바닥 조각의 위 꼭짓점 대신 조각 중심을 소품과 같은 논리 좌표계에서 회전하고 다시 꼭짓점으로 변환 — 90°·180°·270°에서 바닥과 절벽이 대각선 격자만큼 어긋나던 문제를 해결하기 위해.
- `docs/scene-tile-faces.md`: 바닥 조각 중심을 기준으로 하는 회전 공식을 기록 — 각 회전 방향에서 바닥 2×2조각과 면 소품의 중심을 일치시키기 위해.
- `docs/history.md`: 회전 좌표 수정 이력 기록 — 관문 뷰어의 기준 좌표 변경을 추적하기 위해.

## 2026-10-24 관문 절벽 겹그리기 비교 스위치

- `docs/S_checkpoint.html`: 높은 절벽 그림을 현재처럼 두 번 그리거나 원래 위치에 한 번만 그리는 버튼을 추가하고 커서 하이라이트도 선택에 맞춰 변경 — 모서리의 중복 천장·벽면 원인을 화면에서 비교하기 위해.
- `docs/history.md`: 비교 스위치 변경 이력을 기록 — 관문 뷰어의 진단 기능을 추적하기 위해.

## 2026-10-24 관문 바닥 기준 포인팅과 절벽 윗면 높이

- `docs/S_checkpoint.html`: 커서 칸을 천장·벽 높이와 무관하게 원래 바닥 평면으로 역투영하고 벽 픽셀은 별도 표시; 절벽 그림의 윗부분을 이어 그려 146px 천장과 맞춤 — 높은 천장 때문에 아래 칸을 가리킬 수 없던 문제와 윗면의 높이 차이를 해결하기 위해.
- `internal/gui/field.go`: 관문 고지대 천장을 146px 올리고 높은 절벽의 윗부분을 8px 이어 그리되 밑동은 유지 — 게임 화면에서도 천장·절벽 높이와 접지를 맞추기 위해.
- `docs/scene-tile-faces.md`: 바닥 기준 포인팅과 천장·절벽의 픽셀 높이 및 이음 방식을 기록 — 좌표와 높이의 의미를 분명히 하기 위해.
- `docs/history.md`: 변경 파일과 이유를 기록 — 관문 렌더링 수정 이력을 남기기 위해.

## 2026-10-24 관문 벽면 위 커서 선택 복구

- `docs/S_checkpoint.html`: 바닥·천장뿐 아니라 실제 그려진 절벽·성벽·소품의 불투명 픽셀을 커서로 선택하고 해당 칸의 면과 천장을 함께 하이라이트; 파일에서 직접 열어도 동작하도록 그림의 작은 불투명 비트 마스크를 문서에 포함 — 벽 그림 위에서 선택이 끊기던 문제를 해결하기 위해.
- `docs/history.md`: 벽면 선택 복구 이유를 기록 — 뷰어의 진단 동작 변경을 추적하기 위해.

## 2026-10-24 관문 바닥 타일 하이라이트

- `docs/S_checkpoint.html`: 그려진 바닥 조각의 순서를 이용해 커서 아래 실제 지면·고지대 천장 칸을 고르고, 해당 칸의 바닥을 소품과 같은 2초 주기로 흰색 펄스 표시 — 높아진 천장의 위치와 바닥·벽 이음을 눈으로 확인하기 위해.
- `docs/history.md`: 바닥 선택과 하이라이트 변경 이유를 기록 — 관문 뷰어의 진단 기능 이력을 남기기 위해.

## 2026-10-24 관문 고지대 천장과 뒤쪽 경계선

- `tools/scene_maps/compose.py`: 절벽을 흙 지형으로 바꾸기 전에 작은 칸별 고지대 여부를 `highland`에 보존 — 벽이 없는 내부 칸에도 천장을 그리기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: 두 문 상태에 같은 고지대 행을 저장 — 천장 판정과 절벽 면 판정을 분리하기 위해.
- `internal/gui/assets.go`·`field.go`: 고지대 천장을 실제 절벽 윗면 높이에 출력하고 같은 화면 깊이의 조각을 미리 묶으며 0°의 NW·NE 노출 면에 얇은 선을 그림 — 낮은 이웃만 벽을 갖게 하면서 매 프레임 그리기 비용을 줄이기 위해.
- `internal/gui/scene_maps_test.go`: 내부 천장·낮은 이웃·닫힌 문 자료를 검사하고 그리기 비용을 측정 — 경계 규칙과 성능을 확인하기 위해.
- `docs/S_checkpoint.html`: 네 회전의 고지대 천장과 뒤쪽 외곽선을 표시하고 배치 순서 설명을 갱신 — 각 방향의 지형을 직접 확인하기 위해.
- `docs/scene-tile-faces.md`·`presentation-scene.md`: 천장과 벽의 독립 규칙 및 높이·정렬 방식을 기록 — 장면 맵 자료 의미를 명확히 하기 위해.
- `docs/history.md`: 변경 파일과 이유를 기록 — 고지대 표현 결정의 경위를 남기기 위해.

## 2026-10-24 관문 고지대의 원본 노출 면과 회전 표시

- `tools/scene_maps/compose.py`: 고지대 네 이웃과 그림 범위 밖을 비교해 원본 NW·NE·SE·SW 면을 기록하고 앞쪽 면의 강제 뒤집기를 제거 — 같은 고지대 사이에 벽이 생기지 않고 90° 왼쪽 및 270° 모서리의 면을 정확히 돌리기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: 두 문 상태의 절벽 마스크를 원본 고지대 이웃 관계로 재생성하고 새 동쪽 절벽 발치 그림자를 저장 — `(-18,n)`과 `(41,15)`를 포함한 네 경계를 같은 규칙으로 정의하기 위해.
- `internal/gui/field.go`·`scene_maps_test.go`: 기본 0° 카메라에서 화면을 등진 절벽 면을 생략하고 경계·모서리 자료를 검사 — 높은 바닥 위에 뒷면 외곽선이 나타나지 않게 하기 위해.
- `docs/S_checkpoint.html`: 내장 맵과 회전별 면·발치 그림자 선택을 갱신하고 반대 방향 그림이 없는 모서리 그림자를 숨기며 실제 바닥 범위를 명시 — 90°·270° 비교와 좌표 확인을 정확하게 하기 위해.
- `docs/scene-tile-faces.md`·`presentation-scene.md`: 고지대와 노출 면의 구분 및 회전 규칙 설명 — 절벽 타일의 의미를 배치·렌더 문서에 반영하기 위해.
- `docs/history.md`: 고지대 면 판정 변경 이력 기록 — 파일별 수정 이유를 남기기 위해.

## 2026-10-24 관문 절벽의 남동쪽·남서쪽 끝과 윗면 경계

- `tools/scene_maps/compose.py`: 맵 동쪽·남쪽 바깥을 열린 공간으로 판정해 `SE`·`SW` 절벽 면을 만들고 면이 없는 내부 절벽 그림을 생략 — 바깥 모서리의 암벽을 채우고 높은 바닥의 불필요한 외곽선을 없애기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: `(41,n)`, `(n,33)`의 절벽 면과 `(41,33)` 두 면 모서리를 저장하고 내부 `tcliff_00_*`을 제거 — 두 문 상태에서 같은 지형 경계를 표시하기 위해.
- `docs/S_checkpoint.html`: 갱신한 맵 데이터와 절벽 끝 설명을 반영하고 회전 시 화면을 등진 외곽 면을 숨김 — 네 방향 뷰어에서 바깥 경계를 선 조각 없이 확인하기 위해.
- `internal/gui/scene_maps_test.go`: 동쪽·남쪽 면, 모서리, 내부 윗면 생략을 검사 — 절벽 끝이 다시 비거나 외곽선이 생기지 않게 하기 위해.
- `docs/scene-tile-faces.md`: 외곽 면과 내부 바닥의 구분 기록 — 작은 칸 면 배치 규칙을 명확히 하기 위해.
- `docs/history.md`: 절벽 경계 변경 이력 기록 — 파일별 수정 이유를 남기기 위해.

## 2026-10-24 관문 앞쪽 절벽의 장면 높이 복원

- `tools/scene_maps/compose.py`: 앞쪽 절벽을 높은 그림으로 바꾸고 원래 바닥 접점과 `NE` 면 소유를 유지 — 관문 앞 절벽을 뒤쪽과 같은 장면 높이로 표시하기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: 두 문 상태의 앞쪽 절벽 이름과 피벗 보정을 높은 그림에 맞게 갱신 — 게임에서도 절벽이 관문 앞을 실제 높이로 가리게 하기 위해.
- `docs/S_checkpoint.html`: 내장 맵과 회전별 절벽 그림 선택을 갱신하고 자연스러운 가림을 유지 — 네 각도에서 같은 높이의 절벽을 검토하기 위해.
- `internal/gui/scene_maps_test.go`: 앞쪽 절벽의 그림·소유 면·바닥 보정을 확인 — 낮은 절벽으로 되돌아가지 않게 하기 위해.
- `docs/scene-tile-faces.md`·`presentation-scene.md`: 앞뒤 절벽 높이와 가림 규칙 설명 수정 — 자료 형식과 장면 문서를 현재 배치에 맞추기 위해.
- `docs/history.md`: 절벽 높이 변경 이력 기록 — 파일별 수정 이유를 남기기 위해.

## 2026-10-24 관문 회전 화면의 절벽 접점과 성문 끝 정리

- `docs/S_checkpoint.html`: 낮은 절벽의 늦은 그리기 보정을 회전 후 화면 좌표에 적용하고 180°·270°에서는 앞뒤 절벽의 높이를 화면 방향에 맞춰 바꾸며 반대편 그림이 없는 성문 끝 조각을 숨김 — `(17,15)` 주변 절벽이 두 칸 아래로 밀리고 관문 오른쪽이 깨져 보이던 문제를 바로잡기 위해.
- `docs/history.md`: 회전 화면 수정 내역 기록 — 파일별 수정 이유를 남기기 위해.

## 2026-10-24 관문 절벽·성벽·그림자의 작은 칸 면 배치

- `docs/scene-tile-faces.md`: 작은 바닥 칸의 SW·SE·NW·NE 면 좌표, 면별 그림 소유 관계와 관문 확인 기준 기록 — 구조물과 그림자의 위치를 바닥 칸에 명시적으로 연결하기 위해.
- `tools/scene_maps/compose.py`: 절벽의 열린 면·성벽·성문·접촉 그림자·삼각형 그림자를 `small_faces`로 내보냄 — 구조물과 그림자의 소속 칸·면을 명시하기 위해.
- `tools/scene_maps/tall_cliff.py`·`assets_candidate/tiles/tcliff_*.png`: 높은 절벽 생성에서 잘리던 밑단 10px를 복구하고 기준점은 유지 — 절벽 아래 타일의 이음새를 온전히 그리기 위해.
- `assets/graphics/map/extra.png`: 복구한 높은 절벽 그림만 기존 아틀라스 칸에 다시 채움 — 게임 런타임과 HTML에 같은 절벽 그림을 사용하기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: 두 문 상태에 200개의 면 배치 항목을 저장하고 삼각형 그림자를 `(18~22,1)`의 SW 면에 귀속 — 게임과 뷰어가 같은 면 자료를 읽게 하기 위해.
- `internal/gui/assets.go`·`field.go`·`scene_maps_test.go`: 면 항목을 읽어 칸의 면 중심에서 렌더하고 관문·절벽·그림자 소유 칸을 확인 — 런타임에 면 배치 규칙을 적용하기 위해.
- `docs/S_checkpoint.html`: 면 배치를 읽고 반올림·회전·마우스 강조에서 소속 면을 표시 — 관문 이음새를 작은 칸과 면별로 살펴보기 위해.
- `docs/scene-small-grid-trial.md`·`presentation-scene.md`: `small_faces` 자료 형식과 관문 적용 범위 설명 — 장면 맵 문서를 실제 배치 방식에 맞추기 위해.
- `docs/history.md`: 작은 칸 면 배치 변경 이력 기록 — 파일별 수정 이유를 남기기 위해.

## 2026-10-24 관문 해설 화면의 절벽 높이·좌표·회전 비교

- `docs/S_checkpoint.html`: 절벽 그림을 픽셀 단위로 올리는 슬라이더, 커서의 원본 작은 칸·논리 좌표 표시, 0°·90°·180°·270° 회전과 방향별 바닥 경계·절벽·성벽·성문 그림 전환 추가 — 절벽 높이와 사방 배치를 한 화면에서 검토하기 위해.
- `docs/history.md`: 관문 해설 화면 조작 기능의 변경 이유 기록 — 문서 변경 이력을 남기기 위해.

## 2026-10-24 관문 성벽의 바닥 정렬과 남서쪽 끝 조정

- `tools/scene_maps/compose.py`: 성벽·성문 중심을 작은 바닥 칸에 맞게 8px 올리고 앞쪽 언덕 턱에서 성벽 줄을 끝내며 작은 격자 보정 기준을 바닥 칸 중심으로 바꿈 — 관문과 성벽의 이음 및 정수 배치 기준을 바로잡기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: 두 문 상태의 성벽·성문·모서리 그림자 기준점을 옮기고 언덕까지 이어지던 성벽 20조각을 제거하며 기존 소품의 화면 위치를 새 보정값으로 보존 — 실제 장면 배치를 생성 설정에 맞추기 위해.
- `internal/gui/assets.go`: `small_props` 정수 칸을 바닥 작은 칸 중심에서 읽도록 변경 — 반올림 배치가 바닥에서 반 칸 아래로 밀리지 않게 하기 위해.
- `internal/gui/scene_maps_test.go`: 두 문 상태의 새 위치와 성벽 끝 범위를 확인 — 관문 배치가 다시 어긋나지 않게 하기 위해.
- `docs/S_checkpoint.html`: 바뀐 맵 데이터를 다시 넣고 바닥 칸 중심 기준의 반올림 배치 및 관문 위치 설명을 갱신 — 전체 맵 뷰어가 게임 배치와 일치하게 하기 위해.
- `docs/scene-small-grid-trial.md`: 작은 격자 중심의 변환식을 바닥 칸 중심으로 수정 — 좌표 자료 형식을 정확히 설명하기 위해.
- `docs/presentation-scene.md`: 관문 좌표와 언덕 앞 성벽 끝을 갱신 — 장면 배치 규칙을 현재 맵에 맞추기 위해.
- `docs/history.md`: 관문 정렬 변경 기록 — 수정 이유를 남기기 위해.

## 2026-10-24 관문 장면의 작은 배치 격자 시도

- `docs/scene-small-grid-trial.md`: 관문 두 맵의 정수 작은 칸·픽셀 보정 형식과 확인 기준 기록 — 논리 칸의 2×2 격자를 소품 배치에도 적용하기 위해.
- `tools/scene_maps/compose.py`: 관문 소품을 `small_props`로 변환해 내보내는 단계 추가 — 절벽·성문 조각을 정수 작은 칸에 배치하고 기존 픽셀 위치를 보존하기 위해.
- `assets/graphics/maps/S_checkpoint.json`·`S_checkpoint_closed.json`: 2,456개 소품을 작은 칸 정수 좌표와 픽셀 보정으로 저장 — 관문 두 상태에서 새 배치 형식을 시험하기 위해.
- `internal/gui/assets.go`·`scene_maps_test.go`: `small_props`를 기존 소품 좌표로 읽고 관문의 절벽·성문·일반 소품 위치를 확인 — 새 형식과 기존 장면 맵을 함께 읽기 위해.
- `docs/S_checkpoint.html`: 작은 칸 위치의 소품을 표시하고 인물의 2×2 점유 범위를 마우스 강조에 표시 — 배치 격자를 전체 맵에서 살펴보기 위해.
- `docs/S_checkpoint.html`: 반올림 O에서도 작은 칸 `(U,V)`를 유지하고 픽셀 보정만 제거 — 이웃 절벽 조각이 큰 칸 정수화 때문에 같은 위치로 합쳐지던 비교 화면을 바로잡기 위해.
- `docs/presentation-scene.md`: 관문 맵의 `small_props` 형식과 시도 범위 설명 — 장면 맵의 현재 자료 형식을 명확히 하기 위해.
- `docs/history.md`: 작은 배치 격자 시도 변경 이력 기록 — 파일별 변경 이유를 남기기 위해.

## 2026-10-24 관문 맵 렌더링 해설

- `docs/S_checkpoint.html`: `S_checkpoint.json`의 바닥·장식·소품을 게임 타일과 소품 그림으로 단계별 재구성하고 좌표·깊이 정렬 방식을 설명 — 관문 맵의 완성 모습과 그리는 과정을 한 문서에서 확인하기 위해.
- `docs/S_checkpoint.html`: 망루·게시판의 원래 소수 좌표와 정수 칸 반올림 배치를 같은 배경에서 확대 비교 — 좌표를 정수로 바꿀 때 소품이 실제로 얼마나 이동하는지 보여 주기 위해.
- `docs/S_checkpoint.html`: 부분 확대 비교를 스크롤 가능한 전체 맵 뷰어로 바꾸고 ×1·×2·×4·×8 배율, 전체 소품 좌표 반올림 전환, 마우스를 올린 칸의 소품을 2초 주기 사인 곡선으로 하얗게 강조하는 기능 추가 — 관문 전체의 이음새와 칸별 배치를 직접 탐색하기 위해.
- `docs/history.md`: 관문 맵 해설 HTML 추가 기록 — 문서 변경 이력을 남기기 위해.

## 2026-10-24 마을 걷기의 클릭 고리를 도트 시트로, 바닥에

결정 · 누른 타일에 퍼지는 고리를 화면 위 벡터 선 대신 알파 있는 도트 시트(`click_ring`, 금빛 고리 한 겹)로 그리고, 맵 바닥 층에 깐다 · 이유: 사용자 제안("알파값 있는 스프라이트로 애니메이션"), 금빛 한 겹·바닥으로 정함. 효과는 픽셀 시트, 벡터는 맵 칸 도형만이라는 기준과도 맞다 · 영향받는 문서: presentation-town.md, ver5-fx README
- `../tools/spritetool/assets/ver5-fx/seed.py`·`settings.json`·`pixels.json`: `click_ring` 48×26, 8프레임, 감속 퍼짐, 프레임별 알파 1 → 0.18, 밝은 금 `#ffd166` 위 짙은 금 `#a8741e` 테두리
- `assets/graphics/effects/fx.*`: fx 시트만 다시 구움(`build_effects.py`의 다른 시트는 건드리지 않음)
- `internal/gui/field.go`: `groundMarks` — 그림자 위·서 있는 것 아래에 까는 그림
- `internal/gui/town.go`: 벡터 고리(`strokeEllipse`)를 지우고 `click_ring` 프레임을 누른 타일 가운데 바닥에

## 2026-10-24 장수 초상 화풍 시안 롤백

- `tools/illustrations/portraits/painted/*`: 유화풍 장수 초상 후보와 프롬프트를 제거 — 기존 수묵 담채 초상을 유지하기 위해
- `tools/illustrations/portraits/sheets/{watercolor-six-rough,watercolor-six-broad-brush,crayon-six}*`: 수채·과슈·크레파스 6인 시트와 프롬프트를 제거 — 화풍 시안을 롤백하기 위해
- `ver5/docs/history.md`: 시안 추가 기록을 롤백 기록으로 정리 — 삭제된 파일을 현행 제작물로 오해하지 않도록

## 2026-10-24 소품 그림자 끄기 속성, 관문 앞쪽 기둥과 앞 절벽 잇기

결정 · 장면 소품 밑에 엔진이 까는 타원 땅 그림자는 이름 접두어 대신 index.json(`extra`) 항목의 `"shadow": false`로 끈다. 관문 앞 절벽은 반 칸 늦게 그려지는 같은 절벽 그림으로 깔아 성문 앞쪽 기둥 밑동을 덮게 한다 · 이유: 사용자 "삼각형 그림자와 밑동 띠 밑에 동그란 그림자가 또 깔린다", "성문 앞쪽 끝과 앞쪽 절벽이 끊어져 있다". 앞 절벽을 v≥8로 물린 뒤로 기둥과 절벽 사이에 땅이 보였고, v≥7.5에 두면 기둥 조각이 절벽 턱보다 늦게 그려져 턱 위로 삐져나왔다 · 영향받는 문서: presentation-scene.md
- `internal/gui/assets.go`: `propEntry.Shadow`, `Prop.NoShadow`; `field.go`는 `tcliff_` 접두어 예외 대신 `NoShadow`를 봄; `scene.go`의 비트 소품 그림자도 같음
- `tools/build_extra.py`: tiles.json의 `shadow: false`를 index.json에 옮김; `tall_cliff.py`·`join_overlay.py`의 그림에 `shadow: false`
- `tools/scene_maps/tall_cliff.py`: `lcliff_<mask>_<variant>`(원래 높이, 기준점 16px 아래); `compose.tall_cliffs(..., kind='lcliff')`는 기준점을 반 칸 뒤로 옮겨 놓아 그림은 제자리, 그리기 순서만 반 칸 늦음. `S_checkpoint`의 앞 절벽(v≥7.5) 전체에 적용

## 2026-10-24 모서리 그림자를 삼각형으로, 절벽 밑동 그림자, 관문 양쪽을 절벽에 붙임

결정 · 해를 서북서에 두고, 안쪽 모서리에서 왼쪽 면이 오른쪽 면에 드리우는 그림자를 또렷한 직선 경계의 반투명 삼각형으로 그린다. 절벽이 바닥과 만나는 곳에는 밑동 그림자 띠를 깐다. 관문의 양 끝이 절벽에 닿게 성문과 길을 반 칸 옮긴다 · 이유: 사용자 "디더링하는 식이 아니라 알파값을 가진 걸로", "그림자가 절벽에 삼각형처럼", "관문 왼쪽이 절벽과 떨어져 있어", "절벽과 바닥 사이 그림자가 없으니 좀 어색해". 엔진(패킹·로더·그리기)은 부분 알파를 그대로 살린다. 점무늬는 앞 판 덧그림이 밑동 기울기를 열마다 반올림하고 알파를 열마다 바꿔 생긴 것이었다 · 영향받는 문서: presentation-scene.md, tools-build.md
- `tools/scene_maps/join_overlay.py`: 다시 씀. `jshade_<높이>_<j>`(삼각형, 반 칸 조각), `cfoot_sw`·`cfoot_se`(바닥 층). 앞 판의 `join_*` 80장은 지움
- `tools/scene_maps/compose.py`: `corner_shadow`, `wall_corners`(앞 판 `join_overlays` 대신), `cliff_feet`. `S_checkpoint`는 성문 (8.5, 4.5), 길 v≈4~5, 성벽 v 1.0~7.25, 앞 절벽 v≥8. 앞 절벽을 v≥7.5에 두면 성문 앞쪽 기둥의 왼쪽 조각이 절벽 턱 위로 삐져 나왔다. `S_castle_town`·`_burnt` 뒤쪽 성벽 모서리에 모서리 그림자를 넣음

## 2026-10-24 이음 덧그림: 구조물이 높은 면에 붙는 자리의 외곽선과 그림자

결정 · 구조물과 절벽이 붙는 세로 이음선에는 알파만으로 된 덧그림을 얹어 외곽선과 그림자를 준다. 그림 색이 없으므로 어떤 구조물·면 조합에도 쓸 수 있다. 외곽선은 픽셀로 찍고, 부드러운 알파는 파이썬으로 계산한다 · 이유: 사용자 "성벽이랑 절벽이랑 이어지는 부분에 외곽선을 주고 그림자도 있었으면 해. 다만 이게 타일로써 범용적으로 쓸 수 있게끔 알파값이었으면 좋겠어" · 영향받는 문서: presentation-scene.md, tools-build.md
- `tools/scene_maps/join_overlay.py`(새): 이음선 1px 외곽선, 오른쪽 면 그림자(16px 폭, 이음에서·위로 갈수록 옅음, 면 축의 밑동 기울기를 따름), 맨 아래 단의 접촉 그림자. 면 축 조합 uu·uv·vu·vv × 8px 단 20개 = 80장
- `tools/scene_maps/compose.py`: `join_overlays` — 성벽·성문 줄의 뒤쪽 끝 너머 반 칸이 절벽이면, 끝 조각의 가장자리 열(그림의 알파에서 밑동·꼭대기를 잼)에 끝 조각 높이만큼 단을 쌓음. `S_checkpoint`(+닫힌 판)에 적용
- 시험 합성(맵은 바꾸지 않음): u 방향 성벽 끝이 왼쪽 절벽에 닿는 경우, 집이 높은 절벽 앞에 선 경우

## 2026-10-24 관문 양옆 절벽: 전투 맵 절벽을 높게 쌓아서

결정 · 관문 너머 절벽은 전투 맵 절벽 칸의 그림을 그대로 쓰고 바위 면 행만 열두 단 더 쌓아 관문 지붕보다 조금 높게 한다. 길 이쪽 절벽은 한 단 그대로 둔다 · 이유: 처음에 3D 블록아웃을 참고로 codex가 그린 절벽을 도트로 바꿔 썼는데, 사용자가 "안 어울린다"며 기존 절벽을 수직으로 더 쌓으라고 했다(그 그림과 도구 `gen_3d_cliff.py`·`pixelize_cliff.py`·`sources/cliffs.json`은 지웠다). 3D 렌더만 쓴 절벽은 가는 띠 무늬와 밋밋한 옆면이 어설퍼서 처음부터 쓰지 않았다 · 영향받는 문서: presentation-scene.md, tools-build.md
- `tools/scene_maps/tall_cliff.py`(새): 48개 블록(16 모양 × 3 변형) → `tcliff_<mask>_<variant>`, 바위 면 행 21~31을 단마다 다음 변형에서 가져와 쌓음
- `tools/scene_maps/compose.py`: `tall_cliffs`(엔진의 blockSprite·blockVariant와 같은 규칙으로 모양·변형을 고름, 그림 밖은 절벽이 이어진다고 봄); `S_checkpoint`는 v<1 절벽을 높이고 여백을 9칸으로(절벽 끝이 땅 모서리에서 세로로 잘려 보이던 것이 화면 밖으로 감)
- `internal/gui/field.go`: `tcliff_` 소품에는 타원 땅 그림자를 깔지 않음
- 같은 높은 절벽이 어울릴 맵(바꾸지 않음): `S_pass`(거록 협곡)

## 2026-10-23 마을 걷기를 아이소메트리 타일 이동으로

결정 · 마을 걷기를 타일 단위로 바꾼다. 지역은 게임 칸 격자이고 타일마다 걸을 수 있는지 정해지며(소타일·소품 발자국·사람), 유비는 A*로 찾은 길을 따라 타일 중심에서 타일 중심으로 우하·좌상·좌하·우상 네 방향으로만 한 칸씩 걷는다 · 이유: 사용자가 "맵은 타일 위에 배치되고, 그 타일로 이동 가능/불가가 나뉘고, A*로 길을 찾아 네 방향으로 움직이게"를 요청함 · 영향받는 문서: presentation-town.md
- `internal/gui/town.go`: ¼칸 격자·직선 줄이기를 지우고 `tile`·`tileMap`(타일 판정)·A*(네 이웃, 맨해튼 길잡이)·`nearestReachable`로; 사람·표지는 맞닿은 네 타일로 감; 커서·누른 타일·`F3`을 타일 마름모로 그림
- `internal/gui/scene.go`·`player.go`: 앞서 넣은 `even`·`walkState.dur`(고른 걸음)를 되돌림 — 타일 걸음은 한 칸씩이라 장면 걸음 그대로
- `tools/scene_maps/zhuo.py`: 거리의 건물·소품을 타일 중심으로 다시 놓고, 저택 문과 뒷길을 막던 관사 등롱을 빼고, 우물 모퉁이를 넓힘
- `assets/towns/zhuo.json`: 시작·사람·표지·문 자리를 타일 위로, 노인은 큰길 동쪽(7~10)을 오감
- 타일 보기는 `F3` 대신 `R` 키·오른쪽 클릭(사용자 요청). 누른 자리 표시는 마름모 대신 처음의 퍼지는 고리를 누른 타일 가운데에(사용자가 고리가 좋았다고 함)
- `internal/gui/town_test.go`: 타일로 닿는지, `TestTilePath`(네 방향 걸음, 막힌 목표, 가장 가까운 타일)

## 2026-10-23 마을 걷기 데모: 탁군

결정 · 이야기 밖에서 유비가 탁군을 걸어 다니는 데모를 장면 화면(`scene.go`) 위에 얹는다. 지역 하나가 비트 없는 장면 하나, 사람은 배역, 유비는 경로를 걷는 배우 하나다. 클릭한 땅으로 걷고, 사람을 누르면 그 앞까지 가서 대화창으로 이야기하고, 건물을 누르면 문 앞까지 가서 그 건물 안으로 들어간다 · 이유: 사용자가 지금 로직·데이터로 RPG처럼 유비가 돌아다니는 탁군(주점·관사·무기점·도구점·저택)을 요청함 · 영향받는 문서: presentation-town.md(새), README.md
- `internal/town/town.go`(새): 마을 파일(지역·사람·표지·문·소품 발자국) 읽기, 문이 이어진 지역에 돌아오는 문이 있는지 검사
- `internal/gui/town.go`(새): 걷기 모드 — ¼칸 격자 최단 길 찾기와 곧은 다리 줄이기, 사람·건물 그림·표지·땅 클릭 판정, 대화(장면과 같은 두루마리), 문 지나기(어두워졌다 밝아짐), 건물 이름판·커서 이름, `F3` 격자 보기
- `internal/gui/game.go`: `town` 필드, `Update`·`drawFrame`이 걷기 모드로 갈림
- `internal/gui/scene.go`: `even` — 이 배우의 걸음은 길이만큼 시간이 걸림(유비가 짧은 걸음에 느려지지 않게). 이야기 장면은 그대로
- `internal/gui/player.go`: `walkState.dur` — 걸음 시간(0이면 `stepTime`)
- `internal/gui/scenario.go`·`dialogue_scroll.go`: `line.portrait` — 인물 데이터가 없는 사람(유언·추정·장세평·소쌍)의 초상 키
- `assets/towns/zhuo.json`(새): 거리·주점·관사·무기점·도구점·저택과 장비·관우·유언·추정·대장장이·상인·장세평·소쌍·백성들의 대사
- `tools/scene_maps/zhuo.py`(새): `S_zhuo`(12×10 성 안 거리)·`S_zhuo_office`·`S_weapon_shop`·`S_tool_shop`·`S_mansion`을 `compose.py`의 `room`·`outdoor`로 만들고 이 다섯 맵만 씀(다른 맵은 건드리지 않음). 문은 거리 쪽 면으로 돌림(`house_*` +v, `house_*_x` +u). 성문 형식이 바뀌면 다시 돌리면 따라감
- `assets/graphics/maps/S_zhuo.json`·`S_zhuo_office.json`·`S_weapon_shop.json`·`S_tool_shop.json`·`S_mansion.json`(새), `assets/graphics/index.json`: 맵 목록에 다섯 맵 추가
- `cmd/_town`(새): 창으로 걷기. `internal/gui/uidemo_town.go`(새): `-ui town -stage "person:장비,door:주점,…"`으로 클릭을 화면 좌표로 흉내 내 캡처
- `internal/gui/town_test.go`(새): 모든 지역에서 각 입구로부터 문·표지·말하는 사람에 닿는지
- 소품 축척: 소품 카메라 한 칸(모델 1)이 게임 칸 반이라 민가 발자국은 u 1.1·v 0.75칸. 장세평·소쌍은 장면 시트가 없어 문관 시트(`official_black_noncombat`·`official_noncombat`)를 빌림

## 2026-10-23 성문을 아치 가운데에서 갈라 반 칸 조각으로: 성벽과 같은 깊이 순서

결정 · 넓은 성문 그림을 성벽 조각과 같은 반 칸 폭 세로 조각으로 자르고, 각 조각이 자기 땅 위 기준점을 가진 채 그 자리의 성벽 조각을 대신한다 · 이유: 사용자 "쿼터뷰이므로 항상 Y좌표상 위쪽(뒤쪽)에 있는 것을 먼저 찍고 아래쪽을 나중에 찍어야 하는데, 아래쪽 성벽을 찍은 뒤 성문을 찍어서 성문이 이상하게 튀어나온 듯이 보인다" — 다섯 칸 폭 그림 한 장이 성문 점 하나로 정렬되니 앞쪽 성벽과 뒤쪽 성벽의 순서를 한꺼번에 맞출 수 없었다. 날개 자리 성벽 걷어내기와 따로 뗀 먼 쪽 기둥(`sgate_post_*`)은 이 문제를 가린 예외였으므로 없앤다 · 영향받는 문서: presentation-scene.md
- `../assets_candidate/tools/gen_3d_castle.py`: 성문 전체(양쪽 날개·기둥 포함)를 한 번 그려 `slices()`로 반 칸 조각 13장(`sgate_<state>_<axis>_<k>`, `gate_offset`)으로 자름; 옛 두 쪽 그림(`_first`·`_second`)과 `sgate_post_*`는 tiles.json과 파일에서 지움
- `tools/scene_maps/build.py`: `seat_gates`는 성벽 줄을 반 칸씩 깔고 성문 조각으로 그 자리를 바꾸기만 함(옛 viewer 맵의 `_first` 표시도 성문 점으로 읽음); 성문 점이 없는 그림 이름 검사에서 `sgate_`를 뺌
- `tools/scene_maps/compose.py`: `gate()`는 성문 점 하나만 적음; 관문의 가까운 절벽을 반 칸 물림(기둥 밑동이 절벽 그림에 덮여 깨지던 것)
- 사용자 추가 지시 "아치형 문의 중앙을 기준으로 좌우를 나눠서 … 좌상 쪽 반을 먼저 찍고, 성문을 지나가는 병력을 찍은 뒤, 우하 쪽 반을 찍어야 한다": 조각 경계를 아치 가운데에 맞춤(`gate_offset`이 0.5의 배수). `seat_gates`는 성벽 줄의 반 칸 격자를 성문에 맞춰 깔아, 성문이 맵이 정한 자리(길·다리 위)에 그대로 서게 함. 열린 문 통로 안을 막던 어두운 판과 바닥판은 앞쪽 반에 들어가 통로를 지나는 사람을 덮었으므로 뺌(이제 성벽이 성문 뒤로 이어지지 않으니 필요 없음)
- 성 안 민가의 성벽 모서리를 두 줄 모두 −0.25부터(−0.75부터면 바깥 모서리 위로 여장이 삐죽 나옴)
- `tools/scene_maps/export.mjs`: viewer가 이름만 남은 옛 성문 그림(`sgate_*_first/second`)을 찾아도 멈추지 않게
- 확인: 옥문 앞 거리와 관문에서 성문 앞·문 안·뒤, 날개·기둥 앞뒤에 엑스트라를 세운 게임 화면(앞은 위에, 뒤는 반투명으로 가려짐), 팽성(u)·관문(v)·성 안 민가(v) 성문을 병사 둘이 걸어 지나가는 프레임

## 2026-10-23 장면 나무를 codex 그림 → 도트 변환으로

결정 · 장면 나무 14종(소나무 2·활엽수 2·버드나무·고목·불탄 나무·덤불 2·매화 2·복숭아 3)을 3D 렌더 대신, 그 렌더를 참고로 codex가 손도트처럼 다시 그린 그림을 도트로 바꾼 것으로 쓴다(아이템 아이콘과 같은 길). 크기·피벗·팔레트는 3D 렌더 그대로라 심는 자리·그림자는 바뀌지 않는다 · 이유: 사용자가 3D 나무의 의도·실루엣·배치는 좋지만 "어설프게 렌더한 3D 모델" 같다고 했고, 5종 시험을 "매우 좋다, 완전 합격"이라 함 · 영향받는 문서: presentation-scene.md 나무, tools-build.md
- `tools/pixelize_tree.py`(새): 마젠타 키 → 나무로 자르기 → 캔버스(1px 테두리 뺀 크기)로 늘림 → Lab 팔레트 스냅 → 6×6 블록 투표 → 외톨이 정리 → 1px 외곽선. `penalty`·`dark`·`bark`/`foot`/`speck`/`inner` 값
- `../assets_candidate/tiles/sources/`(새): 나무별 codex 그림 512px와 `trees.json`(그림·3D 렌더 팔레트·변환 값·codex 요청문)
- `../assets_candidate/tiles/stree_*.png`·`peach_bloom_*.png`: 변환한 도트로 교체(크기·피벗 그대로, tiles.json 손대지 않음)
- `../tools/unit3d/scene_props3d.py`: 인자 없이 돌려도 `trees.json`에 있는 나무는 덮어쓰지 않음, `--3d`는 그 렌더를 `output/scene-props3d/`에 참고용으로 씀
- 요청문: 참고를 "같은 실루엣으로"라고만 하면 codex가 3D 공 덩이·구멍까지 베껴 변환해도 그대로였다. "참고는 실루엣·크기·빛 방향만, 공·구멍 베끼지 말 것, 16비트 손도트처럼"으로 바꾸고서 됨. 버드나무는 가닥을 굵게(4~6px), 소나무는 밝은 잎을 덜(`penalty`), 복숭아·매화는 덩이 사이 어두운 홈(`dark` .12~.18, 복숭아는 그늘색 `penalty` −8)
- 확인: 나무 쓰는 장면 맵 18개를 `_uidemo-vm -ui scenefile`로 찍어 봄, `check_scenes.py` 0건, `go test ./internal/gui -run 'TestScenesData|TestScenarioNodesAreStaged|TestSceneMap'` 통과

## 2026-10-23 넓은 성문: 날개와 기둥이 붙은 성문을 끊지 않은 성벽 위에

결정 · 장면용 성문 그림에 문루 양옆으로 성벽과 같은 날개(한 칸)와 굵은 기둥을 붙이고, 성벽은 성문 자리에서 끊지 않고 깐 뒤 성문을 그 위에 놓는다. 손으로 적던 성벽 틈(gap) 인자는 없앤다 · 이유: 사용자가 `S_castle_town` 성문 옆 빈틈, `S_jail_street` 성문 옆 어긋남, `S_castle_town_burnt` 성벽 끝 끊김을 찾았다. 틈 숫자를 맵마다 손으로 적어 반 칸씩 어긋났고(조각 하나가 반 칸이라 틈이 남거나 성벽 끝면·뒤 바닥이 비침), 성벽은 땅 끝보다 앞에서 끝났다 · 영향받는 문서: presentation-scene.md
- `../assets_candidate/tools/gen_3d_castle.py`: 성문에 `wing`(성벽 몸체·벽돌 줄·처마·여장)과 `post`(굵은 돌기둥) 추가, 열린 문 통로 안쪽을 어둡게 채움, 먼 쪽 기둥은 `sgate_post_u/v` 그림으로 따로; `--write`가 자기 항목만 tiles.json에 고쳐 씀(gen_all.py는 tiles.json 전체를 다시 씀)
- `tools/scene_maps/build.py`: `seat_gates` — 성벽 줄을 반 칸씩 메우고, 성문보다 뒤에 그려질 날개 자리 조각을 걷고, 먼 쪽 기둥을 놓음. viewer 맵(`S_castle_out`·`S_castle_out_open`)과 compose 맵 모두 이 함수를 거침
- `tools/scene_maps/compose.py`: `wall_u`·`wall_v`에서 gap 인자 삭제, 새 맵의 성벽을 여백 끝까지(`TOWN_M`·`JAIL_M`), 관문 성문을 길 가운데(8.5, 4.0)로 옮기고 수비 진지를 성벽 뒤로
- 확인: 성문을 반 칸 안에서 0·0.1·0.25·0.4·0.5칸 어긋나게 놓은 그리기 시험(u·v, 열림·닫힘)에서 이음이 모두 맞음. 성문과 같은 높이의 조각이 동점으로 성문 위에 그려지던 것을 함께 고침

## 2026-10-23 장면 나무를 실제 크기 3D 나무로, 새 맵의 성문을 장면용으로, 성 안 땅 섞기

결정 · 장면 맵의 나무를 모두 장면용 3D 나무(`stree_*`, 사람 키의 2~3.5배)로 다시 심고, 새 맵의 성문·성벽은 `S_castle_out`의 장면용 성벽·성문(`swall_*`, `sgate_*`)으로, 성 안 맵 땅은 포석 길·흙길·황무지·풀밭을 섞는다. `S_vil_peace`·`S_vil_burnt`의 "미니 집"(전투 맵 마을 칸 그림 `hut`, 작은 둥근 초가 묶음)은 장면 민가로 바꾼다 · 이유: 사용자 "나무가 사람에 비해 너무 작아", "전투 맵용 성문", "S_jail_street 땅이 밋밋", "미니 집 크기가 안 맞아" · 영향받는 문서: presentation-scene.md 장면 맵 표·나무·상석과 방향
- `../tools/unit3d/scene_props3d.py`: 소나무 2·활엽수 2·버드나무·고목·불탄 나무·덤불 2·매화 2, 복숭아나무를 두 배 크기로, 포석 조각 2·우물
- `tools/scene_maps/trees.py`(새): 나무 자리 솎기와 배우 자리 지키기(배우 1.2칸 안·앞을 가리는 나무는 덤불로 바꾸거나 뺌)
- `tools/scene_maps/build.py`: viewer의 타일 나무·숲·매화·불탄 나무를 `trees.plant`로, 물가는 버드나무; `OPEN`(연출 전 맵에서 사람이 설 곳); `HOUSES`
- `tools/scene_maps/compose.py`: `wall_u`·`wall_v`·`gate`·`avenue`·`blobs`; `S_castle_town`·`S_jail_street`·`S_checkpoint`의 전투용 문(`gate_u`)·성벽 칸(`c`)을 장면용 성벽·성문으로; 변형 `S_checkpoint_closed`; 나무를 종 이름으로
- 바뀐 기존 맵(나무): `S_army_camp`·`S_turban_camp`·`S_army_camp_hungry`·`S_road`·`S_hill`·`S_castle_out`·`S_castle_out_open`·`S_garden`·`S_bridge`·`S_pass`·`S_vil_peace`·`S_vil_burnt`(+ 집). 배치 검사 `check_scenes.py`는 바꾸기 전과 같이 0건

## 2026-10-23 1부 새 이야기의 장면 맵과 소품

결정 · 1부(도원결의~동탁 추격전) 장면에 없던 곳을 새 장면 맵 8개로 만들고, 소품은 3D로 모델링해 더한다. 탁군성·궁궐·단양·팽성·맹주 막사는 기존 맵(`S_office`·`S_palace`·`S_vil_peace`·`S_castle_out_open`·`S_council`)을 다시 쓴다 · 이유: `story_2026_10_23.md`, `plan/2026_10_23_story_impl.md` 6.3~6.4, 사용자 규칙(소품은 3D 모델 후 리터칭, 재사용 우선) · 영향받는 문서: presentation-scene.md 장면 맵 표·소품, tools-build.md
- `../tools/unit3d/scene_props3d.py`: 함거·물통·물동이·모닥불·방문 게시판·복숭아나무 3종·떨어진 꽃잎·노점 3종·국밥 솥·걸상·계산대·민가 5종; `--review`가 새 소품(옛 그림 없음)과 이름 고르기·`--out`을 받음; 방문의 수배 얼굴은 `RETOUCH` — 바퀴는 disc()가 세로축만 돌아 따로 짬, 함거 창살은 보는 쪽 두 면만(네 면이면 외곽선이 붙어 갇힌 사람이 안 보임)
- `tools/scene_maps/compose.py`: `S_tavern`·`S_inn`(방), `S_peach_grove`·`S_castle_town`·`S_jail_street`·`S_market`·`S_checkpoint`(야외, `margin`으로 땅이 화면 끝까지), 변형 `S_castle_town_burnt`; `room(pillar=)`, `outdoor(margin=)`, `scatter()`; 변형이 같은 실행에서 만든 맵을 바탕으로 쓸 수 있게
- `internal/gui/field_scenery.go`: `floor_*` 소품이 덮은 칸에는 풀·돌 장식을 뿌리지 않음 — 방 맵(`S_council`·`S_tent_lux` 등)의 마루·깔개 위에 풀이 돋던 것
- `internal/gui/field.go`, `scene.go`: 비트의 `Prop`으로 놓은 서 있는 소품에도 땅 그림자(`beatShadows`) — 함거 등
- `internal/gui/uidemo_scenefile.go`: `-ui scenefile -stage <장면 JSON>`이 이야기 노드 없이 장면 파일 하나를 열어(Cast·Ambient) 새 맵을 게임 화면으로 확인

## 2026-10-23 영역 도주 목표와 `fled` 조건, B04 조조 퇴각 부목표

결정 · `march`·`flee`의 목표로 칸 대신 영역(`GoalArea`)을 줄 수 있고, `fled:X` 조건이 맵 밖으로 빠져나간 유닛을 쓰러진 유닛과 구분한다. B04의 조조군은 여포가 나타나면 `march` 대신 `flee`로 동남쪽 가장자리 전체(영역 2)로 빠져나가고, 부목표는 "조조가 동남쪽 끝으로 무사히 빠져나간다"(`fled:caocao`) · 이유: 사용자 "변수 추격전의 조조도 이상해. 여포 나타나면 도망가야 할거 같은데, 여포랑 싸우고있어", "조조가 성공적으로 동쪽에서 맵 바깥으로 퇴각하는걸 부목표로", "동맹군이 후퇴할때 특정 한 타일로 퇴각 … 동남쪽 전반을 길~게 다 퇴각지점으로 … 아니면 대군이 뭉쳐서 가버리거든" — `march`는 가는 길에 칠 수 있으면 치므로 따라잡은 여포 기병과 싸웠고, `flee`로 빠져나가면 HP 0이라 `alive:caocao`가 거짓이 됐다. 제후군 14명은 (27,1) 한 칸, 조조군은 (26,6) 한 칸을 향해 줄을 섰다 · 영향받는 문서: logic-battle-event.md 4·5절, data-scenario-node.md, conceptual_scenario.md 5장
- `internal/content/content.go`: `Spawn.GoalArea`·`Action.GoalArea`, `ValidPolicy`를 `Stage.ValidPolicy(policy, goal, area)`로(칸·영역 중 하나만, 전투 영역 안에 칸이 있어야), `Stage.GoalCells`
- `internal/content/cond.go`, `internal/core/cond.go`: `fled:X` — `Executed["fled-<스테이지>-<유닛>"]`
- `internal/core/engine.go`: `leaveByFlight`가 목표 칸들 중 어디서든 이탈하고 `fled` 기록을 남김; `types.go`·`battle_events.go`: `Unit.GoalArea`를 스폰·`ai` 동작에서 옮김
- `internal/ai/ai.go`: 목표 길 찾기의 씨앗을 `GoalCells` 전부로, 같은 거리면 덜 걷는 칸(−0.01×걸음) — 동점에서 첫 후보로 돌아가던 것
- `assets/scenario/hero/stage04.json`: 영역 2 "동남쪽 끝"(x=27 열 전체), `lubu-appears`의 `lords`·`wei`를 `flee`·`GoalArea: 2`로, 부목표 `generals`를 `fled:caocao`로
- `internal/gui/uidemo_flee.go`: 데모 도주 목표를 영역 2로; `cmd/_balance`: `-show`가 도주 이탈의 칸을 찍음
- 확인: `_balance -stage B04 -show` 제후군·조조군 15명 넘게 (27,2)~(27,8)에 흩어져 이탈, 조조 8라운드 (27,4); `-ui flee -stage B04` 캡처에서 조홍 (27,2)·조인 (27,3)·일반 부대1 (27,4)로 각자 걸어 나감

## 2026-10-23 AI 정책: "없음" 폐지, 아군 사리기 폐지, 사거리 끝에서 치기

결정 · 적과 AI 아군은 `hold`·`react`·`assault`·`march`·`flee` 중 하나를 반드시 갖고, 적의 `assault`를 보고 아군 AI 전체가 사리던 `charging`을 없애며, 다가가는 이동은 칠 수 있는 칸 중 상대와 가장 먼 칸을 고른다 · 이유: 사용자 "stage1 전투에서 손견, 조조군의 움직임이 이상해. 적에게 달려들지 않고 자기 자리에서 밍기적" — 황건 남쪽 병사가 `assault`라 맵 전체의 아군 AI가 "칠 수 없는데 적 사거리 안인 칸 −40"으로 사거리 바로 밖에서 멈췄다(srpg-cli 5턴 동안 조조군이 북동 모서리에서 맴돎). 사리기가 내 정책이 아니라 맵 어딘가의 적 정책으로 켜졌고, 사리기가 우연히 가려 주던 궁병(최소 사거리 2)이 적 바로 옆에 붙어 못 쏘는 문제도 드러났다("궁병이 달라붙어서 공격 못하는 정책은 없어야"). "없음"은 `react`와 거의 같아(보병은 깨는 거리가 5 대 5) 스토리 상황에 맞춰 둘 중 하나로 고르게 했다 · 영향받는 문서: logic-battle-event.md 5.4, data-scenario-node.md
- `internal/ai/ai.go`: `hold()`·`assaulting()`·적 기본 5걸음 대기를 지우고 `approach()`(칠 수 있는 칸은 40 + 가장 가까운 상대와의 거리, 최소 사거리 안쪽 칸 −5, 아니면 걸어서 다가간 칸당 3); 군주의 위험 칸 감점을 10×n에서 50×n으로 — 사격 칸 점수(40+)에 묻히지 않게
- `internal/core/types.go`: `AIPolicyVersion` greedy-11
- `internal/content/validate.go`, `battle_events.go`: 적 스폰·`AI` 아군 스폰·이벤트 `spawn`에 정책 필수
- `internal/gui/editor.go`, `editor_ui.go`: 에디터가 새로 놓는 적은 `react`, 초기화는 정책·목표를 유지
- `assets/scenario/hero/stage01·02·03·04·06·07.json`: B01 황건 진영 `react`(잠든 진영, 개전 이벤트가 바꿈)·조조군·손견군 `assault`, B02 굶주린 손견군 `react`, B03 목순·무안국 `assault`, B04 제후군 `assault`, B06 안량 `react`, B07 문추·국의·선봉 `react`(2라운드에 `assault`)·공손찬군 `react`(다리 북쪽을 지킴)
- `internal/testdata/mini/scenario/mini/stage01·02.json`: 적에 `react`(예전 기본처럼 기다림)
- `internal/ai/policy_test.go`: 궁병이 사거리 끝(3)에서 쏘는 시험, `assault` 아군이 돌격해 오는 적의 사거리에 들어가는 시험 — 예전 코드에서는 각각 거리 1·8
- 밸런스(10시드, 이전 → 이후): rush B01 9→8, B02 10→9, B03 8→8, B05 0→1 / ai B01 10→10, B02 10→10, B03 9→9, B05 1→0. B04 이후 0승과 `TestRushWinsTheFirstStages` 2/5(B03 패배)는 이전부터 같다

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

## 2026-10-25 중국 전통 문양 타일 시안

- `tools/illustrations/ui/tile-patterns/README.md`: 시안 범위, 재생성 방법, 세 이미지의 생성 프롬프트 기록 — 그림의 출처와 제작 의도를 남기려고.
- `tools/illustrations/ui/tile-patterns/ruyi-cloud.png`, `leiwen.png`, `shou.png`: 여의운문·뇌문·수자문 원본 3종 생성 — 먹빛 청록 배경의 전통 문양 후보를 비교하려고.
- `tools/illustrations/ui/tile-patterns/build_tiles.go`: 가장자리 혼합과 수자문 대칭 반복 처리 추가 — 반복 경계에서 문양이 끊기지 않게 하려고.
- `tools/illustrations/ui/tile-patterns/ruyi-cloud-tile.png`, `leiwen-tile.png`, `shou-tile.png`: 반복용 이미지 3종 생성 — HTML과 이후 배경 시안에 바로 쓰려고.
- `tools/illustrations/ui/tile-patterns/preview.html`: 세 문양의 반복 보기, 크기·경계선·원본 비교·PNG 저장 기능 추가 — 넓은 화면에서 이음새와 밀도를 확인하려고.

## 2026-10-25 전통 문양 타일 대비 조정

- `tools/illustrations/ui/tile-patterns/build_tiles.go`: 여의운문·수자문 대비를 원본의 70%, 뇌문을 17%로 낮추고 뇌문의 평균색을 어둡게 조정 — 뇌문이 다른 두 배경보다 튀지 않게 하려고.
- `tools/illustrations/ui/tile-patterns/ruyi-cloud-tile.png`, `leiwen-tile.png`, `shou-tile.png`: 조정한 대비로 반복용 PNG 재생성 — 세 배경을 더 은은하게 보이게 하려고.
- `tools/illustrations/ui/tile-patterns/README.md`: 대비 계수와 최종 명암 편차 기록 — 시안을 다시 만들 때 같은 톤을 유지하려고.

## 2026-10-25 전통 문양 대비 미리보기 조절

- `tools/illustrations/ui/tile-patterns/preview.html`: 문양마다 독립적인 0~150% 대비 슬라이더, 선택값 표시·브라우저 저장, 화면 명암만 바꾸는 필터 추가 — 사용자가 세 문양의 원하는 대비를 직접 정하려고.
- `tools/illustrations/ui/tile-patterns/README.md`: 슬라이더 값은 PNG에 반영하기 전 미리보기 설정임을 기록 — 선택값을 받은 뒤 최종 타일을 다시 만들려고.

## 2026-10-25 전통 문양 대비 확정

- `tools/illustrations/ui/tile-patterns/build_tiles.go`: 여의운문·수자문 대비를 이전 시안의 80%로 낮추고 뇌문은 유지 — 사용자가 고른 명암을 최종 타일에 반영하려고.
- `tools/illustrations/ui/tile-patterns/ruyi-cloud-tile.png`, `shou-tile.png`: 대비를 낮춘 반복용 PNG 재생성 — 선택한 모습을 파일로 저장하려고.
- `tools/illustrations/ui/tile-patterns/preview.html`: 대비 슬라이더 저장 키를 새 기준으로 바꿈 — 이전 80% 선택값이 새 PNG에 중복 적용되지 않게 하려고.
- `tools/illustrations/ui/tile-patterns/README.md`: 확정된 대비 계수와 명암 편차 기록 — 시안의 재생성 기준을 남기려고.

## 2026-10-25 대사·나레이션·중앙 창 시안

- `docs/ui-mockup/story-ui-proposal.md`: 화면별 문양·나레이션 동작·문구 검사의 범위와 구현 전 확인점 기록 — HTML 시안을 게임 변경과 구분해 검토하려고.
- `docs/ui-mockup/img/story-ui-scene.png`: 현재 게임의 장면 화면 캡처 추가 — 시안을 실제 장면 위에서 판단하려고.
- `docs/ui-mockup/story-ui-proposal.html`: 여의운문 대사창, 용무늬 80% 불투명 중앙 나레이션(우하단 그림자·화면 높이 5% 이동과 페이드), 수자문 UI·중앙 창, 문구 길이 통과/거부와 창 자동 크기 예시 추가 — 사용자가 구현 전에 네 화면과 길이 제한을 비교하려고.
- `docs/ui-mockup/README.md`: 새 HTML 시안 링크 추가 — 기존 UI 시안에서 찾을 수 있게 하려고.

## 2026-10-25 중앙 창 색과 나레이션 표기 조정

- `docs/ui-mockup/story-ui-proposal.html`: 작전 개요·출진 확인의 수자문을 어두운 갈색 계열로 보이게 조정하고 나레이션의 '해 설' 표기를 제거 — 창의 색을 기존 UI에 맞추고 나레이션 문장만 보이게 하려고.
- `docs/ui-mockup/story-ui-proposal.md`: 중앙 창의 갈색 문양과 나레이션 표기 제거를 시안 설명에 반영 — 검토 기준을 화면과 일치시키려고.

## 2026-10-25 대사·나레이션·중앙 창 게임 반영

- `assets/graphics/ui/ruyi-cloud-tile.png`, `assets/graphics/ui/shou-tile.png`: 확정한 타일을 런타임 에셋으로 추가 — 대사창과 중앙 창에 문양을 쓰려고.
- `internal/gui/assets.go`: 문양 타일을 읽고 회색 명도로 변환 — 기존 진영색과 갈색 UI 색을 런타임에서 입히려고.
- `internal/gui/dialogue_box.go`: 대사창에 여의운문을 적용하고 글이 길면 높이를 늘리며, 중앙 용무늬 나레이션의 투명도·그림자·중앙 정렬·5% 이동과 페이드를 구현 — 확정한 화면과 글자 잘림 방지를 적용하려고.
- `internal/gui/scenario.go`, `internal/gui/game.go`: 나레이션을 대사창과 분리하고 퇴장 동작이 끝난 뒤 다음 노드로 진행 — 문장이 사라지는 애니메이션을 보이려고.
- `internal/gui/skin.go`, `internal/gui/windows.go`, `internal/gui/explore.go`: 일반 UI 창에 은은한 수자문을, 작전 개요·출진 확인에 진한 갈색 수자문을 입히고 내용에 맞춰 크기를 조정하며 출진 확인 제목을 구별 — UI 바탕 문양과 중앙 창의 글자 넘침을 고치려고.
- `internal/gui/ui_content_validation.go`, `internal/gui/ui_content_validation_test.go`: 실제 글꼴로 시나리오·탐색 문구·전투목표의 표시 가능 여부를 GUI 초기화 때 검사하고 초과 사례를 시험 — 잘리는 문구를 로딩 때 거부하려고.
- `internal/gui/uidemo_storyconfirm.go`: 출진 확인 창 캡처용 데모 추가 — 실제 렌더링을 확인하려고.
- `assets/scenario/hero/AGENTS.md`: 문구 작성 후 길이 검사 실행 규칙 추가 — 시나리오 작성자가 검사를 빠뜨리지 않게 하려고.
- `docs/ui-mockup/story-ui-proposal.md`: 게임 반영 설계와 적용 범위 기록 — 시안에서 구현으로 옮긴 기준을 남기려고.
- `docs/ui-mockup/story-ui-proposal.html`: 게임 반영 상태를 안내 문구에 반영 — 시안의 이전 문구가 현재 상태를 잘못 설명하지 않게 하려고.
- `docs/presentation-scene.md`: 대사창·나레이션 문양과 배치, 문구 검사를 현행 동작으로 갱신 — 작성 문서가 게임 화면과 일치하게 하려고.

## 2026-10-25 3~5장 이야기·무대·전투맵 비교 시안

- `docs/story-ch03-05-options.html`: 영천의 밤 뒤부터 사수관까지 3~5장 이야기 대본·무대와 단양·팽성·사수관 전투맵에 각각 세 가지 선택안을 정리 — 구현 전에 사건 연결, 장면 연출, 전투 동선을 비교해 고르려고.

## 2026-10-25 궁·상·우 선법의 삼국지 음악 여섯 곡

- `tools/music/README.md`: 여섯 테마의 선법·편성·제작 범위·재생성·음색 출처·검증 결과 기록 — 작곡 의도와 음원의 재현 방법을 남기려고.
- `tools/music/compose.py`: 직접 작성한 악구를 각 32마디로 편곡하고 MIDI Type 1·MP3·전체 MIDI ZIP·독립 HTML 생성 — 궁·상·우 세 선법을 여섯 게임 상황에 맞춰 감상하려고.
- `tools/music/player.html`: 여섯 곡 선택, 재생·일시정지·정지·탐색·음량·한 곡 반복·전체 연속 재생·개별 저장을 갖춘 반응형 플레이어 원본 추가 — 음악을 브라우저에서 비교하고 악보를 가져가려고.
- `assets/music/themes/01-village.mid`, `assets/music/themes/01-village.mp3`: 궁조식 「버들 아래 장터」 추가 — 마을과 정겨움을 피리풍 선율과 발현으로 표현하려고.
- `assets/music/themes/02-court.mid`, `assets/music/themes/02-court.mp3`: 궁조식 「높은 처마의 조회」 추가 — 관청과 높은 사람들의 위엄을 느린 관악·종으로 표현하려고.
- `assets/music/themes/03-intrigue.mid`, `assets/music/themes/03-intrigue.mp3`: 우조식 「발 너머의 밀담」 추가 — 음모를 낮은 발현·관악·쉼으로 표현하려고.
- `assets/music/themes/04-battle.mid`, `assets/music/themes/04-battle.mp3`: 상조식 「진군하는 깃발」 추가 — 전투를 군악풍 선율과 큰북으로 표현하려고.
- `assets/music/themes/05-urgency.mid`, `assets/music/themes/05-urgency.mp3`: 우조식 「봉화가 오르는 밤」 추가 — 다급함을 빠른 발현과 엇박·북으로 표현하려고.
- `assets/music/themes/06-victory.mid`, `assets/music/themes/06-victory.mp3`: 궁조식 「돌아오는 깃발」 추가 — 성취와 환희를 밝은 관악·발현 합주로 표현하려고.
- `assets/music/themes/six-themes-midi.zip`: 여섯 MIDI 악보 묶음 추가 — 전체 곡을 한 번에 저장하려고.
- `docs/music-themes.html`: 여섯 MP3와 MIDI·ZIP을 내장한 오프라인 감상 페이지 추가 — HTML 파일 하나만 열어 모든 곡을 듣고 저장하려고.

## 2026-10-25 인물이 바라보는 방향: 전투 목표 쪽 시작, 이벤트 이동의 끝 방향

- `internal/core/facing.go`: 방향이 없는 스폰을 전투 목표 쪽으로 돌리는 `faceObjective` 추가(행군·퇴각은 목표 칸, 아군은 승리 조건의 대상·구역, 적은 패배 조건의 대상·구역, 없으면 가장 가까운 상대; 출진 장수는 출진 칸 가운데에서, 그룹은 무리 가운데에서 본다) — 출진 칸에 방향이 없으면 모두 `SE`로 서서 탁군에서 유관장이 적을 등지고 등장했고, 도망 전투도 있어 "적 쪽"이 아니라 목표 쪽이어야 해서.
- `internal/core/engine.go`, `internal/core/deployment.go`, `internal/core/battle_events.go`: 스테이지 시작·출진 배치·이벤트 `spawn`/`join` 때 `faceObjective`를 부름 — 처음 놓일 때 한 번 정하려고.
- `internal/core/deployment.go`: `DeploymentPlan`이 빈 칸을 먼저 채우고 막힐 때만 앞 사람을 밀어냄 — 늘 첫 칸부터 빼앗아 출진 순서와 칸 순서가 거꾸로 되어 유비·장비 자리가 바뀌었다.
- `internal/core/battle_events.go`: `moveTogether` 추가, `move` 동작과 컷신 한 묶음의 `move`를 함께 출발시켜 동료가 비울 칸을 빈 칸으로 보고, 끝 방향(`Facing`, 없으면 마지막 걸음)을 `Unit.Facing`에 남기고 이벤트·컷신 단계로 보냄 — 앞사람 자리를 목표로 쓰면 옆으로 비켜 서고, 이동 뒤 방향이 길 찾기에 맡겨지고 불러오면 사라졌다.
- `internal/content/content.go`, `internal/content/battle_events.go`: `Toward`(두 칸 사이 대각선 방향, 같으면 x축) 추가, `Move`에 `Facing` 추가, `Move.Facing`·컷신 `move`의 `Facing` 검증 — 이동 끝 방향을 이벤트가 정하게 하려고.
- `internal/core/types.go`: `Event.Facing`(move의 끝 방향) 추가 — 화면이 끝 방향을 받게 하려고.
- `internal/gui/cutscene.go`, `internal/gui/player.go`, `internal/gui/view.go`: 걷기가 끝나면 받은 방향으로 서고, 규칙 상태의 방향이 바뀌면 화면 유닛도 돌아섬 — 엔진이 정한 방향을 화면이 따르게 하려고.
- `assets/scenario/hero/stage01.json`: 시작 컷신에서 관우 (6,6)·유비 (6,7)·장비 (9,7)로 곧장 앞(`NE`)으로 나서고 `Facing: NE` — 옛 끝 위치(밸런스 그대로)에 교차 없이 적을 보며 서게.
- `assets/scenario/hero/stage02.json`: `Allies` 순서를 옛 실제 배정(유비 (31,29), 관우 (29,31), 장비 (30,30))대로 바꿈 — 배정 순서를 고쳐도 영천 컷신 31개가 전제한 자리가 그대로이게.
- `internal/testdata/mini/scenario/mini/stage01.json`: `Allies` 앞 두 칸 순서를 바꿈 — 옛 배정을 전제한 테스트들이 같은 자리를 쓰게.
- `internal/core/facing_test.go`: 시작 방향(승리 대상·도달 구역·적·적힌 방향), 출진 칸 순서, 컷신 이동의 함께 출발과 끝 방향 테스트.
- `internal/gui/uidemo_opening.go`: `opening` 데모 추가(이야기에서 출전한 것처럼 스테이지를 시작해 배치·시작 컷신 재생) — 시작 방향을 게임 화면으로 확인하려고.
- `cmd/_facecheck/main.go`: 스테이지별 시작 위치·방향과 컷신 이동 경로·끝 방향을 찍는 임시 도구.
- `docs/presentation-facing.md`: 방향이 왜 중요한지(탁군 사례), 시작 방향 규칙, 이벤트 이동 뒤 방향, 출진 칸 순서, 점검 순서 문서 추가 — 이벤트·장면에서 방향을 기본값에 맡기지 않게.
- `../AGENTS.md`, `docs/README.md`, `docs/story-writing.md`, `docs/logic-battle-event.md`, `docs/data-scenario-node.md`: 방향 문서로 이어 주고 `move`의 `Facing`·함께 출발·생략 시 목표 쪽 방향을 적음.
- `docs/fullstory.md`: 3장 낙양 대본을 다시 씀(뇌물 거절로 함거 → 낙양 길에서 조조와 통성명 → 조조의 저택에 묵으며 저잣거리·술집 탐험 → 하진·십상시 이야기 하나로 저택 이벤트 → 노식 상서 복직과 대장군부의 단양 명), 옥 앞 국밥집·하진 회상 삭제, 조조·하진 인물 메모 수정 — 정사·연의의 경위와 인물 처지에 맞추려고.
- `../AGENTS.md`: 이야기의 핵심 원칙(매니아와 가볍게 즐기는 사람 모두가 만족할 표현 거리감과 개연성) 추가 — 학술 용어·강의식 설명·근거 없는 인물 비하를 막으려고.
- `docs/story-ch03-05-build.html`: 3~5장 무대·대본·전투맵 비교 페이지(3장은 다시 쓴 대본, 근거와 다른 장 점검 포함).

## 2026-10-27 영천 구도 회전: 남동 유관장·동북 조조·남서 손견, 한 겹 울타리와 세 칸 문

- `tools/mapgen/layouts/hero/map02.txt`, `assets/scenario/hero/maps/map02.json`: 32x32 → 26x30. 장사성이 서북쪽 강에 등을 대고, 황건 진영을 한 겹 울타리가 ㄷ자로 감쌈(동북·남동·남서 가운데에 세 칸 문). 진영 안쪽 폭 9칸 → 4~5칸, 성문은 동북·남서 벽, 남동쪽에 유관장의 숲 — 두 겹 울타리에 유관장이 흩어지고 진영이 넓어 사모·청룡도로 문 하나를 두고 싸우는 느낌이 없었다.
- `assets/scenario/hero/stage02.json`: 유관장 남동 숲, 조조 동북, 손견 남서에서 각자 제 앞 울타리를 태움(구역 14개 → 10개). 관우 왼편·장비 오른편·유비 문 옆 화공, 불에 쫓긴 황건 넷이 남동 문으로 나와 숲의 유관장에게 급습당함. 성 쪽으로 달아나던 안쪽 울타리 병사 넷(si1~si4)은 파재 곁 so5~so8로 바꿈. 황보숭·주준은 동북·남서 성문으로 나와 `assault`. 파재 출격 5라운드 → 3라운드, 목표는 남동 문 밖 — 바깥에 불이 났는데 안쪽 도적이 성으로 달리는 건 어색했고, 파재가 가운데로 오며 조조·손견·수비대가 먼저 쓰러뜨려 유비 일기토가 사라져서(rush 5판 모두 일기토, B05까지 승리).
- `assets/scenario/hero/story02.json`, `docs/fullstory.md`: 언덕을 남동쪽으로, 두 겹 울타리 묘사를 한 겹·세 문으로, 남쪽 울타리 → 남동쪽, 동·서 → 동북·남서, 3라운드 출격 — 새 구도에 맞춤.
- `internal/core/campaign_content_test.go`: `TestChangsheSiegeUnfolds`의 불·성문 칸, 문 밖으로 나간 병사, 수비대 `assault`, 3라운드 출격으로 고침.
- `tools/mapgen/layouts/hero/stage02.txt`: 새 배치로 다시 뽑음.
- docs/fullstory.md: 2장 S#4를 좌풍의 재물 셈 → 이름 없는 '사신'이 군량 수레를 세며 예물을 바라는 장면으로, 장각은 흩어진 것이 아니라 광종성에 갇힌 것으로 고침. 3장 S#1도 사신으로 맞춤. 유비가 아우에게 하는 말을 하게체로, 결의 전 관우에게는 하오체로 바꿈 — 3장 참소("진채만 높이 쌓았다")와 이어지게, 전문 용어(황문·중랑장)를 빼고, 해라체·반말이 섞이던 유비 말투를 하나로
- docs/data-speech-style.md: 유비 말투를 듣는 사람별로 정리(아우 하게체, 남 하오체, 윗사람 합쇼체, 혼잣말만 해라체) — 대본과 전투 대사의 말투가 달라 같은 사람처럼 들리지 않던 문제
- docs/fullstory.md: 장비가 유비를 '형', 관우를 '둘째 형'(결의 전엔 '운장 형')으로 부르게 맞춤. 3장 S#2의 기도위·노 중랑장·황문을 '황보 장군 휘하'·'노식 공'·'사신'으로 — 호칭 표와 대본이 어긋나던 것, 전문 용어 빼기
- `internal/gui/game.go`: 시작 재생(`replayOpening`)에서 이벤트로 등장하는 유닛(`unit-arrival`)을 등장 차례 전까지 화면에서 뺌 — 끝난 상태로 한꺼번에 그려 조조군·손견군이 첫 화면부터 서 있었다(옛 구도는 구석이라 화면 밖이었을 뿐).
- `internal/gui/hud.go`: 미니맵 점을 규칙 상태 대신 필드에 보이는 유닛 위치로 그림 — 재생 중 미니맵이 아직 오지 않은 군대와 컷신 뒤 자리를 미리 보여 줬다.
- docs/data-speech-style.md: 유비는 아우를 자로 부르고 정이 드러날 때만 '아우'라 함, 장비가 결의 전 관우를 '운장 형'이라 함 — 이름 호칭은 낮추는 말이라
- `internal/gui/game.go`, `internal/gui/turn_banner.go`: 전투가 시작되면(시작·마지막 배치) 개전 재생·브리핑·장면이 모두 끝나 조작할 수 있을 때 "제1턴 · 유비군의 차례"를 띄움 — 1턴 표시는 진영 종료 때만 나와 어느 전투도 첫 턴을 알리지 않았다.
- docs/fullstory.md: 3장 S#4 노인 대사를 벼슬값·세금 → 궁궐 고치는 밭세(영제기 185년)로 — 술집 손님 대화와 겹치지 않게
- docs/fullstory.md: 3장 S#5 관우가 노식을 '노식 장군님'으로, 노인 손님은 조조 집안을 낮춰 말하지 못하고 목소리를 낮춰 '조 기도위 나리'로, S#2 조조 소개는 '기도위'로 되돌림 — 관우는 노식의 제자가 아니고, 서민이 권세 있는 집안을 함부로 부를 수 없으며, 기도위·북부위 정도의 관직명은 매니아에게 익숙하고 처음 보는 사람도 직책으로 읽음
- ../AGENTS.md: 표현 원칙에 '연의 독자에게 익숙한 관직명(기도위·북부위·태수·자사)은 둔다'를 더함 — 전문 용어를 빼다가 관직명까지 지우지 않게
- docs/fullstory.md: 3장 S#5 손님들이 조조를 그냥 '조조'로 부르고, 노인 손님도 '조조 말이오? …였다더군'처럼 들은 소문으로 말하게 함 — 모르는 사람은 유명인을 이름으로 막 부르지만, '그 집 아들'은 아는 사이에서나 쓰는 말이라
- docs/fullstory.md: 3장 S#6 노식 복직을 황보숭의 상주 → 조조가 대장군부 회의에서 하진에게 건의(의병을 단양에 보내고, 그 대가로 노식을 풀어 명성을 얻음)하는 회상으로 바꿈. 제남상·노식의 전언·군보를 빼고 '이번만큼은 작게라도 벼슬'로. 근거와 하진 인물 소개도 맞춤 — 유비가 단양에 가는 이유와 노식이 풀리는 이유를 한 흥정으로 묶고, 하진·조조가 각자 이익으로 움직이게
- docs/fullstory.md: 3장 단양 전투를 '고순 혼자 동남쪽 마을 입구 주둔지에서 버티고, 유비는 남서쪽 고갯길을 뚫고, 도겸은 3턴 북서쪽에서' 치는 구도로. 일정 피해 퇴각·부목표·매복 제거, 두목은 이름 없이, 도겸 대사 교체 — 시간 압박이 있는 구원전으로, 유비에게 아직 무거운 매복은 빼고
- docs/fullstory.md: 단양 전투 입구를 목책-주둔지-목책으로, 고순은 정면 한 명만 상대, 턴 제한 추가 — 에워싸이는 그림이 아니라 입구를 막은 형세
- docs/fullstory.md: 4장 S#1 고순·도겸과 통성명, 도겸이 서주 난을 수습하다 고향을 구하러 왔고 망가진 마을 사람들을 서주로 데려가기로 함, S#2 팽성에서 헤어지려다 장순의 급보. 단양 전투 개요의 도겸도 맞춤 — 도겸의 서주자사 행적과 맞추고, 유비가 팽성에 남는 이유를 길동무로 자연스럽게
- docs/fullstory.md: 팽성 전투에서 협공 이벤트·바깥 고을·일정 피해 퇴각을 빼고 도겸·고순이 함께 싸우고 공손찬이 3턴에 합세하는 구도로 — 사용자가 협공 이벤트를 거절
- docs/fullstory.md: 팽성 전투 부목표를 '달아나는 약탈 수레 되찾기'(보상: 도겸의 군자금)로 — 단양에서 곡식을 털린 일과 잇고 기병 추격과 장순 공격 사이 판단을 만들려고
- docs/fullstory.md: 5장 S#1 공손찬과 유비를 동문 재회로(선주전: 함께 노식에게 배우고 형으로 섬김, 개·말을 좋아한 일화), 팽성 부목표 보상을 보리 4개로 — 처음 만난 사이처럼 쓴 것이 정사와 어긋남, 곡식 수레에서 곡식 보상
- docs/fullstory.md: 5장 S#2와 관우·장비 케미 메모의 장비 호칭을 '형'·'둘째 형'으로 — 호칭 표에 맞춤
- docs/fullstory.md: 5장 S#2 '황제가 붕어하셨다' → '천자께서 돌아가셨대' — 장터 사람 말로
- docs/fullstory.md: 5장 S#3 유비가 조조에게 합쇼체로 맞추고, 노식 소식을 조조가 전하고 유비가 받는 줄을 넣음 — 스승을 위해 움직인 3장과 이어지게
- docs/fullstory.md: 5장 S#3에 동탁 군사의 약탈과 조조가 벼슬을 거절하고 도망친 까닭을, 노식 소식은 '맞서다 벼슬을 잃고 쫓겨났다' 정도로. S#4 관문 대사를 '고향 말 한마디에 마음이 풀린다'로, 갈림길에 조조가 패국에서 군사를 일으키려는 까닭과 유비가 스스로 가는 까닭을 넣음 — 동탁을 치러 가는 동기가 비어 있었고 '수다가 쌉니다'가 가벼웠음
- docs/fullstory.md: 5장 S#5 패국(군사 모으기)·S#6 조조의 막사(가짜 밀조)를 빼고 관문 갈림길에서 서술 한 단락으로 바로 반동탁 연합으로, 뒤 장면 번호와 하후돈 첫 만남 메모를 맞춤 — 패국 장면이 늘어지고 어색해서
- docs/fullstory.md: 5장 가짜 밀조·'형님이 참되시면 됩니다'를 S#3 주막 밤으로 옮기고, S#4 갈림길에서 조조가 유비를 '같이 갑시다'로 꼬드기게 함 — 짚신 장면에서 이어지는 흐름이 자연스럽고, 조조가 유비를 마음에 들어 하는 구도
- docs/fullstory.md: 5장 S#4 조조가 '같이 갑시다'로 꼬드기고 장비가 '패국은 술이 일품'에 넘어감. S#5 동탁 막간을 영걸전 오프닝 오마주 예외로 확정, 헌제 호칭 '태사' → '동상국'(189년의 관직), 정할 것 목록에서 뺌
- docs/fullstory.md: 5장 S#6 공융 인사를 '공자의 후손'으로, 막사 밖 조조·유비 대화를 뺌 — 앞에 없던 상소 이야기가 뜬금없고, 막사 밖 일화는 군더더기
- docs/fullstory.md: 5장 S#6 원소의 중재를 조조의 중재(받기로 한 벼슬이 하진의 죽음으로 흐지부지)로 바꾸고 원술의 코웃음으로 끝냄, 선봉·군량 배정은 S#7로 옮겨 논의 — 조조가 유비를 감싸는 구도
- docs/fullstory.md: 5장 S#7 손견 선봉·원술의 '범에게 날개' 복선, 전령의 급보, 원소가 걱정하는 척 원술을 지목하고 유비의 자원을 맹주의 명으로 바꿈, 조조의 '현덕' 부르기 뺌. 원소 인물 소개에서 '나는 말렸소만' 말버릇을 지우고 행동으로 보이게 — 말버릇 반복이 거슬리고 음흉함이 오히려 사라짐
- docs/fullstory.md: 5장 사수관 전투를 구현된 stage05(손견·정보·황개·한당, 굶주림, 6턴 화웅 출진, 부목표 손견 생존)에 맞추고 화웅은 일기토에서 죽는 것으로. 전투 뒤 온주참화웅(화웅이 다시 싸움을 거는 이야기)은 쓰지 않음 — 사용자 결정
- docs/fullstory.md: 5장 S#8 사수관 들판에서 손견의 감사·고정도(부목표)·원술의 핑계를 되뇌는 한 줄 — 온주참화웅을 빼면서 사라진 보상 장면과 '범에게 날개' 복선의 회수
- docs/fullstory.md: 6장 S#1(손견 감사·고정도, 진무)을 5장 S#8로 합치고 지움 — 같은 장면이 두 번 나오고, 진무는 이때 손견의 병졸일 수 없는 실존 인물. 감사를 병사들에게 돌리는 유비는 살림
- docs/fullstory.md: 5장 S#7 조조의 샛길 군량 대목과 6장 S#2 관우·조조 치하 장면을 지움, 조조 인물 소개에 '관우 한 사람이 아니라 삼형제 모두와 친분' — 전투에 군량이 나오지 않고, 조조와 유관장 전체의 친분이 스토리의 모토
- docs/fullstory.md: 6장 S#3(배탈 난 제후들, 병졸과 밥 나누기)을 지우고 서술 한 줄로 호로관에 이음 — 사용자가 장면을 거절
- docs/fullstory.md: 6장 S#3 맹주의 막사(원소의 안량·문추 탄식, 원술이 의병을 밀고 원소가 '선봉에 서시오'), S#4 연합 진영 탐험(조조의 귀띔, 하후돈·장비 첫 만남, 목순·무안국의 속내, 배앓이 소문), 호로관은 stage06 그대로+장료 — 왜 삼형제만 나가는지와 목순·무안국이 끼어드는 까닭을 이야기로 메움
- docs/fullstory.md: 5장·6장 연합 진영 탐험(도구상, 대사에 나오는 제후만, 6장은 제후마다 제 성격의 핑계), 호로관 NPC를 연의 5회대로 목순(상당)·방열(하내)로 바꾸고 무안국은 공융의 말로만, 목순·방열이 쓰러진 뒤 삼형제 대사를 삼영전으로 이어지게, 7장 원소 대사의 이름 맞춤 — 공융이 비겁하게 보이지 않게, 목순 소속 오류 수정
- docs/fullstory.md: 7장 S#1 원소가 승전의 공을 '맡긴 보람'으로 가져가고 목순·방열의 죽음을 그 주공 탓으로 넘김, 장비가 '뒤를 받친다더니 술상만 받쳤네유' — 말버릇 대신 행동으로, '먼저 내뺀'은 아직 맞지 않음
- docs/fullstory.md: 7장 S#2 전령 대사를 요약에서 대사로
- docs/fullstory.md: 7장 S#3 조조가 추격을 청하는 대사(一戰而天下定)와 웃으며 물러나는 끝을 넣음 — 화내며 박차고 나가면 다음 장면의 꾀와 앞뒤가 어긋남
- docs/fullstory.md: 7장 S#4~S#6을 '막사 밖에서 조조가 출진을 재촉 → 진영 탐험에서 보물 소문을 들음 → 행군 중 흙먼지와 함께 조조가 꾀를 밝힘'으로, 조조의 '보물이 있다고 한 것도 준다고 한 것도 아닌데', 조조 인물 소개의 ★ 말버릇을 지움 — 소문을 먼저 듣고 정체를 나중에 알게 하는 구조, 말버릇 반복 금지
- docs/fullstory.md: 7장 동탁 추격전을 구현된 stage07 근간으로, 복병 없이 후미와 싸우다 3턴에 제후군과 여포가 한 이벤트로 이어져 제후군이 싸우지도 않고 달아나게 함. 조조 쪽 장수 넷을 NPC로, 유비 말투 정리 — 제후들의 비겁함을 더 선명하게, 조조가 약조한 적 없는 소문 꾀와 맞춤
- assets/scenario/hero/story01.json·explore01.json·stage01.json, story02.json·explore02.json·stage02.json: 유비가 아우에게 하게체, 결의 전 관우에게 하오체, 장비가 유비를 '형'·관우를 '운장 형'으로, 관우가 유비에게 합쇼체·노식을 '노식 장군님'으로 — fullstory·data-speech-style의 말투 기준에 맞춤
- assets/scenario/hero/story02.json, assets/scenes/ch02.json(c02-e-luzhi), explore02.json, assets/data/characters_story.json(zuofeng): 2장 노식 진영을 '사신이 군량 수레를 세며 대접을 바라고 노식이 거절, 장각은 광종성에 갇힘'으로 다시 씀. 이름표를 '사신'으로, 비트를 새 줄 순서에 맞춰 옮김(세기 동작 l4, 장비를 달래는 유비 l3, 퇴장 l7), 병사들 대사를 포위 중인 진영으로 — 3장 참소와 이어지게, 재물 자의 처분 삭제
- assets/scenario/hero/story05.json, assets/scenes/ch05.json, assets/scenes/ascii/c05-field.txt(c05-camp.txt 삭제): 사수관 뒤 진영의 온주참화웅을 빼고 들판에서 손견이 감사를 병사들에게 돌리는 유비에게 고정도(부목표)를 주고 '군량 수레가 길이 험해 늦었다지'로 끝냄 — 화웅이 다시 싸움을 거는 이야기가 이상하다는 사용자 결정
- assets/scenario/hero/stage05.json: 화웅 일기토 결과를 퇴각 → 전사(관우의 일격), 유비 대사 하게체 — 같은 결정
- assets/scenario/hero/stage06.json, assets/data/characters.json(fangyue·zhangliao 추가): 호로관에서 무안국 대신 방열(하내)을 목순과 함께 샛길로 내고 둘 다 여포에게 전사, 여포 곁에 장료, 목순·방열이 쓰러진 뒤 삼형제 대사를 '셋이 함께 가세'로, 삼영전 대사의 호칭·말투 — 연의 5회 순서, 공융을 비겁하게 그리지 않음
- assets/scenario/hero/stage07.json(+story07·story08·docs/presentation-facing.md의 이름): '변수 추격전'을 '동탁 추격전'으로, 복병 대신 서영의 후미가 처음부터 길 위, 3턴에 제후군 도착→여포 등장→제후군 즉시 도주→유비·조조가 함께 물러나기로를 한 이벤트로, 원술의 '약조대로 왔소' 삭제 — 어려운 지명을 피하고 제후들의 비겁함을 선명하게
- internal/core/coalition_test.go, internal/duel/coalition_test.go·motion_continuity_test.go·scripts_test.go: 무안국 → 방열(전사), 삼형제 대사 두 묶음, 화웅 전사에 맞춤
- docs/story-writing.md, docs/data-speech-style.md: 노식 예시를 사신 판으로, 군사 전체에게 외치는 명령은 해라체로 둔다는 예외
- docs/story-ch03-05-build.html: fullstory.md의 3~7장을 그대로 읽어 만드는 페이지로 바꾸고, 게임 데이터에 반영한 것·남은 구현을 맨 위에 — 대본과 페이지가 따로 놀지 않게(생성기: 스크래치패드 gen/build2.py)
- docs/fullstory.md, assets/scenario/hero/story02.json(c02-g5~g7): 영천 문 앞에서 손견은 의병장의 이름도 모르고, 조조는 유비가 불 끄러 간 줄 모른 채 진영으로 찾아가겠다고만 함 — 처음 듣는 사람의 버릇과 다음 장면에서야 알 일을 미리 말하던 것
- assets/scenario/hero/story03.json·explore03.json, assets/scenes/ch03.json: 3장을 새 대본으로 — 광종 함거(함거 소품), 낙양 가는 길의 통성명, 낙양 탐험(저택·저잣거리·도구점·술집), 술집 손님 둘의 하진·십상시 대화로 저택 이벤트를 열고, 대장군부 회상(Time memory), 출진 → 단양
- assets/scenario/hero/stage03.json, maps/map03.json, tools/mapgen/layouts/hero/map03.txt·stage03.txt: 단양 전투 — 동남쪽 마을 입구 목책-주둔지-목책에 고순 혼자, 남서 고갯길의 무리, 북동 본대, 3턴 북서에서 도겸, 패배 고순 퇴각·턴 15. 고순은 동맹이지만 직급 commander — 혼자 입구를 막는 전투라 officer면 AI 1/5만 이김(commander로 8/8)
- assets/scenario/hero/story04.json, stage04.json, maps/map04.json, assets/scenes/ch04.json, scenario.json(04장 추가): 4장 — 단양 마을의 통성명·도겸의 제안, 팽성 급보, 팽성 전투(도겸·고순 동맹, 3턴 공손찬, 북쪽으로 달아나는 약탈 수레 셋, 부목표 수레 모두 되찾기 → 5장에서 보리 4)
- assets/data/characters_story.json: gaoshun·taoqian·zhangchun·kongrong·hejin·turban_chief·loot_cart와 대사용 servant·attendant·official·aide·guest_a·guest_b, assets/data/shops.json: 낙양 도구점(luoyang)·종군 상인(camp) — 3~7장 인물과 소모품 상점
- assets/scenario/hero/story05.json·explore05.json, assets/scenes/ch05.json: 5장 앞부분 — 공손찬 재회와 수레 보상, 소문, 주막(짚신·가짜 밀조), 관문과 갈림길, 동탁 막간, 맹주의 막사(공융·원술·조조의 중재·선봉과 군량·급보·원소의 명), 연합 진영 탐험 뒤 사수관
- assets/scenario/hero/story06.json·explore06.json, assets/scenes/ch06.json: 6장 — 원술 투덜거림, 맹주의 막사(안량·문추 탄식, 선봉), 연합 진영 탐험(조조의 귀띔, 하후돈·장비, 제후들의 핑계, 목순·방열), 호로관. 옛 전투 뒤 장면은 7장과 겹쳐 지움
- assets/scenario/hero/story07.json·explore07.json, assets/scenes/ch07.json: 7장 — 원소의 술자리, 막사 밖(장점 찾기·급보), 추격을 거절하는 제후, 조조의 재촉, 보물 소문 탐험, 서쪽으로 가는 길(흙먼지와 조조의 꾀), 동탁 추격전. 전투 뒤 의천검 장면은 그대로 두고 '숲' → '여울가'
- internal/content/stage_level_test.go, internal/core/campaign_content_test.go·coalition_test.go·story_test.go: 전투가 B03·B04 둘 늘어 순번·기준 레벨이 2단계씩 밀린 것, 팽성 부목표와 보리를 확인
- docs/presentation-explore.md: 1부 탐험 표에 3장 낙양과 5~7장 연합 진영
- docs/fullstory.md: 7장 S#6 '동쪽으로 가는 길' → '서쪽으로 가는 길' — 동탁을 쫓아 서쪽으로 간다
- docs/presentation-explore.md, assets/data/characters_story.json(guest_a·guest_b·servant 틀 해제), explore03.json(상인·주인 등 Sprite): 탐험에서 틀 인물과 인물 데이터에 없는 이름은 Sprite가 없으면 보이지 않음 — 낙양 술집 손님들이 화면에 나오지 않던 문제
- assets/scenes/ch06.json·ch07.json, assets/scenes/ascii/c06-start.txt·c07-out.txt·c07-out2.txt: 6장 시작(원술의 투덜거림)과 7장 막사 밖 두 장면(장점 찾기·급보, 조조의 재촉)을 옛 좁은 `S_army_camp`에서 `S_coalition_camp` 맹주 막사 앞 십자 대로로 옮김. 전령은 동쪽 대로에서 달려오고 조조는 대로를 따라 동쪽으로 떠남 — 반동탁 연합 장면에 아직 좁은 진영이 쓰이던 것
- assets/scenario/hero/stage05.json, docs/fullstory.md: 사수관 — 이숙·서량 보병 넷 `hold` → `assault`, 손견·정보·황개·한당 `react` → `assault`. 화웅 출진은 6턴 고정 → 서량병이 모두 쓰러지거나 8턴(`any:[alldefeated:xiliang, round>=8]`), 대사 "이숙, 관문을 지켜라" → "쓸모없는 놈들! 비켜라, 이 화웅이 직접 놈들의 목을 베어 주마!" — 동탁군 절반과 손견군이 제자리에서 움직이지 않아 싸움이 안 되던 것(ai 20판 19승·평균 7.8라운드, rush 17승)
- internal/gui/acquire.go(새 파일)·windows.go·game.go·equipment_window.go, internal/core/explore.go, internal/gui/uidemo_acquire.go·acquire_test.go, internal/core/explore_test.go: 획득 창 — 이야기·탐험 지급과 상점 구매(구매도 `grant` 이벤트를 냄)로 무언가 들어오면 화면 가운데 '획득' 아래 물건마다 패가 한 장씩 차례로 올라옴(처음 얻는 것은 '처음' 표시와 금빛). 장비 패는 특성을, 도구 패는 쓰임을 보여 주고, 누르면 장비·도구 창(설명·보정·특성 단추 / 효과·가진 수), 특성 단추는 장비 창의 특성 설명 창(`chipWindow`로 떼어 냄)을 위에 엶. 창은 바깥 클릭·Esc로 한 겹씩 닫히고 패는 다시 움직이지 않으며, 패는 확인으로만 닫힘. 전투 보상은 결산 창 그대로 — 아이템을 얻어도 화면에 아무것도 뜨지 않던 것(`g.notice`는 그려지지 않았음), 사용자가 고른 시안 C(나란한 패)
- assets/data/items.json, internal/content/content.go(`Item.Icon`), tools/spritetool/assets/ver5-ui(settings·pixels·sources/item_bean·barley·rice·liquor·fine_liquor·baijiu·scroll.png), assets/graphics/ui, internal/gui/events.go(`itemIconOf`)·acquire.go·shop.go·uidemo_acquire.go: 도구 아이콘 7종 — 콩(콩 자루), 보리(보릿단), 쌀(쌀 사발), 술(붉은 천 덮은 술독), 특급주(청자 술병), 빼갈(황토 봉인 술독), 승급서(죽간, 12종 공용). codex 그림→도트 변환. 획득 패·도구 창·상점 줄에 나옴 — 도구가 아이콘 없이 빈 칸으로 나오던 것

## 2026-10-29 전투 맵 울타리·마을·주둔지를 도트 상자로

- ../assets_candidate/tools/gen_pixel_kit.py(battle sites 절), ../assets_candidate/tiles(fence_00..15, village, camp, burnt_village, burnt_camp), assets/graphics/map/extra.png: 성벽·성문과 같은 방식(작은 칸 가로2·세로1 격자 위 상자에 도트 무늬)으로 다시 그림. 울타리는 가로대 두 줄 나무 울타리, 마을은 초가·기와집·짚더미·우물·울짱, 주둔지는 말뚝 목책·지휘 막사·병사 막사·보급 꾸러미·깃대, 불탄 판은 지붕·천막이 없어진 숯 색 — 옛 울타리는 작은 그림을 2배로 키워 성겼고 마을은 노란 네모 넷뿐이던 것
- internal/gui/field_native.go: 울타리를 1배로 그림, 마을·주둔지는 같은 이름의 extra 그림이 있으면 sites.png 대신 그림. internal/gui/field_scenery.go: 깃발 없는 주둔지 대체 그림(`base`) 처리 삭제
- assets/graphics/map/scenery.json, ../assets_candidate/tools/gen_scenery.py: 주둔지 `base`(camp_base) 삭제, 마을 연기를 새 굴뚝 꼭대기(-9,-27)·(20,-24)로
- ../assets_candidate/tools/gen_all.py: 옛 전투 울타리와 sites.png에서 만들던 불탄 마을·주둔지를 뺌(장면 울타리 sfence는 그대로)
- ../assets_candidate/tools/gen_pixel_kit.py(village_0..3, house·ruin), internal/gui/field_native.go(`siteVariants`), assets/graphics/map/scenery.json·../assets_candidate/tools/gen_scenery.py(`village_<n>` 연기): 마을 그림을 네 판으로 — 초가·기와집, ㄱ자 초가와 장독대·텃밭, 부잣집 기와집과 곳간·흙담, 오두막 셋. 칸 좌표 (x+2y) mod 4로 골라 붙은 여덟 칸이 늘 다른 판, 불탄 판은 벽 껍데기만 들쭉날쭉 — 단양처럼 마을이 모인 맵에서 같은 그림이 줄지어 보이던 것
- assets/scenes/ch07.json(c07-out, c07-out2): 막사 밖 두 장면을 S_army_camp에서 연합 진영(S_coalition_camp, 밤) 맹주 막사 앞으로 옮김. 위병 둘이 문을 지키고, 전령은 남쪽 길로 뛰어 들어오며, S#4에서는 삼형제가 막사에서 걸어 나오고 조조는 동쪽 대로로 떠남 — 10/28 "맹주 막사 밖은 연합군 진영, 밤" 결정이 구현 지시에서 빠져 영천 의병 진영 그림이 쓰이던 것
- assets/scenario/hero/explore05_town.json: 주막의 삿갓 쓴 조조 자리를 작은 칸 (2,4)→(3,5)로, c05-inn 장면의 자리와 맞춤 — 말을 걸면 조조가 반 칸 아래로 튀던 것
- internal/gui/scenario.go: 장면이 어두워지는 동안 이미 사라진 마지막 나레이션을 다시 그리지 않음; internal/gui/uidemo_explore.go: 나레이션 끝을 Update와 같은 turnPage로 넘김 — 5장 궁궐 "대신들 가운데…" 나레이션이 사라졌다 다시 뜨던 것
- internal/gui/field.go·field_scene.go·field_render.go(`raisedShadows`, `shadowLayer`, `drawRaisedShadow`): 장면 맵의 높은 땅 위 그림자(사람·덤불·소품)를 따로 모아 두었다가 윗면 칸을 그릴 때 그 칸 마름모만큼 그림 — 맨땅 그림자 층 위에 윗면 조각이 덮여, 8장 연합 진영 위 벼랑의 조조 일행에게 그림자가 없던 것
- assets/scenes/ch08.json(c08-s1): 철군 행렬·공손찬·삼형제가 진영 동·서 문 칸 (38,18)·(1,18)로 드나들고 진영 안은 17행으로 걸음, 군량 수레도 (39,18)로 — 17행 그대로 문 옆 울타리 기둥을 뚫고 지나가던 것

## 2026-10-30 호로관 부목표는 첫 일기토 뒤에 나타나고, 콩주머니는 맞고 버티면 콩을 먹는다

- 결정: 6장 부목표는 처음에 보이지 않고, 목순·방열의 첫 일기토 뒤 유관장 대사("동맹군의 피해가 막심하네…")와 함께 전투 목표 창이 다시 열리며 나타난다. 조건은 "연합군 병사 셋 이상 생존"(여섯 부대 가운데, 상당·하내 구분 없음) · 이유: 사용자 "처음에 그런 부목표가 나오면 이후의 동맹군 병사가 나오는게 예상이 될거아냐", "그냥 연맹군으로 하고, 연맹군이 셋 이상 남아야 한다" · 영향받는 문서: fullstory.md, logic-battle-event.md
- 결정: 콩주머니는 장신구. 공격을 받아 다치고도 버티면 공격 한 번에 한 번 병력을 400 회복한다(가진 콩은 줄지 않는다) · 이유: 사용자 "피격할때마다 알아서 콩을 먹는거야"
- `internal/content`: `goals` 동작에 `Sub`, `Goals.Sub`, `validateSubs`, 조건 `alive:X>=N`, 효과 키 `pouch_heal`. `internal/core`: 바뀐 부목표가 승리 판정·`SubDone`·`SubMarks`·`Observe`를 따름, `pouchHeal`(이벤트 `pouch-heal`). GUI·CLI에 회복 표시
- `assets/scenario/hero/stage06.json`: 연합군 병사 여섯에 `Group: coalition`, `brothers-challenge`에 대사 둘과 `goals`(부목표 `coalition`, 플래그 `flag-hulao-coalition`). `equipment.json`·`traits.json`: `bean_pouch`
- 확인: `go test ./internal/core/ ./internal/cli/ ./internal/session/`, GUI 임시 데모로 일기토 → 대사 → 전투 목표 창 캡처
- 결정: 콩주머니는 7장 첫머리 S#0에서, 호로관에서 돌아온 삼형제가 진영 문으로 들어올 때 살아 돌아온 연합군 병사가 어머니가 콩을 볶아 넣어 준 주머니를 준다. 미달성이면 S#0 없이 막사 안으로 간다. 공융은 주지 않고, 11장 북해는 서주로 가는 길에 황건적에게 공격받는 북해를 본 것뿐이다 · 이유: 사용자 "진영 바깥에서 안쪽으로 걸어오는데, 병사가 나서서 고맙다고 콩주머니 주고 가는건", "북해 구원은 그냥 서주에서 구원 요청이 와서 가는데, 가는길에 북해를 들렀다가 황건적에게 공격받는걸 본거 뿐" · 영향받는 문서: fullstory.md
- 거절: 7장 S#2 막사 밖에 공융·관우·장비를 두고 공융이 주는 안 — 공융이 막사를 들락거리고 조조가 뒤에 나오는 동선이 어색함. 그 전 초안의 공융 대사("안에서는 목순과 방열을 아깝다고만 하더이다")는 다른 인물을 깎아내리는 대사라 거절
- `assets/scenario/hero/story07.json`: 첫머리 `c07-gate-branch`(`flag:flag-hulao-coalition`) → `c07-gate`, `c07-g1`~`c07-g7`(지급 `grant-bean_pouch`). `assets/scenes/ch07.json` `c07-gate`: S_coalition_camp 밤, 화톳불(33,19) 곁 경보병 셋, 삼형제가 동문(38,17)으로 들어와 (30,17)·(31,17)·(31,18), 병사가 (30,18)로 뛰어옴(무릎 꿇기·두드림 지문은 빼기로 함)
- `tools/spritetool/assets/ver5-ui/settings.json`·`pixels.json`·`sources/item_bean_pouch.png`, `assets/graphics/ui/ui.png`·`ui.json`: 콩주머니 아이콘 `item_bean_pouch` — codex 그림(입 벌린 삼베 주머니에 볶은 콩, 붉은 끈) → `pixelize_icon.py`, 콩 사이가 끈의 검붉은색으로 잡혀 콩 그림자색 `v`를 더함. 군량낭(`item_ration`)과 구별되게 입을 벌려 콩을 보임. `equipment.json` `bean_pouch.Icon`
- `assets/scenes/ch07.json` `c07-gate`·`story07.json`: 병사가 넘어진 것처럼 보이던 무릎 꿇기를 빼고 서서 맞음, 유비 "(병사를 일으키며) 일어나게."와 장비 "(병사 어깨를 두드리며)" 지문을 뺌 — 사용자 "병사가 넘어져서 보이고, 일으켜주고 없애주고, (어깨를 두드리며)도 없애죠"

## 2026-10-30 잡병 직급 오합지졸·신병, B01·B02는 오합지졸

- 결정: 직급에 `rabble` 오합지졸(−0.14)과 `recruit` 신병(−0.07)을 더한다. B01·B02 황건 잡병은 오합지졸(L3에서 Lv1, L5에서 Lv3), 다른 판의 "병졸 + Shift −0.07" 잡병은 신병으로 이름만 바꾼다(레벨 그대로) · 이유: 사용자 "잡병을 두고… 오합지졸", "신병 -0.07 오합지졸 -0.14", "1,2탄을 오합지졸, 3,4탄을 신병". 1탄 맵 길을 초원으로 바꾸는 안은 측정(20판 11 → 13승)만 하고 원래대로 둠
- 결정: 영천 유비 대 파재 일기토는 강제하지 않는다(체력 절반에서 일기토 열기, 승리 조건을 일기토로 바꾸기 둘 다 거절). 러시 시험에서 그 요구를 뺌 · 이유: 사용자 "둘다 별로인데"
- 측정: B01 서툰 플레이 40판 16 → 37승(장비 퇴각 33 → 16), 일반 AI 40/40 그대로. 일반 AI의 B01 레벨 상승 관우 +1.8 → +1.7, 유비 +0.9 → +0.7, 장비 +1.7 그대로. 이어서 키우면 서툰 플레이는 절반이 B03에서 멈추고, 게임 AI는 다섯 판 중 넷이 B10까지 깸(stage_lv_balance.md 3.2)
- `assets/data/game.json` `Rules.RankLead`, `assets/scenario/hero/stage01~10.json` 잡병 `Rank`, `internal/ai/rush_test.go`, `docs/stage_lv_balance.md`, `docs/data-scenario-node.md`
- `AGENTS.md`: 체력이 떨어졌다고 일기토를 강제하는 안(체력 절반 이벤트 일기토, 승리 조건을 일기토로 바꿔 그 장수만 끝낼 수 있게 하기) 금지를 적음, 스테이지 직급 목록에 오합지졸·신병, 레벨 식 3 + round(1.5×(n − 1))로 바로잡음 — 사용자 "버그도 많이나고 흐름도 이상하고 도무지 수용할 수 없는 안이야"

## 2026-10-30 장면 맵의 높은 땅은 높이별 층으로 그린다

- 결정: 형식 2 장면 맵을 높이 0의 바닥 → 그림자 → 높이 0에 선 것(절벽 옆면 포함, 정렬) → 높은 땅마다 윗면층(구워 둠) → 그 높이 그림자 → 그 높이에 선 것(정렬) 순서로 그린다. 높은 칸 하나하나를 입체층에서 정렬하지 않는다 · 이유: 8장 「연합 진영 위 벼랑」(S_cliff_coalition)에서 높은 칸 3,840개가 매 프레임 정렬·그리기 대상이고, 그 위 그림자를 칸마다 끼워 그리느라 GPU 명령이 프레임당 530개로 끊겨 버벅임. 사용자 "레이어로 생각했어… Z축 1순위, Y축 2순위", "낮은 땅 & 절벽 옆면, 장식, 그림자, 구조물&인물, 높은땅, 높은땅 장식, 높은땅 그림자, 높은땅 구조물과 인물"에서 옆면만 낮은 땅 정렬로 옮김(뒤에 선 사람을 가려야 해서) · 영향받는 문서: presentation-scene.md
- 거절: 여러 칸 물체(막사·집·실내 벽 등)를 칸 너비 세로 띠로 잘라 단순 y 정렬하는 안 — 정수 배율이 아닐 때(전투 확대 전환 중) 띠 이음매에 빈 픽셀·색 어긋남이 생김(1.37배에서 한 줄 15픽셀 비침, 부드러운 필터는 옅은 줄). 맵마다 수십 개라 기존 `scenemap.Order`로 충분
- `internal/scenemap/plan.go`: `Plan.Levels`(`Level`: `Lift`·`Flat`·`Solid`·`Tops`), `Plan.Pieces`, `blocks`가 윗면을 높이별 층으로·옆면만 조각으로(키 높이는 발치 땅), `Draw.Clip`(옆면 입술). `Build`가 땅마다 따로 정렬
- `internal/gui/field_scene.go`·`field.go`·`field_render.go`·`editor.go`: `sceneLevel`(윗면 그림·그림자 층), `drawSceneLevels`, `castShadow`가 높이별 그림자 층에, `drawItem.level`·`tops`(윗면에 가린 사람의 반투명 표시). `raisedShadows`·`drawRaisedShadow`(칸마다 그림자 조각) 삭제. `cmd/_scene/static/viewer.js`가 층을 그림
- 확인: 8장 벼랑 장면에서 입체 조각 4,141 → 436, 프레임당 GPU 명령 530 → 23, 같은 재생의 CPU 시간 약 절반 이하. 2·4·5·8장 언덕 장면을 바꾸기 전과 픽셀 비교 — 5장 같음, 8장 벼랑 위 화로·덤불 밑단이 앞 칸 윗면에 잘리던 것이 온전히 보임, 4장 윗면 풀 장식이 나무 그림자 아래로 들어감, 2장은 밤 조명 합성 순서 차이(1~2단계 색)뿐

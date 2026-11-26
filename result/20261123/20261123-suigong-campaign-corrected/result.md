# 수공 본편 적용 · 백하 둑 장면맵 · 신야 전투 (2026-11-23)

커밋·푸시·stash·rm -rf 없음. 작업 트리에만 수정했고 기존 미커밋 변경(수공 데모 파일 포함)은 그 위에서 이어 갔다. 다른 수공 작업은 진행 중이 아니었다(result 최근 파일 확인: 조사·검토뿐).

## 대상 확인
- 사용자가 이상하다고 한 것은 하비 회상 맵이 아니라 `assets/scenes/ch25.json`의 `c25-dike`·`c25-dike2`(관우·미축이 둑에서 말하는 장면, 옛 맵 `S_bridge` 12x10)다. 회상 `c25-flash1`·`c25-flash2`는 건드리지 않았다.
- 전투는 회상·수공 뒤의 기존 신야 전투 B25다(`story25.json` `c25-battle` -> `stage25.json`).

## 1. 수공 본편 연결 (설계와 이유)
- 코드를 읽고 정했다: 수공 장면은 전투 필드(`Field.Surge`)가 그리는 것이라 이야기 장면 엔진에는 올릴 수 없고, 구워 둔 물은 한 땅(48x44, 강 u 12~17, 협곡·둑 v 8, 여울 v 30~)에 묶여 있다. 그래서 B25의 땅을 그 땅으로 바꾸고, **B25의 첫 이벤트 `suigong`이 전투 컷신 단계 `surge`로 수공 장면을 전투 화면 위에서 재생**하게 했다. 순서: 백하 둑 장면(`c25-dike`) -> 하비 회상 둘 -> 둑 장면(`c25-dike2`) -> 신야 성 안 화공(`c25-night`) -> 출전 준비 -> **수공 장면(여울 -> 암전 -> 둑 위 관우 -> 둑 터짐 -> 여울에 물 -> 흰 화면)** -> 신야 전투(`routed` 이벤트의 조인·간옹·유비 대사).
- 새 코드: `internal/gui/field_surge_show.go`(데모에서 뽑은 `suigongShow`: 시간표·카메라·행군·발소리·외침·물소리 큐·흰 화면이 걷히며 전투로 복귀), `internal/gui/cutscene.go`(`surge` 묶음 재생), `internal/core/battle_events.go`·`internal/content/battle_events.go`(단계 `surge` 통과·검증, `say`에 `Seconds`), `internal/cli/play.go`(`[컷신] 수공(이름)`), `game.go`·`view.go`(그림 전용 병사 그리기, 시계, 오버레이, 장면 중 말풍선만). 데모(`-ui suigong`)도 같은 `suigongShow`를 쓰고 스테이지의 `suigong` 이벤트 묶음을 읽는다(이름을 코드에 박지 않음. `purity_test.go`에 `boulder` 예외 한 줄).
- 묶음의 의미(문서 `docs/logic-battle-event.md` "수공 장면"): `surge`의 `Unit`=건너는 군세 대장(조홍), `face` 단계의 유닛=둑 위 절벽에 서는 두 사람(관우·미축, 수공 땅의 (19,7)·(20,7)), `say`=말풍선(관우 줄은 명령으로 절벽 장면 끝까지 유지). 여울의 군세(약 80)는 스테이지 병사를 본뜬 그림 전용 `surge#N`이라 규칙 상태에 없고 끝나면 사라진다.
- 이야기 문구는 바꾸지 않았다. 옛 `c25-ford`(S_yanjin) 장면과 노드 `c25-ford`·`c25-e05`·`c25-e06`을 빼고, 그 줄("물이 얕다! 건너라, 건너!" 조홍 / "……터라." 관우)을 그대로 수공 장면의 말풍선으로 옮겼다(데모의 "무너뜨려라!" 대신 이야기의 원래 줄 사용). 사용자 결정 기록(`docs/history.md`의 "둘이 대사 없이 전투로")은 전투 시작 대사에 대한 것이라 유지: 관우·미축의 새 줄은 이야기에 이미 있던 한 줄뿐이다. `docs/fullstory.md` 25장 문단을 맞춰 고쳤다.
- 효과음: 수공 장면의 물소리 큐(`boulder`, `water-big`, `flood-surge`, `flood-crash`, `flood-spray`)와 발소리·발굽 소리가 전투 안에서 그대로 난다.

## 2. 백하 둑 장면맵 `S_baihe_dike`
- 기존 `S_bridge` **12x10 (120칸)** -> 새 **36x30 (1080칸)**: 가로·세로 각 3배(면적 9배). 3배의 뜻이 분명하지 않아 가로세로 3배로 잡았다(가정). 이 맵이 클수록 둑 장면이 수공 땅과 같은 칸을 쓸 수 있어서다.
- 수공 장면과 같은 칸: 강 폭 6칸이 v 방향으로(u 12~17), 협곡 절벽(u 9~11·18~22, v<10, 높이 2), v 8 한 줄이 흙 둑길, 관우(19,7)·미축(20,7)이 오른쪽 절벽 위. 둑 아래쪽 면에 곡식 자루 줄·돌더미, 오른쪽 기슭 뒤에 군막·모닥불·화로·곡식 수레·망루·깃발, 왼쪽 기슭과 하류에 소나무·버드나무.
- 생성: `tools/scene_maps/baihe_dike.py`(`index.json` 등록·`_walkmap` 포함). `ch25.json`의 `c25-dike`·`c25-dike2`는 같은 맵·같은 자리(관우 (19,7), 미축 (20,7), 병사 7명: 둑길 (13,8)(15,8)(16,8), 왼쪽 (9,9), 절벽 (22,9), 기슭 (21,11)(23,11)), 카메라 [18,8]. 회상 전후 같은 장소. `scene_ascii.py whole ch25`로 정수 칸·겹침 확인(이동 없음), `dump`로 ascii 갱신.

## 3. 신야 전투 B25
- 맵: `maps/map25.json`을 수공 땅 48x44로 교체(`tools/mapgen/layouts/hero/map25flood.py`). 동쪽 신야성(성벽·성문·성내, 불·불씨 상태, x 39~46, y 33~39)이 불타고, 성문이 여울을 향한다. 서쪽·동쪽 기슭에 나무·구릉.
- 배치: 유비군 남서쪽 x 2~5, y 33~35(구역 1). 조인·조홍·병사·기병·허저는 서쪽 기슭 x 2~11, y 12~17에서 북서쪽 끝(구역 20: x 0~3, y 0~1)으로 달아난다(옛 배치를 x-8, y+6으로 옮겨 유비군과의 거리를 옛 전투와 같게 유지). 구역 21(허저·기병이 돌아서는 자리)은 x 0~11, y 14~22. 승리·패배·부목표·일기토·이벤트는 그대로.
- 퇴로: 군세가 신야(동)에서 북서(-u)로 여울을 건너다 허리가 끊긴 뒤 서쪽 기슭의 앞부분이 북서쪽 끝으로 달아난다. 강(u 12~17)이 동쪽 경계다.
- 밸런스(`_balance`, 어려움 기준 normal): 옛 B25 AI 5/5 승·7.0라운드 -> 새 B25 AI **5/5 승·6.0라운드**, 어려움(hard) 10/10 승. rush는 옛(0/5)·새(0/5) 모두 같은 패턴(관우 퇴각). 중간에 허저·기병 반응 구역을 바꿔 가며 조정했다.

## 검증
- `go build ./...`, `go vet ./internal/gui ./internal/core ./internal/content ./internal/cli` 통과, 새 파일 gofmt 통과.
- 통과: `./internal/content`(purity 포함), `./internal/cli`, `./internal/session`, `./internal/mapedit`, `./internal/scenemap`, `./internal/replay`, `./internal/ai`, `./internal/autoplay`, gui `-run 'Scene|Story|Cutscene|Suigong|Surge|Explore|Ch25|Battle'`(345초). 수공 테스트(`TestSuigongDemo`·`TestSuigongMarch`)는 새 구조에 맞게 고쳐 통과(행군 속도·발소리 수 일치 등).
- 실패(내 변경 밖으로 보임, 미확인): `./internal/core`의 `TestEveryCampaignDuelStartsWhenItsOfficersMeet/jiangxia-zhaoyun-zhangwu`, `TestNewAt/the_story_before_counts_as_played`(sunqian left), `TestJumpIntoTheOtherTelling/c27v-start`(UnknownWaypoint) — 강하·27장 쪽 다른 미커밋 작업(stage27 삭제 등)과 관련. 전체 gui 테스트는 돌리지 않았다.
- `srpg-cli -slot 9` 플레이: JSON `jump c25-battle` -> 슬롯 9 저장 -> 텍스트 CLI에서 특성 닫기·출전 -> 배치 지도(48x44, 신야성 불, 적 배치), `[컷신] 수공(suigong)`·조홍·관우 줄, 약화 효과와 조인·간옹·유비 대사, 라운드 1 우리 차례 지도까지 확인. 전투 전체 진행은 CLI가 아니라 `_balance` AI로 확인했다(CLI `auto` 입력이 진행되지 않아 몇 수 더 두는 것은 하지 못함).
- GUI 캡처(`caffeinate`+`lockf`, 하나씩): 장면 `scenenode` 둑 두 장면, 전투 오프닝 `suigong-battle`(임시 데모 `uidemo_zz_tmp_suigongbattle.go`: 실제 `deployment_finish`로 이벤트가 재생됨).

## 산출물 (이 폴더)
- `suigong_main_battle.mp4`: 본편 B25 출전 직후의 수공 장면 -> 전투 복귀, 1280x850 60fps 21초, **효과음 포함**(`tools/sfx/dub.py --no-music`). 확인한 것: 오디오 스트림 존재(AAC), mean -28.5dB·max -5.6dB. **귀로 직접 듣지는 못했다.**
- `battle_sheet.png`(여울·조홍 말풍선·관우 명령·흰 화면·전투 복귀 HUD), `battle/frame-*.png`
- `dike/dike_before_line.png`, `dike/dike_miz_line.png`, `dike2/frame-0200.png`: 새 백하 둑 장면(전 맵이 아니라 카메라 [18,8] 화면)

## 남은 한계·선택
- 수공 장면은 출전 준비 화면 **뒤**, 출전을 누른 뒤에 나온다(전투 이벤트라서). 이야기상 수공 -> 출전 준비 순서를 원하면 다른 설계(전투 전 이야기 단계 신설)가 필요하다.
- 수공 뒤 여울 칸은 규칙상 여전히 여울(건널 수 있음)이다. 구운 그림이 여울에서 물이 덮치는 것이라 땅을 깊은 물로 바꾸지 못했다. 동쪽 기슭에 남은 군세는 모델링하지 않았다(전투는 서쪽 기슭의 앞부분만).
- 전투 시작 카메라는 유비군 자리로 돌아가므로 조인군(북)은 처음 화면 밖이다.
- 옛 생성기 `tools/mapgen/layouts/hero/stage25gen.py`는 이미 JSON과 달랐고(관우·미축 위치 등) 이번에도 고치지 않았다 — 실행하면 새 B25를 덮어쓴다. B25는 `map25flood.py`+JSON 직접 편집이 기준이다.
- 맵·장면 구도는 시안 승낙 절차 없이 판단해 만들었다(사용자 확인 필요). `-frames 937`이 데모 영상의 흰 화면 끝이다(`uidemo_suigong.go` 주석).

## 바꾼 파일
- 코드: `ver5/internal/gui/{field_surge_show.go(새), field_surge.go(미커밋 기존), field_surge_test.go, uidemo_suigong.go, uidemo_zz_tmp_suigongbattle.go(새), cutscene.go, game.go, view.go}`, `internal/core/battle_events.go`, `internal/content/{battle_events.go,purity_test.go}`, `internal/cli/play.go`
- 데이터: `assets/scenario/hero/{stage25.json, maps/map25.json, story25.json}`, `assets/scenes/{ch25.json, ascii/c25-dike.txt, ascii/c25-dike2.txt, ascii/c25-ford.txt(삭제)}`, `assets/graphics/{maps/S_baihe_dike.json(새), index.json}`
- 도구·문서: `tools/mapgen/layouts/hero/{map25flood.py(새), map25.txt}`, `tools/scene_maps/baihe_dike.py(새)`, `docs/{fullstory,history,logic-battle-event,presentation-scene}.md`

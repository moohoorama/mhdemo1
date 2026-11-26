# 수공 본편 표시·지형 연결 수정과 전투 진입 확인 (2026-11-23)

커밋·푸시·stash·rm 없음. 작업 트리에만 수정했다. 다른 수공 subagent는 진행 중이 아니었다. 이야기 문구는 바꾸지 않았다(조홍·관우의 기존 줄 그대로).
이 폴더의 `cap0`~`cap3`, `dike0`, `dike1`, `shadow0`, `flow/frame-*.png` 등은 작업 중 임시 캡처다(삭제 금지 규칙 때문에 남김). 결과물은 아래 "산출물"의 이름 붙은 파일이다.

## 1. 조홍이 안 보이고 군세 가운데가 비던 문제 (고침)
- 원인(캡처·디버그로 확인): 본편 경로에서 전투 이벤트가 컷신 단계를 만들 때 `surge` 단계의 `Unit`이 버려졌다(`internal/core/battle_events.go`, `CutStep{Kind:"surge", Text}`만 넘김). 그래서 GUI의 `leader`가 빈 문자열이라 조홍이 처음 자리 (u 47, v 0)에 그대로 주차돼 있었고, 군세 배치(`suigongArmy`)는 조홍 몫으로 한 칸(`lc == 14, v == 31`)을 비워 둔 터라 한가운데 빈자리가 났다. 데모(`-ui suigong`)는 스테이지 묶음을 직접 읽어 `Unit`이 살아 있어 이 문제가 보이지 않았다.
- 수정: `surge` 단계가 `e.matching(st.Unit)`로 푼 유닛 ID(`caohong`)를 실어 보낸다. 조홍이 여울 (25,31)에서 군세와 함께 걸어 들어와 컷 직후 비었던 칸에 선다. 회귀 시험 `TestSurgeStepNamesTheUnitThatLeadsTheArmy`(core) 추가.
- 큰 대화창: 군세 속에서 작은 말풍선으론 누군지 못 알아보므로, 둑 위 두 사람이 아닌 화자(조홍)의 줄은 이야기의 큰 대화창(초상 + 이름판 "조홍", 기존 `Offstage` 경로)으로 띄운다(`field_surge_show.go`의 `suigongLine.big`). 관우의 줄은 그대로 절벽 위 말풍선.
- 화면상 식별: 대사 때(0.6초~) 화면 아래에 조홍 초상·이름판이 있는 큰 대화창이 뜬다. 같은 때 조홍 본인은 금빛 투구의 회색 장수로 오른쪽 아래에서 군세와 함께 걸어 들어온다(약 1초 뒤부터 보임, 파란 병사들과 색이 다르다). 컷 뒤에는 군세 격자 안에 조홍이 들어가 빈자리가 없다.
- 한계: 대사 시작(0.6초) 직후 약 0.5초는 조홍이 화면 오른쪽 가장자리 밖이다(군세가 오른쪽에서 걸어 들어오는 구도 그대로). 큰 대화창이 이를 보완한다.
- 캡처: `caohong_dialog_sheet.png`(대사 타이핑·완료, 조홍 입장), `caohong_dialog_frame100.png`, `cut_no_hole_sheet.png`(둑 → 컷 뒤 여울, 조홍이 격자 안에 있음).

## 2. 둑 장면맵과 수공 땅의 연결 (고침)
- 원인: 장면 맵 `S_baihe_dike`는 절벽을 장면용 높이 2(146px)로 세웠고(수공 땅의 절벽은 26px), 윗면에 풀 조각을 흩뿌렸으며(`.25`로 흙·풀 섞임 = 어색한 언덕 위 초원), 바위·군막·망루가 있었고 나무는 큰 장면용 소나무였다.
- 수정(`tools/scene_maps/baihe_dike.py`, 맵 재생성):
  - 절벽을 전투맵과 같은 26px 낮은 언덕으로: `scene.json`에 높이 7(`lift 26`, 면 `ccliff`) 추가, `tools/scene_maps/tall_cliff.py`가 전투 절벽 칸 그대로를 `ccliff_<mask>_<variant>`로도 내보내고 `build_extra.py`로 묶었다(assets_candidate tiles.json·png 추가, extra.png 갱신). 언덕 모양은 전투맵과 같은 칸(왼쪽 u 9~11, 오른쪽 u 18~22, v≤9).
  - 언덕 윗면은 전부 맨흙(전투맵의 `h`), 풀 조각 제거. 언덕 밖은 풀.
  - 나무는 전투맵(`map25flood.py`)과 같은 칸에 전투의 `forest_N` 그림(작은 어두운 소나무 무리). 군막·모닥불·망루·깃발·등은 전투 땅에 없는 것이라 뺐다. 둑 곡식 자루 줄·거적·수레·화로(밤 조명용)만 남김.
  - `c25-dike`·`c25-dike2` 배치(관우 (19,7), 미축 (20,7), 병사들)는 그대로.
- 같은 장소·같은 방향 비교 캡처 `dike_cmp/compare_final.png`: 장면 `c25-dike`, `c25-dike2`(줌 2, 카메라 [18,8]), 전투맵 B25 같은 칸·줌 2(`dike_cmp/battle_z2`), 수공 둑 위 화면(줌 4). 언덕 높이·모양·모퉁이, 강, 언덕 밖 풀과 나무 무리가 같은 곳으로 이어진다. (이전 상태는 `dike0/frame-0190.png`: 높은 절벽 + 풀 얼룩 + 바위.)
- 한계: 장면은 밤(`Time: night`)이라 흙과 절벽이 전투맵의 낮 색보다 어둡다. 수공 그림 자체(구운 물·둑)는 바꾸지 않았다. 군막 등 진영 소품은 전투맵에 대응물이 없어 뺐다(이야기상 진영 느낌은 약해짐 - 되살리려면 전투맵에도 같은 소품이 필요).

## 3. 백하 둑의 이상한 바위 (제거, 공용 자산은 유지)
- `S_baihe_dike`에서 `rock_pile_0/1/2` 오브젝트를 모두 뺐다(둑 아래, 강 안, 길가).
- 사용처 전체 조사(`assets/graphics/maps/*.json` 개수): `S_changban`(2), `S_changban_broken`(2), `S_chibi_fleet`(1), `S_hanjin_ferry`(3), `S_huarong_marsh`(3), `S_jiangxia_harbor`(2), `S_long_field`(1), `S_runan`(2), `S_vil_peace`(1), `S_wall_top`(2), `S_wall_top_flip`(2). 생성 코드: `tools/scene_maps/{wall_top,runan,longzhong,long_field,southern_routes}.py`, 원본 3D 모델 `tools/unit3d/scene_props3d.py`(`rock_pile`), 등록 `assets_candidate/tiles/tiles.json`·`assets/graphics/index.json` extra. `internal`·`cmd` 코드에는 이름 참조 없음. `.claude/worktrees/*`에도 사본이 있으나 다른 작업 공간이다.
- 결론: 위 11개 맵이 쓰므로 `rock_pile_*` 자산·모델은 삭제하지 않는다. 더 이상 `S_baihe_dike`에서 쓰이지 않을 뿐이다.

## 4. 관우·미축 그림자 (고침)
- 원인: 전투맵 렌더가 유닛 그림자를 한 층(`f.shadows`)에 깔았다가 바닥 위에 합성한 뒤 장면 요소를 y 순서로 그린다. 절벽 윗면은 그 장면 요소(절벽 블록)라 뒤에 그려지면서 윗면 위에 선 유닛의 그림자를 덮었다(윗면 높이 26px에 놓아도 마찬가지). 이전 캡처에서 관우·미축 발밑에 그림자가 없던 것과 일치.
- 수정(`internal/gui/field.go`): 연출 중(`u.actor`)이고 올려져 있는(`u.lift > 0`) 전투맵 유닛의 그림자를 그림자 층 대신 그리기 항목으로 유닛 바로 앞(자기가 선 절벽 블록 뒤, 유닛 앞)에 그린다. 일반 전투 유닛·장면 맵은 건드리지 않는다.
- 캡처: `shadow/shadow_before_after.png`(왼쪽 이전 `campaign-corrected`, 오른쪽 이후), `shadow/after2/frame-0300.png`. 관우(말 탄)와 미축 둘 다 절벽 윗면에 맞는 그림자가 보인다.

## 5. 흰 화면 → 전투 전환, 출전 준비·병력 배치 (조사, 문제 없음)
실제 GUI 흐름을 `-ui suigong-battle`(출전 준비 화면에서 `deployment_finish`로 이벤트가 재생됨) 캡처·영상으로 확인했다. `flow/` 및 `deploy_to_battle_sheet.png`:
1. 출전 준비(`phase deployment`): 맵 B25 위에 "신야 전투 출전 10/10", "배치 10/10", 오른쪽 "출전 부대" 목록, "전투 시작" 버튼. 유비군이 구역 1(남서쪽 x 2~5, y 33~35)에 서 있다. 병력 배치는 이 단계다.
2. "전투 시작" → `deployment_finish` → 첫 이벤트 `suigong`이 곧바로 재생된다(HUD 사라짐). 출전 준비 뒤, 첫 턴 전이다. 이야기상 수공을 출전 준비보다 앞에 두려면 전투 전 이야기 단계가 필요하다(설계 선택, 바꾸지 않음).
3. 수공 약 16초: 여울(조홍 군세) → 암전(검은 화면, 이 때만 어두워짐) → 둑 위 관우 → 둑 터짐(흰 번쩍) → 여울 → 흰 화면이 가득 참(`s.end`).
4. 흰 화면이 가득 찬 순간 `finish()`가 연출용 병사(`surge#N`)를 지우고, 규칙 상태의 유닛 자리·카메라·줌·말풍선 크기를 복원한다. 그 뒤 흰 화면이 0.7초에 걸쳐 걷히며(`suigongClearSecs`) 전투 화면이 드러나고 HUD("유비군 차례 1턴")가 나온다. 즉 "어두워지며"가 아니라 흰색이 서서히 걷히며 들어간다.
5. 배치가 갑자기 바뀌지 않는다: 출전 준비 프레임(5)과 수공 뒤 전투 프레임(1080·1200)에서 유비군 대형·화면 위치가 같다. 수공 중 유비군은 화면 밖에 있었다가 같은 규칙 상태 자리로 돌아온다.
- 문제 없다고 보아 이 부분은 코드를 바꾸지 않았다.

## 산출물 (이 폴더)
- `suigong_main_battle.mp4`: 출전 준비 → 수공 → 흰 화면 → 전투, 1280x850 60fps 20.8초, 효과음 포함(`SRPG_SFX_TRACE` + `tools/sfx/dub.py --no-music`). 오디오 스트림 AAC, mean -28.5dB / max -5.5dB 확인. **귀로 직접 듣지는 못했다.**
- `caohong_dialog_sheet.png`, `caohong_dialog_frame100.png`, `cut_no_hole_sheet.png`: 조홍 대화창·입장·빈자리 없음.
- `dike_cmp/compare_final.png` (+ 원본 프레임 `dike_cmp/*`), 전 상태 `dike0/frame-0190.png`: 둑·수공 지형 비교.
- `shadow/shadow_before_after.png`, `shadow/after2/frame-0300.png`: 그림자.
- `deploy_to_battle_sheet.png`, `flow/frame-*.png`: 출전 준비·전투 전환.

## 검증
- `go build ./...`, `go vet ./internal/gui ./internal/core`, 바꾼 Go 파일 gofmt 통과.
- 통과: `./internal/scenemap`(둑 맵·브레이저 위치 때문에 한 번 실패 `TestLevelsCoverOnlyWhatStandsBehind` → 화로를 옮겨 통과), `./internal/content`, `./internal/cli`, gui `-run 'Suigong|Surge|Ch25|Cutscene'`, core `TestSurgeStepNamesTheUnitThatLeadsTheArmy`.
- 실패(변경 전과 같음, 이번 변경 밖): `./internal/core`의 `TestEveryCampaignDuelStartsWhenItsOfficersMeet`, `TestNewAt`, `TestJumpIntoTheOtherTelling`. 이전 보고서가 적은 3건과 같다. gui 전체·scene 전체 테스트는 돌리지 않았다(8GiB 맥, 부분만).
- 실제 GUI 캡처(하나씩 `lockf`)로 확인한 것: 위 모든 캡처. `go run ./cmd/_uidemo-vm`.

## 바꾼 파일
- 코드: `ver5/internal/core/battle_events.go`, `ver5/internal/core/battle_events_test.go`, `ver5/internal/gui/field.go`, `ver5/internal/gui/field_surge_show.go`(전 상태에서 신규, 이전 보고서에 이어 수정)
- 맵·자산: `ver5/tools/scene_maps/baihe_dike.py`, `ver5/assets/graphics/maps/S_baihe_dike.json`, `ver5/tools/scene_maps/tall_cliff.py`, `ver5/assets/graphics/scene.json`(높이 7), `ver5/assets/graphics/map/extra.png`, `ver5/assets/graphics/index.json`(extra 항목, 다른 미커밋 변경과 섞여 있음), `assets_candidate/tiles/tiles.json`·`assets_candidate/tiles/ccliff_*.png`(48개), `ver5/assets/scenes/ascii/c25-dike.txt`·`c25-dike2.txt`(dump 갱신)
- 문서: `ver5/docs/presentation-scene.md`(높이 7), `ver5/docs/logic-battle-event.md`(수공 장면의 조홍 대화창·그림자)
- 임시로 만들었다가 지움: `uidemo_zz_tmp_dikecam.go`, 디버그 출력 파일.

## 남은 선택·질문
- 장면 구도는 시안 승낙 없이 판단해 고쳤다(전 결과와 같은 한계). 군막 등을 되살릴지, 밤 조명의 흙 색을 더 밝게 할지는 사용자 결정.
- 수공 장면이 출전 준비 뒤에 나오는 현재 순서를 바꿀지(전투 전 이야기 단계 신설 필요).

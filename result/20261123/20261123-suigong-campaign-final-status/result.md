# 수공 본편 작업 완료 여부 (상태 조회, 2026-11-23)

변경 없음(조회만). 다른 수공 subagent 진행 중 아님(go build/test 프로세스 없음).

## 결론: 구현·산출물은 완료, 일부 한계·미확인이 남음
근거: `result/20261123/20261123-suigong-campaign-corrected/result.md`와 파일 존재 확인.
- 존재 확인: `internal/gui/field_surge_show.go`, `assets/graphics/maps/S_baihe_dike.json`, `tools/scene_maps/baihe_dike.py`, `tools/mapgen/layouts/hero/map25flood.py`; corrected 폴더에 `suigong_main_battle.mp4`, `battle_sheet.png`, dike 캡처.
- `go build ./...` 지금 재실행 통과.
- 완료된 것: 수공 본편 연결(B25 첫 이벤트 `suigong`이 `surge` 단계로 재생), 백하 둑 장면맵 36x30, 신야 전투 B25 맵 교체와 밸런스(AI 5/5 승, hard 10/10).

## 남은 것
- `./internal/core` 테스트 3건 실패(jiangxia-zhaoyun-zhangwu 일기토, NewAt sunqian, c27v-start UnknownWaypoint). 보고서는 다른 미커밋 작업(27장)과 관련으로 추정했으나 미확인.
- CLI로 전투 끝까지 플레이하지 못함(`_balance` AI로 대체). 전체 gui 테스트 미실행.
- 영상 효과음은 오디오 스트림만 확인, 귀로 듣지 못함.
- 설계 한계: 수공 장면이 출전 준비 뒤에 나옴, 수공 후 여울 칸 통행 제한 없음, 동쪽 기슭 잔류 군세 미모델링.
- 맵·장면 구도는 시안 승낙 없이 만든 것이라 사용자 확인 필요(둑 맵 3배의 해석도 가정).
- 참고: 요청 본문은 `result/20261123/suigong-campaign-final-status/summary.md`를 지정했고 그 파일은 이미 있었음(내용은 위와 일치, 일부 글자 깨짐). 덮어쓰지 않았다.

## 바꾼 파일
없음.

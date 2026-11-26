# 진행 기록: npc-class-prerequisite-design

시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,
`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.

| 시각 | 구분 | 내용 |
|---|---|---|
| 2026-11-26 07:06:47 KST(Asia/Seoul, UTC+09:00) | 단계 | request/npc-class-prerequisite-design.md -> process/npc-class-prerequisite-design/npc-class-prerequisite-design.md 이동, 작업 시작 |
| 2026-11-26 07:06:47 KST(Asia/Seoul, UTC+09:00) | status:working | 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 07:06:47 KST(Asia/Seoul, UTC+09:00) | heartbeat | 시작 pid=11833 간격=60초 |
| 2026-11-26 07:07:47 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=process / 상태=working / 수행: 요청 접수 / 다음: 요청 읽기 |
| 2026-11-26 07:08:45 KST(Asia/Seoul, UTC+09:00) | 단계 | process/npc-class-prerequisite-design/ -> test/npc-class-prerequisite-design/npc-class-prerequisite-design.md 이동, 테스트 시작 |
| 2026-11-26 07:08:45 KST(Asia/Seoul, UTC+09:00) | status:testing | 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:08:47 KST(Asia/Seoul, UTC+09:00) | heartbeat | 단계=test / 상태=testing / 수행: 테스트 실행 / 다음: 필수 테스트 전부 통과 확인 |
| 2026-11-26 07:08:59 KST(Asia/Seoul, UTC+09:00) | 테스트 | 시작 [곽가병종확인] python3 -c import json; c=json.load(open("ver5/assets/data/characters_story.json"))["Characters"]["guojia"]; t=json.load(open("ver5/assets/data/traits.json"))["Traits"]; d=json.load(open("ver5/assets/data/classes.json"))["Classes"]; assert c["Class"]=="diplomat" and "confusion_debate" in c["Traits"]; assert "stratagem" in d["diplomat"]["Pool"]; assert "stratagem" not in d["scholar"]["Pool"]; assert not t["confusion_debate"].get("Requires") |
| 2026-11-26 07:08:59 KST(Asia/Seoul, UTC+09:00) | 테스트 | 종료 [곽가병종확인] PASS (exit 0) |
| 2026-11-26 07:09:02 KST(Asia/Seoul, UTC+09:00) | 테스트메모 | 완료: 필수 테스트 1개 모두 최근 결과 PASS. 요청 파일을 /Users/younwookim/orca/mhdemo1/result/20261126/npc-class-prerequisite-design/npc-class-prerequisite-design.md 로 옮김 |
| 2026-11-26 07:09:02 KST(Asia/Seoul, UTC+09:00) | status:done | 수행: 완료, result로 이동 |

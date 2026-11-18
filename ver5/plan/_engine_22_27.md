<!-- 2026-11-17 엔진 에이전트. 22~27장 전투(B22~B27, B25V~B27V)를 스테이지 데이터로 짜는 법.
     아래 ```json 블록은 모두 `go test -p 1 -run TestEngineCookbook ./internal/content`가 실제 데이터 묶음에 붙여 로드 검증한다
     (각 블록의 필드가 B21을 비운 바탕 스테이지의 같은 필드를 갈아 끼운다. 아직 데이터에 없는 장수 ID는 진손을 본뜬 대역으로 센다).
     블록을 고치면 테스트를 다시 돌린다. 좌표는 B21 맵(20×32) 안의 예시라 실제 맵에 맞게 옮긴다. -->

# 22~27장 전투 요리책 (엔진)

설계(`plan/_design_22_27.md`)의 아홉 전투가 쓰는 장치를 스테이지 JSON으로 적는 법이다. 대부분은 이미 있는 이벤트·조건으로 되고,
새로 넣은 엔진 기능은 다섯 가지뿐이다(아래 표의 **새**). 필드의 정확한 뜻은 `docs/data-scenario-node.md`, 동작 시점은
`docs/logic-battle-event.md`를 본다.

| 장치 | 쓰는 전투 | 방법 |
|---|---|---|
| 출전을 두세 무리로 나눠 다른 자리에서 시작, 무리마다 필수 장수 | B22 강하, B24 박망파, B25 신야, B26 장판파(조운 혼자), B27 화용도 | **새** `Stage.Groups` (1절) |
| 2단계 승리(습격 뒤 목표가 바뀜) | B22 | 있음: `goals` 동작 (2절) |
| 습격 뒤 "여섯 턴 버티기" | B22 | **새** 조건 `since:이벤트>=N` (2절) |
| 조건이 이어지는 동안 N턴마다 증원 | B22 기각지세, B26 호표기, B25V 매 턴 셋, B26V·B27V 3턴마다 | **새** 이벤트 `Every` (3절) |
| 배·가장자리로 달아나 사라지는 적, 막히면 항복 | B22 | 있음: `flee`+`GoalArea`, `fled:`, `remove` (2절) |
| 병력이 깎인 채 시작 | B24, B25, B27 | **새** 스폰 `HPPct` (4절) |
| 능력 약화·혼란을 건 채 시작 | B24, B25 | 있음: `buff`(능력) + **새** `buff`의 상태이상 `confusion` 등 (4절) |
| 능력이 한 턴 뒤 돌아오는 허저 | B25 | 있음: 허저에게만 `buff` `Turns: 1` (5절) |
| 달아나는 대장이 끝에 닿으면 승리 | B24, B25, B27 | 있음: `flee`+`GoalArea`, 승리 `defeated:`, 부목표 `defeated:`+`!fled:` (4·5절) |
| 깃발수가 사는 동안 진 전체에 버프, 쓰러지면 버프가 걷히고 한 턴 혼란 | B23 | 있음: `buff` `Turns: 0` → 반대 부호 `buff`로 걷기 + **새** 혼란 (6절) |
| 무너지는 진채(적이 서거나 수비 둘 아래), 수비는 성으로 물러남 | B26V, B27V | 있음: `any:[at:enemy:N, !alive:G>=2]`, `ai run`, `flag` (7절) |
| N턴 버티기 또는 대장 격파 | B26V | 있음: 승리 목표 둘 (7절) |
| 스스로 걷는 피난민, 하나라도 잃으면 패배, 모두 닿으면 승리 | B25V | 있음: `Party` AI `run` + 무리마다 `Group` (8절) |
| 개전에 길을 따라 번지는 불 | B24 | 있음: `terrain` 여러 번 + `cutscene` `wait` (4절, B02 영천의 방식) |
| 동맹 AI 수군 | B27V | 있음: `Boats` + `Armies`의 `Ally` + `Party` AI (9절) |
| 그 전투에 못 나오는 장수(관우가 강하·화용도 끝에 가 있음) | B26, B27, B26V | 있음: 이야기 `leave` → 전투 `join`(또는 전투 뒤 이야기 `join`) (7절) |

새 기능 다섯: `Stage.Groups`, 이벤트 `Every`, 조건 `since:`, 스폰 `HPPct`, `buff`의 상태이상(`confusion` `seal` `root` `disarm`).

## 0. 공통 주의

- 일기토를 병력이 깎였다고 여는 이벤트(`hp:X<N` → `duel` 동작)는 쓰지 않는다(AGENTS.md). B23 "조인 병력 삼 할 아래 → 이전이 구하러 온다"는
  이전을 `ai`로 조인 쪽에 보내는 데서 그치고, 일기토는 둘이 맞닿을 때 `Stage.Duels`로 저절로 열린다.
- `defeated:X`는 달아나 빠진(`fled`) 유닛에도 참이다. "달아나기 전에 물리친다"는 `["defeated:X", "!fled:X"]`.
- `matching` 대상은 유닛 ID·`Group`·편(`ally` `enemy` `player`)이다. 진영색(`wu` `jing`)은 대상이 아니다. 동맹 수군을 가리키려면 `Group`을 준다.
- 템플릿 병사(`troop_*`)는 `say`의 `Actor`가 될 수 없다(번호 붙은 ID로 생긴다). 적 무리의 말은 이름 있는 장수나 `Offstage` 줄로.
- 상태 수치(`buff`)는 같은 능력치·같은 방향이면 더 센 값·더 긴 턴이 남는다. 반대 부호를 다시 걸면 **걷힌다**(지워짐). 그래서 "버프 걷기"는 같은 크기의 반대 `buff`다.
- 상태의 턴은 그 유닛 진영의 차례가 끝날 때 하나 준다. 개전에 적에게 `Turns: 1`을 걸면 적의 첫 차례 동안만 간다.

## 1. 출전을 무리로 나누기 — `Stage.Groups` (새)

`Groups`의 무리는 `Areas`의 구역 하나다. 그 구역 안의 `Allies` 칸이 무리의 자리이고, 칸 수가 그 무리의 인원이다(따로 `Max` 없음).
무리의 `Required` 장수는 그 무리 칸에만 설 수 있다. 다른 장수는 아무 무리에나 선다. 그래서 플레이어는 배치 화면에서 장수를 골라 칸을 눌러
누가 어느 무리로 갈지 정한다(필수 장수를 고르면 그 무리 칸만 밝게 빛나고, 자리를 바꾸는 교환도 필수 장수를 무리 밖으로 내보내면 거부된다).
배치 화면의 출전 목록은 장수마다 "배치됨" 대신 무리 이름(본군·별동대)을 보인다. 자동 배치(`DeploymentPlan`: 자동 진행, srpg-cli `auto`, 강제 출전)는
필수 장수를 제 무리 칸에 먼저 맞춘 뒤 나머지를 `Allies` 순서대로 채운다.

검증: 무리 이름은 유일, 구역에 `Allies` 칸이 하나 이상, 두 무리가 한 칸을 나누지 않음, `Required`는 스테이지 `Required`에도 있어야 하고
한 무리에만, 무리 칸 수 이하, `Recommend`가 있으면 제 무리 칸. 스테이지 `Required`는 "로스터에 있으면" 출전이라 고순처럼 없을 수도 있는 장수를 넣어도 된다.

B22 강하(본군 6 북쪽, 별동대 4 남쪽 배 위, 고순이 살아 있으면 별동대에 고정):

```json
{
  "Max": 10,
  "Required": ["liubei", "guanyu", "gaoshun"],
  "Boats": true,
  "Areas": [
    {"ID": 1, "Rect": {"X": 2, "Y": 2, "W": 3, "H": 2}, "Name": "본군 자리", "Description": "강 북쪽 언덕. 유비와 다섯"},
    {"ID": 2, "Rect": {"X": 2, "Y": 28, "W": 4, "H": 1}, "Name": "별동대 자리", "Description": "강 남쪽 물 위의 배. 관우와 셋"}
  ],
  "Allies": [{"X": 2, "Y": 2}, {"X": 3, "Y": 2}, {"X": 4, "Y": 2}, {"X": 2, "Y": 3}, {"X": 3, "Y": 3}, {"X": 4, "Y": 3},
             {"X": 2, "Y": 28}, {"X": 3, "Y": 28}, {"X": 4, "Y": 28}, {"X": 5, "Y": 28}],
  "Recommend": {"liubei": {"X": 3, "Y": 2}, "guanyu": {"X": 2, "Y": 28}},
  "Groups": [
    {"Name": "본군", "Area": 1, "Required": ["liubei"]},
    {"Name": "별동대", "Area": 2, "Required": ["guanyu", "gaoshun"]}
  ]
}
```

- 배 위에서 시작하려면 `Boats: true`와 물 칸의 `Allies`면 된다(물 위 병력은 배를 탄 모습).
- B24 박망파는 무리 셋이다(아래). B25 신야(본대 남서, 관우·미축과 둘 남동)는 `Required`를 `["liubei","guanyu","mizhu"]`, 무리 둘에
  `["liubei"]`·`["guanyu","mizhu"]`. B27 화용도(장비와 둘 북서 복병, 본군 6 남동)는 무리 둘에 `["zhangfei"]`·`["liubei"]`.
  B26 장판파의 "조운 혼자 서북쪽 마을"은 칸 하나짜리 무리 `{"Name": "조운", "Area": …, "Required": ["zhaoyun"]}`(10절).

B24 박망파(유비·조운 남서 2칸, 관우와 둘 북서 산 3칸, 장비와 둘 남동 숲 3칸):

```json
{
  "Max": 8,
  "Required": ["liubei", "zhaoyun", "guanyu", "zhangfei"],
  "Areas": [
    {"ID": 1, "Rect": {"X": 1, "Y": 28, "W": 2, "H": 1}, "Name": "남서쪽 길 끝", "Description": "유비와 조운"},
    {"ID": 2, "Rect": {"X": 1, "Y": 2, "W": 2, "H": 2}, "Name": "북서쪽 산", "Description": "관우의 복병"},
    {"ID": 3, "Rect": {"X": 17, "Y": 28, "W": 2, "H": 2}, "Name": "남동쪽 숲", "Description": "장비의 복병"}
  ],
  "Allies": [{"X": 1, "Y": 28}, {"X": 2, "Y": 28},
             {"X": 1, "Y": 2}, {"X": 2, "Y": 2}, {"X": 1, "Y": 3},
             {"X": 17, "Y": 28}, {"X": 18, "Y": 28}, {"X": 17, "Y": 29}],
  "Recommend": {"liubei": {"X": 1, "Y": 28}, "zhaoyun": {"X": 2, "Y": 28}, "guanyu": {"X": 1, "Y": 2}, "zhangfei": {"X": 17, "Y": 28}},
  "Groups": [
    {"Name": "본대", "Area": 1, "Required": ["liubei", "zhaoyun"]},
    {"Name": "산", "Area": 2, "Required": ["guanyu"]},
    {"Name": "숲", "Area": 3, "Required": ["zhangfei"]}
  ]
}
```

## 2. B22 강하 — 2단계 승리, 배로 달아나기·항복, 기각지세, 여섯 턴 버티기

- **1단계 → 2단계**: 두 대장이 모두 물러나면 `ambush` 이벤트가 복면 무리를 내고 `goals`로 승리 목표를 바꾼다(B07·B21과 같은 방식).
- **여섯 턴 버티기**: 조건 `since:ambush>=6`(새). 이벤트 `ambush`가 마지막으로 실행된 라운드 + 6 이상이면 참. 습격이 라운드 r의 어느 차례에 나도
  라운드 r+6의 아군 차례가 열리는 순간 참이 된다.
- **기각지세**: 이벤트 `Every: 3`(새). 조건이 참인 동안 처음 한 번, 그 뒤 세 라운드가 지날 때마다 다시 실행한다. 조건이 거짓이 된 동안은 쉬고,
  다시 참이 되면 마지막 실행에서 세 라운드가 지났을 때 다시 낸다. "싸우기 시작했다"는 `any:[at:player:진채 둘레, hp:무리<100]`(아군이
  둘레에 들어섰거나 그 무리 누구든 다쳤다)로 쓴다. 한 번 다친 무리는 계속 참이라 양쪽을 다 건드리면 구원병이 끊긴다.
- **달아나기**: 대장이 쓰러지면 남은 무리를 `ai flee` + `GoalArea`(나루)로 바꾼다. 나루 칸에 서면 맵에서 빠지고 `fled:무리`가 참.
  부목표 "하나도 못 빠져나간다"는 `!fled:north`, `!fled:south`.
- **막히면 항복**: 별동대가 나루 앞 물길(구역 7)에 서 있으면 `remove`로 무리를 뺀다(`retreat`라 `fled`는 거짓으로 남는다).
- **이적**: 플레이어블 인물은 `spawn`으로 못 만든다. 군량을 싣고 오는 이적은 전투 `join`(플레이어가 부리는 아군으로 편입, 명단에 남음)이다.
  이미 명단에 든 인물에게 이야기 `join`이 또 오면 레벨만 합류 레벨로 맞추고 지나가므로 해롭지는 않지만, 전투 뒤 합류 장면은 "이미 함께
  싸운 사람"으로 쓴다. 전투에서 부리지 않을 거면 전투 `join`을 빼고 이야기 `join`만 둔다.

```json
{
  "Required": ["liubei", "guanyu"],
  "Boats": true,
  "Areas": [
    {"ID": 3, "Rect": {"X": 0, "Y": 0, "W": 3, "H": 3}, "Name": "북서쪽 숲", "Description": "복면 무리가 나오는 곳"},
    {"ID": 4, "Rect": {"X": 5, "Y": 0, "W": 7, "H": 4}, "Name": "북쪽 진채 둘레", "Description": "아군이 들어서면 장무 쪽이 싸우기 시작한 것"},
    {"ID": 5, "Rect": {"X": 5, "Y": 24, "W": 7, "H": 5}, "Name": "남쪽 진채 둘레", "Description": "아군이 들어서면 진손 쪽이 싸우기 시작한 것"},
    {"ID": 6, "Rect": {"X": 19, "Y": 14, "W": 1, "H": 4}, "Name": "나루", "Description": "반란군의 배가 묶인 강 동쪽 끝. 닿으면 빠져나감"},
    {"ID": 7, "Rect": {"X": 17, "Y": 14, "W": 2, "H": 4}, "Name": "나루 앞 물길", "Description": "별동대가 서면 달아나던 무리가 항복"}
  ],
  "Enemies": [
    {"Character": "jiangwu", "X": 8, "Y": 2, "Rank": "commander", "Group": "north", "Policy": "react", "Faction": "jing"},
    {"Character": "troop_cavalry", "X": 7, "Y": 2, "Group": "north", "Policy": "react", "Faction": "jing"},
    {"Character": "troop_footman", "X": 9, "Y": 2, "Group": "north", "Policy": "react", "Faction": "jing"},
    {"Character": "chensun", "X": 8, "Y": 26, "Rank": "commander", "Group": "south", "Policy": "react", "Faction": "jing"},
    {"Character": "troop_archer", "X": 7, "Y": 26, "Group": "south", "Policy": "react", "Faction": "jing"},
    {"Character": "troop_footman", "X": 9, "Y": 26, "Group": "south", "Policy": "react", "Faction": "jing"}
  ],
  "Victory": [{"If": ["defeated:jiangwu", "defeated:chensun"], "Text": "장무와 진손을 물리친다"}],
  "Defeat": [{"If": ["defeated:guanyu"], "Text": "관우 퇴각"}],
  "Sub": [
    {"ID": "boats", "Text": "반란군의 배가 하나도 빠져나가지 못한다", "If": ["!fled:north", "!fled:south"], "Flag": "flag-b22-boats", "Reward": {"insu_high": 1}},
    {"ID": "masked", "Text": "복면을 쓴 장수를 물리친다", "If": ["defeated:caizhong"], "Flag": "flag-b22-caizhong"}
  ],
  "Events": [
    {"ID": "north-relief", "Every": 3,
     "If": ["turnstart:enemy", "alive:jiangwu", "any:[at:player:5, hp:south<100]", "!any:[at:player:4, hp:north<100]"],
     "Do": [{"Kind": "spawn", "Spawns": [
       {"Character": "troop_cavalry", "X": 12, "Y": 20, "Side": "enemy", "Group": "relief", "Policy": "assault", "Faction": "jing", "Shift": -0.07},
       {"Character": "troop_cavalry", "X": 13, "Y": 20, "Side": "enemy", "Group": "relief", "Policy": "assault", "Faction": "jing", "Shift": -0.07}]}]},
    {"ID": "south-relief", "Every": 3,
     "If": ["turnstart:enemy", "alive:chensun", "any:[at:player:4, hp:north<100]", "!any:[at:player:5, hp:south<100]"],
     "Do": [{"Kind": "spawn", "Spawns": [
       {"Character": "troop_cavalry", "X": 12, "Y": 8, "Side": "enemy", "Group": "relief", "Policy": "assault", "Faction": "jing", "Shift": -0.07},
       {"Character": "troop_cavalry", "X": 13, "Y": 8, "Side": "enemy", "Group": "relief", "Policy": "assault", "Faction": "jing", "Shift": -0.07}]}]},
    {"ID": "relief-hint", "If": ["any:[since:north-relief>=0, since:south-relief>=0]"],
     "Do": [{"Kind": "say", "Lines": [{"Actor": "jianyong", "Text": "현덕, 한쪽만 치면 다른 쪽이 건너오네. 둘을 한날한시에 물어야 하네."}]}]},
    {"ID": "north-flees", "If": ["defeated:jiangwu"],
     "Do": [{"Kind": "ai", "Targets": ["north"], "Policy": "flee", "GoalArea": 6}]},
    {"ID": "south-flees", "If": ["defeated:chensun"],
     "Do": [{"Kind": "ai", "Targets": ["south"], "Policy": "flee", "GoalArea": 6}]},
    {"ID": "north-yields", "If": ["defeated:jiangwu", "alive:north", "at:player:7"],
     "Do": [{"Kind": "say", "Lines": [{"Actor": "guanyu", "Text": "물길은 막혔다. 창을 버려라."}]},
            {"Kind": "remove", "Targets": ["north"]}]},
    {"ID": "south-yields", "If": ["defeated:chensun", "alive:south", "at:player:7"],
     "Do": [{"Kind": "remove", "Targets": ["south"]}]},
    {"ID": "ambush", "If": ["defeated:jiangwu", "defeated:chensun"],
     "Do": [
       {"Kind": "spawn", "Spawns": [
         {"Character": "caizhong", "X": 1, "Y": 1, "Rank": "commander", "Side": "enemy", "Group": "masked", "Policy": "assault", "Faction": "jing"},
         {"Character": "caihua", "X": 2, "Y": 1, "Side": "enemy", "Group": "masked", "Policy": "assault", "Faction": "jing"},
         {"Character": "troop_archer", "X": 0, "Y": 2, "Side": "enemy", "Group": "masked", "Policy": "assault", "Faction": "jing"},
         {"Character": "troop_footman", "X": 1, "Y": 2, "Side": "enemy", "Group": "masked", "Policy": "assault", "Faction": "jing"}]},
       {"Kind": "join", "Spawns": [{"Character": "yiji", "X": 10, "Y": 0, "Facing": "SW"}]},
       {"Kind": "say", "Lines": [{"Actor": "caihua", "Text": "형님, 관우의 배가 돌아오기 전에 끝냅시다!"}]},
       {"Kind": "say", "Lines": [{"Actor": "yiji", "Text": "저 활 쏘는 품을 보십시오. 저것은 양양 수군의 대오입니다."}]},
       {"Kind": "goals", "Victory": [
         {"If": ["since:ambush>=6"], "Text": "복면 무리를 여섯 턴 버틴다"},
         {"If": ["defeated:caizhong", "defeated:caihua"], "Text": "복면 장수 둘을 물리친다"}]}]}
  ]
}
```

## 3. N턴마다 증원 — 이벤트 `Every` (새)

`Every: N`인 이벤트는 조건이 참이면 실행하고, 그 뒤로는 마지막 실행에서 N 라운드가 지난 검사 때마다(조건이 여전히 참이면) 다시 실행한다.
`spawn`을 쓸 수 있다(`Repeat`은 여전히 못 쓴다). `Repeat`과 함께 쓸 수 없고 `join`은 넣을 수 없다. 자리가 모자라 미뤄진 실행은 기록되지 않아
다음 검사에서 다시 시도한다. 실행한 라운드가 상태(`Fired`)에 남아 저장·불러오기·재도전을 따라간다.

- "매 턴 셋씩"(B25V): `"Every": 1, "If": ["turnstart:enemy"]`.
- "3턴마다 넷씩, 조조가 나오기 전까지"(B26V·B27V): `"Every": 3, "If": ["turnstart:enemy", "!flag:…"]` 또는 `"!round>=10"`.
- 조건 없이 `Every`만 주면 첫 검사(개전)부터 돈다. 적 차례 시작에 맞추려면 `turnstart:enemy`를 함께 쓴다.
- 증원이 끝없이 쌓이면 맵이 막혀 스폰이 미뤄진다. 스테이지 레벨표대로 수를 정하고 `Shift: -0.07`로 무리의 레벨을 낮춘다(stage_lv_balance.md).

B26 연의 장판파(조순의 호표기 3턴마다 넷)는 10절에 있다.

## 4. 깎인 채 시작하는 적, 번지는 불, 달아나면 끝나는 전투 — B24 박망파

- **병력**: 스폰 `HPPct`(새, 1~99)는 그 유닛의 최대 병력에 대한 처음 병력 비율이다. 레벨이 난이도마다 달라 `HP`(절댓값) 대신 이것을 쓴다.
  `HP`와 함께 쓸 수 없다.
- **능력 약화**: 조건 없는 이벤트의 `buff`(`Turns` 생략 = 전투 끝까지). 같은 무리에 능력치마다 `buff` 하나씩.
- **혼란**: `buff`의 `Stat`에 상태이상 `confusion`(새, 그 밖에 `seal` 봉책·`root` 속박·`disarm` 봉격)을 쓴다. `Value`는 적지 않고 `Turns`가
  그 차례 수다. 혼란은 움직이지도 행동하지도 반격하지도 못한다. 상태이상 면역 유닛에는 걸리지 않고 연출도 나오지 않는다.
- **혼란이 풀리면 물러난다**: 처음부터 `Policy: "flee"`, `GoalArea`(여울 너머)를 준다. 혼란 동안은 서 있다가 풀리면 달아난다.
- **번지는 불**: B02 영천처럼 작은 구역마다 `terrain`(`State: "F"` 화염)을 차례로 놓는다. `terrain` 하나가 재생에서 0.4초쯤 걸리고, 사이에
  `cutscene`의 `wait`를 끼우면 더 천천히 번진다. `Turns`를 주면 그 라운드 뒤 그을음(`s`)이 된다. 복병이 들어갈 틈은 구역을 비워 둔다.
- **달아나면 승리**: 승리 목표 `defeated:xiahoudun` 하나로 "물리치거나 여울을 건너 달아남"이 모두 된다. 부목표는 `!fled:`를 붙인다.

```json
{
  "Areas": [
    {"ID": 11, "Rect": {"X": 5, "Y": 14, "W": 1, "H": 3}, "Name": "불 1", "Description": "길 서쪽 첫 불"},
    {"ID": 12, "Rect": {"X": 7, "Y": 14, "W": 1, "H": 3}, "Name": "불 2", "Description": "길 동쪽 첫 불"},
    {"ID": 13, "Rect": {"X": 5, "Y": 19, "W": 1, "H": 3}, "Name": "불 3", "Description": "길 서쪽 둘째 불(사이 두 칸은 복병 틈)"},
    {"ID": 14, "Rect": {"X": 7, "Y": 19, "W": 1, "H": 3}, "Name": "불 4", "Description": "길 동쪽 둘째 불"},
    {"ID": 15, "Rect": {"X": 14, "Y": 0, "W": 6, "H": 1}, "Name": "여울 너머", "Description": "하후돈이 닿으면 달아남"}
  ],
  "Enemies": [
    {"Character": "xiahoudun", "X": 6, "Y": 15, "Rank": "commander", "Group": "road", "Policy": "flee", "GoalArea": 15, "HPPct": 50},
    {"Character": "yujin", "X": 6, "Y": 16, "Group": "road", "Policy": "flee", "GoalArea": 15, "HPPct": 50},
    {"Character": "troop_cavalry", "X": 6, "Y": 18, "Group": "road", "Policy": "flee", "GoalArea": 15, "HPPct": 50},
    {"Character": "troop_footman", "X": 6, "Y": 20, "Group": "road", "Policy": "flee", "GoalArea": 15, "HPPct": 50},
    {"Character": "lidian", "X": 15, "Y": 2, "Group": "ford", "Policy": "hold", "HPPct": 50}
  ],
  "Victory": [{"If": ["defeated:xiahoudun"], "Text": "하후돈을 물리치거나 여울 너머로 쫓아낸다"}],
  "Sub": [{"ID": "before-ford", "Text": "하후돈이 여울을 건너기 전에 물리친다", "If": ["defeated:xiahoudun", "!fled:xiahoudun"],
           "Flag": "flag-b24-xiahoudun", "Reward": {"insu_high": 1}}],
  "Events": [
    {"ID": "fire-trap", "Do": [
      {"Kind": "say", "Lines": [{"Actor": "yujin", "Text": "장군, 풀이 바짝 말랐습니다. 불이 무섭습니다!"}]},
      {"Kind": "terrain", "Area": 11, "State": "F", "Turns": 3},
      {"Kind": "terrain", "Area": 12, "State": "F", "Turns": 3},
      {"Kind": "cutscene", "Steps": [[{"Kind": "wait", "Seconds": 0.5}]]},
      {"Kind": "terrain", "Area": 13, "State": "F", "Turns": 3},
      {"Kind": "terrain", "Area": 14, "State": "F", "Turns": 3},
      {"Kind": "say", "Lines": [{"Actor": "xiahoudun", "Text": "……불? 물러나라! 돌아서라!"}]},
      {"Kind": "buff", "Targets": ["road", "ford"], "Stat": "confusion", "Turns": 2},
      {"Kind": "buff", "Targets": ["road", "ford"], "Stat": "attack", "Value": -20},
      {"Kind": "buff", "Targets": ["road", "ford"], "Stat": "defense", "Value": -20},
      {"Kind": "say", "Lines": [{"Actor": "guanyu", "Text": "불이 올랐다. 내려간다!"}]}]}
  ]
}
```

B27 화용도는 같은 틀이다: 조조 무리에 `HPPct`, `Policy: "flee"`, `GoalArea`(북서쪽 끝 띠), 승리 `defeated:caocao`, 부목표
`["defeated:caocao", "!fled:caocao"]`. 장비의 복병은 1절의 무리로 북서쪽 길목에 처음부터 선다(숨은 유닛은 없다). "조조가 길목에 다가오면 장비가
드러난다"는 `at:caocao:길목 앞` 조건의 `say`·`cutscene`으로 쓴다.

## 5. B25 신야 — 모두 깎인 채 북서쪽으로 걷는 조인군, 한 턴 뒤 돌아오는 허저

- 능력 약화는 `rout` 무리에 `Turns` 없이(전투 끝까지), **허저 한 사람에게만 `Turns: 1`** 로 따로 건다. 같은 약화가 허저에게 둘 걸리면 더 긴 쪽이
  남으므로 허저를 `rout` 무리에 넣지 않는다.
- "아군이 조인에게 다가가면 허저가 돌아서서 막는다"는 조인 둘레가 아니라 들판 띠(구역 21)에 아군이 들어서는 조건으로 쓴다(거리 조건은 `adjacent`뿐).
- `mind` `agility` `morale`도 같은 꼴로 줄을 더한다.

```json
{
  "Required": ["liubei", "guanyu", "mizhu"],
  "Areas": [
    {"ID": 20, "Rect": {"X": 0, "Y": 0, "W": 8, "H": 1}, "Name": "완으로 가는 길", "Description": "북서쪽 끝. 조인이 닿으면 달아남"},
    {"ID": 21, "Rect": {"X": 0, "Y": 12, "W": 20, "H": 4}, "Name": "들판 가운데", "Description": "아군이 들어서면 허저가 돌아선다"}
  ],
  "Enemies": [
    {"Character": "caoren", "X": 8, "Y": 18, "Rank": "boss", "Group": "rout", "Policy": "flee", "GoalArea": 20, "HPPct": 70},
    {"Character": "caohong", "X": 7, "Y": 19, "Group": "rout", "Policy": "flee", "GoalArea": 20, "HPPct": 70},
    {"Character": "troop_footman", "X": 6, "Y": 19, "Group": "rout", "Policy": "flee", "GoalArea": 20, "HPPct": 70},
    {"Character": "xuchu", "X": 9, "Y": 19, "Policy": "flee", "GoalArea": 20, "HPPct": 70},
    {"Character": "troop_cavalry", "X": 10, "Y": 19, "Group": "rearguard", "Policy": "flee", "GoalArea": 20, "HPPct": 70}
  ],
  "Victory": [{"If": ["defeated:caoren"], "Text": "조인이 북서쪽 끝으로 빠져나가거나 물리친다"}],
  "Defeat": [{"If": ["defeated:guanyu"], "Text": "관우 퇴각"}, {"If": ["defeated:mizhu"], "Text": "미축 퇴각"}],
  "Sub": [{"ID": "caoren", "Text": "조인이 북서쪽 끝에 닿기 전에 물리친다", "If": ["defeated:caoren", "!fled:caoren"], "Flag": "flag-b25-caoren"}],
  "Events": [
    {"ID": "routed", "Do": [
      {"Kind": "buff", "Targets": ["rout", "rearguard"], "Stat": "attack", "Value": -30},
      {"Kind": "buff", "Targets": ["rout", "rearguard"], "Stat": "defense", "Value": -30},
      {"Kind": "buff", "Targets": ["xuchu"], "Stat": "attack", "Value": -30, "Turns": 1},
      {"Kind": "buff", "Targets": ["xuchu"], "Stat": "defense", "Value": -30, "Turns": 1},
      {"Kind": "say", "Lines": [{"Actor": "caoren", "Text": "불에 물까지……. 완으로 물러난다. 살아서 가는 것이 먼저다!"}]}]},
    {"ID": "rearguard-turns", "If": ["at:player:21"], "Do": [
      {"Kind": "ai", "Targets": ["xuchu", "rearguard"], "Policy": "assault"},
      {"Kind": "say", "Lines": [{"Actor": "xuchu", "Text": "장군은 먼저 가시오. 뒤는 내가 막소!"}]}]}
  ]
}
```

## 6. B23 팔문금쇄진 — 깃발수가 사는 동안의 진 버프, 쓰러지면 걷히고 한 턴 혼란

- 버프는 진 안의 무리(문마다 `Group`, 축 `axis`, 깃발수 `bearer`)에 개전 `buff` `Turns` 생략으로 건다. 구역 안을 드나드는 유닛을 따라가는
  오라는 없다. 진의 군사는 `react`·`hold`로 제자리를 지키니 무리로 거는 것과 같다.
- 깃발수가 쓰러지면 같은 크기의 반대 `buff`로 걷고(반대 부호는 지운다), `confusion` `Turns: 1`, 이어 `ai assault`(혼란이 풀린 뒤 덤빈다).
- 문 하나가 맞으면 양옆이 돕는다: 문마다 `hp:gateN<100` → 양옆 문 무리를 그 문 구역으로 `march`.
- 사문·상문으로 들면 매복: `at:player:사문 앞` → 기병 셋 `spawn`.
- 부목표 "조인과 싸우기 전에 축을 깬다"는 깃발수가 쓰러질 때 조인이 아직 멀쩡하면 플래그를 세우는 이벤트로.
- 단복은 Playable이 아닌 화면용 인물 `danfu`다([기반] 결정). 편성 밖 손님 장수는 `Party`의 AI 아군(`AI: true`)으로 세우고 패배 목표에
  `defeated:danfu`를 둔다. 플레이어가 부리는 손님(`Party`에 `AI` 없이)은 쓰지 않는다: 그런 인물은 임시 인물이 아니라 전투 뒤에도 명단에 남는다.
- "생문 쪽 땅이 한 번 빛난다"를 그리는 동작은 없다. 단복의 `cutscene`(생문 쪽으로 `face` + `say`)으로 대신한다.

```json
{
  "Max": 8,
  "Required": ["liubei", "zhaoyun"],
  "Allies": [{"X": 8, "Y": 28}, {"X": 9, "Y": 28}, {"X": 10, "Y": 28}, {"X": 11, "Y": 28}, {"X": 12, "Y": 28},
             {"X": 8, "Y": 29}, {"X": 9, "Y": 29}, {"X": 11, "Y": 29}, {"X": 12, "Y": 29}],
  "Areas": [
    {"ID": 30, "Rect": {"X": 9, "Y": 16, "W": 3, "H": 2}, "Name": "북문(휴문)", "Description": "문 하나. 맞으면 양옆 문이 돕는다"},
    {"ID": 31, "Rect": {"X": 9, "Y": 13, "W": 3, "H": 1}, "Name": "사문 앞", "Description": "들어서면 매복 기병"},
    {"ID": 32, "Rect": {"X": 10, "Y": 30, "W": 1, "H": 1}, "Name": "신야성", "Description": "적이 서면 패배"}
  ],
  "Party": [{"Character": "danfu", "X": 10, "Y": 27, "AI": true, "Policy": "hold", "Faction": "shu"}],
  "Enemies": [
    {"Character": "caoren", "X": 10, "Y": 11, "Rank": "boss", "Policy": "hold"},
    {"Character": "lidian", "X": 14, "Y": 10, "Rank": "commander", "Policy": "hold"},
    {"Character": "troop_footman", "X": 10, "Y": 16, "Group": "gate1", "Policy": "react"},
    {"Character": "troop_footman", "X": 13, "Y": 17, "Group": "gate2", "Policy": "react"},
    {"Character": "troop_footman", "X": 7, "Y": 17, "Group": "gate8", "Policy": "react"},
    {"Character": "troop_footman", "X": 10, "Y": 19, "Group": "axis", "Policy": "hold"},
    {"Character": "troop_footman", "X": 11, "Y": 19, "Group": "bearer", "Policy": "hold"}
  ],
  "Victory": [{"If": ["defeated:caoren"], "Text": "조인을 물리친다"}],
  "Defeat": [{"If": ["defeated:danfu"], "Text": "단복 퇴각"}, {"If": ["at:enemy:32"], "Text": "적이 신야성에 선다"}],
  "Sub": [{"ID": "axis-first", "Text": "조인과 싸우기 전에 진의 축을 깬다", "If": ["flag:flag-b23-axis-first"], "Flag": "flag-b23-axis"}],
  "Events": [
    {"ID": "formation", "Do": [
      {"Kind": "buff", "Targets": ["gate1", "gate2", "gate8", "axis", "bearer"], "Stat": "defense", "Value": 20}]},
    {"ID": "gate1-help", "If": ["hp:gate1<100", "alive:bearer"], "Do": [
      {"Kind": "ai", "Targets": ["gate2", "gate8"], "Policy": "march", "GoalArea": 30}]},
    {"ID": "dead-gate", "If": ["at:player:31"], "Do": [
      {"Kind": "spawn", "Spawns": [
        {"Character": "troop_cavalry", "X": 6, "Y": 12, "Side": "enemy", "Group": "ambush", "Policy": "assault"},
        {"Character": "troop_cavalry", "X": 14, "Y": 12, "Side": "enemy", "Group": "ambush", "Policy": "assault"},
        {"Character": "troop_cavalry", "X": 10, "Y": 12, "Side": "enemy", "Group": "ambush", "Policy": "assault"}]}]},
    {"ID": "axis-first", "If": ["alldefeated:bearer", "!hp:caoren<100"], "Do": [
      {"Kind": "flag", "Flags": {"flag-b23-axis-first": 1}}]},
    {"ID": "axis-falls", "If": ["alldefeated:bearer"], "Do": [
      {"Kind": "buff", "Targets": ["gate1", "gate2", "gate8", "axis"], "Stat": "defense", "Value": -20},
      {"Kind": "buff", "Targets": ["gate1", "gate2", "gate8", "axis"], "Stat": "confusion", "Turns": 1},
      {"Kind": "ai", "Targets": ["gate1", "gate2", "gate8", "axis", "caoren"], "Policy": "assault"},
      {"Kind": "say", "Lines": [{"Actor": "caoren", "Text": "누구냐. 유비 밑에 누가 있기에!"}]}]}
  ]
}
```

`axis-first`가 `axis-falls`보다 앞에 있어야 같은 검사에서 먼저 본다(이벤트는 데이터 순서로 훑는다). 지금 예에서는 둘 다 조인의 병력만 보므로
순서가 바뀌어도 같다.

## 7. 진채 — B26V 강릉 수비전 (B27V도 같은 규칙)

- 진채는 칸 하나짜리 구역. 수비는 `Party` AI(`hold`) 무리. 진채가 서 있는 동안 버프는 개전 `buff`(방어 %, 사거리 칸).
- 무너짐: `any:[at:enemy:진채, !alive:수비>=2]`(적이 진채 칸에 섬 또는 수비 둘 아래). 버프를 반대 부호로 걷고, 수비를 `run`으로 성 구역에 보내
  (닿으면 맵에서 빠진다, 싸우지 않는다), 플래그를 세운다.
- 패배 "진채 셋이 모두 무너짐"과 부목표 "하나도 잃지 않는다"는 그 플래그들로. 이벤트 `flag`는 전투 조건에서 바로 보인다.
- 승리 "15턴 버티거나 조인 격파"는 승리 목표 둘. `TurnLimit`은 0이거나 15보다 커야 한다.
- 관우가 9턴에 배로 온다: 이야기에서 B26V 전에 `leave`(편성에서 빠짐), 전투에서 `join`(`round>=9`). `join`은 떠나 있던 장수를 레벨을 맞춰 데려온다.
  관평·주창도 같은 `join`에 넣는다.
- 셋 다 지켜 "강릉군이 진채 밖으로 나와 쫓는다"는 `round>=13` 등에서 `ai assault`.

```json
{
  "TurnLimit": 0,
  "Areas": [
    {"ID": 40, "Rect": {"X": 4, "Y": 12, "W": 1, "H": 1}, "Name": "서쪽 진채", "Description": "적이 서면 무너짐"},
    {"ID": 41, "Rect": {"X": 10, "Y": 12, "W": 1, "H": 1}, "Name": "가운데 진채", "Description": "적이 서면 무너짐"},
    {"ID": 42, "Rect": {"X": 16, "Y": 12, "W": 1, "H": 1}, "Name": "동쪽 진채", "Description": "적이 서면 무너짐"},
    {"ID": 43, "Rect": {"X": 10, "Y": 29, "W": 1, "H": 1}, "Name": "강릉성", "Description": "무너진 진채의 수비가 물러나는 곳. 적이 서면 패배"},
    {"ID": 44, "Rect": {"X": 0, "Y": 0, "W": 20, "H": 1}, "Name": "북쪽 끝", "Description": "13턴 뒤 조인군이 물러나는 곳"}
  ],
  "Armies": [{"Faction": "shu", "Name": "유비군", "Ally": true}, {"Faction": "jing", "Name": "강릉군", "Ally": true}, {"Faction": "wei", "Name": "조조군"}],
  "Party": [
    {"Character": "troop_archer", "X": 3, "Y": 12, "AI": true, "Group": "camp-w", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_archer", "X": 5, "Y": 12, "AI": true, "Group": "camp-w", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_footman", "X": 4, "Y": 11, "AI": true, "Group": "camp-w", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_archer", "X": 9, "Y": 12, "AI": true, "Group": "camp-c", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_archer", "X": 11, "Y": 12, "AI": true, "Group": "camp-c", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_footman", "X": 10, "Y": 11, "AI": true, "Group": "camp-c", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_archer", "X": 15, "Y": 12, "AI": true, "Group": "camp-e", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_archer", "X": 17, "Y": 12, "AI": true, "Group": "camp-e", "Policy": "hold", "Faction": "jing"},
    {"Character": "troop_footman", "X": 16, "Y": 11, "AI": true, "Group": "camp-e", "Policy": "hold", "Faction": "jing"}
  ],
  "Enemies": [
    {"Character": "caoren", "X": 10, "Y": 1, "Rank": "boss", "Policy": "assault"},
    {"Character": "troop_cavalry", "X": 9, "Y": 1, "Group": "wei-main", "Policy": "assault"},
    {"Character": "troop_footman", "X": 11, "Y": 1, "Group": "wei-main", "Policy": "assault"}
  ],
  "Victory": [{"If": ["round>=16"], "Text": "15턴을 버틴다"}, {"If": ["defeated:caoren"], "Text": "조인을 물리친다"}],
  "Defeat": [
    {"If": ["flag:flag-b26v-camp-w", "flag:flag-b26v-camp-c", "flag:flag-b26v-camp-e"], "Text": "진채 셋이 모두 무너진다"},
    {"If": ["at:enemy:43"], "Text": "적이 강릉성에 선다"}],
  "Sub": [{"ID": "camps", "Text": "진채를 하나도 잃지 않는다", "If": ["!flag:flag-b26v-camp-w", "!flag:flag-b26v-camp-c", "!flag:flag-b26v-camp-e"],
           "Flag": "flag-b26v-camps", "Reward": {"insu_high": 1}}],
  "Events": [
    {"ID": "camps-stand", "Do": [
      {"Kind": "buff", "Targets": ["camp-w", "camp-c", "camp-e"], "Stat": "defense", "Value": 20},
      {"Kind": "buff", "Targets": ["camp-w", "camp-c", "camp-e"], "Stat": "range", "Value": 1}]},
    {"ID": "camp-w-falls", "If": ["any:[at:enemy:40, !alive:camp-w>=2]"], "Do": [
      {"Kind": "buff", "Targets": ["camp-w"], "Stat": "defense", "Value": -20},
      {"Kind": "buff", "Targets": ["camp-w"], "Stat": "range", "Value": -1},
      {"Kind": "ai", "Targets": ["camp-w"], "Policy": "run", "GoalArea": 43},
      {"Kind": "flag", "Flags": {"flag-b26v-camp-w": 1}},
      {"Kind": "say", "Lines": [{"Actor": "weiyan", "Text": "서쪽 진채가 무너졌다! 성으로 물러나라!", "Offstage": true}]}]},
    {"ID": "camp-c-falls", "If": ["any:[at:enemy:41, !alive:camp-c>=2]"], "Do": [
      {"Kind": "buff", "Targets": ["camp-c"], "Stat": "defense", "Value": -20},
      {"Kind": "buff", "Targets": ["camp-c"], "Stat": "range", "Value": -1},
      {"Kind": "ai", "Targets": ["camp-c"], "Policy": "run", "GoalArea": 43},
      {"Kind": "flag", "Flags": {"flag-b26v-camp-c": 1}}]},
    {"ID": "camp-e-falls", "If": ["any:[at:enemy:42, !alive:camp-e>=2]"], "Do": [
      {"Kind": "buff", "Targets": ["camp-e"], "Stat": "defense", "Value": -20},
      {"Kind": "buff", "Targets": ["camp-e"], "Stat": "range", "Value": -1},
      {"Kind": "ai", "Targets": ["camp-e"], "Policy": "run", "GoalArea": 43},
      {"Kind": "flag", "Flags": {"flag-b26v-camp-e": 1}}]},
    {"ID": "foot-waves", "Every": 3, "If": ["turnstart:enemy", "!round>=13"], "Do": [
      {"Kind": "spawn", "Spawns": [
        {"Character": "troop_footman", "X": 9, "Y": 0, "Side": "enemy", "Group": "wei-main", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_footman", "X": 10, "Y": 0, "Side": "enemy", "Group": "wei-main", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_footman", "X": 11, "Y": 0, "Side": "enemy", "Group": "wei-main", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_footman", "X": 12, "Y": 0, "Side": "enemy", "Group": "wei-main", "Policy": "assault", "Shift": -0.07}]}]},
    {"ID": "guanyu-sails", "If": ["round>=9", "turnstart:ally"], "Do": [
      {"Kind": "join", "Spawns": [{"Character": "guanyu", "X": 19, "Y": 14, "Facing": "SW"}, {"Character": "guanping", "X": 19, "Y": 15, "Facing": "SW"}]},
      {"Kind": "say", "Lines": [{"Actor": "guanyu", "Text": "형님, 늦었습니다. 강하의 배 삼백 척입니다!"}]}]},
    {"ID": "caoren-turns", "If": ["round>=13", "turnstart:enemy"], "Do": [
      {"Kind": "ai", "Targets": ["caoren", "wei-main"], "Policy": "flee", "GoalArea": 44}]}
  ]
}
```

## 8. 피난민 — B25V 장판파(가상)

- 피난민 무리 여섯은 `Party`의 AI 아군(`AI: true`), `Policy: "run"`(걷기만 하고 싸우지 않음, 가다 막히면 기다림), `GoalArea`가 다리 동쪽 진영.
  닿으면 맵에서 빠지고 `fled:refN`이 참이 된다. 반격도 못 하게 하려면 사거리 0인 `villager`(병종 `commoner`)를 쓴다(12장 B12는 `troop_commoner_*`).
- 무리마다 `Group`을 따로(`ref1`~`ref6`) 둬야 "하나라도 잃으면 패배"를 쓸 수 있다. 패배는 무리마다 `["defeated:refN", "!fled:refN"]`
  (빠져나간 무리는 `defeated`도 참이라 `!fled`로 가른다). 승리는 `fled:ref1` … `fled:ref6`을 한 목표에.
- 걷기 3칸은 개전 `buff` `speed` −1(병종 이동 4).
- 적이 "장수보다 피난민을 먼저" 노리게 하는 정책은 없다. AI는 맞받아치지 못하는 피난민을 점수로 고르는 일이 많지만 보장하지는 않는다.
- 매 턴 서쪽에서 셋씩 오는 호표기는 `Every: 1`.

```json
{
  "Max": 8,
  "Required": ["liubei", "zhangfei", "zhaoyun"],
  "Areas": [
    {"ID": 50, "Rect": {"X": 17, "Y": 14, "W": 3, "H": 4}, "Name": "다리 동쪽 진영", "Description": "피난민이 닿으면 건넌 것. 적이 서면 패배"}
  ],
  "Party": [
    {"Character": "villager", "X": 1, "Y": 10, "AI": true, "Group": "ref1", "Policy": "run", "GoalArea": 50, "Faction": "shu"},
    {"Character": "villager", "X": 1, "Y": 11, "AI": true, "Group": "ref2", "Policy": "run", "GoalArea": 50, "Faction": "shu"},
    {"Character": "villager", "X": 1, "Y": 12, "AI": true, "Group": "ref3", "Policy": "run", "GoalArea": 50, "Faction": "shu"},
    {"Character": "villager", "X": 1, "Y": 13, "AI": true, "Group": "ref4", "Policy": "run", "GoalArea": 50, "Faction": "shu"},
    {"Character": "villager", "X": 1, "Y": 14, "AI": true, "Group": "ref5", "Policy": "run", "GoalArea": 50, "Faction": "shu"},
    {"Character": "villager", "X": 1, "Y": 15, "AI": true, "Group": "ref6", "Policy": "run", "GoalArea": 50, "Faction": "shu"}
  ],
  "Enemies": [{"Character": "caochun", "X": 0, "Y": 20, "Rank": "commander", "Policy": "assault"}],
  "Victory": [{"If": ["fled:ref1", "fled:ref2", "fled:ref3", "fled:ref4", "fled:ref5", "fled:ref6"], "Text": "피난민 여섯 무리가 모두 다리를 건넌다"}],
  "Defeat": [
    {"If": ["defeated:ref1", "!fled:ref1"], "Text": "피난민이 한 무리라도 쓰러진다"},
    {"If": ["defeated:ref2", "!fled:ref2"]}, {"If": ["defeated:ref3", "!fled:ref3"]},
    {"If": ["defeated:ref4", "!fled:ref4"]}, {"If": ["defeated:ref5", "!fled:ref5"]},
    {"If": ["defeated:ref6", "!fled:ref6"]},
    {"If": ["defeated:zhangfei"], "Text": "장비 퇴각"}, {"If": ["defeated:zhaoyun"], "Text": "조운 퇴각"},
    {"If": ["at:enemy:50"], "Text": "적이 진영에 선다"}],
  "Events": [
    {"ID": "refugees-slow", "Do": [
      {"Kind": "buff", "Targets": ["ref1", "ref2", "ref3", "ref4", "ref5", "ref6"], "Stat": "speed", "Value": -1}]},
    {"ID": "tiger-riders", "Every": 1, "If": ["turnstart:enemy"], "Do": [
      {"Kind": "spawn", "Spawns": [
        {"Character": "troop_cavalry", "X": 0, "Y": 8, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_cavalry", "X": 0, "Y": 9, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_cavalry", "X": 0, "Y": 10, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07}]}]},
    {"ID": "wenpin-4", "If": ["round>=4", "turnstart:enemy"], "Do": [
      {"Kind": "spawn", "Spawns": [{"Character": "wenpin", "X": 0, "Y": 12, "Rank": "commander", "Side": "enemy", "Policy": "assault"}]}]},
    {"ID": "ford-riders", "If": ["round>=5", "turnstart:enemy"], "Do": [
      {"Kind": "spawn", "Spawns": [
        {"Character": "troop_cavalry", "X": 2, "Y": 28, "Side": "enemy", "Group": "ford", "Policy": "march", "GoalArea": 50},
        {"Character": "troop_cavalry", "X": 3, "Y": 28, "Side": "enemy", "Group": "ford", "Policy": "march", "GoalArea": 50}]}]}
  ]
}
```

패배 목표의 `Text`는 선택이라 첫 줄에만 적었다(브리핑에 같은 줄이 여섯 번 나오지 않게).

## 9. 동맹 수군 — B27V 양양 공방전

- `Boats: true`면 물을 배로 다닌다. 동맹은 `Party`의 AI 아군에 진영색 `wu`, `Armies`에 `{"Faction": "wu", "Ally": true}`.
  동맹은 플레이어 차례 뒤 `support` 차례에 스스로 움직인다. 패배 "주유 퇴각"은 `defeated:zhouyu`.
- 진영색은 조건 대상이 아니므로 동맹 수군 전체를 가리킬 때는 `Group`(`navy`)을 준다. "아군·동맹이 관청 칸에 선다"는 `at:ally:관청`
  (`ally`는 플레이어 장수와 동맹 AI 모두).
- 감녕이 채중·채화를 한꺼번에 벤다: `adjacent:ganning:caizhong` → `remove`. 유기의 뱃머리: `at:navy:수문` → 수문 수군 둘(`Group`) `remove`.
- 진채 넷은 7절과 같다. "서쪽 둘이 무너지면 조조군이 서문으로 달린다"는 `["flag:…-w1", "flag:…-w2"]` → `ai march GoalArea 서문`.

```json
{
  "Boats": true,
  "Armies": [{"Faction": "shu", "Name": "유비군", "Ally": true}, {"Faction": "wu", "Name": "손권군", "Ally": true},
             {"Faction": "jing", "Name": "양양군"}, {"Faction": "wei", "Name": "조조군"}],
  "Areas": [
    {"ID": 60, "Rect": {"X": 10, "Y": 18, "W": 1, "H": 1}, "Name": "관청", "Description": "아군·동맹이 서면 양양이 떨어짐"},
    {"ID": 61, "Rect": {"X": 6, "Y": 18, "W": 1, "H": 1}, "Name": "서문", "Description": "조조군이 서면 포위가 풀려 패배"},
    {"ID": 62, "Rect": {"X": 14, "Y": 16, "W": 2, "H": 2}, "Name": "수문 앞", "Description": "동맹 수군이 닿으면 유기가 이름을 밝힌다"}
  ],
  "Party": [
    {"Character": "zhouyu", "X": 17, "Y": 2, "AI": true, "Rank": "commander", "Group": "navy", "Policy": "assault", "Faction": "wu"},
    {"Character": "ganning", "X": 18, "Y": 2, "AI": true, "Group": "navy", "Policy": "assault", "Faction": "wu"},
    {"Character": "liuqi", "X": 19, "Y": 3, "AI": true, "Group": "navy", "Policy": "assault", "Faction": "wu"},
    {"Character": "troop_archer", "X": 17, "Y": 3, "AI": true, "Group": "navy", "Policy": "assault", "Faction": "wu"}
  ],
  "Enemies": [
    {"Character": "caimao", "X": 15, "Y": 8, "Rank": "commander", "Policy": "assault", "Faction": "jing"},
    {"Character": "caizhong", "X": 14, "Y": 8, "Group": "cai", "Policy": "assault", "Faction": "jing"},
    {"Character": "caihua", "X": 16, "Y": 8, "Group": "cai", "Policy": "assault", "Faction": "jing"},
    {"Character": "troop_archer", "X": 15, "Y": 17, "Group": "sluice-turn", "Policy": "hold", "Faction": "jing"},
    {"Character": "caoren", "X": 10, "Y": 19, "Rank": "boss", "Policy": "hold", "Faction": "wei"},
    {"Character": "xuchu", "X": 1, "Y": 18, "Group": "cao-west", "Policy": "assault", "Faction": "wei"}
  ],
  "Victory": [{"If": ["defeated:caoren"], "Text": "조인을 물리친다"}, {"If": ["at:ally:60"], "Text": "아군이나 동맹이 관청에 선다"}],
  "Defeat": [{"If": ["at:cao-west:61"], "Text": "조조군이 서문에 선다"}, {"If": ["defeated:zhouyu"], "Text": "주유 퇴각"}],
  "Events": [
    {"ID": "ganning-cuts", "If": ["any:[adjacent:ganning:caizhong, adjacent:ganning:caihua]"], "Do": [
      {"Kind": "say", "Lines": [{"Actor": "ganning", "Text": "채씨 형제냐. 한 번에 가라!"}]},
      {"Kind": "remove", "Targets": ["cai"]}]},
    {"ID": "liuqi-calls", "If": ["at:navy:62"], "Do": [
      {"Kind": "say", "Lines": [{"Actor": "liuqi", "Text": "유경승의 맏아들 유기다! 아버님의 군사들은 창을 내려라!"}]},
      {"Kind": "remove", "Targets": ["sluice-turn"]}]}
  ]
}
```

## 10. B26 연의 장판파 — 조운 혼자 시작, 호표기 3턴마다 넷, 곧장 건넜는지

- 조운은 칸 하나짜리 무리(1절). 본군 무리는 다리 어귀.
- "곧장 뚫었다"는 남쪽 숲길(구역 71)에 조운이 들어서면 플래그를 세우고, 부목표를 `!flag:`로.
- 본군이 다리를 건너 마중 나가는 것은 막지 않는다(결정 D4).

```json
{
  "Max": 8,
  "Required": ["liubei", "zhangfei", "zhaoyun"],
  "Areas": [
    {"ID": 70, "Rect": {"X": 18, "Y": 10, "W": 2, "H": 4}, "Name": "다리 동쪽 진영", "Description": "조운이 닿으면 승리, 적이 서면 패배"},
    {"ID": 71, "Rect": {"X": 0, "Y": 26, "W": 20, "H": 6}, "Name": "남쪽 숲길", "Description": "조운이 들어서면 돌아간 것"},
    {"ID": 72, "Rect": {"X": 1, "Y": 1, "W": 1, "H": 1}, "Name": "무너진 마을", "Description": "조운 혼자"},
    {"ID": 73, "Rect": {"X": 14, "Y": 10, "W": 3, "H": 3}, "Name": "다리 어귀", "Description": "본군"}
  ],
  "Allies": [{"X": 1, "Y": 1}, {"X": 14, "Y": 10}, {"X": 15, "Y": 10}, {"X": 16, "Y": 10}, {"X": 14, "Y": 11}, {"X": 15, "Y": 11},
             {"X": 16, "Y": 11}, {"X": 15, "Y": 12}],
  "Recommend": {"zhaoyun": {"X": 1, "Y": 1}, "liubei": {"X": 15, "Y": 11}, "zhangfei": {"X": 16, "Y": 10}},
  "Groups": [
    {"Name": "조운", "Area": 72, "Required": ["zhaoyun"]},
    {"Name": "본군", "Area": 73, "Required": ["liubei", "zhangfei"]}
  ],
  "Enemies": [
    {"Character": "zhanghe", "X": 8, "Y": 14, "Rank": "commander", "Policy": "react"},
    {"Character": "caochun", "X": 6, "Y": 4, "Rank": "commander", "Group": "tiger", "Policy": "assault"}
  ],
  "Victory": [{"If": ["at:zhaoyun:70"], "Text": "조운이 다리 동쪽 진영에 닿는다"}],
  "Defeat": [{"If": ["defeated:zhaoyun"], "Text": "조운 퇴각"}, {"If": ["defeated:zhangfei"], "Text": "장비 퇴각"},
             {"If": ["at:enemy:70"], "Text": "적이 진영에 선다"}],
  "Sub": [{"ID": "straight", "Text": "조운이 들판을 곧장 뚫고 장판교를 건넌다", "If": ["!flag:flag-b26-detour"], "Flag": "flag-b26-straight"}],
  "Events": [
    {"ID": "detour", "If": ["at:zhaoyun:71"], "Do": [{"Kind": "flag", "Flags": {"flag-b26-detour": 1}}]},
    {"ID": "tiger-riders", "Every": 3, "If": ["turnstart:enemy"], "Do": [
      {"Kind": "spawn", "Spawns": [
        {"Character": "troop_cavalry", "X": 0, "Y": 6, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_cavalry", "X": 0, "Y": 7, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_cavalry", "X": 0, "Y": 8, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07},
        {"Character": "troop_cavalry", "X": 0, "Y": 9, "Side": "enemy", "Group": "tiger", "Policy": "assault", "Shift": -0.07}]}]}
  ]
}
```

## 11. 없는 것(엔진에 넣지 않음)

- 적의 표적 고르기 정책(피난민 먼저 노리기). AI 점수에 맡긴다.
- 땅을 잠깐 빛내는 연출 동작. `cutscene`의 `face`·`say`로 대신.
- 구역을 드나드는 유닛을 따라가는 오라 버프. 무리에 거는 `buff`로 대신(진의 군사는 제자리를 지킨다).
- 숨어 있다 드러나는 유닛. 복병은 처음부터 서 있거나 조건이 되면 `spawn`(적)·`join`(플레이어블).
- 출전 준비에서 특정 장수를 못 고르게 하는 규칙. 그 전투에 없을 장수는 이야기 `leave`로 빼 둔다.

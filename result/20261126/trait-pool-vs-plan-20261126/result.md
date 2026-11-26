# 유벽·유비 병종 선행 특성 확인

- 사용자의 지적대로 `bandit`(적병)의 학습 가능 Pool에 `earth`(지계)가 있고 `heavy_infantry`(중보병)의 Pool에 `deputy_appoint`(부관임명)이 있다. 승급 계열의 학습 가능 특성은 상위 병종에도 상속된다.
- Pool은 플레이어가 특성치로 배울 수 있는 목록이다. NPC가 해당 특성을 자동으로 갖는다는 뜻은 아니다. NPC 출현 때는 `Class.Plan`, 개인 `Traits` 순서 및 `TraitAt`으로 `PlannedTraits`가 별도 결정한다.
- 기존 유벽 개인 계획은 mountain_footing → earth_boost여서 16레벨에 지계강화가 주어지지만, 적병/흉적/의적의 자동 계획에서 지계는 30레벨이다. 기존 유비 개인 계획은 exhort → chivalry → benevolence여서 31레벨에 용인술을 얻지만, 중보병·근위병 자동 계획에는 부관임명이 없다.
- 따라서 플레이어 학습 가능 목록 기준으로는 이미 병종에 선행이 있는데 직전 답변에서 이 점을 구분하지 않고 ‘없어서 추가’처럼 말한 것은 부정확했다. 직전 추가는 NPC 자동 계획의 시점도 검사하여 선행이 먼저 있도록 한 조치다. 해당 조치는 개인 특성과 병종 학습 목록의 중복을 만들고, 개인 특성 계획의 뒤쪽 습득 시점도 밀 수 있으므로 설계상 대안을 따로 정해야 한다.
- 근거: ver5/assets/data/classes.json, characters.json, characters_story.json, ver5/internal/content/spawn.go, ver5/internal/core/engine.go. 게임 데이터는 이번 확인에서 수정하지 않았다.

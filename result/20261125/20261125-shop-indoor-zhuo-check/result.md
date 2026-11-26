# 실내 점원과 탁현 주점 콩 판매 확인

## 결론
- 실내 점원: 새 상품 안내 수정 후 사라진 것은 추가 광고 NPC(점원#1)뿐. 실제 거래 주인·상인은 03·08·14장 상점에 남아 있고, 서주 도구점의 기존 늙은 점원도 남아 있다.
- 탁현 주점 콩: 맞다. 커밋 7dc28f1 이전에는 탁현 주인에게 `"Shop": "tavern"`이 있었고, 그 직전 shops.json의 tavern 재고는 bean(콩)·liquor(술)였다. 7dc28f1(1·2장 B안 개편, "장비의 주점은 아무것도 팔지 않고")에서 Shop 연결이 제거되어 지금은 팔지 않는다.
- 현재: explore01에는 Shop이 weapons·tools뿐이고 tools는 콩을 팔지 않는다. 낙양(03장) 주점은 콩을 팔고 외부 점원이 홍보한다.

## 변경 파일
없음 (조사만). 기존 조사 기록 result/20261125/shop-indoor-zhuo-check/summary.md를 git 이력과 현재 파일로 재확인했다.

## 검증
- git show 7dc28f1 -- explore01.json: `"Shop": "tavern"` 제거 확인.
- git show 7dc28f1^:ver5/assets/data/shops.json: tavern Items = bean, liquor 확인.
- grep explore01.json "Shop": weapons, tools만 존재 확인.
- 테스트 해당 없음(코드 변경 없는 조사).

## 선택지
탁현 주점에 콩을 다시 팔게 하려면 explore01 주인에 Shop 연결과 tavern 재고를 복원하면 된다(요청 시 진행).

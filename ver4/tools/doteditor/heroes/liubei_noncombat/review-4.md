# 유비 비전투 (`liubei_noncombat`) 검수 4

판정: **통과**

`render.py sheet liubei_noncombat --anims idle,walk,action --zoom 8`과 격자로 확인했다.
이번에 바뀐 파일은 `walk/SW-0`, `walk/NW-0` 두 개뿐이다(수정 시각 기준). 나머지 6장은 review-3 때와 같다.

- walk/SW/0: 28행 x30–33이 `rrrr` 소매 끝이 되었고 skin 픽셀이 없어졌다. 가까운 팔은 소매 끝에서 멈추고 손은 몸 뒤로 넘어간 것으로 읽힌다.
  옷 위의 주황 얼룩이 사라졌다. 한쪽만 열린 비대칭 실루엣은 그대로라 action/SW/0과 계속 구분된다.
- walk/NW/0: 31행이 x23–31로 `rrqqqrqqr` 이어진 단이 되었다. 손(29–30행 x24–25)은 단 위에 얹혀 보이고 떨어진 초록 점은 없다. 외곽선은 닫혀 있다.

## 남은 지적

없음.

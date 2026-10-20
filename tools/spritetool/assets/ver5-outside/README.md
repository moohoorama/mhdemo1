# ver5 맵 밖 장식 도트

전투 영역 밖 풀밭에 흩뿌리는 장식이다(`internal/gui/field_outside.go`). 캔버스 화소 그대로(유닛과 같은 배율) 그린다.
형식은 `ver5-fx`와 같다(레이어, 팔레트, 투명도).

- `pine`: 20×24, 앵커 (8, 22) 밑동. 프레임이 크기다(7~11, 반쪽 높이). 그림자(`s`, 투명도 0.24), 어두운 몸통(`d`), 왼쪽 밝은 반(`l`).
- `stone`: 3×2 조약돌, 위 왼쪽 두 칸이 밝다.

`seed.py pine stone`이 첫 판을 쓰고 손질은 `pixels.json`에서 한다. ver5 디렉터리에서 `uv run --with pillow python tools/build_effects.py`로
`assets/graphics/map/outside.png`와 `.json`을 굽는다.

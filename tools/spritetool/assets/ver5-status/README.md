# ver5 상태 도트

`pixels.json`이 직접 찍은 픽셀 원본이다. `.`은 완전 투명, `o`는 외곽/그림자,
`m`은 기본색, `h`는 밝은색, `r`은 부적의 붉은 문양이다.
셀·앵커·프레임 순서·팔레트는 `settings.json`을 따른다.

- 상태이상은 9×9 셀, 위쪽 중심 앵커 (4, 0), 정지 한 프레임이다.
- 화살표는 같은 셀의 (2, 1)에서 5×6 픽셀만 사용한다. 각 능력치의 위/아래 순서다.
- 독방울, 사슬, 붉은 X, 불꽃, 물음표, 부적은 어두운 1픽셀 실루엣과 2~3색으로 그린다.
- 화살표는 방향을 살리기 위해 밝은 삼각 머리와 한 픽셀 몸통을 쓴다. 사각 배경이나 별도 무기는 없다.
- 알파는 0/255만 쓰며 확대는 최근접이다. 자동 trim, 블러, 축소로 만든 도트를 사용하지 않는다.
- 게임의 HP/MP 아래에 한 행으로 배치한다. 상태가 있을 때만 바를 올려 머리 위 여백을 확보한다.

ver5 디렉터리에서 `uv run --with pillow python tools/build_status_icons.py`로
`assets/graphics/status/icons.png`와 `icons.json`을 만든다.
전용 빌더는 ver5에만 있으며 공용 spritetool 구현을 변경하지 않는다.

`SRPG_RENDER_TEST=1 go test ./internal/gui -run TestRenderedEditorAndStatuses -count=1`은
실제 `.spr` 유닛 렌더러로 상태별 1배/3배 견본을 `reports/status-pixel-art/preview.png`에 저장한다.

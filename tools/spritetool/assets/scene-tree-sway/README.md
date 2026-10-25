# 장면 나무 흔들림 원화

잎·꽃이 있는 `stree_*`·`peach_bloom_*` 나무의 비교용 4프레임 시트다. 대상별 캔버스, 뿌리 기준점, 팔레트, 외곽선 변환값은 `assets_candidate/tiles/tiles.json`과 `assets_candidate/tiles/sources/trees.json`을 따른다.

- 배치: 한 PNG에 2열×2행, 프레임 0→1→2→3.
- 재생: 프레임마다 250ms, 1000ms 주기.
- 모델: 뿌리·줄기·가지 면 고정, 잎·꽃 면만 산들바람에 움직임.
- 그림: 기존 정지 원화의 색·화풍을 따라 각 3D 자세를 한 장에 그림.
- 완성 도트: 프레임마다 같은 나무별 캔버스와 기준점, 한정 팔레트와 외곽선, 알파 0/255. 줄기·가지는 공통 프레임으로 고정.
- `stree_dead`에는 움직이는 잎이 없어 시트 없이 정지 그림을 쓴다.

각 나무 폴더에 `reference-3d.png`, `painted-sheet.png`, `dot-sheet.png`를 둔다. HTML 시안을 검토하기 전에는 게임 에셋을 바꾸지 않는다.

```sh
uv run --with numpy --with pillow python ver5/tools/render_scene_tree_sway.py
uv run --with pillow python ver5/tools/prepare_scene_tree_sway.py
```

원화 생성에는 내장 이미지 생성 도구를 사용했다. 나무마다 `reference-3d.png`를 **네 자세 참조**, 기존 `assets_candidate/tiles/sources/<이름>.png`를 **화풍·색 참조**로 주었다. 공통 요청: “같은 나무의 4프레임을 2×2로 그린다. 원래 수형과 팔레트, 왼쪽 위 빛, 손도트 질감을 유지한다. 줄기·뿌리·갈색 가지는 네 칸에서 같은 좌표, 잎·꽃만 약한 바람에 움직인다. 네 칸은 같은 크기·뿌리 자리와 넉넉한 여백을 가진다. 배경은 순수한 #FF00FF, 글자·격자·바닥·그림자·블러·그라디언트는 없다.” 나무별 묘사는 소나무의 층진 솔잎, 버드나무의 늘어진 잎, 고목의 불규칙한 수관, 낮은 덤불, 흰 매화, 분홍 복숭아꽃을 각각 지정했다.

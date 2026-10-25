# 나무 움직임 비교 실험

게임에 적용하지 않는 4프레임·250ms 비교 시안이다. 공통 제작 규칙은 `SPRITE_ART_GUIDE.md`와 각 나무의 `assets_candidate/tiles/sources/trees.json` 설정을 따른다.

- `stree_broad_0/normal-map.png`: 기존 3D 모델의 보이는 면에서 만든 카메라 공간 법선. RGB는 법선 xyz를 0~255로 부호 변환한 값이고 알파는 원래 활엽수 도트의 실루엣이다.
- `stree_broad_0/sunlight-sheet.png`: 원래 활엽수 도트의 위치를 유지하고 법선과 조금씩 다른 햇살 방향으로 팔레트 명암만 바꾼 2×2 시트.
- `stree_willow/reference-3d.png`: 기존 버드나무 진폭 0.12의 ⅓인 0.04로 잎 면을 변형한 3D 렌더 2×2 시트. 줄기·가지는 고정한다.
- `stree_willow/dot-sheet.png`: 3D 렌더를 기존 버드나무 팔레트와 외곽선으로 줄인 2×2 시트.

```sh
uv run --with numpy --with pillow python ver5/tools/tree_motion_experiments.py
```

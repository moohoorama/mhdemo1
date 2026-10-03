# OpenCV 진단 및 자동 복원 제안 — 실험용

이 폴더는 현재 런타임으로 선택하지 않았다. `knight-cv-review.html`에서
현재 v6와 비교할 수 있다. 자동 인식은 실제 장비와 투구 하이라이트를 혼동할 수 있어
매칭 점수만으로 최종 수정이라고 판단하지 않는다.

OpenCV 5.0.0: HSV 색상 분할, connectedComponentsWithStats, findContours,
floodFill, 마스크를 사용하는 matchTemplate, 제한된 최근접 회전 및 국소 inpaint.
160프레임의 검출값·물체 위치·수정 제안은 `analysis.json`에 보관한다.
공격 궤적과 다리 사이 공간은 전역 morphological closing으로 메우지 않는다.
템플릿에 맞지 않는 인식은 버리고 걷기 투구 영역은 보호한다.

```sh
python3 -m venv /tmp/knight-opencv-env
/tmp/knight-opencv-env/bin/pip install -r tools/requirements-knight-cv.txt
/tmp/knight-opencv-env/bin/python3 tools/repair_knights_cv.py
```

API 참고: https://docs.opencv.org/4.x/d3/dc0/group__imgproc__shape.html
가이드: ../SPRITE_ART_GUIDE.md

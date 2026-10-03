보병 · 궁병 · 기병 애니메이션 (최신 수정본)

실행: Python 3 설치 후 압축을 푼 폴더에서
python -m http.server 8000 --directory dist
브라우저에서 http://localhost:8000 접속

보병 8방향, 궁병·기병 좌하단 스프라이트와 3D 모델 포함.
보병 발 동작 개선, 궁병·기병 피격 3·4프레임 수정 포함.
dist/retouched: 보병 20프레임 리터칭 WebP
dist/units: 궁병·기병 20프레임 리터칭 WebP 및 3D 포즈 JSON
dist/sprites: 보병 3D 픽셀 렌더
tools: 모델 생성 및 렌더링 Python 소스 (numpy, Pillow 필요)

리터칭 이미지의 체크무늬 배경은 실제 투명 배경이 아닙니다.

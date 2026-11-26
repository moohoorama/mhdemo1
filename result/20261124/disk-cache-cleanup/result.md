# 디스크 캐시 정리 결과 (진행 중)

## 요청

Go, uv, Chrome 캐시와 `Downloads/1993-exe` 삭제.

## 확인한 것

- Go 빌드 캐시 `~/Library/Caches/go-build`: 12K. 이미 비어 있다(다른 세션이 정리한 것으로 보임).
- uv 캐시 `~/.cache/uv`: 1.7G. `uv cache clean`을 백그라운드로 실행 중이다(작업 `bd1jafg7r`). 완료 여부는 아직 확인하지 못했다.
- Chrome 캐시: `~/Library/Caches/Google*`, `com.google.Chrome*`에 해당 경로가 없었다. 삭제하지 않았다.
- `~/Downloads/1993-exe`: 존재하지 않는다. 삭제할 대상이 없다.
- 디스크: 사용률 56%, 여유 약 13 GiB(정리 전 94%, 여유 1.3 GiB에서 크게 늘었다).

## 남은 일

- uv 캐시 정리 완료 확인.
- Chrome 캐시 위치가 다른 곳이면 위치를 알려 주면 정리.
- `1993-exe`가 다른 이름이나 위치라면 경로를 알려 주면 확인 후 삭제.

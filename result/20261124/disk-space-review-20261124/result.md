# 디스크 공간 점검 결과

## 현재 상태

- 전체 228Gi 중 17Gi 사용, 여유 1.3Gi, 사용률 94%.
- 삭제는 하지 않았다. 아래는 후보와 크기다.

## 삭제 후보 (크기 큰 순)

| 경로 | 크기 | 성격 | 비고 |
|---|---|---|---|
| `~/Library/Caches/go-build` | 8.2G | Go 빌드 캐시 | 재생성 가능. `go clean -cache`로 비움 |
| `~/Library/Caches` 전체 | 16G | 앱 캐시 | go-build 포함. 나머지는 앱별 확인 필요 |
| `~/.cache` | 5.0G | uv·pip 등 도구 캐시 | 재생성 가능 |
| scratchpad `exp/` | 3.6G | 실험용 APFS 복제 | 실험 결과는 커밋·result에 있음. 복제본 |
| scratchpad `laya/` | 807M | Laya 가중치 | 실험 B를 이어 할 때 필요 |
| scratchpad `snap/` | 636M | HEAD 스냅샷 | 미사용 |
| scratchpad `head/` | 630M | 스냅샷 추정 | 용도 확인 필요 |
| scratchpad `venvb/` | 530M | 부분 설치 실패한 venv | 미사용 |

scratchpad 전체는 6.2G다.

## 참고

- 저장소 `.git`은 1.5G, `result/`는 267M. 이 둘은 지우지 않는다.
- 가장 안전하고 효과가 큰 것은 Go 빌드 캐시 8.2G다. 다음으로 `~/.cache`와 scratchpad 복제본(`exp/`, `snap/`, `venvb/`)이다.

## 확인 필요

- 삭제 허락 여부. 허락을 주면 항목별로 지운다. `laya/`는 B를 이어 할지 정한 뒤 결정한다.

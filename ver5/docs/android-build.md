# Android APK 빌드 설계

기존 데스크톱 GUI와 규칙 엔진을 재사용한다. `mobile` 패키지는 Android의
앱 전용 경로로 콘텐츠·폰트·세이브를 초기화하고 Ebitengine `mobile.SetGame`에
GUI를 전달한다. Android Activity는 APK에 포함된 런타임 에셋을 앱 내부에
풀고, EbitenView의 시작·중단·재개와 뒤로 가기를 담당한다. 세이브는 에셋과
별도의 디렉터리에 저장한다. 에셋 묶음이 변경된 업데이트에서는 배포 에셋을
교체하므로 앱 내부 맵 편집은 다음 업데이트에서 덮어쓸 수 있다.

입력은 기존 GUI 버튼·카메라·에디터가 공유하는 포인터 함수를 통해 연결한다.
한 손가락은 클릭/드래그, 두 손가락은 이동/핀치 확대이며 Android 뒤로 가기는
Escape와 같다. 데스크톱 마우스와 키보드 입력은 기존 동작을 유지한다.

`tools/build_android.py`는 프로젝트에 고정된 Ebitengine 버전으로 AAR을
만들고, Android SDK의 aapt2·javac·d8·zipalign·apksigner로 테스트 APK를
패키징한다. Gradle/Android Studio 프로젝트 없이 실행할 수 있다.
기본 ABI는 arm64-v8a와 x86_64, 최소 Android 버전은 API 23이다.
앱은 가로 방향으로 실행하며, 인터넷/외부 저장소 권한을 요구하지 않는다.
빌드 산출물과 테스트 서명 키는 `bin/android/`에 두고 Git에서 제외한다.

검증: 터치 상태 전환 단위 테스트, 관련 Go 패키지 테스트, APK 서명 검사,
에뮬레이터에서 시작 메뉴·새 게임·일기토 실행 및 터치/뒤로 가기 확인.
실제 휴대폰의 성능과 모든 캠페인 진행은 별도 검수가 필요하다.

## 검증 결과 (2026-10-13)

- `go test ./internal/gui ./internal/content ./internal/storage ./internal/session ./mobile`
  및 `go vet ./internal/gui ./mobile` 통과. 데스크톱 GUI 빌드도 통과.
- `python3 tools/build_android.py`로 약 20.2 MiB 테스트 APK 생성.
  APK v1/v2/v3 서명 검사 통과, ARM64/x86_64 ELF LOAD 정렬 16KB 확인.
- Android 16(API 36) ARM64 에뮬레이터에 설치·실행. 한글·초상·스프라이트,
  터치로 새 게임/일기토 감상, Android 뒤로 가기 확인.
- 터치로 슬롯 1 저장 후 같은 서명 APK 재설치. 세이브 유지와 불러오기 확인.
- 화면 증거는 로컬 `reports/android/`에 보관한다(Git 제외).
- 핀치 상태 전환은 단위 테스트로 검증했다. 실제 기기 멀티터치와 전체 캠페인,
  x86_64 실행은 아직 검수하지 않았다.

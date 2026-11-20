# Windows 앱 아이콘

- Android와 같은 원본 그림 `assets/graphics/ui/app_icon.png`를 여러 크기의 Windows ICO로 변환한다.
- ICO를 Windows x64 실행 파일의 리소스에 포함하고 NSIS 설치·제거 프로그램에도 지정한다. 앱 바로가기는 게임 실행 파일의 아이콘을 사용한다.
- release 설치 파일을 다시 빌드하고 PE 리소스와 설치 파일 구성·무결성을 확인한다.

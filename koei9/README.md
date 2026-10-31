# 삼국지 9 타일 지도 참고 자료

전체 지도와 도시·지역 목록을 보면서 직접 타일 맵을 그리기 위한 자료 묶음이다. 수집일: 2026-10-30.

## 시작하기

1. `images/`의 전체 지도 이미지를 열어 지형과 도시의 상대 위치를 본다. 다운로드 결과·해상도·출처는 `sources.json`에 기록한다.
2. [tile-guide.md](tile-guide.md)의 순서대로 육지·물·산·길·거점을 나눠 그린다.
3. [cities.csv](cities.csv) 또는 [cities.json](cities.json)을 거점 체크리스트로 쓴다. 일본어 원문 지명을 유지하고 도시·주의 한국어 독음을 덧붙였다.
4. 원작의 논리 격자와 통행 속성은 [map-data.md](map-data.md)를 참고한다.

## 포함 범위

- 일반 도시 45곳과 이민족 거점 5곳의 주·도시·관련 지역 목록.
- 전체 지도 이미지와 원본 다운로드 출처.
- 모딩 제작자가 분석한 200×200 논리 격자, Square 번호와 지형·통행 속성의 한국어 요약.
- 타일 제작을 위한 레이어 구성과 작업 순서 제안.

## 저장한 이미지

| 파일 | 크기 | 용도 |
|---|---|---|
| [images/official-map.jpg](images/official-map.jpg) | 3,200×1,650 | 공식 전체 지도, 큰 지형과 도시 배치 |
| [images/fan-01.jpg](images/fan-01.jpg) | 880×495 | 낙양 주변 부분 확대 예시, 시설 이름과 경계 표시 |
| [images/fan-02.jpg](images/fan-02.jpg) | 880×442 | 팬 제작 전체 지도 축소 미리보기 |

12,800×6,432px 원본의 [MAP_SAN9.rar 다운로드 페이지](https://www.mediafire.com/file/wj9t7s1ycq8oi73/MAP_SAN9.rar/file)는 확인했으나 다운로드 서버 연결이 실패해 원본 압축파일은 저장하지 못했다. 주소와 실패 사유를 `sources.json`에 남겼다. 위 세 이미지는 내려받아 실제 크기와 화면을 확인했다.

**원작의 40,000칸 지형 덤프나 정확한 도시 좌표는 확보하지 않았다.** 도시의 `x`, `y`는 JSON에서 `null`, CSV에서 빈칸이다. 목록에 나오는 지역이 모두 건물이거나 항구인 것은 아니다. 원본 이미지의 해상도와 논리 격자의 크기는 별개이며, 이미지 전체를 200등분한 값이 실제 게임 타일에 맞는다는 보장은 없다.

CSV는 엑셀에서 한국어가 깨지지 않도록 UTF-8 BOM으로 저장했다. JSON의 `id`는 자료 정리용 번호이며 원작 게임 ID가 아니다. 주변 지역명은 원문의 표기를 그대로 보존했으므로 오기 가능성이 있는 지명은 별도 대조해야 한다. `regions_ja`가 빈 배열이면 출처 표의 공란을 뜻한다.

## 출처

- [코에이 공식 제품 설명](https://www.gamecity.ne.jp/products/products/ee/new/san9/san9_top.htm): 전토를 한 장으로 표현하고 전략·전투를 통합했다는 공식 설명.
- [코에이 공식 전체 지도](https://www.gamecity.ne.jp/products/products/ee/new/san9/san9_map.htm): 지도 이미지.
- [艾克氏遊戲人間, 三國志九：大地圖一覽](https://xgodgame.blogspot.com/2018/05/blog-post.html): 2018-05-01 공개, 12,800×6,432px 지도와 다운로드 링크.
- [三国志９攻略情報局, 都市データ](https://sangokushi9.gamefan.info/city/): with 파워업키트 기준 주·도시·지역 표.
- [vonleibniz의 맵 수정 연구](https://game.ali213.net/forum.php?action=printable&mod=viewthread&tid=3635680): 2011년 모딩 경험에 따른 분석. 공식 파일 사양은 아니다.

이미지와 원문 자료의 권리는 각 원저작자에게 있다. 다운로드는 제작 참고용 보관이며 재배포·상용 이용 허락을 뜻하지 않는다. 직접 그리는 타일의 설계 제안은 원작 데이터와 구분했다.

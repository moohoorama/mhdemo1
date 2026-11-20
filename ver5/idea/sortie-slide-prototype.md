# 출전 준비 슬라이딩 시안 — 구현 보관

2026-11-19 구현 후 사용자가 보류하고 원래 화면으로 되돌리기로 한 가안. 아래는 당시 구현의 기록이며 현재 게임에 적용되어 있지 않다.

## 구성

- 편성 보기: 출전 부대·전체 목록·상세 윤곽을 3:6:1로 배치. 출전 부대는 남서쪽 병종 도트로 3×6, 복병은 칸과 출전 인원에서 제외.
- 상세 보기: 출전 윤곽·이름 목록·상세를 1:2:7로 배치. 목록 이름으로 장수 선택. 열전·특성·장비·부관 탭과 소지품 창을 사용.
- 목록과 출전 부대 사이의 왼쪽 화살표로 출전, 오른쪽 화살표로 취소. 기존 군주·필수·복병·부관·스테이지 인원 제약을 사용.
- 영역 전환은 이전·다음 화면을 각각 그려 0.3초 동안 본문을 수평으로 이동하는 방식. 전환 중 입력은 막음.
- 출진 오른쪽에 닫기. 창 밖 클릭·우클릭·Esc로 닫고, 전투 노드에서 출전 준비 버튼으로 다시 엶.
- 전력은 장수 단독 공격·방어·정신·순발·사기·최대 병력 + 최대 병법치×10 + 직접 습득한 특성 비용 합. 출전 목록과 관리창 기본 표에 표시.

## 적용했던 지점과 확인

`sortie.go`가 화면·전환·입력을 담당하고 `sortie_slide.go`가 격자·표·상세·전력 계산을 제공했다. `party.go`에 화면 상태를 두었고, `game.go`에 취소 처리를 추가했다. 자동 진행·기존 데모·상세 능력 막대 간격도 새 배치에 맞췄다.

`TestSortiePower`, `TestSortieSlideControls`, `TestSortieDeputy`, `TestStartSortie`, `TestGearGuide`와 자동 진행 버튼 관련 시험을 통과했다. `sortie-slide` 데모로 휴대폰 크기 두 화면과 전환을 캡처했다. 시험 통과는 디자인이 확정되었다는 뜻은 아니다.

아래에는 당시 코드 변경과 새 파일을 함께 보관한다. 기존 작업 트리의 다른 변경은 포함하지 않는다. 이 기록을 그대로 재적용하기 전에 그 시점의 코드와 다시 대조한다.

## 추적 파일의 당시 변경

```diff
diff --git a/ver5/docs/presentation-ui.md b/ver5/docs/presentation-ui.md
index 35cd68d..50e4bd7 100644
--- a/ver5/docs/presentation-ui.md
+++ b/ver5/docs/presentation-ui.md
@@ -238,20 +238,14 @@ AI 진영(적·동맹)의 차례는 규칙이 한 번에 다 정하고(`ResolveP
 
 ## 출전 준비
 
-스토리의 전투 노드에 닿으면 열린다(`sortie.go`). 강제 출전 스테이지(`Stage.Forced`)는 이 화면 없이 바로 배치로 간다.
-
-- 왼쪽: 장수 줄(56). 맨 앞 상태 칸(64×40, 누르는 범위 줄 높이)이 `출전`·`필수`·`부관`(아래 줄에 상관 이름)·`대기`를 보이고, 누르면 출전 ↔ 대기(군주·필수 장수는 그대로, 부관은 출전할 수 없음, 인원이 차면 알림).
-  장수 줄은 두 줄: 이름(16)과 오른쪽 `Lv`(12) / 병종(12, 빠진 장비 아이콘이 그 뒤)과 오른쪽 `특성치`(12, 배울 것 있으면 초록).
-  머리에는 `출전 준비`와 `출전 n / m`만, `목표`·`메뉴`는 목록 밑(할 일 줄 높이)에 둔다.
-- 복병(`Stage.Reserve`): 줄의 상태 칸이 `복병`(금색), 줄의 `특성치` 자리에 `복병 · 전투 중 출현`이 보이고, 누르면 "복병은 전투 중에 나타나 반드시 출전합니다" 알림만 뜬다(필수와 같이 뺄 수 없다). 장비·특성 탭은 다른 장수와 같아서 장비를 준비할 수 있다. 머리의 `출전 n / m`에는 복병을 세지 않고 `(복병 제외)`를 붙인다. 배치 화면(`출전 배치 화면`)은 복병을 목록·`배치 n / m`·Tab 순환에서 뺀다. 부관 후보 표에서는 필수와 같은 이유(`필수 출전·복병 장수`)로 부관이 될 수 없다.
-- 부관 표시(장수 창 목록·출전 준비·부관 후보 표 공통, `pillDep`·`depRim`): 누구의 부관인 장수의 줄은 파란 바탕에 왼쪽 파란 띠(고르면 띠만 금색), 이름 옆 파란 `부관 · 상관` 표(출전 준비에서는 상태 칸이 대신 말함). 부관을 거느린 장수는 이름 옆 파란 `부관 이름` 표.
-- 오른쪽: 고른 장수의 이름과 탭 넷 `열전`·`특성`·`장비`·`부관`. 할 일 줄은 탭의 행동(`부관 해임`·`○○ 임명`, `○○ 배우기`)을 왼쪽에, 조금 띄워 `소지품`·`출진 ›`을 오른쪽 끝에.
-- 부관 탭(안 4 「후보 표」): 후보를 표로(줄 48) — 후보(얼굴·이름)·지금(`대기`·`출전`·`누구 부관`, 지금 부관은 파란 `현재 부관` 표)·매력·가중치(막대와 %)·오르는 능력(가장 많이 오르는 열전 능력과 `외 n`)·특성 수.
-  머리를 누르면 정렬(처음은 가중치 ▾). 지금 부관은 맨 위에 파란 바탕·왼쪽 파란 띠, 고른 후보는 금 테두리, 임명할 수 없는 후보는 흐리게 맨 아래.
-  표 밑 카드에 고른 후보의 이름·병종·Lv·가중치, 임명하면 생기는 일 또는 못 하는 이유, 더해지는 특성을 작은 칩(30)으로 줄바꿈해 다 보인다. 탭의 절반에 안 들어가면 마지막 칩이 `+n`(누르면 다 보이고 `접기`), 칩을 누르면 특성 설명 창.
-- 짧은 화면(폰 914×411)에서 열전 탭은 얼굴을 작게 하고 병종·이동·사거리를 그 옆에, 육각형은 이름표까지 카드 안에 들게 줄이며, 장비 탭은 합계 카드를 빼고 슬롯 카드가 폭을 다 쓴다(장비 특성은 등급 옆 작은 칩).
-- `출진`은 출전 인원을 스테이지 한도로 자르고(군주·필수 먼저) 배치로 간다. 그 왼쪽 `소지품`은 장수 창을 소지품 보기로 열어 전투 전에 인수를 쓰게 한다.
-- 확인: `go run ./cmd/_uidemo-vm -ui sortie-deputy -node c20-battle -frames 225 -at 30,55,85,100,128,134,140,146,180,220 -size 2400x1080 -uiscale 207`(폰 914×411, 130%·1280×850도).
+스토리의 전투 노드에 닿으면 열린다(`sortie.go`, `sortie_slide.go`). 강제 출전 스테이지는 이 화면 없이 배치로 간다.
+
+- 편성 보기: 출전 부대·전체 부대 표·상세 열기 영역을 3:6:1로 둔다. 출전 부대는 배치 순서대로 가로 셋, 세로 여섯 칸에 남서쪽을 바라보는 병종 도트로 표시한다. 복병은 배치되지 않으므로 칸에서 빼고 출전 인원에서도 제외한다.
+- 표는 기본·전투력·열전·장비로 전환하고 행과 머리글로 선택·정렬한다. 기본의 전력은 장수 단독 공격·방어·정신·순발·사기·최대 병력과 최대 병법치의 10배, 직접 습득한 특성 비용 합이다. 관리창 기본 표에도 같은 수치를 표시한다.
+- 표 왼쪽의 `←`는 선택한 장수를 출전시키고 `→`는 출전 취소한다. 군주·필수·복병, 부관과 스테이지 인원 제한에는 기존 편성 규칙을 적용한다.
+- 오른쪽 상세 영역을 누르면 상세 보기(1:2:7)로 이동한다. 좁은 목록에서는 이름으로 다른 장수를 고르고 열전·특성·장비·부관 탭에서 살피거나 바꾼다. 왼쪽 영역을 누르면 편성 보기로 돌아간다.
+- `소지품`은 장수 창의 소지품 보기를 열고 `출진`은 배치 단계로 간다. `닫기`, 창 밖 클릭, 우클릭, Esc는 창을 닫고 `출전 준비`로 다시 연다. 닫기는 출진의 오른쪽에 둔다.
+- 확인: `go run ./cmd/_uidemo-vm -ui sortie-slide -node c14-battle -frames 130 -at 30,94,120 -size 2400x1080 -uiscale 207`.
 
 ## 부대 상세
 
diff --git a/ver5/internal/gui/campaign_auto.go b/ver5/internal/gui/campaign_auto.go
index ce23a07..4f9b4c9 100644
--- a/ver5/internal/gui/campaign_auto.go
+++ b/ver5/internal/gui/campaign_auto.go
@@ -188,8 +188,8 @@ func (g *Game) presentCampaignCommand(c core.Command) {
 		case "equip", "unequip":
 			g.party.tab, g.party.sortieTab = 2, 2
 		case "deputy":
-			g.party.sortieTab = 3
-			return // the sortie's deputy tab is already on screen
+			g.party.sortieTab, g.party.sortieDetail = 3, true
+			return // show the sortie's deputy tab
 		}
 		if g.overlay() == "" {
 			g.overlays = append(g.overlays, "party")
diff --git a/ver5/internal/gui/game.go b/ver5/internal/gui/game.go
index 86000a4..0fc740d 100644
--- a/ver5/internal/gui/game.go
+++ b/ver5/internal/gui/game.go
@@ -68,8 +68,8 @@ type Game struct {
 	textSeconds               float64   // the clock lines come out on: uiSeconds run at the replay speed
 	cursorHidden              bool
 	uiSeconds                 float64
-	moneyShown                int  // the money as the screen counts it toward the real sum
-	moneyReady                bool // moneyShown has been set from a game
+	moneyShown                int                      // the money as the screen counts it toward the real sum
+	moneyReady                bool                     // moneyShown has been set from a game
 	textCache                 map[string]*ebiten.Image // rendered text by key (ui_anim.go)
 	root                      string
 	assets                    *Assets
@@ -132,7 +132,7 @@ type Game struct {
 	itemEffects               map[string]itemEffect // itemEffect results by revision, officer, slot and piece
 	equip                     equipState
 	waypoints                 waypointList
-	jumpNode                  string // the waypoint the difficulty window starts at; none starts a new game
+	jumpNode                  string          // the waypoint the difficulty window starts at; none starts a new game
 	jumpLevel                 int             // the waypoint setup: the party's level; none is the next battle's
 	jumpFlags                 map[string]bool // the waypoint setup: the battle flags left on
 	question, yes, no         string
@@ -708,6 +708,8 @@ func (g *Game) rightClick() {
 	case g.overlay() != "":
 		g.closeOverlay()
 	case g.popupPending():
+	case g.sortieOpen():
+		g.party.sortieClosed = true
 	case g.deploying():
 		g.deploymentSelected = ""
 	case g.cancel():
@@ -894,6 +896,8 @@ func (g *Game) keys() {
 		case len(g.modals) > 0:
 			g.closeModal()
 		case len(g.learn) > 0:
+		case g.sortieOpen():
+			g.party.sortieClosed = true
 		case g.deploying() && g.deploymentSelected != "":
 			g.deploymentSelected = ""
 		case g.cancel():
diff --git a/ver5/internal/gui/officer_cards.go b/ver5/internal/gui/officer_cards.go
index fccdc0d..70aa4cf 100644
--- a/ver5/internal/gui/officer_cards.go
+++ b/ver5/internal/gui/officer_cards.go
@@ -87,7 +87,7 @@ const statBarStep = 26
 
 // statBars draws hexAxes' six stats a line each from x, y, w wide: the stat's name, its value in valueC, a bar of
 // its share from barX on and, when given, a tail in gold at the right end.
-func (g *Game) statBars(dst *ebiten.Image, x, y, w, barX float64, values, tails [6]string, valueC color.Color, shares [6]float64) {
+func (g *Game) statBars(dst *ebiten.Image, x, y, w, barX float64, values, tails [6]string, valueC color.Color, shares [6]float64, step float64) {
 	tailW := 0.0
 	for _, t := range tails {
 		if t != "" {
@@ -101,7 +101,7 @@ func (g *Game) statBars(dst *ebiten.Image, x, y, w, barX float64, values, tails
 		if tails[i] != "" {
 			g.label(dst, tails[i], x+w-g.width(tails[i], 13), y, 13, gold)
 		}
-		y += statBarStep
+		y += step
 	}
 }
 
@@ -136,12 +136,13 @@ func (g *Game) statsCard(dst *ebiten.Image, c image.Rectangle, compact bool, sta
 		shares[i] = float64(v) / float64(max(1, best[i]))
 		valueW = max(valueW, g.width(values[i], 13))
 	}
-	const head = 30
+	head := 30.0
+	step := min(float64(statBarStep), (float64(c.Dy())-head-18-16)/5)
 	w := float64(c.Dx() - 36)
-	h := head + 6*statBarStep - (statBarStep - 18)
-	x, y := float64(c.Min.X+18), float64(c.Min.Y)+max(16, float64(c.Dy()-h)/2)
+	h := head + 5*step + 18
+	x, y := float64(c.Min.X+18), float64(c.Min.Y)+max(16, (float64(c.Dy())-h)/2)
 	g.label(dst, "능력", x, y, 13, gold)
-	g.statBars(dst, x, y+head, w, 44+valueW+10, values, tails, ink, shares)
+	g.statBars(dst, x, y+head, w, 44+valueW+10, values, tails, ink, shares, step)
 }
 
 // labelSpaced draws s line by line, lead times the size apart.
@@ -367,7 +368,7 @@ func (g *Game) officerEquipment(dst *ebiten.Image, of *core.Character, r image.R
 				grades[i] = bonusGrade(total[i])
 			}
 		}
-		g.statBars(dst, x, y, sumW-36, 76, grades, none, gold, shares)
+		g.statBars(dst, x, y, sumW-36, 76, grades, none, gold, shares, statBarStep)
 		y += 6*statBarStep + 14
 		g.label(dst, "장비 특성", x, y, 13, gold)
 		y += 22
diff --git a/ver5/internal/gui/officer_window.go b/ver5/internal/gui/officer_window.go
index fc13710..5a3ddff 100644
--- a/ver5/internal/gui/officer_window.go
+++ b/ver5/internal/gui/officer_window.go
@@ -243,7 +243,7 @@ func (g *Game) officerTable(dst *ebiten.Image, o core.Observation, roster []core
 				}
 				return g.className(r.c.Class), ink
 			}},
-			num("level", "레벨", func(r *row) int { return r.c.Level }), num("hp", "병력", func(r *row) int { return r.s.MaxHP }),
+			num("level", "레벨", func(r *row) int { return r.c.Level }), num("power", "전력", func(r *row) int { return g.officerPower(r.c, r.s) }), num("hp", "병력", func(r *row) int { return r.s.MaxHP }),
 			num("mp", "병법치", func(r *row) int { return r.s.MaxMP }), num("points", "특성치", func(r *row) int { return r.c.Points }))
 	case 1:
 		for _, s := range statRows[2:] {
diff --git a/ver5/internal/gui/party.go b/ver5/internal/gui/party.go
index ba9cfdb..55e7142 100644
--- a/ver5/internal/gui/party.go
+++ b/ver5/internal/gui/party.go
@@ -20,7 +20,11 @@ type partyView struct {
 	learning     bool // opened by a level-up for the officers still learning; closing it ends their learning
 	filter, item string
 	trait        string
-	tip          string    // the equipment trait whose description shows on 장비, as "slot|trait"
+	tip          string // the equipment trait whose description shows on 장비, as "slot|trait"
+	sortieNode   string
+	sortieClosed bool
+	sortieDetail bool
+	sortieSlide  float64
 	sortieTab    int       // the sortie's tab of the picked officer (sortieTabs)
 	depLead      string    // the lead the 부관 tab last showed, so another lead starts on their own deputy
 	depAll       bool      // the traits the candidate picked brings are all shown, past the lines they get
diff --git a/ver5/internal/gui/sortie.go b/ver5/internal/gui/sortie.go
index 9aed04e..4ac9bb8 100644
--- a/ver5/internal/gui/sortie.go
+++ b/ver5/internal/gui/sortie.go
@@ -20,10 +20,75 @@ func (g *Game) sortieForced(o core.Observation) bool {
 	return g.S.Data.Stages[o.Stage].Forced
 }
 
+func (g *Game) sortieOpen() bool {
+	if g.S == nil || g.S.Engine == nil || g.party.sortieClosed {
+		return false
+	}
+	o := g.S.View()
+	return o.Phase == "scenario" && o.Dialogue.Kind == "battle" && !g.sortieForced(o)
+}
+
 // drawSortie draws the preparation over the battle node, filling the screen up to sheetMax.
 func (g *Game) drawSortie(dst *ebiten.Image, o core.Observation) {
+	p := &g.party
+	if p.sortieNode != o.Node {
+		p.sortieNode, p.sortieClosed, p.sortieDetail, p.sortieSlide = o.Node, false, false, 0
+	}
+	if p.sortieClosed {
+		w, h := sheetSize()
+		x, y := (uiW-w)/2, (uiH-h)/2
+		g.mBtn(dst, image.Rect(x+w-180, y+h-72, x+w-16, y+h-24), "출전 준비", true, func() { p.sortieClosed = false })
+		return
+	}
+	t := (g.uiSeconds - p.sortieSlide) / .3
+	if p.sortieSlide == 0 || t >= 1 || t < 0 {
+		g.drawSortieScreen(dst, o)
+		return
+	}
+	old := ebiten.NewImage(dst.Bounds().Dx(), dst.Bounds().Dy())
+	defer old.Deallocate()
+	target := ebiten.NewImage(dst.Bounds().Dx(), dst.Bounds().Dy())
+	defer target.Deallocate()
+	buttons, wheels, blocks := len(g.buttons), len(g.wheelAreas), len(g.blocks)
+	p.sortieDetail = !p.sortieDetail
+	g.drawSortieScreen(old, o)
+	p.sortieDetail = !p.sortieDetail
+	g.buttons = g.buttons[:buttons]
+	g.wheelAreas = g.wheelAreas[:wheels]
+	g.blocks = g.blocks[:blocks]
+	g.drawSortieScreen(target, o)
+	direction := 1.0
+	if !p.sortieDetail {
+		direction = -1
+	}
+	dst.DrawImage(target, nil)
+	width, height := sheetSize()
+	x, y := (uiW-width)/2, (uiH-height)/2
+	body := image.Rect(x+1, y+56, x+width-1, y+height-64)
+	rect(dst, float32(body.Min.X), float32(body.Min.Y), float32(body.Dx()), float32(body.Dy()), color.NRGBA{R: 44, G: 30, B: 24, A: 255})
+	clip := image.Rect(int(drawOX+float64(body.Min.X)*drawPx), int(drawOY+float64(body.Min.Y)*drawPx), int(drawOX+float64(body.Max.X)*drawPx), int(drawOY+float64(body.Max.Y)*drawPx)).Intersect(dst.Bounds())
+	if !clip.Empty() {
+		surface := dst.SubImage(clip).(*ebiten.Image)
+		op := &ebiten.DrawImageOptions{}
+		op.GeoM.Translate(-direction*float64(width)*t*drawPx, 0)
+		surface.DrawImage(old, op)
+		op = &ebiten.DrawImageOptions{}
+		op.GeoM.Translate(direction*float64(width)*(1-t)*drawPx, 0)
+		surface.DrawImage(target, op)
+	}
+	g.buttons = g.buttons[:buttons]
+	g.wheelAreas = g.wheelAreas[:wheels]
+}
+
+func (g *Game) drawSortieScreen(dst *ebiten.Image, o core.Observation) {
 	w, h := sheetSize()
 	x, y := (uiW-w)/2, (uiH-h)/2
+	close := func() { g.party.sortieClosed = true }
+	for _, r := range []image.Rectangle{image.Rect(0, 0, uiW, y), image.Rect(0, y, x, y+h), image.Rect(x+w, y, uiW, y+h), image.Rect(0, y+h, uiW, uiH)} {
+		if !r.Empty() {
+			g.addButton(r, "", close)
+		}
+	}
 	g.window(dst, image.Rect(x, y, x+w, y+h))
 	g.block(image.Rect(x, y, x+w, y+h))
 	ids := g.sortieCards(o)
@@ -31,149 +96,60 @@ func (g *Game) drawSortie(dst *ebiten.Image, o core.Observation) {
 		g.selected = ids[0]
 	}
 	st := g.S.Data.Stages[o.Stage]
-	listW := min(340, w*9/25)
-	rect(dst, float32(x+listW), float32(y+8), 1, float32(h-16), goldDim)
-	g.label(dst, "출전 준비", float64(x+16), float64(y+12), 18, gold)
-	tx := float64(x+28) + g.width("출전 준비", 18)
 	count := fmt.Sprintf("출전 %d / %d", g.sortieCount(o, o.Deployment), st.Max)
 	if len(st.Reserve) > 0 {
 		count += " (복병 제외)"
 	}
-	g.label(dst, count, tx, float64(y+17), 13, muted)
-	rect(dst, float32(x+8), float32(y+56), float32(listW-16), 1, goldDim)
-	// 목표 and 메뉴 under the list, on the line of the actions, so the title row holds only the title and the count
-	foot := y + h - 60
-	listFoot := foot
-	if s := g.campaignSlot(); s != nil {
-		listFoot -= 44
-		g.cornerButton(dst, image.Rect(x+8, listFoot, x+listW-8, listFoot+34), s)
-	}
-	if s := g.campaignPolicySlot(); s != nil {
-		listFoot -= 44
-		g.cornerButton(dst, image.Rect(x+8, listFoot, x+listW-8, listFoot+34), s)
-	}
-	g.mBtn(dst, image.Rect(x+8, foot, x+8+88, foot+touchH), "목표", false, g.openObjective)
-	g.mBtn(dst, image.Rect(x+8+96, foot, x+8+96+88, foot+touchH), "메뉴", false, func() { g.openOverlay("menu") })
-	_, gearOwner, _, guide := g.gearStep()
-	guide = guide && g.overlay() == ""
-	g.listRows(dst, &g.party.officers, image.Rect(x+6, y+58, x+listW-4, listFoot-6), len(ids), listRowH, func(i int, rr image.Rectangle) {
-		g.sortieRow(dst, o, rr, ids[i])
-		if guide && ids[i] == gearOwner && ids[i] != g.selected {
-			g.guideRim(dst, rr.Inset(4))
-		}
-	}, func(i int) { g.selected, g.party.trait, g.party.tip = ids[i], "", "" })
-
-	of := g.officer(o, g.selected)
-	if of == nil {
-		return
-	}
-	// the picked officer as in the officers' window, with a 부관 tab
-	rx, right := x+listW+16, x+w-16
-	g.label(dst, g.name(of.ID), float64(rx), float64(y+12), 18, gold)
+	g.label(dst, "출전 준비", float64(x+16), float64(y+12), 18, gold)
+	g.label(dst, count, float64(x+140), float64(y+17), 13, muted)
 	p := &g.party
-	p.sortieTab = min(p.sortieTab, len(sortieTabs)-1)
-	g.tabRow(dst, rx+int(g.width(g.name(of.ID), 18))+18, y+4, sortieTabs, p.sortieTab, func(i int) { p.sortieTab, p.tip = i, "" })
-	if tx := rx + int(g.width(g.name(of.ID), 18)) + 18; guide && of.ID == gearOwner && p.sortieTab != 2 {
-		g.guideRim(dst, g.tabRect(tx, y+4, sortieTabs, 2))
+	leftW, listW := w*3/10, w*6/10
+	if p.sortieDetail {
+		leftW, listW = w/10, w*2/10
 	}
-	rect(dst, float32(rx), float32(y+52), float32(right-rx), 1, goldDim)
-	body := image.Rect(rx, y+60, right, y+h-64)
+	foot := y + h - 60
+	bodyTop := y + 56
+	listX, detailX := x+leftW, x+leftW+listW
+	rect(dst, float32(listX), float32(bodyTop), 1, float32(foot-bodyTop-4), goldDim)
+	rect(dst, float32(detailX), float32(bodyTop), 1, float32(foot-bodyTop-4), goldDim)
 	var acts []action
-	switch p.sortieTab {
-	case 0:
-		g.officerBio(dst, o, of, body)
-	case 1:
-		acts = g.officerTraits(dst, of, body)
-	case 2:
-		if lead := g.servedBy(o, of.ID); lead != "" { // a deputy wears nothing
-			c := image.Rect(body.Min.X, body.Min.Y, body.Max.X, body.Min.Y+64)
-			g.card(dst, c)
-			g.label(dst, g.name(lead)+"의 부관이라 장비를 들지 않습니다. 장비는 창고에 있습니다.", float64(c.Min.X+18), float64(c.Min.Y+22), 14, muted)
-		} else {
-			g.officerEquipment(dst, of, body)
+	if p.sortieDetail {
+		back := func() { p.sortieDetail = false; p.sortieSlide = max(g.uiSeconds, .0001) }
+		g.addButton(image.Rect(x+4, bodyTop+52, listX-4, foot-4), "", back)
+		g.mBtn(dst, image.Rect(x+4, bodyTop+4, listX-4, bodyTop+52), "‹", false, back)
+		g.sortieNames(dst, o, ids, image.Rect(listX+4, bodyTop, detailX-4, foot-4))
+		acts = g.sortieDetailBody(dst, o, image.Rect(detailX+12, y+4, x+w-12, foot-4))
+	} else {
+		g.sortieGrid(dst, o, image.Rect(x+6, bodyTop, listX-4, foot-4))
+		g.sortieTable(dst, o, ids, image.Rect(listX+4, bodyTop, detailX-4, foot-4))
+		forward := func() { p.sortieDetail = true; p.sortieSlide = max(g.uiSeconds, .0001) }
+		g.addButton(image.Rect(detailX+4, bodyTop+52, x+w-4, foot-4), "", forward)
+		g.mBtn(dst, image.Rect(detailX+4, bodyTop+4, x+w-4, bodyTop+52), "상세 ›", false, forward)
+		if _, owner, _, guide := g.gearStep(); guide && g.overlay() == "" && owner == g.selected {
+			g.guideRim(dst, image.Rect(detailX+4, bodyTop+4, x+w-4, bodyTop+52))
 		}
-	case 3:
-		acts = g.officerDeputy(dst, o, of, body)
 	}
-	belongings := action{"소지품", false, func() { // the seals can be used before the battle too
+	belongings := action{"소지품", false, func() {
 		p.items, p.table, p.insu = true, false, insuState{}
 		g.openOverlay("party")
 	}}
-	// the tab's actions on the left, apart from 소지품 and 출진 at the right
-	row := []action{belongings, {"출진 ›", true, g.startSortie}}
-	if left := g.actionRow(dst, right, y+h-60, row...); len(acts) > 0 {
-		g.actionRow(dst, left-closeGap, y+h-60, acts...)
+	row := []action{belongings, {"출진 ›", true, g.startSortie}, {"닫기", false, close}}
+	if edge := g.actionRow(dst, x+w-16, foot, row...); len(acts) > 0 {
+		g.actionRow(dst, edge-closeGap, foot, acts...)
 	}
-	if _, ok := g.promoteGuide(); ok && g.overlay() == "" && !guide {
-		g.promoteRim(dst, g.rowActionRect(right, y+h-60, belongings.label, row...))
-	}
-	if g.overlay() == "" && g.deputyInfo.id != "" {
-		g.chipWindow(dst, "sortie:deputy-info", g.deputyInfo, func() { g.deputyInfo = equipChip{} })
+	if _, _, _, guide := g.gearStep(); !guide && g.overlay() == "" {
+		if _, promote := g.promoteGuide(); promote {
+			g.promoteRim(dst, g.rowActionRect(x+w-16, foot, belongings.label, row...))
+		}
 	}
+	g.mBtn(dst, image.Rect(x+8, foot, x+96, foot+touchH), "목표", false, g.openObjective)
+	g.mBtn(dst, image.Rect(x+104, foot, x+192, foot+touchH), "메뉴", false, func() { g.openOverlay("menu") })
+
 }
 
 // sortieTabs are the tabs of the picked officer in the sortie preparation: the officers' window's and 부관.
 var sortieTabs = append(append([]string{}, officerTabs...), "부관")
 
-// sortieRow is an officer's row: the state pill (출전, 필수, 부관 over whom, 대기), pressed to send out or call
-// back, then the officer as in the officers' list, the missing gear after the class.
-func (g *Game) sortieRow(dst *ebiten.Image, o core.Observation, rr image.Rectangle, id string) {
-	of := g.officer(o, id)
-	if of == nil {
-		return
-	}
-	picked := id == g.selected
-	lead := g.servedBy(o, id)
-	label, body, rim, fg := "대기", color.NRGBA{}, color.NRGBA{R: 90, G: 70, B: 48, A: 255}, faintInk
-	switch {
-	case lead != "":
-		label, body, rim, fg = "부관", pillDep, depRim, depInk
-	case slices.Contains(o.Deployment, id):
-		label, body, rim, fg = "출전", pillOut, gold, gold
-		if g.sortieReserve(o, id) {
-			label = "복병"
-		} else if g.sortieLocked(id) {
-			label = "필수"
-		}
-	}
-	// the officer's lines right of the state pill
-	pill := image.Rect(rr.Min.X+10, rr.Min.Y+(listRowH-40)/2, rr.Min.X+10+sortiePillW, rr.Min.Y+(listRowH+40)/2)
-	row := rr
-	row.Min.X = pill.Max.X - 2
-	g.officerGround(dst, o, rr, id, picked)
-	note, noteC := fmt.Sprintf("특성치 %d", of.Points), muted
-	if g.hasAffordable(*of) {
-		noteC = good
-	}
-	if g.sortieReserve(o, id) && lead == "" {
-		note, noteC = "복병 · 전투 중 출현", gold
-	}
-	classEnd := g.officerLines(dst, o, row, *of, true, note, noteC)
-	if lead == "" { // a deputy wears nothing
-		g.missingGear(dst, of.Class, of.Equipment, classEnd+6, float64(row.Min.Y+28))
-	}
-	if body.A > 0 {
-		rect(dst, float32(pill.Min.X), float32(pill.Min.Y), float32(pill.Dx()), float32(pill.Dy()), body)
-	}
-	strokeRect(dst, float32(pill.Min.X)+.5, float32(pill.Min.Y)+.5, float32(pill.Dx()-1), float32(pill.Dy()-1), 1, rim)
-	cx := float64(pill.Min.X+pill.Max.X) / 2
-	if lead != "" { // 부관 over whom it serves
-		who := ellipsis(g, g.name(lead), 12, pill.Dx()-8)
-		g.label(dst, label, cx-g.width(label, 13)/2, float64(pill.Min.Y+2), 13, fg)
-		g.label(dst, who, cx-g.width(who, 12)/2, float64(pill.Min.Y+21), 12, fg)
-	} else {
-		g.label(dst, label, cx-g.width(label, 13)/2, float64(pill.Min.Y)+(40-13*1.45)/2, 13, fg)
-	}
-	tag := label
-	if lead != "" {
-		tag = "부관 · " + g.name(lead)
-	}
-	g.addButton(image.Rect(rr.Min.X, rr.Min.Y, pill.Max.X+6, rr.Max.Y), tag, func() { g.togglePrep(id) })
-}
-
-// sortiePillW is how wide the state pill of a sortie row is.
-const sortiePillW = 64
-
 // togglePrep sends a waiting officer out or calls one back; the lord and the stage's required officers always go,
 // and a deputy cannot go while serving.
 func (g *Game) togglePrep(id string) {
diff --git a/ver5/internal/gui/uidemo_sortie.go b/ver5/internal/gui/uidemo_sortie.go
index 5c24eb4..5fa3a91 100644
--- a/ver5/internal/gui/uidemo_sortie.go
+++ b/ver5/internal/gui/uidemo_sortie.go
@@ -12,7 +12,7 @@ func init() { demos["sortie"] = demoSortie }
 
 // demoSortie jumps to the -node battle node with every officer's level-up learning closed, names the second
 // officer of the sortie who is not the lord the first one's deputy (c14-battle: 손건 under 고순) and walks the sortie preparation for cmd/_uidemo-vm: the list with the lord picked (20),
-// 고순 with his deputy (50), the last officer called back to wait by a press on his 출전 pill (80), the deputy window (110), the officer's
+// 고순 with his deputy (50), the last officer called back to wait by a press on his 출전 취소 화살표 (80), the deputy window (110), the officer's
 // trait tab (140), the unit detail on 전투력 (155), 열전 (180) and the deputy's traits (190) and, after 출진, the deployment (215).
 func demoSortie(d *Demo) error {
 	g := d.g
@@ -44,7 +44,7 @@ func demoSortie(d *Demo) error {
 		switch f {
 		case 50:
 			g.selected = head
-		case 80: // press the last 출전 pill as a player would, so the row under it cannot steal the press
+		case 80: // press the 출전 취소 arrow as a player would, so the row under it cannot steal the press
 			for i := len(g.buttons) - 1; i >= 0; i-- {
 				if b := g.buttons[i]; b.label == "출전" {
 					at := b.rect.Min.Add(b.rect.Size().Div(2))
diff --git a/ver5/internal/gui/uidemo_sortie_deputy.go b/ver5/internal/gui/uidemo_sortie_deputy.go
index b901b29..995f7df 100644
--- a/ver5/internal/gui/uidemo_sortie_deputy.go
+++ b/ver5/internal/gui/uidemo_sortie_deputy.go
@@ -67,7 +67,7 @@ func demoSortieDeputy(d *Demo) error {
 		p := &g.party
 		switch f {
 		case 1:
-			g.selected, p.sortieTab = lead, 3
+			g.selected, p.sortieTab, p.sortieDetail, p.sortieNode = lead, 3, true, g.S.View().Node
 		case 40:
 			g.deputyPick = look
 		case 60:

```

## 당시 새 파일: `docs/plan-sortie-slide.md`

````text
# 출전 준비 화면 개편 설계

출전 준비를 출전 부대·전체 부대 표·선택 부대 상세 세 영역으로 나눈다. 편성 보기에서 폭은 3:6:1, 상세 보기에서는 1:2:7이며 영역 경계를 애니메이션으로 이동한다. 상세 보기 목록은 이름만 남기고 선택을 유지한다. 출전 부대는 출전 순서대로 좌상단부터 가로 셋으로 배치하며 병종 그림은 남서 방향이다. 빈 칸은 최대 18개까지 보여 주되 실제 출전 제한은 스테이지를 따른다.

전체 표는 관리창의 기본·전투력·열전·장비 열을 공유하고 소지품을 볼 수 있어야 한다. 표 선택과 출전 변경은 분리한다. 왼쪽 화살표는 선택한 부대를 출전시키고 오른쪽 화살표는 취소한다. 기존 필수·군주·부관·복병·인원 제한 검증을 사용하며 복병은 출전 인원에서 제외한다. 상세에서는 기존 특성·장비·부관 조작과 열전을 유지한다. 출진은 배치로 이어지고 닫기는 그 오른쪽에 둔다.

전력은 표시용 수치로, 장수의 공격·방어·정신·순발·사기·최대 병력과 최대 병법치의 10배, 습득한 특성의 비용 합이다. 관리 표와 출전 표에서 같은 계산을 사용한다. 부관 보정 적용 여부와 장비·타고난 특성을 비용에 포함하는지에 따라 값이 달라지므로 게임의 기존 표 기준을 따른다: 장수 단독 능력치와 직접 습득한 특성의 비용만 계산한다.

검증: 편성·필수·부관·복병과 정렬, 상세 조작 및 소지품 복귀를 시험하고 기존 출전 데모에 두 상태와 전환을 추가해 일반 화면과 휴대폰 화면을 캡처한다. 기존 자동 진행과 길 안내도 확인한다.

````

## 당시 새 파일: `internal/gui/sortie_slide.go`

````text
package gui

import (
	"fmt"
	"image"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

func (g *Game) sortieGrid(dst *ebiten.Image, o core.Observation, r image.Rectangle) {
	g.label(dst, "출전 부대", float64(r.Min.X+4), float64(r.Min.Y+6), 15, gold)
	if slot := g.campaignSlot(); slot != nil {
		g.cornerButton(dst, image.Rect(r.Max.X-90, r.Min.Y, r.Max.X-2, r.Min.Y+30), slot)
	}
	if slot := g.campaignPolicySlot(); slot != nil {
		g.cornerButton(dst, image.Rect(r.Max.X-184, r.Min.Y, r.Max.X-94, r.Min.Y+30), slot)
	}
	ids := g.placeable(o)
	top := r.Min.Y + 34
	cw := max(1, r.Dx()/3)
	ch := max(1, (r.Max.Y-top)/6)
	for i := 0; i < 18; i++ {
		x, y := r.Min.X+(i%3)*cw, top+(i/3)*ch
		cell := image.Rect(x+2, y+2, x+cw-2, y+ch-2)
		if cell.Max.Y > r.Max.Y {
			break
		}
		strokeRect(dst, float32(cell.Min.X), float32(cell.Min.Y), float32(cell.Dx()), float32(cell.Dy()), 1, cardRim)
		if i >= len(ids) {
			continue
		}
		id := ids[i]
		of := g.officer(o, id)
		if of == nil {
			continue
		}
		if id == g.selected {
			strokeRect(dst, float32(cell.Min.X+1), float32(cell.Min.Y+1), float32(cell.Dx()-2), float32(cell.Dy()-2), 1, gold)
		}
		a := g.battleClassArt(of.Def, of.Class, "", "", "ally")
		if a != nil {
			z := math.Min(2, float64(max(1, cell.Dy()-20))/float64(max(1, a.TopOffset())))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(z, z)
			op.GeoM.Translate(float64(cell.Min.X+cell.Dx()/2)-a.Pivot[0]*z, float64(cell.Max.Y-19)-a.Pivot[1]*z)
			drawScaled(dst, a.Frame("SW", "idle", 0), op)
		}
		g.centered(dst, ellipsis(g, g.name(id), 12, cell.Dx()-4), float64(cell.Min.X+cell.Dx()/2), float64(cell.Max.Y-18), 12, ink)
		g.addButton(cell, "", func() { g.selected = id; g.party.trait = "" })
	}
}

func (g *Game) sortieNames(dst *ebiten.Image, o core.Observation, ids []string, r image.Rectangle) {
	g.label(dst, "부대", float64(r.Min.X+4), float64(r.Min.Y+6), 15, gold)
	r.Min.Y += 34
	g.listRows(dst, &g.party.officers, r, len(ids), touchH, func(i int, rr image.Rectangle) {
		id := ids[i]
		g.officerGround(dst, o, rr, id, id == g.selected)
		g.label(dst, ellipsis(g, g.name(id), 15, rr.Dx()-12), float64(rr.Min.X+8), float64(rr.Min.Y+12), 15, ink)
	}, func(i int) { g.selected = ids[i]; g.party.trait = ""; g.party.tip = "" })
}

func (g *Game) sortieTable(dst *ebiten.Image, o core.Observation, ids []string, r image.Rectangle) {
	p := &g.party
	arrow := image.Rect(r.Min.X+2, r.Min.Y+4, r.Min.X+48, r.Min.Y+52)
	var add, remove func()
	if !slices.Contains(o.Deployment, g.selected) {
		add = func() { g.togglePrep(g.selected) }
	} else {
		remove = func() { g.togglePrep(g.selected) }
	}
	g.mBtn(dst, arrow, "←", false, add)
	arrow.Min.Y = r.Max.Y - touchH - 4
	arrow.Max.Y = arrow.Min.Y + touchH
	g.mBtn(dst, arrow, "→", false, remove)
	tx := r.Min.X + 54
	g.tabRow(dst, tx, r.Min.Y, officerViews, p.view, func(i int) { p.view = i; p.officerList.order = tableSort{} })
	type row struct {
		c     core.Character
		s     core.Stats
		attrs [6]int
	}
	rows := make([]row, 0, len(ids))
	for _, id := range ids {
		if c := g.officer(o, id); c != nil {
			s, a, _ := g.S.Engine.DeputyStats(id, "")
			rows = append(rows, row{*c, s, a})
		}
	}
	num := func(key, label string, f func(*row) int) column[row] {
		return column[row]{key: key, label: label, get: func(r *row) (string, float64) { v := f(r); return fmt.Sprint(v), float64(v) }}
	}
	cols := []column[row]{{key: "name", label: "장수", w: 95, text: true, left: true, get: func(r *row) (string, float64) { return g.name(r.c.ID), 0 }, after: func(dst *ebiten.Image, r *row, cell image.Rectangle) {
		if _, owner, _, guide := g.gearStep(); guide && g.overlay() == "" && owner == r.c.ID && owner != g.selected {
			g.guideRim(dst, cell.Inset(2))
		}
	}}}
	switch p.view {
	case 0:
		cols = append(cols, column[row]{key: "state", label: "상태", w: 55, text: true, get: func(r *row) (string, float64) {
			if g.sortieReserve(o, r.c.ID) {
				return "복병", 0
			}
			if g.servedBy(o, r.c.ID) != "" {
				return "부관", 0
			}
			if g.sortieLocked(r.c.ID) {
				return "필수", 0
			}
			if slices.Contains(o.Deployment, r.c.ID) {
				return "출전", 0
			}
			return "대기", 0
		}}, column[row]{key: "class", label: "병종", text: true, get: func(r *row) (string, float64) { return g.className(r.c.Class), 0 }}, num("level", "Lv", func(r *row) int { return r.c.Level }), num("power", "전력", func(r *row) int { return g.officerPower(r.c, r.s) }), num("hp", "병력", func(r *row) int { return r.s.MaxHP }), num("mp", "병법치", func(r *row) int { return r.s.MaxMP }), num("points", "특성치", func(r *row) int { return r.c.Points }))
	case 1:
		for _, v := range statRows[2:] {
			v := v
			cols = append(cols, num(v.name, v.name, func(r *row) int { return v.get(r.s) }))
		}
	case 2:
		for i, n := range attrNames {
			i := i
			cols = append(cols, num(n, n, func(r *row) int { return r.attrs[i] }))
		}
	case 3:
		for _, slot := range equipSlots {
			slot := slot
			cols = append(cols, column[row]{key: slot, label: slotNames[slot], text: true, get: func(r *row) (string, float64) {
				if id := r.c.Equipment[slot]; id != "" {
					return g.itemName(id), 0
				}
				return "—", 0
			}})
		}
	}
	tableRect := image.Rect(tx, r.Min.Y+54, r.Max.X-2, r.Max.Y-4)
	g.wheelAreas = append(g.wheelAreas, wheelArea{unframed(tableRect), &p.officerList})
	drawTable(g, dst, &p.officerList, table[row]{cols: cols, rowH: touchH, headH: 32, size: 13, center: true, selected: func(r *row) bool { return r.c.ID == g.selected }, pick: func(r *row) { g.selected = r.c.ID; p.trait = ""; p.tip = "" }}, rows, tableRect)
}

func (g *Game) officerPower(c core.Character, s core.Stats) int {
	n := s.Attack + s.Defense + s.Mind + s.Agility + s.Morale + s.MaxHP + s.MaxMP*10
	for _, id := range c.Learned {
		n += g.S.Data.Traits[id].Cost
	}
	return n
}

func (g *Game) sortieDetailBody(dst *ebiten.Image, o core.Observation, r image.Rectangle) []action {
	of := g.officer(o, g.selected)
	if of == nil {
		return nil
	}
	p := &g.party
	g.label(dst, g.name(of.ID), float64(r.Min.X), float64(r.Min.Y+8), 18, gold)
	tx := r.Min.X + int(g.width(g.name(of.ID), 18)) + 18
	p.sortieTab = min(p.sortieTab, len(sortieTabs)-1)
	g.tabRow(dst, tx, r.Min.Y, sortieTabs, p.sortieTab, func(i int) { p.sortieTab = i; p.tip = "" })
	if _, owner, _, guide := g.gearStep(); guide && g.overlay() == "" && owner == of.ID && p.sortieTab != 2 {
		g.guideRim(dst, g.tabRect(tx, r.Min.Y, sortieTabs, 2))
	}
	rect(dst, float32(r.Min.X), float32(r.Min.Y+52), float32(r.Dx()), 1, goldDim)
	body := image.Rect(r.Min.X, r.Min.Y+60, r.Max.X, r.Max.Y)
	var acts []action
	switch p.sortieTab {
	case 0:
		g.officerBio(dst, o, of, body)
	case 1:
		acts = g.officerTraits(dst, of, body)
	case 2:
		if lead := g.servedBy(o, of.ID); lead != "" {
			g.label(dst, g.name(lead)+"의 부관이라 장비를 들지 않습니다.", float64(body.Min.X), float64(body.Min.Y+16), 14, muted)
		} else {
			g.officerEquipment(dst, of, body)
		}
	case 3:
		acts = g.officerDeputy(dst, o, of, body)
	}
	if g.overlay() == "" && g.deputyInfo.id != "" {
		g.chipWindow(dst, "sortie:deputy-info", g.deputyInfo, func() { g.deputyInfo = equipChip{} })
	}
	return acts
}

````

## 당시 새 파일: `internal/gui/sortie_slide_test.go`

````text
package gui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
)

func TestSortiePower(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "sortie-slide", Node: "c14-battle"})
	require.NoError(t, err)
	g := d.g
	id := sorted(g.S.Data.Traits)[0]
	c := core.Character{Learned: []string{id}, Points: 999}
	s := core.Stats{Attack: 1, Defense: 2, Mind: 3, Agility: 4, Morale: 5, MaxHP: 6, MaxMP: 7, Recovery: 999}
	assert.Equal(t, 91+g.S.Data.Traits[id].Cost, g.officerPower(c, s))
}

func TestSortieSlideControls(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "sortie-slide", Node: "c14-battle"})
	require.NoError(t, err)
	g := d.g
	g.fit(1280, 850)
	draw := func() { g.advance(1.0 / 60); dst := ebiten.NewImage(1280, 850); defer dst.Deallocate(); g.Draw(dst) }
	click := func(label string) {
		t.Helper()
		for _, b := range g.buttons {
			if b.label == label {
				b.click()
				return
			}
		}
		t.Fatalf("button %q not found", label)
	}
	draw()
	var removable string
	for _, id := range g.S.View().Deployment {
		if !g.sortieLocked(id) {
			removable = id
			break
		}
	}
	require.NotEmpty(t, removable)
	g.selected = removable
	draw()
	click("→")
	assert.NotContains(t, g.S.View().Deployment, removable)
	draw()
	click("←")
	assert.Contains(t, g.S.View().Deployment, removable)
	draw()
	click("상세 ›")
	assert.True(t, g.party.sortieDetail)
	for i := 0; i < 20; i++ {
		g.advance(1.0 / 60)
	}
	draw()
	var closeRect, startRect image.Rectangle
	for _, b := range g.buttons {
		if b.label == "닫기" {
			closeRect = b.rect
		}
		if b.label == "출진 ›" {
			startRect = b.rect
		}
	}
	assert.Greater(t, closeRect.Min.X, startRect.Max.X)
	click("닫기")
	assert.True(t, g.party.sortieClosed)
	draw()
	click("출전 준비")
	assert.False(t, g.party.sortieClosed)
	g.rightClick()
	assert.True(t, g.party.sortieClosed)
}

````

## 당시 새 파일: `internal/gui/uidemo_sortie_slide.go`

````text
package gui

import (
	"fmt"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["sortie-slide"] = demoSortieSlide }

func demoSortieSlide(d *Demo) error {
	g := d.g
	if d.node == "" {
		return fmt.Errorf("sortie-slide: pass -node")
	}
	g.request(session.Request{Op: "jump", Seed: 1, Node: d.node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	d.step = func(f int) {
		if f == 90 {
			g.party.sortieDetail = true
			g.party.sortieSlide = g.uiSeconds
		}
	}
	return nil
}

````

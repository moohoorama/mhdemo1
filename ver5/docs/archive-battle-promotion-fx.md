# 보관: 전투 중 승급 연출

2026-11-13 인수를 전투 밖에서만 쓰게 하면서 코드에서 뺀 연출. 탐험·출전 준비 승급에 다시 쓸 때 여기서 가져온다.

## 흐름 (약 2.7초)

1. 카메라가 그 장수에게 간다 (`panTime`).
2. 인수 아이콘(`UI["item_insu"]`)이 머리 위로 떠오른다 (0.5초, 효과 `insu` 1.1초).
3. 몸이 1초에 걸쳐 하얘지고(`bleach`), 사방에서 빛 알갱이 24개가 가슴으로 모인다 (`promote_gather`, 1.0초).
4. 화면이 번쩍이고 `promote` 효과음 (0.15초).
5. 새 병종 모습으로 바뀌고, 흰빛이 0.8초에 걸쳐 빠지며 알갱이 26개가 터져 나간다 (`promote`, 0.9초).

## 남아 있는 자산

- `assets/graphics/effects/fx.json`의 `promote_gather`·`promote`·`promote_mote`
- `assets/sounds/promote.ogg`
- `UI["item_insu"]` 아이콘

## 원래 코드

전체는 태그 `archive/battle-promotion-fx`(5147a31)에 있다.

```sh
git show archive/battle-promotion-fx:ver5/internal/gui/player.go        # promotion(), bleach 필드·update
git show archive/battle-promotion-fx:ver5/internal/gui/effects.go       # insu·promote_gather·promote 생성과 그리기
git show archive/battle-promotion-fx:ver5/internal/gui/field.go         # 흰 실루엣 max(.9*charge, bleach)
git show archive/battle-promotion-fx:ver5/internal/gui/uidemo_branch.go # -ui branch 데모
git show archive/battle-promotion-fx:ver5/internal/gui/uidemo_branch_deputy.go
```

### player.go

```go
// unitVis 필드
bleach     float64 // 0..1: how white the sprite is drawn while it is promoted
bleachTo   float64 // where bleach is heading, at bleachRate a second
bleachRate float64

// unitVis.update
if u.bleach < u.bleachTo {
	u.bleach = math.Min(u.bleachTo, u.bleach+dt*u.bleachRate)
} else {
	u.bleach = math.Max(u.bleachTo, u.bleach-dt*u.bleachRate)
}

// perform: item 명령
if g.S.Data.Items[c.Item].Effect == content.ItemPromote {
	g.promotion(add, actor)
	break
}

// promotion plays a unit taking its new rank: the seal rises over its head, it turns white while gold
// rings and motes close in, the screen flashes, it takes its new look in a pillar of light and the
// white drains away. The promotion window follows the replay.
func (g *Game) promotion(add func(float64, func()), u *unitVis) {
	if u == nil {
		add(.1, func() {})
		return
	}
	add(panTime, func() { g.panToCell(int(u.u), int(u.v)) })
	add(.5, func() { g.spawn("insu", nil, u, 1.1, white) })
	add(1.0, func() {
		u.bleachTo, u.bleachRate = 1, 1
		g.spawn("promote_gather", nil, u, 1.0, white)
	})
	add(.15, func() {
		g.hitFlashT, g.hitFlashCrit = hitFlash, false
		g.sfx("promote")
	})
	add(.9, func() {
		if final := g.unitView(g.player.final, u.id); final != nil {
			g.syncUnit(*final, g.player.final)
		}
		u.play("idle")
		u.bleachTo, u.bleachRate = 0, 1.25
		g.spawn("promote", nil, u, 1.0, white)
	})
}
```

### field.go (drawUnit)

```go
if k := max(.9*u.charge, u.bleach); k > 0 && ghost == 0 { // gathering a critical blow or being promoted: a white silhouette fades in over the sprite
```

### effects.go

```go
// spawn
case "promote_gather":
	// motes come in from all round to the chest, ending where they meet
	for i := 0; i < 24; i++ {
		a := r.Float64() * 2 * math.Pi
		d := 34 + r.Float64()*26
		life := .7 + r.Float64()*.3
		x, y := math.Cos(a)*d, math.Sin(a)*d*.5+chest*.6
		add(particle{x: x, y: y, vx: -x / life, vy: (chest*.6 - y) / life, life: life, sprite: "promote_mote", in: true})
	}
case "promote":
	for i := 0; i < 26; i++ {
		a := r.Float64() * 2 * math.Pi
		v := 50 + r.Float64()*70
		add(particle{x: 0, y: chest * .6, vx: math.Cos(a) * v, vy: math.Sin(a)*v*.5 - 30, g: -40,
			life: .5 + r.Float64()*.4, sprite: "promote_mote"})
	}

// drawEffects
case "insu": // the seal of the new rank rises over the head, dot for dot with the sprites
	k := 1 - math.Pow(1-min(1, e.t/.35), 2)
	a := min(1, e.t/.15, (e.dur-e.t)/.2)
	sh := g.assets.UI["item_insu"]
	sx, sy := g.toScreen(x, y-e.lift-e.height-14-6*k)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(sh.Frames[0].Bounds().Dx())/2, -float64(sh.Frames[0].Bounds().Dy())/2)
	op.GeoM.Scale(z/4, z/4)
	op.GeoM.Translate(math.Round(sx), math.Round(sy))
	op.ColorScale.ScaleAlpha(float32(a))
	drawScaled(dst, sh.Frames[0], op)
case "promote_gather", "promote":
	sh := g.assets.FX[e.kind]
	sh.draw(dst, sh.frameOver(k), x, y-e.lift, g.toScreen, z)
```

## 다시 쓸 때 주의

- 전투 리플레이(`perform`의 beat)와 전투 필드(`unitVis`) 위에서 돌던 연출이다. 탐험·출전 준비에는 필드 유닛이 없으므로, 탐험 걷기 인물이나 카드 초상 위에서 돌리려면 그리기 대상을 바꿔야 한다.
- `syncUnit`으로 새 병종 스프라이트로 갈아 끼우는 시점이 흰빛이 가장 짙을 때다. 바뀌는 순간을 가리는 역할이므로 순서를 유지한다.

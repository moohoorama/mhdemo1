// Temporary tool: renders Bubble.DrawText with assorted texts into reports/bubble/cases.png.
//
//	go run ./cmd/_bubble
package main

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"srpg/internal/duelview"
)

type sample struct{ name, body string }

var samples = []sample{
	{"갑", "이 땅에 다시 한실의 깃발을 세우리라."},
	{"을", "형님, 걱정 마십시오. 이 관운장이 있는 한 누구도 형님의 털끝 하나 건드리지 못합니다."},
	{"병", "이놈들아! 연인 장익덕이 여기 있다! 어디 한번 덤벼 보거라!"},
	{"정", "寧教我負天下人, 休教天下人負我."},
	{"무", "첫째, 형주를 취하여 근본을 삼으십시오.\n둘째, 익주를 얻어 발판으로 삼으십시오.\n셋째, 손권과 손잡고 조조를 막으십시오.\n넷째, 때를 기다려 중원을 도모하십시오.\n다섯째, 백성의 마음을 얻으십시오.\n여섯째, 인재를 널리 구하십시오."},
	{"기", "내 방천화극 앞에서 살아 돌아간 자는 없다. 그것이 이 여봉선의 이름값이다."},
	{"갑", "가로로 아주아주아주아주아주아주아주아주아주아주아주아주아주 길게 이어지는 한 줄짜리 대사이며 공백 없이붙어있는경우도테스트합니다"},
	{"경", "네."},
	{"아주 긴 이름을 가진 장수", "짧은 대사"},
	{"", "!"},
	{"", "?"},
	{"", "•"},
	{"", "••"},
	{"", "•••"},
	{"Zhuge Liang", "A long English sentence should wrap at spaces rather than in the middle of a word, like this."},
	{"을", "일\n이\n삼\n사\n오\n육\n칠"},
}

func main() {
	f, err := os.Open("assets/fonts/NotoSansKR.ttf")
	check(err)
	src, err := text.NewGoTextFaceSource(f)
	check(err)
	b, err := duelview.LoadBubble("assets/graphics/ui/speech-bubble", src)
	check(err)
	check(ebiten.RunGame(&game{b: b}))
}

type game struct {
	b    *duelview.Bubble
	done bool
}

func (g *game) Layout(int, int) (int, int) { return 1600, 1100 }
func (g *game) Update() error              { return nil }

func (g *game) Draw(screen *ebiten.Image) {
	if g.done {
		return
	}
	g.done = true
	screen.Fill(color.NRGBA{196, 170, 120, 255})
	x, y, rowH := 20.0, 20.0, 0.0
	for _, s := range samples {
		w, h := g.b.Size(s.name, s.body)
		if x+w > 1580 {
			x, y, rowH = 20, y+rowH+24, 0
		}
		g.b.DrawText(screen, s.name, s.body, x, y, 1)
		x += w + 24
		rowH = max(rowH, h)
	}
	out := filepath.Join("reports", "bubble", "cases.png")
	o, err := os.Create(out)
	check(err)
	check(png.Encode(o, screen))
	check(o.Close())
	fmt.Println(out)
	os.Exit(0)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

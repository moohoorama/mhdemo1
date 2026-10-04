package gui

import (
	"reflect"
	"testing"
)

func TestSplitLines(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []line
	}{
		{"title as speaker", "도원결의: 의용군을 일으킨다.", []line{{speakers: []string{"도원결의"}, text: "의용군을 일으킨다."}}},
		{"plain", "황건적의 난 — 출전 준비", []line{{text: "황건적의 난 — 출전 준비"}}},
		{"two speakers", "유비: 뜻을 세웁시다. 관우·장비: 함께하겠습니다.", []line{
			{speakers: []string{"유비"}, text: "뜻을 세웁시다."},
			{speakers: []string{"관우", "장비"}, text: "함께하겠습니다."},
		}},
		{"narration then speaker", "동탁 타도군 일어나다. 유비: 나아갑시다.", []line{
			{text: "동탁 타도군 일어나다."},
			{speakers: []string{"유비"}, text: "나아갑시다."},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := splitLines(c.text); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestWrapAndParticle(t *testing.T) {
	t.Run("wrap at spaces", func(t *testing.T) {
		if got := wrap("이번 턴 적 옆을 지나가도 멈추지 않습니다", 12); got != "이번 턴 적 옆을\n지나가도 멈추지\n않습니다" {
			t.Fatalf("%q", got)
		}
	})
	t.Run("particle", func(t *testing.T) {
		for name, want := range map[string]string{"장비": "장비와", "관우": "관우와", "여포": "여포와", "화웅": "화웅과"} {
			if got := with(name); got != want {
				t.Fatalf("%s: %s", name, got)
			}
		}
	})
}

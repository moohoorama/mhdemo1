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

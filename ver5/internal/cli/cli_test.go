package cli

import (
	"bytes"
	"encoding/json"
	"srpg/internal/content"
	"srpg/internal/session"
	"strings"
	"testing"
)

func game(t *testing.T) *session.Session {
	t.Helper()
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(d, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestJSONLines(t *testing.T) {
	s := game(t)
	in := strings.NewReader("bad\n{\"id\":1,\"op\":\"new\",\"seed\":5}\n{\"op\":\"command\",\"revision\":99,\"command\":{\"kind\":\"next\"}}\n{\"op\":\"command\",\"revision\":0,\"command\":{\"kind\":\"next\"}}\n{\"op\":\"save\",\"slot\":\"1\"}\n{\"op\":\"quit\"}\n")
	var out bytes.Buffer
	if err := RunJSON(s, in, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 6 {
		t.Fatal(out.String())
	}
	for i, line := range lines {
		var r session.Response
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatal("non JSON output", line)
		}
		if (i == 0 || i == 2) == r.OK {
			t.Fatal("response", i, line)
		}
	}
	if s.Screen != "quit" {
		t.Fatal("quit")
	}
}
func TestHumanMenusSettingsAndSave(t *testing.T) {
	s := game(t)
	var out bytes.Buffer
	input := "3\nsettings ai step\n1\nnext\nchoose go\nsave 1\nsave 1\ny\nmenu\nresume\nstart\nmove hero 3 3\nend\nai\nsave 2\nload 1\nquit\n"
	if err := RunHuman(s, strings.NewReader(input), &out, 1); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"1. 시작하기", "Hello", "Prepare", "덮어쓸까요", "enemy", "계속하기"} {
		if !strings.Contains(out.String(), text) {
			t.Fatal("missing UI", text)
		}
	}
	if !s.Store.Exists("1") || !s.Store.Exists("2") || s.Settings.AI != "step" {
		t.Fatal("save/settings")
	}
}
func TestParseRejectsMalformedMoves(t *testing.T) {
	for _, s := range []string{"move 유비 nan 3", "attack", "item 유비", "unknown"} {
		if _, err := Parse(s); err == nil {
			t.Fatal("accepted", s)
		}
	}
}

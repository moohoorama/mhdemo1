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
func TestHumanPlaysByNumbers(t *testing.T) {
	s := game(t)
	var out bytes.Buffer
	// start, page the line, take the choice, sortie, deploy, pick Arrow, move it, wait, end the turn
	input := "1\n\n1\n1\n1\n1\n1\n3,2\n2\n2\nq\n"
	if err := RunHuman(s, "../testdata/mini", strings.NewReader(input), &out, 1); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"(1) 시작하기", "Hero: Hello", "출전 준비", "우리 차례", "bos!", "✓ (3,2)로 이동", "적군 차례"} {
		if !strings.Contains(out.String(), text) {
			t.Fatalf("missing %q in\n%s", text, out.String())
		}
	}
}
func TestParseRejectsMalformedMoves(t *testing.T) {
	for _, s := range []string{"move 유비 nan 3", "attack", "item 유비", "unknown"} {
		if _, err := Parse(s); err == nil {
			t.Fatal("accepted", s)
		}
	}
}

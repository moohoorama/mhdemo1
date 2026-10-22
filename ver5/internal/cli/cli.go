package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
	"strconv"
	"strings"
)

const Help = "next | choose 선택ID | deploy 장수... | equip/unequip 장수 장비ID | start\nplace 장수 x y | unplace 장수 | deployment_finish\nmove 장수 x y | attack 장수 적ID | skill 장수 병법 대상 | item 장수 아이템 대상\ndeputy 주장 부관 | undeputy 주장\nlearn 장수 특성 | learn_close 장수 | wait 장수 | end | ai(적 한 명령) | auto(현재 진영)\nmap | status | legal [장수] | preview attack 장수 대상 | save/load 슬롯 | slots\nsettings [ai auto/step | color true/false | detail true/false | speed 밀리초]\nmenu | resume | title | retry | quit"

func Parse(line string) (core.Command, error) {
	p := strings.Fields(line)
	if len(p) == 0 {
		return core.Command{}, fmt.Errorf("EmptyCommand")
	}
	c := core.Command{Kind: p[0]}
	argc := func(n int) error {
		if len(p) != n {
			return fmt.Errorf("Usage: %s", Help)
		}
		return nil
	}
	switch c.Kind {
	case "next", "start", "end", "continue", "deployment_finish":
		if err := argc(1); err != nil {
			return c, err
		}
	case "choose":
		if err := argc(2); err != nil {
			return c, err
		}
		c.Option = p[1]
	case "deploy":
		if len(p) < 2 {
			return c, fmt.Errorf("InvalidDeployment")
		}
		c.Deployment = p[1:]
	case "wait", "undeputy", "learn_close", "unplace":
		if err := argc(2); err != nil {
			return c, err
		}
		c.Actor = p[1]
	case "attack", "deputy":
		if err := argc(3); err != nil {
			return c, err
		}
		c.Actor = p[1]
		c.Target = p[2]
	case "equip", "unequip", "learn":
		if err := argc(3); err != nil {
			return c, err
		}
		c.Actor = p[1]
		c.Item = p[2]
		c.Trait = p[2]
	case "move", "place":
		if err := argc(4); err != nil {
			return c, err
		}
		c.Actor = p[1]
		var err error
		c.X, err = strconv.Atoi(p[2])
		if err != nil {
			return c, err
		}
		c.Y, err = strconv.Atoi(p[3])
		if err != nil {
			return c, err
		}
	case "skill", "item":
		if err := argc(4); err != nil {
			return c, err
		}
		c.Actor = p[1]
		c.Skill = p[2]
		c.Item = p[2]
		c.Target = p[3]
	default:
		return c, fmt.Errorf("UnknownCommand")
	}
	return c, nil
}
func RunJSON(s *session.Session, in io.Reader, out io.Writer) error {
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	enc := json.NewEncoder(out)
	for scan.Scan() {
		var q session.Request
		if err := json.Unmarshal(scan.Bytes(), &q); err != nil {
			if err = enc.Encode(session.Response{OK: false, Error: "InvalidJSON"}); err != nil {
				return err
			}
			continue
		}
		r := s.Handle(q)
		if err := enc.Encode(r); err != nil {
			return err
		}
		if r.OK && s.Screen == "quit" {
			return nil
		}
	}
	if err := scan.Err(); err != nil {
		return err
	}
	return nil
}
func speakerNames(d *content.Data, speaker string) string {
	names := []string{}
	for _, id := range strings.Split(speaker, "·") {
		names = append(names, d.Characters[id].Name)
	}
	return strings.Join(names, "·")
}

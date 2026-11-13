// Temporary tool: plays chapters 17–19 from a waypoint through their walks, lines and choices by the walk's guide,
// answering choices from -pick (node=option;...), and prints every line, the flags and where it stops.
//
//	go run ./cmd/_c17walk -at c17-walk -pick c17-a-c1=주지 않는다,c17-a-c2=주지 않는다
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	at := flag.String("at", "c17-walk", "waypoint")
	pick := flag.String("pick", "", "node=option,...")
	quiet := flag.Bool("q", false, "no lines")
	flag.Parse()
	d, err := content.Load("assets/data")
	check(err)
	e, err := core.NewAt(d, 1, *at, "")
	check(err)
	picks := map[string]string{}
	for _, p := range strings.Split(*pick, ";") {
		if k, v, ok := strings.Cut(p, "="); ok {
			picks[k] = v
		}
	}
	for i := 0; i < 2000; i++ {
		s := e.Snapshot()
		if len(s.Learning) > 0 {
			_, err := e.Apply(core.Command{Kind: "learn_close", Actor: s.Learning[0]})
			check(err)
			continue
		}
		if s.Phase != "scenario" {
			fmt.Println("== phase", s.Phase, "ending:", e.Ending())
			break
		}
		if e.Walking() {
			cmds := e.ExploreNext()
			if cmds == nil {
				fmt.Println("== walk stuck in", s.Spot.Area, "flags", s.Flags)
				break
			}
			for _, c := range cmds {
				if _, err := e.Apply(c); err != nil {
					fmt.Println("== walk error", c, err)
					os.Exit(1)
				}
			}
			fmt.Printf("-- walk %s %s → node %s\n", cmds[len(cmds)-1].Kind, cmds[len(cmds)-1].Target, e.Snapshot().Node)
			continue
		}
		n := d.Node(s.Node)
		switch n.Kind {
		case "say", "narrate":
			if !*quiet {
				fmt.Printf("%-10s %s: %s\n", n.ID, n.Speaker, n.Text)
			}
			_, err = e.Apply(core.Command{Kind: "next"})
		case "choice":
			o := picks[n.ID]
			fmt.Printf("%-10s [선택] %s → %s\n", n.ID, n.Text, o)
			_, err = e.Apply(core.Command{Kind: "choose", Option: o})
		case "battle":
			fmt.Println("== battle", n.Stage, "roster", s.Deployment, "flags", s.Flags)
			return
		default:
			fmt.Println("== stops at", n.Kind, n.ID)
			return
		}
		if err != nil {
			fmt.Println("== error at", n.ID, err)
			os.Exit(1)
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

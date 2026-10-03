package main

import (
	"encoding/json"
	"os"
	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	d, err := content.Load("assets/content/campaign.json")
	if err != nil {
		panic(err)
	}
	e := core.New(d, 1)
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "결의"}, {Kind: "start"}} {
		if _, err = e.Apply(c); err != nil {
			panic(err)
		}
	}
	json.NewEncoder(os.Stdout).Encode(e.Observe())
}

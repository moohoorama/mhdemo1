// _mapfmt rewrites scene map files (format 2) the way scenemap.Encode lays them out, so a tool's map passes
// TestEncode.
//
//	go run ./cmd/_mapfmt assets/graphics/maps/S_zhuo.json ...
package main

import (
	"fmt"
	"os"

	"srpg/internal/scenemap"
)

func main() {
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		m, v2, err := scenemap.Parse(data)
		if err != nil || !v2 {
			panic(fmt.Sprintf("%s: %v (format 2: %v)", path, err, v2))
		}
		out, err := m.Encode()
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			panic(err)
		}
	}
}

// Deprecated command alias; use go run ./cmd/assetgen.
package main

import (
	"demo1/internal/assetbuild"
	"log"
)

func main() {
	if err := assetbuild.Generate("assets/objects", "assets/generated"); err != nil {
		log.Fatal(err)
	}
}

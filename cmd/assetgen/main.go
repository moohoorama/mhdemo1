// assetgen prepares runtime sheets without opening a game window.
package main

import (
	"demo1/internal/assetbuild"
	"flag"
	"log"
)

func main() {
	source := flag.String("source", "assets/objects", "source object PNG directory")
	out := flag.String("out", "assets/generated", "generated runtime asset directory")
	flag.Parse()
	if err := assetbuild.Generate(*source, *out); err != nil {
		log.Fatal(err)
	}
	log.Printf("Generated runtime sheets and catalog in %s", *out)
}

package main

import (
	"fmt"
	"os"

	"github.com/nullmonk/gnome"
)

/* To embed assets directly into the binary, you can use the following:

//go:embed assets/*
var assets embed.FS
assets, _ := fs.Sub(assets, "assets") // strip "assets/" from the embedded asset names
scripts, err := gnome.GetScripts(assets) // Now all the scripts will have the correct assets

*/

func main() {
	// An unescape-able fs that only allows access to files in the given directory
	rootFs, err := os.OpenRoot(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "[!]", err)
		return
	}
	assets := rootFs.FS()

	scripts, err := gnome.GetScripts(assets)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[!]", err)
		return
	}
	gnome.Run(scripts, nil, nil)
}

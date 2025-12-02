package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/nullmonk/gnome"
	"github.com/nullmonk/gnome/backends/httpfs"
)

/* To embed assets directly into the binary, you can use the following:

//go:embed assets/*
var assets embed.FS
assets, _ := fs.Sub(assets, "assets") // strip "assets/" from the embedded asset names
scripts, err := gnome.GetScripts(assets) // Now all the scripts will have the correct assets

*/

func main() {
	assets := &httpfs.HTTPFS{
		Url:    "http://localhost:8080/",
		Client: http.DefaultClient,
	}
	//assets := os.DirFS("assets")
	scripts, err := gnome.GetScripts(assets)
	fmt.Println(scripts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[!]", err)
		return
	}
	gnome.Run(scripts, nil, nil)
}

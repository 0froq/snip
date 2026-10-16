// snip is a minimal macOS clipboard-manager feasibility prototype (MyGo native UI).
package main

import (
	"log"

	"github.com/0froq/snip/app"
	"github.com/egoist/mygo"
)

func main() {
	cfg, path := app.LoadConfigFromDefaultPaths()
	snip := app.New(cfg, path)
	mygo.App.WhenReady(snip.WhenReady)
	mygo.App.OnWindowAllClosed(func() {
		// Menu-bar style: keep running for clipboard watch.
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

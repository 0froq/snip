// snip is a minimal macOS clipboard-manager feasibility prototype (MyGo native UI).
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/0froq/snip/app"
)

func main() {
	cfg := app.LoadConfigFromDefaultPaths()
	snip := app.New(cfg)
	mygo.App.WhenReady(snip.WhenReady)
	mygo.App.OnWindowAllClosed(func() {
		// Menu-bar style: keep running for clipboard watch.
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

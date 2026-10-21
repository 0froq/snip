// snip is a minimal macOS clipboard-manager feasibility prototype (MyGo native UI).
package main

import (
	"log"
	"os"
	"strings"

	"github.com/0froq/snip/app"
	"github.com/egoist/mygo"
)

func configOverride(args []string) string {
	if v := strings.TrimSpace(os.Getenv("SNIP_CONFIG")); v != "" {
		return v
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "-config" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], "-config=") {
			return strings.TrimPrefix(args[i], "-config=")
		}
	}
	return ""
}

func main() {
	cfg, path := app.LoadConfigFromDefaultPaths(configOverride(os.Args[1:]))
	snip := app.New(cfg, path)
	mygo.App.WhenReady(snip.WhenReady)
	mygo.App.OnWindowAllClosed(func() {
		// Menu-bar style: keep running for clipboard watch.
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

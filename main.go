package main

import (
	"flag"

	"github.com/nikitamarchenko/hta/internal/cmd/v1"
	"github.com/nikitamarchenko/hta/internal/cmd/v2"
)

func main() {
	uiVersion := flag.Int("ui-version", 2, "1: tui\n2: tui on bubbletea")
	debug := flag.Int("debug", 0, "use debug for debug")
	filename := flag.String("filename", "./hta.json", "hta db")

	flag.Parse()

	switch *uiVersion {
	case 1:
		cmd_v1.Run(*debug, *filename)
	case 2:
		cmd_v2.Run(*debug, *filename)
	default:
		flag.PrintDefaults()
	}
}

package util

import (
	"fmt"
	"os"
)

var debug int

func DebugF(format string, a ...any) {
	if debug > 0 {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

func Debug2F(format string, a ...any) {
	if debug > 1 {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

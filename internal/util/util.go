package util

import (
	"fmt"
	"os"
)

var Debug int

func DebugF(format string, a ...any) {
	if Debug > 0 {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}

func Debug2F(format string, a ...any) {
	if Debug > 1 {
		fmt.Fprintf(os.Stdout, format, a...)
	}
}
